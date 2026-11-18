// gamedumper 是数据转储服务器进程（对应原版 GameDumper.bat →
// FsLoader.exe GameDumper.ini → FsDumper.dll + FsDumpLogic.dll 形态）。
//
// 承载 internal/fsdumper：Outer :4006 行协议口（dump_role_msg 双向 +
// 资源转储巡检）+ Echo :15009 维护口（fscomm.EchoSock）。DB 面直连
// MySQL（官方 CAccess→Store 内部协议不复刻，见包文档）。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	"github.com/lizhanyin/9yin-server-go/internal/fsdumper"
)

const version = "0.1.0-gamedumper"

func main() {
	cfgPath := flag.String("c", "", "GameDumper.ini 路径（缺省取工作目录；可用 NINEYIN_DUMPER_CONFIG）")
	resDir := flag.String("res", "", "Res 资源根（缺省 = 配置目录 ..\\Res）")
	dsnOverride := flag.String("dsn", "", "MySQL DSN 直供（绕过 ConnectString 解析；可用 NINEYIN_DUMPER_DSN）")
	flag.Parse()

	path := *cfgPath
	if path == "" {
		path = os.Getenv("NINEYIN_DUMPER_CONFIG")
	}
	if path == "" {
		path = "GameDumper.ini"
	}
	cfg, err := fsdumper.LoadConfig(path)
	if err != nil {
		fatalf("读配置失败: %v", err)
	}

	res := *resDir
	if res == "" {
		res = filepath.Join(filepath.Dir(filepath.Clean(path)), "..", "Res")
	}
	if st, err := os.Stat(res); err != nil || !st.IsDir() {
		fatalf("Res 资源根不可用: %s", res)
	}
	cfg.ResourceDir = res

	dsn := *dsnOverride
	if dsn == "" {
		dsn = os.Getenv("NINEYIN_DUMPER_DSN")
	}
	logf := func(format string, args ...any) {
		fmt.Printf("[dumper] "+format+"\n", args...)
	}

	// DB 面：官方四路 CAccess（role/public/manage/output）；
	// DSN 未给时按 OLEDB ConnectString 解析。
	openAccess := func(name string) *sql.DB {
		cs := cfg.AccessDSN(name)
		if cs == "" {
			return nil
		}
		d := dsn
		if d == "" {
			var err error
			d, err = oledbToDSN(cs)
			if err != nil {
				fatalf("access %s ConnectString 解析失败: %v", name, err)
			}
		}
		db, err := sql.Open("mysql", d)
		if err != nil {
			fatalf("access %s 打开失败: %v", name, err)
		}
		db.SetMaxOpenConns(2)
		if err := db.Ping(); err != nil {
			fatalf("access %s 连接失败(%s): %v", name, maskDSN(d), err)
		}
		return db
	}
	roleDB := openAccess("role")
	manageDB := openAccess("manage")
	outputDB := openAccess("output")
	if roleDB == nil && manageDB == nil {
		fatalf("[Role]/[Manage] ConnectString 均为空，无处落库")
	}
	pick := func(prefer, alt *sql.DB) *sql.DB {
		if prefer != nil {
			return prefer
		}
		return alt
	}

	// 业务装配（官方 dump_callee：DumpRole + DumpRes 两单例）。
	dbinfo, err := os.ReadFile(filepath.Join(res, "dumpconfig", "dump_role_dbinfo.xml"))
	if err != nil {
		fatalf("dump_role_dbinfo.xml: %v", err)
	}
	dbrec, err := os.ReadFile(filepath.Join(res, "dumpconfig", "dump_role_record.xml"))
	if err != nil {
		fatalf("dump_role_record.xml: %v", err)
	}
	role, err := fsdumper.NewDumpRole(dbinfo, dbrec, logf)
	if err != nil {
		fatalf("DumpRole 装配失败: %v", err)
	}
	dres, err := fsdumper.NewDumpRes(res, logf)
	if err != nil {
		fatalf("DumpRes 装配失败: %v", err)
	}
	callee := fsdumper.NewDefaultCallee(role, dres)

	outerPort := cfg.OuterPort
	if outerPort == 0 {
		outerPort = -1 // 官方 [Outer] Port=0 → 不开外口
	}
	outer := fsdumper.NewServer(cfg.OuterAddr, outerPort, callee, logf)
	outer.Kernel.ResourcePath = res
	// 先注册四路 Access 再 Start（官方 AddAccess 先于 game_dumper:Start；
	// 顺序颠倒会让 DumpRole EnsureTable 找不到 output 库）。
	for name, db := range map[string]*sql.DB{
		"role": roleDB, "public": pick(nil, roleDB), "manage": pick(manageDB, roleDB), "output": pick(outputDB, pick(manageDB, roleDB)),
	} {
		if db != nil {
			outer.Kernel.AddAccess(name, fsdumper.SQLDB{DBHandle: db})
		}
	}
	if err := outer.Start(); err != nil {
		fatalf("Outer 启动失败: %v", err)
	}

	// Echo 维护口（官方 echo.lua：quit/end）。
	echo := fscomm.NewEchoSock(cfg.EchoAddr, cfg.EchoPort, fscomm.EchoSockHandler{
		OnCommand: map[string]fscomm.EchoLineHandler{
			"quit": func([]string) {
				logf("server will shutdown...")
				outer.Stop()
				os.Exit(0)
			},
			"end": func([]string) {},
		},
		OnInput: func(line string) {
			logf("Undefined command: %s", line)
		},
	})
	if err := echo.Start(); err != nil {
		fatalf("Echo 启动失败: %v", err)
	}

	logf("App name dumper")
	logf("Echo socket %s:%d", cfg.EchoAddr, cfg.EchoPort)
	logf("Game dumper server started (version %s)", version)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logf("echo socket shutdown...")
	echo.Stop()
	logf("dumper server shutdown...")
	outer.Stop()
}

// oledbToDSN 官方 ConnectString（SQLOLEDB 方言）→ go-sql-driver DSN。
// Provider=SQLOLEDB;Password=pw;User ID=sa;Initial Catalog=nx_base;Data Source=127.0.0.1:3306
// → sa:pw@tcp(127.0.0.1:3306)/nx_base?charset=utf8mb4&parseTime=true
func oledbToDSN(cs string) (string, error) {
	parts := map[string]string{}
	for _, kv := range strings.Split(cs, ";") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		parts[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	user := parts["user id"]
	pass := parts["password"]
	src := parts["data source"]
	db := parts["initial catalog"]
	if user == "" || src == "" || db == "" {
		return "", fmt.Errorf("缺 User ID/Data Source/Initial Catalog")
	}
	if _, _, err := net.SplitHostPort(src); err != nil {
		src += ":3306"
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user, pass, src, db), nil
}

func maskDSN(d string) string {
	if i := strings.IndexByte(d, ':'); i >= 0 {
		return d[:i] + ":****" + d[strings.LastIndex(d, "@"):]
	}
	return d
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[gamedumper] "+format+"\n", args...)
	os.Exit(1)
}

var _ = context.Background
