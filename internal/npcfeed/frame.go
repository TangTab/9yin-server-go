package npcfeed

// frame.go 2433 帧配方（cmd/protocol-probe npcAddObject/officialNPC0x28Frame）：
//
// 0x0D：63B 头（obj/own/x/y/z/orient + 36B 保留 + u16 属性数@61）+
//   头行块（OrderedProperties）+ 官方 16-prop 追加块（c.pcapng #1777
//   逐项定版）：CollideRadius（寻路避障关键；宝箱 NpcType=161/采集
//   NpcType=153 不发碰撞体，尾行改交互半径 OpenRange=2.0/GatherRange=3.0）
//   /State=stand/Type=4/Resource+ConfigID+Photo 三件套/ActionSpeed=1.0/
//   Level/BossLevel/IDCard 五空槽/BornAction=stand/WeaponMode=hand1_1/
//   三比例=100。索引一律按名经 clientdata 解析（表漂移免疫）。
//
// 0x28 mode=0x46：官方 19 字段明文形态（npc.pcapng #1841/#1898 罗塞塔），
//   ConfigID/Resource/NpcType/Level/CollideRadius 喂解析真值。

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// npc0x28BaseScale 变换块 scale 槽基线（npc.pcapng 实测非常量：NPC 1.25/
// 玩家形 1.28/Boss 5.0——客户端模型底缩放的服务端回显，保持 NPC 基线）。
const npc0x28BaseScale = float32(1.25)

// Form 物化帧形态。
type Form int

const (
	// FormDual 双发（默认，2433 真机成功会话 56232 形态）：每条 NPC 先发
	// 0x0D 大档（头行块——Name/NpcNickName/Photo/BornAction/ActionSet 等
	// 头顶 UI 与动作驱动字段）再发 0x28 v3（官方建档通道）。
	FormDual Form = iota
	// Form0x28Only 仅 0x28 v3（旧默认）。
	Form0x28Only
	// Form0DOnly 仅裸 0x0D 大档（判别/对照）。
	Form0DOnly
)

// Frames 批量帧出口。每条 NPC 的双发保持 0x0D 在前、0x28 在后
// （同 2433 物化序：AddObject 建档→属性头行跟进）。
func Frames(spawns []Spawn, form Form) ([][]byte, error) {
	frames := make([][]byte, 0, len(spawns)*2)
	for i, s := range spawns {
		if form == FormDual || form == Form0DOnly {
			frame, err := s.Frame0D()
			if err != nil {
				return nil, fmt.Errorf("npcfeed: 条 %d (%s): %w", i, s.ConfigID, err)
			}
			frames = append(frames, frame)
		}
		if form == FormDual || form == Form0x28Only {
			frames = append(frames, s.Frame0x28())
		}
	}
	return frames, nil
}

// Frame0D 单条 0x0D 物化帧。
func (s Spawn) Frame0D() ([]byte, error) {
	if s.ObjectID == 0 || s.OwnerID == 0 {
		return nil, fmt.Errorf("zero ident (%x,%x)", s.ObjectID, s.OwnerID)
	}
	head := s.OrderedProperties()
	official := officialProps(s)
	msg := make([]byte, 63, 64+len(head)*8+len(official)*8)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], s.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], s.OwnerID)
	binary.LittleEndian.PutUint32(msg[9:], math.Float32bits(s.X))
	binary.LittleEndian.PutUint32(msg[13:], math.Float32bits(s.Y))
	binary.LittleEndian.PutUint32(msg[17:], math.Float32bits(s.Z))
	binary.LittleEndian.PutUint32(msg[21:], math.Float32bits(s.Orient))
	for _, prop := range head {
		msg = appendProp(msg, prop)
	}
	for _, prop := range official {
		msg = appendProp(msg, prop)
	}
	binary.LittleEndian.PutUint16(msg[61:], uint16(len(head)+len(official)))
	return msg, nil
}

// Frame0x28 单条 0x28 mode=0x46 明文物化帧——2433 线 npc_detail_v3.go
// 分形态方言（npc.pcapng #0/#1 官方样本逐字节定版，2026-11-11）：
//
//	帧 = [0x28][0x46][u32 帧长-1][u32 体长][体]
//	体 = [obj u32][own u32][12×f32 变换块 posA/posB 同值+scale 1.25]
//	     [u32 1][u16 字段数][值对]
//
//	#0 普通 NPC（WeaponName 空，~26 字段）：CollideRadius/State/Type/
//	  Resource/RotatePara/ConfigID/Photo/NpcType/CursorShape/ActionSpeed/
//	  [AIEffect]/Level/BossLevel/IDCard/BornAction/WeaponMode/HPRatio/
//	  MPRatio/ParryRatio/HitHPRatio/HasKarma/Character/KarmaLevel/
//	  OriginPara1-3（业报/性格/出身常量族——对象建档字段，缺行疑致
//	  客户端建档不完整渲染裁剪）。
//	#1 带武器 NPC（WeaponName 非空，~23 字段）：CollideRadius 后 Fixed
//	  （schema>0 才发）/无 Photo/CursorShape/MPRatio/karma 族/WeaponMode
//	  前 WeaponName、后 FixedBeHitAct/SelfForceID。
//
//	串值=[u32 含NUL][GBK+NUL]（身份串全 ASCII，GBK==原文）；整数值按
//	0x09 表条目线型定宽（byte/word/int32 分派）。
func (s Spawn) Frame0x28() []byte {
	tpl := s.Template
	body := make([]byte, 0, 360)
	body = binary.LittleEndian.AppendUint32(body, s.ObjectID)
	body = binary.LittleEndian.AppendUint32(body, s.OwnerID)
	for _, f := range []float32{s.X, s.Y, s.Z, s.Orient, s.X, s.Y, s.Z, s.Orient, npc0x28BaseScale, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1)
	countAt := len(body)
	body = binary.LittleEndian.AppendUint16(body, 0) // count 占位

	count := 0
	putStr := func(name, val string) {
		body = binary.LittleEndian.AppendUint16(body, clientdata.MustPlayerProperty(name).Index)
		raw := append([]byte(nil), val...)
		raw = append(raw, 0)
		body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
		body = append(body, raw...)
		count++
	}
	putFloat := func(name string, val float32) {
		body = binary.LittleEndian.AppendUint16(body, clientdata.MustPlayerProperty(name).Index)
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(val))
		count++
	}
	// putInt 按表线型定宽（byte/word/int32）——官方 0x46 样本字段宽度
	// 跟表走，硬编码宽度换表即错位。
	putInt := func(name string, val int32) {
		entry := clientdata.MustPlayerProperty(name)
		body = binary.LittleEndian.AppendUint16(body, entry.Index)
		switch entry.Type {
		case clientdata.WireByte:
			body = append(body, byte(val))
		case clientdata.WireWord:
			body = binary.LittleEndian.AppendUint16(body, uint16(val))
		default:
			body = binary.LittleEndian.AppendUint32(body, uint32(val))
		}
		count++
	}
	// tryInt 表内缺席名跳过（fork 表 AIEffect 教训：缺席名 Must 即 panic）。
	tryInt := func(name string, val int32) {
		if _, ok := clientdata.PlayerProperty(name); ok {
			putInt(name, val)
		}
	}
	tplInt := func(name string) int {
		v, err := strconv.Atoi(strings.TrimSpace(tpl[name]))
		if err != nil {
			return 0
		}
		return v
	}

	// 形态判据（2433 v3 同款）：WeaponName 非空=带武器 #1 形。
	weaponName := tpl["WeaponName"]
	armed := weaponName != ""

	// 数值真值（模板列 > 战斗行 > 官方常量基线）。
	npcType := int32(14)
	if v := tplInt("NpcType"); v > 0 {
		npcType = int32(v)
	}
	level := int32(0)
	if s.Combat != nil {
		if lv := s.Combat.Int("Level"); lv > 0 && lv < 256 {
			level = int32(lv)
		}
	}
	if level == 0 {
		if lv := tplInt("Level"); lv > 0 && lv < 256 {
			level = int32(lv)
		}
	}
	bossLevel := int32(1)
	if b := tplInt("BossLevel"); b > 0 && b < 256 {
		bossLevel = int32(b)
	}
	collide := float32(0.905) // 2433 v3 兼容层基线；schema 真值界内透传
	if c, err := strconv.ParseFloat(strings.TrimSpace(tpl["CollideRadius"]), 32); err == nil && c > 0.1 && c < 16 {
		collide = float32(c)
	}
	resource, configID, photo := s.resourceTrio()

	// ---- #0/#1 分形态字段序（官方样本序，位置敏感） ----
	putFloat("CollideRadius", collide)
	if armed {
		// #1：Fixed 紧随 CollideRadius（固定不动标记，0=可动不发）。
		if fixed := tplInt("Fixed"); fixed > 0 {
			tryInt("Fixed", int32(fixed))
		}
	}
	putStr("State", "stand")
	putInt("Type", 4)
	if resource != "" {
		putStr("Resource", resource)
		rotate := tpl["RotatePara"]
		if rotate == "" {
			rotate = "0.0,1.000,0.0" // 2433 v3 缺省（非零中轴）
		}
		putStr("RotatePara", rotate)
		putStr("ConfigID", configID)
		if !armed {
			// Photo 仅 #0 形（官方 #1 带武器样本无 Photo 行）。
			putStr("Photo", photo)
		}
	}
	putInt("NpcType", npcType)
	if !armed {
		// #0：CursorShape=2 紧随 NpcType（鼠标形态，schema 真值优先）。
		cursor := int32(2)
		if v := tplInt("CursorShape"); v > 0 {
			cursor = int32(v)
		}
		tryInt("CursorShape", cursor)
	}
	putFloat("ActionSpeed", 1.0)
	// AIEffect 可选行（官方表在场才发；fork 表玩家域缺席——真机教训
	// 2026-11-11：Must 在缺席方言 panic 致会话静默关闭）。
	if _, ok := clientdata.PlayerProperty("AIEffect"); ok {
		putStr("AIEffect", "nil")
	}
	if level > 0 {
		putInt("Level", level)
	}
	putInt("BossLevel", bossLevel)
	putStr("IDCard", "null,null,null,null,null")
	bornAction := tpl["BornAction"]
	if bornAction == "" {
		bornAction = "stand"
	}
	putStr("BornAction", bornAction)
	if armed {
		// #1：WeaponName 在 WeaponMode 前（武器模型窄串）。
		if _, ok := clientdata.PlayerProperty("WeaponName"); ok {
			putStr("WeaponName", weaponName)
		}
	}
	weaponMode := tpl["WeaponMode"]
	if weaponMode == "" {
		weaponMode = "hand1_1"
	}
	putStr("WeaponMode", weaponMode)
	if armed {
		// #1：WeaponMode 后 FixedBeHitAct=1（固定受击动作）+ SelfForceID
		//（阵营；模板 Force>0 优先，缺省外服私服样本 999）。
		tryInt("FixedBeHitAct", 1)
		force := int32(999)
		if v := tplInt("Force"); v > 0 {
			force = int32(v)
		}
		tryInt("SelfForceID", force)
	}
	putInt("HPRatio", 100)
	if !armed {
		// #0：MPRatio 紧随 HPRatio（样本值 11）。
		mpRatio := int32(11)
		if v := tplInt("MPRatio"); v > 0 {
			mpRatio = int32(v)
		}
		tryInt("MPRatio", mpRatio)
	}
	putInt("ParryRatio", 100)
	putInt("HitHPRatio", 100)
	if !armed {
		// #0 尾部常量族：业报/性格/出身三参数（样本 1/1/3 与 7/1/5）。
		for _, row := range []struct {
			name     string
			fallback int32
		}{
			{"HasKarma", 1}, {"Character", 1}, {"KarmaLevel", 3},
			{"OriginPara1", 7}, {"OriginPara2", 1}, {"OriginPara3", 5},
		} {
			val := row.fallback
			if v := tplInt(row.name); v > 0 {
				val = int32(v)
			}
			tryInt(row.name, val)
		}
	}
	binary.LittleEndian.PutUint16(body[countAt:], uint16(count))

	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))
	return append(frame, body...)
}

// resourceTrio Resource/ConfigID/Photo 三件套：模板 Resource 直用 /
// Material 列拼 npc\world 前缀族两分支（2433 官方形）；Photo 优先模板
// 真值，缺席按外服规则派生；ConfigID 模板列 > 摆放 id 兜底。
func (s Spawn) resourceTrio() (resource, configID, photo string) {
	tpl := s.Template
	configID = tpl["ConfigID"]
	if r := tpl["Resource"]; r != "" {
		resource = r
		if configID == "" {
			configID = s.ConfigID
		}
		photo = tpl["Photo"]
		if photo == "" {
			photo = `icon\func` + r + ".png"
		}
	} else if m := tpl["Material"]; m != "" {
		resource = `npc\world` + m
		configID = m
		photo = tpl["Photo"]
		if photo == "" {
			photo = `icon\func` + resource + ".png"
		}
	}
	if configID == "" {
		configID = s.ConfigID // 兜底摆放 id
	}
	return resource, configID, photo
}

// officialProps 0x0D 官方 16-prop 追加块（按名解析索引；宝箱/采集分族方言）。
func officialProps(s Spawn) []Prop {
	collide := clientdata.MustPlayerProperty("CollideRadius")
	openRange := clientdata.MustPlayerProperty("OpenRange")
	gatherRange := clientdata.MustPlayerProperty("GatherRange")
	state := clientdata.MustPlayerProperty("State")
	propType := clientdata.MustPlayerProperty("Type")
	resource := clientdata.MustPlayerProperty("Resource")
	configID := clientdata.MustPlayerProperty("ConfigID")
	photo := clientdata.MustPlayerProperty("Photo")
	actionSpeed := clientdata.MustPlayerProperty("ActionSpeed")
	level := clientdata.MustPlayerProperty("Level")
	bossLevel := clientdata.MustPlayerProperty("BossLevel")
	idCard := clientdata.MustPlayerProperty("IDCard")
	bornAction := clientdata.MustPlayerProperty("BornAction")
	weaponMode := clientdata.MustPlayerProperty("WeaponMode")
	hpRatio := clientdata.MustPlayerProperty("HPRatio")
	parryRatio := clientdata.MustPlayerProperty("ParryRatio")
	hitHPRatio := clientdata.MustPlayerProperty("HitHPRatio")

	tpl := s.Template
	tplInt := func(name string) int {
		v, err := strconv.Atoi(strings.TrimSpace(tpl[name]))
		if err != nil {
			return 0
		}
		return v
	}
	resourceVal, configIDVal, photoVal := s.resourceTrio()
	npcTypeVal := int32(0)
	if v := tplInt("NpcType"); v > 0 {
		npcTypeVal = int32(v)
	}
	levelVal := int32(0)
	if s.Combat != nil {
		if lv := s.Combat.Int("Level"); lv > 0 && lv < 256 {
			levelVal = int32(lv)
		}
	}
	if levelVal == 0 {
		if lv := tplInt("Level"); lv > 0 && lv < 256 {
			levelVal = int32(lv)
		}
	}
	bossLevelVal := int32(1)
	if b := tplInt("BossLevel"); b > 0 && b < 256 {
		bossLevelVal = int32(b)
	}
	collideVal := float32(0.9)
	if c, err := strconv.ParseFloat(strings.TrimSpace(tpl["CollideRadius"]), 32); err == nil && c > 0.1 && c < 16 {
		collideVal = float32(c)
	}

	boxKind := npcTypeVal == 161 || strings.HasPrefix(configIDVal, "BoxNpc")
	gatherKind := npcTypeVal == 153 || strings.HasPrefix(configIDVal, "GatherClone")

	props := make([]Prop, 0, 20)
	if !boxKind && !gatherKind {
		props = append(props, Prop{Index: collide.Index, Name: collide.Name, Value: floatVal(collideVal)})
	}
	props = append(props,
		Prop{Index: state.Index, Name: state.Name, Value: stringVal("stand")},
		intProp(propType, 4), // 外服 #1777 Type=4（NPC 对象类别）
	)
	if resourceVal != "" {
		props = append(props,
			Prop{Index: resource.Index, Name: resource.Name, Value: stringVal(resourceVal)},
			Prop{Index: configID.Index, Name: configID.Name, Value: stringVal(configIDVal)},
		)
		if photoVal != "" {
			props = append(props, Prop{Index: photo.Index, Name: photo.Name, Value: stringVal(photoVal)})
		}
	}
	props = append(props, floatProp(actionSpeed, 1.0))
	if levelVal > 0 {
		props = append(props, intProp(level, levelVal))
	}
	props = append(props,
		intProp(bossLevel, bossLevelVal),
		// 外服 #1777 IDCard 五槽空身份卡。
		Prop{Index: idCard.Index, Name: idCard.Name, Value: stringVal("null,null,null,null,null")},
		Prop{Index: bornAction.Index, Name: bornAction.Name, Value: stringVal("stand")},
		Prop{Index: weaponMode.Index, Name: weaponMode.Name, Value: stringVal("hand1_1")},
		intProp(hpRatio, 100),
		intProp(parryRatio, 100),
		intProp(hitHPRatio, 100),
	)
	// 宝箱/采集族尾行=交互半径（npc.pcapng 定版 2.0/3.0）。
	if boxKind {
		props = append(props, floatProp(openRange, 2.0))
	} else if gatherKind {
		props = append(props, floatProp(gatherRange, 3.0))
	}
	return props
}

// intProp 按表线型分派整数值（byte/word/int32）。
func intProp(entry clientdata.PropertyEntry, value int32) Prop {
	switch entry.Type {
	case clientdata.WireByte:
		return Prop{Index: entry.Index, Name: entry.Name, Value: byteVal(uint8(value))}
	case clientdata.WireWord:
		return Prop{Index: entry.Index, Name: entry.Name, Value: wordVal(uint16(value))}
	default:
		return Prop{Index: entry.Index, Name: entry.Name, Value: int32Val(value)}
	}
}

func floatProp(entry clientdata.PropertyEntry, value float32) Prop {
	return Prop{Index: entry.Index, Name: entry.Name, Value: floatVal(value)}
}

// appendProp 按线型追加 [idx u16][值]。
func appendProp(msg []byte, prop Prop) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, prop.Index)
	switch prop.Value.Type {
	case clientdata.WireByte:
		return append(msg, prop.Value.U8)
	case clientdata.WireWord:
		return binary.LittleEndian.AppendUint16(msg, prop.Value.U16)
	case clientdata.WireInt32:
		return binary.LittleEndian.AppendUint32(msg, uint32(prop.Value.I32))
	case clientdata.WireInt64:
		return binary.LittleEndian.AppendUint64(msg, prop.Value.U64)
	case clientdata.WireFloat32:
		return binary.LittleEndian.AppendUint32(msg, math.Float32bits(prop.Value.F32))
	case clientdata.WireString:
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len(prop.Value.Text)+1))
		msg = append(msg, prop.Value.Text...)
		return append(msg, 0)
	case clientdata.WireWideString:
		units := utf16.Encode([]rune(prop.Value.Text))
		msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
		for _, unit := range units {
			msg = binary.LittleEndian.AppendUint16(msg, unit)
		}
		return binary.LittleEndian.AppendUint16(msg, 0)
	case clientdata.WireObject:
		return binary.LittleEndian.AppendUint64(msg, prop.Value.U64)
	default:
		return msg
	}
}
