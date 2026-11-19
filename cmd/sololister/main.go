// sololister 是区服列表进程，对应原版 game_lister.exe（小写系直启：
// fskit.dll + fscomm.dll + fssololister.dll 三件套，MainEntity=SoloLister）。
//
// 与 cmd/lister 的关系：lister 是同一官方 ini 上的早期载体（静态形态
// login.Lister + 大写系 FsLister 集群形态 fslister）；本进程是
// FsSoloLister.dll 的**全量复刻**（internal/fssololister）——
//
//   - UDP 双口注册面：配置口（InnerAddr:BroadPort，全门：魔数 0x3001/
//     kind 精确长度 0x84·0x48/分区门）+ 本地环回中继口（InnerAddr≠
//     127.0.0.1 时建立，合法包原样转发广播目标）
//   - TCP 客户端机（:4000）：solo 行方言（\r\n 皆终结/控制符不丢/
//     0xFF 截断）+ 版本门 + 单请求 + 30s/5s 超时关闭
//   - 表生命周期：world 建服（含 recommend 繁忙度直通）、entry 建线、
//     5s 双心跳扫描除名、空槽复用
//   - IP 黑白名单（BlockOrAllow -1/0/1 + 范围表，accept 即拦）
//
// 静态条目旗标（-name/-host/-port/-extra）显式给出时合成本进程注册心跳
// （官方语义：注册是持续重发，3s < 5s 扫描周期），独立部署即可用。
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

	"github.com/lizhanyin/9yin-server-go/internal/fssololister"
	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// defaultGameListerINI 是内置默认配置（官方小写系 game_lister.ini 方言，
// 含 [Inner]/[Broadcast] 注册面接线）：未指定 -config 且当前目录没有
// game_lister.ini 时释放到当前目录并采用（已存在的绝不覆盖）。
//
//go:embed game_lister.ini
var defaultGameListerINI []byte

const version = "0.1.0-sololister"

func main() {
	var (
		addr   = flag.String("addr", "", "客户端监听地址（缺省取配置 [Outer]，官方客户端编死 4000）")
		name   = flag.String("name", "", "静态区服显示名（显式给出即合成本进程注册心跳）")
		host   = flag.String("host", "", "静态条目通告地址（空=监听 host，通配回退 127.0.0.1）")
		port   = flag.Int("port", 9600, "静态条目通告端口")
		busy   = flag.Int("busy", 0, "静态条目繁忙度推荐值（>10 时官方直通 busy）")
		config = flag.String("config", "", "配置文件路径（默认：当前目录 game_lister.ini，没有则释放内置默认并采用）")
	)
	var extras extraFlags
	flag.Var(&extras, "extra", "追加静态区服条目 name=host:port[=busy]，可重复（官方多服连排方言）")
	flag.Parse()

	fmt.Printf("9yin-server-go sololister %s（FsSoloLister.dll 复刻）\n", version)

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
	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })

	sl := fssololister.New()
	sl.IDC = int32(cfg.IDC)
	sl.DistrictID = int32(cfg.DistrictID)
	sl.SetListerID(int32(cfg.ListerID))
	sl.MaxConnects = int32(cfg.MaxPlayer)
	sl.ClientVersion = int32(cfg.ClientVersion)
	sl.InnerAddr = cfg.InnerAddr
	sl.BroadAddr = cfg.BroadcastAddr
	sl.BroadPort = cfg.BroadcastPort
	if sl.BroadPort == 0 {
		sl.BroadPort = 15000 // 官方 lua ReadInteger 缺省
	}
	if sl.BroadAddr == "" {
		sl.BroadAddr = "127.255.255.255" // 官方 lua ReadString 缺省
	}
	listen := *addr
	if listen == "" {
		host := cfg.OuterAddr
		if host == "" {
			host = "127.0.0.1"
		}
		listen = gonet.JoinHostPort(host, strconv.Itoa(cfg.OuterPort))
	}
	sl.Logf = func(format string, args ...any) { fmt.Printf("[lister] "+format+"\n", args...) }

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 注册面（UDP 双口）
	rs := &fssololister.RegistrationServer{SL: sl}
	regAddr, err := rs.Listen()
	if err != nil {
		fatal(fmt.Errorf("注册面绑定失败（Inner=%s Broad=%d）: %w", sl.InnerAddr, sl.BroadPort, err))
	}
	rs.Serve()

	// 客户端面（TCP）
	cs := &fssololister.ClientServer{SL: sl, Addr: listen}
	clientAddr, err := cs.Listen()
	if err != nil {
		fatal(err)
	}
	go func() { _ = cs.Serve() }()

	stopTick := fssololister.ServeSoloLister(sl)
	sl.MarkStarted()

	// 静态条目合成注册心跳（world 包带 busy 推荐值——官方 recommend 通道）
	var seeds []login.ServerEntry
	if given["name"] || given["host"] || given["port"] || given["busy"] {
		seeds = append(seeds, login.ResolveEntryHost(
			login.ServerEntry{Name: *name, Host: *host, Port: *port, Busy: *busy}, listen))
	}
	seeds = append(seeds, extras...)
	stopSeed := make(chan struct{})
	if len(seeds) > 0 {
		go func() {
			beat := func() {
				for i, e := range seeds {
					id := int32(2000 + i)
					sl.RecvWorldInfo(fssololister.WorldInfo{
						ServerID: id, IP: e.Host, Name: e.Name,
						Max: int32(cfg.MaxPlayer), Visible: true, Recommend: int32(e.Busy),
					})
					sl.RecvEntryInfo(fssololister.EntryInfo{
						ServerID: id, LineIndex: 0, LineID: int32(cfg.IDC),
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

	fmt.Printf("svrlist 就绪：%s（分区 %d IDC %d 版本门 %d）\n", clientAddr, sl.DistrictID, sl.IDC, sl.ClientVersion)
	fmt.Printf("注册面：%s（Inner=%s → 广播 %s:%d，魔数 0x3001/分区门/5s 双心跳）\n",
		regAddr, sl.InnerAddr, sl.BroadAddr, sl.BroadPort)
	fmt.Printf("客户端机：solo 方言（\\r\\n 皆终结·单请求·30s 请求/5s 滞留关闭）；黑白名单 %d\n", sl.BlockOrAllow)
	if len(seeds) > 0 {
		fmt.Printf("静态条目以 3s 注册心跳合成：%s（+%d 追加）\n", seeds[0].Name, len(seeds)-1)
	} else {
		fmt.Println("表初始为空：等待 world/entry 进程向注册面广播（BuildWorldPacket/BuildEntryPacket）")
	}

	<-ctx.Done()
	close(stopSeed)
	stopTick()
	_ = cs.Close()
	_ = rs.Close()
	fmt.Println("sololister: 已停止")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sololister:", err)
	os.Exit(1)
}

// extraFlags 收集重复的 -extra name=host:port[=busy] 旗标。
type extraFlags []login.ServerEntry

func (e *extraFlags) String() string { return "" }

func (e *extraFlags) Set(v string) error {
	parts := strings.SplitN(v, "=", 3)
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
