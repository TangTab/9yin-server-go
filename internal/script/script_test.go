package script

// 宿主与公式引擎测试：nx_* 面、脚本装载与钩子契约、预算与错误隔离、
// 公式转译装载与结算管线（数值对拍官服女号抓包的新号面板口径）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

// fakeObject 测试对象：一张属性表。
type fakeObject struct {
	kind  Kind
	props map[string]Value
}

func (o *fakeObject) Type() int      { return int(o.kind) }
func (o *fakeObject) Config() string { return "cfg_" + o.props["ConfigID"].Text }
func (o *fakeObject) Query(name string) (Value, bool) {
	v, ok := o.props[name]
	return v, ok
}
func (o *fakeObject) Set(name string, v Value) bool {
	o.props[name] = v
	return true
}

// fakeWorld 测试世界：句柄 → 对象表 + 战斗动作流水。
type fakeWorld struct {
	objects map[Handle]*fakeObject
	next    uint64
	scene   Handle
	damages []float64
	customs [][]Value
	created []string
	broken  []Handle
}

func (w *fakeWorld) add(kind Kind, props map[string]Value) Handle {
	w.next++
	if props == nil {
		props = map[string]Value{}
	}
	h := Handle{Kind: kind, ID: w.next}
	w.objects[h] = &fakeObject{kind: kind, props: props}
	return h
}

func (w *fakeWorld) Lookup(h Handle) (Object, bool) {
	o, ok := w.objects[h]
	return o, ok
}
func (w *fakeWorld) Scene() Handle { return w.scene }
func (w *fakeWorld) CreateFromConfig(configID string) (Handle, bool) {
	w.created = append(w.created, configID)
	return w.add(KindHelper, map[string]Value{"ConfigID": String(configID)}), true
}
func (w *fakeWorld) Destroy(h Handle) bool                        { delete(w.objects, h); return true }
func (w *fakeWorld) AddBuffer(target, caster, buffer Handle) bool { return true }
func (w *fakeWorld) RemoveBuffer(target, buffer Handle) bool      { return true }
func (w *fakeWorld) FindBuffer(target Handle, configID string) (Handle, bool) {
	return NullHandle, false
}
func (w *fakeWorld) Damage(attacker, defender Handle, damageType int, amount float64) float64 {
	w.damages = append(w.damages, amount)
	return amount
}
func (w *fakeWorld) Custom(self Handle, args ...Value)               { w.customs = append(w.customs, args) }
func (w *fakeWorld) SysInfo(self Handle, tipType int, args ...Value) {}
func (w *fakeWorld) Around(center Handle, radius float64, types, limit int) []Handle {
	return nil
}
func (w *fakeWorld) BreakFlow(h Handle) { w.broken = append(w.broken, h) }

// newTestVM 带脚本根的 VM（脚本按需落盘）。
func newTestVM(t *testing.T, world World, files map[string]string) *VM {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vm := New(world, Options{LuaDir: dir, CallTimeout: 200 * time.Millisecond})
	t.Cleanup(vm.Close)
	return vm
}

// 官方脚本范式（skill_sample.lua）：钩子契约 + nx_* 调用。
const skillSample = `
require("define")
local hits = 0
function skill_before_hit(self, skill)
	if nx_type(self) ~= TYPE_PLAYER then return 0 end
	return 1
end
function get_damage(self, skill, target)
	local power = nx_query_property(self, "MeleePower") + nx_query_property(self, "MeleePowerAdd")
	return power * 2, 0, 0, 0, 0
end
function skill_after_hit(self, skill, target)
	hits = hits + 1
	local buffer = nx_create_from_config(nx_get_scene(), "buf_test")
	if not nx_exists(buffer) then return end
	nx_set_property(buffer, "Level", nx_int(2))
	nx_add_buffer(target, self, buffer)
	nx_damage(self, target, nx_float(7))
	return hits
end
function skill_break(self, skill)
	nx_break_any_flow(self)
end
`

// nx_* 转换、属性查询、buff 创建与消息；钩子契约与官方 skill_sample 一致。
func TestSkillHooksFollowOfficialContract(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}, scene: Handle{Kind: KindScene, ID: 9}}
	player := world.add(KindPlayer, map[string]Value{"MeleePower": Int(3)})
	npc := world.add(KindNPC, nil)
	vm := newTestVM(t, world, map[string]string{"define.lua": "TYPE_PLAYER = 2\nTYPE_NPC = 4\n", "skill/sample.lua": skillSample})

	before, ok, err := vm.Call("skill/sample", "skill_before_hit", Ref(player), Nil())
	if err != nil || !ok || before[0].IntValue() != 1 {
		t.Fatalf("skill_before_hit: ok=%t err=%v out=%v", ok, err, before)
	}
	damage, _, err := vm.Call("skill/sample", "get_damage", Ref(player), Nil(), Ref(npc))
	if err != nil || len(damage) != 5 || damage[0].Number() != 6 {
		t.Fatalf("get_damage 应返回 5 系伤害且外功=6: %v %v", damage, err)
	}
	after, _, err := vm.Call("skill/sample", "skill_after_hit", Ref(player), Nil(), Ref(npc))
	if err != nil || after[0].IntValue() != 1 || len(world.created) != 1 || world.created[0] != "buf_test" {
		t.Fatalf("skill_after_hit: %v %v created=%v", after, err, world.created)
	}
	if len(world.damages) != 1 || world.damages[0] != 7 {
		t.Fatalf("nx_damage 未落: %v", world.damages)
	}
	// 无此函数 → 无钩子（ok=false），不报错。
	if _, ok, err := vm.Call("skill/sample", "no_such_hook"); err != nil || ok {
		t.Fatalf("缺席钩子应 ok=false: %t %v", ok, err)
	}
	// 未实现的 nx_* 走兜底空函数，脚本不中断。
	if _, _, err := vm.Call("skill/sample", "skill_break", Ref(player), Nil()); err != nil || len(world.broken) != 1 {
		t.Fatalf("skill_break/nx_break_any_flow: %v broken=%v", err, world.broken)
	}
}

// require 经脚本根装载（反斜杠模块名）；脚本缺席只报一次并禁用。
func TestRequireResolvesBackslashModules(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}}
	vm := newTestVM(t, world, map[string]string{
		"define.lua":             "ANSWER = 42",
		"share/skill_define.lua": `require("define"); SKILL_X = ANSWER + 1`,
		"rule/use.lua":           `require("share\\skill_define"); function value() return SKILL_X, nx_number(nx_widestr("7")) end`,
	})
	out, ok, err := vm.Call("rule/use", "value")
	if err != nil || !ok || out[0].IntValue() != 43 || out[1].IntValue() != 7 {
		t.Fatalf("require 链: ok=%t err=%v out=%v", ok, err, out)
	}
	fn := vm.Function("missing/script", "value")
	if fn != nil || vm.Has("missing/script") {
		t.Fatal("缺席脚本应无函数")
	}
}

// 钩子脚本各有一份函数环境（官方每份技能 / buff 脚本是独立上下文）：两份
// 脚本都定义 get_damage 时各调各的；require 进来的模块全局也各归各——官方
// AddBuff 模块同名不同实现，共用全局会互相覆盖。此前 Function 取的是全局
// 函数，技能会误调别的脚本的 get_damage。
func TestHookScriptsAreIsolated(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}}
	vm := newTestVM(t, world, map[string]string{
		"define.lua": "TYPE_PLAYER = 2",
		"buf/a.lua": `require("define"); function get_damage() return "buf-a" end` + "\n" +
			`function tp() return TYPE_PLAYER end`,
		"skill/b.lua": `require("skill\\AddBuff\\heal"); function get_damage() return AddBuff() .. "-b" end`,
		"skill/c.lua": `require("skill\\AddBuff\\revive"); function get_damage() return AddBuff() .. "-c" end` +
			"\n" + `function tp() return TYPE_PLAYER end`,
		"skill/addbuff/heal.lua":   `function AddBuff() return "heal" end`,
		"skill/addbuff/revive.lua": `function AddBuff() return "revive" end`,
	})
	for script, want := range map[string]string{"buf/a": "buf-a", "skill/b": "heal-b", "skill/c": "revive-c"} {
		out, err := vm.Function(script, "get_damage").Call()
		if err != nil || len(out) != 1 || out[0].Text != want {
			t.Errorf("%s.get_damage = %v, %v; want %q", script, out, err, want)
		}
	}
	// 没要求过的脚本读不到别人的模块全局；模块全局同样各归各：a 自己
	// require 过 define，读得到 TYPE_PLAYER；c 没 require，读不到（官方脚本
	// 各自 require("define") 正为此）。
	if fn := vm.Function("buf/a", "AddBuff"); fn != nil {
		t.Error("钩子脚本的函数应只认本脚本环境里定义的")
	}
	if out, err := vm.Function("buf/a", "tp").Call(); err != nil || out[0].IntValue() != 2 {
		t.Errorf("buf/a.tp = %v, %v; want 2", out, err)
	}
	if out, err := vm.Function("skill/c", "tp").Call(); err != nil || out[0].Kind != KindNil {
		t.Errorf("skill/c 没 require define，不应读得到 TYPE_PLAYER: %v, %v", out, err)
	}
}

// 预算：死循环脚本超时后只禁用该脚本，VM 与其余脚本照常。
func TestBudgetDisablesOnlyOffendingScript(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}}
	vm := newTestVM(t, world, map[string]string{
		"bad.lua":  "function spin() while true do end end",
		"good.lua": "function value() return 5 end",
	})
	started := time.Now()
	if _, err := vm.Function("bad", "spin").Call(); err == nil {
		t.Fatal("死循环应超时出错")
	}
	if time.Since(started) > 5*time.Second {
		t.Fatalf("预算未生效: %v", time.Since(started))
	}
	if fn := vm.Function("bad", "spin"); fn != nil {
		t.Fatalf("出错脚本应保持禁用（不再给出函数），得到 %v", fn)
	}
	out, err := vm.Function("good", "value").Call()
	if err != nil || out[0].IntValue() != 5 {
		t.Fatalf("其余脚本不受影响: %v %v", out, err)
	}
}

// 公式引擎：转译装载 + 结算管线；数值对拍官服女号抓包（新号面板
// MeleePower=3：普攻 2、锁定 8/暴击 16、16.99/2 → 8、33.17 → 33）。
func TestFormulaEngineMatchesOfficialNewbieDamage(t *testing.T) {
	vm := New(&fakeWorld{objects: map[Handle]*fakeObject{}}, Options{})
	t.Cleanup(vm.Close)
	specs := make([]FormulaSpec, 0, 5)
	for _, row := range [][2]string{{"372", "(P1+((P3+P4)*P2*0.1))/P5"}, {"237", "(P1+((P4+P5)*P2*0.3))/P3"},
		{"9", "R[P1,P2]"}, {"19", "MIN[5,40]*0.05+20"}, {"334", "6000/#CurTargetCount"}} {
		body, err := skilldata.Transpile(row[1])
		if err != nil {
			t.Fatalf("转译 %s: %v", row[0], err)
		}
		id, _ := strconv.Atoi(row[0])
		specs = append(specs, FormulaSpec{ID: id, Body: body})
	}
	// 公式 372（普攻）：P1/P2 字面量、P3/P4 攻方属性、P5 击数。
	specs[0].Params = [12]FormulaParam{{Source: 5}, {Source: 5}, {Source: 0, Prop: "MeleePower"}, {Source: 0, Prop: "MeleePowerAdd"}, {Source: 1, Prop: "HurtCount"}}
	// 公式 237（长拳三招共用）：P1/P2/P3 字面量、P4/P5 攻方属性。
	specs[1].Params = [12]FormulaParam{{Source: 5}, {Source: 5}, {Source: 5}, {Source: 0, Prop: "MeleePower"}, {Source: 0, Prop: "MeleePowerAdd"}}
	specs[2].Params = [12]FormulaParam{{Source: 5}, {Source: 5}}
	engine := NewEngine(vm, specs)
	if installed, skipped := engine.Installed(); installed != 5 || skipped != 0 {
		t.Fatalf("装载 %d 跳过 %d", installed, skipped)
	}
	attacker := map[string]float64{"MeleePower": 3, "MeleePowerAdd": 0}
	noLuck := []float64{0.99, 0.99, 0.5}
	cases := []struct {
		name string
		ctx  HitContext
		want float64
	}{
		{"普攻 2 击", HitContext{FormulaID: 372, Params: []string{"5", "1", ""}, HurtCount: 2, Attacker: attacker, Rand: noLuck}, 2},
		{"锁定分母 3", HitContext{FormulaID: 237, Params: []string{"25", "1", "3"}, Attacker: attacker, Rand: noLuck}, 8},
		{"锁定暴击 ×2", HitContext{FormulaID: 237, Params: []string{"25", "1", "3"}, Attacker: attacker, CritValue: 16, Rand: []float64{0.99, 0.0}}, 16},
		{"cqgf06 分母 2", HitContext{FormulaID: 237, Params: []string{"16", "1.1", "2"}, Attacker: attacker, Rand: noLuck}, 8},
		{"cqgf04 单击", HitContext{FormulaID: 237, Params: []string{"32", "1.3", "1"}, Attacker: attacker, Rand: noLuck}, 33},
		{"区间掷值", HitContext{FormulaID: 9, Params: []string{"10", "20"}, Attacker: attacker, Rand: []float64{0.99, 0.99, 0.5}}, 15},
		{"MIN 取小", HitContext{FormulaID: 19, Attacker: attacker, Rand: noLuck}, 20},
		{"#计数引用", HitContext{FormulaID: 334, TargetCount: 4, Attacker: attacker, Rand: noLuck}, 1500},
	}
	for _, c := range cases {
		got := engine.CalcHit(c.ctx)
		if got.Fallback || got.Miss || got.Damage != c.want {
			t.Errorf("%s: got %+v want damage=%.0f", c.name, got, c.want)
		}
	}
	// Settle：技能脚本 get_damage 给出原值，引擎只跑结算管线（取整 → 暴击
	// → 招架 → 保底）。
	settle := engine.Settle(HitContext{Attacker: attacker, Rand: noLuck}, 8.63)
	if settle.Fallback || settle.Damage != 8 {
		t.Errorf("Settle 取整: %+v", settle)
	}
	if settle := engine.Settle(HitContext{Attacker: attacker, CritValue: 16, Rand: []float64{0.99, 0.0}}, 8.63); settle.Fallback || !settle.Crit || settle.Damage != 16 {
		t.Errorf("Settle 暴击（整伤翻倍）: %+v", settle)
	}
	if settle := engine.Settle(HitContext{Attacker: attacker, Parrying: true, Rand: noLuck}, 9); settle.Fallback || !settle.Parried || settle.Damage != 1 {
		t.Errorf("Settle 招架: %+v", settle)
	}
	// 未命中：守方闪避远高于命中。
	miss := engine.CalcHit(HitContext{FormulaID: 237, Params: []string{"25", "1", "3"}, Attacker: attacker,
		Evasion: 1000, Rand: []float64{0.0, 0.99}})
	if !miss.Miss || miss.Damage != 0 {
		t.Errorf("未命中: %+v", miss)
	}
	// 招架：穿透比 0.1、保底 1。
	parried := engine.CalcHit(HitContext{FormulaID: 237, Params: []string{"25", "1", "3"}, Attacker: attacker, Parrying: true, Rand: noLuck})
	if !parried.Parried || parried.Damage != 1 {
		t.Errorf("招架: %+v", parried)
	}
	// 公式缺席 → 回退。
	if got := engine.CalcHit(HitContext{FormulaID: 999999}); !got.Fallback {
		t.Errorf("缺席公式应回退: %+v", got)
	}
}

// 回调再入：Go 回调（由 VM 持锁调用）里经 Reentrant 取函数并调用可以
// 完成；同一处若误用 *VM（加锁面）则自锁——用带超时的探针断言它不会返回。
func TestReentrantCallerInsideGoCallback(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}}
	vm := New(world, Options{CallTimeout: 200 * time.Millisecond})
	t.Cleanup(vm.Close)
	done := make(chan string, 1)
	vm.l.SetGlobal("go_callback", vm.l.NewFunction(func(L *lua.LState) int {
		caller := Reentrant{VM: vm}
		if out, err := caller.Function("rule/lib", "answer").Call(); err == nil {
			done <- fmt.Sprintf("reentrant:%d", out[0].IntValue())
		} else {
			done <- "reentrant-err:" + err.Error()
		}
		return 0
	}))
	if err := vm.RunFileSource("rule/lib", "function answer() return 42 end"); err != nil {
		t.Fatal(err)
	}
	if err := vm.RunFileSource("rule/use", "go_callback()"); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got != "reentrant:42" {
			t.Fatalf("回调再入结果 %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("回调内经 Reentrant 再入未返回")
	}
}

// 嵌套调用不破坏外层上下文：外层经 Call（带预算）进入，Go 回调里再入调用
// 返回后，外层脚本须继续执行且不被禁用。此前嵌套调用结束时无条件摘掉
// LState 的 context，外层主循环随即对着 nil 取 Done() 崩溃——线上「技能
// 钩子 → nx_add_buffer → buff 的 buffer_active 钩子」一走就废掉技能脚本。
func TestNestedCallKeepsOuterContext(t *testing.T) {
	world := &fakeWorld{objects: map[Handle]*fakeObject{}}
	vm := New(world, Options{CallTimeout: time.Second})
	t.Cleanup(vm.Close)
	vm.l.SetGlobal("go_callback", vm.l.NewFunction(func(L *lua.LState) int {
		out, err := (Reentrant{VM: vm}).Function("rule/lib", "answer").Call()
		if err != nil {
			L.RaiseError("%s", err.Error())
		}
		// 回调返回后外层还有指令要跑——崩溃就发生在那第一条指令上。
		L.Push(lua.LNumber(out[0].IntValue() * 2))
		return 1
	}))
	for key, source := range map[string]string{
		"rule/lib": "function answer() return 21 end",
		"rule/use": "function run() local v = go_callback() return v + 1, 'done' end",
	} {
		if err := vm.RunFileSource(key, source); err != nil {
			t.Fatal(err)
		}
	}
	for n := 1; n <= 2; n++ { // 第二次照常：脚本没被禁用
		out, ok, err := vm.Call("rule/use", "run")
		if !ok || err != nil || len(out) != 2 || out[0].IntValue() != 43 || out[1].Text != "done" {
			t.Fatalf("第 %d 次外层调用 = %+v, %t, %v; want [43 done]", n, out, ok, err)
		}
	}
}
