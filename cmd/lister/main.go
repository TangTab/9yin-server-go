// lister 是区服列表分发进程（对应原版 game_lister.exe + FsLister.dll /
// GameLister 组件），承载 internal/login 的 svrlist ASCII 行协议服务：
// 客户端连上发 "svrlist"，回一条区服条目（名字 \uXXXX 转义 + 通告地址）。
//
// 替代旧分支的 server-cmd/loopback-lister.ps1（方言 golden 见
// testdata/captures/svrlist-{request,response}.bin）。solo 形态下本服务
// 内嵌在 cmd/solo（-svrlist），本入口用于独立部署/多服分发。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/lizhanyin/9yin-server-go/internal/login"
)

const version = "0.1.0-lister"

func main() {
	var (
		addr = flag.String("addr", "127.0.0.1:9601", "svrlist 监听地址")
		name = flag.String("name", "本地测试", "下发的服务器显示名")
		host = flag.String("host", "", "通告的游戏服地址（空=监听 host，通配回退 127.0.0.1）")
		port = flag.Int("port", 9600, "通告的游戏服端口")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go lister %s\n", version)

	l := &login.Lister{
		Addr:  *addr,
		Entry: login.ResolveEntryHost(login.ServerEntry{Name: *name, Host: *host, Port: *port}, *addr),
		Logf:  func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	listenAddr, err := l.Listen()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("svrlist 就绪：%s → 通告 %s:%d（%s）\n",
		listenAddr, l.Entry.Host, l.Entry.Port, l.Entry.Name)

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
