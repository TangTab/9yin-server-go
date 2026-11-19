package fsloader

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// FxModule 装载协议常量（对齐官方导出面）。
const (
	// ModuleType 官方 FxModule_GetType 须返回的模块类型串。
	ModuleType = fskit.ModuleType // "FxModule"
	// ModuleVersion 官方模块版本常量（FxKit/FsComm 系同为 100）。
	ModuleVersion = fskit.ModuleVersion
)

// EntityDecl 对齐官方 IEntCreator 声明（fskit.EntityDecl 的装载面）。
type EntityDecl = fskit.EntityDecl

// FuncDecl 对齐官方 IFuncCreator 声明。
type FuncDecl = fskit.ExtFuncDecl

// Module 对齐官方模块 DLL 的 FxModule 8 导出装载协议：
// GetType/GetVersion/GetSpace/Init/GetEntCreator/GetFuncCreator/
// GetIntCreator/GetLogicCreator。Go 侧以接口方法等价（进程内注册，
// 官方为 CRT 静态构造即注册——装载时链表已就绪，语义相同）。
type Module interface {
	// Name 模块名（官方即 DLL 文件名，如 "fskit.dll"；大小写不敏感装载）。
	Name() string
	// Type 模块类型（须为 ModuleType="FxModule"）。
	Type() string
	// Version 模块版本（CDllManager 版本门）。
	Version() int
	// Space 模块域（官方 FxModule_GetSpace，命名空间串）。
	Space() string
	// Init 官方 FxModule_Init(g_core)：内核句柄注入，唯一动作。
	Init(core Core) error
	// Entities 实体 creator 表（官方 GetEntCreator 拉取的链表数据）。
	Entities() []EntityDecl
	// Funcs 全局扩展函数表（官方 GetFuncCreator；无则 nil）。
	Funcs() []FuncDecl
}

// DeclModule 以声明表直接拼装的 Module（模块适配器：把既有的
// fskit.EntityTypes/ExtFuncs 等声明接进装载协议）。
type DeclModule struct {
	ModuleName    string
	ModuleSpace   string
	ModuleVersion int
	ModuleInit    func(core Core) error
	ModuleEnts    []EntityDecl
	ModuleFuncs   []FuncDecl
}

// Name 实现 Module。
func (m *DeclModule) Name() string { return m.ModuleName }

// Type 实现 Module。
func (m *DeclModule) Type() string { return ModuleType }

// Version 实现 Module。
func (m *DeclModule) Version() int {
	if m.ModuleVersion == 0 {
		return ModuleVersion
	}
	return m.ModuleVersion
}

// Space 实现 Module。
func (m *DeclModule) Space() string { return m.ModuleSpace }

// Init 实现 Module。
func (m *DeclModule) Init(core Core) error {
	if m.ModuleInit != nil {
		return m.ModuleInit(core)
	}
	return nil
}

// Entities 实现 Module。
func (m *DeclModule) Entities() []EntityDecl { return m.ModuleEnts }

// Funcs 实现 Module。
func (m *DeclModule) Funcs() []FuncDecl { return m.ModuleFuncs }

// ModuleRegistry 复刻官方 FxCore 的 CDllManager：模块装载器。
// 官方 Load 的四重校验（module interface not found / version not match /
// already loaded / repeat * name）与收编注册表（实体/函数/接口/逻辑名）
// 逐条对齐；Go 侧为编译期注册 + Startup 期装载。
type ModuleRegistry struct {
	mu       sync.Mutex
	factory  map[string]Module // key = 小写模块名
	loaded   map[string]bool   // 官方防重复装载
	entNames map[string]string // 实体名 → 模块名（repeat entity name 门）
	funcNames map[string]string
	logicNames map[string]string
	intNames  map[string]string // 官方接口 creator 域
}

// NewModuleRegistry 创建空注册表。
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		factory:   make(map[string]Module),
		loaded:    make(map[string]bool),
		entNames:  make(map[string]string),
		funcNames: make(map[string]string),
		logicNames: make(map[string]string),
		intNames:  make(map[string]string),
	}
}

// Register 编译期注册模块工厂（官方等价：DLL 放在部署目录）。
// 重复注册为装配错误（官方目录里同名 DLL 二选一）。
func (r *ModuleRegistry) Register(m Module) error {
	if m.Type() != ModuleType {
		return fmt.Errorf("fsloader: module type not %s: %s", ModuleType, m.Name())
	}
	key := lower(m.Name())
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.factory[key]; dup {
		return fmt.Errorf("fsloader: module already registered: %s", m.Name())
	}
	r.factory[key] = m
	return nil
}

// Load 复刻官方 CDllManager::Load(name)（Startup 期逐模块调用）：
// 未注册=module interface not found；类型/版本不符=not match；
// 重复装载=already loaded；实体/函数重名=repeat * name。
func (r *ModuleRegistry) Load(name string, core Core) (Module, error) {
	key := lower(name)
	r.mu.Lock()
	m, ok := r.factory[key]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("(CDllManager::Load)module interface not found: %s", name)
	}
	if m.Type() != ModuleType {
		r.mu.Unlock()
		return nil, fmt.Errorf("(CDllManager::Load)module type error: %s", name)
	}
	if m.Version() != ModuleVersion {
		r.mu.Unlock()
		return nil, fmt.Errorf("(CDllManager::Load)module version not match: %s (%d≠%d)",
			name, m.Version(), ModuleVersion)
	}
	if r.loaded[key] {
		r.mu.Unlock()
		return nil, fmt.Errorf("(CDllManager::Load)module already loaded: %s", name)
	}
	r.loaded[key] = true
	// 重名收编门（装载序内持锁，检查+登记原子）。
	for _, e := range m.Entities() {
		if owner, dup := r.entNames[lower(e.TypeName)]; dup {
			r.mu.Unlock()
			return nil, fmt.Errorf("(CDllManager::Load)repeat entity name: %s (%s 与 %s)",
				e.TypeName, name, owner)
		}
		r.entNames[lower(e.TypeName)] = name
	}
	for _, f := range m.Funcs() {
		if owner, dup := r.funcNames[lower(f.Name)]; dup {
			r.mu.Unlock()
			return nil, fmt.Errorf("(CDllManager::Load)repeat function name: %s (%s 与 %s)",
				f.Name, name, owner)
		}
		r.funcNames[lower(f.Name)] = name
	}
	r.mu.Unlock()

	// FxModule_Init 在收编后调用（官方装载序：校验→收编→Init）。
	if err := m.Init(core); err != nil {
		return nil, fmt.Errorf("(CDllManager::Load)init failed: %s: %w", name, err)
	}
	return m, nil
}

// LoadAll 复刻官方 Startup 的模块装载段：names 空=全部注册模块
//（官方空 [Module] → 装载 loadall.dll 语义）；装载失败即失败
//（官方文案 "(CDllManager::Load)load module failed" 上抛为致命）。
func (r *ModuleRegistry) LoadAll(names []string, core Core) ([]Module, error) {
	if len(names) == 0 {
		r.mu.Lock()
		names = make([]string, 0, len(r.factory))
		for k := range r.factory {
			names = append(names, k)
		}
		r.mu.Unlock()
		sort.Strings(names) // 全装时按名稳定序
	}
	loaded := make([]Module, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		m, err := r.Load(name, core)
		if err != nil {
			return nil, fmt.Errorf("(CDllManager::Load)load module failed: %w", err)
		}
		loaded = append(loaded, m)
	}
	return loaded, nil
}

// Loaded 已装载模块名表（GetModules 面）。
func (r *ModuleRegistry) Loaded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.loaded))
	for k := range r.loaded {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Registered 已注册（未装载）模块名表（空 [Module]=全部装载的取数面）。
func (r *ModuleRegistry) Registered() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.factory))
	for k := range r.factory {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FindEntity 装载后的实体类型查找（官方 CEntFactory::Create 前置）。
func (r *ModuleRegistry) FindEntity(typeName string) (EntityDecl, bool) {
	r.mu.Lock()
	owner, ok := r.entNames[lower(typeName)]
	m := r.factory[lower(owner)]
	r.mu.Unlock()
	if !ok || m == nil {
		return EntityDecl{}, false
	}
	for _, e := range m.Entities() {
		if lower(e.TypeName) == lower(typeName) {
			return e, true
		}
	}
	return EntityDecl{}, false
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
