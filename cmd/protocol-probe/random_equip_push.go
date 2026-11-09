package main

// 娃娃/宝物随机属性的持久化与下发（第一批接线）。
//
// 官服 record 方言（artifacts a-e.pcapng 实证，盘点文档 §3.6）：
//   0x11 flag=1：u32 viewID + u32 objectIndex + u16 record + u16 源行 +
//                u16 行数，一帧可批多行（属性行 3-4 行同帧实证）。
//   宝物：RandomPropRec(400)=[propID,value f64,0,0,0,0]、
//         RandomSkillRec(401)=[skillstore下标,0,value f64,0]；
//   娃娃：RandomPropRec 行 [propID,value,0,有效1,品质1-4,等级1-10]。
// 持久化：role_item_random_props（迁移 0016），以物品实例 uid 为主键。

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

const (
	recordRandomProp  uint16 = 400 // 官方 447 schema：RandomPropRec
	recordRandomSkill uint16 = 401 // RandomSkillRec
	recordPropModify  uint16 = 394 // PropModifyPackRec：模板属性包（equipment.ini @PropModifyPackRec），单列 [packID]
	recordSkillModify uint16 = 395 // SkillModifyPackRec：模板技能包，单列 [packID]
)

const randomPropsQueryTimeout = 5 * time.Second

// serverRecordAddRowsView 视图对象 record 追加（0x11 flag=1：viewID+objectIndex
// 寻址，一帧多行）。行按官方 schema 列型编码；越界列数报错。
func serverRecordAddRowsView(viewID, objectIndex uint16, recordIndex uint16, rows [][]recordCell) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	msg := make([]byte, 16)
	msg[0] = 0x11
	msg[1] = 1 // 视图对象（官服娃娃/宝物行帧实证；0=场景对象）
	binary.LittleEndian.PutUint32(msg[2:], uint32(viewID))
	binary.LittleEndian.PutUint32(msg[6:], uint32(objectIndex))
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 源行 0 起（新对象/整推）
	binary.LittleEndian.PutUint16(msg[14:], uint16(len(rows)))
	for _, row := range rows {
		if len(row) > len(schema.ColumnTypes) {
			return nil, fmt.Errorf("record %q row has %d cells, want at most %d",
				schema.Name, len(row), len(schema.ColumnTypes))
		}
		for column, columnType := range schema.ColumnTypes {
			var cell *recordCell
			if column < len(row) {
				cell = &row[column]
			}
			encoded, err := encodeRecordCell(schema.Name, column, columnType, cell)
			if err != nil {
				return nil, err
			}
			msg = append(msg, encoded...)
		}
	}
	return msg, nil
}

// encodeRecordCell 按官方线型码编码一格（1=byte 2=u16 3=int32 4=int64
// 6=f64 7=窄字符串 8=宽字符串 9=object；缺失补零/空——Java recordAddCells 同语义）。
func encodeRecordCell(recordName string, column int, columnType byte, cell *recordCell) ([]byte, error) {
	switch columnType {
	case recordColumnInt:
		if cell == nil || cell.intValue == nil {
			return []byte{0, 0, 0, 0}, nil
		}
		if !cell.onlyInt() {
			return nil, fmt.Errorf("record %q column %d requires int32", recordName, column)
		}
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, *cell.intValue)
		return buf, nil
	case recordColumnWord:
		if cell == nil || cell.intValue == nil {
			return []byte{0, 0}, nil
		}
		if !cell.onlyInt() {
			return nil, fmt.Errorf("record %q column %d requires word", recordName, column)
		}
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(*cell.intValue&0xFFFF))
		return buf, nil
	case recordColumnByte:
		if cell == nil || cell.byteValue == nil {
			return []byte{0}, nil
		}
		if !cell.onlyByte() {
			return nil, fmt.Errorf("record %q column %d requires byte", recordName, column)
		}
		if *cell.byteValue > math.MaxUint8 {
			return nil, fmt.Errorf("record %q column %d byte %d out of range", recordName, column, *cell.byteValue)
		}
		return []byte{byte(*cell.byteValue)}, nil
	case recordColumnInt64:
		if cell == nil || cell.int64Value == nil {
			return []byte{0, 0, 0, 0, 0, 0, 0, 0}, nil
		}
		if !cell.onlyInt64() {
			return nil, fmt.Errorf("record %q column %d requires int64", recordName, column)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(*cell.int64Value))
		return buf, nil
	case recordColumnDouble:
		if cell == nil || cell.doubleValue == nil {
			buf := make([]byte, 8)
			return buf, nil
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, math.Float64bits(*cell.doubleValue))
		return buf, nil
	default:
		// 窄/宽字符串与 object 引用：娃娃/宝物行全整型，缺失补零占位。
		switch columnType {
		case 7, 8:
			if columnType == 7 {
				return []byte{0, 0, 0, 0}, nil
			}
			return []byte{0, 0, 0, 0}, nil
		case 9:
			return []byte{0, 0, 0, 0}, nil
		}
		return nil, fmt.Errorf("record %q column %d unknown type %d", recordName, column, columnType)
	}
}

// randomPropRecRows RandomPropRec 行（六列：id/value/0/flag/品质/等级；
// 宝物后四列官服恒 0，由快照字段决定）。
func randomPropRecRows(lines []randomPropLine) [][]recordCell {
	rows := make([][]recordCell, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, []recordCell{
			recordInt(uint32(line.PropID)),
			recordFloat64(line.Value),
			recordInt(0),
			recordInt(uint32(line.Flag)),
			recordInt(uint32(line.Quality)),
			recordInt(uint32(line.Level)),
		})
	}
	return rows
}

// randomSkillRecRows RandomSkillRec 行（四列：store/SubSkill下标/value/unlock）。
func randomSkillRecRows(lines []randomSkillLine) [][]recordCell {
	rows := make([][]recordCell, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, []recordCell{
			recordInt(uint32(line.StoreID)),
			recordInt(uint32(line.Sub)),
			recordFloat64(line.Value),
			recordInt(uint32(line.Unlock)),
		})
	}
	return rows
}

// pushRandomPropsFrames 把一件物品的随机属性以视图 record 整推到客户端
// （纸娃娃 view1 槽位 或 背包页槽位；对应行先 serverViewRemove 重建的场景
// 由调用方保证，srcRow 恒 0）。
func pushRandomPropsFrames(link sceneMessageConnection, viewID, objectIndex uint16, snapshot itemRandomProps) error {
	if len(snapshot.PropRows) > 0 {
		frame, err := serverRecordAddRowsView(viewID, objectIndex, recordRandomProp, randomPropRecRows(snapshot.PropRows))
		if err != nil {
			return fmt.Errorf("encode RandomPropRec: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write RandomPropRec: %w", err)
		}
	}
	if len(snapshot.SkillRows) > 0 {
		frame, err := serverRecordAddRowsView(viewID, objectIndex, recordRandomSkill, randomSkillRecRows(snapshot.SkillRows))
		if err != nil {
			return fmt.Errorf("encode RandomSkillRec: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write RandomSkillRec: %w", err)
		}
	}
	return nil
}

// ---- 持久化（role_item_random_props，uid 主键） ----

type randomPropsDBStore struct {
	db *sql.DB
}

func newRandomPropsDBStore(db *sql.DB) *randomPropsDBStore {
	return &randomPropsDBStore{db: db}
}

// Save upsert 一件物品的随机属性快照（JSON 列）。
func (store *randomPropsDBStore) Save(ctx context.Context, roleID role.RoleID, uid, configID string, snapshot itemRandomProps) error {
	if store == nil || store.db == nil || uid == "" {
		return nil
	}
	props, err := json.Marshal(snapshot.PropRows)
	if err != nil {
		return err
	}
	skills, err := json.Marshal(snapshot.SkillRows)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, randomPropsQueryTimeout)
	defer cancel()
	_, err = store.db.ExecContext(ctx,
		`INSERT INTO role_item_random_props (role_id, uid, config_id, kind, prop_rows, skill_rows)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE config_id=VALUES(config_id), kind=VALUES(kind),
		   prop_rows=VALUES(prop_rows), skill_rows=VALUES(skill_rows)`,
		roleID, uid, configID, snapshot.Kind, string(props), string(skills))
	if err != nil {
		return fmt.Errorf("save random props %s: %w", configID, err)
	}
	return nil
}

// DeleteByUID 覆盖重掷前清旧行（同 uid 重 roll）。
func (store *randomPropsDBStore) DeleteByUID(ctx context.Context, roleID role.RoleID, uid string) error {
	if store == nil || store.db == nil || uid == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, randomPropsQueryTimeout)
	defer cancel()
	_, err := store.db.ExecContext(ctx,
		`DELETE FROM role_item_random_props WHERE role_id = ? AND uid = ?`, roleID, uid)
	return err
}

// LoadAll 登录批读：uid → 快照（无表/无行返回空 map，不视为错误）。
func (store *randomPropsDBStore) LoadAll(ctx context.Context, roleID role.RoleID) map[string]itemRandomProps {
	snapshots := map[string]itemRandomProps{}
	if store == nil || store.db == nil || roleID == 0 {
		return snapshots
	}
	ctx, cancel := context.WithTimeout(ctx, randomPropsQueryTimeout)
	defer cancel()
	rows, err := store.db.QueryContext(ctx,
		`SELECT uid, kind, prop_rows, skill_rows FROM role_item_random_props WHERE role_id = ?`, roleID)
	if err != nil {
		return snapshots
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var kind int
		var propRows, skillRows string
		if err := rows.Scan(&uid, &kind, &propRows, &skillRows); err != nil {
			continue
		}
		snapshot := itemRandomProps{Kind: kind}
		_ = json.Unmarshal([]byte(propRows), &snapshot.PropRows)
		_ = json.Unmarshal([]byte(skillRows), &snapshot.SkillRows)
		normalizeRandomPropsDialect(&snapshot)
		snapshots[uid] = snapshot
	}
	return snapshots
}

// normalizeRandomPropsDialect 把旧快照归一到官方线上方言：娃娃基础行
// flag=1+quality=0（曾致客户端露 ui.newtreasure.quality_0 键名）强制回到
// 官方 [propID,value,0,0,0,0]；附加行 quality 恒 1-4 不受影响。
func normalizeRandomPropsDialect(snapshot *itemRandomProps) {
	if snapshot.Kind != randomKindWawa {
		return
	}
	for i := range snapshot.PropRows {
		if snapshot.PropRows[i].Quality == 0 && snapshot.PropRows[i].Flag != 0 {
			snapshot.PropRows[i].Flag = 0
			snapshot.PropRows[i].Level = 0
		}
	}
}

// rollRandomPropsSnapshot 掷值入口：宝物/娃娃各走管线，随机源复用物品链
// 全局源（itemUseRandom，与开箱同源可测）。
func rollRandomPropsSnapshot(configID string, kind int) (itemRandomProps, error) {
	catalog := randomEquipCatalogOf()
	if catalog.err != nil {
		return itemRandomProps{}, catalog.err
	}
	switch kind {
	case randomKindTreaure:
		return catalog.rollTreasure(configID, itemUseRandom())
	case randomKindWawa:
		return catalog.rollWawa(configID, itemUseRandom())
	}
	return itemRandomProps{}, fmt.Errorf("not a random equip: %s", configID)
}

// ---- playerActor 集成（实力点数 + 镇派兵器面板 4 属性） ----

// effectivePowerLevel 实力点数权威口径（盘点 §2.7）：GM set_power_level
// 显式覆盖优先；否则 = max(已学内功当前层 varprop.NeiGongLevel) 自动推导。
// 目录装载失败回落会话种子（151）。
func (p *playerActor) effectivePowerLevel() int32 {
	p.mu.Lock()
	set := p.gm.powerLevelSet
	manual := p.gm.powerLevel
	books := make([]neigongPowerBook, 0, len(p.progress.books))
	for _, book := range p.progress.books {
		books = append(books, neigongPowerBook{staticData: book.staticData, level: book.level})
	}
	p.mu.Unlock()
	if set {
		return manual
	}
	catalog := randomEquipCatalogOf()
	if catalog.err != nil {
		return manual // 种子兜底
	}
	if derived := catalog.derivePowerLevel(books); derived > 0 {
		return int32(derived)
	}
	return manual
}

// zhenpaiEquipBonus 镇派兵器 4 条动态属性的面板注入位（progress 消费）。
type zhenpaiEquipBonus struct {
	maxHP, maxMP, qinggong, parry int32
}

// refreshZhenpaiEquipBonuses 按当前穿戴武器 + 实力点数重算 4 条动态属性，
// 并联动血/蓝池上限（syncEquippedNeiGongResourcesLocked 消费同一
// derivedResourceAdds 注入位）；返回是否有变化（调用方据此决定是否补推
// 面板帧）。登录时序：restoreSessionInventory 之后、sendPlayerSpawn 之前
// 调用，保证 0x0D 档/权威 vital 0x10/面板身份帧三处同源。
func (p *playerActor) refreshZhenpaiEquipBonuses() bool {
	var weapon string
	for _, equip := range p.sessionEquips() {
		if equip.slot == 22 {
			weapon = equip.configID
			break
		}
	}
	bonus := zhenpaiEquipBonus{}
	if weapon != "" {
		power := int(p.effectivePowerLevel())
		values := randomEquipCatalogOf().zhenpaiDynamicValues(weapon, power)
		bonus = zhenpaiEquipBonus{
			maxHP:    int32(values["MaxHPAdd"]),
			maxMP:    int32(values["MaxMPAdd"]),
			qinggong: int32(values["MaxQingGongPointAdd"]),
			parry:    int32(values["MaxParryAdd"]),
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.progress.zhenpaiEquip == bonus {
		return false
	}
	p.progress.zhenpaiEquip = bonus
	p.syncEquippedNeiGongResourcesLocked()
	return true
}

// pushZhenpaiPanelFrame 镇派兵器 4 属性变化后的面板补帧（581/582/475/694
// 官方总量位——与 progress 注入位同源）。
func (p *playerActor) pushZhenpaiPanelFrame(link sceneMessageConnection) error {
	p.mu.Lock()
	maxHPAdd, maxMPAdd := p.progress.derivedResourceAdds()
	parry := p.progress.parryValueTotal()
	qinggongAdd := p.progress.zhenpaiEquip.qinggong
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 581, Name: "MaxHPAdd", Value: clientdata.Int32Value(maxHPAdd)},
		{Index: 582, Name: "MaxMPAdd", Value: clientdata.Int32Value(maxMPAdd)},
		{Index: propParryValue, Name: "ParryValue", Value: clientdata.Int32Value(parry)},
		{Index: 694, Name: "MaxQingGongPointAdd", Value: clientdata.Int32Value(qinggongAdd)},
	})
	if err != nil {
		return err
	}
	return link.WriteFrame(frame)
}

// ---- itemSession 集成（宝物/娃娃 record 整推 + 登录装载） ----

// ensureRandomProps 物品的随机属性快照：持久化有则用之（登录后穿戴等
// 场景），无则现掷并存（GM 发放首件）。
func (h *itemSession) ensureRandomProps(configID, uid string, kind int) (itemRandomProps, error) {
	if h.randomProps != nil && uid != "" {
		if snapshots := h.randomProps.LoadAll(h.ctx, h.roleID); len(snapshots) > 0 {
			if snapshot, ok := snapshots[uid]; ok {
				return snapshot, nil
			}
		}
	}
	snapshot, err := rollRandomPropsSnapshot(configID, kind)
	if err != nil {
		return itemRandomProps{}, err
	}
	if h.randomProps != nil && uid != "" {
		if err := h.randomProps.Save(h.ctx, h.roleID, uid, configID, snapshot); err != nil {
			log.Printf("%s: save random props %s: %v", h.remote, configID, err)
		}
	}
	return snapshot, nil
}

// pushModifyPackRecords 模板包物品（镇派兵器等）的官方词条通道：
// PropModifyPackRec(394)/SkillModifyPackRec(395) 每包一行单 int32 列，
// 客户端按本地 proppack/skillpack 渲染并按 PowerLevel 自行命中条件行
//（c.pcapng 修为书 394×6 行同帧实证；PropBuilt@1954 属打造/洗练实例
// 通道，模板武器不走，且与 EffectPackID 互斥——见 equipmentViewAdd）。
func pushModifyPackRecords(link sceneMessageConnection, viewID, objectIndex uint16, configID string) {
	catalog := randomEquipCatalogOf()
	fields, ok := catalog.itemFields[configID]
	if !ok || (len(fields.propPacks) == 0 && len(fields.skillPacks) == 0) {
		return
	}
	for _, group := range []struct {
		index uint16
		packs []int
	}{
		{recordPropModify, fields.propPacks},
		{recordSkillModify, fields.skillPacks},
	} {
		if len(group.packs) == 0 {
			continue
		}
		rows := make([][]recordCell, 0, len(group.packs))
		for _, packID := range group.packs {
			rows = append(rows, []recordCell{recordInt(uint32(packID))})
		}
		frame, err := serverRecordAddRowsView(viewID, objectIndex, group.index, rows)
		if err != nil {
			log.Printf("encode modify packs %s rec %d: %v", configID, group.index, err)
			continue
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("write modify packs %s rec %d: %v", configID, group.index, err)
		}
	}
}

// pushRandomPropsForEquip 穿戴后整推（纸娃娃 view1 槽位）：模板包 record
// + 宝物/娃娃随机 record。
func (h *itemSession) pushRandomPropsForEquip(configID, uid string, slot int32) {
	pushModifyPackRecords(h.link, viewDoll, uint16(slot), configID)
	kind := randomEquipCatalogOf().randomEquipKind(configID)
	if kind == randomKindNone {
		return
	}
	snapshot, err := h.ensureRandomProps(configID, uid, kind)
	if err != nil {
		log.Printf("%s: roll %s: %v", h.remote, configID, err)
		return
	}
	if err := pushRandomPropsFrames(h.link, viewDoll, uint16(slot), snapshot); err != nil {
		log.Printf("%s: push random props equip[%d]: %v", h.remote, slot, err)
	}
}

// pushRandomPropsForBag 入包后整推（背包页槽位）：模板包 record + 随机 record。
func (h *itemSession) pushRandomPropsForBag(configID, uid string, viewID, slot int32) {
	pushModifyPackRecords(h.link, bagViewOf(viewID), uint16(slot), configID)
	kind := randomEquipCatalogOf().randomEquipKind(configID)
	if kind == randomKindNone {
		return
	}
	snapshot, err := h.ensureRandomProps(configID, uid, kind)
	if err != nil {
		log.Printf("%s: roll %s: %v", h.remote, configID, err)
		return
	}
	if err := pushRandomPropsFrames(h.link, bagViewOf(viewID), uint16(slot), snapshot); err != nil {
		log.Printf("%s: push random props bag[%d:%d]: %v", h.remote, viewID, slot, err)
	}
}

// pushLoginRandomProps 登录后按 uid 批取快照，为穿戴行（纸娃娃 view1）与
// 背包行（对应页）各整推 record；无快照的普通物品零帧。
func pushLoginRandomProps(link sceneMessageConnection, store *randomPropsDBStore, ctx context.Context, roleID role.RoleID, equips []equipRow, items []bagRow) {
	if store == nil || store.db == nil {
		return
	}
	catalog := randomEquipCatalogOf()
	if catalog.err != nil {
		return
	}
	snapshots := store.LoadAll(ctx, roleID)
	pushed := 0
	// 模板包 record（镇派兵器等，randomKindNone 但有 packs）先行；
	// 快照为空也要推（随机属性行才依赖 uid 快照）。
	for _, equip := range equips {
		pushModifyPackRecords(link, viewDoll, uint16(equip.slot), equip.configID)
		if catalog.randomEquipKind(equip.configID) == randomKindNone {
			continue
		}
		snapshot, ok := snapshots[equip.uid]
		if !ok {
			continue
		}
		if err := pushRandomPropsFrames(link, viewDoll, uint16(equip.slot), snapshot); err != nil {
			log.Printf("login random props equip[%d] %s: %v", equip.slot, equip.configID, err)
			continue
		}
		pushed++
	}
	for _, item := range items {
		if item.slot < 1 || item.slot > int32(bagSlotMax) {
			continue
		}
		pushModifyPackRecords(link, bagViewOf(item.viewID), uint16(item.slot), item.configID)
		if catalog.randomEquipKind(item.configID) == randomKindNone {
			continue
		}
		snapshot, ok := snapshots[item.uid]
		if !ok {
			continue
		}
		if err := pushRandomPropsFrames(link, bagViewOf(item.viewID), uint16(item.slot), snapshot); err != nil {
			log.Printf("login random props bag[%d:%d] %s: %v", item.viewID, item.slot, item.configID, err)
			continue
		}
		pushed++
	}
	if pushed > 0 {
		log.Printf("login random props rows pushed=%d role=%d", pushed, roleID)
	}
}
