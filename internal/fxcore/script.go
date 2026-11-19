package fxcore

import (
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ScriptHost 脚本宿主面（官方 CLuaScript；Lua VM=里程碑二）。
// Go 侧先以注册回调承接：装载=读文件+mtime 登记，执行位由宿主实现。
type ScriptHost interface {
	// Load 装载脚本文件（返回脚本内部句柄；nil=失败=官方 load failed）。
	Load(name, file string) any
	// Call 调用脚本函数（args/result 透传；false=官方 run failed）。
	Call(handle any, function string, args []any, result *[]any) bool
	// Reload 重载脚本（mtime 变更后的再装载）。
	Reload(handle any, name, file string) bool
}

// Script 复刻官方脚本登记项（CScpManager 表项）。
type Script struct {
	Name    string
	File    string
	Handle  any // 宿主句柄（Lua里程碑二前为 nil）
	ModTime time.Time
	Loads   int // 重载计数（观测面）
}

// ScriptManager 复刻 CScpManager：脚本注册表 + mtime 热重载。
// 官方 Execute（泵每拍）对全部脚本 ReloadScript——内部比对 mtime，
// 变更才真重载（[Script] Reload=1 开关在 Core 层守门）。
type ScriptManager struct {
	mu      sync.Mutex
	scripts map[string]*Script // key=小写名
	order   []string
	host    ScriptHost
}

// NewScriptManager 创建空表（host 可为 nil=仅登记不执行）。
func NewScriptManager(host ScriptHost) *ScriptManager {
	return &ScriptManager{scripts: make(map[string]*Script)}
}

// SetHost 更换宿主。
func (m *ScriptManager) SetHost(h ScriptHost) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.host = h
}

// AddScript 复刻 CScpManager::AddScript：装载并登记。
// 文件缺失=装载失败（官方 "(CLuaHelper::LoadFile)open file %s failed"）。
func (m *ScriptManager) AddScript(name, file string) (*Script, error) {
	fi, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(name)
	if _, dup := m.scripts[key]; dup {
		return m.scripts[key], nil
	}
	s := &Script{Name: name, File: file, ModTime: fi.ModTime()}
	if m.host != nil {
		s.Handle = m.host.Load(name, file)
	}
	m.scripts[key] = s
	m.order = append(m.order, key)
	return s, nil
}

// FindScript 复刻官方查找（命中返回登记项）。
func (m *ScriptManager) FindScript(name string) (*Script, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.scripts[strings.ToLower(name)]
	return s, ok
}

// Call 经宿主调用脚本函数（官方 nx_call/nx_execute 面）。
func (m *ScriptManager) Call(name, function string, args []any, result *[]any) bool {
	s, ok := m.FindScript(name)
	if !ok {
		return false
	}
	m.mu.Lock()
	host := m.host
	m.mu.Unlock()
	if host == nil || s.Handle == nil {
		return false
	}
	return host.Call(s.Handle, function, args, result)
}

// Execute 复刻 CScpManager::Execute（泵每拍）：mtime 轮询热重载。
// 返回本拍重载数。hostReload=false 时仅观测不重载（宿主缺席）。
func (m *ScriptManager) Execute() int {
	m.mu.Lock()
	keys := make([]string, len(m.order))
	copy(keys, m.order)
	scripts := make([]*Script, 0, len(keys))
	for _, k := range keys {
		if s := m.scripts[k]; s != nil {
			scripts = append(scripts, s)
		}
	}
	host := m.host
	m.mu.Unlock()

	reloaded := 0
	for _, s := range scripts {
		fi, err := os.Stat(s.File)
		if err != nil || fi.ModTime().Equal(s.ModTime) {
			continue
		}
		s.ModTime = fi.ModTime()
		if host != nil && s.Handle != nil {
			host.Reload(s.Handle, s.Name, s.File)
		}
		s.Loads++
		reloaded++
	}
	return reloaded
}

// Snapshot 登记名快照。
func (m *ScriptManager) Snapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.order))
	for _, k := range m.order {
		if s := m.scripts[k]; s != nil {
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}
