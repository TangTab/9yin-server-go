package script

// api.go nx_* API 面：官方脚本（testdata/lua，Res/lua 同构）调用的宿主函数。
//
// 对象一律是 Handle 句柄（userdata），属性/记录/战斗动作经 World 接口交
// 宿主实现（internal/fssoloworld 的战斗适配层）。函数集按官方语料的使用
// 频度落地战斗相关子集：
//
//	转换   nx_int/nx_number/nx_float/nx_double/nx_string/nx_widestr/nx_boolean/
//	       nx_int64/nx_null/nx_null_obj/nx_object
//	对象   nx_exists/nx_type/nx_get_scene/nx_get_config/nx_get_script/
//	       nx_get_position/nx_get_child/nx_get_child_count
//	属性   nx_query_property/nx_find_property/nx_set_property/nx_inc_property
//	记录   nx_add_record_row_values/nx_query_record_value/nx_get_record_rows
//	脚本   nx_execute（require 另由装载器覆盖）
//	消息   nx_custom/nx_sysinfo
//	buff   nx_create_from_config/nx_destroy_self/nx_add_buffer/nx_remove_buffer/
//	       nx_find_buffer
//	战斗   nx_damage/nx_damage_target/nx_get_around_list/nx_distance/
//	       nx_break_any_flow/nx_use_skill_to_target/nx_query_skill_int/
//	       nx_query_skill_string/nx_find_skill/nx_get_current_neigong
//
// 未落地的 nx_* 名字由全局 __index 兜底：返回空函数并 trace 一次——官方
// 脚本不因个别 API 缺席而中断（缺失面留痕，按需补）。

import (
	"bytes"
	"math"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// World 宿主世界：nx_* 的对象面由宿主实现。
type World interface {
	// Lookup 句柄 → 活对象视图（false = 已失效，nx_exists 为假）。
	Lookup(h Handle) (Object, bool)
	// Scene 当前场景句柄。
	Scene() Handle
	// CreateFromConfig 按 ConfigID 建辅助对象（buff/技能）。
	CreateFromConfig(configID string) (Handle, bool)
	// Destroy 销毁对象（nx_destroy_self）。
	Destroy(h Handle) bool
	// AddBuffer 把 buff 挂到目标身上（nx_add_buffer）。
	AddBuffer(target, caster, buffer Handle) bool
	// RemoveBuffer 摘除 buff。
	RemoveBuffer(target, buffer Handle) bool
	// FindBuffer 按 ConfigID 找目标身上的 buff。
	FindBuffer(target Handle, configID string) (Handle, bool)
	// Damage 结算一次伤害（nx_damage；damageType = DAMAGE_TYPE_*），返回实伤。
	Damage(attacker, defender Handle, damageType int, amount float64) float64
	// Custom 自定义消息下行（nx_custom；args[0] = 消息号）。
	Custom(self Handle, args ...Value)
	// SysInfo 提示信息（nx_sysinfo；tipType = TIPSTYPE_*）。
	SysInfo(self Handle, tipType int, args ...Value)
	// Around 周围对象（nx_get_around_list：半径、TYPE_* 位掩码、上限）。
	Around(center Handle, radius float64, types, limit int) []Handle
}

// Object 活对象视图：属性操作面。
type Object interface {
	// Type 对象类别（TYPE_* 位值）。
	Type() int
	// Config ConfigID（nx_get_config）。
	Config() string
	// Query 属性（nx_query_property / nx_find_property）。
	Query(name string) (Value, bool)
	// Set 属性（nx_set_property）。
	Set(name string, v Value) bool
}

// ---- 可选能力面（宿主按对象类别实现，类型断言取用） ----

// Positioner 有位置的对象。
type Positioner interface{ Position() ([4]float64, bool) }

// Recorder 有记录表的对象（buff 的 DirectPropertyRec 等、玩家的 Task_Record 等）。
type Recorder interface {
	// AddRecordRow 追加一行（值按列序），返回行号（-1 = 失败）。
	AddRecordRow(record string, values []Value) int
	// QueryRecord 取一格。
	QueryRecord(record string, row, col int) (Value, bool)
	// RecordRows 行数。
	RecordRows(record string) int
}

// Container 有子对象的对象（背包/技能容器）。
type Container interface {
	Child(name string) (Handle, bool)
	ChildCount() int
}

// Scripted 有逻辑类脚本名的对象（nx_get_script，如 "BossNpc"）。
type Scripted interface{ Script() string }

// SkillStatter 技能/buff 对象的静态包查询（nx_query_skill_int/string：
// 列名形如 Skill_EffectType，去前缀即属性名）。
type SkillStatter interface {
	SkillStatic(staticData int, column string) (Value, bool)
}

// ---- World 可选能力面 ----

// IDResolver 数字对象号 → 句柄（nx_object(list[i])）。
type IDResolver interface {
	ByID(kind Kind, id uint64) (Handle, bool)
}

// BufferDamager 经 buff 结算伤害（nx_damage_target：buff 的 get_damage）。
type BufferDamager interface {
	DamageByBuffer(attacker, target, buffer Handle) float64
}

// SkillFinder 技能查询。
type SkillFinder interface {
	FindSkill(owner Handle, configID string) (Handle, bool)
	AddSkill(owner, skill Handle) bool
}

// NeigongFinder 内功查询。
type NeigongFinder interface {
	CurrentNeigong(owner Handle) (Handle, bool)
	FindNeigong(owner Handle, configID string) (Handle, bool)
}

// FlowBreaker 打断当前流程（nx_break_any_flow / nx_stop）。
type FlowBreaker interface{ BreakFlow(h Handle) }

// SkillCaster 让对象对目标施放技能（nx_use_skill_to_target）。
type SkillCaster interface {
	UseSkillToTarget(self Handle, skill Handle, target Handle) bool
}

// CountBeater 定时器（nx_add_countbeat：间隔毫秒、次数；到点回调脚本函数）。
type CountBeater interface {
	AddCountBeat(owner Handle, callback string, intervalMS, count int) bool
}

// register 登记 nx_* 面与装载器。
func (vm *VM) register() {
	L := vm.l
	mt := L.NewTable()
	mt.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		if h, ok := toHandle(L, 1); ok {
			L.Push(lua.LString(h.String()))
		} else {
			L.Push(lua.LString("null"))
		}
		return 1
	}))
	L.SetField(L.Get(lua.RegistryIndex), objectMeta, mt)

	// require/loadfile/dofile 经资源仓读脚本根。
	L.SetGlobal("require", L.NewFunction(vm.fnRequire))
	L.SetGlobal("loadfile", L.NewFunction(vm.fnLoadfile))
	L.SetGlobal("dofile", L.NewFunction(vm.fnDofile))

	for name, fn := range map[string]lua.LGFunction{
		"nx_int": vm.fnInt, "nx_number": vm.fnNumber, "nx_float": vm.fnNumber,
		"nx_double": vm.fnNumber, "nx_int64": vm.fnInt, "nx_boolean": vm.fnBoolean,
		"nx_string": vm.fnString, "nx_widestr": vm.fnString,
		"nx_null": vm.fnNull, "nx_null_obj": vm.fnNull, "nx_object": vm.fnObject,
		"nx_exists": vm.fnExists, "nx_type": vm.fnType, "nx_get_scene": vm.fnGetScene,
		"nx_query_property": vm.fnQueryProperty, "nx_find_property": vm.fnFindProperty,
		"nx_set_property": vm.fnSetProperty, "nx_inc_property": vm.fnIncProperty,
		"nx_get_config": vm.fnGetConfig, "nx_get_script": vm.fnGetScript,
		"nx_get_position": vm.fnGetPosition, "nx_get_child": vm.fnGetChild,
		"nx_get_child_count":       vm.fnGetChildCount,
		"nx_add_record_row_values": vm.fnAddRecordRow, "nx_query_record_value": vm.fnQueryRecord,
		"nx_get_record_rows": vm.fnRecordRows,
		"nx_execute":         vm.fnExecute, "nx_custom": vm.fnCustom, "nx_sysinfo": vm.fnSysinfo,
		"nx_create_from_config": vm.fnCreateFromConfig, "nx_destroy_self": vm.fnDestroy,
		"nx_add_buffer": vm.fnAddBuffer, "nx_remove_buffer": vm.fnRemoveBuffer,
		"nx_find_buffer": vm.fnFindBuffer,
		"nx_damage":      vm.fnDamage, "nx_damage_target": vm.fnDamageTarget,
		"nx_get_around_list": vm.fnAround, "nx_distance": vm.fnDistance, "nx_distance3d": vm.fnDistance,
		"nx_break_any_flow": vm.fnBreakFlow, "nx_stop": vm.fnBreakFlow,
		"nx_use_skill_to_target": vm.fnUseSkill, "nx_query_skill_int": vm.fnSkillStatic,
		"nx_query_skill_string": vm.fnSkillStatic,
		"nx_find_skill":         vm.fnFindSkill, "nx_add_skill": vm.fnAddSkill,
		"nx_get_current_neigong": vm.fnCurrentNeigong, "nx_find_neigong": vm.fnFindNeigong,
		"nx_add_countbeat": vm.fnAddCountBeat,
	} {
		L.SetGlobal(name, L.NewFunction(fn))
	}

	// 未落地的 nx_* 名字：空函数兜底 + 一次 trace。
	L.SetGlobal("__nx_unsupported", L.NewFunction(func(L *lua.LState) int {
		name := string(L.CheckString(1))
		if !vm.stubbed[name] {
			vm.stubbed[name] = true
			vm.trace("script: nx_%s 未实现（调用按空处理）", name)
		}
		return 0
	}))
	if err := L.DoString(`setmetatable(_G, { __index = function(_, k)
		if type(k) == "string" and k:sub(1, 3) == "nx_" then
			__nx_unsupported(k)
			local stub = function() return end
			rawset(_G, k, stub)
			return stub
		end
		return nil
	end })`); err != nil {
		vm.trace("script: 全局兜底安装失败: %v", err)
	}
}

// ---- 小件 ----

func toHandle(L *lua.LState, idx int) (Handle, bool) {
	ud, ok := L.Get(idx).(*lua.LUserData)
	if !ok {
		return NullHandle, false
	}
	h, ok := ud.Value.(Handle)
	return h, ok && h.Valid()
}

// argValues 自 start 起的全部参数。
func argValues(L *lua.LState, start int) []Value {
	var out []Value
	for i := start; i <= L.GetTop(); i++ {
		out = append(out, fromLua(L, L.Get(i)))
	}
	return out
}

func (vm *VM) object(L *lua.LState, idx int) Object {
	h, ok := toHandle(L, idx)
	if !ok {
		return nil
	}
	obj, _ := vm.world.Lookup(h)
	return obj
}

// ---- 转换 ----

func (vm *VM) fnInt(L *lua.LState) int {
	L.Push(lua.LNumber(int64(argValue(L, 1).Number())))
	return 1
}

func (vm *VM) fnNumber(L *lua.LState) int {
	L.Push(lua.LNumber(argValue(L, 1).Number()))
	return 1
}

func (vm *VM) fnBoolean(L *lua.LState) int {
	L.Push(lua.LBool(argValue(L, 1).Truthy()))
	return 1
}

func (vm *VM) fnString(L *lua.LState) int {
	v := argValue(L, 1)
	switch v.Kind {
	case KindString, KindWide:
		L.Push(lua.LString(v.Text))
	case KindNil:
		L.Push(lua.LString(""))
	case KindObject:
		L.Push(lua.LString(v.Obj.String()))
	default:
		L.Push(lua.LNumber(v.Number()))
	}
	return 1
}

func (vm *VM) fnNull(L *lua.LState) int {
	L.Push(toLua(L, Ref(NullHandle)))
	return 1
}

func (vm *VM) fnObject(L *lua.LState) int {
	v := argValue(L, 1)
	if v.Kind == KindObject {
		L.Push(toLua(L, v))
		return 1
	}
	if resolver, ok := vm.world.(IDResolver); ok {
		for _, kind := range []Kind{KindNPC, KindPlayer, KindItem} {
			if h, ok := resolver.ByID(kind, uint64(v.Number())); ok {
				L.Push(toLua(L, Ref(h)))
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

// ---- 对象 ----

func (vm *VM) fnExists(L *lua.LState) int {
	h, ok := toHandle(L, 1)
	if ok {
		_, ok = vm.world.Lookup(h)
	}
	L.Push(lua.LBool(ok))
	return 1
}

func (vm *VM) fnType(L *lua.LState) int {
	if obj := vm.object(L, 1); obj != nil {
		L.Push(lua.LNumber(obj.Type()))
	} else {
		L.Push(lua.LNumber(0))
	}
	return 1
}

func (vm *VM) fnGetScene(L *lua.LState) int {
	if h := vm.world.Scene(); h.Valid() {
		L.Push(toLua(L, Ref(h)))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

func (vm *VM) fnQueryProperty(L *lua.LState) int {
	v := Value{Kind: KindInt}
	if obj := vm.object(L, 1); obj != nil {
		if got, ok := obj.Query(L.CheckString(2)); ok {
			v = got
		}
	}
	L.Push(toLua(L, v))
	return 1
}

func (vm *VM) fnFindProperty(L *lua.LState) int {
	found := false
	if obj := vm.object(L, 1); obj != nil {
		_, found = obj.Query(L.CheckString(2))
	}
	L.Push(lua.LBool(found))
	return 1
}

func (vm *VM) fnSetProperty(L *lua.LState) int {
	done := false
	if obj := vm.object(L, 1); obj != nil {
		done = obj.Set(L.CheckString(2), argValue(L, 3))
	}
	L.Push(lua.LBool(done))
	return 1
}

func (vm *VM) fnIncProperty(L *lua.LState) int {
	v := argValue(L, 3)
	if obj := vm.object(L, 1); obj != nil {
		name := L.CheckString(2)
		old, _ := obj.Query(name)
		if v.Kind == KindString || v.Kind == KindWide {
			v = String(old.Text + v.Text)
		} else {
			v = Float(old.Number() + v.Number())
		}
		obj.Set(name, v)
	}
	L.Push(toLua(L, v))
	return 1
}

func (vm *VM) fnGetConfig(L *lua.LState) int {
	L.Push(lua.LString(""))
	if obj := vm.object(L, 1); obj != nil {
		L.Replace(2, lua.LString(obj.Config()))
	}
	return 1
}

func (vm *VM) fnGetScript(L *lua.LState) int {
	L.Push(lua.LString(""))
	if obj := vm.object(L, 1); obj != nil {
		if scripted, ok := obj.(Scripted); ok {
			L.Replace(2, lua.LString(scripted.Script()))
		}
	}
	return 1
}

func (vm *VM) fnGetPosition(L *lua.LState) int {
	var pos [4]float64
	if obj := vm.object(L, 1); obj != nil {
		if positioned, ok := obj.(Positioner); ok {
			pos, _ = positioned.Position()
		}
	}
	for _, v := range pos {
		L.Push(lua.LNumber(v))
	}
	return 4
}

func (vm *VM) fnGetChild(L *lua.LState) int {
	if obj := vm.object(L, 1); obj != nil {
		if container, ok := obj.(Container); ok {
			if h, ok := container.Child(L.CheckString(2)); ok {
				L.Push(toLua(L, Ref(h)))
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (vm *VM) fnGetChildCount(L *lua.LState) int {
	count := 0
	if obj := vm.object(L, 1); obj != nil {
		if container, ok := obj.(Container); ok {
			count = container.ChildCount()
		}
	}
	L.Push(lua.LNumber(count))
	return 1
}

// ---- 记录 ----

func (vm *VM) fnAddRecordRow(L *lua.LState) int {
	row := -1
	if obj := vm.object(L, 1); obj != nil {
		if recorder, ok := obj.(Recorder); ok {
			row = recorder.AddRecordRow(L.CheckString(2), argValues(L, 3))
		}
	}
	L.Push(lua.LNumber(row))
	return 1
}

func (vm *VM) fnQueryRecord(L *lua.LState) int {
	v := Value{Kind: KindInt}
	if obj := vm.object(L, 1); obj != nil {
		if recorder, ok := obj.(Recorder); ok {
			if got, found := recorder.QueryRecord(L.CheckString(2), int(argValue(L, 3).Number()), int(argValue(L, 4).Number())); found {
				v = got
			}
		}
	}
	L.Push(toLua(L, v))
	return 1
}

func (vm *VM) fnRecordRows(L *lua.LState) int {
	count := 0
	if obj := vm.object(L, 1); obj != nil {
		if recorder, ok := obj.(Recorder); ok {
			count = recorder.RecordRows(L.CheckString(2))
		}
	}
	L.Push(lua.LNumber(count))
	return 1
}

// ---- 脚本 / 消息 ----

func (vm *VM) fnExecute(L *lua.LState) int {
	// Lua 回调内再入：外层调用已持锁，走不加锁变体。
	fn := vm.functionLocked(L.CheckString(1), L.CheckString(2))
	if fn == nil {
		L.RaiseError("nx_execute: 脚本函数缺席 %s", L.CheckString(1)+"."+L.CheckString(2))
	}
	args := argValues(L, 3)
	var out []Value
	if err := fn.callUnlocked(func(L *lua.LState) {
		for _, arg := range args {
			L.Push(toLua(L, arg))
		}
	}, len(args), func(L *lua.LState, v lua.LValue) {
		out = append(out, fromLua(L, v))
	}); err != nil {
		L.RaiseError("%v", err)
	}
	// 官方 nx_execute 返回结果表；单返回值亦包成表（added_value[1]）。
	table := L.NewTable()
	for i, v := range out {
		table.RawSetInt(i+1, toLua(L, v))
	}
	L.Push(table)
	return 1
}

// fnRequire require("模块")（全局语境）。
func (vm *VM) fnRequire(L *lua.LState) int { return vm.fnRequireIn(L, globalEnv) }

// fnRequireIn require 执行到钩子脚本 owner 的环境；返回 true。
func (vm *VM) fnRequireIn(L *lua.LState, owner string) int {
	key := scriptPath(L.CheckString(1))
	if err := vm.runInto(owner, key); err != nil {
		L.RaiseError("require %s: %v", key, err)
	}
	L.Push(lua.LBool(true))
	return 1
}

func (vm *VM) fnLoadfile(L *lua.LState) int { return vm.loadfile(L, nil) }

// fnLoadfileIn 钩子脚本 owner 环境里的 loadfile：编出的块环境 = owner 的函数环境。
func (vm *VM) fnLoadfileIn(L *lua.LState, owner string) int {
	return vm.loadfile(L, vm.hookEnv(owner))
}

// loadfile 编译一份源为块（失败返回 nil 与错误信；块的环境 = env，nil 即全局）。
func (vm *VM) loadfile(L *lua.LState, env *lua.LTable) int {
	path := L.CheckString(1)
	raw, err := vm.readScript(scriptPath(path))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	fn, err := vm.l.Load(bytes.NewReader(raw), path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	if env != nil {
		vm.l.SetFEnv(fn, env)
	}
	L.Push(fn)
	return 1
}

func (vm *VM) fnDofile(L *lua.LState) int { return vm.fnDofileIn(L, globalEnv) }

// fnDofileIn dofile 执行到钩子脚本 owner 的环境。
func (vm *VM) fnDofileIn(L *lua.LState, owner string) int {
	if err := vm.runInto(owner, scriptPath(L.CheckString(1))); err != nil {
		L.RaiseError("%v", err)
	}
	return 0
}

func (vm *VM) fnCustom(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		vm.world.Custom(h, argValues(L, 2)...)
	}
	return 0
}

func (vm *VM) fnSysinfo(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		vm.world.SysInfo(h, int(argValue(L, 2).Number()), argValues(L, 3)...)
	}
	return 0
}

// ---- buff ----

func (vm *VM) fnCreateFromConfig(L *lua.LState) int {
	// nx_create_from_config(scene|container, configID)
	configID := L.CheckString(2)
	if h, ok := vm.world.CreateFromConfig(configID); ok {
		L.Push(toLua(L, Ref(h)))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func (vm *VM) fnDestroy(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		L.Push(lua.LBool(vm.world.Destroy(h)))
		return 1
	}
	L.Push(lua.LBool(false))
	return 1
}

func (vm *VM) fnAddBuffer(L *lua.LState) int {
	target, caster, buffer := argValue(L, 1), argValue(L, 2), argValue(L, 3)
	ok := target.Kind == KindObject && buffer.Kind == KindObject &&
		vm.world.AddBuffer(target.Obj, caster.Obj, buffer.Obj)
	L.Push(lua.LBool(ok))
	return 1
}

func (vm *VM) fnRemoveBuffer(L *lua.LState) int {
	target, buffer := argValue(L, 1), argValue(L, 2)
	ok := target.Kind == KindObject && buffer.Kind == KindObject &&
		vm.world.RemoveBuffer(target.Obj, buffer.Obj)
	L.Push(lua.LBool(ok))
	return 1
}

func (vm *VM) fnFindBuffer(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		if buffer, ok := vm.world.FindBuffer(h, L.CheckString(2)); ok {
			L.Push(toLua(L, Ref(buffer)))
			return 1
		}
	}
	L.Push(lua.LNil)
	return 1
}

// ---- 战斗 ----

func (vm *VM) fnDamage(L *lua.LState) int {
	// nx_damage(attacker, defender, amount[, damageType])
	attacker, defender, amount := argValue(L, 1), argValue(L, 2), argValue(L, 3)
	damageType := int(argValue(L, 4).Number())
	if attacker.Kind != KindObject || defender.Kind != KindObject {
		L.Push(lua.LNumber(0))
		return 1
	}
	L.Push(lua.LNumber(vm.world.Damage(attacker.Obj, defender.Obj, damageType, amount.Number())))
	return 1
}

func (vm *VM) fnDamageTarget(L *lua.LState) int {
	attacker, target, buffer := argValue(L, 1), argValue(L, 2), argValue(L, 3)
	dealt := 0.0
	if attacker.Kind == KindObject && target.Kind == KindObject && buffer.Kind == KindObject {
		if damager, ok := vm.world.(BufferDamager); ok {
			dealt = damager.DamageByBuffer(attacker.Obj, target.Obj, buffer.Obj)
		}
	}
	L.Push(lua.LNumber(dealt))
	return 1
}

func (vm *VM) fnAround(L *lua.LState) int {
	table := L.NewTable()
	if h, ok := toHandle(L, 1); ok {
		list := vm.world.Around(h, argValue(L, 2).Number(), int(argValue(L, 3).Number()), int(argValue(L, 4).Number()))
		for i, found := range list {
			table.RawSetInt(i+1, toLua(L, Ref(found)))
		}
	}
	L.Push(table)
	return 1
}

func (vm *VM) fnDistance(L *lua.LState) int {
	distance := math.MaxFloat64
	a, b := vm.object(L, 1), vm.object(L, 2)
	if a != nil && b != nil {
		if pa, ok := a.(Positioner); ok {
			if pb, ok := b.(Positioner); ok {
				x, _ := pa.Position()
				y, _ := pb.Position()
				distance = math.Sqrt((x[0]-y[0])*(x[0]-y[0]) + (x[1]-y[1])*(x[1]-y[1]) + (x[2]-y[2])*(x[2]-y[2]))
			}
		}
	}
	L.Push(lua.LNumber(distance))
	return 1
}

func (vm *VM) fnBreakFlow(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		if breaker, ok := vm.world.(FlowBreaker); ok {
			breaker.BreakFlow(h)
		}
	}
	return 0
}

func (vm *VM) fnUseSkill(L *lua.LState) int {
	self, skill, target := argValue(L, 1), argValue(L, 2), argValue(L, 3)
	ok := false
	if self.Kind == KindObject && skill.Kind == KindObject {
		if caster, has := vm.world.(SkillCaster); has {
			ok = caster.UseSkillToTarget(self.Obj, skill.Obj, target.Obj)
		}
	}
	L.Push(lua.LBool(ok))
	return 1
}

func (vm *VM) fnSkillStatic(L *lua.LState) int {
	v := Value{Kind: KindInt}
	if obj := vm.object(L, 1); obj != nil {
		if statter, ok := obj.(SkillStatter); ok {
			column := strings.TrimPrefix(L.CheckString(3), "Skill_")
			if got, found := statter.SkillStatic(int(argValue(L, 2).Number()), column); found {
				v = got
			}
		}
	}
	L.Push(toLua(L, v))
	return 1
}

func (vm *VM) fnFindSkill(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		if finder, has := vm.world.(SkillFinder); has {
			if skill, found := finder.FindSkill(h, L.CheckString(2)); found {
				L.Push(toLua(L, Ref(skill)))
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (vm *VM) fnAddSkill(L *lua.LState) int {
	owner, skill := argValue(L, 1), argValue(L, 2)
	ok := false
	if owner.Kind == KindObject && skill.Kind == KindObject {
		if finder, has := vm.world.(SkillFinder); has {
			ok = finder.AddSkill(owner.Obj, skill.Obj)
		}
	}
	L.Push(lua.LBool(ok))
	return 1
}

func (vm *VM) fnCurrentNeigong(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		if finder, has := vm.world.(NeigongFinder); has {
			if book, found := finder.CurrentNeigong(h); found {
				L.Push(toLua(L, Ref(book)))
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (vm *VM) fnFindNeigong(L *lua.LState) int {
	if h, ok := toHandle(L, 1); ok {
		if finder, has := vm.world.(NeigongFinder); has {
			if book, found := finder.FindNeigong(h, L.CheckString(2)); found {
				L.Push(toLua(L, Ref(book)))
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (vm *VM) fnAddCountBeat(L *lua.LState) int {
	ok := false
	if h, okHandle := toHandle(L, 1); okHandle {
		if beater, has := vm.world.(CountBeater); has {
			ok = beater.AddCountBeat(h, L.CheckString(2), int(argValue(L, 3).Number()), int(argValue(L, 4).Number()))
		}
	}
	L.Push(lua.LBool(ok))
	return 1
}
