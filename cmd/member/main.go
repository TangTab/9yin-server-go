// member 是场景服务进程（对应原版 game_member.exe + FsMember.dll）：
// 场景/玩家注册表、world/public/store 三消息队列、entry 客户端流转、
// 状态机与 member 信息广播；维护面 EchoSock + 进程注册 AppManager。
//
// 装配逻辑在 internal/fsmember/assemble.go（Assemble/Assembly，官方
// main.lua 启停序）——本入口只做旗标/配置解析；同一装配器亦被
// cmd/loader 的「同名多角色」路径复用（game_member.exe 裸跑形态）。
//
// 权威文档：docs/official/FsMember全解与Go复刻.md。
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/lizhanyin/9yin-server-go/internal/fsmember"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

//go:embed GameMember.ini
var defaultGameMemberINI []byte

const version = "0.2.0-fsmember-replica"

func main() {
	var (
		configPath = flag.String("config", "GameMember.ini", "GameMember.ini 配置路径（不存在时用旗标/缺省值）")
		memberID   = flag.Uint("member-id", 0, "场景服务器编号（覆盖配置 [Main] MemberID）")
		innerPort  = flag.Int("inner-port", 0, "内网监听端口（0=取缺省 16300+MemberID）")
		echoPort   = flag.Int("echo-port", 0, "维护面端口（0=取缺省 15300+MemberID）")
		worldAddr  = flag.String("world", "", "world 内网地址（空=不外连）")
		publicAddr = flag.String("public", "", "public 内网地址（空=不外连）")
		filterPath = flag.String("filter", "", "名字过滤表 filter.txt 路径（空=不加载）")
		locPath    = flag.String("locations", "", "落地点表 location.ini 路径（空=不加载）")
		scenesMem  = flag.String("scenes-member", "", "静态场景分配表 scenes_member.txt 路径（空=不加载）")
		tps        = flag.Int("tps", 100, "DoWait 每秒帧数")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go member %s\n", version)

	var cfg fsmember.Config
	if data, err := os.ReadFile(*configPath); err == nil {
		cfg, err = fsmember.ParseGameMemberINI(data)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("已加载配置：%s（District=%d Server=%d Member=%d）\n",
			*configPath, cfg.DistrictID, cfg.ServerID, cfg.MemberID)
	} else {
		cfg = fsmember.DefaultConfig()
		fmt.Println("未找到配置文件，采用官方缺省（GameMember.ini 方言）")
	}
	if flagVisited("member-id") {
		cfg.MemberID = uint32(*memberID)
		cfg.InnerPort = 16300 + int(cfg.MemberID)
		cfg.EchoPort = 15300 + int(cfg.MemberID)
	}
	if *innerPort != 0 {
		cfg.InnerPort = *innerPort
	}
	if *echoPort != 0 {
		cfg.EchoPort = *echoPort
	}

	// 角色装配（官方 main.lua 装配段的 Go 等价）
	asm, err := fsmember.Assemble(nil, cfg, fsmember.AssembleOptions{
		WorldAddr:  *worldAddr,
		PublicAddr: *publicAddr,
		TPS:        *tps,
	})
	if err != nil {
		fatal(err)
	}
	m := asm.Member

	// 可选资源装载（官方 LoadNameFilter/LoadLocations/静态场景分配）
	if *filterPath != "" {
		if data, err := res.ReadFile(*filterPath); err == nil {
			words := fsmember.ParseFilterText(data)
			fmt.Printf("load name filter: %d words\n", len(words))
		}
	}
	if *locPath != "" {
		if data, err := res.ReadFile(*locPath); err == nil {
			locations := fsmember.NewLocations()
			_ = locations.Load(data)
			fmt.Printf("load locations: %d points\n", locations.Count())
		}
	}
	if *scenesMem != "" {
		if data, err := res.ReadFile(*scenesMem); err == nil {
			for _, id := range fsmember.MemberScenesFor(data, cfg.MemberID) {
				if _, err := m.Scenes.CreateScene(id, fsmember.ScenePrototype,
					fmt.Sprintf("scene_%d", id), 0); err == nil {
					fmt.Printf("create scene %d (static)\n", id)
				}
			}
		}
	}

	if err := asm.Start(); err != nil {
		fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	asm.Stop() // 官方 close() 停机序
}

func flagVisited(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "member:", err)
	os.Exit(1)
}
