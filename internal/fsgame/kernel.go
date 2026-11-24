package fsgame

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Kernel 复刻宿主 CKernel（IKernel 507 虚槽）的登录闭环子面：
// 类注册表 / 对象表 / 六类回调触发 / 场景表 / 存档出口 / 模块互访。
// 官方为纯虚接口 + 宿主实现；Go 侧以单一结构承载，玩法模块仅依赖此面。
type Kernel struct {
	mu sync.Mutex

	classes map[string]*ClassDef
	objects map[PersistID]*GameObject
	nextID  uint32

	bus *Bus

	merged map[string]*ClassDef // 类 → 祖先链合并 schema 缓存（role 继承 player）

	modules map[string]Module // 模块名 → 实例（GetLogicModule 面）

	// 场景面（GetScene*/GetSceneId/GetScenePlayerList 的最小载体）。
	scenes    map[int32]*SceneInfo
	sceneObjs map[int32]PersistID // 场景对象句柄
	curScene  int32

	// 环境面（GetDistrictId/GetServerId/GetMemberId）。
	DistrictID int32
	ServerID   int32
	MemberID   int32

	// ResourcePath 官方 GetResourcePath（宿主 ..\res）。
	ResourcePath string

	// Lua 桥（run_logic_lua_script 面；VM 承载沿 FxCore 里程碑二决策）。
	Lua func(script string, args VarList) error

	// SaveOut 存档出口（SaveChunk/SavePlayerData → store 服链路挂点）。
	SaveOut func(role PersistID, chunk string, props map[string]Var, records map[string][]VarList)

	// Trace 日志面（CKernel::Trace）。
	Trace func(msg string)
}

// SceneInfo 场景登记（scenes.ini 装载产物；fssolomember 已有完整装载链）。
type SceneInfo struct {
	ID       int32
	Script   string
	Config   string
	Class    string // NormalScene / CloneScene / ...
	Clonable bool
}

// NewKernel 创建内核（DistrictID=1/ServerID=11 对齐 game_unite.ini [Main]）。
func NewKernel() *Kernel {
	k := &Kernel{
		classes:    make(map[string]*ClassDef),
		objects:    make(map[PersistID]*GameObject),
		nextID:     1,
		bus:        NewBus(),
		merged:     make(map[string]*ClassDef),
		modules:    make(map[string]Module),
		scenes:     make(map[int32]*SceneInfo),
		sceneObjs:  make(map[int32]PersistID),
		DistrictID: 1,
		ServerID:   11,
	}
	k.Trace = func(string) {}
	return k
}

// Bus 总线（模块 Init 期注册面）。
func (k *Kernel) Bus() *Bus { return k.bus }

// Class 按名取类定义。
func (k *Kernel) Class(name string) *ClassDef { return k.classes[name] }

// AddLogicClass 登记类（官方 AddLogicClass(name, id, base)）。
func (k *Kernel) AddLogicClass(c ClassDef) error {
	if c.Name == "" {
		return fmt.Errorf("fsgame: 空类名")
	}
	if c.Parent != "" && k.classes[c.Parent] == nil {
		return fmt.Errorf("fsgame: 类 %s 父类 %s 未登记", c.Name, c.Parent)
	}
	withEngineInternals(&c)
	c.seal()
	k.classes[c.Name] = &c
	return nil
}

// LoadLogicClassXML 解析 logic_class.xml 并按树序（父先子后）登记全部类。
func (k *Kernel) LoadLogicClassXML(data []byte) error {
	var root xmlLogicClass
	if err := xml.NewDecoder(bytesReader(data)).Decode(&root); err != nil {
		return fmt.Errorf("fsgame: logic_class.xml: %w", err)
	}
	var walk func(nodes []xmlClassNode, parent string) error
	walk = func(nodes []xmlClassNode, parent string) error {
		for _, n := range nodes {
			c := ClassDef{Name: n.ID, Parent: parent, Type: n.Type, Desc: n.Desc}
			c.seal()
			if err := k.AddLogicClass(c); err != nil {
				return err
			}
			if err := walk(n.Children, n.ID); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root.Classes, "")
}

// LoadStruct 装载类 schema（createclass_from_xml 等价：logic_class 树节点
// + struct/<class>.xml 属性/记录定义），装载完成触发类级 OnCreateClass。
// 官方 XML 的 <includes> 段（相对 Res 根、反斜杠路径）按 ResourcePath 递归装载。
func (k *Kernel) LoadStruct(class string, xmlData []byte) error {
	cd := k.classes[class]
	if cd == nil {
		return fmt.Errorf("fsgame: 类 %s 未登记（先装载 logic_class.xml）", class)
	}
	includes, err := ParseStructXML(bytesReader(xmlData), cd)
	if err != nil {
		return err
	}
	for _, inc := range includes {
		if k.ResourcePath == "" {
			return fmt.Errorf("fsgame: %s 引用 include %s 但 ResourcePath 未设置", class, inc)
		}
		rel := filepath.FromSlash(strings.ReplaceAll(inc, "\\", "/"))
		data, err := res.ReadFile(filepath.Join(k.ResourcePath, rel))
		if err != nil {
			return fmt.Errorf("fsgame: %s include %s: %w", class, inc, err)
		}
		if err := k.loadStructNoEvent(class, data, inc, map[string]bool{inc: true}); err != nil {
			return err
		}
	}
	for name := range k.merged { // schema 变更后重建合并缓存
		if name == class || k.isDescendant(name, class) {
			delete(k.merged, name)
		}
	}
	k.bus.FireClassEvent(k, class, "OnCreateClass")
	return nil
}

// loadStructNoEvent includes 递归装载体（不触发 OnCreateClass——官方整树装载
// 完成后才回调一次；visited 防环）。
func (k *Kernel) loadStructNoEvent(class string, xmlData []byte, src string, visited map[string]bool) error {
	cd := k.classes[class]
	if cd == nil {
		return fmt.Errorf("fsgame: 类 %s 未登记", class)
	}
	includes, err := ParseStructXML(bytesReader(xmlData), cd)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	for _, inc := range includes {
		if visited[inc] {
			continue
		}
		visited[inc] = true
		rel := filepath.FromSlash(strings.ReplaceAll(inc, "\\", "/"))
		data, err := res.ReadFile(filepath.Join(k.ResourcePath, rel))
		if err != nil {
			return fmt.Errorf("fsgame: %s include %s: %w", class, inc, err)
		}
		if err := k.loadStructNoEvent(class, data, inc, visited); err != nil {
			return err
		}
	}
	return nil
}

// MergedClass 类链合并 schema：本体 + 祖先（属性/记录，子类同名覆盖）。
// 官方子类对象（role/player、AttackNpc/VisibleNpc）共享祖先 schema。
func (k *Kernel) MergedClass(class string) *ClassDef {
	if m, ok := k.merged[class]; ok {
		return m
	}
	cd := k.classes[class]
	if cd == nil {
		return nil
	}
	chain := []*ClassDef{cd}
	for p := cd.Parent; p != ""; {
		pc := k.classes[p]
		if pc == nil {
			break
		}
		chain = append(chain, pc)
		p = pc.Parent
	}
	m := &ClassDef{Name: cd.Name, Parent: cd.Parent, Type: cd.Type, Desc: cd.Desc}
	for i := len(chain) - 1; i >= 0; i-- { // 祖先先、子孙后（子类覆盖）
		m.Props = append(m.Props, chain[i].Props...)
		m.Records = append(m.Records, chain[i].Records...)
	}
	// 同名去重（保留后写=子类）
	dedupe := func() {
		seenP := make(map[string]bool)
		props := m.Props[:0]
		for _, p := range m.Props {
			if !seenP[p.Name] {
				seenP[p.Name] = true
				props = append(props, p)
			}
		}
		m.Props = props
		seenR := make(map[string]bool)
		recs := m.Records[:0]
		for _, r := range m.Records {
			if !seenR[r.Name] {
				seenR[r.Name] = true
				recs = append(recs, r)
			}
		}
		m.Records = recs
	}
	dedupe()
	m.seal()
	k.merged[class] = m
	return m
}

// isDescendant name 是否 base 的（含跨级）子孙。
func (k *Kernel) isDescendant(name, base string) bool {
	for c := k.classes[name]; c != nil && c.Parent != ""; c = k.classes[c.Parent] {
		if c.Parent == base {
			return true
		}
	}
	return false
}

// CreateObject 物化对象（官方 CreateObject(class, config, id, x, y, z, orient)；
// Go 面只取类，坐标经属性面写）。触发类 OnCreate 事件。
func (k *Kernel) CreateObject(class string) (PersistID, error) {
	cd := k.MergedClass(class)
	if cd == nil {
		return NullPersistID, fmt.Errorf("fsgame: 未登记类 %s", class)
	}
	k.mu.Lock()
	id := MakePersistID(k.nextID, 1)
	k.nextID++
	obj := newGameObject(id, cd)
	obj.setRecNotifier(func(record string, op, row int32, vals VarList) {
		k.bus.FireRecHook(k, id, record, op, row)
	})
	k.objects[id] = obj
	k.mu.Unlock()
	k.bus.FireEvent(k, class, "OnCreate", id, NullPersistID, nil)
	return id, nil
}

// GetGameObj 取对象（官方 GetGameObj(PERSISTID)；悬挂/未知返回 nil）。
func (k *Kernel) GetGameObj(id PersistID) *GameObject {
	return k.objects[id]
}

// GetLogicModule 按名取模块（官方 IKernel::GetLogicModule；跨模块协作面，
// 如 LocationModule 缓存 AsynCtrlModule/NpcSwitchLocateModule）。
func (k *Kernel) GetLogicModule(name string) Module { return k.modules[name] }

// SeekRoleName 按角色名查在线玩家（官方 IKernel::SeekRoleName；好友/组队/
// 私聊等社交面的目标定位。name 属性为引擎内部属性）。
func (k *Kernel) SeekRoleName(name string) PersistID {
	for id, o := range k.objects {
		if o.ClassName() == "player" || k.isDescendant(o.ClassName(), "player") {
			if v, ok := o.Get("name").(string); ok && v == name {
				return id
			}
		}
	}
	return NullPersistID
}

// PlayersSnapshot 在线玩家句柄集（社交面遍历：好友在线检查/组队广播）。
func (k *Kernel) PlayersSnapshot() []PersistID {
	var out []PersistID
	for id, o := range k.objects {
		if o.ClassName() == "player" || k.isDescendant(o.ClassName(), "player") {
			out = append(out, id)
		}
	}
	return out
}

// SetInt 写 int 属性并触发临界（官方 SetInt + DeclareCritical 通知链）。
func (k *Kernel) SetInt(id PersistID, prop string, v int32) error {
	o := k.objects[id]
	if o == nil {
		return fmt.Errorf("fsgame: 对象 %v 不存在", id)
	}
	old := o.Get(prop)
	if err := o.Set(prop, v); err != nil {
		return err
	}
	if old != Var(v) {
		k.bus.FireCritical(k, o.ClassName(), prop, id, old)
	}
	return nil
}

// SetString 写 string 属性并触发临界。
func (k *Kernel) SetString(id PersistID, prop, v string) error {
	o := k.objects[id]
	if o == nil {
		return fmt.Errorf("fsgame: 对象 %v 不存在", id)
	}
	old := o.Get(prop)
	if err := o.Set(prop, v); err != nil {
		return err
	}
	if old != Var(v) {
		k.bus.FireCritical(k, o.ClassName(), prop, id, old)
	}
	return nil
}

// SetProp 通用写（触发临界）。
func (k *Kernel) SetProp(id PersistID, prop string, v Var) error {
	o := k.objects[id]
	if o == nil {
		return fmt.Errorf("fsgame: 对象 %v 不存在", id)
	}
	old := o.Get(prop)
	if err := o.Set(prop, v); err != nil {
		return err
	}
	if old != v {
		k.bus.FireCritical(k, o.ClassName(), prop, id, old)
	}
	return nil
}

// RunEvent 触发事件（官方 RunEvent/SendEvent；沿类继承链多播）。
func (k *Kernel) RunEvent(class, event string, self, from PersistID, args VarList) int {
	return k.bus.FireEvent(k, class, event, self, from, args)
}

// DispatchCommand 客户端整型命令入口（AddIntCustomHook 注册面）。
func (k *Kernel) DispatchCommand(class string, cmd int32, self, from PersistID, args VarList) bool {
	return k.bus.DispatchCommand(k, class, cmd, self, from, args)
}

// DispatchStringCommand 客户端字符串命令入口（AddCommandHook 注册面）。
func (k *Kernel) DispatchStringCommand(class, name string, self, from PersistID, args VarList) bool {
	return k.bus.DispatchStringCommand(k, class, name, self, from, args)
}

// DispatchIntCommand 命令通道入口（AddIntCommandHook 注册面；与 custom
// 通道分表，见 Bus）。
func (k *Kernel) DispatchIntCommand(class string, cmd int32, self, from PersistID, args VarList) bool {
	return k.bus.DispatchIntCommand(k, class, cmd, self, from, args)
}

// AddScene 登记场景（fssolomember LoadScenesINI 的内核侧落点）。
func (k *Kernel) AddScene(info SceneInfo, sceneObj PersistID) {
	k.scenes[info.ID] = &info
	k.sceneObjs[info.ID] = sceneObj
	if k.curScene == 0 {
		k.curScene = info.ID
	}
}

// GetSceneId 当前场景号（官方 GetSceneId）。
func (k *Kernel) GetSceneId() int32 { return k.curScene }

// GetScene 场景对象句柄（官方 GetScene）。
func (k *Kernel) GetScene() PersistID { return k.sceneObjs[k.curScene] }

// Scenes 全部场景号（升序）。
func (k *Kernel) Scenes() []int32 {
	out := make([]int32, 0, len(k.scenes))
	for id := range k.scenes {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// SavePlayerData 存档出口（官方 SavePlayerData/SaveChunk → store 链；
// Go 面由宿主挂 SaveOut 转发到 fssolostore 管理族）。
func (k *Kernel) SavePlayerData(role PersistID, chunk string) error {
	o := k.objects[role]
	if o == nil {
		return fmt.Errorf("fsgame: 存档对象 %v 不存在", role)
	}
	props, recs := o.Archive()
	if k.SaveOut != nil {
		k.SaveOut(role, chunk, props, recs)
	}
	return nil
}

// Tracef 日志（CKernel::Trace）。
func (k *Kernel) Tracef(format string, a ...any) {
	k.Trace(fmt.Sprintf(format, a...))
}

// ObjectsSnapshot 对象句柄快照（心跳泵遍历面）。
func (k *Kernel) ObjectsSnapshot() []PersistID {
	out := make([]PersistID, 0, len(k.objects))
	for id := range k.objects {
		out = append(out, id)
	}
	return out
}

// RemoveObject 摘除对象并触发 OnDestroy（官方 DeleteObject 语义的登录子面）。
func (k *Kernel) RemoveObject(id PersistID) {
	o := k.objects[id]
	if o == nil {
		return
	}
	delete(k.objects, id)
	k.bus.FireEvent(k, o.ClassName(), "OnDestroy", id, NullPersistID, nil)
}

// unixNow 秒级时钟（util_get_now 等价）。
func unixNow() int64 { return time.Now().Unix() }

// nextMidnight 次日零点（本地时区；日上限类刷新窗锚点）。
func nextMidnight() int64 {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1).Unix()
}
