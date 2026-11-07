package main

import (
	"fmt"
	"math"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// GM 面板域（货币/属性/状态批次）的玩家侧状态与帧构建器。参考 Java
// LoginFlow applyCapital/applyPowerLevel/applyAlignment/applySetPkMode/
// applyVitals/applyFillSpeed/applyActDayLimit 同源；线缆事实（索引/线型/
// 帧序）以 GmWeatherPresets 同期的 official_property_table 玩家域为准。
const (
	propPowerLevel     uint16 = 210 // PowerLevel Int32 实力等级
	propLevelTitle     uint16 = 801 // LevelTitle String 实力称号键
	propCharacterFlag  uint16 = 319 // CharacterFlag Int32 性格标记
	propCharacterValue uint16 = 320 // CharacterValue Int32 性格值
	propJustice        uint16 = 321 // Justice Int32 善值
	propEvil           uint16 = 322 // Evil Int32 恶值
	propFreeBase       uint16 = 323 // FreeBase Int32 自由善恶基数
	propPKValue        uint16 = 427 // PKValue Int32 杀戮值
	propPKMode         uint16 = 437 // PKMode Byte PK 模式（官方线型 1）
	propEne            uint16 = 705 // Ene Int32 体力
	propMaxEne         uint16 = 706 // MaxEne Int32 体力上限
	propResumeEne      uint16 = 707 // ResumeEne Int32 体力恢复
	propSat            uint16 = 708 // Sat Int32 饱食
	propMaxSat         uint16 = 709 // MaxSat Int32 饱食上限
	propDrunkLevel     uint16 = 723 // DrunkLevel Int32 酒量
	propMount          uint16 = 77  // Mount String 当前坐骑 configId（空=无）
	// propTitleID@360 归 title_system.go 声明（称号域独占），此处共用。
	propCurRideMount    uint16 = 361  // CurRideMount Object 骑乘态坐骑对象（无=0）
	propPrestigeExp     uint16 = 848  // PrestigeExp Int32 声望总量
	propBingLuType      uint16 = 2153 // BingLuType Int32 兵录槽类型（无数据源发 0）
	propMaxBingLuLevel  uint16 = 2154 // MaxBingLuLevel Int32 兵录最高等级
	propNorFacultyAdd   uint16 = 774  // NorFacultyAdd Int32 修炼速度基数
	propActDayValue     uint16 = 790  // ActDayValue Int32 演武每日额度（文）
	propActCostSuiYin   uint16 = 791  // ActCostSuiYin Int32 今日已耗碎银（文）
	propActDayValueLast uint16 = 795  // ActDayValueLast Int32 剩余总量（文）
	propActDayValueUsed uint16 = 796  // ActDayValueUsed Int32 今日已用（文）
	propActDayValueMax  uint16 = 797  // ActDayValueMax Int32 每日上限（文，0=不限制）
)

// actDayUnlimitedSentinel 与 Java PlayerProgress.UNLIMITED_ACT_DAY_VALUE
// 一致：不限模式给 999999999 占位，客户端演武页显示“不限制”。
const actDayUnlimitedSentinel int32 = 999_999_999

// gmBaseAttributeCap 与 Java PlayerProgress.BASE_ATTRIBUTE_CAP 一致。
const gmBaseAttributeCap int32 = 1_000_000

// gmVitalsState 是 GM set_vitals 的生存属性层（Java PlayerProgress.Vitals
// 的 GM 可写子集；max_hp_override/max_mp_override 仅快照透传）。
type gmVitalsState struct {
	ene, maxEne, resumeEne, sat, maxSat, pkValue, drunkLevel int32
}

// gmPanelDomain 是 GM 面板域的服务器权威会话状态（Java PanelState 对应
// 物）。会话级：重登回种子（实力 151 与选人界面同档），GM set 后会话内
// 生效，与 Java set_pk_mode/power_level 口径一致；仅 vitals 经
// role_faculty 快照跨登录保留。零值即合法（Java 身份块缺发则面板整块
// 空白，因此零也要下发而非省略帧）。
type gmPanelDomain struct {
	powerLevel int32 // 种子 roleListPowerLevel（151=title027）
	justice    int32
	evil       int32
	freeBase   int32
	pkMode     int32
	vitalsSet  bool
	vitals     gmVitalsState
}

// setGuanYin 与 setSilver 同款：写会话状态并返回单属性 0x10。
func (p *playerActor) setGuanYin(value int32) {
	p.mu.Lock()
	p.guanYin = value
	p.mu.Unlock()
}

func (p *playerActor) guanYinUpdate() ([]byte, error) {
	p.mu.Lock()
	guanYin := p.guanYin
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propCapitalType2, Name: "CapitalType2", Value: clientdata.Int64Value(int64(guanYin))},
	})
}

func (p *playerActor) guanYinValue() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.guanYin
}

// setPowerLevel 记录实力等级并返回 [PowerLevel, LevelTitle] 双属性帧。
// 称号键复用选人界面的同一分档（Java applyPowerLevel 同口径）。
func (p *playerActor) setPowerLevel(value int32) ([]byte, error) {
	p.mu.Lock()
	p.gm.powerLevel = value
	p.mu.Unlock()
	return p.powerLevelFrame()
}

// powerLevelFrame 总是产出双属性帧（实力等级是面板身份块的必填位，
// Java identityProperties 无条件携带；种子 roleListPowerLevel=151）。
func (p *playerActor) powerLevelFrame() ([]byte, error) {
	p.mu.Lock()
	powerLevel := p.gm.powerLevel
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propPowerLevel, Name: "PowerLevel", Value: clientdata.Int32Value(powerLevel)},
		{Index: propLevelTitle, Name: "LevelTitle", Value: clientdata.StringValue(roleListTitle(int(powerLevel)))},
	})
}

// setAlignment 只接受本次提供的值（0=跳过，与前端 +v||0 折叠语义一致），
// 帧序固定 Justice→Evil→FreeBase（Java applyAlignment 同序）。
func (p *playerActor) setAlignment(justice, evil, freeBase int32) ([]byte, error) {
	p.mu.Lock()
	if justice > 0 {
		p.gm.justice = justice
	}
	if evil > 0 {
		p.gm.evil = evil
	}
	if freeBase > 0 {
		p.gm.freeBase = freeBase
	}
	p.mu.Unlock()
	return p.alignmentFrame()
}

// alignmentFrame 常发三属性（零值合法——面板侠恶位缺发即空白）。
func (p *playerActor) alignmentFrame() ([]byte, error) {
	p.mu.Lock()
	justice, evil, freeBase := p.gm.justice, p.gm.evil, p.gm.freeBase
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propJustice, Name: "Justice", Value: clientdata.Int32Value(justice)},
		{Index: propEvil, Name: "Evil", Value: clientdata.Int32Value(evil)},
		{Index: propFreeBase, Name: "FreeBase", Value: clientdata.Int32Value(freeBase)},
	})
}

func (p *playerActor) setPKMode(mode int32) ([]byte, error) {
	p.mu.Lock()
	p.gm.pkMode = mode
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propPKMode, Name: "PKMode", Value: clientdata.ByteValue(byte(mode))},
	})
}

// setGMVitals 全量覆写生存属性层并按 Java 固定序返回瘦帧
// （Ene→MaxEne→ResumeEne→Sat→MaxSat→DrunkLevel→PKValue；官方表无
// MaxEneAdd/MaxSatAdd，前端两个 Add 字段忽略——Java applyVitals 同）。
func (p *playerActor) setGMVitals(vitals gmVitalsState) ([]byte, error) {
	p.mu.Lock()
	p.gm.vitals, p.gm.vitalsSet = vitals, true
	frame, frameErr := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propEne, Name: "Ene", Value: clientdata.Int32Value(vitals.ene)},
		{Index: propMaxEne, Name: "MaxEne", Value: clientdata.Int32Value(vitals.maxEne)},
		{Index: propResumeEne, Name: "ResumeEne", Value: clientdata.Int32Value(vitals.resumeEne)},
		{Index: propSat, Name: "Sat", Value: clientdata.Int32Value(vitals.sat)},
		{Index: propMaxSat, Name: "MaxSat", Value: clientdata.Int32Value(vitals.maxSat)},
		{Index: propDrunkLevel, Name: "DrunkLevel", Value: clientdata.Int32Value(vitals.drunkLevel)},
		{Index: propPKValue, Name: "PKValue", Value: clientdata.Int32Value(vitals.pkValue)},
	})
	p.mu.Unlock()
	return frame, frameErr
}

// identityFrame 是面板身份块整帧（Java PanelState.identityProperties
// 逐位对应：实力/称号/性格/侠善/恶/狂邪/体力/饱食/杀戮/酒量/PK 模式，
// 属性序与 Java 一致）。零值也要发——参考服登录帧 exe 表 256-266 段
// 缺发则面板整块空白（实力一行不显示的根因）。
func (p *playerActor) identityFrame() ([]byte, error) {
	p.mu.Lock()
	gm := p.gm
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propPowerLevel, Name: "PowerLevel", Value: clientdata.Int32Value(gm.powerLevel)},
		{Index: propLevelTitle, Name: "LevelTitle", Value: clientdata.StringValue(roleListTitle(int(gm.powerLevel)))},
		{Index: propCharacterFlag, Name: "CharacterFlag", Value: clientdata.Int32Value(0)},
		{Index: propCharacterValue, Name: "CharacterValue", Value: clientdata.Int32Value(0)},
		{Index: propJustice, Name: "Justice", Value: clientdata.Int32Value(gm.justice)},
		{Index: propEvil, Name: "Evil", Value: clientdata.Int32Value(gm.evil)},
		{Index: propFreeBase, Name: "FreeBase", Value: clientdata.Int32Value(gm.freeBase)},
		{Index: propEne, Name: "Ene", Value: clientdata.Int32Value(gm.vitals.ene)},
		{Index: propMaxEne, Name: "MaxEne", Value: clientdata.Int32Value(gm.vitals.maxEne)},
		{Index: propResumeEne, Name: "ResumeEne", Value: clientdata.Int32Value(gm.vitals.resumeEne)},
		{Index: propSat, Name: "Sat", Value: clientdata.Int32Value(gm.vitals.sat)},
		{Index: propMaxSat, Name: "MaxSat", Value: clientdata.Int32Value(gm.vitals.maxSat)},
		{Index: propPKValue, Name: "PKValue", Value: clientdata.Int32Value(gm.vitals.pkValue)},
		{Index: propDrunkLevel, Name: "DrunkLevel", Value: clientdata.Int32Value(gm.vitals.drunkLevel)},
		{Index: propPKMode, Name: "PKMode", Value: clientdata.ByteValue(byte(gm.pkMode))},
	})
}

// domainFrame 是面板域块整帧（Java PanelState.domainProperties 保守
// 语义：TitleID 取佩戴位，坐骑/兵录/声望暂无数据源发零值，真机待验后
// 按抓包修正）。与身份块同帧级合并进 gmPanelFrames。
func (p *playerActor) domainFrame() ([]byte, error) {
	p.mu.Lock()
	titleID := p.titleID
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propTitleID, Name: "TitleID", Value: clientdata.Int32Value(titleID)},
		{Index: propMount, Name: "Mount", Value: clientdata.StringValue("")},
		{Index: propCurRideMount, Name: "CurRideMount", Value: clientdata.ObjectValue(0, 0)},
		{Index: propBingLuType, Name: "BingLuType", Value: clientdata.Int32Value(0)},
		{Index: propMaxBingLuLevel, Name: "MaxBingLuLevel", Value: clientdata.Int32Value(0)},
		{Index: propPrestigeExp, Name: "PrestigeExp", Value: clientdata.Int32Value(0)},
	})
}

// gmPanelFrames 生成 spawn 后的面板域补推帧（身份块 + 域块各一帧，
// 常态必有——Java identityProperties/domainProperties 随 0x0D 档下发的
// 瘦帧对应物）。切图重进与登录同口径（0x0C 清玩家对象后重放）。
func (p *playerActor) gmPanelFrames() ([][]byte, error) {
	frames := make([][]byte, 0, 2)
	for _, build := range []func() ([]byte, error){
		p.identityFrame,
		p.domainFrame,
	} {
		frame, err := build()
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// setFacultyValue 修写修炼总量并返回常规全量 0x10（stateProperties 已
// 内嵌 progress.properties()，携 Faculty@798，Java vitalUpdate 同形）。
func (p *playerActor) setFacultyValue(value int32) ([]byte, error) {
	p.mu.Lock()
	if value < 0 {
		value = 0
	}
	p.progress.faculty = value
	p.mu.Unlock()
	return p.vitalUpdate()
}

// setBaseAttributes 覆盖式写入基础五维（0=跳过该维），返回全量 0x10
// （Str/Sta/Dex/Ing/Spi@485-489 由 properties() 携带；装备内功加成在
// properties() 内叠加，基础层与 Java setBaseAttributes 同为覆写语义）。
// 五维是 MaxHPAdd/MaxMPAdd 官方公式的属性分量来源，写入后必须重算
// vitals 推导上限。
func (p *playerActor) setBaseAttributes(str, sta, dex, ing, spi int32) ([]byte, error) {
	p.mu.Lock()
	if str > 0 {
		p.progress.attributes.str = str
	}
	if sta > 0 {
		p.progress.attributes.sta = sta
	}
	if dex > 0 {
		p.progress.attributes.dex = dex
	}
	if ing > 0 {
		p.progress.attributes.ing = ing
	}
	if spi > 0 {
		p.progress.attributes.spi = spi
	}
	p.syncEquippedNeiGongResourcesLocked()
	p.mu.Unlock()
	return p.vitalUpdate()
}

// setFillSpeed 实际修速=round(base×multiplier)（Java PlayerProgress.
// setFillSpeed 同式），返回 [FillSpeed@773, NorFacultyAdd@774] 瘦帧
// （Java 走全量批携带同两属性，瘦帧线缆等价且不动登录帧形）。
func (p *playerActor) setFillSpeed(base int32, multiplier float32) (int32, []byte, error) {
	actual := int32(math.Round(float64(base) * float64(multiplier)))
	p.mu.Lock()
	p.progress.fillSpeed = actual
	p.progress.fillSpeedBase = base
	p.progress.fillSpeedMultiplier = multiplier
	p.mu.Unlock()
	frame, frameErr := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propFillSpeed, Name: "FillSpeed", Value: clientdata.Int32Value(actual)},
		{Index: propNorFacultyAdd, Name: "NorFacultyAdd", Value: clientdata.Int32Value(base)},
	})
	return actual, frame, frameErr
}

// setActDayLimit 入参为“两”（0=不限制），换算与重置语义照搬 Java
// setActDayLimit：两→文×1000；不限 → Max=0/Value=Last=999999999；
// 两种分支都重置今日已耗/已用。返回值是面板回显用的“两”。
func (p *playerActor) setActDayLimit(liang int32) (int32, []byte, error) {
	p.mu.Lock()
	p.progress.actCostSuiYin = 0
	p.progress.actDayValueUsed = 0
	if liang <= 0 {
		p.progress.actDayValueMax = 0
		p.progress.actDayValue = actDayUnlimitedSentinel
		p.progress.actDayValueLast = actDayUnlimitedSentinel
	} else {
		wen := int64(liang) * 1000
		if wen > int64(actDayUnlimitedSentinel) {
			wen = int64(actDayUnlimitedSentinel)
		}
		p.progress.actDayValueMax = int32(wen)
		p.progress.actDayValue = int32(wen)
	}
	actValue, actCost, actLast, actUsed, actMax := p.progress.actDayValue, p.progress.actCostSuiYin,
		p.progress.actDayValueLast, p.progress.actDayValueUsed, p.progress.actDayValueMax
	p.mu.Unlock()
	frame, frameErr := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propActDayValue, Name: "ActDayValue", Value: clientdata.Int32Value(actValue)},
		{Index: propActCostSuiYin, Name: "ActCostSuiYin", Value: clientdata.Int32Value(actCost)},
		{Index: propActDayValueLast, Name: "ActDayValueLast", Value: clientdata.Int32Value(actLast)},
		{Index: propActDayValueUsed, Name: "ActDayValueUsed", Value: clientdata.Int32Value(actUsed)},
		{Index: propActDayValueMax, Name: "ActDayValueMax", Value: clientdata.Int32Value(actMax)},
	})
	display := int32(0)
	if liang > 0 {
		display = liang
	}
	return display, frame, frameErr
}

// facultyValue 供 GM 顶栏修为字段回显。
func (p *playerActor) facultyValue() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.faculty
}

// gmWritePanelFrames 把面板身份/域块补推帧写到连接（登录 spawn 后与
// 0x0C 切图重进两处调用；常态恒有帧——缺发则人物面板整块空白）。
func gmWritePanelFrames(link sceneMessageConnection, player *playerActor) error {
	frames, err := player.gmPanelFrames()
	if err != nil {
		return fmt.Errorf("encode panel frames: %w", err)
	}
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}
