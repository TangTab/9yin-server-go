package main

import "fmt"

const viewportSkill uint16 = 40

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

// starterSkillViews 是当前已学习的武学集合。zs_default_01 是已安装的
// 现代版原生 打坐 前置（StaticData=22，ItemType=1000，
// 最高等级=1）；它的使用路径是特殊的原生 240 路线，而不是常规的
// 211 战斗路线。其余条目是既有的 花神七式 以及
// 已安装的 心佛掌 七式 TaoLu。
var starterSkillViews = []learnedSkillView{
	{configID: "zs_default_01", staticData: 22, itemType: 1000, level: 1, maxLevel: 1},
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

// starterSkillViewFrames 在 stage_main 就绪屏障之后构建原生的
// 已学习武学容器。包含 StaticData 和 ItemType 是因为
// GoodsGrid 在解析本地图标和武学展示时会从
// 视图对象中读取它们。
func starterSkillViewFrames() ([][]byte, error) {
	frames := [][]byte{
		serverCreateView(serverViewSpec{ID: viewportSkill, Capacity: uint16(len(starterSkillViews))}),
	}
	for index, skill := range starterSkillViews {
		frame, err := serverViewAdd(viewportSkill, uint16(index+1), []serverViewProperty{
			// 视图行取官方表玩家域(末现):ConfigID@1440(String)、
			// Level@1465(Byte)、PauseTime@2240(Int32)。
			viewString(1440, skill.configID),
			viewInt(propStaticData, skill.staticData),
			viewInt(propItemType, skill.itemType),
			viewByte(1465, byte(skill.level)),
			viewInt(propMaxLevel, skill.maxLevel),
			viewInt(propCurFillValue, skill.fill),
			viewInt(propTotalFillValue, skill.total),
			viewInt(propPauseTime, int32(skill.pauseTime)),
			viewByte(propSkillCanUse, 1),
		})
		if err != nil {
			return nil, fmt.Errorf("encode SkillContainer %s: %w", skill.configID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
