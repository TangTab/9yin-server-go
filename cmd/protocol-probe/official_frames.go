package main

// 官服帧级定版 builders（f/g/b.pcapng 2026-11-08 帧级实证，见
// docs/冲锋与打坐官服抓包分析及Go移植.md §3.2.2）：
//
//   0x13 ServerRecordGrid 逐格写：[13][00][obj][owner][rec u16][nrows u16=1]
//          [row u16][col u8][按 schema 列型的单格值]
//   0x11 ServerRecordAdd 空行：官服 CD 行先铺零行、再逐格 0x13 填充
//   rec114 fight_relation（schema [obj,word]）：行加 0x11 / 逐击 0x13 行0列1(u16) /
//          撤销 0x12 行0（g/f 击杀簇与直收簇实证）
//   0x2D 多属性帧：[2D][01][00][obj][owner][n u16]([idx u16][i32])×n
//          （官服 NPC 血量刷新通道 HPRatio/HitHPRatio；Go 旧 28/112 形态已废弃）
//   msg24 c1=10 怒气段序 / c1=29 击退（f 44.539、g 39.072/56.468 击杀簇实证）
//   msg284 空值标记帧（g 直收簇 CurSkillTarget 后恒发，语义未定位，逐位镜像）
//
// CD 行方言（2026-11-09 真机定版）：现行=**行0 整行**（serverRecordAddCells
// 一帧带全 4 列），native-menu 反编译同款真机转圈双验；官服抓包方言
//（0x11 空行行97个人/98公共 + 0x13 逐格 + 行3=施法窗 1s 非 CD）帧与
// g 14.660 逐字节同构仍不转圈，客户端消费前置待考证（疑登录期视图
// 绑定），官服槽号与逐格 builders 保留在库供考证启用：
//   个人行 97：col0 类别 / col1 起 / col2 止 / col3 0xFFFF；铺完重写行3 施法窗
//   公共行 98：col0=50000 / col3=CoolDownTeam；无行3 对
//   过期行在下一次受理时惰性 0x12 清扫

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

const (
	recordColumnObject byte = 9

	// 官服 CD 行双槽（f/g 玩家实证 97/98；b 玩家 93/94——槽号因人而异，
	// 按最常见固定对实现，客户端按 rec/行号镜像渲染，不参与业务判定）。
	cooldownRowPersonal uint16 = 97
	cooldownRowPublic   uint16 = 98
	// cooldownRowWindow：官服在个人行铺完后重写的「当前施法窗」行
	//（f 42.948 / g 22.157 个人行 col1/col2 后各跟一对行3 写，语义未定位）。
	cooldownRowWindow uint16 = 3
)

// recordObjectCell 是 recordCell 的 obj 对（rec114 col0）变体。
func recordObject(id, owner uint32) recordCell {
	value := uint64(id) | uint64(owner)<<32
	return recordCell{int64Value: &[]int64{int64(value)}[0], objectCell: true}
}

// serverRecordAddEmptyRow 铺一行全零的 0x11（官服 CD 行先加空行再逐格填）。
func serverRecordAddEmptyRow(objectID, ownerID uint32, recordIndex, row uint16) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	msg := make([]byte, 16, 16+32)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], row)
	binary.LittleEndian.PutUint16(msg[14:], 1)
	for _, colType := range schema.ColumnTypes {
		switch colType {
		case recordColumnByte:
			msg = append(msg, 0)
		case recordColumnWord:
			msg = binary.LittleEndian.AppendUint16(msg, 0)
		case recordColumnInt, recordColumnString:
			// 字符串零格与官服一致：u32(1)+NUL。
			if colType == recordColumnString {
				msg = binary.LittleEndian.AppendUint32(msg, 1)
				msg = append(msg, 0)
			} else {
				msg = binary.LittleEndian.AppendUint32(msg, 0)
			}
		case recordColumnInt64, recordColumnDouble:
			msg = binary.LittleEndian.AppendUint64(msg, 0)
		case recordColumnWideString:
			msg = binary.LittleEndian.AppendUint32(msg, 2)
			msg = binary.LittleEndian.AppendUint16(msg, 0)
		case recordColumnObject:
			msg = binary.LittleEndian.AppendUint64(msg, 0)
		default:
			return nil, fmt.Errorf("record %q column type %d unsupported for empty row", schema.Name, colType)
		}
	}
	return msg, nil
}

// appendRecordCellValue 按 schema 列型追加单格值（0x11 内联与 0x13 逐格共用）。
func appendRecordCellValue(msg []byte, schema clientdata.RecordSchema, col int, cell recordCell) ([]byte, error) {
	name := schema.Name
	switch schema.ColumnTypes[col] {
	case recordColumnInt:
		if cell.intValue == nil {
			return binary.LittleEndian.AppendUint32(msg, 0), nil
		}
		return binary.LittleEndian.AppendUint32(msg, *cell.intValue), nil
	case recordColumnWord:
		if cell.intValue == nil {
			return binary.LittleEndian.AppendUint16(msg, 0), nil
		}
		return binary.LittleEndian.AppendUint16(msg, uint16(*cell.intValue&0xFFFF)), nil
	case recordColumnInt64:
		if cell.int64Value == nil {
			return binary.LittleEndian.AppendUint64(msg, 0), nil
		}
		return binary.LittleEndian.AppendUint64(msg, uint64(*cell.int64Value)), nil
	case recordColumnObject:
		if !cell.objectCell || cell.int64Value == nil {
			return binary.LittleEndian.AppendUint64(msg, 0), nil
		}
		return binary.LittleEndian.AppendUint64(msg, uint64(*cell.int64Value)), nil
	case recordColumnString:
		if cell.text == nil {
			msg = binary.LittleEndian.AppendUint32(msg, 1)
			return append(msg, 0), nil
		}
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*cell.text)+1))
		msg = append(msg, (*cell.text)...)
		return append(msg, 0), nil
	default:
		return nil, fmt.Errorf("record %q column %d type %d unsupported in grid cell", name, col, schema.ColumnTypes[col])
	}
}

// serverRecordGridCell 即官服 0x13 逐格写：
//
//	13 | 00 | obj u32 | owner u32 | rec u16 | nrows u16(=1) | row u16 | col u8 | 值
func serverRecordGridCell(objectID, ownerID uint32, recordIndex, row uint16, col uint8, cell recordCell) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	if int(col) >= len(schema.ColumnTypes) {
		return nil, fmt.Errorf("record %q column %d out of range", schema.Name, col)
	}
	msg := make([]byte, 17, 26)
	msg[0] = 0x13
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 1)
	binary.LittleEndian.PutUint16(msg[14:], row)
	msg[16] = col
	return appendRecordCellValue(msg, schema, int(col), cell)
}

// ---- fight_relation（rec114，schema [obj,word]）----

// fightRelationRowAddFrame 直收/落地簇的行加帧：0x11 行0，col0=目标 obj、
// col1=0（g 22.157 / f 43.233 len=26 逐字节）。
func fightRelationRowAddFrame(targetID, targetOwner uint32) []byte {
	msg := make([]byte, 26)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], playerObjectID)
	binary.LittleEndian.PutUint32(msg[6:], playerOwnerID)
	binary.LittleEndian.PutUint16(msg[10:], recordFightRelation)
	binary.LittleEndian.PutUint16(msg[12:], 0)
	binary.LittleEndian.PutUint16(msg[14:], 1)
	binary.LittleEndian.PutUint32(msg[16:], targetID)
	binary.LittleEndian.PutUint32(msg[20:], targetOwner)
	// col1 (u16) 补零
	return msg
}

// fightRelationHitCellFrame 逐击累计格：0x13 rec114 行0 列1（u16）。
// 官服值恒 +500/击（g 三样本每击 +500 实证，与伤害值无关），语义未定位。
func fightRelationHitCellFrame(hits int32) []byte {
	msg := make([]byte, 19)
	msg[0] = 0x13
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], playerObjectID)
	binary.LittleEndian.PutUint32(msg[6:], playerOwnerID)
	binary.LittleEndian.PutUint16(msg[10:], recordFightRelation)
	binary.LittleEndian.PutUint16(msg[12:], 1)
	binary.LittleEndian.PutUint16(msg[14:], 0)
	msg[16] = 1
	binary.LittleEndian.PutUint16(msg[17:], uint16(hits*500&0xFFFF))
	return msg
}

// fightRelationRowDeleteFrame 击杀簇的撤销帧：0x12 del rec114 行0。
func fightRelationRowDeleteFrame() []byte {
	return serverRecordDelRow(playerObjectID, playerOwnerID, recordFightRelation, 0)
}

// ---- 0x2D 多属性帧 ----

// multiPropUpdateFrame 官服 0x2D 多属性形态：
//
//	2D | 01 | 00 | obj u32 | owner u32 | n u16 | (idx u16 + i32 值)×n
//
// 官服 NPC 战斗血量（HPRatio/HitHPRatio）与击杀清零（六属性全零）都走
// 本帧（f 43.548/44.668、g 22.592/39.286 实证），非 Go 旧 28/112 单形态。
func multiPropUpdateFrame(objectID, ownerID uint32, props []clientdata.IndexedProperty) ([]byte, error) {
	if len(props) == 0 || len(props) > 0xFFFF {
		return nil, fmt.Errorf("multi prop frame needs 1..65535 props, got %d", len(props))
	}
	msg := make([]byte, 13, 13+6*len(props))
	msg[0] = 0x2D
	msg[1] = 1
	msg[2] = 0
	binary.LittleEndian.PutUint32(msg[3:], objectID)
	binary.LittleEndian.PutUint32(msg[7:], ownerID)
	binary.LittleEndian.PutUint16(msg[11:], uint16(len(props)))
	for _, prop := range props {
		var intVal int32
		switch prop.Value.Type {
		case clientdata.WireInt32:
			intVal = prop.Value.I32
		case clientdata.WireFloat32:
			intVal = int32(prop.Value.F32)
		default:
			return nil, fmt.Errorf("multi prop %s: 0x2D 只承 i32 值", prop.Name)
		}
		msg = binary.LittleEndian.AppendUint16(msg, prop.Index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(intVal))
	}
	return msg, nil
}

// ---- msg24 段序/击退 ----

// rageComboFloatFrame 即 msg24 c1=10 怒气段序：[24, obj, 10, i32 段序]。
// 官服自第 2 段起发，值为段序（2/3/4，f/g 实证）。
func rageComboFloatFrame(count int32) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(playerObjectID, playerOwnerID),
		customInt(10),
		customInt(count),
	)
}

// knockbackFloatFrame 即 msg24 c1=29 击退（击杀簇，末段致死同毫秒）：
// [24, obj(受害), 29, obj(攻击者), 窄串受害模型, 宽串攻击者名]。
func knockbackFloatFrame(victimID, victimOwner, attackerID, attackerOwner uint32, victimModel, attackerWideName string) ([]byte, error) {
	return serverCustomIntMessage(24,
		customObject(victimID, victimOwner),
		customInt(29),
		customObject(attackerID, attackerOwner),
		customString(victimModel),
		customWideString(attackerWideName),
	)
}

// rushCastMark284Frame 即 msg284 空值帧：官服直收簇 CurSkillTarget 之后
// 恒发（g 三样本逐位一致），语义未定位，原样镜像。
func rushCastMark284Frame() ([]byte, error) {
	return serverCustomIntMessage(284)
}

// officialStatePingFrames 官服状态四连（f/g 全样本实证：受理簇/直收簇尾
// msg707 之后同毫秒必发，且全程 ~3-4s 周期重发；值恒定）：
//
//	msg586[5] + msg645[1] + msg646[1] + msg750[20]
//
// 语义未定位（Java 参考服与逆向笔记均无）。缺它时本服客户端在受理簇后
// 不进入自动接近（64785 会话实证：簇后信道干净仍不自走，卡 skill_pause
// 停滞态）——按 msg707[15] 同款「原样镜像」策略补齐。
func officialStatePingFrames() [4][]byte {
	frames := [4][]byte{}
	for i, spec := range []struct {
		id  int32
		val int32
	}{{586, 5}, {645, 1}, {646, 1}, {750, 20}} {
		frame, err := serverCustomIntMessage(spec.id, customInt(spec.val))
		if err != nil {
			continue
		}
		frames[i] = frame
	}
	return frames
}

// ---- CD 行官服方言（槽位覆写模型，玩家级双槽） ----

// cooldownRowBook 记录 CD 行的已下发状态（惰性清扫判定）。行0 整行
// 方言下个人/公共共用行 0（后写覆盖），expires 取两者较晚。
type cooldownRowBook struct {
	live     bool
	category int32
	team     int32
	expires  time.Time
}

// sweepExpiredCooldownRows 惰性清扫：CD 行过期后在下一次受理时发 0x12
// 删行 0，避免客户端残留上一招的 CD 行（官服 0x12 清扫语义的行0 简化）。
func (p *playerActor) sweepExpiredCooldownRows(now time.Time) [][]byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	var frames [][]byte
	book := &p.cdPersonalRowBook
	if book.live && now.After(book.expires) {
		book.live = false
		frames = append(frames, serverRecordDelRow(playerObjectID, playerOwnerID, recordCooldown, 0))
	}
	return frames
}

// writeOfficialCooldownRows 落出招 CD 双行（受理即结的普通出招、直收簇、
// c61 触发簇共用）。**行0 整行方言**（0x12 recordAddCells）：native-menu
// 反编译同款（skillCooldownFrameWithDuration，行0+UnixMilli）与本服
// fb5df8e 构建真机双验转圈——2026-11-09 官服逐格方言（0x11 空行行97/98+
// 0x13 逐格+行3 施法窗）真机不转圈（纪元修复后仍不认，帧与 g 14.660
// 逐字节同构），客户端消费该方言的前置（疑视图绑定/初始化）待考证，
// 见 docs/冲锋与打坐官服抓包分析及Go移植.md §3.2.2。
func (p *playerActor) writeOfficialCooldownRows(link sceneMessageConnection,
	category int32, team int32, personal time.Duration, public time.Duration, now time.Time) error {
	// 字面定义（测试/自增益零成本）不落 CD 行——官方从不发类别 0 的行。
	if category == 0 && personal <= 0 && public <= 0 {
		return nil
	}
	personalFrame, publicFrame, err := skillCooldownFrames(combatSkillDefinition{
		cooldownCategory: category,
		cooldownTeam:     team,
		personalCD:       personal,
		publicCD:         public,
	}, now)
	if err != nil {
		return fmt.Errorf("encode cooldown rows: %w", err)
	}
	if err := link.WriteFrame(personalFrame); err != nil {
		return err
	}
	if publicFrame != nil {
		if err := link.WriteFrame(publicFrame); err != nil {
			return err
		}
	}
	expires := personal
	if public > expires {
		expires = public
	}
	p.cdPersonalRowBook = cooldownRowBook{live: true, category: category, team: team, expires: now.Add(expires)}
	return nil
}
