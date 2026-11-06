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
	// CLIENT_CUSTOMMSG_SITCROSS in the current modern client.  This is a
	// native sit/stand toggle, not a combat-skill ConfigID.
	clientCustomSitCross int32 = 240

	// Keep the test on Ctrl+0 (shortcut index 9), away from the two starter
	// combat slots the local test character normally uses. The native animation
	// string `sitdown` is not a FuncManager record; the learned skill is.
	sitcrossShortcutIndex int32 = 9
	sitcrossShortcutKind        = "skill"
	sitcrossShortcutID          = "zs_default_01"

	// grid_shortcut_1 is the lower numeric shortcut bar.  It has ten cells per
	// page, so its second page begins at index 10.  grid_shortcut_2 is the
	// separate upper Ctrl+number auxiliary bar; it begins at 200 and is not the
	// user-requested second numeric shortcut page.
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

// playerShortcut mirrors one row of shortcut_rec:
// shortcut index, shortcut kind ("skill", "item", ...), and config ID.
type playerShortcut struct {
	index int32
	kind  string
	id    string
}

// serverRecordDelRow encodes FxNet2 ServerRecordDelRow (0x12). The current
// receiver requires exactly 14 bytes and interprets the final u16 as the
// zero-based row number in the named record.
func serverRecordDelRow(objectID, ownerID uint32, recordIndex, row uint16) []byte {
	msg := make([]byte, 14)
	msg[0] = 0x12
	msg[1] = 0 // scene object
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

// grantShortcutRows restores the player-attached record after a live scene
// transition. ServerExitScene destroys the native player object and all of
// its record rows even though the connection-level schema remains alive.
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

// grantSitcrossShortcut installs the learned native sitcross shortcut after
// ClientReady. The client recognizes zs_default_01 as a special native action
// and uses CustomSend 240 instead of the 211 combat route.
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

// grantHeartBuddhaPalmShortcuts fills cells 1..7 of the lower numeric
// shortcut bar's second page with the complete 心佛掌 set through the native
// shortcut_rec record. Each requested destination is authoritative.
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

// grantTaiJiQuanGuPuShortcuts fills cells 1..8 of numeric page three with
// the complete learned 太极拳(古谱) routine.
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

// handleSitcrossCustom handles the observed modern native sitcross request:
// activity TVarList(240, int startOrStop).  It deliberately returns ordinary
// 0x10 role properties rather than a legacy S->C 0x1E custom notification.
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
	case 1: // captured from pressing the learned zs_default_01 shortcut
		next = entity.LogicStateSitcross
	case 0, 2:
		// The current client sends 2 after a local interruption (including an
		// incoming hostile hit).  Treat both observed cancellation forms as a
		// normal stop: rejecting it tears down the whole session just when the
		// actor must leave LS_SITCROSS for combat.
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
		// The client repeats start while the key remains held.  Do not turn a
		// repeated start into a toggle or flood it with identical 0x10 frames.
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

// handleShortcutCustom implements the modern shortcut_game.lua contract.
// Set uses shortcut index/kind/config ID. Remove uses the current record row,
// not the visible shortcut index.
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
