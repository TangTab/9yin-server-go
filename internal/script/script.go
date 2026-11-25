package script

// script.go Lua 虚拟机：装载、调用、预算与错误隔离。
//
// 一台 VM 进程内单实例，调用串行（整台 VM 一把互斥锁）。两套入口：公开
// 入口（Function / Call / RunFile）自带加锁；Lua 回调内部（nx_execute /
// require / dofile 再入）一律走不加锁的 *Locked 变体——此时锁已由外层
// 调用持有。每次调用带超时预算（gopher-lua 的 context 检查点），脚本死
// 循环不会拖住战斗链。
//
// 错误隔离（与 2433 线 internal/combatcalc 同策略）：运行期出错只禁用
// **出错的那份脚本**（其后的调用直接返回 ErrScriptDisabled），引擎与其余
// 脚本不受影响——服务器永不因脚本层停摆，调用方按回退面结算。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Options VM 装配项。
type Options struct {
	// LuaDir 脚本根（官方 Res/lua 同构：define.lua / Rule / skill / buf /
	// share …）。空串 = 只有内置引擎，脚本查询一律落空。
	LuaDir string
	// CallTimeout 单次调用预算（缺省 50ms）。
	CallTimeout time.Duration
	// Logf 日志钩。
	Logf func(format string, args ...any)
}

// 错误域。
var (
	// ErrScriptDisabled 脚本此前运行期出错，已被禁用。
	ErrScriptDisabled = errors.New("script: 脚本已禁用（此前运行期出错）")
	// ErrNoScripts 未配置脚本根。
	ErrNoScripts = errors.New("script: 未配置脚本根")
)

// Func 脚本函数句柄：绑定所属脚本，出错只禁用该脚本。
type Func struct {
	vm  *VM
	key string // 脚本键（显示用）
	id  string // 失败身份（sourceID）
	fn  *lua.LFunction
	// reentrant 经 Reentrant 取得：宿主回调内调用，VM 锁已由外层调用
	// 持有，Call 不再重复加锁。
	reentrant bool
}

// Caller 脚本调用面（取脚本函数）。两种实现对应两种语境，不可混用：
//
//	*VM        普通协程（未持 VM 锁）：Function 加锁；
//	Reentrant  宿主回调内（World 的 nx_* 落点——回调由 VM 在持锁协程上
//	           同步调用）：不加锁。回调里用 *VM 会自锁（VM 锁不可重入）。
type Caller interface {
	Function(script, name string) *Func
}

// Reentrant 宿主回调语境的脚本调用面。只能在世界回调（及其同步调用到的
// 宿主代码）内部使用；在别的协程上用会与持锁协程并发访问 VM。
type Reentrant struct{ VM *VM }

// Function 取脚本函数（不加锁；所得到的 Func.Call 亦不加锁）。
func (r Reentrant) Function(script, name string) *Func {
	if r.VM == nil {
		return nil
	}
	fn := r.VM.functionLocked(script, name)
	if fn != nil {
		fn.reentrant = true
	}
	return fn
}

// VM 一台 Lua 虚拟机。
type VM struct {
	world World
	opt   Options

	mu sync.Mutex // 整台 VM 一把锁

	l       *lua.LState
	loaded  map[string]bool
	failed  map[string]bool
	missed  map[string]bool
	stubbed map[string]bool
	envs    map[string]*lua.LTable // 钩子脚本的函数环境（键 = 脚本键）
}

// New 创建 VM 并登记 nx_* API 面。
func New(world World, opt Options) *VM {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.CallTimeout <= 0 {
		opt.CallTimeout = 50 * time.Millisecond
	}
	vm := &VM{world: world, opt: opt,
		loaded: map[string]bool{}, failed: map[string]bool{}, missed: map[string]bool{}, stubbed: map[string]bool{},
		envs: map[string]*lua.LTable{}}
	vm.l = lua.NewState()
	vm.register()
	return vm
}

// Close 释放 VM。
func (vm *VM) Close() { vm.l.Close() }

func (vm *VM) trace(format string, args ...any) { vm.opt.Logf(format, args...) }

// ---- 锁 ----

func (vm *VM) lock()   { vm.mu.Lock() }
func (vm *VM) unlock() { vm.mu.Unlock() }

// ---- 脚本装载 ----

// scriptPath 归一脚本键：反斜杠→斜杠、小写、去 .lua 后缀。官方脚本写法
// 兼容：require("share\\skill_define") / nx_execute("rull\\skill_effect.lua")
// / 技能表 LuaScript 列 "skill\jn_normal\…\x.lua"。
func scriptPath(name string) string {
	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if strings.HasPrefix(key, "lua/") {
		key = strings.TrimPrefix(key, "lua/")
	}
	return strings.TrimSuffix(key, ".lua")
}

func (vm *VM) fileOf(key string) string {
	return filepath.Join(vm.opt.LuaDir, filepath.FromSlash(key+".lua"))
}

// readScript 经资源仓读脚本（编码自适应：合法 UTF-8 直通，否则 GBK）。
func (vm *VM) readScript(key string) ([]byte, error) {
	if vm.opt.LuaDir == "" {
		return nil, ErrNoScripts
	}
	raw, err := res.ReadFile(vm.fileOf(key))
	if err != nil {
		return nil, err
	}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	if !utf8.Valid(raw) {
		if decoded, derr := simplifiedchinese.GBK.NewDecoder().Bytes(raw); derr == nil {
			raw = decoded
		}
	}
	return raw, nil
}

// Has 脚本文件是否存在（不装载）。
func (vm *VM) Has(script string) bool {
	_, err := vm.readScript(scriptPath(script))
	return err == nil
}

// globalEnv 全局环境的环境键（见 sourceID）。
const globalEnv = ""

// sourceID 一份源的身份 = 所属环境 + 源键。官方宿主里每份技能 / buff 脚本
// 是一个独立的 Lua 上下文：脚本都把 get_damage / skill_before_hit 定义在
// 文件作用域，require 进来的模块全局（define 常量、同名的 AddBuff）也
// 各归各。本 VM 用「每份钩子脚本一张函数环境」等价实现——环境是独立的
// 表，__index 指向全局表（nx_* 与内置引擎读得到），写只落在本环境；
// 环境里的 require / dofile / loadfile 是绑定本环境的版本。环境键为空 =
// 全局环境（RunFile 装的模块、内置引擎）。
func sourceID(env, key string) string { return env + "\x00" + key }

// runFile 把文件当**模块**执行到全局环境（require / dofile / RunFile）。
// 调用方持锁。
func (vm *VM) runFile(key string) error { return vm.runInto(globalEnv, key) }

// runHook 装载钩子脚本：执行到它自己的函数环境。调用方持锁。
func (vm *VM) runHook(key string) error { return vm.runInto(key, key) }

// runInto 读文件 key 执行到环境 env（幂等；缺席或装载失败记失败不再重试）。
// 调用方持锁。
func (vm *VM) runInto(env, key string) error {
	id := sourceID(env, key)
	if vm.failed[id] {
		return ErrScriptDisabled
	}
	if vm.loaded[id] {
		return nil
	}
	raw, err := vm.readScript(key)
	if err != nil {
		vm.failed[id] = true
		if !vm.missed[key] {
			vm.missed[key] = true
			vm.trace("script: 脚本缺席 %s", key)
		}
		return err
	}
	target := (*lua.LTable)(nil)
	if env != globalEnv {
		target = vm.hookEnv(env)
	}
	return vm.runSource(id, key, raw, target)
}

// runSource 把一段源执行到指定环境（env 为 nil = 全局环境）。幂等；出错只
// 记失败本身份。调用方持锁。
func (vm *VM) runSource(id, name string, raw []byte, env *lua.LTable) error {
	if vm.failed[id] {
		return ErrScriptDisabled
	}
	if vm.loaded[id] {
		return nil
	}
	fn, err := vm.l.Load(bytes.NewReader(raw), name)
	if err != nil {
		vm.failed[id] = true
		vm.trace("script: 脚本编译失败 %s: %v", name, err)
		return err
	}
	if env != nil {
		vm.l.SetFEnv(fn, env)
	}
	vm.l.Push(fn)
	vm.loaded[id] = true // 无论执行成败都只执行一次
	if err := vm.l.PCall(0, lua.MultRet, nil); err != nil {
		vm.failed[id] = true
		vm.l.SetTop(0)
		vm.trace("script: 脚本执行失败 %s: %v", name, err)
		return err
	}
	vm.l.SetTop(0)
	return nil
}

// hookEnv 取（或建）钩子脚本的函数环境；require / dofile / loadfile 绑定到
// 本环境。调用方持锁。
func (vm *VM) hookEnv(owner string) *lua.LTable {
	if env, ok := vm.envs[owner]; ok {
		return env
	}
	L := vm.l
	env, meta := L.NewTable(), L.NewTable()
	meta.RawSetString("__index", L.Get(lua.GlobalsIndex))
	env.Metatable = meta
	bind := func(fn func(*lua.LState, string) int) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int { return fn(L, owner) })
	}
	env.RawSetString("require", bind(vm.fnRequireIn))
	env.RawSetString("dofile", bind(vm.fnDofileIn))
	env.RawSetString("loadfile", bind(vm.fnLoadfileIn))
	vm.envs[owner] = env
	return env
}

// RunFile 把脚本文件当模块执行（外部入口；require 的 Go 侧等价）。
func (vm *VM) RunFile(script string) error {
	vm.lock()
	defer vm.unlock()
	return vm.runFile(scriptPath(script))
}

// RunFileSource 以键 key 执行一段脚本源，语义同钩子脚本（自己的函数环境；
// 幂等与错误隔离同 runInto）——测试夹具用；文件脚本一律经 Function /
// RunFile 按路径装载。
func (vm *VM) RunFileSource(key string, source string) error {
	vm.lock()
	defer vm.unlock()
	key = scriptPath(key)
	return vm.runSource(sourceID(key, key), key, []byte(source), vm.hookEnv(key))
}

// Function 取脚本内函数（首次调用装载脚本）。脚本缺席、已禁用或无此
// 函数返回 nil——调用方按「无钩子」处理。
func (vm *VM) Function(script, name string) *Func {
	vm.lock()
	defer vm.unlock()
	return vm.functionLocked(script, name)
}

// functionLocked Function 的不加锁变体（调用方持锁）。只认本脚本（及它
// require 进来的模块）在自己的环境里定义的函数——别的脚本的同名全局
// 函数读不到。
func (vm *VM) functionLocked(script, name string) *Func {
	key := scriptPath(script)
	id := sourceID(key, key)
	if err := vm.runHook(key); err != nil && vm.failed[id] {
		return nil
	}
	fn, ok := vm.envs[key].RawGetString(name).(*lua.LFunction)
	if !ok {
		return nil
	}
	return &Func{vm: vm, key: key, id: id, fn: fn}
}

// Call 调用脚本函数（nx_execute 语义）；无此函数返回 (nil, false, nil)。
func (vm *VM) Call(script, name string, args ...Value) ([]Value, bool, error) {
	fn := vm.Function(script, name)
	if fn == nil {
		return nil, false, nil
	}
	out, err := fn.Call(args...)
	return out, true, err
}

// Call 执行函数（带预算）。普通协程取得（*VM.Function）的函数加锁调用；
// 回调语境取得（Reentrant.Function）的函数不加锁。
func (f *Func) Call(args ...Value) ([]Value, error) {
	if !f.reentrant {
		f.vm.lock()
		defer f.vm.unlock()
	}
	var out []Value
	err := f.callUnlocked(func(L *lua.LState) {
		for _, arg := range args {
			L.Push(toLua(L, arg))
		}
	}, len(args), func(L *lua.LState, v lua.LValue) {
		out = append(out, fromLua(L, v))
	})
	return out, err
}

// callUnlocked 压参 → 调用 → 逐返回值回调。调用方持锁；出错禁用所属脚本。
//
// 回调里再入的调用是嵌套的：本次的预算上下文用完须还原成进入前的那一个。
// 若直接摘掉，外层主循环（mainLoopWithContext）下一条指令就对着 nil 上下文
// 取 Done() 崩溃——被 PCall 收成错误后，**外层**脚本被误禁用。
func (f *Func) callUnlocked(push func(*lua.LState), nargs int, take func(*lua.LState, lua.LValue)) error {
	vm := f.vm
	if vm.failed[f.id] {
		return ErrScriptDisabled
	}
	outer := vm.l.Context()
	ctx, cancel := context.WithTimeout(context.Background(), vm.opt.CallTimeout)
	vm.l.SetContext(ctx)
	defer func() {
		if outer != nil {
			vm.l.SetContext(outer)
		} else {
			vm.l.RemoveContext()
		}
		cancel()
	}()
	top := vm.l.GetTop()
	vm.l.Push(f.fn)
	push(vm.l)
	if err := vm.l.PCall(nargs, lua.MultRet, nil); err != nil {
		vm.failed[f.id] = true
		vm.l.SetTop(top)
		vm.trace("script: %s 运行期出错，该脚本已禁用: %v", f.key, err)
		return fmt.Errorf("%w: %s: %v", ErrScriptDisabled, f.key, err)
	}
	for i := top + 1; i <= vm.l.GetTop(); i++ {
		take(vm.l, vm.l.Get(i))
	}
	vm.l.SetTop(top)
	return nil
}
