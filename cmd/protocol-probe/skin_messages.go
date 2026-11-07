package main

// 风物志协议帧（Java SkinMessages 的 Go 移植，docs/装备外观与风物志
// 挂载.md V3 定稿）：
//
//   - 收录 = CardRec[3] 位图（32 皮肤/行，serial=皮肤号+1——皮肤 1=行 0
//     位 1、31=行 0 位 31、32=行 1 位 0，与 Task_Completed 引擎位判定
//     同式）；显式行号全推含零行，0x12 倒序删行。
//   - 挂载 = UseCardRec[2,2]（皮肤号 + 显示类型 0）追加行、0x12 倒序删。
//   - 显示开关 = 单帧 0x10：ShowEquipType(408, byte) + CardID(2117, int32)。
//
// **永不发裸 custom 180**：S2C 180 是子操作分发（lua64 on_card_msg：
// 0 开兑换窗/1 开替换窗/2 开收录窗/3 收录回执/4 武道购买刷新），裸皮肤
// 号会被当 opt_type 误开窗——收录状态由 CardRec 记录帧驱动
// CollectCardManager(IsGetCard) + data_binder 刷新。

import (
	"github.com/local/9yin-go-server/internal/clientdata"
)

const (
	skinCardSubUse     int32 = 0 // 挂载（须已收录；同部件互斥替换）
	skinCardSubCancel  int32 = 1 // 卸载
	skinCardSubCollect int32 = 7 // 收录（背包有对应卡片则消耗 1 张）
)

// skinBitmapRows 收录集 → 位图行（serial=皮肤号+1；空集也产出 1 行 0
// 值——建立通道让客户端清空旧位图）。
func skinBitmapRows(collected []int32) (map[uint32]uint32, uint32) {
	rows := make(map[uint32]uint32)
	var maxRow uint32
	for _, id := range collected {
		if id < 0 {
			continue
		}
		row, mask := skinBitmapBit(uint32(id))
		rows[row] |= mask
		if row > maxRow {
			maxRow = row
		}
	}
	return rows, maxRow + 1
}

// skinBitmapBit 皮肤号 → (行, 位掩码)。serial=id+1；serial 整除 32 时
// 归上一行第 32 位（serial=32→行1位0 即 id=31→行0位31 的下一位）。
func skinBitmapBit(id uint32) (uint32, uint32) {
	serial := id + 1
	row := serial / 32
	bit := serial % 32
	if bit == 0 {
		row--
		bit = 32
	}
	return row, uint32(1) << (bit - 1)
}

// skinCollectedFrames 位图全推（显式行号 0..rowCount-1，含零值行）。
func skinCollectedFrames(objectID, ownerID uint32, rows map[uint32]uint32, rowCount uint32) [][]byte {
	frames := make([][]byte, 0, rowCount)
	for row := uint32(0); row < rowCount; row++ {
		frames = append(frames, serverRecordAddInt32Row(objectID, ownerID, recordCard, row, rows[row]))
	}
	return frames
}

// skinCollectedClearFrames 位图清行（0x12 倒序——升序删会重打包行号）。
func skinCollectedClearFrames(objectID, ownerID uint32, rowCount uint32) [][]byte {
	return skinDelRowFrames(objectID, ownerID, recordCard, rowCount)
}

// skinMountedFrames 挂载行全推（按挂载序追加 [皮肤号, 显示类型 0]）。
func skinMountedFrames(objectID, ownerID uint32, mounted []int32) ([][]byte, error) {
	frames := make([][]byte, 0, len(mounted))
	for _, id := range mounted {
		frame, err := serverRecordAddCells(objectID, ownerID, recordUseCard, []recordCell{
			recordInt(uint32(id)), recordInt(0),
		})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// skinDelRowFrames 通用倒序删行（清位图/清挂载共用）。
func skinDelRowFrames(objectID, ownerID uint32, recordIndex uint16, rowCount uint32) [][]byte {
	frames := make([][]byte, 0, rowCount)
	for row := int32(rowCount) - 1; row >= 0; row-- {
		frames = append(frames, serverRecordDelRow(objectID, ownerID, recordIndex, uint16(row)))
	}
	return frames
}

// skinShowEquipTypeFrame 显示开关单帧 0x10（ShowEquipType + CardID；
// 卸载后 showType 回落存档偏好而非恒 0）。
func skinShowEquipTypeFrame(objectID, ownerID uint32, showEquipType int, cardID int32) ([]byte, error) {
	return sceneObjectProperties(objectID, ownerID, 0, []clientdata.IndexedProperty{
		{Index: propShowEquipType, Name: "ShowEquipType", Value: clientdata.ByteValue(uint8(showEquipType))},
		{Index: 2117, Name: "CardID", Value: clientdata.Int32Value(cardID)},
	})
}

// pushSkinRefreshFrames 整刷编排（Java skinRefreshFrames 同序）：
// 清位图 → 推位图 → 清挂载行 → 推挂载 → 显示开关帧。刷新前由调用方
// 读 player.skinRows/skinMountedRows（旧推计数），刷新后回写新计数。
func pushSkinRefreshFrames(conn sceneMessageConnection, player *playerActor) error {
	state := player.skinSnapshot()
	pref := effectiveShowEquipType(player)
	// 显示开关：挂载非空 → 2；卸载后回落存档偏好（无档 0）。
	showType := pref
	var lastSkin int32
	if len(state.mounted) > 0 {
		showType = showEquipFashion
		lastSkin = state.mounted[len(state.mounted)-1]
	}
	player.mu.Lock()
	oldRows, oldMounted := player.skinRows, player.skinMountedRows
	player.mu.Unlock()

	for _, frame := range skinCollectedClearFrames(playerObjectID, playerOwnerID, oldRows) {
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	rows, rowCount := skinBitmapRows(state.collected)
	for _, frame := range skinCollectedFrames(playerObjectID, playerOwnerID, rows, rowCount) {
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	for _, frame := range skinDelRowFrames(playerObjectID, playerOwnerID, recordUseCard, oldMounted) {
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	mountedFrames, err := skinMountedFrames(playerObjectID, playerOwnerID, state.mounted)
	if err != nil {
		return err
	}
	for _, frame := range mountedFrames {
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	frame, err := skinShowEquipTypeFrame(playerObjectID, playerOwnerID, showType, lastSkin)
	if err != nil {
		return err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return err
	}

	player.mu.Lock()
	player.skinRows = rowCount
	player.skinMountedRows = uint32(len(state.mounted))
	player.mu.Unlock()
	return nil
}

// pushSavedSkins 登录 ClientReady 恢复（无存档/双空集零帧——客户端保持
// 未收录态）。位置：键位推送后、FashionSuitRec 行前。
func pushSavedSkins(conn sceneMessageConnection, player *playerActor) error {
	state := player.skinSnapshot()
	if len(state.collected) == 0 && len(state.mounted) == 0 {
		return nil
	}
	return pushSkinRefreshFrames(conn, player)
}
