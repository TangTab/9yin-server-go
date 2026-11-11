package main

// 0x28 v3 生成器——mode=0x46 明文方言（2026-11-11 定版）。
//
// 背景与证据链见 docs/op28-fxnet2-lz-decoder.md：0x47 帧 = LZ 字典压缩流
// （崩溃根因=模板保真段携带池回引 token，客户端会话字典无此槽→NULL 解引用），
// 官方自己在 l.pcapng 用了 6 帧 mode=0x46（纯 memcpy 路径，fxnet2 0x126d）。
// v3 程序化构造明文帧——零 LZ、零字典依赖，任何会话状态下安全。
//
// 帧布局（官方 0x46 帧 #2537/#2855 逐字节定版，tools/op28/parse_template46.py）：
//
//	[0]     0x28 子操作码
//	[1]     0x46 mode（bit0=0 纯拷贝 / bit1=0 单字节长头位形保留 4B 长度槽）
//	[2:6]   len-1 u32（帧长-1，0x47/0x46 族共同不变量）
//	[6:10]  明文体长 u32（=帧长-10；0x47 帧此槽曾被误读为模板号，实为
//	        LZ 解压输出长度，f6=290 系巧合）
//	[10:]   明文体：
//	  [0:4]   obj u32
//	  [4:8]   own u32
//	  [8:60]  变换块 13×f32 槽（52B）：
//	          [8:24]  posA x,y,z,yaw（当前位）
//	          [24:40] posB x,y,z,yaw（目标位；npc.pcapng 走动物化 8/8 帧
//	                  A≠B（B=段末目的地），静立/停留与 A 同值）
//	          [40]    scale（npc.pcapng 非常量：NPC/怪 1.25、玩家形 1.28、
//	                  Boss 形 5.0；服务端无数据源，取 NPC 基线 1.25）
//	          [44]    第二朝向（官方样本 2π 或 0，取 0）
//	          [48:56] 保留零
//	          [56:60] 原始 u32 1（按 f32 读=1.4e-45，非浮点语义）
//	  [60:62] u16 字段数
//	  [62:]   字段×N：[idx u16][值]，宽度由 0x09 属性表列型决定
//	          （f32=4B / u8=1B / u32=4B / 串=[len u32][GBK+NUL]）

import (
	"encoding/binary"
	"math"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// encodeGBK 串值 GBK 编码（与客户端 0x09 表方言一致；x/text 同仓依赖）。
func encodeGBK(text string) ([]byte, error) {
	result, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(text))
	return result, err
}

// npcDetail0x28BaseScale 变换块 scale 槽基线。npc.pcapng（2026-11-11
// 官服 0x46 帧）实测非常量：NPC/怪 1.25、玩家形 1.28、Boss 形 5.0——
// 该槽疑为客户端模型底缩放（xmod 内烘）的服务端回显，模板表 Scale 列
// （"1.2,1.2,1.2" 三元组）与官服槽值无公式关系（boss 模板 1.2 vs 槽
// 5.0），服务端无数据源，故保持 NPC 基线 1.25。
const npcDetail0x28BaseScale = float32(1.25)

// npcDetail0x28V3Frame 生成 mode=0x46 明文 0x28 NPC 详情帧。
// target 非 nil 时 posB=目标位（官服走动物化 A=当前位/B=段末目的地，
// npc.pcapng 8/8 帧均 A≠B；静立/停留 NPC 官服 A=B，传 nil 即同值）。
// props 需按官方语义排好序（见 officialNPCEntityProperties），列型编码
// 以各 Value.Type 为准（officialNPCIntProp 族已按当前表分派）。
func npcDetail0x28V3Frame(objectID, ownerID uint32, transform worldcore.Transform, target *worldcore.Transform, props []clientdata.IndexedProperty) []byte {
	posB := transform
	if target != nil {
		posB = *target
	}
	body := make([]byte, 0, 320)
	body = binary.LittleEndian.AppendUint32(body, objectID)
	body = binary.LittleEndian.AppendUint32(body, ownerID)
	// 变换块：posA=当前位、posB=目标位（静立同值），scale 基线见常量注。
	for _, f := range []float32{transform.X, transform.Y, transform.Z, transform.Orient,
		posB.X, posB.Y, posB.Z, posB.Orient, npcDetail0x28BaseScale, 0, 0, 0} {
		body = binary.LittleEndian.AppendUint32(body, math.Float32bits(f))
	}
	body = binary.LittleEndian.AppendUint32(body, 1) // 原始 u32 标志位，非 f32
	body = binary.LittleEndian.AppendUint16(body, uint16(len(props)))
	for _, prop := range props {
		body = binary.LittleEndian.AppendUint16(body, prop.Index)
		body = appendDetail0x28Value(body, prop.Value)
	}
	frame := make([]byte, 0, len(body)+10)
	frame = append(frame, 0x28, 0x46)
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)+9)) // len-1
	frame = binary.LittleEndian.AppendUint32(frame, uint32(len(body)))   // 明文体长
	frame = append(frame, body...)
	return frame
}

// appendDetail0x28Value 按 0x09 表列型定宽编码值。串=[len u32][GBK+NUL]
// （len 含 NUL——官方 State"walk\0"=5、IDCard 25 实证）。列型为 0（不可
// 编码）或超集线型的属性跳过：字段数以实际写入为准由调用方保证一致。
func appendDetail0x28Value(out []byte, value clientdata.Value) []byte {
	switch value.Type {
	case clientdata.WireByte:
		return append(out, value.U8)
	case clientdata.WireWord:
		return binary.LittleEndian.AppendUint16(out, value.U16)
	case clientdata.WireInt32:
		return binary.LittleEndian.AppendUint32(out, uint32(value.I32))
	case clientdata.WireInt64, clientdata.WireObject:
		return binary.LittleEndian.AppendUint64(out, value.U64)
	case clientdata.WireFloat32:
		return binary.LittleEndian.AppendUint32(out, math.Float32bits(value.F32))
	case clientdata.WireString:
		text := append(gbkBytes(value.Text), 0)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(text)))
		return append(out, text...)
	default:
		return nil
	}
}

// gbkBytes 把串值编成 GBK（NPC 身份串实为 ASCII 路径，非 ASCII 走
// GBK 转码保持与 0x0D 属性帧同一来源方言）。
func gbkBytes(text string) []byte {
	for i := 0; i < len(text); i++ {
		if text[i] > 0x7f {
			encoded, err := encodeGBK(text)
			if err == nil {
				return encoded
			}
			break
		}
	}
	return []byte(text)
}

// officialNPCEntityProperties 从场景实体 schema 属性构造 0x28 v3 字段集，
// 按官方 0x46 样本帧的分形态方言发送（tools/op28/parse_template46.py
// #0/#1 全字段图，2026-11-11 晚补齐）：
//
//	#0 普通NPC（26 字段）：Photo 在场；NpcType 后插 CursorShape、
//	  HPRatio 后插 MPRatio、尾随 HasKarma/Character/KarmaLevel/
//	  OriginPara1-3 官服常量族；
//	#1 带武器NPC（23 字段）：判据=schema WeaponName 非空；CollideRadius
//	  后插 Fixed（schema>0 才发）、WeaponMode 前插 WeaponName、其后
//	  FixedBeHitAct/SelfForceID（schema Force>0 优先，缺省官服 999）；
//	 不发 Photo/CursorShape/MPRatio/karma/origin 族。
//	  BufferInfo1/2 为活 buff 槽 CSV（P4 方言）——物化期 NPC 无 buff，省略。
//
// Level/BossLevel/NpcType/RotatePara/State 取 schema 真值，其余为官服
// 常量——与 officialNPCProperties 同源语义，但面向 sceneEntity 数据形状。
// model 为空时省略 Resource/ConfigID（#0 形连同 Photo）三行。
func officialNPCEntityProperties(schema []clientdata.IndexedProperty, model, configID string) []clientdata.IndexedProperty {
	lookup := func(name string) (clientdata.Value, bool) {
		for _, prop := range schema {
			if prop.Name == name {
				return prop.Value, true
			}
		}
		return clientdata.Value{}, false
	}
	must := clientdata.MustPlayerProperty
	collideRadius, state, propType := must("CollideRadius"), must("State"), must("Type")
	resource, config, photo := must("Resource"), must("ConfigID"), must("Photo")
	rotatePara, npcType, actionSpeed := must("RotatePara"), must("NpcType"), must("ActionSpeed")
	level, bossLevel := must("Level"), must("BossLevel")
	idCard, bornAction, weaponMode := must("IDCard"), must("BornAction"), must("WeaponMode")
	hpRatio, parryRatio, hitHPRatio := must("HPRatio"), must("ParryRatio"), must("HitHPRatio")

	// 形态判据：带武器 NPC 走官方 #1 形（npc 目录 schema 的 WeaponName 列，
	// 默认空串=徒手——npc_resolver.go 默认层）。
	weaponName := schemaString(lookup, "WeaponName", "")
	armed := weaponName != ""

	// CollideRadius 按体型的真值优先（npc.pcapng 0x46 怪物形=0.915/Boss
	// 形=1.0），schema 缺席回落兼容层缺省 0.905。
	collideValue := float32(0.905)
	if v, ok := lookup("CollideRadius"); ok && v.F32 > 0.1 && v.F32 < 16 {
		collideValue = v.F32
	}

	props := []clientdata.IndexedProperty{
		clientdata.IndexedProperty{Index: collideRadius.Index, Name: collideRadius.Name, Value: clientdata.Float32Value(collideValue)},
		clientdata.IndexedProperty{Index: state.Index, Name: state.Name, Value: clientdata.StringValue(schemaString(lookup, "State", "stand"))},
		officialNPCIntProp(propType, 4),
	}
	if armed {
		// 官方 #1：Fixed 紧随 CollideRadius（固定不动标记，schema 0=可动不发）。
		if fixedEntry, ok := clientdata.PlayerProperty("Fixed"); ok {
			if value, has := lookup("Fixed"); has && value.I32 > 0 {
				props = append(props, officialNPCIntProp(fixedEntry, value.I32))
			}
		}
	}
	resourceValue := ""
	if model != "" {
		resourceValue = "npc\\world" + model
	}
	if value, ok := lookup("Resource"); ok && value.Text != "" {
		resourceValue = value.Text
	}
	if resourceValue != "" {
		configValue := configID
		if value, ok := lookup("ConfigID"); ok && value.Text != "" {
			configValue = value.Text
		}
		props = append(props,
			clientdata.IndexedProperty{Index: resource.Index, Name: resource.Name, Value: clientdata.StringValue(resourceValue)},
			clientdata.IndexedProperty{Index: rotatePara.Index, Name: rotatePara.Name,
				Value: clientdata.StringValue(schemaString(lookup, "RotatePara", "0.0,1.000,0.0"))},
			clientdata.IndexedProperty{Index: config.Index, Name: config.Name, Value: clientdata.StringValue(configValue)},
		)
		// Photo 仅 #0 形（官方 #1 带武器样本无 Photo 行）。
		if !armed {
			props = append(props, clientdata.IndexedProperty{Index: photo.Index, Name: photo.Name,
				Value: clientdata.StringValue("icon\\func" + resourceValue + ".png")})
		}
	}
	npcTypeValue := int32(14)
	if value, ok := lookup("NpcType"); ok && value.I32 > 0 {
		npcTypeValue = value.I32
	}
	levelValue := int32(0)
	if value, ok := lookup("Level"); ok && value.I32 > 0 && value.I32 < 256 {
		levelValue = value.I32
	}
	bossLevelValue := int32(1)
	if value, ok := lookup("BossLevel"); ok && value.I32 > 0 && value.I32 < 256 {
		bossLevelValue = value.I32
	}
	// AIEffect 为可选行（官方表@84 在场、fork 表玩家域缺席——真机教训
	// 2026-11-11：MustPlayerProperty 在 fork 表 panic 致会话静默关闭）。
	optional := []clientdata.IndexedProperty(nil)
	if aiEffect, ok := clientdata.PlayerProperty("AIEffect"); ok {
		optional = append(optional, clientdata.IndexedProperty{Index: aiEffect.Index, Name: aiEffect.Name, Value: clientdata.StringValue("nil")})
	}
	props = append(props, officialNPCIntProp(npcType, npcTypeValue))
	if !armed {
		// 官方 #0：CursorShape=2 紧随 NpcType（鼠标形态，schema 真值优先）。
		if entry, ok := clientdata.PlayerProperty("CursorShape"); ok {
			props = append(props, officialNPCIntProp(entry, schemaInt(lookup, "CursorShape", 2)))
		}
	}
	props = append(props, officialNPCFloatProp(actionSpeed, 1.0))
	props = append(props, optional...)
	if levelValue > 0 {
		props = append(props, officialNPCIntProp(level, levelValue))
	}
	props = append(props,
		officialNPCIntProp(bossLevel, bossLevelValue),
		clientdata.IndexedProperty{Index: idCard.Index, Name: idCard.Name, Value: clientdata.StringValue("null,null,null,null,null")},
		clientdata.IndexedProperty{Index: bornAction.Index, Name: bornAction.Name, Value: clientdata.StringValue("stand")},
	)
	if armed {
		// 官方 #1：WeaponName 在 WeaponMode 前（武器模型窄串）。
		if entry, ok := clientdata.PlayerProperty("WeaponName"); ok {
			props = append(props, clientdata.IndexedProperty{Index: entry.Index, Name: entry.Name, Value: clientdata.StringValue(weaponName)})
		}
	}
	weaponModeValue := schemaString(lookup, "WeaponMode", "hand1_1")
	props = append(props, clientdata.IndexedProperty{Index: weaponMode.Index, Name: weaponMode.Name, Value: clientdata.StringValue(weaponModeValue)})
	if armed {
		// 官方 #1：WeaponMode 后 FixedBeHitAct=1（固定受击动作）+ SelfForceID
		//（阵营；schema Force>0 优先，缺省官服样本 999）。
		if entry, ok := clientdata.PlayerProperty("FixedBeHitAct"); ok {
			props = append(props, officialNPCIntProp(entry, 1))
		}
		if entry, ok := clientdata.PlayerProperty("SelfForceID"); ok {
			force := int32(999)
			if value, has := lookup("Force"); has && value.I32 > 0 {
				force = value.I32
			}
			props = append(props, officialNPCIntProp(entry, force))
		}
	}
	props = append(props, officialNPCIntProp(hpRatio, 100))
	if !armed {
		// 官方 #0：MPRatio 紧随 HPRatio（样本值 11）。
		if entry, ok := clientdata.PlayerProperty("MPRatio"); ok {
			props = append(props, officialNPCIntProp(entry, schemaInt(lookup, "MPRatio", 11)))
		}
	}
	props = append(props,
		officialNPCIntProp(parryRatio, 100),
		officialNPCIntProp(hitHPRatio, 100),
	)
	if !armed {
		// 官方 #0 尾部常量族：业报/性格/出身三参数（样本 1/1/3 与 7/1/5）。
		for _, row := range []struct {
			name     string
			fallback int32
		}{
			{"HasKarma", 1}, {"Character", 1}, {"KarmaLevel", 3},
			{"OriginPara1", 7}, {"OriginPara2", 1}, {"OriginPara3", 5},
		} {
			if entry, ok := clientdata.PlayerProperty(row.name); ok {
				props = append(props, officialNPCIntProp(entry, schemaInt(lookup, row.name, row.fallback)))
			}
		}
	}
	return props
}

func schemaString(lookup func(string) (clientdata.Value, bool), name, fallback string) string {
	if value, ok := lookup(name); ok && value.Text != "" {
		return value.Text
	}
	return fallback
}

// schemaInt 整数行 schema 真值优先（>0），缺省回落官服常量。
func schemaInt(lookup func(string) (clientdata.Value, bool), name string, fallback int32) int32 {
	if value, ok := lookup(name); ok && value.I32 > 0 {
		return value.I32
	}
	return fallback
}
