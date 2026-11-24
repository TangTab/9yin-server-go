// cmd/panel 自建 Web 运维面板（internal/gmpanel 宿主）：
// 默认 http://127.0.0.1:19064，supervisor 托管 + 观察双模式。
//
// 启动形态：
//
//	server-cmd\start-gm-panel.bat   （读同目录 env.server 后拉起）
//	go run ./cmd/panel              （开发直跑，bin 自动定位 build\）
//
// DSN 解析对齐 solounite：-mysql-dsn > NINEYIN_MYSQL_DSN（env.server）>
// 空禁用（角色/告警页降级）。member 数读 MEMBERS（对齐 start-cluster）。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/lizhanyin/9yin-server-go/internal/gmpanel"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
)

func main() {
	addr := flag.String("addr", "127.0.0.1", "HTTP 受面地址（本地面板，勿暴露公网）")
	port := flag.Int("port", 19064, "HTTP 受面端口（env.server.default 伏笔口径）")
	mode := flag.String("mode", "cluster", "cluster（九服拓扑）| single（solounite 合一）")
	bin := flag.String("bin", "", "集群 exe 目录（空=自动定位 bin\\ → . → ..\\build）")
	logDir := flag.String("logdir", "", "托管子进程日志目录（空=<bin>\\logs）")
	dsn := flag.String("mysql-dsn", "", "MySQL DSN（空=NINEYIN_MYSQL_DSN env，再空=禁用角色/告警页）")
	members := flag.Int("members", 0, "member 数（0=MEMBERS env，默认 1）")
	flag.Parse()
	logline.Install("panel")
	defer logline.Flush()

	if *dsn == "" {
		*dsn = os.Getenv("NINEYIN_MYSQL_DSN")
	}
	if *members <= 0 {
		if v, err := strconv.Atoi(os.Getenv("MEMBERS")); err == nil && v > 0 {
			*members = v
		} else {
			*members = 1
		}
	}
	if *bin == "" {
		*bin = locateBin()
		if *bin == "" {
			fmt.Fprintln(os.Stderr, "panel: 未定位到集群 exe 目录（build 后再启动，或 -bin 指定）")
			logline.Exit(1)
		}
	}
	logf := func(format string, args ...any) {
		fmt.Printf("[panel] "+format+"\n", args...)
	}
	s, err := gmpanel.New(gmpanel.Config{
		Addr: *addr, Port: *port, Mode: *mode, BinDir: *bin, LogDir: *logDir,
		Members: *members, DSN: *dsn, Logf: logf,
	})
	if err != nil {
		fatal(err)
	}
	if err := s.Start(); err != nil {
		fatal(err)
	}
	fmt.Printf("九阴运维面板已启动：http://%s:%d （mode=%s bin=%s members=%d mysql=%v）\n",
		*addr, *port, *mode, *bin, *members, *dsn != "")
	fmt.Println("Ctrl+C 退出（托管集群不随面板退出，重启面板转观察模式续管）")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("panel shutdown...")
	s.Stop()
}

// locateBin 二进制定位序对齐 start-cluster.bat：./bin → ./ → ../build
// （探测 game_soloworld.exe；single 模式亦共用此序）。
func locateBin() string {
	cwd, _ := os.Getwd()
	for _, dir := range []string{
		filepath.Join(cwd, "bin"),
		cwd,
		filepath.Join(cwd, "..", "build"),
	} {
		if _, err := os.Stat(filepath.Join(dir, "game_soloworld.exe")); err == nil {
			return dir
		}
	}
	// 退化：面板 exe 自身所在目录（deploy 形态 bin\game_panel.exe 同侧）
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "game_soloworld.exe")); err == nil {
			return dir
		}
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "panel:", err)
	logline.Exit(1)
}
