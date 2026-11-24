// Package skilldata 服务端技能配置只读装载。
//
// 数据源是 tools/datagen skill 的构建产物 ini/Skill/*.xml（黑金模型，见
// docs/official/技能服务端XML构建链与m211受理链.md）：
//
//	Skill_New/<逻辑类>.xml        技能对象行（ID=技能 ConfigID，script 逻辑类）
//	skill_static.xml              静态包行（ID=StaticData；Min/MaxVarPropNo 指逐级行）
//	Skill_VarProp/<逻辑类>.xml    逐级可变属性行（按数值 ID 引用下列包）
//	Skill_Consume / TargetShape / HitShape / Flow / LockData / UseLimit / RushRange
//
// 服务端只读构建产物、不读设计态表；Definition 把四层联查成一份技能定义
// （进程级缓存，按 (技能, 等级) 复用）。列名即官服运行时词汇（官服 Lua 按
// 源表列名取技能对象属性），本包字段注释沿用各列名。
package skilldata

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Script 技能对象逻辑类（skill_new.script = 服务端对象类）。
type Script string

// 主动技能的两个逻辑类；其余（NeiGong/QingGong/RideSkill…）不可经 m211 施放。
const (
	ScriptNormal Script = "SkillNormal"
	ScriptLock   Script = "SkillLock"
)

// 错误域。
var (
	// ErrNotFound 技能 ConfigID 不在技能对象行里（未入库的技能）。
	ErrNotFound = errors.New("skilldata: 技能不存在")
	// ErrUnsupported 逻辑类不是 SkillNormal/SkillLock（内功、轻功等非主动技能）。
	ErrUnsupported = errors.New("skilldata: 非主动技能逻辑类")
)

// 产物文件名（相对 ini/Skill）。
const (
	fileStatic    = "skill_static.xml"
	fileConsume   = "Skill_Consume.xml"
	fileTarget    = "Skill_TargetShape.xml"
	fileHit       = "Skill_HitShape.xml"
	fileFlow      = "Skill_Flow.xml"
	fileLock      = "Skill_LockData.xml"
	fileRushRange = "Skill_RushRange.xml"
	dirNew        = "Skill_New"
	dirVarProp    = "Skill_VarProp"
)

// varPropSpanLimit 逐级行扫描跨度上限（防脏数据把 Min/Max 配成巨区间）。
const varPropSpanLimit = 4096

// Shape 范围包行（attack_targetshape / attack_hitshape）。
type Shape struct {
	Kind    int     // TargetShape/HitShape：0 非范围；1 以自己为圆心的圆内；3 自己正面扇形；4 自己正面矩形
	Para1   float64 // 圆最小半径 / 扇形角度 / 矩形宽
	Para2   float64 // 圆最大半径 / 扇形半径 / 矩形长
	Comment string  // 设计注释
}

// Consume 消耗包行（skill_consume）。
type Consume struct {
	HP, MP, SP int     // AConsumeHP / AConsumeMP / AConsumeSP：直接消耗点数
	QingGong   int     // AConsumeQGP：轻功点
	PercentHP  float64 // PConsumeHP：百分比消耗 HP
	PercentMP  float64 // PConsumeMP：百分比消耗 MP
	ItemID     string  // ConsumeItemID
	ItemCount  int     // ConsumeItemCount
}

// Flow 流程包行（rule/flow_config）。
type Flow struct {
	HitActions     []string // HitAction 逗号列表（普攻连招段）
	HitTimeMS      int      // HitTime
	PrepareTimeMS  int      // PrepareTime
	LeadTimeMS     int      // LeadTime
	LeadSepTimeMS  int      // LeadSepTime
	LockBuffTimeMS int      // LockBuffTime：出招锁定窗
	GRushAction    string   // GRushAction
	ARushAction    string   // ARushAction
	WRushAction    string   // WRushAction
	FailedAction   string   // FailedAction
}

// Lock 锁定包行（skill_lock_data，ID=LockType）。
type Lock struct {
	OK              bool
	Ratio           int     // LockRatio
	SrcTime         int     // LockSrcTime：己方锁定时间 ms
	TargetTime      int     // LockTargetTime：目标锁定时间 ms
	SrcDist         float64 // LockSrcDist：锁定己方位移
	TargetDist      float64 // LockTargetDist：锁定对方位移
	SrcDodgeTime    int     // SrcDodgeTime
	TargetDodgeTime int     // TargetDodgeTime
	SrcDodgeDist    float64 // SrcDodgeDist
	TargetDodgeDist float64 // TargetDodgeDist
	SrcParryTime    int     // SrcParryTime
	TargetParryTime int     // TargetParryTime
	SrcParryDist    float64 // SrcParryDist
	TargetParryDist float64 // TargetParryDist
	HitAction       string  // HitAction
	DodgeAction     string  // DodgeAction
	ParryAction     string  // ParryAction
	// 冲锋动作：地面/空中/水中三档 + 锁定失败动作。
	GroundRushAction string  // LockGRushAction
	AirRushAction    string  // LockARushAction
	WaterRushAction  string  // LockWRushAction
	FailedAction     string  // LockFailedAction
	BindDistance     float64 // BindDistance（缺省 0 = 取默认贴身距离）
}

// RushRange 冲锋范围常量（rush_range [config]）。
type RushRange struct {
	Ground, Air, Water          float64 // GRushRange / ARushRange / WRushRange
	GroundAdd, AirAdd, WaterAdd float64 // *RangeAdd
}

// Definition 一份技能定义：技能对象行 + 静态包行 + 逐级可变属性行 + 引用包。
type Definition struct {
	ID     string
	Script Script
	Level  int
	Levels int // 逐级行数（静态行 Min..MaxVarPropNo 区间内实有行）

	ItemType    int // ItemType：1000 普通技能 / 1001 锁定技 …
	QTypeID     int
	StaticData  int
	VarPropNo   int
	UseLimit    int // Skill_Use_Limit：使用限制包 ID（装备/道具限制）
	PauseTimeMS int // PauseTime：暂存窗（锁定技远距受理后的 skill_pause 时长）

	TaoLu            string // TaoLu：所属套路
	CoolDownCategory int    // CoolDownCategory：个人冷却类别（cooldown_rec col0）
	CoolDownTeam     int    // CoolDownTeam：冷却影响分组
	CoolEffectType   int    // CoolEffectType：冷却传递方式（决定公共冷却行形态）
	PublicCoolDownMS int    // AddCoolDownTime：附加（公共）冷却时长
	CoolDownMS       int    // CoolDownTime：个人冷却时长
	EffectType       int    // EffectType：0 无类别 / 1 实招 / 2 虚招 / 3 架招
	TargetType       int    // TargetType：位组合 1 地面 / 2 自己 / 4 选择对象 / 8 自己脚底地面
	TargetSelect     int
	DamageType       int
	IsDamage         int
	SkillType        int
	LaunchType       int // LauchType：0 主动 / 1 被动
	NoLimit          bool
	CanRush          bool // CanRush：冲锋技（远距受理 → 进带授予 c61）
	CanAirUse        bool
	HeightDiff       float64 // HeightDiff：攻击高度差上限
	HitCount         int     // HitCount：范围最大影响人数

	CastRange   float64 // 射程 = TargetShape.Para2（圆/扇形=半径，矩形=长）
	TargetShape Shape
	HitShape    Shape
	Consume     Consume
	Flow        Flow
	Lock        Lock
	LuaScript   string // LuaScript：逐技能脚本（特殊技能才挂；普通技能为空）
	// 伤害计算包 / 事件包 ID（命中结算阶段联查，本阶段不展开）。
	DamageCalculate int
	EventProcess    int
}

// NeedsTarget 是否需要选择对象目标（TargetType 含「选择对象」位且非自身可选）。
func (d Definition) NeedsTarget() bool { return d.TargetType&4 != 0 && d.TargetType&2 == 0 }

// Catalog 一份构建产物的只读目录。
type Catalog struct {
	dir      string
	tablesMu sync.Mutex // 护 tables（与 defsMu 分离：compile 持后者期间要经 table() 取前者）
	tables   map[string]*xmlTable
	classes  []Script // Skill_New 下实有的逻辑类
	defsMu   sync.Mutex
	defs     map[defKey]defEntry
}

type defKey struct {
	id    string
	level int
}

type defEntry struct {
	def Definition
	err error
}

// LoadDir 装载 ini/Skill 目录（不经进程级缓存）。
func LoadDir(dir string) (*Catalog, error) {
	static := filepath.Join(dir, fileStatic)
	info, err := res.Stat(static)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("skilldata: 构建产物缺席 %s（go run ./tools/datagen skill 构建）: %w", static, err)
	}
	entries, err := res.ReadDir(filepath.Join(dir, dirNew))
	if err != nil {
		return nil, fmt.Errorf("skilldata: 技能对象行目录缺席 %s: %w", filepath.Join(dir, dirNew), err)
	}
	var classes []Script
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		classes = append(classes, Script(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))))
	}
	if len(classes) == 0 {
		return nil, fmt.Errorf("skilldata: %s 下无逻辑类分档", filepath.Join(dir, dirNew))
	}
	return &Catalog{
		dir:     dir,
		tables:  map[string]*xmlTable{},
		classes: classes,
		defs:    map[defKey]defEntry{},
	}, nil
}

// catalogCache 进程级缓存：同一目录复用（索引与定义缓存只建一次）。
var catalogCache struct {
	mu    sync.Mutex
	byDir map[string]*Catalog
}

// Load 数据根（含 ini/Skill）装载，进程级缓存。
func Load(root string) (*Catalog, error) {
	dir := filepath.Join(root, "ini", "Skill")
	catalogCache.mu.Lock()
	defer catalogCache.mu.Unlock()
	if catalog := catalogCache.byDir[dir]; catalog != nil {
		return catalog, nil
	}
	catalog, err := LoadDir(dir)
	if err != nil {
		return nil, err
	}
	if catalogCache.byDir == nil {
		catalogCache.byDir = map[string]*Catalog{}
	}
	catalogCache.byDir[dir] = catalog
	return catalog, nil
}

// table 取（或懒建）一张产表；文件读取与建索引由 xmlTable 自身的 once 把关。
func (c *Catalog) table(rel string) *xmlTable {
	c.tablesMu.Lock()
	table := c.tables[rel]
	if table == nil {
		table = &xmlTable{}
		c.tables[rel] = table
	}
	c.tablesMu.Unlock()
	table.ensure(filepath.Join(c.dir, filepath.FromSlash(rel)))
	return table
}

// Definition 按 ConfigID + 等级取技能定义（等级 <1 按 1）。结果含错误一并缓存。
func (c *Catalog) Definition(id string, level int) (Definition, error) {
	if level < 1 {
		level = 1
	}
	key := defKey{id, level}
	c.defsMu.Lock()
	entry, cached := c.defs[key]
	c.defsMu.Unlock()
	if cached {
		return entry.def, entry.err
	}
	def, err := c.compile(id, level)
	entry = defEntry{def: def, err: err}
	c.defsMu.Lock()
	if _, raced := c.defs[key]; !raced {
		c.defs[key] = entry
	} else {
		entry = c.defs[key] // 并发首编以先到者为准
	}
	c.defsMu.Unlock()
	return entry.def, entry.err
}

// RushRange 冲锋范围常量（产物缺席全零）。
func (c *Catalog) RushRange() RushRange {
	config, ok := c.table(fileRushRange).lookup("config")
	if !ok {
		return RushRange{}
	}
	return RushRange{
		Ground: config.float("GRushRange", 0), Air: config.float("ARushRange", 0), Water: config.float("WRushRange", 0),
		GroundAdd: config.float("GRushRangeAdd", 0), AirAdd: config.float("ARushRangeAdd", 0), WaterAdd: config.float("WRushRangeAdd", 0),
	}
}

// compile 四层联查：技能对象行 → 静态包行 → 逐级可变属性行 → 引用包。
func (c *Catalog) compile(id string, level int) (def Definition, err error) {
	var object row
	var script Script
	for _, class := range c.classes {
		if candidate, ok := c.table(dirNew + "/" + string(class) + ".xml").lookup(id); ok {
			object, script = candidate, class
			break
		}
	}
	if object == nil {
		return def, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if script != ScriptNormal && script != ScriptLock {
		return def, fmt.Errorf("%w: %s（%s）", ErrUnsupported, id, script)
	}
	def.ID, def.Script, def.Level = id, script, level
	def.ItemType = object.int("ItemType", 0)
	def.QTypeID = object.int("QTypeID", 0)
	def.StaticData = object.int("StaticData", 0)
	def.PublicCoolDownMS = object.int("AddCoolDownTime", 0)
	def.PauseTimeMS = object.int("PauseTime", 0)
	def.UseLimit = object.int("Skill_Use_Limit", 0)
	if def.StaticData == 0 {
		return def, fmt.Errorf("skilldata: 技能 %s 缺 StaticData", id)
	}

	static, ok := c.table(fileStatic).lookup(strconv.Itoa(def.StaticData))
	if !ok {
		return def, fmt.Errorf("skilldata: 技能 %s 的静态包行 %d 缺失: %w", id, def.StaticData, ErrNotFound)
	}
	def.TaoLu = static.text("TaoLu")
	def.CoolDownCategory = static.int("CoolDownCategory", 0)
	def.CoolDownTeam = static.int("CoolDownTeam", 0)
	def.CoolEffectType = static.int("CoolEffectType", 0)
	def.EffectType = static.int("EffectType", 0)
	def.TargetType = static.int("TargetType", 0)
	def.TargetSelect = static.int("TargetSelect", 0)
	def.DamageType = static.int("DamageType", 0)
	def.IsDamage = static.int("IsDamage", 0)
	def.SkillType = static.int("SkillType", 0)
	def.LaunchType = static.int("LauchType", 0)
	def.NoLimit = static.flag("NoLimit")
	def.CanRush = static.flag("CanRush")
	def.CanAirUse = static.flag("CanAirUse")
	def.LuaScript = static.text("LuaScript")

	// 逐级可变属性行：静态行 Min..MaxVarPropNo 区间内取 Level 相符者。
	varPropNo, levels, ok := pickVarProp(c.table(dirVarProp+"/"+string(script)+".xml"),
		static.int("MinVarPropNo", 0), static.int("MaxVarPropNo", 0), level)
	if !ok {
		return def, fmt.Errorf("skilldata: 技能 %s 的逐级可变属性行缺席（StaticData=%d）", id, def.StaticData)
	}
	properties, _ := c.table(dirVarProp + "/" + string(script) + ".xml").lookup(varPropNo)
	def.VarPropNo, def.Levels = atoiDefault(varPropNo, 0), levels
	def.CoolDownMS = properties.int("CoolDownTime", 0)
	def.HeightDiff = properties.float("HeightDiff", 0)
	def.HitCount = properties.int("HitCount", 0)
	def.DamageCalculate = properties.int("DamageCalculate", 0)
	def.EventProcess = properties.int("EventProcess", 0)

	// 引用包（按数值 ID；包或行缺席按零值处理）。
	if pack, ok := c.table(fileConsume).lookup(properties.ref("Consume")); ok {
		def.Consume = Consume{
			HP: pack.int("AConsumeHP", 0), MP: pack.int("AConsumeMP", 0), SP: pack.int("AConsumeSP", 0),
			QingGong:  pack.int("AConsumeQGP", 0),
			PercentHP: pack.float("PConsumeHP", 0), PercentMP: pack.float("PConsumeMP", 0),
			ItemID: pack.text("ConsumeItemID"), ItemCount: pack.int("ConsumeItemCount", 0),
		}
	}
	def.TargetShape = c.shape(fileTarget, "TargetShape", properties.ref("TargetShapePkg"))
	def.CastRange = def.TargetShape.Para2
	def.HitShape = c.shape(fileHit, "HitShape", properties.ref("HitShapePkg"))
	if pack, ok := c.table(fileFlow).lookup(properties.ref("Flow")); ok {
		def.Flow = Flow{
			HitActions: pack.csv("HitAction"), HitTimeMS: pack.int("HitTime", 0),
			PrepareTimeMS: pack.int("PrepareTime", 0), LeadTimeMS: pack.int("LeadTime", 0),
			LeadSepTimeMS: pack.int("LeadSepTime", 0), LockBuffTimeMS: pack.int("LockBuffTime", 0),
			GRushAction: pack.text("GRushAction"), ARushAction: pack.text("ARushAction"),
			WRushAction: pack.text("WRushAction"), FailedAction: pack.text("FailedAction"),
		}
	}
	if lockType := static.int("LockType", 0); lockType != 0 {
		if pack, ok := c.table(fileLock).lookup(strconv.Itoa(lockType)); ok {
			def.Lock = Lock{
				OK:               true,
				Ratio:            pack.int("LockRatio", 0),
				SrcTime:          pack.int("LockSrcTime", 0),
				TargetTime:       pack.int("LockTargetTime", 0),
				SrcDist:          pack.float("LockSrcDist", 0),
				TargetDist:       pack.float("LockTargetDist", 0),
				SrcDodgeTime:     pack.int("SrcDodgeTime", 0),
				TargetDodgeTime:  pack.int("TargetDodgeTime", 0),
				SrcDodgeDist:     pack.float("SrcDodgeDist", 0),
				TargetDodgeDist:  pack.float("TargetDodgeDist", 0),
				SrcParryTime:     pack.int("SrcParryTime", 0),
				TargetParryTime:  pack.int("TargetParryTime", 0),
				SrcParryDist:     pack.float("SrcParryDist", 0),
				TargetParryDist:  pack.float("TargetParryDist", 0),
				HitAction:        pack.text("HitAction"),
				DodgeAction:      pack.text("DodgeAction"),
				ParryAction:      pack.text("ParryAction"),
				GroundRushAction: pack.text("LockGRushAction"),
				AirRushAction:    pack.text("LockARushAction"),
				WaterRushAction:  pack.text("LockWRushAction"),
				FailedAction:     pack.text("LockFailedAction"),
				BindDistance:     pack.float("BindDistance", 0),
			}
		}
	}
	return def, nil
}

// shape 取范围包行；prefix 为 TargetShape / HitShape（两表列名仅前缀不同）。
func (c *Catalog) shape(file, prefix, ref string) Shape {
	var shape Shape
	if ref == "" {
		return shape
	}
	pack, ok := c.table(file).lookup(ref)
	if !ok {
		return shape
	}
	return Shape{
		Kind:    pack.int(prefix, 0),
		Para1:   pack.float(prefix+"Para1", 0),
		Para2:   pack.float(prefix+"Para2", 0),
		Comment: pack.text("Comment"),
	}
}

// pickVarProp 在 [minNo, maxNo]（maxNo<=minNo 视为单级行）内找 Level 相符的
// 逐级行；没有相符等级回落区间内首个实有行。返回行 ID、实有行数、是否存在。
func pickVarProp(table *xmlTable, minNo, maxNo, level int) (id string, count int, ok bool) {
	if minNo <= 0 {
		return "", 0, false
	}
	if maxNo < minNo {
		maxNo = minNo
	}
	if maxNo-minNo > varPropSpanLimit {
		maxNo = minNo + varPropSpanLimit
	}
	first, match := "", ""
	for no := minNo; no <= maxNo; no++ {
		candidate, exists := table.lookup(strconv.Itoa(no))
		if !exists {
			continue
		}
		count++
		if first == "" {
			first = strconv.Itoa(no)
		}
		if match == "" && candidate.int("Level", 1) == level {
			match = strconv.Itoa(no)
		}
	}
	if match != "" {
		return match, count, true
	}
	return first, count, first != ""
}

func atoiDefault(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
