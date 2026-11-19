package fsmember

import (
	"encoding/binary"
)

// ProcessEntry 复刻 PutEntryMessage（§7）：entry（src 8..0x17）消息不进
// 三队列，直投两路——
//
//	type==3（set_client_online）→ CPlayerBuffers（PlayerSingleBuffer 语义）
//	其余 → SendToScene(…, 0x10 CommandClientMessage, …)
func (m *Member) ProcessEntry(ident, typ uint32, payload []byte) {
	if typ == ForwSetOnline && len(payload) >= 8 {
		ident := binary.LittleEndian.Uint32(payload[0:])
		serial := binary.LittleEndian.Uint32(payload[4:])
		m.PutPlayerBuffer(ident, serial, payload[8:])
		return
	}
	if m.OnSceneCommand != nil {
		m.OnSceneCommand(0, SceneCmdClientMessage, payload)
	}
}

// SendToClient 复刻 GameMember::SendToClient（§7 定案）：
// serial 高 16 位=EntryIdx → dst=8+EntryIdx；type=7（entry_forw
// send_to_client）；帧体 {ident,serial}(8B)+payload。
func (m *Member) SendToClient(ident, serial uint32, payload []byte) error {
	if m.SendToEntry == nil {
		return nil
	}
	body := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(body[0:], ident)
	binary.LittleEndian.PutUint32(body[4:], serial)
	copy(body[8:], payload)
	return m.SendToEntry(EntryIdentOfSerial(serial), ForwSendClient, body)
}

// SendClientOnline 复刻 SendClientOnline（forw 15 send_online 的二级
// kind=2 仅在线广播语义留给 entry 侧；member 侧仅透传）。
func (m *Member) SendClientOnline(payload []byte) error {
	if m.SendToEntry == nil {
		return nil
	}
	return m.SendToEntry(EntryIdentBase, 15, payload)
}

// ProcessPublic 复刻 ProcessPublicMsg 十四型（§9）：数据操作落 Pub
// 镜像，PublicMessage(0xD) 透传场景（"scene not active/no scene" 门）。
func (m *Member) ProcessPublic(ident, typ uint32, payload []byte) {
	readStr := func(off, n int) string {
		if len(payload) < off+n {
			return ""
		}
		return cstring(payload[off : off+n])
	}
	readName := func(off int) string { return readStr(off, 32) }
	switch typ {
	case PublicSpaces:
		var names []string
		for off := 0; off+32 <= len(payload); off += 32 {
			if n := readStr(off, 32); n != "" {
				names = append(names, n)
			}
		}
		if err := m.Pub.DeclareSpaces(names); err != nil {
			m.logf("%v", err)
		}
	case PublicData:
		if err := m.Pub.EnsureData(readName(0), readName(32)); err != nil {
			m.logf("%v", err)
		}
	case PublicAdd:
		if err := m.Pub.AddData(readName(0), readName(32)); err != nil {
			m.logf("%v", err)
		}
	case PublicRemove:
		if err := m.Pub.RemoveData(readName(0), readName(32)); err != nil {
			m.logf("%v", err)
		}
	case PublicAddAttr:
		if err := m.Pub.SetAttr(readName(0), readName(32), readName(64), readStr(96, 32)); err != nil {
			m.logf("%v", err)
		}
	case PublicRemoveAttr:
		if err := m.Pub.DelAttr(readName(0), readName(32), readName(64)); err != nil {
			m.logf("%v", err)
		}
	case PublicAddRec:
		if err := m.Pub.AddRec(readName(0), readName(32), readName(64), nil); err != nil {
			m.logf("%v", err)
		}
	case PublicRemoveRec:
		if err := m.Pub.DelRec(readName(0), readName(32), readName(64)); err != nil {
			m.logf("%v", err)
		}
	case PublicRecAddRow, PublicRecRemoveRow, PublicRecClear, PublicRecChange, PublicRecSetRow:
		// 行级操作：载荷细节（列数可变）待 world/public 联调对拍后细化，
		// 记录面保持镜像容器（Pub.Recs）就绪。
		m.logf("public rec message %#x (%d bytes)", typ, len(payload))
	case PublicMessage:
		if len(payload) < 4 {
			m.logf("(GameMember::PublicMessage)no scene")
			return
		}
		sceneID := binary.LittleEndian.Uint32(payload[0:])
		s, ok := m.Scenes.Get(sceneID)
		if !ok {
			m.logf("(GameMember::PublicMessage)no scene: %d", sceneID)
			return
		}
		if !s.Active {
			m.logf("(GameMember::PublicMessage)scene not active: %d", sceneID)
			return
		}
		if m.OnSceneCommand != nil {
			m.OnSceneCommand(sceneID, PublicMessage, payload[4:])
		}
	default:
	}
}

// ProcessStore 复刻 ProcessStoreMsg 四型（§10）。
func (m *Member) ProcessStore(ident, typ uint32, payload []byte) {
	readU32 := func(off int) uint32 {
		if len(payload) < off+4 {
			return 0
		}
		return binary.LittleEndian.Uint32(payload[off:])
	}
	switch typ {
	case StoreChunkInfo:
		var ids []uint32
		for off := 4; off+4 <= len(payload); off += 4 {
			ids = append(ids, readU32(off))
		}
		if err := m.Chunks.SetInfo(readU32(0), ids); err != nil {
			m.logf("%v", err)
		}
	case StoreLoadChunk:
		var ids []uint32
		for off := 4; off+4 <= len(payload); off += 4 {
			ids = append(ids, readU32(off))
		}
		if err := m.Chunks.Load(readU32(0), ids); err != nil {
			m.logf("%v", err)
		}
	case StoreSaveChunk:
		if err := m.Chunks.Save(readU32(0), readU32(4)); err != nil {
			m.logf("%v", err)
		}
	case StoreDeleteChunk:
		if err := m.Chunks.Delete(readU32(0), readU32(4)); err != nil {
			m.logf("%v", err)
		}
	default:
	}
}
