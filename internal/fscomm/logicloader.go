package fscomm

import (
	"fmt"
	"sync"
)

// LogicType 官方 FxModule_GetType 的模块类型面（logicloader.cpp 校验
// "FxGameLogic" 类型串后才收编；Go 侧以字符串类型标签等价表达）。
const (
	LogicGame = "FxGameLogic" // 官方唯一收编类型（游戏逻辑模块）
)

// LogicFactory 逻辑模块工厂（官方 LoadLibrary+FxModule_Init 后由
// 宿主拿到模块句柄；Go 侧为进程内工厂函数）。
type LogicFactory func() any

// LogicLoader 复刻官方 LogicLoader：游戏逻辑模块装载器。
// 官方 Load(type, path)：LoadLibrary → 取 FxModule_GetType → 类型
// 非 "FxGameLogic" 报 "(LogicLoader::Load)dll type error" 拒收。
// Go 形态：模块在编译期注册工厂，Load 按名+类型校验后实例化。
type LogicLoader struct {
	mu       sync.Mutex
	factory  map[string]LogicFactory
	typ      map[string]string
	loaded   []any
}

// NewLogicLoader 创建空装载器。
func NewLogicLoader() *LogicLoader {
	return &LogicLoader{factory: make(map[string]LogicFactory), typ: make(map[string]string)}
}

// Register 注册逻辑模块工厂（编译期等价于官方 [Logic] 模块清单：
// game_unite.ini Module0=fsgame.dll / Module1=SnailEditor.dll）。
func (l *LogicLoader) Register(name, typ string, f LogicFactory) error {
	if f == nil {
		return fmt.Errorf("fscomm: nil logic factory %s", name)
	}
	if typ != LogicGame {
		return fmt.Errorf("fscomm: (LogicLoader::Load)dll type error: %s is %q", name, typ)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.factory[name] = f
	l.typ[name] = typ
	return nil
}

// Load 装载并实例化（官方 LogicLoader::Load）。
// 未注册=官方 "(LogicLoader::Load)load module failed"。
func (l *LogicLoader) Load(name string) (any, error) {
	l.mu.Lock()
	f, ok := l.factory[name]
	l.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("fscomm: (LogicLoader::Load)load module failed: %s", name)
	}
	inst := f()
	l.mu.Lock()
	l.loaded = append(l.loaded, inst)
	l.mu.Unlock()
	return inst, nil
}

// Count 已装载实例数（官方 GetCount）。
func (l *LogicLoader) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.loaded)
}

// Modules 已注册模块名清单（官方 GetModules 面向 [Logic] 装配回读）。
func (l *LogicLoader) Modules() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, 0, len(l.factory))
	for name := range l.factory {
		out = append(out, name)
	}
	return out
}
