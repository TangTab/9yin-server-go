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
	frames := [][]byte{mustCreateView(ViewportSkill, uint16(len(state.Skills)))}
	for index, skill := range state.Skills {
		frame, err := ViewAddFrame(ViewportSkill, uint16(index+1), skillViewRow(skill))
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
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
	frames := [][]byte{mustCreateView(ViewportNeiGong, uint16(len(state.NeiGong)))}
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
	frames := [][]byte{mustCreateView(ViewportQingGong, uint16(len(state.QingGongSkills)))}
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
// 40/43/44/45/46 数据视图——44/45 经脉穴位建号态为空容器）。
var emptyContainerViews = []struct {
	ID  uint16
	Cap uint16
}{
	{165, 6}, {128, 5}, {41, 20}, {47, 32}, {48, 32}, {44, 128}, {45, 32},
	{170, 8}, {1, 60}, {64, 32}, {174, 32}, {175, 32}, {176, 32}, {177, 32},
	{178, 32}, {179, 32}, {180, 32}, {181, 32}, {2, 128}, {159, 16}, {3, 128},
	{121, 32}, {122, 32}, {123, 32}, {124, 32}, {125, 32}, {126, 32},
}

// EmptyViewFrames 空容器视图族（7B 建视图帧）。
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
