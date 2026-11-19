package snaileditor

import (
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/snaileditornet"
)

// Module 官方 SnailEditorServerModule（游戏宿主内的逻辑模块）等价：
// 持 MsgManager、延迟 SVN 路径表、EditorUpdate 命令分发表，
// 并在 StartSnailEditor 时把自身回调注入 Net 客户端。
type Module struct {
	mu      sync.Mutex
	started bool
	msgs    *MsgManager

	// 需要延迟 SVN 更新的路径（官方 m_vNeedSvnUpdatePathsLowerCase）。
	needSvnUpdate []string

	// Reload 目标：m_mapUpdate 等价（路径前缀 → 重载动作）。
	reload map[string]func(paths []string)

	// 命令分发表（EditorUpdate::CustomEditor 命令面）。
	commands map[string]func(args []string) string

	// Trace 官方 kernel->vt[0x4C] 文本面。
	Trace func(format string, args ...any)

	net *snaileditornet.Client
}

// New 构造（Init 等价：不依赖 kernel 句柄，事件/心跳由宿主驱动）。
func New() *Module {
	m := &Module{
		msgs:     NewMsgManager(),
		reload:   map[string]func(paths []string){},
		commands: map[string]func(args []string) string{},
		Trace:    func(f string, a ...any) { log.Printf(f, a...) },
	}
	m.registerDefaultCommands()
	return m
}

// Msgs 暴露 MsgManager（宿主 HB 泵或测试用）。
func (m *Module) Msgs() *MsgManager { return m.msgs }

// SetReloadHandler 注册重载动作（m_mapUpdate[路径前缀]=动作）。
// 路径前缀匹配（大小写/分隔符不敏感）。
func (m *Module) SetReloadHandler(prefix string, f func(paths []string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reload[foldLower(prefix)] = f
}

// PushMsg 官方 SnailEditorServerModule::PushMsg(len, msg)（Net type=1 回调，
// cdecl 跨 DLL 边界）：0x800 截断 → g_buf → MsgManager.PushMsg。
func (m *Module) PushMsg(n int, msg string) {
	if len(msg) >= 0x800 {
		msg = msg[:0x800]
	}
	m.Trace("(SnailEditorServerModule::PushMsg) buf:%s", msg)
	m.msgs.PushMsg(0, 0, msg)
}

// FuncDelaySvnUpdateCallback 官方延迟 SVN 更新回调（Net type=0）：
// 并入 m_vNeedSvnUpdatePathsLowerCase。
func (m *Module) FuncDelaySvnUpdateCallback(x int, path string) {
	p := foldLower(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.needSvnUpdate {
		if e == p {
			return
		}
	}
	m.needSvnUpdate = append(m.needSvnUpdate, p)
	m.Trace("delay update: %s", path)
}

// TakeNeedSvnUpdates 取出并清空延迟路径表（帧边界 SvnUpdate 用）。
func (m *Module) TakeNeedSvnUpdates() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.needSvnUpdate
	m.needSvnUpdate = nil
	return out
}

// IsFileInReloadPath 官方语义：路径命中任一注册的重载前缀。
func (m *Module) IsFileInReloadPath(path string) bool {
	p := foldLower(path)
	m.mu.Lock()
	defer m.mu.Unlock()
	for prefix := range m.reload {
		if prefix == "" || strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// HB_ProcessMsg 官方心跳（每帧）：PopMsg → 按 '*' 分割 → 过滤 →
// EditorUpdate::Reload。返回是否处理了消息。
func (m *Module) HB_ProcessMsg() bool {
	msg, ok := m.msgs.PopMsg()
	if !ok {
		return false
	}
	var files []string
	for _, f := range strings.Split(msg.Data, "*") {
		f = strings.TrimSpace(f)
		if f != "" && m.IsFileInReloadPath(f) {
			files = append(files, f)
		}
	}
	m.Reload(files)
	return true
}

// Reload 官方 EditorUpdate::Reload：按前缀分组分发到注册动作。
func (m *Module) Reload(files []string) {
	if len(files) == 0 {
		return
	}
	m.mu.Lock()
	prefixes := make([]string, 0, len(m.reload))
	for p := range m.reload {
		prefixes = append(prefixes, p)
	}
	handlers := make(map[string]func([]string), len(m.reload))
	for p, f := range m.reload {
		handlers[p] = f
	}
	m.mu.Unlock()
	// 长前缀优先，避免 "" 兜底吞掉全部。
	sort.Sort(sort.Reverse(sort.StringSlice(prefixes)))
	for _, p := range prefixes {
		var group []string
		for _, f := range files {
			if p == "" || strings.HasPrefix(foldLower(f), p) {
				group = append(group, f)
			}
		}
		if len(group) > 0 && handlers[p] != nil {
			handlers[p](group)
		}
	}
}

// RunCommand EditorUpdate::CustomEditor 等价：命令名 + 参数列表分发。
// 未注册命令返回空串 + false。
func (m *Module) RunCommand(name string, args []string) (string, bool) {
	m.mu.Lock()
	f := m.commands[name]
	m.mu.Unlock()
	if f == nil {
		return "", false
	}
	return f(args), true
}

// RegisterCommand 注册编辑器命令（宿主/玩法层扩展）。
func (m *Module) RegisterCommand(name string, f func(args []string) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands[name] = f
}

// registerDefaultCommands 官方 OnCustomEditor* 命令面骨架
// （真实落盘/内核效果由宿主注入，此处保留语义与回执）。
func (m *Module) registerDefaultCommands() {
	ack := func(s string) string { return s }
	m.commands["CustomEditorStart"] = func([]string) string { return ack("start") }
	m.commands["CustomEditorEnd"] = func([]string) string { return ack("end") }
	m.commands["CustomEditorInfoUpdate"] = func(a []string) string { return ack(strings.Join(a, ",")) }
	m.commands["CustomEditorReloadRes"] = func(a []string) string {
		m.PushMsg(len(strings.Join(a, "*")), strings.Join(a, "*"))
		return ack("reload-res")
	}
	for _, kind := range []string{"", "Base", "Ai", "Task", "Event", "CustomEvent"} {
		kind := kind
		m.commands["CustomEditorWriteObject"+kind] = func(a []string) string {
			return ack("write-object-" + strings.ToLower(kind) + ":" + strings.Join(a, ","))
		}
	}
	for _, op := range []string{"Write", "Del", "Add", "End"} {
		op := op
		m.commands["CustomEditorIni"+op] = func(a []string) string {
			return ack("ini-" + strings.ToLower(op) + ":" + strings.Join(a, ","))
		}
	}
}

// StartSnailEditor 官方 StartSnailEditor 装配链等价（Net 客户端注入回调）。
// cfg 为官方 xml 配置；addr 为编辑器 GUI 的 "ip:port"。
func (m *Module) StartSnailEditor(cfg ModuleXML, addr string) (started bool, err error) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return true, nil
	}
	m.started = true
	m.mu.Unlock()

	defer func() {
		if err != nil {
			m.mu.Lock()
			m.started = false
			m.mu.Unlock()
		}
	}()

	cli := snaileditornet.NewClient(snaileditornet.Config{
		SvnServerIP:   cfg.Config.SvnServerIP,
		SvnServerPort: cfg.Config.SvnServerPort,
		SvnLocalRoot:  cfg.Config.SvnLocalRootPath,
		WatchPaths:    cfg.WatchPaths(),
	}, snaileditornet.Callbacks{
		OnDelaySvnUpdate:    func(p string) { m.FuncDelaySvnUpdateCallback(0, p) },
		OnEditorCommand:     func(d string) { m.PushMsg(len(d), d) },
		OnSvnUpdateComplete: func(ps []string) { m.Trace("SvnUpdateFinish %v", ps) },
		OnConnState: func(up bool, remote string) {
			if up {
				m.Trace("connected %s", remote)
			} else {
				m.Trace("disconnected")
			}
		},
	})
	if err := cli.Connect(addr); err != nil {
		m.Trace("(SnailEditorServerModule::StartSnailEditor) gConnect error: %v", err)
		return false, err
	}
	cli.StartMsgLoop()
	if err := cli.StartShareMemLoop(); err != nil {
		m.Trace("(SnailEditorServerModule::StartSnailEditor) init error: %v", err)
		// 共享内存读线程失败不致命（编辑器命令通路仍可用）。
	}
	m.net = cli
	m.Trace("(SnailEditorServerModule::StartSnailEditor) StartSnailEditor complete.")
	return true, nil
}

// NetClient 暴露底层 Net 客户端（广播/发帧）。
func (m *Module) NetClient() *snaileditornet.Client { return m.net }

// Shut 官方 Shut：断 Net。
func (m *Module) Shut() {
	if m.net != nil {
		m.net.Close()
		m.net = nil
	}
}
