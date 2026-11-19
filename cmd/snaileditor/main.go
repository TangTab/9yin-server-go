// snaileditor 是 SnailEditor 编辑器管线演示宿主，对应原版
// SnailEditor.dll + SnailEditorNet.dll + SharingMemory.dll 三件
// （BGserverCJ\Debug 形态，游戏宿主内的编辑器逻辑模块）。
//
// 官方形态：FsMember 宿主按 ini [Logic] 装载 snaileditor.dll；
// 模块 Init 注册 scene.OnCreate/player/HB_ProcessMsg，StartSnailEditor
// 读 ini\SnailEditorServerModule.xml + 注册表 TortoiseProc 路径，
// 动态加载 SnailEditorNet.dll 外连编辑器 GUI（FMSG 帧 0x3E8..0x3F1），
// 经共享内存 SNAILEDITOR_SvnUpdateFinish 收 SVN 完成事件，命令帧
// （'*' 分隔路径列表）入 MsgManager 由心跳 HB_ProcessMsg 泵给
// EditorUpdate::Reload 热重载。本进程 = 该链路的 Go 等价单进程演示
// （internal/snaileditor + internal/snaileditornet + internal/snailsharemem），
// 默认再起一个回环"编辑器 GUI"端以便无外部依赖联调。
//
// 权威文档：docs/official/SnailEditor四件套全解与Go复刻.md
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/snaileditor"
	"github.com/lizhanyin/9yin-server-go/internal/snaileditornet"
)

const version = "0.1.0-snaileditor"

func main() {
	var (
		configPath = flag.String("config", "", "SnailEditorServerModule.xml 路径（官方 ini\\SnailEditorServerModule.xml）")
		guiAddr    = flag.String("gui", "", "编辑器 GUI 地址 ip:port（官方 config 段 SvnServerIP/SvnServerPort；缺省本机回环演示 GUI）")
		guiListen  = flag.String("gui-listen", "127.0.0.1:16010", "演示 GUI 监听地址（-gui 为空时启用）")
		noGUI      = flag.Bool("no-gui", false, "不内置演示 GUI（配合外部编辑器客户端）")
		dump       = flag.Bool("dump", false, "打印收到的命令/重载/SVN 事件")
	)
	flag.Parse()
	fmt.Printf("snaileditor %s — SnailEditor 管线宿主（Go 复刻）\n", version)

	// ① 配置（官方 xml；缺省内置演示配置）
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[config] SvnLocalRoot=%s SvnServer=%s:%d watch=%v\n",
		cfg.Config.SvnLocalRootPath, cfg.Config.SvnServerIP, cfg.Config.SvnServerPort, cfg.WatchPaths())
	fmt.Printf("[config] TortoiseProc=%s\n", snaileditornet.TortoiseProcPath())

	// ② 演示 GUI（官方编辑器客户端的 Server 侧角色）
	guiTarget := *guiAddr
	if guiTarget == "" && !*noGUI {
		srv, err := snaileditornet.NewServer(*guiListen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] 演示 GUI 监听失败: %v\n", err)
			os.Exit(1)
		}
		defer srv.Close()
		guiTarget = srv.Addr().String()
		fmt.Printf("[gui] 演示编辑器 GUI 监听 %s（FMSG 0x3E8..0x3F1）\n", guiTarget)
		srv.OnFrame(func(id uint32, payload []byte) {
			if *dump {
				fmt.Printf("[gui<-] FMSG %#x %q\n", id, payload)
			}
		})
	}

	// ③ 游戏侧模块装配 + 回调面
	mod := snaileditornet2Module(cfg, *dump)
	if guiTarget != "" {
		ok, err := mod.StartSnailEditor(cfg, guiTarget)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL] StartSnailEditor: %v\n", err)
			os.Exit(1)
		}
		_ = ok
		fmt.Println("[module] StartSnailEditor complete")
	}

	// ④ 心跳泵（官方 HB_ProcessMsg 每帧）
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			fmt.Println("[module] Shut")
			mod.Shut()
			return
		case <-tick.C:
			for mod.HB_ProcessMsg() {
			}
		}
	}
}

// snaileditornet2Module 构造模块并接 Trace/重载回调。
func snaileditornet2Module(cfg snaileditor.ModuleXML, dump bool) *snaileditor.Module {
	m := snaileditor.New()
	if dump {
		m.Trace = func(f string, a ...any) { fmt.Printf("[trace] "+f+"\n", a...) }
	}
	m.SetReloadHandler("", func(paths []string) {
		fmt.Printf("[reload] %v\n", paths)
	})
	return m
}

// loadConfig 官方 xml；无则内置演示配置。
func loadConfig(path string) (snaileditor.ModuleXML, error) {
	if path != "" {
		return snaileditor.LoadModuleConfig(path)
	}
	var cfg snaileditor.ModuleXML
	// encoding/xml 无法反向构造，直接手写默认值的等价 XML。
	const def = `<root>
<config SvnLocalRootPath=".\" SvnServerIP="127.0.0.1" SvnServerPort="16010"/>
<watch>
  <path value="lua\"/>
  <path value="ini\"/>
</watch>
</root>`
	_ = def
	cfg.Config.SvnLocalRootPath = "."
	cfg.Config.SvnServerIP = "127.0.0.1"
	cfg.Config.SvnServerPort = 16010
	cfg.Watch.Paths = append(cfg.Watch.Paths,
		struct {
			Value string `xml:"value,attr"`
		}{Value: `lua\`},
		struct {
			Value string `xml:"value,attr"`
		}{Value: `ini\`},
	)
	return cfg, nil
}
