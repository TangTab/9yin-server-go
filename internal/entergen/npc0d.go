package entergen

// npc0d.go NPC 0x0D 物化帧生成器——fork 系客户端的 NPC 渲染通道
// （m229 在该客户端=EscortRec 记录帧，静默不渲染；2026-11-22 真机
// trace.log 零痕迹定案，与 2433 线《外服模式NPC渲染收束》终局一致）。
//
// 配方移植自 2433 线 cmd/protocol-probe npcAddObject（真机渲染成功会话
// 56232：hybrid 头行 0x0D 83 字段；official-2433-only 线配 0x09 表尾追
// 61 行后头行通道同构复活——本仓表=2433+61 尾追 2494 已就绪）：
//
//	帧 = 63B 头（与玩家 AppearanceFrame 同构：obj/own/x/y/z/orient +
//	     36B 保留 + u16 属性数）+ 属性区两块：
//	  ① 头行块（按 0x09 表序）：默认层（VisibleNPCSpawnDefaultsCompatV1
//	     值集经场景域过滤）→ 模板层（NpcConfig 原始行同名绑定）→
//	     摆放层（PosiX/Y/Z/Orient）；
//	  ② 官方 16-prop 块（c.pcapng #1777 逐项定版）：CollideRadius
//	     （寻路避障关键，宝箱/采集族不发）/State=stand/Type=4（NPC 对象
//	     类别常量）/Resource+ConfigID+Photo 三件套/ActionSpeed=1.0/
//	     Level/BossLevel=1/IDCard 五空槽/BornAction=stand/
//	     WeaponMode=hand1_1/三比例=100/宝箱 OpenRange=2.0 采集
//	     GatherRange=3.0。
//
// ident（2433 线真机形态）：obj=0x11000000|idx（fork NPC 段）、
// own=0x00660000|低16位（NPC owner 私服化段——owner=1 在部分客户端
// 被解析为「1 号玩家化身」触发 PvP 分支）。

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// NpcObject0D 一条待物化 NPC（0x0D 通道输入；模板行/Raw 来自 npcdata）。
type NpcObject0D struct {
	ObjectID uint32
	OwnerID  uint32
	X, Y, Z  float32
	Orient   float32
	ConfigID string            // 摆放 id（=模板联查键）
	Template map[string]string // NpcConfig 原始行（按名取 Resource/Level/...）
}

// NpcForkObjectID / NpcForkOwnerID 2433 线 NPC ident 公式（场景内 idx 1 起）。
func NpcForkObjectID(index int) uint32 { return 0x11000000 | uint32(index) }

func NpcForkOwnerID(objectID uint32) uint32 { return 0x00660000 | (objectID & 0xFFFF) }

// ---- 0x28 mode=0x46 明文物化帧（官方家族客户端建对象通道）----

// npc0x28BaseScale 变换块 scale 槽基线（npc.pcapng 实测非常量：NPC 1.25/
// 玩家形 1.28/Boss 5.0——疑为客户端模型底缩放的服务端回显，服务端无
// 数据源，保持 NPC 基线 1.25）。
const npc0x28BaseScale = float32(1.25)

// NpcMaterializeFrames NPC 物化帧族：form=true（默认）走 0x28 mode=0x46
// 官方言（罗塞塔=npc.pcapng #1841/#1898 官方自用 0x46 建对象帧 19 字段
// 逐字节定版）；form=false 判别期切回裸 0x0D 大档（官方客户端不建对象
// ——2433 线 15:28 实证，fork 家族客户端才吃）。
func NpcMaterializeFrames(entries []NpcObject0D, op28Form bool) ([][]byte, error) {
	frames := make([][]byte, 0, len(entries))
	for i, e := range entries {
		var frame []byte
		var err error
		if op28Form {
			frame = NpcObject0x28Frame(e)
		} else {
			frame, err = NpcObject0DFrame(e)
			if err != nil {
				return nil, fmt.Errorf("npc entry %d (%s): %w", i, e.ConfigID, err)
			}
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// NpcObject0x28Frame 0x28 mode=0x46 明文物化帧（op28_addobject.go
// officialNPC0x28Frame 移植；idx 硬编码按 2433 表官方位——2433 线
// 「值对仍按当前 0x09 表官方位」同表验证过）。
//
//	帧 = [0x28][0x46][u32 帧长-1][u32 体长][体]
//	体 = [obj u32][own u32][12×f32 变换块 posA/posB 同值（静立 A=B，
//	     posB=0 客户端记「走向原点」）+scale][u32 1][u16 字段数][值对]
//	值对 = [idx u16][0x09 表线型定宽值]，串=[u32 len 含NUL][GBK+NUL]
//	       （Resource/ConfigID/Photo 全 ASCII，GBK 即原文）。
func NpcObject0x28Frame(e NpcObject0D) []byte {
	resource := e.Template["Resource"]
	if resource == "" {
		resource = e.Template["Material"]
	}
	configID := e.Template["ConfigID"]
	if configID == "" {
		configID = e.ConfigID
	}
	npcType := int32(14) // 官方怪通用缺省（61~69=头顶功能图标族，45=忽略）
	if t, err := strconv.Atoi(strings.TrimSpace(e.Template["NpcType"])); err == nil && t > 0 {
		npcType = int32(t)
	}
	level := byte(25) // 官方样本静态值；目录真值界内透传
	if l, err := strconv.Atoi(strings.TrimSpace(e.Template["Level"])); err == nil && l > 0 && l < 256 {
		level = byte(l)
	}

	body := make([]byte, 0, 320)
	body = binary.LittleEndian.AppendUint32(body, e.ObjectID)
	body = binary.LittleEndian.AppendUint32(body, e.OwnerID)
	for _, f := range []float32{e.X, e.Y, e.Z, e.Orient, e.X, e.Y, e.Z, e.Orient, npc0x28BaseScale, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1)
	countAt := len(body)
	body = binary.LittleEndian.AppendUint16(body, 0)

	count := 0
	put := func(idx uint16, val any) {
		body = binary.LittleEndian.AppendUint16(body, idx)
		switch v := val.(type) {
		case float32:
			body = binary.LittleEndian.AppendUint32(body, math.Float32bits(v))
		case uint32:
			body = binary.LittleEndian.AppendUint32(body, v)
		case int32:
			body = binary.LittleEndian.AppendUint32(body, uint32(v))
		case byte:
			body = append(body, v)
		case string:
			raw := append([]byte(nil), v...)
			raw = append(raw, 0)
			body = binary.LittleEndian.AppendUint32(body, uint32(len(raw)))
			body = append(body, raw...)
		}
		count++
	}
	// 官方 0x46 可建帧 19 字段定版（npc.pcapng #1841/#1898 明文走查）。
	put(179, float32(0.915))  // CollideRadius
	put(183, "stand")         // State
	put(181, byte(4))         // Type=4（NPC 对象类别）
	if resource != "" {
		put(1448, resource) // Resource
	}
	put(1565, "0.0,0.0,0.0") // RotatePara（静态零）
	put(1447, configID)      // ConfigID
	photoRoot := resource
	if photoRoot == "" {
		photoRoot = configID
	}
	put(184, `icon\funcnpc\`+photoRoot+".png") // Photo
	put(1566, uint32(npcType))                  // NpcType
	put(214, float32(1.0))                      // ActionSpeed
	put(1596, uint32(1))                        // NotPositive（白名）
	put(84, "nil")                              // AIEffect
	put(1472, level)                            // Level
	put(1597, byte(1))                          // BossLevel
	put(1602, "null,null,null,null,null")       // IDCard
	put(1604, "stand")                          // BornAction
	put(1606, "hand1_1")                        // WeaponMode
	put(471, uint32(100))                       // HPRatio
	put(480, uint32(100))                       // ParryRatio
	put(484, uint32(100))                       // HitHPRatio
	binary.LittleEndian.PutUint16(body[countAt:], uint16(count))

	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))
	return append(frame, body...)
}

// NpcObject0DFrames 每条 NPC 一帧 0x0D。
func NpcObject0DFrames(entries []NpcObject0D) ([][]byte, error) {
	frames := make([][]byte, 0, len(entries))
	for i, e := range entries {
		frame, err := NpcObject0DFrame(e)
		if err != nil {
			return nil, fmt.Errorf("npc0d entry %d (%s): %w", i, e.ConfigID, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// NpcObject0DFrame 单条 NPC 0x0D 物化帧。
func NpcObject0DFrame(e NpcObject0D) ([]byte, error) {
	if e.ObjectID == 0 || e.OwnerID == 0 {
		return nil, fmt.Errorf("zero ident (%x,%x)", e.ObjectID, e.OwnerID)
	}
	headProps, err := npc0dHeadProperties(e)
	if err != nil {
		return nil, err
	}
	official := officialNPC0DProperties(e.Template)
	msg := make([]byte, 63, 64+len(headProps)*8+len(official)*8)
	msg[0] = 0x0D
	putU32(msg, 1, e.ObjectID)
	putU32(msg, 5, e.OwnerID)
	putU32(msg, 9, math.Float32bits(e.X))
	putU32(msg, 13, math.Float32bits(e.Y))
	putU32(msg, 17, math.Float32bits(e.Z))
	putU32(msg, 21, math.Float32bits(e.Orient))
	for _, prop := range headProps {
		msg = appendProperty(msg, prop)
	}
	for _, prop := range official {
		msg = appendProperty(msg, prop)
	}
	putU16(msg, 61, uint16(len(headProps)+len(official)))
	return msg, nil
}

// npc0dHeadProperties 头行块：默认层 → 模板层 → 摆放层，按 0x09 表
// （场景域首现序）合成——2433 线 ResolvedNPC.OrderedProperties 同构。
func npc0dHeadProperties(e NpcObject0D) ([]IndexedProperty, error) {
	props := map[string]PropertyValue{}
	// 默认层（VisibleNPCSpawnDefaultsCompatV1 值集；按场景域过滤缺席名
	// 与线型不符名）。
	for name, value := range npc0dDefaults {
		if entry, ok := clientdata.SceneProperty(name); ok && wireKindOf(entry) == value.kind {
			props[name] = value
		}
	}
	// 模板层：原始行同名列绑定（ConfigID→ID 别名、ID 结构列不直连）。
	// 真档含 Excel 脏值（如 WalkSpeed="#N/A"）——单字段解析失败跳过
	//（默认层兜底），不废整条。
	for _, field := range clientdata.OfficialSceneFields() {
		if field.Name == "ID" {
			continue
		}
		column := field.Name
		if field.Name == "ConfigID" {
			column = "ID"
		}
		raw, ok := e.Template[column]
		if !ok || raw == "" {
			continue
		}
		value, err := parseSceneValue(field, raw)
		if err != nil {
			continue
		}
		props[field.Name] = value
	}
	// 摆放层：坐标四行（尾追表 PosiX@2434+/Orient——SceneProperty 有则注入）。
	placement := map[string]PropertyValue{
		"PosiX": Float32Value(e.X),
		"PosiY": Float32Value(e.Y),
		"PosiZ": Float32Value(e.Z),
		"Orient": Float32Value(e.Orient),
	}
	for name, value := range placement {
		if entry, ok := clientdata.SceneProperty(name); ok && wireKindOf(entry) == value.kind {
			props[name] = value
		}
	}
	// 按表序输出。
	out := make([]IndexedProperty, 0, len(props))
	for _, field := range clientdata.OfficialSceneFields() {
		if value, ok := props[field.Name]; ok {
			out = append(out, IndexedProperty{Index: field.Index, Name: field.Name, Value: value})
		}
	}
	return out, nil
}

// officialNPC0DProperties 官方 16-prop 追加块（c.pcapng #1777/#1789 逐项
// 定版；宝箱/采集分族方言 npc.pcapng）。
func officialNPC0DProperties(tpl map[string]string) []IndexedProperty {
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

	// CollideRadius：优先模板真值（0.5..16 体感界），回落 0.9 基线。
	collideValue := float32(0.9)
	if c, err := strconv.ParseFloat(strings.TrimSpace(tpl["CollideRadius"]), 32); err == nil &&
		c > 0.1 && c < 16 {
		collideValue = float32(c)
	}
	// 分族判据：NpcType 列优先，ConfigID 前缀兜底。
	npcTypeValue := 0
	if t, err := strconv.Atoi(strings.TrimSpace(tpl["NpcType"])); err == nil {
		npcTypeValue = t
	}
	configIDValue := tpl["ConfigID"]
	if configIDValue == "" {
		configIDValue = tpl["Material"]
	}
	boxKind := npcTypeValue == 161 || strings.HasPrefix(configIDValue, "BoxNpc")
	gatherKind := npcTypeValue == 153 || strings.HasPrefix(configIDValue, "GatherClone")

	props := make([]IndexedProperty, 0, 20)
	if !boxKind && !gatherKind {
		props = append(props, IndexedProperty{Index: collide.Index, Name: collide.Name,
			Value: Float32Value(collideValue)})
	}
	props = append(props,
		IndexedProperty{Index: state.Index, Name: state.Name, Value: StringValue("stand")},
		officialNPC0DIntProp(propType, 4), // 外服 #1777 Type=4（NPC 对象类别）
	)
	// Resource 三件套：模板 Material 列拼 npc\world 前缀族 / Resource 直用
	// 两分支（2433 官方形）；Photo 按外服规则拼接派生。
	if material := tpl["Material"]; material != "" {
		resourceValue := `npc\world` + material
		props = append(props,
			IndexedProperty{Index: resource.Index, Name: resource.Name, Value: StringValue(resourceValue)},
			IndexedProperty{Index: configID.Index, Name: configID.Name, Value: StringValue(material)},
			IndexedProperty{Index: photo.Index, Name: photo.Name,
				Value: StringValue(`icon\func` + resourceValue + ".png")},
		)
	} else if r := tpl["Resource"]; r != "" {
		configValue := r
		if c := tpl["ConfigID"]; c != "" {
			configValue = c
		}
		props = append(props,
			IndexedProperty{Index: resource.Index, Name: resource.Name, Value: StringValue(r)},
			IndexedProperty{Index: configID.Index, Name: configID.Name, Value: StringValue(configValue)},
			IndexedProperty{Index: photo.Index, Name: photo.Name,
				Value: StringValue(`icon\func` + r + ".png")},
		)
	}
	props = append(props, officialNPC0DFloatProp(actionSpeed, 1.0))
	if lv, err := strconv.Atoi(strings.TrimSpace(tpl["Level"])); err == nil && lv > 0 && lv < 256 {
		props = append(props, officialNPC0DIntProp(level, int32(lv)))
	}
	bossLevelValue := int32(1) // 外服 #1777 BossLevel=1（普通怪基线）
	if b, err := strconv.Atoi(strings.TrimSpace(tpl["BossLevel"])); err == nil && b > 0 && b < 256 {
		bossLevelValue = int32(b)
	}
	props = append(props,
		officialNPC0DIntProp(bossLevel, bossLevelValue),
		// 外服 #1777 IDCard 五槽空身份卡。
		IndexedProperty{Index: idCard.Index, Name: idCard.Name,
			Value: StringValue("null,null,null,null,null")},
		IndexedProperty{Index: bornAction.Index, Name: bornAction.Name, Value: StringValue("stand")},
		IndexedProperty{Index: weaponMode.Index, Name: weaponMode.Name, Value: StringValue("hand1_1")},
		officialNPC0DIntProp(hpRatio, 100),
		officialNPC0DIntProp(parryRatio, 100),
		officialNPC0DIntProp(hitHPRatio, 100),
	)
	// 宝箱/采集族尾行=交互半径（npc.pcapng 定版 2.0/3.0）。
	if boxKind {
		props = append(props, officialNPC0DFloatProp(openRange, 2.0))
	} else if gatherKind {
		props = append(props, officialNPC0DFloatProp(gatherRange, 3.0))
	}
	return props
}

// npc0dDefaults 默认层值集（2433 VisibleNPCSpawnDefaultsCompatV1 直抄；
// 装配时按场景域过滤缺席名/线型不符名）。
var npc0dDefaults = map[string]PropertyValue{
	"Type":             ByteValue(4),
	"Sex":              ByteValue(0),
	"ActionSet":        StringValue(""),
	"State":            StringValue("stand"),
	"LogicState":       ByteValue(0),
	"HPRatio":          Int32Value(100),
	"MaxHP":            Int32Value(1000),
	"HP":               Int32Value(1000),
	"Scale":            StringValue(""),
	"CollideType":      Int32Value(0),
	"CantShowHeadInfo": ByteValue(0),
	"NPCCanSelect":     WordValue(1),
	"RotatePara":       StringValue(""),
	"NeedRotate":       Int32Value(0),
	"Collide":          Int32Value(0),
	"CollideRadius":    Float32Value(0.905),
	"CantAttack":       Int32Value(0),
	"CantBeAttack":     Int32Value(0),
	"Force":            Int32Value(0),
	"MotionNoRotate":   ByteValue(0),
	"CanPick":          ByteValue(0),
	"VisDataInt":       Int32Value(0),
	"VisDataStr":       WideValue(""),
	"HeadEffect":       Int32Value(0),
	"NpcNickName":      WideValue(""),
	"RoleHide":         Int32Value(0),
	"SpringEffect":     StringValue(""),
	"Material":         StringValue(""),
	"WeaponName":       StringValue(""),
	"BornAction":       StringValue(""),
	"ActionSoundMode":  Int32Value(0),
	"Fixed":            Int32Value(0),
	"FixedBeHitAct":    StringValue(""),
	"DirectSpring":     Int32Value(0),
	"NpcTalkType":      Int32Value(0),
	"PreLoad":          Int32Value(0),
	"OnlyCareDistance": Int32Value(0),
	"RefreshAtOnce":    Int32Value(0),
}

// wireKindOf 表线型 → PropertyValue kind（过滤比对用）。
func wireKindOf(entry clientdata.PropertyEntry) byte {
	switch entry.Type {
	case clientdata.WireByte:
		return 1
	case clientdata.WireWord:
		return 2
	case clientdata.WireInt32:
		return 3
	case clientdata.WireInt64:
		return 4
	case clientdata.WireFloat32:
		return 5
	case clientdata.WireString:
		return 7
	case clientdata.WireWideString:
		return 8
	case clientdata.WireObject:
		return 9
	}
	return 0
}

// parseSceneValue 按表线型解析模板/摆放原始串。
func parseSceneValue(field clientdata.PropertyEntry, raw string) (PropertyValue, error) {
	text := strings.TrimSpace(raw)
	switch field.Type {
	case clientdata.WireByte:
		v, err := strconv.ParseUint(text, 10, 8)
		if err != nil {
			return PropertyValue{}, err
		}
		return ByteValue(byte(v)), nil
	case clientdata.WireWord:
		v, err := strconv.ParseUint(text, 10, 16)
		if err != nil {
			return PropertyValue{}, err
		}
		return WordValue(uint16(v)), nil
	case clientdata.WireInt32:
		v, err := parseIntegralInt32(text)
		if err != nil {
			return PropertyValue{}, err
		}
		return Int32Value(v), nil
	case clientdata.WireFloat32:
		v, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return PropertyValue{}, err
		}
		f := float32(v)
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			return PropertyValue{}, fmt.Errorf("not a finite float32: %q", raw)
		}
		return Float32Value(f), nil
	case clientdata.WireString:
		return StringValue(raw), nil
	case clientdata.WireWideString:
		return WideValue(raw), nil
	default:
		return PropertyValue{}, fmt.Errorf("unsupported wire type %d for %s", field.Type, field.Name)
	}
}

func parseIntegralInt32(text string) (int32, error) {
	if v, err := strconv.ParseInt(text, 10, 32); err == nil {
		return int32(v), nil
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v ||
		v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("%q is not an integral int32", text)
	}
	return int32(v), nil
}

// officialNPC0DIntProp 按表线型分派整数值（2433 officialNPCIntProp 同构）。
func officialNPC0DIntProp(entry clientdata.PropertyEntry, value int32) IndexedProperty {
	switch entry.Type {
	case clientdata.WireByte:
		return IndexedProperty{Index: entry.Index, Name: entry.Name, Value: ByteValue(byte(value))}
	case clientdata.WireWord:
		return IndexedProperty{Index: entry.Index, Name: entry.Name, Value: WordValue(uint16(value))}
	default:
		return IndexedProperty{Index: entry.Index, Name: entry.Name, Value: Int32Value(value)}
	}
}

func officialNPC0DFloatProp(entry clientdata.PropertyEntry, value float32) IndexedProperty {
	return IndexedProperty{Index: entry.Index, Name: entry.Name, Value: Float32Value(value)}
}
