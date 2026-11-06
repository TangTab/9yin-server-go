// sql-replay 将 migrations/ 中的 SQL 迁移重放到 MySQL。
//
// 幂等：已应用且校验和一致的版本自动跳过；文件被改动过的版本报
// ErrChecksumMismatch 拒绝重放，防止同一版本出现两套 schema。
//
// DSN 解析优先级：-dsn 参数 > NINEYIN_MYSQL_DSN 环境变量 > env.server。
package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/migrations"
)

func main() {
	dsnFlag := flag.String("dsn", "", "MySQL DSN（默认依次取 NINEYIN_MYSQL_DSN 环境变量、env.server）")
	envFile := flag.String("env", "env.server", "DSN 兜底配置文件（-dsn 与环境变量均未提供时读取）")
	dir := flag.String("dir", "migrations", "磁盘上的 migrations 目录")
	embedded := flag.Bool("embedded", false, "改用编译时嵌入的迁移集合（忽略 -dir）")
	list := flag.Bool("list", false, "列出迁移版本、文件名与校验和，不连接数据库")
	status := flag.Bool("status", false, "对照台账显示已应用/待应用，只读不修改")
	force := flag.Bool("force", false, "无视台账全量重放（存档文件幂等；用于缺表补建、按存档恢复配置）")
	timeout := flag.Duration("timeout", 2*time.Minute, "整体超时")
	flag.Parse()

	all, err := loadMigrations(*dir, *embedded)
	if err != nil {
		log.Fatalf("sql-replay: %v", err)
	}
	if *list {
		for _, migration := range all {
			fmt.Printf("%d\t%s\t%x\n", migration.Version, migration.Name, migration.Checksum)
		}
		return
	}

	dsn, dsnSource, err := resolveDSN(*dsnFlag, *envFile)
	if err != nil {
		log.Fatalf("sql-replay: %v", err)
	}
	normalized, err := role.ProductionDSN(dsn)
	if err != nil {
		log.Fatalf("sql-replay: invalid DSN: %v", err)
	}
	config, err := mysql.ParseDSN(normalized)
	if err != nil {
		log.Fatalf("sql-replay: normalize DSN: %v", err)
	}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("sql-replay: ping: %v", err)
	}

	runner := migrations.Runner{DB: db, Migrations: all, Force: *force}
	fmt.Printf("sql-replay: target %s@%s/%s (DSN from %s)\n", config.User, config.Addr, config.DBName, dsnSource)
	states, err := runner.Status(ctx, all)
	if err != nil {
		log.Fatalf("sql-replay: %v", err)
	}
	printStatus(states)
	if *status {
		return
	}

	pending := 0
	for _, state := range states {
		if !state.Applied {
			pending++
		}
	}
	if pending > 0 {
		fmt.Printf("sql-replay: applying %d pending migration(s)...\n", pending)
	}
	if err := runner.Up(ctx); err != nil {
		log.Fatalf("sql-replay: %v", err)
	}
	after, err := runner.Status(ctx, all)
	if err != nil {
		log.Fatalf("sql-replay: verify: %v", err)
	}
	for _, state := range after {
		if !state.Applied || !state.ChecksumOK {
			log.Fatalf("sql-replay: version %d (%s) not verified after replay", state.Version, state.Name)
		}
	}
	fmt.Println("sql-replay: all migrations applied and verified")
}

// loadMigrations 载入待重放的迁移集合（磁盘目录或编译时嵌入）。
func loadMigrations(dir string, embedded bool) ([]migrations.Migration, error) {
	if embedded {
		return migrations.Embedded()
	}
	if info, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("migrations 目录不可用: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", dir)
	}
	all, err := migrations.Load(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", dir, err)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("%s 下没有 *.sql", dir)
	}
	return all, nil
}

// resolveDSN 按优先级解析 DSN，返回来源描述便于回显。
func resolveDSN(flagDSN, envFile string) (string, string, error) {
	if flagDSN != "" {
		return flagDSN, "-dsn", nil
	}
	if value := os.Getenv("NINEYIN_MYSQL_DSN"); value != "" {
		return value, "env NINEYIN_MYSQL_DSN", nil
	}
	if data, err := os.ReadFile(envFile); err == nil {
		if value := parseEnvServer(data)["NINEYIN_MYSQL_DSN"]; value != "" {
			return value, envFile, nil
		}
	}
	return "", "", fmt.Errorf("缺少 DSN：请用 -dsn、NINEYIN_MYSQL_DSN 环境变量或在 %s 中配置 NINEYIN_MYSQL_DSN", envFile)
}

// parseEnvServer 解析 env.server 中形如 set "VAR=VALUE" 的行（与
// deploy.bat 的读取规则一致：只认 set 开头的行，:: / # 注释与其它内容忽略）。
func parseEnvServer(data []byte) map[string]string {
	values := make(map[string]string)
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "::") || strings.HasPrefix(line, "#") {
			continue
		}
		rest, ok := strings.CutPrefix(line, "set")
		if !ok || (rest != "" && rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		key, value, ok := strings.Cut(strings.Trim(strings.TrimSpace(rest), `"`), "=")
		if !ok {
			continue
		}
		if key = strings.TrimSpace(key); key != "" {
			values[key] = value
		}
	}
	return values
}

func printStatus(states []migrations.MigrationStatus) {
	for _, state := range states {
		label := "pending"
		when := "-"
		if state.Applied {
			label = "applied"
			if !state.ChecksumOK {
				label = "DRIFT"
			}
			when = state.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("%5d  %-7s  %-19s  %s\n", state.Version, label, when, state.Name)
	}
}
