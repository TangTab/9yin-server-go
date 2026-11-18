// lister 是区服列表分发进程（对应原版 game_lister.exe + FsSoloLister.dll），
// 承载 internal/login 的 svrlist ASCII 行协议服务：客户端连上发
// "svrlist"，回区服条目（编码规格 = 官方 text_encode_msg 反汇编终局：
// `$` 窄串 / `#` 宽串逐码元转义 / 空格分隔 / 多服连排 / busy 繁忙度）。
//
// 替代旧分支的 server-cmd/loopback-lister.ps1（方言 golden 见
// testdata/captures/svrlist-{request,response}.bin）。solo 形态下本服务
// 内嵌在 cmd/solo（-svrlist），本入口用于独立部署/多服分发。
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

	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// defaultGameListerINI 是内置默认配置（小写系 game_lister.ini 方言，
// 对齐官方九阴 solo/FsSoloLister 体系）：未指定 -config 且当前目录没有
// game_lister.ini 时释放到当前目录并采用（已存在的绝不覆盖）。
//
//go:embed game_lister.ini
var defaultGameListerINI []byte

const version = "0.2.0-lister"

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:9601", "svrlist 监听地址（显式给出时优先于配置 [Outer]）")
		name       = flag.String("name", "本地测试", "下发的服务器显示名")
		host       = flag.String("host", "", "通告的游戏服地址（空=监听 host，通配回退 127.0.0.1）")
		port       = flag.Int("port", 9600, "通告的游戏服端口")
		busy       = flag.Int("busy", 0, "繁忙度尾标（官方公式：推荐>10 直通，否则 cur*5/max）")
		minVersion = flag.Int("min-version", 0, "最小客户端版本门（0=不设门；>0 须 \"svrlist <ver>\" 达标，否则回 $version_error）")
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
	fmt.Printf("已加载配置：IDC=%d DistrictID=%d ListerID=%d MaxPlayer=%d SockThreads=%d（MaxPlayer/SockThreads 单服部署未接线）\n",
		cfg.IDC, cfg.DistrictID, cfg.ListerID, cfg.MaxPlayer, cfg.SockThreads)

	l := &login.Lister{
		Addr:             listen,
		Entry:            login.ResolveEntryHost(entry, listen),
		Extra:            extras,
		MinClientVersion: gate,
		Logf:             func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	listenAddr, err := l.Listen()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("svrlist 就绪：%s → 通告 %s:%d busy=%d（%s）+%d 追加条目\n",
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
