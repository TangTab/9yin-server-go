package main

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/clientdata"
)

const viewportSkill uint16 = 40

// View40 行官方补集属性索引（official_property_table.txt 玩家域，均唯一名）：
// UseLimit@2238/VarPropNo@2239/AConsumeHP@2245/AConsumeMP@2246/AConsumeSP@2252/
// AConsumeQGP@2254/TargetShapePara2@2260/HitShapePara1@2261/HitShapePara2@2262/
// HPHealMul@2265/MPHealMul@2266/Flow@2267/LeadTime@2270/LeadSepTime@2271（Int32）；
// Quality@2119(Byte)；CoolDownTime@2034(Int32)；五系 DamageMul@518..522(Float32)。
var (
	propSkillUseLimit    = clientdata.MustPlayerProperty("UseLimit").Index         // UseLimit（fork=2238）
	propVarPropNo        = clientdata.MustPlayerProperty("VarPropNo").Index        // VarPropNo（fork=2239）
	propAConsumeHP       = clientdata.MustPlayerProperty("AConsumeHP").Index       // AConsumeHP（fork=2245）
	propAConsumeMP       = clientdata.MustPlayerProperty("AConsumeMP").Index       // AConsumeMP（fork=2246）
	propAConsumeSP       = clientdata.MustPlayerProperty("AConsumeSP").Index       // AConsumeSP（fork=2252）
	propAConsumeQGP      = clientdata.MustPlayerProperty("AConsumeQGP").Index      // AConsumeQGP（fork=2254）
	propTargetShapePara2 = clientdata.MustPlayerProperty("TargetShapePara2").Index // TargetShapePara2（fork=2260）
	propHitShapePara1    = clientdata.MustPlayerProperty("HitShapePara1").Index    // HitShapePara1（fork=2261）
	propHitShapePara2    = clientdata.MustPlayerProperty("HitShapePara2").Index    // HitShapePara2（fork=2262）
	propHPHealMul        = clientdata.MustPlayerProperty("HPHealMul").Index        // HPHealMul（fork=2265）
	propMPHealMul        = clientdata.MustPlayerProperty("MPHealMul").Index        // MPHealMul（fork=2266）
	propFlow             = clientdata.MustPlayerProperty("Flow").Index             // Flow（fork=2267）
	propLeadTime         = clientdata.MustPlayerProperty("LeadTime").Index         // LeadTime（fork=2270）
	propLeadSepTime      = clientdata.MustPlayerProperty("LeadSepTime").Index      // LeadSepTime（fork=2271）
	propQuality          = clientdata.MustPlayerProperty("Quality").Index          // Quality（fork=2119）
	propCoolDownTime     = clientdata.MustPlayerProperty("CoolDownTime").Index     // CoolDownTime（fork=2034）
	propPhyDamageMul     = clientdata.MustPlayerProperty("PhyDamageMul").Index     // PhyDamageMul（fork=518）
	propStiDamageMul     = clientdata.MustPlayerProperty("StiDamageMul").Index     // StiDamageMul（fork=519）
	propJujDamageMul     = clientdata.MustPlayerProperty("JujDamageMul").Index     // JujDamageMul（fork=520）
	propNegDamageMul     = clientdata.MustPlayerProperty("NegDamageMul").Index     // NegDamageMul（fork=521）
	propMasDamageMul     = clientdata.MustPlayerProperty("MasDamageMul").Index     // MasDamageMul（fork=522）
)

// starterSkillRowIDs 标记 starter 基线行：这些行走 Java StarterSkills.skillRowProperties
// 方言（加发 LeadTime/LeadSepTime 固定族）；DB 追加行走 StarterGrants.dbSkillViewAdd
// 方言（Lead 系不发）。UseLimit 两种方言一致：仅 skill_new 真值 >0 稀疏发（官方
// 55866 对拍，Java 的 starter 回落 9 系误泛化，勿镜像）。
var starterSkillRowIDs = func() map[string]bool {
	ids := make(map[string]bool, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		ids[skill.configID] = true
	}
	return ids
}()

// learnedSkillView 是现代版 SkillContainer 使用的权威最小集。
// 套路本身没有单独的视图对象：客户端按 skill_static.ini 的
// TaoLu 字段对这些已学习的武学进行分组。
type learnedSkillView struct {
	configID   string
	staticData int32
	itemType   int32
	level      int32
	maxLevel   int32
	fill       int32
	total      int32
	pauseTime  float32
}

// starterSkillViews 是当前已学习的武学集合。前五件是默认分类
// 「基础武学 → 三脚猫招式」五件套（bug.txt：官服全角色默认已学，
// Java StarterSkills.DEFAULT_BASIC_SKILLS 对拍）：剑普攻 / 飞刀普攻 /
// 打坐调息 / 自断经脉 / 骑术精通。普攻族固定发剑+飞刀两件（官方观测
// 同款，不随门派变——其余武器普攻 zs_normal_ 与 hw_normal_ 族官方未授）。
// zs_default_01 的使用路径是特殊的原生 240 路线，而不是常规的 211 战斗
// 路线。其余条目是既有的 花神七式、心佛掌 七式与太极拳(古谱) TaoLu。
var starterSkillViews = []learnedSkillView{
	{configID: "zs_normal_02", staticData: 15, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "hw_normal_fd", staticData: 626, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "zs_default_01", staticData: 22, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "zs_suicide", staticData: 55, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "riding_start", staticData: 21515, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "CS_yhwq_hsqs01", staticData: 12636, itemType: 1000, level: 1, maxLevel: 11, total: 156550, pauseTime: 43 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs02", staticData: 12637, itemType: 1001, level: 1, maxLevel: 11, total: 156550, pauseTime: 65 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs03", staticData: 12638, itemType: 1000, level: 1, maxLevel: 11, total: 156550, pauseTime: 30 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs04", staticData: 12639, itemType: 1000, level: 1, maxLevel: 9, total: 156550, pauseTime: 62 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs05", staticData: 12640, itemType: 1000, level: 1, maxLevel: 4, total: 156550, pauseTime: 113 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs06", staticData: 12641, itemType: 1000, level: 1, maxLevel: 11, total: 156550, pauseTime: 35 * 1000.0 / 30},
	{configID: "CS_yhwq_hsqs07", staticData: 12642, itemType: 1001, level: 1, maxLevel: 11, total: 156550, pauseTime: 147 * 1000.0 / 30},
	// Heart Buddha Palm / 心佛掌。已安装的 skill_type.ini 将这个
	// TaoLu 精确定义为 CS_jh_xfz01..07，而 skill_new.ini/static_to_id.ini
	// 把每个 ConfigID 绑定到下方对应的 StaticData。招式 03/04/07 只有
	// 单条 varprop 行，因此显示等级上限为一级。
	{configID: "CS_jh_xfz01", staticData: 13189, itemType: 1001, level: 1, maxLevel: 12},
	{configID: "CS_jh_xfz02", staticData: 13190, itemType: 1000, level: 1, maxLevel: 12},
	{configID: "CS_jh_xfz03", staticData: 13205, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "CS_jh_xfz04", staticData: 13206, itemType: 1000, level: 1, maxLevel: 1},
	{configID: "CS_jh_xfz05", staticData: 13207, itemType: 1000, level: 1, maxLevel: 12},
	{configID: "CS_jh_xfz06", staticData: 13208, itemType: 1000, level: 1, maxLevel: 12},
	{configID: "CS_jh_xfz07", staticData: 13203, itemType: 1000, level: 1, maxLevel: 1},
	// 太极拳(古谱)。八个已安装的 SkillContainer 对象都归在
	// TaoLu=CS_wd_tjq 之下。MaxLevel 遵循各条 static 记录的精确
	// varprop 范围；PauseTime 遵循客户端动作时间线，单位为毫秒。
	{configID: "CS_wd_tjq01", staticData: 4057, itemType: 1000, level: 1, maxLevel: 19, pauseTime: 52 * 1000.0 / 30},
	{configID: "CS_wd_tjq02", staticData: 4058, itemType: 1001, level: 1, maxLevel: 19, pauseTime: 50 * 1000.0 / 30},
	{configID: "CS_wd_tjq03", staticData: 4238, itemType: 1000, level: 1, maxLevel: 19, pauseTime: 35 * 1000.0 / 30},
	{configID: "CS_wd_tjq04", staticData: 4045, itemType: 1000, level: 1, maxLevel: 19, pauseTime: 45 * 1000.0 / 30},
	{configID: "CS_wd_tjq05", staticData: 5451, itemType: 1000, level: 1, maxLevel: 19, pauseTime: 29 * 1000.0 / 30},
	{configID: "CS_wd_tjq06", staticData: 5452, itemType: 1000, level: 1, maxLevel: 19, pauseTime: 61 * 1000.0 / 30},
	{configID: "CS_wd_tjq07", staticData: 5453, itemType: 1000, level: 1, maxLevel: 3, pauseTime: 102 * 1000.0 / 30},
	{configID: "CS_wd_tjq08", staticData: 2107, itemType: 1000, level: 1, maxLevel: 19, pauseTime: (757.5 + 58*1000.0/30)},
}

// mergeLearnedSkills 合并 starter 基线与 DB 已学集（Java
// StarterSkills.withDefaultBasics 同口径的并集语义：老角色 DB 行缺
// 三脚猫五件套时由 starter 基线自动补齐，内存合成不落库，幂等可重放）。
// starter 在前、顺序与数值不变：心佛掌/太极拳快捷栏的行号锚定在 starter 顺序上；
// DB 行按 configID 去重追加，与 starter 重叠时 starter 条目优先。
func mergeLearnedSkills(dbRows []learnedSkillView) []learnedSkillView {
	if len(dbRows) == 0 {
		return starterSkillViews
	}
	seen := make(map[string]bool, len(starterSkillViews)+len(dbRows))
	for _, skill := range starterSkillViews {
		seen[skill.configID] = true
	}
	merged := append([]learnedSkillView(nil), starterSkillViews...)
	for _, row := range dbRows {
		if seen[row.configID] {
			continue
		}
		seen[row.configID] = true
		merged = append(merged, row)
	}
	return merged
}

// starterSkillViewFrames 是无 DB 行时的兜底路径（行为与历史版本一致）。
func starterSkillViewFrames() ([][]byte, error) {
	return learnedSkillViewFrames(starterSkillViews)
}

// learnedSkillViewFrames 在 stage_main 就绪屏障之后构建原生的
// 已学习武学容器。包含 StaticData 和 ItemType 是因为
// GoodsGrid 在解析本地图标和武学展示时会从
// 视图对象中读取它们。
func learnedSkillViewFrames(skills []learnedSkillView) ([][]byte, error) {
	frames := [][]byte{
		serverCreateView(serverViewSpec{ID: viewportSkill, Capacity: uint16(len(skills))}),
	}
	for index, skill := range skills {
		frame, err := serverViewAdd(viewportSkill, uint16(index+1), skillRowProperties(skill))
		if err != nil {
			return nil, fmt.Errorf("encode SkillContainer %s: %w", skill.configID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// skillRowProperties 构建单条 View40 行的属性集：最小集 + 官方补集（Java
// StarterSkills.skillRowProperties / StarterGrants.dbSkillViewAdd 对齐，登录
// merge 集与秘籍学技能 rebuild 共用本入口）。
//
// 补集语义（参考服 207 行 View40 对拍实证 2026-11-05）：VarPropNo/CoolDownTime
// 每行必发（CoolDownTime 即客户端技能 CD 文本源，缺 0 兜底显示 1s）；AConsume
// 族仅 >0 才发（怒招由此带出 AConsumeSP → 客户端显示怒气消耗并在怒气不足时灰
// 图标）；TargetShapePara2 即技能最大射程（快捷栏 tooltip 距离/施法限距读它）、
// HitShapePara1/2 地面圈指示，均仅 >0 才发；Quality/恢复与五系伤害倍率=1、
// Flow=5 固定族——缺席会让面板显示空/零。starter 行加发 LeadTime/LeadSepTime
// （Java starter 方言；官方 DB 行从不发）。UseLimit 仅 skill_new 真值 >0 才发：
// 官方 55866 抓包 207 行里 128 行稀疏携带、值 9 全包仅 zs_default_01（打坐）
// 1 次，太极拳/花神/心佛掌整段不发；Java 的「回落 9」是把打坐行误泛化——
// 下发 9 会让客户端按 skill_use_limit 规则 9「无法在战斗状态使用」在入战
// 瞬间把图标判红（服务端并无此闸，出现红图标但实际可出招的假象）。
func skillRowProperties(skill learnedSkillView) []serverViewProperty {
	starterRow := starterSkillRowIDs[skill.configID]
	var supplement *skillRowSupplement
	if catalog := installedCombatSkillCatalog; catalog != nil {
		supplement = catalog.skillRowData(skill.configID, skill.level)
	}
	var varPropNo, coolDownMs, useLimit int32
	if supplement != nil {
		varPropNo, coolDownMs, useLimit = supplement.varPropNo, supplement.coolDownMs, supplement.useLimit
	}
	properties := []serverViewProperty{
		// 视图行取官方表玩家域(末现):ConfigID@1440(String)、
		// Level@1465(Byte)、PauseTime@2240(Int32)。
		viewString(clientdata.MustPlayerProperty("ConfigID").Index, skill.configID),
		viewInt(propStaticData, skill.staticData),
		viewInt(propItemType, skill.itemType),
		viewByte(clientdata.MustPlayerProperty("Level").Index, byte(skill.level)),
		viewInt(propMaxLevel, skill.maxLevel),
		viewInt(propCurFillValue, skill.fill),
		viewInt(propTotalFillValue, skill.total),
		viewInt(propPauseTime, int32(skill.pauseTime)),
		viewByte(propSkillCanUse, 1),
		viewInt(propVarPropNo, varPropNo),
		viewByte(propQuality, 1),
		viewInt(propWuXing, skillWuXing(skill.configID)),
		viewInt(propCoolDownTime, coolDownMs),
		viewFloat(propHPHealMul, 1),
		viewFloat(propMPHealMul, 1),
		viewFloat(propPhyDamageMul, 1),
		viewFloat(propStiDamageMul, 1),
		viewFloat(propJujDamageMul, 1),
		viewFloat(propNegDamageMul, 1),
		viewFloat(propMasDamageMul, 1),
		viewInt(propFlow, 5),
	}
	if starterRow {
		properties = append(properties,
			viewInt(propLeadTime, 10000),
			viewInt(propLeadSepTime, 1500))
	}
	if useLimit > 0 {
		properties = append(properties, viewInt(propSkillUseLimit, useLimit))
	}
	if supplement == nil {
		return properties
	}
	if supplement.aConsumeHP > 0 {
		properties = append(properties, viewInt(propAConsumeHP, supplement.aConsumeHP))
	}
	if supplement.aConsumeMP > 0 {
		properties = append(properties, viewInt(propAConsumeMP, supplement.aConsumeMP))
	}
	if supplement.aConsumeSP > 0 {
		properties = append(properties, viewInt(propAConsumeSP, supplement.aConsumeSP))
	}
	if supplement.aConsumeQGP > 0 {
		properties = append(properties, viewInt(propAConsumeQGP, supplement.aConsumeQGP))
	}
	if supplement.targetShapePara2 > 0 {
		properties = append(properties, viewFloat(propTargetShapePara2, supplement.targetShapePara2))
	}
	if supplement.hitShapePara1 > 0 {
		properties = append(properties, viewFloat(propHitShapePara1, supplement.hitShapePara1))
	}
	if supplement.hitShapePara2 > 0 {
		properties = append(properties, viewFloat(propHitShapePara2, supplement.hitShapePara2))
	}
	return properties
}

// skillWuXing 是技能行五行/属性（官服 55866 抓包实证 View40 每行都带
// WuXing@2243）：wuxuebaseinfo.ini 目录值（刚1/柔2/阴3/阳4），缺行回退
// 5=太极——Java StarterSkills.skillWuXing 同款，客户端对 ≤0/>7 才退回
// ui_train_elemt 缺省文案。
func skillWuXing(configID string) int32 {
	if catalog, err := loadModernNeiGongCatalog(); err == nil {
		if value := catalog.wuXingOf(configID); value != 0 {
			return value
		}
	}
	return wuxingTaiJi
}
