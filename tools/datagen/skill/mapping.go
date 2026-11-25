// mapping.go 源表规格与审定映射（本工具的核心）。
//
// 技能源表是服务端设计态表，列名即服务端词汇，缺省原样直通；只有人工
// 核对过的绑定才改名/丢弃（注释给 review 依据），没把握的保持 native
// 直通并在报告里标出处，不发明。
package skill

// tableSpec 一张源表的转换规格。
type tableSpec struct {
	Source string   // 相对输入根的源文件（-in；Rule=true 时相对 -rule）
	Rule   bool     // 源在 -rule 根
	Output string   // 输出相对 <out> 的路径；%s = 分组键
	Group  string   // 分组键（"script"=按逻辑类分档；空=单文件）
	Title  string   // 报告用中文名
	Lead   []string // 前置列序（ID 恒最前）
	// Headers 表头元数据行的节名（[-1]=列序头、[0]=列宽头）：装载器
	// 元数据非数据，丢弃并入报告。
	Headers []string
	Rename  map[string]string // 改名映射（源列 → 输出属性）
	Drop    map[string]string // 丢弃列 → 理由
	// Fixups 审定修正（节名 → 键的输出值）：源表脏数据在构建期按审定值改写
	// 并入报告。产物手改会被下次重建覆盖，手修定版一律落在这里。
	Fixups map[string]valueFix
	Note   string // 审定说明
}

// valueFix 一处审定修正。
type valueFix struct {
	Key    string // 输出属性名
	Value  string // 审定值
	Reason string // 依据
}

// specs 全部源表规格（顺序即构建序）。
var specs = []tableSpec{
	{
		Source: "skill_new.ini", Output: "Skill_New/%s.xml", Group: "script",
		Title: "技能对象行（skill_new）",
		Lead:  []string{"script", "StaticData", "ItemType", "QTypeID"},
		Rename: map[string]string{
			"UseLimit": "Skill_Use_Limit",
		},
		Note: "按 script 逻辑类分档（服务端对象类，对应黑金按玩法域分文件）；UseLimit→Skill_Use_Limit 取 struct/skill/skillbase.xml 属性名（除消耗限制外的使用限制包，引用 Skill_UseLimit）",
	},
	{
		// 输出名取已跟踪路径的原名（小写）：git 索引里就是 skill_static.xml，
		// 换大小写命名会被当作该路径的内容修改而非新增，日后在区分大小写的
		// 系统上检出仍是小写名——产物名必须与之逐字符一致。
		Source: "skill_static.ini", Output: "skill_static.xml",
		Title:   "静态包行（skill_static，ID=StaticData）",
		Lead:    []string{"TaoLu", "CoolDownCategory", "CoolDownTeam", "CoolEffectType"},
		Headers: []string{"-1", "0"},
		Drop: map[string]string{
			"Photo": "客户端技能图标路径（显示参数），服务端不消费",
		},
		Note: "列名＝lua/share/skill_static_data_define.lua 的 Skill_* 静态包列名与黑金 skill_static.xml 样本属性名",
	},
	{
		Source: "skill_normal_varprop.ini", Output: "Skill_VarProp/SkillNormal.xml",
		Title: "逐级可变属性行（SkillNormal，ID=VarPropNo）",
		Lead:  []string{"Level", "CoolDownTime", "Consume", "DamageCalculate", "TargetShapePkg", "HitShapePkg", "Flow"},
		Note:  "以 Consume/DamageCalculate/TargetShapePkg/HitShapePkg/Flow/EventProcess 数值 ID 引用各包（引用式解耦，运行时联查）",
	},
	{
		Source: "skill_lock_varprop.ini", Output: "Skill_VarProp/SkillLock.xml",
		Title: "逐级可变属性行（SkillLock，ID=VarPropNo）",
		Lead:  []string{"Level", "CoolDownTime", "Consume", "DamageCalculate", "TargetShapePkg", "HitShapePkg"},
		Note:  "与 SkillNormal 同构，ID 空间独立（静态行 Min/MaxVarPropNo 按逻辑类选表）",
	},
	{Source: "skill_consume.ini", Output: "Skill_Consume.xml", Title: "消耗包（skill_consume）",
		Headers: []string{"-1", "0"}, Note: "列名＝struct/rule/fight/consume.xml 属性名"},
	{Source: "attack_targetshape.ini", Output: "Skill_TargetShape.xml", Title: "射程范围包（attack_targetshape）",
		Headers: []string{"-1", "0"}, Note: "列名＝struct/rule/fight/range_new.xml 属性名；Comment 为设计注释原样保留"},
	{Source: "attack_hitshape.ini", Output: "Skill_HitShape.xml", Title: "作用范围包（attack_hitshape）",
		Headers: []string{"-1", "0"}},
	{Source: "skill_lock_data.ini", Output: "Skill_LockData.xml", Title: "锁定包（skill_lock_data，ID=LockType）",
		Headers: []string{"-1", "0"}, Lead: []string{"LockRatio", "LockSrcTime", "LockTargetTime", "LockSrcDist", "LockTargetDist"}},
	{Source: "skill_use_limit.ini", Output: "Skill_UseLimit.xml", Title: "使用限制包（skill_use_limit，ID=skill_new UseLimit）",
		Note: "源表每节可含两个 r= 条件行，重复键输出为 r/r_2"},
	{Source: "flow_config.ini", Rule: true, Output: "Skill_Flow.xml", Title: "流程包（rule/flow_config，ID=Flow）",
		Headers: []string{"-1", "0"}, Lead: []string{"HitAction", "HitTime", "PrepareTime", "LockBuffTime"},
		Note: "列名＝struct/rule/fight/flow_new.xml 属性名"},
	{Source: "rush_range.ini", Output: "Skill_RushRange.xml", Title: "冲锋范围常量（rush_range [config]）"},

	// ---- 命中结算三件套（2026-11-25 战斗接入）：伤害计算包 → 公式 → buff 族 ----
	//
	// 与技能对象行同构的引用链：逐级行 DamageCalculate（数值 ID）→
	// Skill_DamageCalculate 行的 r= 列表（公式 ID,伤害类型,参数…，节内多行
	// 输出为 r/r_2…）→ Skill_Formula 行（Src1..12/Prop1..12 参数来源 +
	// Formula 表达式）。buff 与技能同模型：buff_new 对象行 → buff_static 静态
	// 包行（ID=StaticData，Min/MaxVarPropNo 选逐级行）→ Buff_VarProp 逐级行。
	{Source: "damage_calculate.ini", Output: "Skill_DamageCalculate.xml",
		Title: "伤害计算包（damage_calculate，ID=DamageCalculate）",
		Note:  "每节一条或多条 r=（公式ID,伤害类型,参数…），多条输出为 r/r_2…；服务端按序取全部 r 列"},
	{Source: "formula.ini", Output: "Skill_Formula.xml", Title: "伤害公式（formula，ID=公式号）",
		Headers: []string{"-1", "0"},
		Lead:    []string{"Formula", "Src1", "Prop1"},
		Fixups: map[string]valueFix{
			"1116": {Key: "Formula", Value: "(P1+((P2+P3+P4+P5)*0.065))",
				Reason: "源值多一个最外层左括号（括号不配平，公式引擎装载期编译失败）；去掉后语义不变。2026-11-25 手修定版"},
		},
		Note: "Src/Prop 成对：来源类别 + 属性名；Formula 为 P1..P12/R[a,b]/MIN[..]/MAX[..] 表达式；Comment/TipFormat* 为设计注释与客户端提示串，原样保留"},
	{Source: "buff_new.ini", Output: "Buff_New/%s.xml", Group: "script",
		Title: "buff 对象行（buff_new）",
		Lead:  []string{"script", "StaticData"},
		Note:  "按 script 逻辑类分档（同技能对象行；现为 Buffer 一类）"},
	{Source: "buff_static.ini", Output: "buff_static.xml",
		Title:   "buff 静态包行（buff_static，ID=StaticData）",
		Headers: []string{"-1", "0"},
		Lead:    []string{"Category", "EffectType", "LifeTimeType", "EventProcess", "LuaScript", "MinVarPropNo", "MaxVarPropNo"},
		Drop: map[string]string{
			"Photo": "客户端 buff 图标路径（显示参数），服务端不消费",
		},
		Note: "输出名与 skill_static.xml 同取小写；LuaScript 为 buff 钩子脚本（buffer_active/deactive/timer/break）"},
	{Source: "skill_static_to_id.ini", Output: "Skill_StaticToId.xml",
		Title: "静态包号 → 技能 ID（skill_static_to_id，ID=StaticData）",
		Note:  "NPC 战斗行的技能列（SkillID1..8）记静态包号，经此表得 ConfigID 再联查技能定义"},
	{Source: "buff_varprop.ini", Output: "Buff_VarProp/Buffer.xml",
		Title: "buff 逐级行（buff_varprop，ID=VarPropNo）",
		Lead:  []string{"Level", "LifeTime", "EffectTimer", "EffectRatio", "EventProcess"},
		Note:  "buff_new 现只有 Buffer 一个逻辑类，逐级行单档；LifeTime/EffectTimer 毫秒"},
}

// vocabKind 输出列的词汇出处分类。
const (
	kindDirect  = "direct"  // struct 服务端 schema 属性名
	kindHeijin  = "heijin"  // 黑金 ini/Skill 样本属性名
	kindDefine  = "define"  // lua 静态包列名
	kindNative  = "native"  // 仅见于九阴源表的服务端原生列
	kindRename  = "rename"  // 审定改名
	kindDropped = "dropped" // 审定丢弃
)

// vocabulary 三处真档的属性名并集（出处判定用）。
type vocabulary struct {
	structProps map[string]bool
	heijinAttrs map[string]bool
	defineNames map[string]bool
}

// classify 判定一个输出属性的出处类别。
func (v *vocabulary) classify(attr string) string {
	switch {
	case v.structProps[attr]:
		return kindDirect
	case v.heijinAttrs[attr]:
		return kindHeijin
	case v.defineNames[attr]:
		return kindDefine
	}
	return kindNative
}

// validAttrName XML 属性名合法性（NameStartChar 限定 ASCII 字母/下划线，
// 后续允许字母数字与 _ . -；源表畸形键不得进输出）。
func validAttrName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_':
		case i > 0 && (r >= '0' && r <= '9' || r == '.' || r == '-'):
		default:
			return false
		}
	}
	return true
}
