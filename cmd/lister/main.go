// lister 是区服列表分发进程，对应原版两个同源组件（BGserverCJ 双体系）：
//
//   - 静态形态（默认，game_lister.exe + FsSoloLister.dll 小写系）：
//     internal/login 的 svrlist ASCII 行协议——静态条目 + 可选版本门
//   - 集群形态（-broadcast 或 ini [Broadcast] Port，GameLister.bat +
//     FsLister.dll 大写系）：internal/fslister——UDP 双心跳注册面
//     （world 建服/entry 建线，5s 扫描除名）+ 客户端机（60s 请求超时/
//     每连接一答/5s 温和关闭）+ 选线（最小 cur/max 负载率）与
//     busy 聚合（服级 cur*5/max；无版本门——官方 FsLister 无此属性）
//
// 方言编码规格 = 官方 text_encode_msg 反汇编终局（两系同构）：
// `$` 窄串 / `#` 宽串逐码元转义 / 空格分隔 / 多服连排 / busy 繁忙度。
// golden 见 testdata/captures/svrlist-{request,response}.bin。
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	gonet "net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fslister"
	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// defaultGameListerINI 是内置默认配置（小写系 game_lister.ini 方言，
// 对齐官方九阴 solo/FsSoloLister 体系）：未指定 -config 且当前目录没有
// game_lister.ini 时释放到当前目录并采用（已存在的绝不覆盖）。
//
//go:embed game_lister.ini
var defaultGameListerINI []byte

const version = "0.3.0-lister"

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:9601", "svrlist 监听地址（显式给出时优先于配置 [Outer]）")
		name       = flag.String("name", "本地测试", "下发的服务器显示名")
		host       = flag.String("host", "", "通告的游戏服地址（空=监听 host，通配回退 127.0.0.1）")
		port       = flag.Int("port", 9600, "通告的游戏服端口")
		busy       = flag.Int("busy", 0, "繁忙度尾标（静态形态公式：推荐>10 直通，否则 cur*5/max；集群形态忽略——按注册表实时计算）")
		minVersion = flag.Int("min-version", 0, "最小客户端版本门（静态形态；0=不设门；集群形态无此门——官方 FsLister 无 ClientVersion 属性）")
		broadcast  = flag.String("broadcast", "", "集群注册面（FsLister 形态）：UDP 绑定口（\"15000\" 或 \"addr:port\"，显式给出即启用；off 强制关闭=静态形态）。默认取配置 [Broadcast] Port，缺省=静态形态")
		idcFlag    = flag.Int("idc", 0, "网络接入商编号 IDC（集群形态选线过滤：>0 时仅下发 line_id==IDC 的线；静态形态仅呈现）")
	)
	var extras extraFlags
	flag.Var(&extras, "extra", "追加区服条目 name=host:port[=busy]，可重复（官方多服连排方言）")
	config := flag.String("config", "", "配置文件路径（默认：当前目录 game_lister.ini，没有则释放内置默认并采用；大写 Game*.ini 黑金系/小写 game_*.ini 九阴系方言均兼容）；监听/通告取 [Outer]、版本门取 [Main] ClientVersion；显式旗标优先")
	flag.Parse()

	fmt.Printf("9yin-server-go lister %s\n", version)

	// 配置装载：显式 -config 须存在（缺文件报错）；默认路径当前目录
	// game_lister.ini（小写系官方命名），不存在则释放内置默认
	//（已存在的绝不覆盖）。
	var cfgData []byte
	released := false
	if *config != "" {
		data, err := os.ReadFile(*config)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		data, created, err := login.LoadOrCreateGameListerINI("game_lister.ini", defaultGameListerINI)
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := login.ParseGameListerINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_lister.ini，已释放内置默认配置（修改后重启生效）")
	}

	// 显式旗标优先于配置：记录哪些旗标被用户显式设置。
	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })

	// 集群形态判定：-broadcast 显式给出（off=强制关）> 配置 [Broadcast] Port
	broadPort := 0
	if given["broadcast"] {
		if p, ok := parseBroadcast(*broadcast); ok {
			broadPort = p
		}
	} else if cfg.BroadcastPort > 0 {
		broadPort = cfg.BroadcastPort
	}

	entry := login.ServerEntry{Name: *name, Host: *host, Port: *port, Busy: *busy}
	listen, gate := *addr, *minVersion
	if !given["addr"] {
		listen = gonet.JoinHostPort(cfg.OuterAddr, strconv.Itoa(cfg.OuterPort))
	}
	if !given["host"] {
		entry.Host = cfg.OuterAddr
	}
	if !given["port"] {
		entry.Port = cfg.OuterPort
	}
	if !given["min-version"] {
		gate = cfg.ClientVersion
	}
	idc, district, listerID := cfg.IDC, cfg.DistrictID, cfg.ListerID
	if given["idc"] {
		idc = *idcFlag
	}
	fmt.Printf("已加载配置：IDC=%d DistrictID=%d ListerID=%d MaxPlayer=%d SockThreads=%d\n",
		idc, district, listerID, cfg.MaxPlayer, cfg.SockThreads)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if broadPort > 0 {
		runCluster(ctx, cfg, broadPort, listen, idc, district, listerID, entry, extras, given)
		return
	}

	entry = login.ResolveEntryHost(entry, listen)
	l := &login.Lister{
		Addr:             listen,
		Entry:            entry,
		Extra:            extras,
		MinClientVersion: gate,
		Logf:             func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}

	listenAddr, err := l.Listen()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("svrlist 就绪（静态形态）：%s → 通告 %s:%d busy=%d（%s）+%d 追加条目\n",
		listenAddr, l.Entry.Host, l.Entry.Port, l.Entry.Busy, l.Entry.Name, len(l.Extra))

	done := make(chan struct{})
	go func() { l.Serve(); close(done) }()

	<-ctx.Done()
	if err := l.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "lister:", err)
	}
	<-done
	fmt.Println("lister: 已停止")
}

// runCluster 是 FsLister（大写系）集群形态：UDP 注册面 + 官方客户端机
// + 5s 扫描主循环。静态条目旗标若被显式给出，则合成为本进程的注册
// 心跳（等价一个每 3s 重发的 world/entry 进程），保持独立部署可用。
func runCluster(ctx context.Context, cfg login.GameListerConfig, broadPort int, listen string,
	idc, district, listerID int, entry login.ServerEntry, extras []login.ServerEntry, given map[string]bool) {
	gl := fslister.NewGameLister(idc, district, listerID, cfg.MaxPlayer)
	gl.Logf = func(format string, args ...any) { fmt.Printf("[lister] "+format+"\n", args...) }

	rs := &fslister.RegistrationServer{GL: gl, Port: broadPort}
	regAddr, err := rs.Listen()
	if err != nil {
		fatal(err)
	}
	go func() { _ = rs.Serve() }()

	cs := &fslister.ClientServer{GL: gl, Addr: listen}
	clientAddr, err := cs.Listen()
	if err != nil {
		fatal(err)
	}
	go func() { _ = cs.Serve() }()

	stopTick := fslister.ServeGameLister(gl)

	// 静态条目合成注册心跳（官方语义：注册是持续重发，3s < 5s 扫描周期）
	var seeds []login.ServerEntry
	if given["name"] || given["host"] || given["port"] {
		seeds = append(seeds, login.ResolveEntryHost(entry, listen))
	}
	seeds = append(seeds, extras...)
	stopSeed := make(chan struct{})
	if len(seeds) > 0 {
		go func() {
			beat := func() {
				for i, e := range seeds {
					id := int32(1000 + i)
					gl.RecvWorldInfo(fslister.WorldInfo{
						ServerID: id, IP: e.Host, Name: e.Name,
						Max: int32(cfg.MaxPlayer), Visible: true,
					})
					gl.RecvEntryInfo(fslister.EntryInfo{
						ServerID: id, LineIndex: 0, LineID: int32(idc),
						IP: e.Host, Port: int32(e.Port), Max: int32(cfg.MaxPlayer),
					})
				}
			}
			beat()
			t := time.NewTicker(3 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					beat()
				case <-stopSeed:
					return
				}
			}
		}()
	}

	fmt.Printf("svrlist 就绪（集群形态，FsLister.dll 复刻）：客户端 %s / 注册面 %s（分区 %d）\n",
		clientAddr, regAddr, district)
	fmt.Printf("客户端机：60s 请求超时·每连接一答·应答后 5s 关闭；表：256 服×16 线，5s 双心跳扫描；busy=cur*5/max 实时聚合\n")
	if len(seeds) > 0 {
		fmt.Printf("静态条目以 3s 注册心跳合成：%s（+%d 追加）\n", seeds[0].Name, len(seeds)-1)
	} else {
		fmt.Println("表初始为空：等待 world/entry 进程向注册面广播（kind/magic 0x300/分区门方言见 internal/fslister）")
	}

	<-ctx.Done()
	close(stopSeed)
	stopTick()
	_ = cs.Close()
	_ = rs.Close()
	fmt.Println("lister: 已停止")
}

// parseBroadcast 解析 -broadcast 旗标："1500x"/"15000" 或 "127.0.0.1:15000"；
// "off" 关闭；其余视为关闭并提示。
func parseBroadcast(v string) (int, bool) {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "off") {
		return 0, false
	}
	if p, err := strconv.Atoi(v); err == nil {
		return p, true
	}
	if _, portS, err := gonet.SplitHostPort(v); err == nil {
		if p, err := strconv.Atoi(portS); err == nil {
			return p, true
		}
	}
	fmt.Fprintf(os.Stderr, "lister: -broadcast %q 无法解析（用端口号或 addr:port 或 off）\n", v)
	os.Exit(1)
	return 0, false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "lister:", err)
	os.Exit(1)
}

// extraFlags 收集重复的 -extra name=host:port[=busy] 旗标。
type extraFlags []login.ServerEntry

func (e *extraFlags) String() string { return "" }

func (e *extraFlags) Set(v string) error {
	parts := strings.Split(v, "=")
	if len(parts) < 2 {
		return fmt.Errorf("-extra %q：须为 name=host:port[=busy]", v)
	}
	hostPort := parts[1]
	busy := 0
	if len(parts) == 3 {
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("-extra %q：busy 须为整数", v)
		}
		busy = n
	}
	host, portS, err := gonet.SplitHostPort(hostPort)
	if err != nil {
		return fmt.Errorf("-extra %q：host:port 解析失败: %v", v, err)
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return fmt.Errorf("-extra %q：端口须为整数", v)
	}
	*e = append(*e, login.ServerEntry{Name: parts[0], Host: host, Port: port, Busy: busy})
	return nil
}
