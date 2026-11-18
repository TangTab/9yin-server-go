package fscomm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// AppInfo 对齐官方 book_data_t 一槽（appbook.cpp；布局见 doc.go）：
// used/name/path/addr/info + 尾部 state/procid。
type AppInfo struct {
	Name   string `json:"name"`   // 槽名（=进程 AppName）
	Path   string `json:"path"`   // 进程可执行文件全路径
	Addr   string `json:"addr"`   // 通讯地址（官方 AppAddress）
	State  int    `json:"state"`  // WAPS_* 进程状态（官方 SetAppState）
	ProcID int    `json:"procid"` // 操作系统进程号
	Started time.Time `json:"started"`
}

// AppManager 复刻官方 AppManager+CAppBook：跨进程应用注册表。
//
// 官方机制：共享内存 "memory_<name>"（0x6208 = 8 头 + 32×0x310 槽）+
// 命名互斥量 "mutex_<name>"（同名互斥=单实例锁）+ DoWait 线程每
// 3000ms 扫描摘除死亡进程槽（互斥量消失判死）。
//
// Go 复刻：注册表 = 目录下一份 JSON 文件（原子重写）；单实例 =
// O_EXCL 锁文件（写 pid，启动时清陈旧锁）；探活 = 进程存在性
// （Windows OpenProcess / POSIX kill 0），判定口径与命名互斥量
// 等价——进程死亡即失效。32 槽上限保留（官方 m_nSize=0x20）。
type AppManager struct {
	name  string // 书名（官方按 AppName 建书；跨进程同书名互见）
	dir   string // 注册表目录
	self  AppInfo

	mu       sync.Mutex
	lockFile string
	stopOnce sync.Once
	stopped  chan struct{}
	wg       sync.WaitGroup
	needCheck bool // 官方 NeedCheckApp：是否启用死亡清扫
}

// MaxApps 官方槽位上限 0x20。
const MaxApps = 32

// sweepInterval 官方 DoWait 周期 3000ms（0x1005cbaa cmp eax, 0xbb8）。
const sweepInterval = 3000 * time.Millisecond

// NewAppManager 打开（或创建）名为 name 的应用注册表。
// dir 空 = os.TempDir()/fscomm-appbook。
// 同名注册表内 AppName 单实例：第二实例 Register 返回 ErrAppExists
// （对齐官方 "(AppManager::Create)app mutex %s exclusive failed"）。
func NewAppManager(name, dir string) (*AppManager, error) {
	if name == "" {
		return nil, errors.New("fscomm: empty app book name")
	}
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "fscomm-appbook")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &AppManager{
		name:     name,
		dir:      dir,
		stopped:  make(chan struct{}),
		needCheck: true,
	}, nil
}

func (m *AppManager) bookPath() string  { return filepath.Join(m.dir, m.name+".json") }
func (m *AppManager) lockPath(l string) string { return filepath.Join(m.dir, m.name+"."+l+".lock") }

// heldLocks 进程内持锁登记：同一进程对同一 (书,AppName) 的重复
// 注册也须拒绝——pid 探活无法区分同进程实例，以此补齐
// （官方 named mutex 天然同进程也互斥）。
var heldLocks sync.Map // absLockPath -> struct{}

// Register 登记自身（官方 AppManager::Create 全链：单实例锁 +
// 建书 + 落槽 {AppName, exe 路径, pid}）。addr 对齐官方 AppAddress。
// 成功后启动 3s 清扫线程（官方 DoWait；StartSweeper 可显式控制）。
func (m *AppManager) Register(appName, addr string, state int) error {
	if appName == "" {
		return errors.New("fscomm: empty app name")
	}
	lock := m.lockPath(appName)
	abs, _ := filepath.Abs(lock)
	if _, held := heldLocks.Load(abs); held {
		return fmt.Errorf("fscomm: (AppManager::Create)app mutex %s exclusive failed", appName)
	}
	pid := os.Getpid()
	// 单实例：锁文件存在且属主进程仍活 → 拒绝；属主已死 → 清陈旧锁
	if data, err := os.ReadFile(lock); err == nil {
		if old, perr := strconv.Atoi(string(data)); perr == nil && old != pid && processAlive(old) {
			return fmt.Errorf("fscomm: (AppManager::Create)app mutex %s exclusive failed", appName)
		}
		os.Remove(lock)
	}
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("fscomm: (AppManager::Create)app mutex %s exclusive failed: %w", appName, err)
	}
	f.WriteString(strconv.Itoa(pid))
	f.Close()
	heldLocks.Store(abs, struct{}{})
	m.mu.Lock()
	m.lockFile = abs
	m.self = AppInfo{
		Name:   appName,
		Path:   osExecutable(),
		Addr:   addr,
		State:  state,
		ProcID: pid,
		Started: time.Now(),
	}
	m.mu.Unlock()
	err = m.mutate(func(apps *[]AppInfo) {
		for i := range *apps {
			if (*apps)[i].Name == appName { // 覆盖旧槽（上次残留）
				(*apps)[i] = m.self
				return
			}
		}
		if len(*apps) >= MaxApps {
			return
		}
		*apps = append(*apps, m.self)
	})
	if err != nil {
		return err
	}
	if m.needCheck {
		m.StartSweeper()
	}
	return nil
}

// StartSweeper 启动官方 DoWait 线程（幂等）。Register 已自动调用。
func (m *AppManager) StartSweeper() {
	m.wg.Add(1)
	go m.sweepLoop()
}

func (m *AppManager) sweepLoop() {
	defer m.wg.Done()
	t := time.NewTicker(sweepInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.Sweep()
		case <-m.stopped:
			return
		}
	}
}

// Sweep 单轮清扫：摘除死亡进程槽（官方 DoWait 主体）。
func (m *AppManager) Sweep() {
	_ = m.mutate(func(apps *[]AppInfo) {
		kept := (*apps)[:0]
		for _, a := range *apps {
			if a.ProcID == 0 || processAlive(a.ProcID) {
				kept = append(kept, a)
			}
		}
		*apps = kept
	})
}

// SetAppState 更新自身状态（官方 SetAppState；WAPS_* 语义由宿主定义）。
func (m *AppManager) SetAppState(state int) error {
	m.mu.Lock()
	m.self.State = state
	self := m.self
	m.mu.Unlock()
	return m.mutate(func(apps *[]AppInfo) {
		for i := range *apps {
			if (*apps)[i].Name == self.Name {
				(*apps)[i].State = state
			}
		}
	})
}

// SetAppInfo 更新自身地址等（官方 SetAppInfo）。
func (m *AppManager) SetAppInfo(addr string) error {
	m.mu.Lock()
	m.self.Addr = addr
	m.mu.Unlock()
	return m.mutate(func(apps *[]AppInfo) {
		for i := range *apps {
			if (*apps)[i].Name == m.selfName() {
				(*apps)[i].Addr = addr
			}
		}
	})
}

func (m *AppManager) selfName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.self.Name
}

// GetAppList 全量快照（官方 GetAppList → TVarList）。
func (m *AppManager) GetAppList() []AppInfo {
	apps := m.load()
	out := make([]AppInfo, len(apps))
	copy(out, apps)
	return out
}

// GetAppCount 官方 GetAppCount。
func (m *AppManager) GetAppCount() int { return len(m.load()) }

// GetAppAddress 按名查地址（官方 GetAppAddress）。
func (m *AppManager) GetAppAddress(name string) (string, bool) {
	for _, a := range m.load() {
		if a.Name == name {
			return a.Addr, true
		}
	}
	return "", false
}

func (m *AppManager) load() []AppInfo {
	data, err := os.ReadFile(m.bookPath())
	if err != nil {
		return nil
	}
	var apps []AppInfo
	if json.Unmarshal(data, &apps) != nil {
		return nil
	}
	return apps
}

// mutate 读-改-写；compare-and-swap 语义靠临时文件+原子改名。
func (m *AppManager) mutate(fn func(apps *[]AppInfo)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	apps := m.loadRaw()
	fn(&apps)
	data, err := json.Marshal(apps)
	if err != nil {
		return err
	}
	tmp := m.bookPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.bookPath())
}

func (m *AppManager) loadRaw() []AppInfo {
	data, err := os.ReadFile(m.bookPath())
	if err != nil {
		return nil
	}
	var apps []AppInfo
	json.Unmarshal(data, &apps)
	return apps
}

// Close 摘除自身槽并释放单实例锁（官方 AppManager::Destroy）。
func (m *AppManager) Close() {
	m.stopOnce.Do(func() { close(m.stopped) })
	self := m.selfName()
	_ = m.mutate(func(apps *[]AppInfo) {
		kept := (*apps)[:0]
		for _, a := range *apps {
			if a.Name != self {
				kept = append(kept, a)
			}
		}
		*apps = kept
	})
	m.mu.Lock()
	lock := m.lockFile
	m.lockFile = ""
	m.mu.Unlock()
	if lock != "" {
		heldLocks.Delete(lock)
		os.Remove(lock)
	}
	m.wg.Wait()
}

// processAlive 进程存在性探活（官方 named-mutex 存在性的等价口径），
// 平台实现见 processAlive_*.go。
func osExecutable() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	return p
}
