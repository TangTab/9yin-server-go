// solomember 是 game_member.exe 的 Go 等价进程载体（小写系 solo 场景服）。
//
// 与官方同型：按 `<exe基名>.ini` 发现配置（或 -config 指定），缺省释放
// 内置 game_member.ini 方言；-district/-server/-member/-inner/-broadcast
// 旗标覆盖 ini。官方 [Module] DLL 装载不复刻——进程内直接装配
// internal/fssolomember（cmd/auction/cmd/sololister 同范式）。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lizhanyin/9yin-server-go/internal/fssolomember"
)

// defaultINI 快照 game_member.ini 方言（BGserverCJ\Debug\game_member.ini）。
const defaultINI = `[System]
Sleep=-1
MainEntity=SoloMember
MainScript=main

[Module]
fskit.dll
fscomm.dll
fscollide.dll
fssolomember.dll

[Script]
Path=..\lua\solomember
Reload=1

[Resource]
Path=..\res

[Main]
DistrictID=1
ServerID=11
MemberID=0

[Logic]
Modules=1
Module0=fsgame.dll

[Inner]
//Addr=127.0.0.1
//Port=5001

[Broadcast]
//Addr=127.255.255.255
//Port=4001
`

func main() {
	var (
		configPath = flag.String("config", "", "game_member.ini 路径（空=内置方言）")
		district   = flag.Uint("district", 0, "覆盖 [Main] DistrictID")
		server     = flag.Uint("server", 0, "覆盖 [Main] ServerID")
		member     = flag.Uint("member", 0, "覆盖 [Main] MemberID")
		innerPort  = flag.Int("inner-port", -1, "覆盖 [Inner] Port")
		broadPort  = flag.Int("broad-port", -1, "覆盖 [Broadcast] Port")
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
	} else if _, err := os.Stat("game_member.ini"); err == nil {
		if b, err := os.ReadFile("game_member.ini"); err == nil {
			data = b // 官方 <exe基名>.ini 发现序
		}
	}
	cfg, err := fssolomember.ParseGameMemberINI(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		os.Exit(1)
	}
	if *district != 0 {
		cfg.DistrictID = uint32(*district)
	}
	if *server != 0 {
		cfg.ServerID = uint32(*server)
	}
	if *member != 0 {
		cfg.MemberID = uint32(*member)
	}
	if *innerPort >= 0 {
		cfg.InnerPort = *innerPort
	}
	if *broadPort >= 0 {
		cfg.BroadPort = *broadPort
	}

	srv := fssolomember.New(fssolomember.Options{
		Config: cfg,
		Logf:   func(f string, a ...any) { fmt.Printf(f+"\n", a...) },
		OnExit: func() { fmt.Println("member server shutdown...") },
	})
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("app name member_%d_%d\n", cfg.ServerID, cfg.MemberID)
	fmt.Printf("inner socket %s:%d\n", cfg.InnerAddr, srv.InnerPort())
	fmt.Printf("broadcast socket %s:%d\n", cfg.BroadAddr, cfg.BroadPort)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	srv.Stop()
	fmt.Println(strings.Repeat("-", 20))
}
