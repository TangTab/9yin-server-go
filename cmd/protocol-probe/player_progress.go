package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// 官方属性表(2370 项)玩家域索引(同名多处取末现)。0x09 表按位置绑定,
// 这些值必须与 internal/clientdata/official_property_table.txt 逐项一致;
// official_property_indices_test.go 按名断言防漂移。
const (
	propCurNeiGong     uint16 = 653 // CurNeiGong String
	propFaculty        uint16 = 798 // Faculty Int32
	propFacultyState   uint16 = 758
	propFacultyStyle   uint16 = 759
	propFacultyName    uint16 = 760
	propCurFillValue   uint16 = 764
	propTotalFillValue uint16 = 765
	propCurLevel       uint16 = 763
	propFillSpeed      uint16 = 773
	propPowerValue     uint16 = 2287
	propMaxPowerValue  uint16 = 2288
	propSP             uint16 = 467 // SP Int32:怒气
	propStr            uint16 = 485
	propSta            uint16 = 489
	propDex            uint16 = 487
	propIng            uint16 = 486
	propSpi            uint16 = 488
	propMeleePower     uint16 = 500
	propMagicPower     uint16 = 502
	propPhyHit         uint16 = 556
	propMagicHit       uint16 = 559
	propDodge          uint16 = 562
	propMaxParry       uint16 = 476
	propPhyDef         uint16 = 584
	propStiDef         uint16 = 586
	propNegDef         uint16 = 588
	propMasDef         uint16 = 589
	propJujDef         uint16 = 587
	propMinMeleeDamage uint16 = 510
	propMaxMeleeDamage uint16 = 509
	propMaxHPAdd       uint16 = 581
	propMaxMPAdd       uint16 = 582
	propMaxLevel       uint16 = 2083
	// View 43 的条目是普通客户端对象。GoodsGrid 读取这两个属性来解析
	// 当前的内功静态记录及其图片；对现代客户端而言仅有 ConfigID 是
	// 不够的。
	propStaticData uint16 = 2237
	propItemType   uint16 = 1889
	// 装备内功的 PropModify 加成。
	propMaxParryAdd    uint16 = 477
	propMinMagicDefAdd uint16 = 597
)

// MaxSP(580)/MaxSPAdd(583) 官方表存在。MaxSP 只在登录/切图全量批由
// player_actor.go 的 loginVitalUpdate 携带一次(49840 帧实证仅 #16 一处),
// 其余任何批次不得携带;MaxSPAdd 从不下发。这里不定义常量,防止误用。

// activeFacultyCardReward 刻意只在玩家从三张卡中选出一张（C->S 1006/23）
// 之后才发放。主动页面并不是一个按分钟计的修炼计时器。
const activeFacultyCardReward int32 = 20_000

const (
	viewportNeiGong uint16 = 43

	facultyStateNone    int32 = 0
	facultyStateConvert int32 = 2
	facultyStyleNormal  int32 = 1
	facultyStyleAct     int32 = 2
	logicStateFaculty   byte  = 108

	normalFacultyFillPerMinute int32 = 200

	// 快照额外书的显示兜底（与 starter 目录 1 级书的初值同款）。
	defaultExtraNeiGongMaxLevel int32 = 36
	defaultExtraNeiGongTotal    int32 = 700
	defaultExtraNeiGongPower    int32 = 120
)

func facultyBool(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

type learnedNeiGong struct {
	configID   string
	staticData int32
	itemType   int32
	level      int32
	maxLevel   int32
	fill       int32
	total      int32
	power      int32
	maxPower   int32
	// 这些是从现代 neigong_varprop.ini 记录读取的装备等级效果。它们是
	// 服务器状态：不能只有 UI 一个地方知道装备内功产生了效果。
	buffID         string
	buffStaticData uint32
	buffLevel      int32
	strAdd         int32
	staAdd         int32
	dexAdd         int32
	ingAdd         int32
	spiAdd         int32
	maxHPAdd       int32
	maxMPAdd       int32
	maxParryAdd    int32
	minMagicDefAdd int32
	// skillModifiers 是当前等级确切的 SkillModifyPack 行，按目标 SkillID
	// 索引。它们保存在已装备秘籍的状态上，让 211 裁定器只应用点名该次
	// 施放的修正项。
	skillModifiers map[string][]modernNeiGongModifier
}

// playerProgress 刻意是一个轻量的服务器持有状态模型。其 ConfigID 来自
// 已安装的现代 share/skill/neigong/neigong.ini；在逐级 varprop 加载器
// 加入之前，数值都是本地测试的初始值。线上名称采用当前 Lua/C++ UI
// 绑定的那些名字。
type playerProgress struct {
	books        []learnedNeiGong
	curNeiGong   string
	faculty      int32
	facultyState int32
	facultyStyle int32
	facultyName  string
	fillSpeed    int32
	lastAdvance  time.Time
	attributes   playerAttributes
}

type playerAttributes struct {
	sp, str, sta, dex, ing, spi                        int32
	meleePower, magicPower, phyHit, magicHit, dodge    int32
	maxParry, phyDef, stiDef, negDef, masDef, jujDef   int32
	minMeleeDamage, maxMeleeDamage, maxHPAdd, maxMPAdd int32
}

// equippedResourceAdds 返回的正是已装备内功 varprop 携带的资源加成。
// 当前现代客户端直接以 MaxMP + MaxMPAdd 的方式消费 MaxMPAdd，因此对
// 服务器 Actor 的总资源上限而言，同一个原始值
// 就是权威值。
func (p *playerProgress) equippedResourceAdds() (maxHPAdd, maxMPAddRaw int32) {
	if equipped := p.book(p.curNeiGong); equipped != nil {
		return equipped.maxHPAdd, equipped.maxMPAdd
	}
	return 0, 0
}

func newPlayerProgress() playerProgress {
	books := []learnedNeiGong{
		// 数值取自已安装的现代
		// share/skill/neigong/neigong.ini，而非旧服务器的别名。
		// 两仪护心功：30 级是刻意设定的本地测试基线。其进度值是已安装
		// 现代 wuxuelevelinfo.ini [ng_wd_001] 第 30 行（273200），不是
		// 遗留数值。已安装文件中有两条同名 QName 的记录。经典记录
		// （StaticData 5 / varprop 220130）是官方的可见运行 Buff 路线；
		// mind_ 重复项使用明确隐藏的 Buff（Visible=0）。两行的 30 级
		// 属性完全相同。
		{configID: "ng_wd_001", staticData: 5, itemType: 1002, level: 30, maxLevel: 36, total: 273200, power: 4550, maxPower: 4550,
			buffID: "buf_ng_wd_001", buffStaticData: 5954, buffLevel: 3,
			strAdd: 45, staAdd: 31, dexAdd: 13, ingAdd: 46, spiAdd: 13,
			maxHPAdd: 1630, maxMPAdd: 490, maxParryAdd: 840, minMagicDefAdd: 45},
		// 移花宫及其花无缺路线共用明玉神功。不存在单独的花无缺内功
		// ConfigID；CS_yhwq_hsqs 是男性版本的套路。使用经典的可见 Buff
		// 记录和确切的 30 级行 810630，而不是它隐藏的 mind_ 重复项。
		{configID: "ng_yh_001", staticData: 114, itemType: 1002, level: 30, maxLevel: 71, total: 579600, power: 8180, maxPower: 8180,
			buffID: "buf_ng_yh_001", buffStaticData: 13073, buffLevel: 3,
			strAdd: 55, staAdd: 57, dexAdd: 70, ingAdd: 97, spiAdd: 67,
			maxHPAdd: 4050, maxMPAdd: 1130, maxParryAdd: 5040, minMagicDefAdd: 100},
		// 桃花岛的碧波心法使用同一套现代资源曲线：经典 StaticData 113，
		// level-30 varprop 810530，且可见 Buff 是 buf_ng_th_001。不要使用
		// mind_ StaticData 1113；它的 Buff 明确是隐藏路线。
		{configID: "ng_th_001", staticData: 113, itemType: 1002, level: 30, maxLevel: 71, total: 579600, power: 8180, maxPower: 8180,
			buffID: "buf_ng_th_001", buffStaticData: 13434, buffLevel: 3,
			strAdd: 66, staAdd: 56, dexAdd: 52, ingAdd: 103, spiAdd: 68,
			maxHPAdd: 4050, maxMPAdd: 1135, maxParryAdd: 5070, minMagicDefAdd: 100},
		// 这里同样使用经典的可见记录。1006/1008 是 mind_ 重复项：它们
		// 显示的 Buff 路线被明确隐藏。
		{configID: "ng_wd_002", staticData: 6, itemType: 1002, level: 1, maxLevel: 36, total: 700, power: 120, maxPower: 120},
		{configID: "ng_jh_001", staticData: 1, itemType: 1002, level: 1, maxLevel: 36, total: 700, power: 120, maxPower: 120},
		{configID: "ng_em_001", staticData: 8, itemType: 1002, level: 1, maxLevel: 36, total: 700, power: 120, maxPower: 120},
	}
	progress := playerProgress{
		books:        books,
		curNeiGong:   books[0].configID,
		faculty:      9_999_999,
		facultyState: facultyStateConvert,
		facultyStyle: facultyStyleNormal,
		facultyName:  books[0].configID,
		fillSpeed:    200,
		lastAdvance:  time.Now().UTC(),
		attributes: playerAttributes{
			sp: maxPlayerSP, str: 30, sta: 30, dex: 30, ing: 30, spi: 30,
			meleePower: 80, magicPower: 80, phyHit: 50, magicHit: 50,
			dodge: 35, maxParry: 100, phyDef: 60, stiDef: 50, negDef: 50,
			masDef: 50, jujDef: 50, minMeleeDamage: 20, maxMeleeDamage: 35,
		},
	}
	// 上面的内联值只是无资源时的兜底。每本已安装的现代秘籍都会立即被
	// 其 StaticData/当前等级的 varprop 和 PropModify 结果替换。
	progress.refreshAllModernEffects()
	return progress
}

func (book *learnedNeiGong) applyModernEffect() error {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return err
	}
	effect, err := catalog.effect(book.staticData, book.level)
	if err != nil {
		return err
	}
	book.buffID = effect.buffID
	book.buffStaticData = effect.staticData
	book.buffLevel = effect.buffLevel
	book.strAdd = effect.stats["StrAdd"]
	book.staAdd = effect.stats["StaAdd"]
	book.dexAdd = effect.stats["DexAdd"]
	book.ingAdd = effect.stats["IngAdd"]
	book.spiAdd = effect.stats["SpiAdd"]
	book.maxHPAdd = effect.stats["MaxHPAdd"]
	book.maxMPAdd = effect.stats["MaxMPAdd"]
	book.maxParryAdd = effect.stats["MaxParryAdd"]
	book.minMagicDefAdd = effect.stats["MinMagicDefAdd"]
	book.skillModifiers = make(map[string][]modernNeiGongModifier, len(effect.skillModifiers))
	for skillID, modifiers := range effect.skillModifiers {
		book.skillModifiers[skillID] = append([]modernNeiGongModifier(nil), modifiers...)
	}
	return nil
}

// refreshAllModernEffects 刻意做成尽力而为，只为保证启动可移植性。生产
// 环境使用上面固定的现代资源目录；在本地缺少资源包时保留旧值，让单元
// 级别的协议工作保持可用，而不是把实际游戏效果替换成
// 零值。
func (p *playerProgress) refreshAllModernEffects() {
	for index := range p.books {
		if err := p.books[index].applyModernEffect(); err != nil {
			continue
		}
	}
}

func (p *playerProgress) book(id string) *learnedNeiGong {
	for index := range p.books {
		if p.books[index].configID == id {
			return &p.books[index]
		}
	}
	return nil
}

func (p *playerProgress) targetBook() *learnedNeiGong {
	if p.facultyName != "" {
		if book := p.book(p.facultyName); book != nil {
			return book
		}
	}
	return p.book(p.curNeiGong)
}

func (p *playerProgress) properties() []clientdata.IndexedProperty {
	book := p.targetBook()
	level, maxLevel, fill, total, power, maxPower := int32(0), int32(0), int32(0), int32(0), int32(0), int32(0)
	if book != nil {
		level, maxLevel, fill, total, power, maxPower = book.level, book.maxLevel, book.fill, book.total, book.power, book.maxPower
	}
	a := p.attributes
	maxParryAdd, minMagicDefAdd := int32(0), int32(0)
	// 将当前装备的内功应用到服务器持有的角色状态上。对应的 BuffStore
	// 条目由 playerActor 安装；可见的主属性必须遵循同一个权威来源。
	if equipped := p.book(p.curNeiGong); equipped != nil {
		a.str += equipped.strAdd
		a.sta += equipped.staAdd
		a.dex += equipped.dexAdd
		a.ing += equipped.ingAdd
		a.spi += equipped.spiAdd
		a.maxHPAdd += equipped.maxHPAdd
		a.maxMPAdd += equipped.maxMPAdd
		maxParryAdd = equipped.maxParryAdd
		minMagicDefAdd = equipped.minMagicDefAdd
	}
	return []clientdata.IndexedProperty{
		{Index: propCurNeiGong, Name: "CurNeiGong", Value: clientdata.StringValue(p.curNeiGong)},
		{Index: propFaculty, Name: "Faculty", Value: clientdata.Int32Value(p.faculty)},
		{Index: propFacultyState, Name: "FacultyState", Value: clientdata.Int32Value(p.facultyState)},
		{Index: propFacultyStyle, Name: "FacultyStyle", Value: clientdata.Int32Value(p.facultyStyle)},
		{Index: propFacultyName, Name: "FacultyName", Value: clientdata.StringValue(p.facultyName)},
		{Index: propCurFillValue, Name: "CurFillValue", Value: clientdata.Int32Value(fill)},
		{Index: propTotalFillValue, Name: "TotalFillValue", Value: clientdata.Int32Value(total)},
		{Index: propCurLevel, Name: "CurLevel", Value: clientdata.Int32Value(level)},
		{Index: propFillSpeed, Name: "FillSpeed", Value: clientdata.Int32Value(p.fillSpeed)},
		{Index: propPowerValue, Name: "PowerValue", Value: clientdata.Int32Value(power)},
		{Index: propMaxPowerValue, Name: "MaxPowerValue", Value: clientdata.Int32Value(maxPower)},
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(a.sp)},
		{Index: propStr, Name: "Str", Value: clientdata.Int32Value(a.str)},
		{Index: propSta, Name: "Sta", Value: clientdata.Int32Value(a.sta)},
		{Index: propDex, Name: "Dex", Value: clientdata.Int32Value(a.dex)},
		{Index: propIng, Name: "Ing", Value: clientdata.Int32Value(a.ing)},
		{Index: propSpi, Name: "Spi", Value: clientdata.Int32Value(a.spi)},
		{Index: propMeleePower, Name: "MeleePower", Value: clientdata.Int32Value(a.meleePower)},
		{Index: propMagicPower, Name: "MagicPower", Value: clientdata.Int32Value(a.magicPower)},
		{Index: propPhyHit, Name: "PhyHit", Value: clientdata.Int32Value(a.phyHit)},
		{Index: propMagicHit, Name: "MagicHit", Value: clientdata.Int32Value(a.magicHit)},
		{Index: propDodge, Name: "Dodge", Value: clientdata.Int32Value(a.dodge)},
		{Index: propMaxParry, Name: "MaxParry", Value: clientdata.Int32Value(a.maxParry)},
		{Index: propPhyDef, Name: "PhyDef", Value: clientdata.Int32Value(a.phyDef)},
		{Index: propStiDef, Name: "StiDef", Value: clientdata.Int32Value(a.stiDef)},
		{Index: propNegDef, Name: "NegDef", Value: clientdata.Int32Value(a.negDef)},
		{Index: propMasDef, Name: "MasDef", Value: clientdata.Int32Value(a.masDef)},
		{Index: propJujDef, Name: "JujDef", Value: clientdata.Int32Value(a.jujDef)},
		{Index: propMinMeleeDamage, Name: "MinMeleeDamage", Value: clientdata.Int32Value(a.minMeleeDamage)},
		{Index: propMaxMeleeDamage, Name: "MaxMeleeDamage", Value: clientdata.Int32Value(a.maxMeleeDamage)},
		{Index: propMaxHPAdd, Name: "MaxHPAdd", Value: clientdata.Int32Value(a.maxHPAdd)},
		{Index: propMaxMPAdd, Name: "MaxMPAdd", Value: clientdata.Int32Value(a.maxMPAdd)},
		{Index: propMaxParryAdd, Name: "MaxParryAdd", Value: clientdata.Int32Value(maxParryAdd)},
		{Index: propMinMagicDefAdd, Name: "MinMagicDefAdd", Value: clientdata.Int32Value(minMagicDefAdd)},
		{Index: propMaxLevel, Name: "MaxLevel", Value: clientdata.Int32Value(maxLevel)},
	}
}

func (p *playerProgress) viewFrames() ([][]byte, error) {
	frames := [][]byte{serverCreateView(serverViewSpec{ID: viewportNeiGong, Capacity: uint16(len(p.books))})}
	for index := range p.books {
		frame, err := p.bookFrame(uint16(index + 1))
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func (p *playerProgress) bookFrame(slot uint16) ([]byte, error) {
	if slot == 0 || int(slot) > len(p.books) {
		return nil, fmt.Errorf("invalid neigong slot %d", slot)
	}
	book := p.books[slot-1]
	return serverViewAdd(viewportNeiGong, slot, []serverViewProperty{
		viewString(1440, book.configID),
		viewInt(propStaticData, book.staticData),
		viewInt(propItemType, book.itemType),
		viewByte(1465, byte(book.level)),
		viewInt(propMaxLevel, book.maxLevel),
		viewInt(propCurFillValue, book.fill),
		viewInt(propTotalFillValue, book.total),
		viewInt(propPowerValue, book.power),
		viewInt(propMaxPowerValue, book.maxPower),
	})
}

func (p *playerProgress) equip(id string) error {
	if p.book(id) == nil {
		return fmt.Errorf("not learned neigong %q", id)
	}
	p.curNeiGong = id
	return nil
}

func (p *playerProgress) facultyReady(id string) (uint16, error) {
	var slot uint16
	for index := range p.books {
		if p.books[index].configID == id {
			slot = uint16(index + 1)
			break
		}
	}
	if slot == 0 {
		return 0, fmt.Errorf("not learned neigong %q", id)
	}
	p.facultyName = id
	// form_wuxue_faculty.prop_callback_ConvertState 只渲染状态 0（空闲，
	// 显示普通开始按钮）和状态 2（转换中）。这里发送客户端常量 READY=1
	// 会让控件停留在 XML 默认状态，直到某个演武请求把状态改为 2——这
	// 正是玩家必须先点一次演武、内修才显示为可用的原因。
	p.facultyState = facultyStateNone
	p.facultyStyle = facultyStyleNormal
	p.fillSpeed = 0
	p.lastAdvance = time.Now().UTC()
	return slot, nil
}

func (p *playerProgress) facultyBegin(style int32, now time.Time) error {
	if p.book(p.facultyName) == nil {
		return fmt.Errorf("no selected learned neigong")
	}
	if style != facultyStyleAct {
		style = facultyStyleNormal
	}
	p.facultyState = facultyStateConvert
	p.facultyStyle = style
	p.fillSpeed = normalFacultyFillPerMinute
	if style == facultyStyleAct {
		// 演武在选卡之后才结算收益。这里若通告每分钟速率，会导致服务器
		// 在卡牌页面尚未打开时也推进进度。
		p.fillSpeed = 0
	}
	p.lastAdvance = now.UTC()
	return nil
}

func (p *playerProgress) facultyExit() {
	p.facultyState = facultyStateNone
	p.facultyStyle = facultyStyleNormal
	p.fillSpeed = 0
	p.lastAdvance = time.Now().UTC()
}

// advanceFaculty 按通告的 FillSpeed 结算整分钟数。余数保留在 lastAdvance
// 中，重连或重启都不会丢失部分时间。普通修炼和主动的演武状态消费同一
// 份权威内功进度；restore 刻意重置活动会话的时钟，因此演武只在该活动
// 会话运行期间累积，绝不会获得离线进度。
func (p *playerProgress) advanceFaculty(now time.Time) (uint16, bool) {
	if p.facultyState != facultyStateConvert || p.facultyStyle != facultyStyleNormal || p.fillSpeed <= 0 {
		p.lastAdvance = now.UTC()
		return 0, false
	}
	now = now.UTC()
	if p.lastAdvance.IsZero() || now.Before(p.lastAdvance) {
		p.lastAdvance = now
		return 0, false
	}
	minutes := int32(now.Sub(p.lastAdvance) / time.Minute)
	if minutes <= 0 {
		return 0, false
	}
	p.lastAdvance = p.lastAdvance.Add(time.Duration(minutes) * time.Minute)
	for index := range p.books {
		book := &p.books[index]
		if book.configID != p.facultyName {
			continue
		}
		fill := int64(p.fillSpeed) * int64(minutes)
		cost := (fill + 1) / 2
		if cost > int64(p.faculty) {
			cost = int64(p.faculty)
			fill = cost * 2
		}
		if fill <= 0 {
			p.facultyState = facultyStateNone
			p.fillSpeed = 0
			return 0, false
		}
		p.faculty -= int32(cost)
		book.fill += int32(fill)
		for book.fill >= book.total && book.level < book.maxLevel {
			book.fill -= book.total
			book.level++
			book.total += 200
			book.maxPower += 20
			book.power = book.maxPower
		}
		// 升级会改变所选 varprop、BufferLevel、PropModify 包和
		// SkillModify 包。在下一次 0x10/View 更新前重新求值。
		_ = book.applyModernEffect()
		if book.level >= book.maxLevel {
			book.fill = 0
			p.facultyState = facultyStateNone
			p.fillSpeed = 0
		}
		return uint16(index + 1), true
	}
	return 0, false
}

// awardActiveFaculty 恰好结算一张演武卡。选择索引和支付来源在协议边界
// 校验；本地测试领域当前使用固定的 20000 点卡牌结果，同时让运行时保持
// 主动状态，以配合客户端两秒的卡牌动画/冷却。
func (p *playerProgress) awardActiveFaculty() (slot uint16, awarded, levelUp, maxLevel bool, err error) {
	if p.facultyState != facultyStateConvert || p.facultyStyle != facultyStyleAct {
		// 当前表单在玩家关闭或切换修炼模式时，可能在 UI 队列里留下一条
		// 延迟的 SUB_CLIENT_ACT_PAY。它是一条过期的业务请求，不是畸形
		// 的传输消息，绝不能因此拆除整个游戏连接。
		return 0, false, false, false, nil
	}
	for index := range p.books {
		book := &p.books[index]
		if book.configID != p.facultyName {
			continue
		}
		if book.level >= book.maxLevel {
			return uint16(index + 1), false, false, true, nil
		}
		previousLevel := book.level
		book.fill += activeFacultyCardReward
		for book.fill >= book.total && book.level < book.maxLevel {
			book.fill -= book.total
			book.level++
			book.total += 200
			book.maxPower += 20
			book.power = book.maxPower
		}
		_ = book.applyModernEffect()
		if book.level >= book.maxLevel {
			book.fill = 0
		}
		return uint16(index + 1), true, book.level != previousLevel, book.level >= book.maxLevel, nil
	}
	return 0, false, false, false, fmt.Errorf("active faculty selected book %q is not learned", p.facultyName)
}

func (p *playerProgress) snapshot() facultyProgressSnapshot {
	value := facultyProgressSnapshot{
		CurNeiGong:     p.curNeiGong,
		Faculty:        p.faculty,
		FacultyState:   p.facultyState,
		FacultyStyle:   p.facultyStyle,
		FacultyName:    p.facultyName,
		FillSpeed:      p.fillSpeed,
		LastAdvanceUTC: p.lastAdvance.UTC(),
		Books:          make([]facultyBookSnapshot, 0, len(p.books)),
	}
	for _, book := range p.books {
		value.Books = append(value.Books, facultyBookSnapshot{
			ConfigID: book.configID, Level: book.level, Fill: book.fill,
			Total: book.total, Power: book.power, MaxPower: book.maxPower,
		})
	}
	return value
}

func (p *playerProgress) restore(value facultyProgressSnapshot, now time.Time) {
	// 快照书集合是权威（Java PlayerProgress 同口径）：先回填 starter
	// 目录之外的书，随后的 CurNeiGong/FacultyName 校验才能命中它们。
	p.restoreExtraBooks(value)
	if p.book(value.CurNeiGong) != nil {
		p.curNeiGong = value.CurNeiGong
	}
	if value.Faculty >= 0 {
		p.faculty = value.Faculty
	}
	if p.book(value.FacultyName) != nil {
		p.facultyName = value.FacultyName
	}
	p.facultyState = value.FacultyState
	if value.FacultyStyle == facultyStyleAct {
		p.facultyStyle = facultyStyleAct
	} else {
		p.facultyStyle = facultyStyleNormal
	}
	p.fillSpeed = value.FillSpeed
	p.lastAdvance = value.LastAdvanceUTC.UTC()
	for _, saved := range value.Books {
		book := p.book(saved.ConfigID)
		if book == nil || saved.Level <= 0 || saved.Level > book.maxLevel || saved.Total <= 0 || saved.Fill < 0 {
			continue
		}
		book.level = saved.Level
		book.fill = saved.Fill
		book.total = saved.Total
		book.power = saved.Power
		book.maxPower = saved.MaxPower
	}
	if p.facultyState == facultyStateConvert && p.facultyStyle == facultyStyleNormal {
		_, _ = p.advanceFaculty(now)
	} else {
		p.lastAdvance = now.UTC()
	}
	p.refreshAllModernEffects()
}

// restoreExtraBooks 把快照中超出 starter 目录的书从 neigong.ini 目录
// 回填后追加。目录查无此书时丢弃（客户端无法把未知的 ConfigID 解析成
// wuxue 对象，faculty.lua 的普通开始按钮会因此保持禁用）；进度仍由
// restore 的公共覆盖循环按快照值校验写入。无资源目录时（单元测试）
// 整体跳过，行为与历史版本一致。
func (p *playerProgress) restoreExtraBooks(value facultyProgressSnapshot) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return
	}
	appended := make([]string, 0, 4)
	for _, saved := range value.Books {
		if p.book(saved.ConfigID) != nil {
			continue
		}
		entry, ok := catalog.book(saved.ConfigID)
		if !ok || entry.staticData <= 0 {
			log.Printf("faculty snapshot book %s missing from neigong.ini catalog; dropped", saved.ConfigID)
			continue
		}
		p.books = append(p.books, learnedNeiGong{
			configID:   saved.ConfigID,
			staticData: entry.staticData,
			itemType:   entry.itemType,
			level:      1,
			maxLevel:   extraNeiGongMaxLevel(catalog, entry.staticData, saved),
			total:      defaultExtraNeiGongTotal,
			power:      defaultExtraNeiGongPower,
			maxPower:   defaultExtraNeiGongPower,
		})
		appended = append(appended, saved.ConfigID)
	}
	if len(appended) > 0 {
		log.Printf("faculty snapshot restored %d catalog backfilled books: %v", len(appended), appended)
	}
}

// extraNeiGongMaxLevel 取显示上限：快照 extras 的 max_level（新版服
// 写入）优先，否则从 varprop 区间推导，最终回退 starter 目录的 36。
func extraNeiGongMaxLevel(catalog *modernNeiGongCatalog, staticData int32, saved facultyBookSnapshot) int32 {
	if value := snapshotExtraInt(saved.extras, "max_level"); value > 0 {
		return value
	}
	if value := catalog.maxLevelOf(staticData); value > 0 {
		return value
	}
	return defaultExtraNeiGongMaxLevel
}

// snapshotExtraInt 读取快照透传键里的整数值；缺失或非数字返回 0。
func snapshotExtraInt(extras map[string]json.RawMessage, key string) int32 {
	raw, ok := extras[key]
	if !ok || len(raw) == 0 {
		return 0
	}
	var value int32
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0
	}
	return value
}

// S2CFacultyMessage 是当前表 custom_faculty_msg 接收器对应的 C++
// GameMessageHandler 数字入口。该表采用 slot = 2*messageID + 15 的布局：
// 其条目是槽位 397，因此 ID 为 191。同一公式还被现网的 NPC 对话和剧情
// 条目独立印证：槽位 101/781 分别解析为其可用 ID 43/383。
const S2CFacultyMessage int32 = 191

// encodeFacultyMessage 走现代的数字 GameMessageHandler 分发器。它把
// ID 191 解析到 custom_handler.custom_faculty_msg，再调用
// faculty.on_msg(subcommand, arg2)。以字符串形式发送 Lua 函数名会被
// 当前客户端拒绝。
func encodeFacultyMessage(subcommand int32, arg2 int32) ([]byte, error) {
	return serverCustomIntMessage(S2CFacultyMessage,
		customInt(subcommand),
		customInt(arg2),
	)
}

func encodeFacultyActiveBegin() ([]byte, error) {
	return encodeFacultyMessage(22, 0) // faculty.SUB_SERVER_ACT_BEGIN
}

// handlePlayerProgressCustom 接受客户端发起的 TVarList 请求。
func handlePlayerProgressCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return false, nil
	}
	if custom.Values[0].Int32 != 215 && custom.Values[0].Int32 != 1006 {
		return false, nil
	}
	if player == nil {
		return true, fmt.Errorf("player progress request before scene player exists")
	}

	syncState := func() error {
		frame, err := player.vitalUpdate()
		if err != nil {
			return err
		}
		return link.WriteFrame(frame)
	}
	switch custom.Values[0].Int32 {
	case 215: // CLIENT_CUSTOMMSG_USE_NEIGONG
		if len(custom.Values) < 2 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
			return true, fmt.Errorf("215 requires a learned neigong ConfigID string")
		}
		if err := player.equipNeiGong(custom.Values[1].Text); err != nil {
			return true, err
		}
		return true, syncState()

	case 1006: // CLIENT_CUSTOMMSG_FACULTY_INFO
		if len(custom.Values) < 2 || custom.Values[1].Type != 2 {
			return true, fmt.Errorf("1006 requires an int subcommand")
		}
		switch custom.Values[1].Int32 {
		case 1: // SUB_CLIENT_CONVERT_READY
			if len(custom.Values) < 3 || custom.Values[2].Type != 6 || custom.Values[2].Text == "" {
				return true, fmt.Errorf("1006 sub=1 requires a learned neigong ConfigID string")
			}
			slot, err := player.facultyReady(custom.Values[2].Text)
			if err != nil {
				// View=43 的选择流量可能在客户端切换其他武学面板期间到达。
				// 一个非内功的 ConfigID 并不意味着游戏会话损坏，绝不能
				// 因此断开玩家。
				return true, nil
			}
			// 在 FacultyName 到达客户端之前刷新所选的 View=43 对象。
			// set_control_state 会立即解析武学对象并读取 Level/MaxLevel；
			// 此前这个刷新只在 1006/21 之后发生，这正是为什么先点一次演武
			// 看起来像是解锁了普通的内修按钮。
			frame, err := player.progressBookFrame(slot)
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
			if err := syncState(); err != nil {
				return true, err
			}
			return true, nil
		case 11: // SUB_CLIENT_NORMAL_BEGIN
			slot, advanced, err := player.facultyBegin(facultyStyleNormal)
			if err != nil {
				return true, err
			}
			if err := syncState(); err != nil {
				return true, err
			}
			if advanced {
				frame, err := player.progressBookFrame(slot)
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(frame); err != nil {
					return true, err
				}
			}
			return true, nil
		case 21: // SUB_CLIENT_ACT_BEGIN
			slot, advanced, err := player.facultyBegin(facultyStyleAct)
			if err != nil {
				return true, err
			}
			if err := syncState(); err != nil {
				return true, err
			}
			if advanced {
				frame, err := player.progressBookFrame(slot)
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(frame); err != nil {
					return true, err
				}
			}
			frame, err := encodeFacultyActiveBegin()
			if err != nil {
				return true, err
			}
			if err := link.WriteFrame(frame); err != nil {
				return true, err
			}
			return true, nil
		case 22: // SUB_CLIENT_ACT_EXIT
			player.facultyExit()
			return true, syncState()
		case 23: // SUB_CLIENT_ACT_PAY: 卡牌索引 + 支付类型
			if len(custom.Values) < 4 {
				return true, nil
			}
			// 现网现代客户端把两个选择都以 TVarList double 发出：
			// 1006, 23, double:0, double:1。它们是枚举值而不是数量，因此
			// 只接受精确的整型数值。
			card, cardOK := custom.Values[2].exactInt32()
			payment, paymentOK := custom.Values[3].exactInt32()
			if !cardOK || !paymentOK {
				return true, nil
			}
			if card < 0 || card > 2 {
				return true, nil
			}
			if payment != 1 && payment != 2 && payment != 4 {
				return true, nil
			}
			slot, awarded, levelUp, maxLevel, err := player.progress.awardActiveFaculty()
			if err != nil {
				return true, err
			}
			if awarded {
				// 修炼面板从它的 View=43 物品对象读取 Level/Fill。请在主动
				// 表单仍然存在时先发送该对象；否则关闭路径可能让面板默认的
				// 一级对象在重新打开时
				// 仍然处于绑定状态。
				frame, err := player.progressBookFrame(slot)
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(frame); err != nil {
					return true, err
				}
			}
			if err := syncState(); err != nil {
				return true, err
			}
			if levelUp {
				// 客户端不会仅凭 View=43 推断出升级。faculty.lua 把 sub=14 映射
				// 到 on_msg_level_up()，后者读取已同步的 CurLevel 并刷新面板/
				// 通知。
				levelFrame, err := encodeFacultyMessage(14, facultyBool(maxLevel))
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(levelFrame); err != nil {
					return true, err
				}
			}
			if levelUp || maxLevel {
				// 完整的权威等级状态现已到达客户端：先是 View=43 秘籍，然后
				// 是角色属性，最后是原生升级通知。直到现在才关闭主动页面。
				// 客户端表单内部的 exit_flag 仍会抑制其过期的 C2S 1006/22
				// 回调。
				exitFrame, err := encodeFacultyMessage(23, 0) // SUB_SERVER_ACT_EXIT
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(exitFrame); err != nil {
					return true, err
				}
				return true, nil
			}
			frame, err := encodeFacultyMessage(24, 0) // SUB_SERVER_ACT_PLAY
			if err != nil {
				return true, err
			}
			return true, link.WriteFrame(frame)
		case 12: // SUB_CLIENT_NORMAL_EXIT
			player.facultyExit()
			if err := syncState(); err != nil {
				return true, err
			}
			return true, nil
		case 24:
			// 修炼表单只在较早的 1006/1 选择回复之后才绑定 FacultyName。在
			// 恢复权威状态之前先脉冲这个客户端属性，让 set_control_state
			// 在每次打开时都会执行。
			if frame, needed, err := player.facultyNameResetFrame(); err != nil {
				return true, err
			} else if needed {
				if err := link.WriteFrame(frame); err != nil {
					return true, err
				}
			}
			if err := syncState(); err != nil {
				return true, err
			}
			return true, nil
		case 31, 32:
			// 这些是客户端针对经脉流转的上报/打开请求。它们的深层业务系统
			// 刻意不做伪造，但接受它们可以让共享的 1006 总线保持规整。
			if err := syncState(); err != nil {
				return true, err
			}
			return true, nil
		default:
			return true, fmt.Errorf("unsupported 1006 subcommand %d", custom.Values[1].Int32)
		}
	}
	return false, nil
}
