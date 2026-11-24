// world 是世界服进程（对应原版 FsLoader.exe GameWorld.ini → FsServer.dll，
// 集群形态 MainConsole + ..\Lua\GameWorld\main.lua）：
// 六消息队列（inner/store/entry/member/public/charge）+ DoWait 主泵 +
// 世界 13 态/玩家 14 态/场景 4 态状态机 + 矩阵卡密保 + 120B 区服信息广播；
// 维护面 EchoSock + 内网 Adapter 受面。
//
// 权威文档：docs/official/FsServer全解与Go复刻.md。
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/lizhanyin/9yin-server-go/internal/fsserver"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

//go:embed GameWorld.ini
var defaultGameWorldINI []byte

const version = "0.1.0-fsserver-replica"

func main() {
	var (
		configPath = flag.String("config", "GameWorld.ini", "GameWorld.ini 配置路径")
		releaseINI = flag.Bool("release-ini", false, "当前目录缺 GameWorld.ini 时释放内置官方默认（已存在绝不覆盖）")
		district   = flag.Uint("district", 0, "游戏分区编号（0=取配置；缺省 1）")
		serverID   = flag.Uint("server-id", 0, "服务器编号（0=取配置；缺省 1）")
		innerPort  = flag.Int("inner-port", 0, "内网监听端口（0=取缺省 16001）")
		echoPort   = flag.Int("echo-port", 0, "维护面端口（0=取缺省 15001）")
		scenesINI  = flag.String("scenes", "", "场景配置 scenes.ini 路径（空=不加载）")
		filterPath = flag.String("filter", "", "名字过滤表 filter.txt 路径（空=不加载）")
		tps        = flag.Int("tps", 100, "DoWait 每秒帧数")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go world %s\n", version)

	var cfg fsserver.Config
	if data, err := os.ReadFile(*configPath); err == nil {
		cfg, err = fsserver.ParseGameWorldINI(data)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("已加载配置：%s（District=%d Server=%d）\n", *configPath, cfg.DistrictID, cfg.ServerID)
	} else if *releaseINI {
		if err := os.WriteFile(*configPath, defaultGameWorldINI, 0644); err != nil {
			fatal(err)
		}
		cfg, _ = fsserver.ParseGameWorldINI(defaultGameWorldINI)
		fmt.Printf("已释放内置官方配置：%s\n", *configPath)
	} else {
		cfg = fsserver.DefaultConfig()
		fmt.Println("未找到配置文件，采用官方缺省（GameWorld.ini 方言）")
	}
	// 显式旗标优先（flag.Visit 语义）
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "district":
			cfg.DistrictID = uint32(*district)
		case "server-id":
			cfg.ServerID = uint32(*serverID)
		case "inner-port":
			cfg.InnerPort = *innerPort
		case "echo-port":
			cfg.EchoPort = *echoPort
		}
	})

	asm, err := fsserver.Assemble(cfg, fsserver.AssembleOptions{TPS: *tps})
	if err != nil {
		fatal(err)
	}
	w := asm.World

	// 资源装载（官方 LoadNameFilter/load_scene_config）
	if *filterPath != "" {
		if _, err := res.ReadFile(*filterPath); err == nil {
			fmt.Println("load name filter: (接入面)")
		}
	}
	if *scenesINI != "" {
		if data, err := res.ReadFile(*scenesINI); err == nil {
			n := loadScenes(w, data)
			fmt.Printf("load scene config: %d scenes\n", n)
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

// loadScenes 官方 load_scene_config（scenes.ini：节=场景号，键 Member/
// Script/Config/Clonable）。
func loadScenes(w *fsserver.World, data []byte) int {
	n := 0
	sec := ""
	saw := map[string]map[string]string{}
	for _, raw := range splitLines(data) {
		line := trimSpace(raw)
		if line == "" || line[0] == ';' || (len(line) > 1 && line[0:2] == "//") {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			sec = line[1 : len(line)-1]
			saw[sec] = map[string]string{}
			continue
		}
		if i := indexEq(line); i > 0 && sec != "" {
			saw[sec][lower(line[:i])] = trimSpace(line[i+1:])
		}
	}
	for id, kv := range saw {
		n64 := parseInt(id)
		member := parseInt(kv["member"])
		script, config := kv["script"], kv["config"]
		var err error
		if kv["clonable"] == "true" || kv["clonable"] == "1" {
			_, err = w.Scenes.AddPrototypeScene(uint32(n64), uint32(member), pick(script, "scene"), config)
		} else {
			_, err = w.Scenes.AddScene(uint32(n64), uint32(member), pick(script, "scene"), config)
		}
		if err == nil {
			n++
		}
	}
	return n
}

func pick(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func splitLines(b []byte) []string {
	out := []string{}
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, string(b[start:i]))
			start = i + 1
		}
	}
	out = append(out, string(b[start:]))
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

func indexEq(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i
		}
	}
	return -1
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "world:", err)
	os.Exit(1)
}
