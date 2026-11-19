package fswatch

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// ProcState 官方 AppManager 视角的受管进程状态。
type ProcState struct {
	Name   string
	Cmd    string
	Handle *exec.Cmd
	Exited bool
}

// ProcManager 官方 AppManager（FsComm 提供，本包内嵌等价）：
// start_app/stop_app/app_list 受面。
type ProcManager struct {
	mu     sync.Mutex
	cfg    *Config
	procs  map[string]*ProcState
	blocks map[string]bool // 拉起失败的 server（no_exe_path 缓存）
	logf   func(string, ...any)
}

// NewProcManager AppManager:Start 等价。
func NewProcManager(cfg *Config, logf func(string, ...any)) *ProcManager {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &ProcManager{cfg: cfg, procs: map[string]*ProcState{}, blocks: map[string]bool{}, logf: logf}
}

// StartApp 官方 ReceiveStartApp（0x7c540）结果机：
// no_exe_path（[Servers]/[ExePath] 缺）→ in_running → succeed/failed；
// 日志 "Start server program"。
func (m *ProcManager) StartApp(server string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cfg.DestPath(server); !ok && !m.hasServer(server) {
		return "no_exe_path"
	}
	if p, ok := m.procs[server]; ok && p.Handle != nil && p.Handle.ProcessState == nil {
		// ProcessState 在 Wait 返回后才非空 → 进程仍在运行。
		return "in_running"
	}
	cmdline := m.serverCmd(server)
	if cmdline == "" {
		return "no_exe_path"
	}
	m.logf("Start server program %s", server)
	parts := strings.Fields(cmdline)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		m.logf("start %s failed: %v", server, err)
		return "failed"
	}
	m.procs[server] = &ProcState{Name: server, Cmd: cmdline, Handle: cmd}
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if p, ok := m.procs[server]; ok {
			p.Exited = true
		}
		m.mu.Unlock()
	}()
	return "succeed"
}

// StopApp 官方 ReceiveStopApp（0x7c900）：not_found → succeed/failed；
// 日志 "Stop server program"。
func (m *ProcManager) StopApp(server string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.procs[server]
	if !ok || p.Handle == nil || p.Handle.Process == nil || p.Exited {
		return "not_found"
	}
	m.logf("Stop server program %s", server)
	if err := p.Handle.Process.Kill(); err != nil {
		return "failed"
	}
	p.Exited = true
	return "succeed"
}

// AppList 官方 ReceiveAppList（0x77830）：AppManager vtable+0x3c 应用表 →
// varlist ["app_list", name, state, ...]（state：running/stopped）。
func (m *ProcManager) AppList() *VarList {
	m.mu.Lock()
	defer m.mu.Unlock()
	l := &VarList{}
	l.AddString("app_list")
	for _, s := range m.cfg.Servers {
		state := "stopped"
		if p, ok := m.procs[s.Name]; ok && !p.Exited {
			state = "running"
		}
		l.AddString(s.Name)
		l.AddString(state)
	}
	return l
}

func (m *ProcManager) hasServer(name string) bool {
	for _, s := range m.cfg.Servers {
		if strings.EqualFold(s.Name, name) {
			return true
		}
	}
	return false
}

func (m *ProcManager) serverCmd(name string) string {
	for _, s := range m.cfg.Servers {
		if strings.EqualFold(s.Name, name) {
			return s.Path
		}
	}
	return ""
}

// String 调试面。
func (m *ProcManager) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	for _, s := range m.cfg.Servers {
		state := "stopped"
		if p, ok := m.procs[s.Name]; ok && !p.Exited {
			state = "running"
		}
		fmt.Fprintf(&b, "%s=%s ", s.Name, state)
	}
	return strings.TrimSpace(b.String())
}
