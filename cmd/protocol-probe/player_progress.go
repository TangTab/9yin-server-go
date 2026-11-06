package main

import (
	"fmt"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// These fields are deliberately appended after the live motion/Buff schema.
// The modern client resolves role binds by the negotiated property name, but
// the index must never reuse the already-working qinggong or BuffStore slots.
const (
	propCurNeiGong uint16 = 171 + iota
	propFaculty
	propFacultyState
	propFacultyStyle
	propFacultyName
	propCurFillValue
	propTotalFillValue
	propCurLevel
	propFillSpeed
	propPowerValue
	propMaxPowerValue
	propSP
	propStr
	propSta
	propDex
	propIng
	propSpi
	propMeleePower
	propMagicPower
	propPhyHit
	propMagicHit
	propDodge
	propMaxParry
	propPhyDef
	propStiDef
	propNegDef
	propMasDef
	propJujDef
	propMinMeleeDamage
	propMaxMeleeDamage
	propMaxHPAdd
	propMaxMPAdd
	propMaxLevel
	// View 43 items are normal client objects.  GoodsGrid reads these two
	// properties to resolve the current neigong static record and its photo;
	// ConfigID alone is not sufficient for the modern client.
	propStaticData
	propItemType
	// Equipped inner-power PropModify additions. Keep these appended so the
	// already-live View=43 field indices remain stable.
	propMaxParryAdd
	propMinMagicDefAdd
)

// activeFacultyCardReward is deliberately applied only after the player has
// selected one of the three cards (C->S 1006/23).  The active page is not a
// per-minute cultivation timer.
const activeFacultyCardReward int32 = 20_000

const (
	viewportNeiGong uint16 = 43

	facultyStateNone    int32 = 0
	facultyStateConvert int32 = 2
	facultyStyleNormal  int32 = 1
	facultyStyleAct     int32 = 2
	logicStateFaculty   byte  = 108

	normalFacultyFillPerMinute int32 = 200
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
	// These are the equipped-level effects read from the modern
	// neigong_varprop.ini record.  They are server state: the UI must not be
	// the only place that knows an equipped inner power has an effect.
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
	// skillModifiers are the current level's exact SkillModifyPack rows,
	// indexed by the target SkillID. They are kept on the equipped-book state
	// so the 211 resolver can apply only the modifier that names the cast.
	skillModifiers map[string][]modernNeiGongModifier
}

// playerProgress is intentionally a modest server-owned state model.  Its
// ConfigIDs come from the installed modern share/skill/neigong/neigong.ini;
// values are local-test starter values until the per-level varprop loader is
// added.  The wire-facing names are the ones bound by the current Lua/C++ UI.
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

// equippedResourceAdds returns exactly the resource additions carried by the
// equipped inner-power varprop. The current modern client consumes MaxMPAdd
// directly as MaxMP + MaxMPAdd, so the same raw value is authoritative for
// the server Actor's total resource cap.
func (p *playerProgress) equippedResourceAdds() (maxHPAdd, maxMPAddRaw int32) {
	if equipped := p.book(p.curNeiGong); equipped != nil {
		return equipped.maxHPAdd, equipped.maxMPAdd
	}
	return 0, 0
}

func newPlayerProgress() playerProgress {
	books := []learnedNeiGong{
		// Values are from the installed modern
		// share/skill/neigong/neigong.ini, not old-server aliases.
		// Two Instruments Heart-Protection Skill: level 30 is a deliberate
		// local-test baseline.  Its progress value is the installed modern
		// wuxuelevelinfo.ini [ng_wd_001] row 30 (273200), not a legacy value.
		// The installed file contains two records with the same QName. The
		// classic record (StaticData 5 / varprop 220130) is the official
		// visible operation-buff route; the mind_ duplicate uses an explicitly
		// hidden buff (Visible=0). Both rows carry the same level-30 stats.
		{configID: "ng_wd_001", staticData: 5, itemType: 1002, level: 30, maxLevel: 36, total: 273200, power: 4550, maxPower: 4550,
			buffID: "buf_ng_wd_001", buffStaticData: 5954, buffLevel: 3,
			strAdd: 45, staAdd: 31, dexAdd: 13, ingAdd: 46, spiAdd: 13,
			maxHPAdd: 1630, maxMPAdd: 490, maxParryAdd: 840, minMagicDefAdd: 45},
		// 移花宫 and its 花无缺 route share 明玉神功.  There is no separate
		// 花无缺 inner-power ConfigID; CS_yhwq_hsqs is the male-version
		// routine. Use the classic visible-buff record and exact level-30 row
		// 810630 rather than its hidden mind_ duplicate.
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
		// Use the classic, visible records here as well.  1006/1008 are the
		// mind_ duplicates: their displayed buff route is explicitly hidden.
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
			sp: 100, str: 30, sta: 30, dex: 30, ing: 30, spi: 30,
			meleePower: 80, magicPower: 80, phyHit: 50, magicHit: 50,
			dodge: 35, maxParry: 100, phyDef: 60, stiDef: 50, negDef: 50,
			masDef: 50, jujDef: 50, minMeleeDamage: 20, maxMeleeDamage: 35,
		},
	}
	// The inline values above are only a no-resource fallback. Every installed
	// modern book is immediately replaced by its StaticData/current-level
	// varprop and PropModify result.
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

// refreshAllModernEffects is intentionally best-effort only for startup
// portability. Production uses the fixed modern-resource directory above;
// retaining the previous values on a missing local package lets unit-level
// protocol work remain usable without replacing an actual game effect with
// zeroes.
func (p *playerProgress) refreshAllModernEffects() {
	for index := range p.books {
		if err := p.books[index].applyModernEffect(); err != nil {
			continue
		}
	}
}

func playerProgressFields() []clientdata.FieldSpec {
	return []clientdata.FieldSpec{
		{Index: propCurNeiGong, Name: "CurNeiGong", Type: clientdata.WireString},
		{Index: propFaculty, Name: "Faculty", Type: clientdata.WireInt32},
		{Index: propFacultyState, Name: "FacultyState", Type: clientdata.WireInt32},
		{Index: propFacultyStyle, Name: "FacultyStyle", Type: clientdata.WireInt32},
		{Index: propFacultyName, Name: "FacultyName", Type: clientdata.WireString},
		{Index: propCurFillValue, Name: "CurFillValue", Type: clientdata.WireInt32},
		{Index: propTotalFillValue, Name: "TotalFillValue", Type: clientdata.WireInt32},
		{Index: propCurLevel, Name: "CurLevel", Type: clientdata.WireInt32},
		{Index: propFillSpeed, Name: "FillSpeed", Type: clientdata.WireInt32},
		{Index: propPowerValue, Name: "PowerValue", Type: clientdata.WireInt32},
		{Index: propMaxPowerValue, Name: "MaxPowerValue", Type: clientdata.WireInt32},
		{Index: propSP, Name: "SP", Type: clientdata.WireInt32},
		{Index: propStr, Name: "Str", Type: clientdata.WireInt32},
		{Index: propSta, Name: "Sta", Type: clientdata.WireInt32},
		{Index: propDex, Name: "Dex", Type: clientdata.WireInt32},
		{Index: propIng, Name: "Ing", Type: clientdata.WireInt32},
		{Index: propSpi, Name: "Spi", Type: clientdata.WireInt32},
		{Index: propMeleePower, Name: "MeleePower", Type: clientdata.WireInt32},
		{Index: propMagicPower, Name: "MagicPower", Type: clientdata.WireInt32},
		{Index: propPhyHit, Name: "PhyHit", Type: clientdata.WireInt32},
		{Index: propMagicHit, Name: "MagicHit", Type: clientdata.WireInt32},
		{Index: propDodge, Name: "Dodge", Type: clientdata.WireInt32},
		{Index: propMaxParry, Name: "MaxParry", Type: clientdata.WireInt32},
		{Index: propPhyDef, Name: "PhyDef", Type: clientdata.WireInt32},
		{Index: propStiDef, Name: "StiDef", Type: clientdata.WireInt32},
		{Index: propNegDef, Name: "NegDef", Type: clientdata.WireInt32},
		{Index: propMasDef, Name: "MasDef", Type: clientdata.WireInt32},
		{Index: propJujDef, Name: "JujDef", Type: clientdata.WireInt32},
		{Index: propMinMeleeDamage, Name: "MinMeleeDamage", Type: clientdata.WireInt32},
		{Index: propMaxMeleeDamage, Name: "MaxMeleeDamage", Type: clientdata.WireInt32},
		{Index: propMaxHPAdd, Name: "MaxHPAdd", Type: clientdata.WireInt32},
		{Index: propMaxMPAdd, Name: "MaxMPAdd", Type: clientdata.WireInt32},
		{Index: propMaxLevel, Name: "MaxLevel", Type: clientdata.WireInt32},
		{Index: propStaticData, Name: "StaticData", Type: clientdata.WireInt32},
		{Index: propItemType, Name: "ItemType", Type: clientdata.WireInt32},
		{Index: propMaxParryAdd, Name: "MaxParryAdd", Type: clientdata.WireInt32},
		{Index: propMinMagicDefAdd, Name: "MinMagicDefAdd", Type: clientdata.WireInt32},
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
	// Apply the currently equipped inner power to the server-owned role
	// state.  The matching BuffStore entry is installed by playerActor; the
	// visible primary attributes must follow the same authoritative source.
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
		viewString(7, book.configID),
		viewInt(propStaticData, book.staticData),
		viewInt(propItemType, book.itemType),
		viewInt(6, book.level),
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
	// form_wuxue_faculty.prop_callback_ConvertState only renders state 0
	// (idle, show the normal-start button) and state 2 (converting).  Sending
	// the client constant READY=1 here leaves the controls in their XML
	// defaults until an 演武 request changes the state to 2, which is why a
	// player had to click 演武 once before 内修 appeared usable.
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
		// 演武 pays out after the card selection.  Advertising a minute rate
		// here caused the server to advance progress even when no card page
		// had been opened.
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

// advanceFaculty settles whole elapsed minutes at the advertised FillSpeed.
// The remainder stays in lastAdvance so a reconnect or restart cannot lose
// partial time.  Both normal cultivation and the active 演武 state consume
// the same authoritative inner-power progress; restore deliberately resets
// an active session's clock, so 演武 accrues only while that live session is
// running and never receives offline progress.
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
		// A level changes its selected varprop, BufferLevel, PropModify packs
		// and SkillModify packs. Re-evaluate before the next 0x10/View update.
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

// awardActiveFaculty settles exactly one 演武 card.  The selection index and
// payment source are validated at the protocol boundary; the local test realm
// currently uses a fixed 20,000-point card result while the runtime is kept
// in active state for the client's two-second card animation/cooldown.
func (p *playerProgress) awardActiveFaculty() (slot uint16, awarded, levelUp, maxLevel bool, err error) {
	if p.facultyState != facultyStateConvert || p.facultyStyle != facultyStyleAct {
		// The current form can leave one delayed SUB_CLIENT_ACT_PAY in its UI
		// queue when the player closes or changes the faculty mode. It is a
		// stale business request, not a malformed transport message, and must
		// not tear down the complete game connection.
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

// S2CFacultyMessage is the numeric C++ GameMessageHandler entry for the
// current table's custom_faculty_msg receiver.  The table uses the layout
// slot = 2*messageID + 15: its entry is slot 397, hence ID 191.  This same
// formula is independently grounded by the live NPC-talk and drama entries:
// slots 101/781 resolve to their working IDs 43/383.
const S2CFacultyMessage int32 = 191

// encodeFacultyMessage goes through the modern numeric GameMessageHandler
// dispatcher.  It resolves ID 191 to custom_handler.custom_faculty_msg, then
// faculty.on_msg(subcommand, arg2).  Sending the Lua function name as a
// string is rejected by the current client.
func encodeFacultyMessage(subcommand int32, arg2 int32) ([]byte, error) {
	return serverCustomIntMessage(S2CFacultyMessage,
		customInt(subcommand),
		customInt(arg2),
	)
}

func encodeFacultyActiveBegin() ([]byte, error) {
	return encodeFacultyMessage(22, 0) // faculty.SUB_SERVER_ACT_BEGIN
}

// handlePlayerProgressCustom accepts client-originated TVarList requests.
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
				// View=43 selection traffic can arrive while the client is changing
				// another wuxue panel.  A non-neigong ConfigID is not a corrupt
				// game session and must never disconnect the player.
				return true, nil
			}
			// Refresh the selected View=43 object before FacultyName reaches
			// the client.  set_control_state immediately resolves the wuxue
			// object and reads Level/MaxLevel; previously this refresh happened
			// only after 1006/21, which is why clicking 演武 once appeared to
			// unlock the normal 内修 button.
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
		case 23: // SUB_CLIENT_ACT_PAY: card index + payment type
			if len(custom.Values) < 4 {
				return true, nil
			}
			// The live modern client emits both choices as TVarList doubles:
			// 1006, 23, double:0, double:1.  They are enum values, not
			// quantities, so accept only exact integral numeric values.
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
				// The faculty panel reads Level/Fill from its View=43 goods object.
				// Send that object first, while the active form is still present;
				// otherwise the close path can leave the panel's default level-one
				// object bound when it reopens.
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
				// The client does not infer a level transition from View=43 alone.
				// faculty.lua maps sub=14 to on_msg_level_up(), which reads the
				// already-synchronized CurLevel and refreshes the panel/notification.
				levelFrame, err := encodeFacultyMessage(14, facultyBool(maxLevel))
				if err != nil {
					return true, err
				}
				if err := link.WriteFrame(levelFrame); err != nil {
					return true, err
				}
			}
			if levelUp || maxLevel {
				// The full authoritative level state is now on the client: first the
				// View=43 book, then role properties, then the native level notice.
				// Only now close the active page.  exit_flag inside the client form
				// still suppresses its stale C2S 1006/22 callback.
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
			// The faculty form binds FacultyName only after the earlier 1006/1
			// selection reply.  Pulse the client property before restoring the
			// authoritative state so set_control_state runs on every open.
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
			// These are client reports/open requests for meridian flows. Their
			// deep business systems are intentionally not faked, but accepting
			// them keeps the shared 1006 bus well-formed.
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
