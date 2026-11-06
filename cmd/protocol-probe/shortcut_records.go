package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/local/9yin-go-server/internal/entity"
)

const (
	clientCustomSetShortcut    int32 = 200
	clientCustomRemoveShortcut int32 = 201
	// 当前现代客户端中的 CLIENT_CUSTOMMSG_SITCROSS。这是一个
	// 原生的坐下/起立切换，并非战斗技能的 ConfigID。
	clientCustomSitCross int32 = 240

	// 把测试放在 Ctrl+0（快捷栏索引 9）上，避开本地测试角色通常
	// 使用的两个初始战斗槽位。原生动画串 `sitdown` 不是
	// FuncManager 记录；已学习的技能才是。
	sitcrossShortcutIndex int32 = 9
	sitcrossShortcutKind        = "skill"
	sitcrossShortcutID          = "zs_default_01"

	// grid_shortcut_1 是下方的数字快捷栏。它每页有十个格子，
	// 因此第二页从索引 10 开始。grid_shortcut_2 是独立的
	// 上方 Ctrl+数字辅助栏；它从 200 开始，并不是用户
	// 要求的第二个数字快捷栏页。
	heartBuddhaPalmShortcutBaseIndex int32 = 10
	heartBuddhaPalmShortcutKind            = "skill"
	taiJiQuanGuPuShortcutBaseIndex   int32 = 20
	taiJiQuanGuPuShortcutKind              = "skill"
)

var heartBuddhaPalmShortcutIDs = []string{
	"CS_jh_xfz01", "CS_jh_xfz02", "CS_jh_xfz03", "CS_jh_xfz04",
	"CS_jh_xfz05", "CS_jh_xfz06", "CS_jh_xfz07",
}

var taiJiQuanGuPuShortcutIDs = []string{
	"CS_wd_tjq01", "CS_wd_tjq02", "CS_wd_tjq03", "CS_wd_tjq04",
	"CS_wd_tjq05", "CS_wd_tjq06", "CS_wd_tjq07", "CS_wd_tjq08",
}

// playerShortcut 对应 shortcut_rec 的一行数据：
// 快捷栏索引、快捷栏类型（"skill"、"item" 等）以及 config ID。
type playerShortcut struct {
	index int32
	kind  string
	id    string
}

// serverRecordDelRow 编码 FxNet2 ServerRecordDelRow (0x12)。当前的
// 接收端要求恰好 14 字节，并把末尾的 u16 解释为
// 指定记录中从 0 开始计数的行号。
func serverRecordDelRow(objectID, ownerID uint32, recordIndex, row uint16) []byte {
	msg := make([]byte, 14)
	msg[0] = 0x12
	msg[1] = 0 // 场景对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], row)
	return msg
}

func (p *playerActor) setShortcut(index int32, kind, id string) (oldRow int, existed, unchanged bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for row, shortcut := range p.shortcuts {
		if shortcut.index != index {
			continue
		}
		if shortcut.kind == kind && shortcut.id == id {
			return row, true, true
		}
		p.shortcuts = append(p.shortcuts[:row], p.shortcuts[row+1:]...)
		p.shortcuts = append(p.shortcuts, playerShortcut{index: index, kind: kind, id: id})
		return row, true, false
	}
	p.shortcuts = append(p.shortcuts, playerShortcut{index: index, kind: kind, id: id})
	return 0, false, false
}

func (p *playerActor) removeShortcutRow(row int) (playerShortcut, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if row < 0 || row >= len(p.shortcuts) {
		return playerShortcut{}, false
	}
	shortcut := p.shortcuts[row]
	p.shortcuts = append(p.shortcuts[:row], p.shortcuts[row+1:]...)
	return shortcut, true
}

func (p *playerActor) shortcutSnapshot() []playerShortcut {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]playerShortcut(nil), p.shortcuts...)
}

func shortcutAddFrame(shortcut playerShortcut) ([]byte, error) {
	if shortcut.index < 0 {
		return nil, fmt.Errorf("shortcut index %d is negative", shortcut.index)
	}
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordShortcut, []recordCell{
		recordInt(uint32(shortcut.index)),
		recordString(shortcut.kind),
		recordString(shortcut.id),
	})
}

// grantShortcutRows 在一次实际场景切换之后恢复挂在玩家身上的记录。
// ServerExitScene 会销毁原生玩家对象及其全部记录行，
// 即使连接层面的 schema 仍然存活。
func grantShortcutRows(conn sceneMessageConnection, player *playerActor) error {
	if player == nil {
		return nil
	}
	for _, shortcut := range player.shortcutSnapshot() {
		frame, err := shortcutAddFrame(shortcut)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("restore shortcut index %d: %w", shortcut.index, err)
		}
	}
	return nil
}

// grantSitcrossShortcut 在 ClientReady 之后安装已学习的原生打坐快捷栏条目。
// 客户端把 zs_default_01 识别为一种特殊的原生动作，
// 并使用 CustomSend 240 而不是 211 战斗路线。
func grantSitcrossShortcut(conn sceneMessageConnection, player *playerActor) (bool, error) {
	if player == nil {
		return false, nil
	}

	player.mu.Lock()
	for _, shortcut := range player.shortcuts {
		if shortcut.index == sitcrossShortcutIndex {
			player.mu.Unlock()
			return false, nil
		}
	}
	shortcut := playerShortcut{index: sitcrossShortcutIndex, kind: sitcrossShortcutKind, id: sitcrossShortcutID}
	player.shortcuts = append(player.shortcuts, shortcut)
	player.mu.Unlock()

	frame, err := shortcutAddFrame(shortcut)
	if err != nil {
		return false, err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return false, fmt.Errorf("add sitcross shortcut: %w", err)
	}
	return true, nil
}

// grantHeartBuddhaPalmShortcuts 通过原生 shortcut_rec 记录，用完整的
// 心佛掌套路填满下方数字快捷栏第二页的 1..7 格。
// 每个请求的目标槽位均为权威设置。
func grantHeartBuddhaPalmShortcuts(conn sceneMessageConnection, player *playerActor) (int, error) {
	if player == nil {
		return 0, nil
	}
	granted := 0
	for offset, id := range heartBuddhaPalmShortcutIDs {
		index := heartBuddhaPalmShortcutBaseIndex + int32(offset)
		oldRow, existed, unchanged := player.setShortcut(index, heartBuddhaPalmShortcutKind, id)
		if unchanged {
			continue
		}
		if existed {
			if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordShortcut, uint16(oldRow))); err != nil {
				return granted, fmt.Errorf("replace heart buddha palm shortcut row %d: %w", oldRow, err)
			}
		}
		frame, err := shortcutAddFrame(playerShortcut{index: index, kind: heartBuddhaPalmShortcutKind, id: id})
		if err != nil {
			return granted, err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return granted, fmt.Errorf("add heart buddha palm shortcut index %d: %w", index, err)
		}
		granted++
	}
	return granted, nil
}

// grantTaiJiQuanGuPuShortcuts 用完整已学习的 太极拳(古谱) 套路
// 填满数字快捷栏第三页的 1..8 格。
func grantTaiJiQuanGuPuShortcuts(conn sceneMessageConnection, player *playerActor) (int, error) {
	if player == nil {
		return 0, nil
	}
	granted := 0
	for offset, id := range taiJiQuanGuPuShortcutIDs {
		index := taiJiQuanGuPuShortcutBaseIndex + int32(offset)
		oldRow, existed, unchanged := player.setShortcut(index, taiJiQuanGuPuShortcutKind, id)
		if unchanged {
			continue
		}
		if existed {
			if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordShortcut, uint16(oldRow))); err != nil {
				return granted, fmt.Errorf("replace tai ji shortcut row %d: %w", oldRow, err)
			}
		}
		frame, err := shortcutAddFrame(playerShortcut{index: index, kind: taiJiQuanGuPuShortcutKind, id: id})
		if err != nil {
			return granted, err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return granted, fmt.Errorf("add tai ji shortcut index %d: %w", index, err)
		}
		granted++
	}
	return granted, nil
}

// handleSitcrossCustom 处理观测到的现代原生打坐请求：
// activity TVarList(240, int startOrStop)。它刻意返回普通的
// 0x10 角色属性，而不是旧版的 S->C 0x1E 自定义通知。
func handleSitcrossCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSitCross {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject sitcross before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 2 {
		return true, fmt.Errorf("sitcross expects TVarList(240, int start_or_stop), got %v", custom.Values)
	}

	var next uint8
	switch custom.Values[1].Int32 {
	case 1: // 从按下已学习的 zs_default_01 快捷键的抓包中捕获
		next = entity.LogicStateSitcross
	case 0, 2:
		// 当前客户端在本地被打断（包括被敌方攻击命中）之后发送 2。
		// 把观测到的两种取消形式都当作正常停止处理：
		// 拒绝它会在角色正要离开 LS_SITCROSS 转入战斗的
		// 关头拆掉整个会话。
		next = entity.LogicStateNormal
	default:
		log.Printf("%s: ignore unknown sitcross start_or_stop=%d", remote, custom.Values[1].Int32)
		return true, nil
	}

	current := player.actor.Snapshot().LogicState
	if current == entity.LogicStateDied {
		log.Printf("%s: reject sitcross while dead", remote)
		return true, nil
	}
	if current == next {
		// 按住按键期间客户端会重复发送 start。不要把重复的
		// start 变成切换开关，也不要用相同的 0x10 帧去淹没它。
		log.Printf("%s: sitcross state already %d; duplicate request ignored", remote, next)
		return true, nil
	}

	player.actor.SetLogicState(next)
	frame, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode sitcross state=%d: %w", next, err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("send sitcross state=%d: %w", next, err)
	}
	log.Printf("%s: sitcross native state %d -> %d via normal 0x10", remote, current, next)
	return true, nil
}

// handleShortcutCustom 实现现代版 shortcut_game.lua 的约定。
// Set 使用快捷栏索引/类型/config ID。Remove 使用当前的记录行，
// 而不是可见的快捷栏索引。
func handleShortcutCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 {
		return false, nil
	}
	messageID := custom.Values[0].Int32
	if messageID != clientCustomSetShortcut && messageID != clientCustomRemoveShortcut {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject shortcut request before player spawn", remote)
		return true, nil
	}

	switch messageID {
	case clientCustomSetShortcut:
		if len(custom.Values) != 4 ||
			custom.Values[1].Type != 2 ||
			custom.Values[2].Type != 6 ||
			custom.Values[3].Type != 6 {
			return true, fmt.Errorf("set shortcut expects int index and two strings, got %v", custom.Values)
		}
		shortcut := playerShortcut{
			index: custom.Values[1].Int32,
			kind:  custom.Values[2].Text,
			id:    custom.Values[3].Text,
		}
		if shortcut.index < 0 || shortcut.index > 4096 {
			return true, fmt.Errorf("shortcut index %d is out of range", shortcut.index)
		}
		if shortcut.kind == "" || shortcut.id == "" || len(shortcut.kind) > 64 || len(shortcut.id) > 256 {
			return true, fmt.Errorf("invalid shortcut kind/id lengths %d/%d", len(shortcut.kind), len(shortcut.id))
		}
		oldRow, existed, unchanged := player.setShortcut(shortcut.index, shortcut.kind, shortcut.id)
		if unchanged {
			log.Printf("%s: shortcut unchanged index=%d kind=%s id=%s", remote, shortcut.index, shortcut.kind, shortcut.id)
			return true, nil
		}
		if existed {
			if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordShortcut, uint16(oldRow))); err != nil {
				return true, fmt.Errorf("delete replaced shortcut row %d: %w", oldRow, err)
			}
		}
		frame, err := shortcutAddFrame(shortcut)
		if err != nil {
			return true, err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return true, fmt.Errorf("add shortcut index %d: %w", shortcut.index, err)
		}
		log.Printf("%s: shortcut set index=%d kind=%s id=%s replaced=%t", remote, shortcut.index, shortcut.kind, shortcut.id, existed)
		return true, nil

	case clientCustomRemoveShortcut:
		if len(custom.Values) != 2 || custom.Values[1].Type != 2 {
			return true, fmt.Errorf("remove shortcut expects int row, got %v", custom.Values)
		}
		row := int(custom.Values[1].Int32)
		shortcut, exists := player.removeShortcutRow(row)
		if !exists {
			log.Printf("%s: ignore missing shortcut row=%d", remote, row)
			return true, nil
		}
		if err := conn.WriteFrame(serverRecordDelRow(playerObjectID, playerOwnerID, recordShortcut, uint16(row))); err != nil {
			return true, fmt.Errorf("remove shortcut row %d: %w", row, err)
		}
		log.Printf("%s: shortcut removed row=%d index=%d kind=%s id=%s", remote, row, shortcut.index, shortcut.kind, shortcut.id)
		return true, nil
	}
	return false, nil
}
