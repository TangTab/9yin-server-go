package fxcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/fsloader"
)

// officialAPINames 官方 FxCore.dll nx_* 全量名单（字符串表提取，82 项，
// 含 nx_main/nx_main_config 两个全局名）。对拍基准。
var officialAPINames = []string{
	"nx_assert", "nx_bind_logic", "nx_bind_script", "nx_boolean", "nx_call",
	"nx_callback", "nx_create", "nx_current", "nx_custom", "nx_custom_list",
	"nx_debug_log", "nx_decimals", "nx_destroy", "nx_double", "nx_dump_file",
	"nx_elapse", "nx_execute", "nx_file_delete", "nx_file_exists",
	"nx_find_custom", "nx_find_kind", "nx_find_method", "nx_find_property",
	"nx_find_script", "nx_find_scriptl", "nx_find_value", "nx_float",
	"nx_function", "nx_gen_event", "nx_id_equal", "nx_int", "nx_int64",
	"nx_is_debug", "nx_is_kind", "nx_is_null", "nx_is_valid", "nx_kill",
	"nx_log", "nx_logic_name", "nx_lookup", "nx_main", "nx_main_config",
	"nx_method", "nx_method_list", "nx_msgbox", "nx_name", "nx_null",
	"nx_number", "nx_object", "nx_path_create", "nx_path_delete",
	"nx_path_exists", "nx_pause", "nx_platform", "nx_property",
	"nx_property_list", "nx_quit", "nx_remove_value", "nx_resource_path",
	"nx_running", "nx_script_name", "nx_script_path", "nx_set_custom",
	"nx_set_property", "nx_set_single", "nx_set_value", "nx_single",
	"nx_space", "nx_strict_pause", "nx_string", "nx_time_begin",
	"nx_time_elapse", "nx_total_seconds", "nx_type", "nx_value",
	"nx_value_list", "nx_wait_event", "nx_widestr", "nx_work_path",
	"nx_ws_equal", "nx_ws_length", "nx_ws_lower", "nx_ws_upper",
}

func TestAPITableMatchesOfficial(t *testing.T) {
	table := APINames()
	set := map[string]bool{}
	for _, n := range table {
		if set[n] {
			t.Errorf("重复注册: %s", n)
		}
		set[n] = true
	}
	for _, n := range officialAPINames {
		if !set[n] {
			t.Errorf("官方函数缺失: %s", n)
		}
		delete(set, n)
	}
	for n := range set {
		t.Errorf("非官方函数混入: %s", n)
	}
	// Lua 依赖组声明存在。
	lua := 0
	for _, d := range APITable {
		if d.Status == "lua" {
			lua++
		}
	}
	if lua == 0 {
		t.Error("Lua 依赖组不应为空")
	}
}

//---- 实体面 ----

func testStore() *EntityStore {
	s := NewEntityStore()
	base := &EntInfo{TypeName: "ArrayList", Space: "FsKit",
		New: func(args *fskit.VarList) any { return map[string]any{} }}
	child := &EntInfo{TypeName: "IniDocument", ParentName: "ArrayList", Space: "FsKit",
		New: func(args *fskit.VarList) any { return map[string]any{} }}
	s.AddCreator(base)
	s.AddCreator(child)
	return s
}

func TestEntityLifecycle(t *testing.T) {
	s := testStore()
	// 重名注册拒收（官方 "entity name repeat"）。
	if err := s.AddCreator(&EntInfo{TypeName: "ArrayList",
		New: func(*fskit.VarList) any { return nil }}); err == nil ||
		!strings.Contains(err.Error(), "entity name repeat") {
		t.Fatalf("重名注册应拒收: %v", err)
	}
	e, err := s.Create("arraylist", "MainCon", nil) // 大小写不敏感
	if err != nil {
		t.Fatal(err)
	}
	if s.Lookup("maincon") != e {
		t.Error("Lookup 应大小写不敏感命中")
	}
	if s.Count() != 1 {
		t.Errorf("实体数=%d 期望 1", s.Count())
	}
	// 重名实体拒收。
	if _, err := s.Create("ArrayList", "MainCon", nil); err == nil {
		t.Error("重名实体应拒收")
	}
	// 前缀查找。
	var names []string
	if n := s.LookupMore("main", &names); n != 1 || names[0] != "MainCon" {
		t.Errorf("LookupMore=%d %v", n, names)
	}
	// 继承链。
	if !e.Info.IsKindOf("ArrayList") || !e.Info.IsKindOf("IniDocument") == false {
		t.Error("IsKindOf 自身应命中")
	}
	child, _ := s.Create("IniDocument", "doc1", nil)
	if !child.Info.IsKindOf("ArrayList") {
		t.Error("子类型应 IsKindOf 父类型")
	}
	// 单例。
	s.SetSingle(e)
	if s.GetSingle("ArrayList") != e {
		t.Error("GetSingle 应命中")
	}
	// 删除：标记 → 帧尾销毁（官方 SetDeleted/Destroy 序）。
	s.SetDeleted(e)
	if s.Lookup("MainCon") != nil {
		t.Error("删除后 Lookup 不应命中")
	}
	if s.GetByID(e.ID) != nil {
		t.Error("删除后 GetByID 不应命中")
	}
	if n := s.FlushDeleted(); n != 1 {
		t.Errorf("帧尾销毁数=%d 期望 1", n)
	}
	if s.GetSingle("ArrayList") != nil {
		t.Error("销毁应清单例位")
	}
	// 代际安全：悬挂句柄 nil。
	if s.GetByID(e.ID) != nil {
		t.Error("悬挂句柄应 nil")
	}
}

func TestFrameExecuteOrderAndRemove(t *testing.T) {
	s := testStore()
	var order []string
	s.AddFrameExecute("a", func(*Core, float64) { order = append(order, "a") })
	s.AddFrameExecute("b", func(*Core, float64) { order = append(order, "b") })
	s.AddFrameExecute("c", func(*Core, float64) { order = append(order, "c") })
	s.RemoveFrameExecute("b")
	s.executeFrames(nil, 0.016)
	if len(order) != 2 || order[0] != "a" || order[1] != "c" {
		t.Errorf("帧执行序=%v 期望 [a c]", order)
	}
}

//---- 事件/异步 ----

func TestAsyncEventWake(t *testing.T) {
	m := NewAsyncManager()
	var got *Event
	waiter := m.Add("script:waiter", func(p *AsyncProc, ev *Event) bool {
		if ev == nil {
			p.AddWaitEvent(EntityID(42)) // 挂起等待 42 号实体事件
			return false
		}
		got = ev // 唤醒携带事件
		p.done = true
		return false
	})
	m.ExecutePass1() // 运行 → 挂起
	if waiter.Done() {
		t.Fatal("等待者不应结束")
	}
	m.GenEvent(&Event{Object: EntityID(7), Name: "miss"}) // 不匹配目标
	m.ExecutePass1()
	if got != nil {
		t.Fatal("不匹配事件不应唤醒")
	}
	m.GenEvent(&Event{Object: EntityID(42), Name: "hit", Args: []any{"x"}})
	m.ExecutePass1()
	if got == nil || got.Name != "hit" {
		t.Fatal("匹配事件应唤醒等待者")
	}
	if n := m.ExecutePass2(); n != 1 {
		t.Errorf("终止过程清理数=%d 期望 1", n)
	}
	if m.Count() != 0 {
		t.Error("清理后过程表应空")
	}
}

func TestAsyncKillAndPause(t *testing.T) {
	m := NewAsyncManager()
	p := m.Add("loop", func(*AsyncProc, *Event) bool { return true })
	m.Kill(p.ID)
	m.ExecutePass1()
	if m.ExecutePass2() != 1 {
		t.Error("Kill 后应清除")
	}
	p2 := m.Add("paused", func(*AsyncProc, *Event) bool { return true })
	p2.Pause(true) // 严格暂停：事件不唤醒
	called := false
	p2.Body = func(*AsyncProc, *Event) bool { called = true; return true }
	m.ExecutePass1()
	if called {
		t.Error("暂停过程不应运行")
	}
}

//---- 脚本热重载 ----

func TestScriptHotReload(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.lua")
	os.WriteFile(file, []byte("-- v1\n"), 0o644)
	m := NewScriptManager(nil)
	s, err := m.AddScript("main", file)
	if err != nil {
		t.Fatal(err)
	}
	if m.Execute() != 0 {
		t.Error("mtime 未变不应重载")
	}
	// 改 mtime（官方轮询判据；Windows mtime 粒度 2s——显式设未来时刻）。
	future := time.Now().Add(3 * time.Second)
	os.Chtimes(file, future, future)
	if n := m.Execute(); n != 1 || s.Loads != 1 {
		t.Errorf("mtime 变更应重载: n=%d loads=%d", n, s.Loads)
	}
	if _, ok := m.FindScript("MAIN"); !ok {
		t.Error("FindScript 应大小写不敏感")
	}
}

//---- 内核 Startup/泵 ----

func newTestCore(t *testing.T) (*Core, string) {
	t.Helper()
	dir := t.TempDir()
	luaDir := filepath.Join(dir, "lua")
	os.MkdirAll(luaDir, 0o755)
	os.WriteFile(filepath.Join(luaDir, "main.lua"), []byte("-- main\n"), 0o644)

	registry := fsloader.NewModuleRegistry()
	registry.Register(&fsloader.DeclModule{ModuleName: "fskit.dll", ModuleSpace: "FsKit"})
	core := New(registry, Hooks{})
	core.Entities.AddCreator(&EntInfo{
		TypeName: "MainCon", Space: "FxCore",
		New: func(*fskit.VarList) any { return map[string]any{} },
	})
	return core, luaDir
}

func testConfig(luaDir string) *fsloader.CoreConfig {
	return &fsloader.CoreConfig{
		MainEntity: "MainCon",
		MainScript: "main",
		ScriptPath: luaDir + string(os.PathSeparator),
		Modules:    []string{"fskit.dll"},
	}
}

func TestCoreStartupSequence(t *testing.T) {
	core, luaDir := newTestCore(t)
	if err := core.Startup(testConfig(luaDir)); err != nil {
		t.Fatalf("Startup: %v", err)
	}
	if core.Main == nil || core.Entities.Lookup("MainCon") != core.Main {
		t.Error("主实体应物化并注册")
	}
	if _, ok := core.Scripts.FindScript("main"); !ok {
		t.Error("主脚本应登记")
	}
	if len(core.registry.Loaded()) != 1 {
		t.Error("应装载 1 模块")
	}
	// 主脚本缺文件 → 官方致命文案（官方 Startup 单次语义——新内核复测）。
	badCore, _ := newTestCore(t)
	bad := testConfig(luaDir)
	bad.MainScript = "nope"
	if err := badCore.Startup(bad); err == nil ||
		!strings.Contains(err.Error(), "(CCore::Startup)run main script failed") {
		t.Fatalf("缺脚本文案: %v", err)
	}
	// 主逻辑无钩子 → 官方致命文案。
	logicCore, _ := newTestCore(t)
	logic := testConfig(luaDir)
	logic.MainLogic = "GameLogic"
	if err := logicCore.Startup(logic); err == nil ||
		!strings.Contains(err.Error(), "(CCore::Startup)bind main logic failed") {
		t.Fatalf("主逻辑文案: %v", err)
	}
}

func TestCorePumpFixedStep(t *testing.T) {
	core, luaDir := newTestCore(t)
	if err := core.Startup(testConfig(luaDir)); err != nil {
		t.Fatal(err)
	}
	var steps []float64
	core.Entities.AddFrameExecute("test", func(_ *Core, step float64) {
		steps = append(steps, step)
	})
	core.SetFPSCap(50) // step = 0.02
	// 首拍：帧钟 0 ≤ 虚拟时间 → 立即推进一个定步（官方语义），
	// 余量 offset≈0 可能再调一次（尾帧余量拍）。
	core.Process()
	if len(steps) == 0 || steps[0] != 0.02 {
		t.Fatalf("首拍应推进一个定步: %v", steps)
	}
	steps = nil
	// 虚拟时间前跳 5 步：应补 5 个定步 + 1 个尾帧余量拍（offset）。
	core.virtualTime += 5 * 0.02
	core.Process()
	full := 0
	for i, s := range steps {
		if s == 0.02 {
			full++
		} else if i != len(steps)-1 {
			t.Fatalf("非末位出现余量步: %v", steps)
		}
	}
	if full != 5 {
		t.Fatalf("应补 5 个定步: %v", steps)
	}
	// 补帧上限 10（官方 [ebp-0x174]<0x64 循环上限）。
	steps = nil
	core.virtualTime += 20 * 0.02
	core.Process()
	if len(steps) != 11 { // 10 定步 + 1 尾帧余量（offset 拍）
		t.Fatalf("补帧上限语义: %d 步 %v", len(steps), steps)
	}
	if core.StatsView().Frame < 0 {
		t.Error("帧统计不应为负")
	}
}

func TestCorePumpFullSpeedAndQuit(t *testing.T) {
	core, luaDir := newTestCore(t)
	if err := core.Startup(testConfig(luaDir)); err != nil {
		t.Fatal(err)
	}
	calls := 0
	core.Entities.AddFrameExecute("test", func(*Core, float64) { calls++ })
	for i := 0; i < 5; i++ {
		if !core.Process() {
			t.Fatal("未停机 Process 应 true")
		}
	}
	if calls != 5 {
		t.Errorf("全速模式每拍一帧: calls=%d", calls)
	}
	core.Quit()
	if core.Process() {
		t.Error("Quit 后 Process 应 false（官方停机标志）")
	}
	core.Shutdown()
}

func TestCoreDriveEngineApp(t *testing.T) {
	core, luaDir := newTestCore(t)
	if err := core.Startup(testConfig(luaDir)); err != nil {
		t.Fatal(err)
	}
	app := NewEngineApp(0)
	core.Drive(app, "world")
	var ticked uint64
	app.Sched.AddFunc("probe", func(w *ecs.World, dt float64) {
		ticked = w.CurrentTick()
	})
	core.Process()
	if ticked != 1 {
		t.Errorf("Drive 应推进 engine.App 一帧: tick=%d", ticked)
	}
}

//---- API 桥 ----

func TestAPICallEntityOps(t *testing.T) {
	core, luaDir := newTestCore(t)
	core.Startup(testConfig(luaDir))
	// nx_create + nx_lookup。
	args := &fskit.VarList{}
	args.AddString("MainCon")
	args.AddString("extra1")
	res := &fskit.VarList{}
	if !APICall(core, "nx_create", args, res) {
		t.Fatal("nx_create 失败")
	}
	id := res.GetInt(0)
	args2 := &fskit.VarList{}
	args2.AddString("extra1")
	res2 := &fskit.VarList{}
	if !APICall(core, "nx_lookup", args2, res2) || res2.GetInt(0) != id {
		t.Fatal("nx_lookup 应命中")
	}
	// nx_name/nx_type。
	args3 := &fskit.VarList{}
	args3.AddInt(id)
	resName := &fskit.VarList{}
	APICall(core, "nx_name", args3, resName)
	if resName.GetString(0) != "extra1" {
		t.Fatalf("nx_name=%q", resName.GetString(0))
	}
	resType := &fskit.VarList{}
	APICall(core, "nx_type", args3, resType)
	if resType.GetString(0) != "MainCon" {
		t.Fatalf("nx_type=%q", resType.GetString(0))
	}
	// nx_gen_event + 等待者唤醒。
	waiter := core.Async.Add("w", func(p *AsyncProc, ev *Event) bool {
		if ev == nil {
			p.AddWaitEvent(EntityID(id))
			return false
		}
		p.done = true
		return false
	})
	core.Process() // 挂起
	args4 := &fskit.VarList{}
	args4.AddInt(id)
	APICall(core, "nx_gen_event", args4, &fskit.VarList{})
	core.Process() // 唤醒 + 清理
	if !waiter.Done() || core.Async.Count() != 0 {
		t.Fatal("事件应唤醒并清理等待者")
	}
	// nx_quit。
	if !APICall(core, "nx_quit", &fskit.VarList{}, &fskit.VarList{}) {
		t.Fatal("nx_quit 失败")
	}
	if core.Process() {
		t.Error("nx_quit 后应停机")
	}
	// Lua 依赖函数不可调。
	if APICall(core, "nx_call", &fskit.VarList{}, &fskit.VarList{}) {
		t.Error("nx_call 为 Lua 依赖，Go 桥应 false")
	}
}

//---- 时钟 ----

func TestGameTimer(t *testing.T) {
	tm := NewGameTimer()
	if tm.Execute(0.016) != 0.016 || tm.SimTime() != 0.016 {
		t.Error("定步推进错误")
	}
	if tm.SimElapsed() != 0.016 {
		t.Error("SimElapsed 错误")
	}
	tm.Execute(0.016)
	if tm.SimTime() != 0.032 {
		t.Error("累计错误")
	}
	if clampFrame(0.5) != 0.25 {
		t.Error("dt 钳制语义")
	}
	if u64Tick() == 0 {
		t.Error("u64Tick")
	}
}
