package gmpanel

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fswatch"
)

// Config 面板配置（cmd/panel flag 层装配）。
type Config struct {
	Addr    string // HTTP 受面地址（默认 127.0.0.1，勿暴露公网）
	Port    int    // 默认 19064（env.server.default 伏笔口径）
	Mode    string // "cluster"（九服拓扑）| "single"（solounite 合一）
	BinDir  string // 集群 exe 目录（子进程工作目录，ini 释放侧）
	LogDir  string // 托管子进程 stdout 落盘目录（默认 logs）
	Members   int  // cluster 模式 member 数（默认 1）
	WorldPort int  // world echo 口覆盖（0 = 清单默认 15001；测试注动态口）
	DSN       string // MySQL DSN（空 = 禁用角色/告警页）
	Logf      func(format string, args ...any)
}

func (c *Config) logger() func(string, ...any) {
	if c.Logf != nil {
		return c.Logf
	}
	return func(string, ...any) {}
}

// Server 面板服务：supervisor + echo 客户端 + MySQL（惰性重连）+ HTTP。
type Server struct {
	cfg  Config
	logf func(string, ...any)
	sup  *Supervisor
	sys  *fswatch.Sampler

	dbMu    sync.Mutex
	db      *fssolostore.MySQLStore
	dbNext  time.Time // 惰性重连限频（失败后 30s 内不再重试）
	dbReady bool      // 曾连上过（日志口径）

	ln net.Listener
}

// New 装配面板（不启动）。Port=0 保留（动态口，测试注入）；Addr/Port
// 的产品默认值在 cmd/panel 的 flag 层（固定口填在这里会让测试面板叠
// 19064 假阴性——本仓「测试与服务解耦」约定）。DSN 非空时立即尝试连库，
// 失败不致命（角色/告警页降级，后续请求惰性重连）。
func New(cfg Config) (*Server, error) {
	if cfg.Mode == "" {
		cfg.Mode = "cluster"
	}
	if cfg.Members <= 0 {
		cfg.Members = 1
	}
	s := &Server{cfg: cfg, logf: cfg.logger(), sys: fswatch.NewSampler()}
	s.sup = NewSupervisor(cfg)
	if cfg.DSN != "" {
		if db, err := fssolostore.OpenMySQLStore(context.Background(), cfg.DSN, s.logf); err != nil {
			s.logf("gmpanel: mysql 不可用（角色/告警页降级）: %v", err)
		} else {
			s.db, s.dbReady = db, true
		}
	}
	return s, nil
}

// store 惰性取库：nil 且 DSN 非空时限频重开（MySQL 后启/重启场景）。
func (s *Server) store() (*fssolostore.MySQLStore, error) {
	if s.cfg.DSN == "" {
		return nil, fmt.Errorf("面板未配置 MySQL DSN（-mysql-dsn / NINEYIN_MYSQL_DSN）")
	}
	s.dbMu.Lock()
	defer s.dbMu.Unlock()
	if s.db != nil {
		if err := s.db.Ping(); err == nil {
			return s.db, nil
		}
		s.db.Close()
		s.db = nil
	}
	if time.Now().Before(s.dbNext) {
		return nil, fmt.Errorf("mysql 不可用（重试限频中）")
	}
	db, err := fssolostore.OpenMySQLStore(context.Background(), s.cfg.DSN, s.logf)
	if err != nil {
		s.dbNext = time.Now().Add(30 * time.Second)
		return nil, fmt.Errorf("mysql 不可用: %w", err)
	}
	if !s.dbReady {
		s.logf("gmpanel: mysql 已连上（角色/告警页恢复）")
		s.dbReady = true
	}
	s.db = db
	return s.db, nil
}

// worldCli world echo 维护口客户端（cluster=soloworld、single=solounite，
// 两形态同用 15001 口径；从 supervisor 的 world spec 取探活口）。
func (s *Server) worldCli() *EchoCli {
	if spec := s.sup.Spec("world"); spec != nil {
		return NewEchoCli(spec.ProbeHost, spec.ProbePort)
	}
	return NewEchoCli("127.0.0.1", 15001)
}

// Start 开 HTTP 受面（Addr 空 = 127.0.0.1；Port 0 = 动态口）。
func (s *Server) Start() error {
	addr := s.cfg.Addr
	if addr == "" {
		addr = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, s.cfg.Port))
	if err != nil {
		return err
	}
	s.ln = ln
	go func() {
		srv := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 5 * time.Second}
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logf("gmpanel: http serve 退出: %v", err)
		}
	}()
	s.logf("gmpanel: http://%s:%d mode=%s bin=%s", s.cfg.Addr, s.cfg.Port, s.cfg.Mode, s.cfg.BinDir)
	return nil
}

// Addr 返回实际受面（测试动态口取回）。
func (s *Server) Addr() net.Addr {
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

// Stop 关 HTTP 与子进程日志句柄（不杀托管集群——面板退出后子进程
// 转外部进程，重启面板即观察模式续管）。
func (s *Server) Stop() {
	if s.ln != nil {
		s.ln.Close()
	}
	s.sup.Close()
	s.dbMu.Lock()
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	s.dbMu.Unlock()
}
