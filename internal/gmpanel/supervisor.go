package gmpanel

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// svcSpec 单服描述：探活口（echo 维护面或业务口）+ 优雅停命令。
type svcSpec struct {
	Name      string   // 面板标识（log/world/store/…/member0..N-1）
	Exe       string   // game_*.exe
	Args      []string // 启动参数（member：-member N）
	Title     string   // 中文说明
	ProbeHost string   // 探活口地址（默认 127.0.0.1）
	ProbePort int      // 探活口（echo 维护口优先；lister 用客户端口 4000）
	StopEcho  string   // 优雅停命令（发到探活口的 echo 方言），空 = 只能强杀
}

// stopSequence 停止序（对齐 stop-cluster.bat / 官方 kill.bat 逆序）。
var stopSequence = []string{"member", "entry", "lister", "public", "store", "world", "log", "auction", "monitor"}

// clusterSpecs 集群清单（序 = start-cluster.bat 启动序）。优雅停面：
// log/world/store/public/entry/monitor echo 支持 quit；lister/member/
// auction 无（降级强杀）。
func clusterSpecs(members int) []*svcSpec {
	specs := []*svcSpec{
		{Name: "log", Exe: "game_logserver.exe", Title: "日志聚合服", ProbePort: 15011, StopEcho: "quit"},
		{Name: "world", Exe: "game_soloworld.exe", Title: "世界服", ProbePort: 15001, StopEcho: "quit"},
		{Name: "store", Exe: "game_store.exe", Title: "存储枢纽（MySQL）", ProbePort: 15100, StopEcho: "quit"},
		{Name: "public", Exe: "game_solopublic.exe", Title: "公共数据服", ProbePort: 15003, StopEcho: "quit"},
		{Name: "lister", Exe: "game_sololister.exe", Title: "区服列表（客户端入口 4000）", ProbePort: 4000},
		{Name: "entry", Exe: "game_soloentry.exe", Title: "接入网关（Outer 2001）", ProbePort: 15200, StopEcho: "quit"},
	}
	for i := 0; i < members; i++ {
		specs = append(specs, &svcSpec{
			Name: fmt.Sprintf("member%d", i), Exe: "game_solomember.exe",
			Args: []string{"-member", fmt.Sprint(i)}, Title: fmt.Sprintf("场景服 %d（echo %d）", i, 15300+i),
			ProbePort: 15300 + i,
		})
	}
	specs = append(specs,
		&svcSpec{Name: "auction", Exe: "game_auction.exe", Title: "拍卖行", ProbePort: 16001},
		&svcSpec{Name: "monitor", Exe: "game_monitor.exe", Title: "监控服（Outer 4008）", ProbePort: 15012, StopEcho: "quit"},
	)
	return specs
}

// singleSpecs 合一形态：solounite 内嵌七实体（echo 复用 world 15001 命令
// 族，含 quit）+ sidecar lister（start-single.bat WITHLISTER=1 形态）。
func singleSpecs() []*svcSpec {
	return []*svcSpec{
		{Name: "world", Exe: "game_solounite.exe", Title: "合一服 solounite（七实体）", ProbePort: 15001, StopEcho: "quit"},
		{Name: "lister", Exe: "game_sololister.exe", Title: "区服列表（客户端入口 4000）", ProbePort: 4000},
	}
}

// SvcState 面板视角的单服状态。
type SvcState struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	Exe       string `json:"exe"`
	Args      string `json:"args"`
	ProbePort int    `json:"probePort"`
	State     string `json:"state"`    // running（托管）/ external（观察）/ stopped
	PID       int    `json:"pid"`      // 托管子进程 PID（外部进程不追溯）
	CanGrace  bool   `json:"canGrace"` // 是否支持 echo quit 优雅停
}

// managedProc 托管子进程（含 stdout/stderr 落盘句柄）。
type managedProc struct {
	cmd  *exec.Cmd
	log  *os.File
	done chan struct{}
}

// Supervisor 进程编排：托管（exec.Command + 日志重定向）与观察（端口探活）
// 双模式。并发安全；面板退出不杀托管进程（Close 只关日志句柄）。
type Supervisor struct {
	mu       sync.Mutex
	cfg      Config
	logf     func(string, ...any)
	specs    []*svcSpec
	procs    map[string]*managedProc
	lastExit map[string]string // name → 上次退出摘记（面板日志口径）
}

// NewSupervisor 按模式装配清单；LogDir 空则落 binDir/logs（对齐 bat
// /d "%BIN%" 的「ini/日志落二进制同侧」语义）。
func NewSupervisor(cfg Config) *Supervisor {
	var specs []*svcSpec
	if cfg.Mode == "single" {
		specs = singleSpecs()
	} else {
		specs = clusterSpecs(cfg.Members)
	}
	for _, sp := range specs {
		if sp.ProbeHost == "" {
			sp.ProbeHost = "127.0.0.1"
		}
	}
	if cfg.WorldPort > 0 {
		if sp := specByName(specs, "world"); sp != nil {
			sp.ProbePort = cfg.WorldPort // 测试注入动态口
		}
	}
	if cfg.LogDir == "" && cfg.BinDir != "" {
		cfg.LogDir = filepath.Join(cfg.BinDir, "logs")
	}
	if cfg.BinDir != "" {
		if abs, err := filepath.Abs(cfg.BinDir); err == nil {
			cfg.BinDir = abs // exec.Command 相对 exe + cmd.Dir 在 CreateProcess 下解析不一致，统一绝对路径
		}
		if cfg.LogDir != "" {
			if abs, err := filepath.Abs(cfg.LogDir); err == nil {
				cfg.LogDir = abs
			}
		}
	}
	return &Supervisor{cfg: cfg, logf: cfg.logger(), specs: specs,
		procs: map[string]*managedProc{}, lastExit: map[string]string{}}
}

// Specs 启动序清单。
func (s *Supervisor) Specs() []*svcSpec { return s.specs }

// Spec 按名取（无则 nil）。
func (s *Supervisor) Spec(name string) *svcSpec {
	return specByName(s.specs, name)
}

func specByName(specs []*svcSpec, name string) *svcSpec {
	for _, sp := range specs {
		if sp.Name == name {
			return sp
		}
	}
	return nil
}

// probe 端口探活（echo 口拨通即挂断，1s 超时）。
func (sp *svcSpec) probe() bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", sp.ProbeHost, sp.ProbePort), time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// StatusAll 全量状态（启动序）。
func (s *Supervisor) StatusAll() []SvcState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SvcState, 0, len(s.specs))
	for _, sp := range s.specs {
		st := SvcState{Name: sp.Name, Title: sp.Title, Exe: sp.Exe,
			Args: strings.Join(sp.Args, " "), ProbePort: sp.ProbePort, CanGrace: sp.StopEcho != ""}
		if p := s.procs[sp.Name]; p != nil && procAlive(p) {
			st.State, st.PID = "running", p.cmd.Process.Pid
		} else if sp.probe() {
			st.State = "external"
		} else {
			st.State = "stopped"
		}
		out = append(out, st)
	}
	return out
}

// Start 托管拉起单服（stdout/stderr → logs/<name>-<YYYYMMDD>.log 追加）。
// 已在跑（托管/外部）返回提示不重复拉。
func (s *Supervisor) Start(name string) (string, error) {
	sp := s.Spec(name)
	if sp == nil {
		return "", fmt.Errorf("unknown service %q", name)
	}
	s.mu.Lock()
	if p := s.procs[name]; p != nil && procAlive(p) {
		s.mu.Unlock()
		return fmt.Sprintf("%s 已在运行（托管 PID %d）", name, p.cmd.Process.Pid), nil
	}
	s.mu.Unlock()
	if sp.probe() {
		return fmt.Sprintf("%s 已在运行（外部进程，观察模式）", name), nil
	}
	if s.cfg.BinDir == "" {
		return "", fmt.Errorf("bin 目录未定位（build 后再启动面板）")
	}
	exePath := filepath.Join(s.cfg.BinDir, sp.Exe)
	if _, err := os.Stat(exePath); err != nil {
		return "", fmt.Errorf("找不到 %s", exePath)
	}
	if err := os.MkdirAll(s.cfg.LogDir, 0o755); err != nil {
		return "", err
	}
	logPath := filepath.Join(s.cfg.LogDir, fmt.Sprintf("%s-%s.log", name, time.Now().Format("20060102")))
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(lf, "---- %s started %s ----\r\n", name, time.Now().Format("2006-01-02 15:04:05"))
	cmd := exec.Command(exePath, sp.Args...)
	cmd.Dir = s.cfg.BinDir // 对齐 bat /d：ini 释放/数据落二进制同侧
	cmd.Stdout, cmd.Stderr = lf, lf
	if err := cmd.Start(); err != nil {
		lf.Close()
		return "", err
	}
	done := make(chan struct{})
	go func() {
		err := cmd.Wait()
		close(done)
		lf.Close()
		s.mu.Lock()
		if s.procs[name] != nil && s.procs[name].cmd == cmd {
			delete(s.procs, name)
			s.lastExit[name] = fmt.Sprintf("exit %v", err)
		}
		s.mu.Unlock()
		s.logf("gmpanel: %s 退出（%v）", name, err)
	}()
	s.mu.Lock()
	s.procs[name] = &managedProc{cmd: cmd, log: lf, done: done}
	s.mu.Unlock()
	return fmt.Sprintf("%s 已启动（PID %d，日志 %s）", name, cmd.Process.Pid, logPath), nil
}

// Stop 停单服。梯度：托管 → echo quit 等 3s → Kill；外部 → echo quit 等
// 3s → netstat 解析端口 PID → taskkill /PID（失败回落 /IM，member 多实例
// 会全杀，消息里注明）。
func (s *Supervisor) Stop(name string) (string, error) {
	sp := s.Spec(name)
	if sp == nil {
		return "", fmt.Errorf("unknown service %q", name)
	}
	s.mu.Lock()
	p := s.procs[name]
	s.mu.Unlock()
	if p == nil && !sp.probe() {
		return fmt.Sprintf("%s 未在运行", name), nil
	}
	// ① 优雅：echo quit（两模式同享）
	if sp.StopEcho != "" {
		c := NewEchoCli(sp.ProbeHost, sp.ProbePort)
		_, _ = c.Command(sp.StopEcho)
		if waitStopped(sp, p, 3*time.Second) {
			return fmt.Sprintf("%s 已优雅停止（echo %s）", name, sp.StopEcho), nil
		}
	}
	// ② 托管强杀
	if p != nil {
		if err := p.cmd.Process.Kill(); err != nil && procAlive(p) {
			return "", fmt.Errorf("kill %s: %w", name, err)
		}
		waitDone(p.done, 3*time.Second)
		return fmt.Sprintf("%s 已强制停止（无优雅停面或超时）", name), nil
	}
	// ③ 外部强杀：端口 → PID → taskkill
	if pid, ok := portOwnerPID(sp.ProbePort); ok {
		if out, err := exec.Command("taskkill", "/F", "/PID", fmt.Sprint(pid)).CombinedOutput(); err == nil {
			return fmt.Sprintf("%s 已停止（外部进程 taskkill /PID %d）", name, pid), nil
		} else {
			s.logf("gmpanel: taskkill /PID %d 失败: %v: %s", pid, err, strings.TrimSpace(string(out)))
		}
	}
	if out, err := exec.Command("taskkill", "/F", "/IM", sp.Exe).CombinedOutput(); err != nil {
		return "", fmt.Errorf("taskkill %s 失败: %v: %s", sp.Exe, err, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("%s 已停止（外部进程 taskkill /IM %s；member 多实例会一并停止）", name, sp.Exe), nil
}

// procAlive 托管进程是否仍在跑（done chan 非阻塞读，无竞态——
// ProcessState 由 Wait goroutine 写，读取侧一律走 chan）。
func procAlive(p *managedProc) bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// waitStopped 等待服务停：托管走 done chan（ProcessState 由 Wait
// goroutine 写，等待侧不得裸读）；外部看探活消失。
func waitStopped(sp *svcSpec, p *managedProc, d time.Duration) bool {
	if p != nil {
		return waitDone(p.done, d)
	}
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if !sp.probe() {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return !sp.probe()
}

func waitDone(done chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-done:
		return true
	case <-t.C:
		return false
	}
}

// Restart 停 → 半秒 → 起。
func (s *Supervisor) Restart(name string) (string, error) {
	msg, err := s.Stop(name)
	if err != nil {
		return msg, err
	}
	time.Sleep(500 * time.Millisecond)
	msg2, err := s.Start(name)
	return msg + "；" + msg2, err
}

// StartAll 按启动序全拉（1s 间隔对齐 bat timeout /t 1，保证依赖序）。
func (s *Supervisor) StartAll() []string {
	var out []string
	for _, sp := range s.specs {
		msg, err := s.Start(sp.Name)
		if err != nil {
			msg = "✘ " + sp.Name + ": " + err.Error()
			out = append(out, msg)
			break // 依赖序断裂即止（后续服起不来白拉）
		}
		out = append(out, "✔ "+msg)
		time.Sleep(time.Second)
	}
	return out
}

// StopAll 按 stop-cluster.bat 逆序全停。
func (s *Supervisor) StopAll() []string {
	// 停止序以 stopSequence 的段名排序（member* 归 member 段）
	rank := func(name string) int {
		for i, seg := range stopSequence {
			if seg == "member" {
				if strings.HasPrefix(name, "member") {
					return i
				}
				continue
			}
			if name == seg {
				return i
			}
		}
		return len(stopSequence)
	}
	specs := append([]*svcSpec(nil), s.specs...)
	sort.SliceStable(specs, func(i, j int) bool { return rank(specs[i].Name) < rank(specs[j].Name) })
	var out []string
	for _, sp := range specs {
		msg, err := s.Stop(sp.Name)
		if err != nil {
			msg = "✘ " + sp.Name + ": " + err.Error()
		} else {
			msg = "✔ " + msg
		}
		out = append(out, msg)
		time.Sleep(300 * time.Millisecond)
	}
	return out
}

// LatestLog 最新日志文件路径（无则空）。
func (s *Supervisor) LatestLog(name string) string {
	pattern := filepath.Join(s.cfg.LogDir, name+"-*.log")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches) // YYYYMMDD 字典序 = 时间序
	return matches[len(matches)-1]
}

// Close 收尾：只关托管进程的日志句柄（不杀进程——面板退出后子进程
// 存活，重启面板转观察模式续管）。
func (s *Supervisor) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, p := range s.procs {
		if p.log != nil {
			p.log.Close()
		}
		delete(s.procs, name)
	}
}

// portOwnerPID netstat -ano 解析监听端口的 PID（Windows；失败 false，
// 调用方回落 /IM）。本地地址列 host 任意、端口须全等（:1500 不得命中
// :15001），LISTENING 态。
func portOwnerPID(port int) (int, bool) {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return 0, false
	}
	want := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) < 5 || f[3] != "LISTENING" {
			continue
		}
		addr := f[1]
		colon := strings.LastIndex(addr, ":")
		if colon < 0 || addr[colon:] != want {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(f[len(f)-1], "%d", &pid); err == nil && pid > 0 {
			return pid, true
		}
	}
	return 0, false
}
