package fxcore

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// nx_* Lua API 面（官方字符串表全量枚举 73 函数，FxCore.dll 注册进
// Lua 全局）。类别标注 + Go 桥实现状态：
//
//	✅ = Go 桥已实现（apiCall 分发，宿主无需 Lua 即可用）
//	⏳ = Lua VM 依赖（里程碑二接线；当前声明于表）
type APIDecl struct {
	Name     string
	Category string // entity/property/method/script/logic/event/timer/path/logic/system/value/widestr
	Status   string // "go" | "lua"
	Call     func(c *Core, args *fskit.VarList, result *fskit.VarList) bool
}

// APITable 官方 73 函数全量声明（实现状态见 Status）。
var APITable = buildAPITable()

func buildAPITable() []APIDecl {
	t := []APIDecl{
		// ---- 实体 ----
		{Name: "nx_create", Category: "entity", Status: "go", Call: apiCreate},
		{Name: "nx_destroy", Category: "entity", Status: "go", Call: apiDestroy},
		{Name: "nx_lookup", Category: "entity", Status: "go", Call: apiLookup},
		{Name: "nx_name", Category: "entity", Status: "go", Call: apiName},
		{Name: "nx_type", Category: "entity", Status: "go", Call: apiType},
		{Name: "nx_space", Category: "entity", Status: "go", Call: apiSpace},
		{Name: "nx_is_valid", Category: "entity", Status: "go", Call: apiIsValid},
		{Name: "nx_is_null", Category: "entity", Status: "go", Call: apiIsNull},
		{Name: "nx_id_equal", Category: "entity", Status: "go", Call: apiIDEqual},
		{Name: "nx_single", Category: "entity", Status: "go", Call: apiSingle},
		{Name: "nx_set_single", Category: "entity", Status: "go", Call: apiSetSingle},
		{Name: "nx_set_value", Category: "entity", Status: "go", Call: apiSetValue},
		{Name: "nx_value", Category: "entity", Status: "go", Call: apiValue},
		{Name: "nx_find_value", Category: "entity", Status: "go", Call: apiFindValue},
		{Name: "nx_remove_value", Category: "entity", Status: "go", Call: apiRemoveValue},
		{Name: "nx_custom", Category: "entity", Status: "go", Call: apiCustom},
		{Name: "nx_set_custom", Category: "entity", Status: "go", Call: apiSetCustom},
		{Name: "nx_find_custom", Category: "entity", Status: "go", Call: apiFindCustom},
		{Name: "nx_custom_list", Category: "entity", Status: "go", Call: apiCustomList},
		{Name: "nx_path_create", Category: "entity", Status: "go", Call: apiPathCreate},
		{Name: "nx_path_delete", Category: "entity", Status: "go", Call: apiPathDelete},
		{Name: "nx_path_exists", Category: "entity", Status: "go", Call: apiPathExists},
		// ---- 类型 ----
		{Name: "nx_is_kind", Category: "entity", Status: "go", Call: apiIsKind},
		{Name: "nx_find_kind", Category: "entity", Status: "go", Call: apiFindKind},
		// ---- 属性/方法 ----
		{Name: "nx_find_property", Category: "property", Status: "go", Call: apiFindProperty},
		{Name: "nx_property", Category: "property", Status: "go", Call: apiProperty},
		{Name: "nx_set_property", Category: "property", Status: "go", Call: apiSetProperty},
		{Name: "nx_property_list", Category: "property", Status: "go", Call: apiPropertyList},
		{Name: "nx_find_method", Category: "method", Status: "go", Call: apiFindMethod},
		{Name: "nx_method", Category: "method", Status: "go", Call: apiMethod},
		{Name: "nx_method_list", Category: "method", Status: "go", Call: apiMethodList},
		// ---- 事件/异步 ----
		{Name: "nx_gen_event", Category: "event", Status: "go", Call: apiGenEvent},
		{Name: "nx_wait_event", Category: "event", Status: "go", Call: apiWaitEvent},
		{Name: "nx_pause", Category: "event", Status: "go", Call: apiPause},
		{Name: "nx_strict_pause", Category: "event", Status: "go", Call: apiStrictPause},
		{Name: "nx_kill", Category: "event", Status: "go", Call: apiKill},
		{Name: "nx_running", Category: "event", Status: "go", Call: apiRunning},
		// ---- 时钟 ----
		{Name: "nx_time_begin", Category: "timer", Status: "go", Call: apiTimeBegin},
		{Name: "nx_time_elapse", Category: "timer", Status: "go", Call: apiTimeElapse},
		{Name: "nx_elapse", Category: "timer", Status: "go", Call: apiElapse},
		{Name: "nx_total_seconds", Category: "timer", Status: "go", Call: apiTotalSeconds},
		// ---- 路径/环境 ----
		{Name: "nx_work_path", Category: "path", Status: "go", Call: apiWorkPath},
		{Name: "nx_script_path", Category: "path", Status: "go", Call: apiScriptPath},
		{Name: "nx_resource_path", Category: "path", Status: "go", Call: apiResourcePath},
		{Name: "nx_file_exists", Category: "path", Status: "go", Call: apiFileExists},
		{Name: "nx_file_delete", Category: "path", Status: "go", Call: apiFileDelete},
		{Name: "nx_platform", Category: "path", Status: "go", Call: apiPlatform},
		// ---- 日志/系统 ----
		{Name: "nx_log", Category: "log", Status: "go", Call: apiLog},
		{Name: "nx_debug_log", Category: "log", Status: "go", Call: apiDebugLog},
		{Name: "nx_is_debug", Category: "log", Status: "go", Call: apiIsDebug},
		{Name: "nx_dump_file", Category: "log", Status: "go", Call: apiDumpFile},
		{Name: "nx_msgbox", Category: "log", Status: "go", Call: apiMsgbox},
		{Name: "nx_quit", Category: "system", Status: "go", Call: apiQuit},
		{Name: "nx_assert", Category: "system", Status: "go", Call: apiAssert},
		{Name: "nx_current", Category: "system", Status: "go", Call: apiCurrent},
		{Name: "nx_execute", Category: "system", Status: "go", Call: apiExecute},
		// ---- 脚本/逻辑（Lua VM 依赖） ----
		{Name: "nx_call", Category: "script", Status: "lua"},
		{Name: "nx_find_script", Category: "script", Status: "lua"},
		{Name: "nx_find_scriptl", Category: "script", Status: "lua"},
		{Name: "nx_script_name", Category: "script", Status: "lua"},
		{Name: "nx_bind_script", Category: "script", Status: "lua"},
		{Name: "nx_bind_logic", Category: "script", Status: "lua"},
		{Name: "nx_callback", Category: "script", Status: "lua"},
		{Name: "nx_logic_name", Category: "script", Status: "lua"},
		// ---- 值构造（Lua 侧类型面） ----
		{Name: "nx_int", Category: "value", Status: "lua"},
		{Name: "nx_int64", Category: "value", Status: "lua"},
		{Name: "nx_float", Category: "value", Status: "lua"},
		{Name: "nx_double", Category: "value", Status: "lua"},
		{Name: "nx_boolean", Category: "value", Status: "lua"},
		{Name: "nx_string", Category: "value", Status: "lua"},
		{Name: "nx_widestr", Category: "value", Status: "lua"},
		{Name: "nx_null", Category: "value", Status: "lua"},
		{Name: "nx_object", Category: "value", Status: "lua"},
		{Name: "nx_number", Category: "value", Status: "lua"},
		{Name: "nx_decimals", Category: "value", Status: "lua"},
		{Name: "nx_function", Category: "value", Status: "lua"},
		{Name: "nx_value_list", Category: "value", Status: "lua"},
		// ---- 宽串工具 ----
		{Name: "nx_ws_equal", Category: "widestr", Status: "lua"},
		{Name: "nx_ws_length", Category: "widestr", Status: "lua"},
		{Name: "nx_ws_lower", Category: "widestr", Status: "lua"},
		{Name: "nx_ws_upper", Category: "widestr", Status: "lua"},
		// ---- 主入口（装载器注入的全局） ----
		{Name: "nx_main", Category: "system", Status: "go"},
		{Name: "nx_main_config", Category: "system", Status: "go"},
	}
	sort.Slice(t, func(i, j int) bool { return t[i].Name < t[j].Name })
	return t
}

// APICall Go 桥分发：按名调用（未实现/Lua 依赖返回 false）。
func APICall(c *Core, name string, args *fskit.VarList, result *fskit.VarList) bool {
	for _, d := range APITable {
		if d.Name == name {
			if d.Call == nil {
				return false
			}
			return d.Call(c, args, result)
		}
	}
	return false
}

// APINames 全量名表（对拍面：与官方 73 函数清单逐名核对）。
func APINames() []string {
	out := make([]string, 0, len(APITable))
	for _, d := range APITable {
		out = append(out, d.Name)
	}
	return out
}

// ---- 实现体（官方 nx_* 的 CCore 方法映射） ----

func apiCreate(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 1 {
		return false // 官方 "nx_create)need at least 1 argument"
	}
	typeName := a.GetString(0)
	name := ""
	if a.Count() > 1 {
		name = a.GetString(1)
	}
	e, err := c.Entities.Create(typeName, name, a)
	if err != nil {
		return false
	}
	r.AddInt(int64(uint64(e.ID)))
	return true
}

func apiDestroy(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || !e.CanDelByScript() {
		return false // 官方 "this entity can't delete by script"
	}
	c.Entities.SetDeleted(e)
	return true
}

func apiLookup(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.Lookup(a.GetString(0))
	if e == nil {
		return false
	}
	r.AddInt(int64(uint64(e.ID)))
	return true
}

func apiName(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	r.AddString(e.Name)
	return true
}

func apiType(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	r.AddString(e.Info.TypeName)
	return true
}

func apiSpace(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	r.AddString(e.Info.Space)
	return true
}

func apiIsValid(c *Core, a, r *fskit.VarList) bool {
	r.AddInt(boolInt(c.Entities.GetByID(EntityID(a.GetInt(0))) != nil))
	return true
}

func apiIsNull(c *Core, a, r *fskit.VarList) bool {
	r.AddInt(boolInt(EntityID(a.GetInt(0)) == 0))
	return true
}

func apiIDEqual(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 2 {
		return false // 官方 "need entity id 1/2"
	}
	r.AddInt(boolInt(a.GetInt(0) == a.GetInt(1)))
	return true
}

func apiSingle(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetSingle(a.GetString(0))
	if e == nil {
		return false
	}
	r.AddInt(int64(uint64(e.ID)))
	return true
}

func apiSetSingle(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	c.Entities.SetSingle(e)
	return true
}

func apiSetValue(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	e.props[a.GetString(1)] = apiArgAt(a, 2)
	return true
}

func apiValue(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	apiResultAdd(r, e.props[a.GetString(1)])
	return true
}

func apiFindValue(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	_, ok := e.props[a.GetString(1)]
	r.AddInt(boolInt(ok))
	return true
}

func apiRemoveValue(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	delete(e.props, a.GetString(1))
	return true
}

func apiCustom(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	apiResultAdd(r, e.userData)
	return true
}

func apiSetCustom(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	e.userData = apiArgAt(a, 1)
	return true
}

func apiFindCustom(c *Core, a, r *fskit.VarList) bool { return apiCustom(c, a, r) }

func apiCustomList(c *Core, a, r *fskit.VarList) bool {
	r.AddString("userdata")
	return true
}

func apiPathCreate(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 1 {
		return false
	}
	e := c.Entities.Lookup(a.GetString(0))
	if e != nil {
		r.AddInt(int64(uint64(e.ID)))
		return true
	}
	ne, err := c.Entities.Create("ArrayList", a.GetString(0), nil)
	if err != nil {
		return false
	}
	r.AddInt(int64(uint64(ne.ID)))
	return true
}

func apiPathDelete(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.Lookup(a.GetString(0))
	if e == nil {
		return false
	}
	c.Entities.SetDeleted(e)
	return true
}

func apiPathExists(c *Core, a, r *fskit.VarList) bool {
	r.AddInt(boolInt(c.Entities.Lookup(a.GetString(0)) != nil))
	return true
}

func apiIsKind(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	r.AddInt(boolInt(e.Info.IsKindOf(a.GetString(1))))
	return true
}

func apiFindKind(c *Core, a, r *fskit.VarList) bool {
	info, ok := c.Entities.FindInfo(a.GetString(0))
	if !ok {
		return false
	}
	r.AddString(info.TypeName)
	if info.ParentName != "" {
		r.AddString(info.ParentName)
	}
	return true
}

func apiFindProperty(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	for _, p := range e.Info.Props {
		if strings.EqualFold(p.Name, a.GetString(1)) {
			r.AddString(p.Name)
			return true
		}
	}
	return false
}

func apiProperty(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	key := a.GetString(1)
	for _, p := range e.Info.Props {
		if strings.EqualFold(p.Name, key) {
			apiResultAdd(r, e.props[key])
			return true
		}
	}
	apiResultAdd(r, e.props[key])
	return true
}

func apiSetProperty(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 3 {
		return false
	}
	e.props[a.GetString(1)] = apiArgAt(a, 2)
	return true
}

func apiPropertyList(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	for _, p := range e.Info.Props {
		r.AddString(p.Name)
	}
	return true
}

func apiFindMethod(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	for _, m := range e.Info.Methods {
		if strings.EqualFold(m.Name, a.GetString(1)) {
			r.AddString(m.Name)
			return true
		}
	}
	return false
}

func apiMethod(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil || a.Count() < 2 {
		return false
	}
	name := a.GetString(1)
	for _, m := range e.Info.Methods {
		if strings.EqualFold(m.Name, name) && m.Call != nil {
			return m.Call(e, a, r)
		}
	}
	return false
}

func apiMethodList(c *Core, a, r *fskit.VarList) bool {
	e := c.Entities.GetByID(EntityID(a.GetInt(0)))
	if e == nil {
		return false
	}
	for _, m := range e.Info.Methods {
		r.AddString(m.Name)
	}
	return true
}

func apiGenEvent(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 1 {
		return false
	}
	c.Async.GenEvent(&Event{Object: EntityID(a.GetInt(0))})
	return true
}

func apiWaitEvent(c *Core, a, r *fskit.VarList) bool {
	// Go 桥形态：宿主过程自行 AddWaitEvent；此处校验目标并回带。
	if a.Count() < 1 {
		return false
	}
	r.AddInt(int64(uint64(EntityID(a.GetInt(0)))))
	return true
}

func apiPause(c *Core, a, r *fskit.VarList) bool {
	if p := c.Async.Find(int(a.GetInt(0))); p != nil {
		p.Pause(false)
		return true
	}
	return false
}

func apiStrictPause(c *Core, a, r *fskit.VarList) bool {
	if p := c.Async.Find(int(a.GetInt(0))); p != nil {
		p.Pause(true)
		return true
	}
	return false
}

func apiKill(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 1 {
		return false
	}
	if p := c.Async.Find(int(a.GetInt(0))); p != nil {
		p.done = true
		return true
	}
	return false
}

func apiRunning(c *Core, a, r *fskit.VarList) bool {
	p := c.Async.Find(int(a.GetInt(0)))
	r.AddInt(boolInt(p != nil && !p.done))
	return true
}

// ---- 时钟 ----

var timeBegins = map[string]float64{}

func apiTimeBegin(c *Core, a, r *fskit.VarList) bool {
	key := "default"
	if a.Count() > 0 {
		key = a.GetString(0)
	}
	timeBegins[key] = c.Timer.Now()
	return true
}

func apiTimeElapse(c *Core, a, r *fskit.VarList) bool {
	key := "default"
	if a.Count() > 0 {
		key = a.GetString(0)
	}
	r.AddFloat(c.Timer.Now() - timeBegins[key])
	return true
}

func apiElapse(c *Core, a, r *fskit.VarList) bool {
	r.AddFloat(c.lastElapsed)
	return true
}

func apiTotalSeconds(c *Core, a, r *fskit.VarList) bool {
	r.AddFloat(c.totalTime)
	return true
}

// ---- 路径/环境 ----

func apiWorkPath(c *Core, a, r *fskit.VarList) bool {
	if c.Config != nil && c.Config.ScriptPath != "" {
		r.AddString(c.Config.ScriptPath)
		return true
	}
	r.AddString("./")
	return true
}

func apiScriptPath(c *Core, a, r *fskit.VarList) bool {
	if c.Config == nil {
		return false
	}
	r.AddString(c.Config.ScriptPath)
	return true
}

func apiResourcePath(c *Core, a, r *fskit.VarList) bool {
	if c.Config == nil {
		return false
	}
	r.AddString(c.Config.ResourcePath)
	return true
}

func apiFileExists(c *Core, a, r *fskit.VarList) bool {
	r.AddInt(boolInt(fileExists(a.GetString(0))))
	return true
}

func apiFileDelete(c *Core, a, r *fskit.VarList) bool {
	if err := os.Remove(a.GetString(0)); err != nil {
		return false
	}
	return true
}

func apiPlatform(c *Core, a, r *fskit.VarList) bool {
	r.AddString("windows")
	return true
}

// ---- 日志/系统 ----

func apiLog(c *Core, a, r *fskit.VarList) bool {
	c.Log.Trace("%s", a.GetString(0))
	return true
}

func apiDebugLog(c *Core, a, r *fskit.VarList) bool {
	r.AddInt(boolInt(c.Config != nil && c.Config.DebugLog))
	return true
}

func apiIsDebug(c *Core, a, r *fskit.VarList) bool {
	return apiDebugLog(c, a, r)
}

func apiDumpFile(c *Core, a, r *fskit.VarList) bool {
	if a.Count() < 2 {
		return false
	}
	c.Log.Trace("dump %s: %s", a.GetString(0), a.GetString(1))
	return true
}

func apiMsgbox(c *Core, a, r *fskit.VarList) bool {
	c.Log.Trace("msgbox: %s", a.GetString(0))
	return true
}

func apiQuit(c *Core, a, r *fskit.VarList) bool {
	c.Quit()
	return true
}

func apiAssert(c *Core, a, r *fskit.VarList) bool {
	if a.Count() > 0 && a.GetInt(0) == 0 {
		c.Log.Trace("(nx_assert)assertion failed")
		return false
	}
	return true
}

func apiCurrent(c *Core, a, r *fskit.VarList) bool {
	if c.Main == nil {
		return false
	}
	r.AddInt(int64(uint64(c.Main.ID)))
	return true
}

func apiExecute(c *Core, a, r *fskit.VarList) bool {
	// 官方=执行脚本文件（Lua VM 依赖）；Go 桥登记并报延迟。
	if a.Count() < 1 {
		return false
	}
	if s, ok := c.Scripts.FindScript(a.GetString(0)); ok {
		c.Log.Trace("nx_execute: script %s located (lua pending), loads=%d", s.Name, s.Loads)
		return true
	}
	return false
}

// ---- 小工具 ----

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func apiArgAt(a *fskit.VarList, i int) any {
	if a == nil || a.Count() <= i {
		return nil
	}
	switch a.GetKind(i) {
	case fskit.VarInt:
		return a.GetInt(i)
	case fskit.VarFloat:
		return a.GetFloat(i)
	default:
		return a.GetString(i)
	}
}

func apiResultAdd(r *fskit.VarList, v any) {
	switch t := v.(type) {
	case nil:
		r.AddString("")
	case int:
		r.AddInt(int64(t))
	case int64:
		r.AddInt(t)
	case float64:
		r.AddFloat(t)
	case bool:
		r.AddString(fmt.Sprintf("%v", t))
	default:
		r.AddString(fmt.Sprintf("%v", t))
	}
}

// fileExists 官方 nx_file_exists 语义面。
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
