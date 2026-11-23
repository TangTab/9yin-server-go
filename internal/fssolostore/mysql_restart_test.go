package fssolostore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---- 真库端到端：隔离 mysqld 实例的杀库重启 ----

// mysqlTestBinEnv 开关：mysqld.exe 路径（如部署根 mysql\bin\mysqld.exe）。
// 设了才跑；实例起在临时数据目录 + 独立端口，不碰本机库。
const mysqlTestBinEnv = "NINEYIN_MYSQL_TEST_MYSQLD"

// isolatedMySQL 临时数据目录里的 mysqld（root 空密码，仅回环地址）。
type isolatedMySQL struct {
	t      *testing.T
	bin    string
	base   string
	data   string
	port   int
	logDir string

	mu   sync.Mutex
	proc *mysqlProc
}

// mysqlProc 一个在跑（或刚被杀）的 mysqld：Windows 上 mysqld.exe 还会
// 再拉一个子 mysqld 当真服务进程，所以杀要按进程树杀。
type mysqlProc struct {
	cmd     *exec.Cmd
	log     *os.File // console 输出落文件：不走管道，子进程不继承句柄
	exit    chan struct{}
	waitErr error // exit 落定后可读
}

// newIsolatedMySQL 校验二进制、备好临时数据目录与端口（不启动）。
func newIsolatedMySQL(t *testing.T) *isolatedMySQL {
	t.Helper()
	bin := os.Getenv(mysqlTestBinEnv)
	if bin == "" {
		t.Skipf("设 %s=<mysqld.exe 路径> 才跑真库杀库重启集成测试", mysqlTestBinEnv)
	}
	bin = filepath.FromSlash(bin)
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("mysqld 不可用: %v", err)
	}
	dir := t.TempDir()
	m := &isolatedMySQL{
		t:      t,
		bin:    bin,
		base:   filepath.Dir(filepath.Dir(bin)),
		data:   filepath.Join(dir, "data"),
		logDir: dir,
		port:   freeTCPPort(t),
	}
	t.Cleanup(m.Kill) // 晚于 TempDir 注册 → 先于目录清理执行（LIFO）
	return m
}

func (m *isolatedMySQL) Addr() string { return fmt.Sprintf("127.0.0.1:%d", m.port) }

func (m *isolatedMySQL) args(extra ...string) []string {
	return append([]string{
		"--no-defaults", // 不读部署根 my.ini：端口/数据目录全由此处指定
		"--basedir=" + m.base,
		"--datadir=" + m.data,
	}, extra...)
}

// Init 初始化数据目录。
func (m *isolatedMySQL) Init() {
	m.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, m.bin, m.args("--initialize-insecure", "--console")...).CombinedOutput()
	if err != nil {
		m.t.Fatalf("初始化临时数据目录: %v\n%s", err, out)
	}
}

// Start 拉起实例并等到能登录（崩溃恢复期间端口未开/未就绪都算没起）。
func (m *isolatedMySQL) Start() {
	m.t.Helper()
	m.mu.Lock()
	if m.proc != nil {
		m.mu.Unlock()
		m.t.Fatal("实例已在运行")
	}
	// console 输出落文件而非管道：管道句柄会被子 mysqld 继承，杀完父进程
	// 后 Wait 永远等不到 EOF。
	log, err := os.Create(filepath.Join(m.logDir, fmt.Sprintf("mysqld-%d.log", time.Now().UnixNano())))
	if err != nil {
		m.mu.Unlock()
		m.t.Fatalf("建 mysqld 日志文件: %v", err)
	}
	cmd := exec.Command(m.bin, m.args(
		"--port="+strconv.Itoa(m.port),
		"--bind-address=127.0.0.1",
		"--console",
		"--mysqlx=OFF",
		"--innodb-buffer-pool-size=64M",
		"--character-set-server=utf8mb4",
	)...)
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		log.Close()
		m.mu.Unlock()
		m.t.Fatalf("拉起 mysqld: %v", err)
	}
	p := &mysqlProc{cmd: cmd, log: log, exit: make(chan struct{})}
	m.proc = p
	m.mu.Unlock()
	go func() {
		p.waitErr = cmd.Wait()
		close(p.exit)
	}()

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for {
		db, err := sql.Open("mysql", "root:@tcp("+m.Addr()+")/")
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, lastErr = db.ExecContext(ctx, "SELECT 1")
			cancel()
			db.Close()
			if lastErr == nil {
				return
			}
		} else {
			lastErr = err
		}
		select {
		case <-p.exit:
			m.t.Fatalf("mysqld 启动即退出: %v\n%s", p.waitErr, m.logTail(p))
		default:
		}
		if time.Now().After(deadline) {
			m.t.Fatalf("mysqld 90s 内未就绪: %v\n%s", lastErr, m.logTail(p))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// logTail 某个进程 console 输出的末尾（失败入证）。
func (m *isolatedMySQL) logTail(p *mysqlProc) string {
	name := p.log.Name()
	data, err := os.ReadFile(name)
	if err != nil {
		return err.Error()
	}
	if len(data) > 2000 {
		data = data[len(data)-2000:]
	}
	return string(data)
}

// Kill 杀库（模拟崩溃）并等进程树退出——句柄释放后才能用同一数据目录
// 再拉起。幂等；测试收尾也走它。
func (m *isolatedMySQL) Kill() {
	m.mu.Lock()
	p := m.proc
	m.proc = nil
	m.mu.Unlock()
	if p == nil {
		return
	}
	pid := strconv.Itoa(p.cmd.Process.Pid)
	if runtime.GOOS == "windows" {
		// /T 连子 mysqld 一起杀：只杀父进程的话真服务进程还活着、占着
		// 端口与数据目录。
		_ = exec.Command("taskkill", "/F", "/T", "/PID", pid).Run()
	} else {
		_ = p.cmd.Process.Kill()
	}
	<-p.exit
	p.log.Close()
}

// mustExec 以给定 DSN 执行一条管理语句。
func mustExec(t *testing.T, dsn, query string) {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// TestMySQLStoreRealRestart 真 mysqld 杀库重启端到端。验证四件事：
//   - 重启窗口内的存档挂起重发，不丢（背景存档流零失败）；
//   - 窗口内的首登挂起重发，库回来后照常放行；
//   - 库没起时启动连库等到库起，而不是 Ping 一次就退出；
//   - 库回来后 Ping 转绿、数据是窗口内最后写入的那份。
func TestMySQLStoreRealRestart(t *testing.T) {
	m := newIsolatedMySQL(t)
	t.Setenv("NINEYIN_DB_RETRY_TIMEOUT", "120s") // 崩溃恢复时长不定，放宽单次重试总时限
	m.Init()
	m.Start()
	mustExec(t, "root:@tcp("+m.Addr()+")/", "CREATE DATABASE IF NOT EXISTS nx_retry_test")
	dsn := "root:@tcp(" + m.Addr() + ")/nx_retry_test"
	logf := func(format string, args ...any) { t.Logf(format, args...) }
	store, err := OpenMySQLStore(context.Background(), dsn, logf)
	if err != nil {
		t.Fatalf("连隔离库: %v", err)
	}
	defer store.Close()

	// 场景号：高 16 位 = server_id，取 3 万段（nx_chunks.scene_id 是 INT，
	// 6 万段会溢出有符号 32 位）；块号固定——隔离库里无碰撞之虞。
	scene := uint32(30001)<<16 | 777
	load := func(id uint32) string {
		t.Helper()
		got, err := store.LoadChunk(scene, []uint32{id})
		if err != nil {
			t.Fatalf("回读块 %d: %v", id, err)
		}
		return string(got[id])
	}
	if err := store.SaveChunk(scene, 1, []byte("before")); err != nil {
		t.Fatalf("预热存档: %v", err)
	}

	// ---- 第一轮：存档途中杀库 ----
	var (
		stop     = make(chan struct{})
		wg       sync.WaitGroup
		failures atomic.Int64
	)
	for i := 0; i < 4; i++ { // 背景存档流：模拟在线玩家的周期存档
		id := uint32(100 + i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; ; n++ {
				select {
				case <-stop:
					return
				default:
				}
				if err := store.SaveChunk(scene, id, []byte(fmt.Sprintf("v%d", n))); err != nil {
					failures.Add(1)
					t.Logf("背景存档块 %d 第 %d 次失败: %v", id, n, err)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}()
	}
	time.Sleep(100 * time.Millisecond)

	m.Kill()
	saved := make(chan error, 1)
	go func() { saved <- store.SaveChunk(scene, 1, []byte("during-restart")) }()
	select {
	case err := <-saved:
		t.Fatalf("库已停，窗口内的存档不该立即返回: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	m.Start()
	if err := <-saved; err != nil {
		t.Fatalf("窗口内的存档应重发成功: %v", err)
	}
	close(stop)
	wg.Wait()
	if n := failures.Load(); n != 0 {
		t.Errorf("重启窗口内背景存档失败 %d 次", n)
	}
	if got := load(1); got != "during-restart" {
		t.Errorf("重启窗口内的存档丢失，回读 %q", got)
	}
	if err := store.Ping(); err != nil {
		t.Errorf("库恢复后 Ping 应转绿: %v", err)
	}

	// ---- 第二轮：登录途中杀库 ----
	m.Kill()
	logon := make(chan error, 1)
	go func() {
		_, ok, err := store.AccountLogon("retry-user", "pw", "127.0.0.1", 7)
		if err == nil && !ok {
			err = fmt.Errorf("首登被拒")
		}
		logon <- err
	}()
	select {
	case err := <-logon:
		t.Fatalf("库已停，窗口内的登录不该立即返回: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	m.Start()
	if err := <-logon; err != nil {
		t.Errorf("窗口内的首登应重发成功: %v", err)
	}

	// ---- 第三轮：库没起时启动连库 ----
	m.Kill()
	opened := make(chan error, 1)
	go func() { // 临时句柄，只为验证启动等待
		db, err := OpenMySQLStore(context.Background(), dsn, logf)
		if err == nil {
			db.Close()
		}
		opened <- err
	}()
	select {
	case err := <-opened:
		t.Fatalf("库没起，启动连库不该返回: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	m.Start()
	if err := <-opened; err != nil {
		t.Errorf("库起后启动连库应成功: %v", err)
	}
	if err := store.Ping(); err != nil {
		t.Errorf("老句柄应随库恢复自愈: %v", err)
	}
}
