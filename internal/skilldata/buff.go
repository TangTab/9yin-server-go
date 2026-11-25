package skilldata

// buff.go buff 定义的只读联查（Buff_New/<逻辑类>.xml → buff_static.xml →
// Buff_VarProp/Buffer.xml），与技能同模型：对象行（ID=ConfigID）→ 静态包行
// （ID=StaticData，Min/MaxVarPropNo 选逐级行）→ 逐级行。
//
// 关键列（官方 SKILL常用函数.txt / BUFF常用函数.txt 的说明）：
//
//	Category        类别（m169 buff 加删帧的末位 int）
//	EffectType      0 瞬时 / 1 X 秒内有效 / 2 X 秒内 Y 次事件有效 / 3 X 秒内每 Y 秒触发
//	LifeTimeType    0 只计在线时间 / 1 计全部时间（含下线）
//	LifeTime        生命期毫秒（逐级行；0 = 持久）
//	EffectTimer     计时间隔或计次次数（毫秒）
//	EffectRatio     每次触发成功率（百分比）
//	LuaScript       buff 钩子脚本（buffer_active/deactive/timer/break）

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// 产物文件名（相对 ini/Skill）。
const (
	fileBuffStatic = "buff_static.xml"
	dirBuffNew     = "Buff_New"
	fileBuffVar    = "Buff_VarProp/Buffer.xml"
)

// BuffDefinition 一份 buff 定义：对象行 + 静态包行 + 逐级行。
type BuffDefinition struct {
	ID         string // ConfigID
	Script     string // 逻辑类（现均为 Buffer）
	StaticData int
	Level      int

	Category         int    // 静态包 Category
	EffectType       int    // 作用类型
	LifeTimeType     int    // 生命期类型
	Visible          bool   // 客户端可见
	IsDamage         bool   // 效果显示为伤害
	Immunity         int    // 免疫类别
	CantCustomRemove bool   // 不可由脚本移除
	OffLineClear     bool   // 下线清除
	DeadClear        bool   // 死亡清除
	EndFightClear    bool   // 脱战清除
	LeaveSceneClear  bool   // 离场清除
	EventProcess     int    // 事件包 ID
	LuaScript        string // 钩子脚本路径

	LifeTimeMS    int // 生命期（毫秒，0 = 持久）
	EffectTimerMS int // 触发间隔/次数
	EffectRatio   int // 成功率（百分比）
	OverlayCount  int // 叠加上限
	HitCount      int
	EffectRange   int
}

type buffKey struct {
	id    string
	level int
}

type buffEntry struct {
	def BuffDefinition
	err error
}

// Buff 按 ConfigID + 等级取 buff 定义（等级 <1 按 1）。结果含错误一并缓存。
func (c *Catalog) Buff(id string, level int) (BuffDefinition, error) {
	if level < 1 {
		level = 1
	}
	key := buffKey{id, level}
	c.buffsMu.Lock()
	entry, cached := c.buffs[key]
	c.buffsMu.Unlock()
	if cached {
		return entry.def, entry.err
	}
	def, err := c.compileBuff(id, level)
	entry = buffEntry{def, err}
	c.buffsMu.Lock()
	if _, raced := c.buffs[key]; !raced {
		c.buffs[key] = entry
	} else {
		entry = c.buffs[key]
	}
	c.buffsMu.Unlock()
	return entry.def, entry.err
}

// BuffExists buff ConfigID 是否在对象行里（技能的 buf_<技能> 关联 buff 判定）。
func (c *Catalog) BuffExists(id string) bool {
	_, err := c.Buff(id, 1)
	return err == nil
}

// compileBuff 对象行 → 静态包行 → 逐级行。
func (c *Catalog) compileBuff(id string, level int) (def BuffDefinition, err error) {
	var object row
	// Buff_New 按逻辑类分档（现只有 Buffer 一类；缺省行归 Unknown）。
	entries, derr := res.ReadDir(filepath.Join(c.dir, filepath.FromSlash(dirBuffNew)))
	if derr == nil {
		for _, entry := range entries {
			if candidate, ok := c.table(dirBuffNew + "/" + entry.Name()).lookup(id); ok {
				object = candidate
				def.Script = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				break
			}
		}
	}
	if object == nil {
		return def, fmt.Errorf("%w: buff %s", ErrNotFound, id)
	}
	def.ID, def.Level = id, level
	def.StaticData = object.int("StaticData", 0)
	if def.StaticData == 0 {
		return def, fmt.Errorf("skilldata: buff %s 缺 StaticData", id)
	}
	static, ok := c.table(fileBuffStatic).lookup(strconv.Itoa(def.StaticData))
	if !ok {
		return def, fmt.Errorf("skilldata: buff %s 的静态包行 %d 缺失: %w", id, def.StaticData, ErrNotFound)
	}
	def.Category = static.int("Category", 0)
	def.EffectType = static.int("EffectType", 0)
	def.LifeTimeType = static.int("LifeTimeType", 0)
	def.Visible = static.flag("Visible")
	def.IsDamage = static.flag("IsDamage")
	def.Immunity = static.int("Immunity", 0)
	def.CantCustomRemove = static.flag("CantCustomRemove")
	def.OffLineClear = static.flag("OffLineClear")
	def.DeadClear = static.flag("DeadClear")
	def.EndFightClear = static.flag("EndFightClear")
	def.LeaveSceneClear = static.flag("LeaveSceneClear")
	def.EventProcess = static.int("EventProcess", 0)
	def.LuaScript = static.text("LuaScript")

	// 逐级行：静态行 Min..MaxVarPropNo 内取 Level 相符者。
	if varPropNo, _, ok := pickVarProp(c.table(fileBuffVar),
		static.int("MinVarPropNo", 0), static.int("MaxVarPropNo", 0), level); ok {
		properties, _ := c.table(fileBuffVar).lookup(varPropNo)
		def.LifeTimeMS = properties.int("LifeTime", 0)
		def.EffectTimerMS = properties.int("EffectTimer", 0)
		def.EffectRatio = properties.int("EffectRatio", 100)
		def.OverlayCount = properties.int("OverlayCount", 0)
		def.HitCount = properties.int("HitCount", 0)
		def.EffectRange = properties.int("EffectRange", 0)
	}
	// 对象行自带的值优先（buf_Relive_2 等把生命期直接写在对象行）。
	if life := object.int("LifeTime", 0); life > 0 {
		def.LifeTimeMS = life
	}
	if timer := object.int("EffectTimer", 0); timer > 0 {
		def.EffectTimerMS = timer
	}
	if ratio := object.int("EffectRatio", 0); ratio > 0 {
		def.EffectRatio = ratio
	}
	return def, nil
}
