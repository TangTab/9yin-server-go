package fswatch

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WatchState 官方 GameWatch::GetState 四态（this+0x28 → 跳转表
// 0x1006d400：0=Unknown 1=Opened 2=Paused 3=Closed）。
type WatchState int

const (
	StateUnknown WatchState = iota
	StateOpened
	StatePaused
	StateClosed
)

func (s WatchState) String() string {
	switch s {
	case StateOpened:
		return "Opened"
	case StatePaused:
		return "Paused"
	case StateClosed:
		return "Closed"
	}
	return "Unknown"
}

// GameWatch 官方 GameWatch 实体（IEntity）的 Go 等价：
// 8 命令分发（ProcessSockMsg 0x77170）+ 三密码 + 响应面。
type GameWatch struct {
	mu   sync.Mutex
	cfg  *Config
	logf func(string, ...any)

	state      WatchState
	query      *QueryService
	procs      *ProcManager
	sampler    *Sampler
	updates    map[string]*UpdateSession // set_file 分块拼装（server\file）
	sendTo     func(cnx int, l *VarList) bool
	commandDial func(addr string) (net.Conn, error) // command 子命令 UserSock 面（测试注入）
}

// NewGameWatch 装配官方 main.lua 注入序：DBSystem/AppManager/CPURate。
func NewGameWatch(cfg *Config, backend QueryBackend, logf func(string, ...any)) *GameWatch {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	g := &GameWatch{
		cfg:     cfg,
		logf:    logf,
		updates: map[string]*UpdateSession{},
	}
	g.query = NewQueryService(backend, logf)
	g.procs = NewProcManager(cfg, logf)
	g.commandDial = func(addr string) (net.Conn, error) { return net.Dial("tcp", addr) }
	return g
}

// Start 官方 GameWatch::Start（0x6e040）：CTickTimer 初始化 + CCpuRate
// 初始化 + CThread::Start + state=1（Opened）。
func (g *GameWatch) Start() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sampler = NewSampler()
	g.state = StateOpened
	g.logf("Game watch server started")
}

// Stop 官方 GameWatch::Stop（0x6e1d0）：state=3（Closed）+ 采样/查询停机。
func (g *GameWatch) Stop() {
	g.mu.Lock()
	sampler := g.sampler
	g.state = StateClosed
	g.mu.Unlock()
	if sampler != nil {
		sampler.Stop()
	}
	g.query.Stop()
}

// State 官方 GetState 面。
func (g *GameWatch) State() WatchState {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.state
}

// SetSender 注入 MassSock 发送面（CMassSockHook::SendCustomMsg 等价）。
func (g *GameWatch) SetSender(fn func(cnx int, l *VarList) bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sendTo = fn
}

// SetCommandDial 注入 command 子命令的 UserSock 拨号面（测试用）。
func (g *GameWatch) SetCommandDial(fn func(addr string) (net.Conn, error)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.commandDial = fn
}

func (g *GameWatch) send(cnx int, l *VarList) bool {
	g.mu.Lock()
	fn := g.sendTo
	g.mu.Unlock()
	if fn == nil {
		return false
	}
	return fn(cnx, l)
}

// receiveStartApp 官方 0x7c540：args[1]=server args[2]=密码。
func (g *GameWatch) receiveStartApp(cnx int, args *VarList) {
	server := args.StringAt(1)
	pwd := args.StringAt(2)
	if !checkPwd(pwd, g.cfg.MonitorPassword) {
		g.SendResultMsg(cnx, "start_app", "error_password")
		return
	}
	g.SendResultMsg(cnx, "start_app", g.procs.StartApp(server))
}

// receiveStopApp 官方 0x7c900：args[1]=server args[2]=密码。
func (g *GameWatch) receiveStopApp(cnx int, args *VarList) {
	server := args.StringAt(1)
	pwd := args.StringAt(2)
	if !checkPwd(pwd, g.cfg.MonitorPassword) {
		g.SendResultMsg(cnx, "stop_app", "error_password")
		return
	}
	g.SendResultMsg(cnx, "stop_app", g.procs.StopApp(server))
}

// SendResultMsg 官方 0x6f460：varlist ["result", cmd, result] → SendCustomMsg。
func (g *GameWatch) SendResultMsg(cnx int, cmd, result string) {
	l := &VarList{}
	l.AddString("result")
	l.AddString(cmd)
	l.AddString(result)
	g.send(cnx, l)
}

// ProcessLine 官方 CMassSockHook::ReceiveMsg + GameWatch::ProcessSockMsg
// 合一入口：一行文本（已去 \r\n）→ 响应。返回 false = 帧解码失败
// （官方日志 "(GameWatch::ProcessSockMsg)decode failed"）。
func (g *GameWatch) ProcessLine(cnx int, line string) bool {
	// 官方 ReceiveMsg 先把 <0x20 控制字符替换为 \0 分段；行受面已按
	// \r\n 切行，这里再按残余控制字符兜底分段逐条处理。
	for _, seg := range strings.FieldsFunc(line, func(r rune) bool { return r < 0x20 }) {
		g.processSockMsg(cnx, seg)
	}
	return true
}

// processSockMsg 官方 ProcessSockMsg（0x77170）。
func (g *GameWatch) processSockMsg(cnx int, msg string) {
	args, ok := DecodeText(msg)
	if !ok {
		g.logf("(GameWatch::ProcessSockMsg)decode failed")
		g.SendResultMsg(cnx, "result", "error_request")
		return
	}
	if args.Count() == 0 {
		g.logf("(GameWatch::ProcessSockMsg)no message type")
		g.SendResultMsg(cnx, "result", "error_request")
		return
	}
	cmd := args.StringAt(0)
	switch strings.ToLower(cmd) {
	case "servers":
		g.receiveServers(cnx)
	case "app_list":
		g.send(cnx, g.procs.AppList())
	case "query":
		g.receiveQuery(cnx, args)
	case "update":
		g.receiveUpdate(cnx, args)
	case "sys_info":
		g.receiveSysInfo(cnx)
	case "command":
		g.receiveCommand(cnx, args)
	case "start_app":
		g.receiveStartApp(cnx, args)
	case "stop_app":
		g.receiveStopApp(cnx, args)
	default:
		g.logf("(GameWatch::ProcessSockMsg)message %s not processed", cmd)
		g.SendResultMsg(cnx, "result", "error_request")
	}
}

// receiveServers 官方 0x77630：["servers", name, cmdline, ...]。
func (g *GameWatch) receiveServers(cnx int) {
	l := &VarList{}
	l.AddString("servers")
	for _, s := range g.cfg.Servers {
		l.AddString(s.Name)
		l.AddString(s.Path)
	}
	g.send(cnx, l)
}

// checkPwd 官方三密码 stricmp 面（_stricmp 明文比对，大小写不敏感）。
func checkPwd(got, want string) bool { return strings.EqualFold(got, want) }

// receiveQuery 官方 0x77b60：args[1]=子命令 args[2]=密码 args[3..]=参数。
func (g *GameWatch) receiveQuery(cnx int, args *VarList) {
	sub := args.StringAt(1)
	pwd := args.StringAt(2)
	if !checkPwd(pwd, g.cfg.QueryPassword) {
		g.logf("(GameWatch::ProcessQuery)error password")
		g.SendResultMsg(cnx, "query", "error_password")
		return
	}
	msgID, ok := queryCommands[strings.ToLower(sub)]
	if !ok {
		g.SendResultMsg(cnx, "query", "error_request")
		return
	}
	rest := make([]string, 0, 4)
	for i := 3; i < args.Count(); i++ {
		rest = append(rest, args.StringAt(i))
	}
	// request_log_data 的时间参数校验面（'error_time'）：首参须为可解析时间。
	if msgID == MsgRequestLogData && len(rest) > 0 {
		if _, err := parseLogTime(rest[0]); err != nil {
			g.SendResultMsg(cnx, "query", "error_time")
			return
		}
	}
	ok2 := g.query.Put(msgID, sub, rest, cnx, func(res *QueryResult) {
		g.publish(cnx, res)
	})
	if !ok2 {
		g.SendResultMsg(cnx, "query", "failed")
		return
	}
	if len(g.cfg.Access) == 0 && isDefaultBackend(g.query) {
		// 官方无数据源面：no_database 立即回（不等线程）。
		g.SendResultMsg(cnx, "query", "no_database")
	}
}

// publish 官方 ProcessDbMsg（0x747c0）→ PublishXxx：行式回推
// ["<publish>", 行字段...]，错误回 result query no_database/failed。
func (g *GameWatch) publish(cnx int, res *QueryResult) {
	if res.Err != nil {
		msg := "failed"
		if res.Err.Error() == "no_database" {
			msg = "no_database"
		}
		g.SendResultMsg(cnx, "query", msg)
		return
	}
	l := &VarList{}
	l.AddString(res.Publish)
	for _, row := range res.Rows {
		for _, cell := range row {
			l.AddString(cell)
		}
	}
	g.send(cnx, l)
}

func isDefaultBackend(q *QueryService) bool {
	_, ok := q.backend.(LocalQueryBackend)
	return ok
}

// parseLogTime 官方 request_log_data 时间参数（'error_time' 校验面）。
func parseLogTime(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix(), nil
		}
	}
	return 0, fmt.Errorf("bad time %q", s)
}

// receiveSysInfo 官方 0x7bb30（无密码）：["sys_info", cpu, load, total, avail]。
func (g *GameWatch) receiveSysInfo(cnx int) {
	g.mu.Lock()
	sampler := g.sampler
	g.mu.Unlock()
	var si SysInfo
	if sampler != nil {
		si = sampler.Info()
	}
	l := &VarList{}
	l.AddString("sys_info")
	l.AddFloat(si.CPURate)
	l.AddFloat(si.MemLoad)
	l.AddFloat(si.TotalPhy)
	l.AddFloat(si.AvailPhy)
	g.send(cnx, l)
}

// receiveCommand 官方 0x7be80：args[1]=addr:port args[2]=密码 args[3]=命令行。
func (g *GameWatch) receiveCommand(cnx int, args *VarList) {
	ap := args.StringAt(1)
	pwd := args.StringAt(2)
	line := args.StringAt(3)
	if !checkPwd(pwd, g.cfg.MonitorPassword) {
		g.SendResultMsg(cnx, "command", "error_password")
		return
	}
	i := strings.LastIndexByte(ap, ':')
	if i <= 0 || i == len(ap)-1 {
		g.SendResultMsg(cnx, "command", "error_address")
		return
	}
	host := ap[:i]
	port, err := strconv.Atoi(ap[i+1:])
	if err != nil {
		g.SendResultMsg(cnx, "command", "error_address")
		return
	}
	g.mu.Lock()
	dial := g.commandDial
	g.mu.Unlock()
	conn, err := dial(net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		g.SendResultMsg(cnx, "command", "error_create_sock")
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(line + "\r\n")); err != nil {
		g.SendResultMsg(cnx, "command", "failed")
		return
	}
	g.SendResultMsg(cnx, "command", "succeed")
}

// receiveUpdate 官方 0x79170：args[1]=子命令 args[2]=密码。
func (g *GameWatch) receiveUpdate(cnx int, args *VarList) {
	sub := strings.ToLower(args.StringAt(1))
	pwd := args.StringAt(2)
	if !checkPwd(pwd, g.cfg.UpdatePassword) {
		g.SendResultMsg(cnx, "update", "error_password")
		return
	}
	switch sub {
	case "get_dir":
		g.updateGetDir(cnx, args)
	case "get_file":
		g.updateGetFile(cnx, args)
	case "set_dir":
		g.updateSetDir(cnx, args)
	case "set_file":
		g.updateSetFile(cnx, args)
	default:
		g.SendResultMsg(cnx, "update", "error_request")
	}
}

// updateGetDir 官方 get_dir：ReadDirInfo → StoreArchive → SendDirData 分块。
func (g *GameWatch) updateGetDir(cnx int, args *VarList) {
	server := args.StringAt(3)
	rel := args.StringAt(4)
	root, ok := g.cfg.DestPath(server)
	if !ok {
		g.SendResultMsg(cnx, "update", "no dest path")
		return
	}
	dir, err := ReadDirInfo(root, rel)
	if err != nil {
		g.logf("(CWatch::ProcessUpdate)refresh dir info failed: %v", err)
		g.SendResultMsg(cnx, "update", "refresh dir info failed")
		return
	}
	head := []string{"update", "get_dir", server, rel}
	if !sendBufferBlocks(head, dir.StoreArchive(), func(l *VarList) bool { return g.send(cnx, l) }) {
		g.logf("(CWatch::ProcessUpdate)send dir data failed")
		g.SendResultMsg(cnx, "update", "send dir data error")
	}
}

// updateGetFile 官方 get_file：64KB 分块下发。
func (g *GameWatch) updateGetFile(cnx int, args *VarList) {
	server := args.StringAt(3)
	file := args.StringAt(4)
	root, ok := g.cfg.DestPath(server)
	if !ok {
		g.SendResultMsg(cnx, "update", "no dest path")
		return
	}
	full := filepath.Join(root, filepath.FromSlash(file))
	f, err := os.Open(full)
	if err != nil {
		g.SendResultMsg(cnx, "update", "file data size error")
		return
	}
	defer f.Close()
	head := []string{"update", "get_file", server, file}
	if !sendFileBlocks(head, f, func(l *VarList) bool { return g.send(cnx, l) }) {
		g.logf("(CWatch::ProcessUpdate)send file failed")
		g.SendResultMsg(cnx, "update", "send file error")
	}
}

// updateSetDir 官方 set_dir：root path error / dir already exists /
// create dir failed → succeed。
func (g *GameWatch) updateSetDir(cnx int, args *VarList) {
	server := args.StringAt(3)
	rel := args.StringAt(4)
	root, ok := g.cfg.DestPath(server)
	if !ok || root == "" {
		g.SendResultMsg(cnx, "update", "root path error")
		return
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	if st, err := os.Stat(full); err == nil {
		if st.IsDir() {
			g.SendResultMsg(cnx, "update", "dir already exists")
			return
		}
		g.SendResultMsg(cnx, "update", "root path error")
		return
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		g.SendResultMsg(cnx, "update", "create dir failed")
		return
	}
	g.SendResultMsg(cnx, "update", "succeed")
}

// updateSetFile 官方 set_file：args[3]=server args[4]=file args[5]=size
// args[6]=time args[7]=md5 args[8]=*data 分块（可多帧续传）。
func (g *GameWatch) updateSetFile(cnx int, args *VarList) {
	server := args.StringAt(3)
	file := args.StringAt(4)
	size := args.IntAt(5)
	mtime := args.IntAt(6)
	md5hex := args.StringAt(7)
	data := args.DataAt(8)
	if size <= 0 || file == "" {
		g.SendResultMsg(cnx, "update", "file data size error")
		return
	}
	root, ok := g.cfg.DestPath(server)
	if !ok {
		g.SendResultMsg(cnx, "update", "no dest path")
		return
	}
	g.mu.Lock()
	key := server + "\\" + file
	s := g.updates[key]
	if s == nil || s.file != filepath.FromSlash(file) || s.size != size {
		s = &UpdateSession{root: root, file: filepath.FromSlash(file), size: size, modTime: mtime, md5: md5hex}
		g.updates[key] = s
	}
	s.feed(int64(len(s.buf)), data) // 官方按 offset 拼装；行受面按到达序累计
	g.mu.Unlock()
	if !s.complete() {
		// 官方续传语义：未齐 size 前静默收块（'file cache error' 在缓存
		// 失配时出现）。
		g.SendResultMsg(cnx, "update", "continue")
		return
	}
	g.mu.Lock()
	delete(g.updates, key)
	g.mu.Unlock()
	res, _ := s.flush(root)
	g.SendResultMsg(cnx, "update", res)
}
