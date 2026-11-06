package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/migrations"
)

func main() {
	dsn := flag.String("dsn", os.Getenv("NINEYIN_MYSQL_DSN"), "MySQL DSN (or NINEYIN_MYSQL_DSN)")
	timeout := flag.Duration("timeout", 2*time.Minute, "overall migration timeout")
	list := flag.Bool("list", false, "print embedded migrations and checksums without connecting")
	flag.Parse()
	if *list {
		all, err := migrations.Embedded()
		if err != nil {
			log.Fatal(err)
		}
		for _, migration := range all {
			fmt.Printf("%d\t%s\t%x\n", migration.Version, migration.Name, migration.Checksum)
		}
		return
	}
	if *dsn == "" {
		log.Fatal("role-migrate: -dsn or NINEYIN_MYSQL_DSN is required")
	}
	normalized, err := role.ProductionDSN(*dsn)
	if err != nil {
		log.Fatalf("role-migrate: invalid DSN: %v", err)
	}
	config, err := mysql.ParseDSN(normalized)
	if err != nil {
		log.Fatalf("role-migrate: normalize DSN: %v", err)
	}
	if config.DBName == "" {
		log.Fatal("role-migrate: DSN must name a database")
	}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("role-migrate: ping: %v", err)
	}
	if err := (migrations.Runner{DB: db}).Up(ctx); err != nil {
		log.Fatalf("role-migrate: %v", err)
	}
	fmt.Println("role-migrate: all embedded migrations applied and verified")
}
