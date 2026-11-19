// soloworld 是 game_world.exe 的 Go 等价进程载体（小写系 solo 世界服）。
//
// 与官方同型：按 `<exe基名>.ini` 发现配置（或 -config 指定），缺省释放
// 内置 game_world.ini 方言；-district/-server/-inner/-broadcast/-echo 旗标
// 覆盖 ini。官方 [Module] DLL 装载不复刻——进程内直接装配
// internal/fssoloworld（cmd/solomember/cmd/sololister 同范式）。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
)

// defaultINI 快照 game_world.ini 方言（BGserverCJ\Debug\game_world.ini）。
const defaultINI = `[System]
Sleep=-1
MainEntity=SoloWorld
MainScript=main

[Module]
fskit.dll
fscomm.dll
fssoloworld.dll

[Script]
Path=..\lua\soloworld
Reload=1

[Resource]
Path=..\res

[Main]
DistrictID=3
ServerID=11
ServerName=server_11
RoleNameSizeLimit=18

[Inner]
Addr=127.0.0.1

[Broadcast]
Addr=127.255.255.255
Port=4001

[ManageServer]
Inuse=true
Addr=127.0.0.1
Port=4002

[Monitor]
Inuse=true
Addr=127.0.0.1
Port=4008
`

func main() {
	var (
		configPath = flag.String("config", "", "game_world.ini 路径（空=内置方言）")
		scenesINI  = flag.String("scenes", `..\res\scenes.ini`, "场景配置（scenes.ini；空=不装载）")
		district   = flag.Int("district", 0, "覆盖 [Main] DistrictID")
		server     = flag.Int("server", 0, "覆盖 [Main] ServerID")
		innerPort  = flag.Int("inner-port", -1, "覆盖 [Inner] Port")
		broadPort  = flag.Int("broad-port", -1, "覆盖 [Broadcast] Port")
		echoPort   = flag.Int("echo-port", -1, "覆盖 [Echo] Port")
	)
	flag.Parse()

	data := []byte(defaultINI)
	if *configPath != "" {
		b, err := os.ReadFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read config: %v\n", err)
			os.Exit(1)
		}
		data = b
	} else if _, err := os.Stat("game_world.ini"); err == nil {
		if b, err := os.ReadFile("game_world.ini"); err == nil {
			data = b // 官方 <exe基名>.ini 发现序
		}
	}
	cfg, err := fssoloworld.ParseGameWorldINI(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		os.Exit(1)
	}
	if *district != 0 {
		cfg.DistrictID = *district
	}
	if *server != 0 {
		cfg.ServerID = *server
	}
	if *innerPort >= 0 {
		cfg.InnerPort = *innerPort
	}
	if *broadPort >= 0 {
		cfg.BroadPort = *broadPort
	}
	if *echoPort >= 0 {
		cfg.EchoPort = *echoPort
	}

	srv := fssoloworld.New(fssoloworld.Options{
		Config: cfg,
		Logf:   func(f string, a ...any) { fmt.Printf(f+"\n", a...) },
		OnClosed: func() {
			fmt.Println("world server shutdown...")
			os.Exit(0)
		},
		Scenes: loadScenes(*scenesINI),
	})
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("app name %s\n", cfg.AppName())
	fmt.Printf("echo socket %s:%d\n", cfg.EchoAddr, cfg.EchoPort)
	fmt.Printf("inner socket %s:%d\n", cfg.InnerAddr, srv.InnerPort())
	fmt.Printf("broadcast socket %s:%d\n", cfg.BroadAddr, cfg.BroadPort)
	fmt.Printf("game server %d starting...\n", cfg.ServerID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	srv.Stop()
	fmt.Println(strings.Repeat("-", 20))
}

// loadScenes 官方 main.lua load_scene_config 的薄封装：文件缺失静默
// 跳过（官方 LoadFromFile 失败即 return false）。
func loadScenes(path string) []fssoloworld.SceneSpec {
	if path == "" {
		return nil
	}
	specs, err := fssoloworld.LoadScenesINI(path)
	if err != nil {
		return nil
	}
	if len(specs) > 0 {
		fmt.Println(fssoloworld.FormatScenesCount(len(specs)))
	}
	return specs
}
