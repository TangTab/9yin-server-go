package entergen

// 视图簇生成器：View40 武学 / View43 内功 / View46 轻功容器 / 空容器族。
// 行属性集对齐 b-line skillRowProperties / bookFrame / grantStarterQingGong
// （官方 c.pcapng 逐位走查定版；idx 全部按 2433 表名解析）。

import (
	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 视图编号（b-line viewport 常量同源）。
const (
	ViewportSkill    = 40 // 武学技能容器
	ViewportNeiGong  = 43 // 内功书容器
	ViewportXueWei   = 44 // 穴位
	ViewportJingMai  = 45 // 经脉
	ViewportQingGong = 46 // 轻功容器
)

// 三数据视图的建视图容量（官服登录 0x15 帧实测：资料\初始账号登陆-完整
// 引导-女.pcapng 24.05s，`15 28 00 20 03` / `15 2b 00 00 02` / `15 2e 00
// 00 02`）。是容器上限而非登录行数——武学书使用（msg31）学会新招式要在
// 登录行集之后追加行，按行数建容量会越界。
const (
	skillViewCapacity    = 800
	neiGongViewCapacity  = 512
	qingGongViewCapacity = 512
)

func playerIdx(name string) uint16 { return clientdata.MustPlayerProperty(name).Index }

// skillViewRow View40 单行属性集（b-line skillRowProperties；VarPropNo/
// CD/消耗/射程无资源目录时置零，Quality/倍率/Flow 固定族）。
func skillViewRow(skill rolestate.StarterSkill) []ViewProperty {
	return []ViewProperty{
		{playerIdx("ConfigID"), StringValue(skill.ConfigID)},
		{playerIdx("StaticData"), Int32Value(int32(skill.StaticData))},
		{playerIdx("ItemType"), Int32Value(int32(skill.ItemType))},
		{playerIdx("Level"), ByteValue(byte(clampU8(skill.Level)))},
		{playerIdx("MaxLevel"), Int32Value(int32(skill.MaxLevel))},
		{playerIdx("CurFillValue"), Int32Value(0)},
		{playerIdx("TotalFillValue"), Int32Value(int32(skill.Total))},
		{playerIdx("PauseTime"), Int32Value(int32(skill.PauseTime))},
		{playerIdx("CanUse"), ByteValue(1)},
		{playerIdx("VarPropNo"), Int32Value(0)},
		{playerIdx("Quality"), ByteValue(1)},
		{playerIdx("WuXing"), Int32Value(0)},
		{playerIdx("CoolDownTime"), Int32Value(0)},
		{playerIdx("HPHealMul"), Float32Value(1)},
		{playerIdx("MPHealMul"), Float32Value(1)},
		{playerIdx("PhyDamageMul"), Float32Value(1)},
		{playerIdx("StiDamageMul"), Float32Value(1)},
		{playerIdx("JujDamageMul"), Float32Value(1)},
		{playerIdx("NegDamageMul"), Float32Value(1)},
		{playerIdx("MasDamageMul"), Float32Value(1)},
		{playerIdx("Flow"), Int32Value(5)},
	}
}

// SkillViewFrames View40：建视图 + 每技能一行。
func SkillViewFrames(state rolestate.RoleState) ([][]byte, error) {
	frames := [][]byte{mustCreateView(ViewportSkill, skillViewCapacity)}
	for index, skill := range state.Skills {
		frame, err := SkillRowFrame(index, skill)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// SkillRowFrame View40 追加一行（行号 = 在 roleState.Skills 里的 0 基
// 下标 + 1；武学书使用学会新招式时在登录行集之后追加——官服抓包 m31 #1
// `0x18 view=40 idx=8` 同形）。
func SkillRowFrame(index int, skill rolestate.StarterSkill) ([]byte, error) {
	return ViewAddFrame(ViewportSkill, uint16(index+1), skillViewRow(skill))
}

// bookViewRow View43 书行（b-line bookFrame：ConfigID/ItemType/Level/
// StaticData/MaxLevel/进度族/Power 族 + varprop 档五维 Add/buff）。
func bookViewRow(book rolestate.NeiGongBook) []ViewProperty {
	props := []ViewProperty{
		{playerIdx("ConfigID"), StringValue(book.ConfigID)},
		{playerIdx("ItemType"), Int32Value(int32(book.ItemType))},
		{playerIdx("Level"), ByteValue(byte(clampU8(book.Level)))},
		{playerIdx("StaticData"), Int32Value(int32(book.StaticData))},
		{playerIdx("MaxLevel"), Int32Value(int32(book.MaxLevel))},
		{playerIdx("CurFillValue"), Int32Value(0)},
		{playerIdx("TotalFillValue"), Int32Value(int32(book.Total))},
		{playerIdx("PowerValue"), Int32Value(int32(book.Power))},
		{playerIdx("MaxPowerValue"), Int32Value(int32(book.MaxPower))},
		{playerIdx("WuXing"), Int32Value(0)},
	}
	// varprop 档族（StarterSet 建号档已带五维 Add/buff——无目录时的
	// 官方形状来源）。
	school := book.ConfigID
	for i := len(school) - 1; i >= 0; i-- {
		if school[i] == '_' {
			school = school[:i]
			break
		}
	}
	props = append(props,
		ViewProperty{playerIdx("School"), StringValue(school)},
		ViewProperty{playerIdx("VarPropNo"), Int32Value(0)},
		ViewProperty{playerIdx("NeiGongLevel"), Int32Value(int32(book.Level))},
		ViewProperty{playerIdx("StrAdd"), Int32Value(int32(book.StrAdd))},
		ViewProperty{playerIdx("StaAdd"), Int32Value(int32(book.StaAdd))},
		ViewProperty{playerIdx("DexAdd"), Int32Value(int32(book.DexAdd))},
		ViewProperty{playerIdx("IngAdd"), Int32Value(int32(book.IngAdd))},
		ViewProperty{playerIdx("SpiAdd"), Int32Value(int32(book.SpiAdd))},
	)
	if book.BuffID != "" {
		props = append(props,
			ViewProperty{playerIdx("BufferID"), StringValue(book.BuffID)},
			ViewProperty{playerIdx("BufferLevel"), Int32Value(int32(book.BuffLevel))},
		)
	}
	props = append(props, ViewProperty{playerIdx("Quality"), ByteValue(1)})
	return props
}

// NeiGongViewFrames View43：建视图 + 每书一行。
func NeiGongViewFrames(state rolestate.RoleState) ([][]byte, error) {
	frames := [][]byte{mustCreateView(ViewportNeiGong, neiGongViewCapacity)}
	for index, book := range state.NeiGong {
		frame, err := ViewAddFrame(ViewportNeiGong, uint16(index+1), bookViewRow(book))
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// QingGongViewFrames View46 轻功容器（b-line grantStarterQingGong：
// 建视图 + 每实例一行 ConfigID@场景域 idx7 + IsActive）。
func QingGongViewFrames(state rolestate.RoleState) ([][]byte, error) {
	configEntry, ok := clientdata.SceneProperty("ConfigID")
	if !ok {
		configEntry = clientdata.MustPlayerProperty("ConfigID")
	}
	activeEntry := clientdata.MustPlayerProperty("IsActive")
	frames := [][]byte{mustCreateView(ViewportQingGong, qingGongViewCapacity)}
	for index, id := range state.QingGongSkills {
		frame, err := ViewAddFrame(ViewportQingGong, uint16(index+1), []ViewProperty{
			{configEntry.Index, StringValue(id)},
			{activeEntry.Index, ByteValue(1)},
		})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// emptyContainerViews 官方登录簇的空容器视图（初始账号 30 视图集减
// 40/43/44/45/46 数据视图与背包视图族——44/45 经脉穴位建号态为空容器，
// 背包族见 BagViewCreateFrames）。
var emptyContainerViews = []struct {
	ID  uint16
	Cap uint16
}{
	{165, 6}, {128, 5}, {41, 20}, {47, 32}, {48, 32}, {44, 128}, {45, 32},
	{170, 8}, {64, 32}, {159, 16},
}

// 背包视图族容量（官服初始账号登陆-完整引导-女.pcapng 8032 线
// #1232/#1347..#1381 定版）：8 主页（4 分类页 + 4 时装镜像页）cap=90 且
// 携 BaseCap=18——客户端按基础容量渲染页格行数（3 行×6 格），缺发该
// 属性时除装备页外其余页行数萎缩（2026-11-27 真机：装备 3 行、其余 1 行）；
// 8 扩展页 cap=2；纸娃娃 View1 cap=52。
const (
	bagMainViewCapacity = 90
	bagExtViewCapacity  = 2
	bagViewBaseCap      = 18
	dollViewCapacity    = 52
)

// bagMainViews / bagExtViews 背包主页与扩展页（官服建视图序：纸娃娃 →
// 镜像族 → 分类页 → 扩展页；扩展页 3/122/124/126 与镜像扩展页
// 175/177/179/181 是扩容锚点行容器，见 2433 线 bagExtPages）。
var (
	bagMainViews = []uint16{174, 176, 178, 180, 2, 121, 123, 125}
	bagExtViews  = []uint16{175, 177, 179, 181, 3, 122, 124, 126}
)

// BagViewCreateFrames 背包视图族建视图帧（17 视图；行推送前建好——
// 官服登录簇序）。
func BagViewCreateFrames() ([][]byte, error) {
	frames := make([][]byte, 0, 1+len(bagMainViews)+len(bagExtViews))
	doll, err := CreateViewFrame(equipDollView, dollViewCapacity, nil)
	if err != nil {
		return nil, err
	}
	frames = append(frames, doll)
	baseCap := []ViewProperty{{playerIdx("BaseCap"), Int32Value(bagViewBaseCap)}}
	for _, viewID := range bagMainViews {
		frame, err := CreateViewFrame(viewID, bagMainViewCapacity, baseCap)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	for _, viewID := range bagExtViews {
		frame, err := CreateViewFrame(viewID, bagExtViewCapacity, nil)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// EmptyViewFrames 空容器视图族（7B 建视图帧；背包族之外）。
func EmptyViewFrames() ([][]byte, error) {
	frames := make([][]byte, 0, len(emptyContainerViews))
	for _, v := range emptyContainerViews {
		frame, err := CreateViewFrame(v.ID, v.Cap, nil)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func mustCreateView(viewID, capacity uint16) []byte {
	frame, err := CreateViewFrame(viewID, capacity, nil)
	if err != nil {
		panic(err)
	}
	return frame
}
