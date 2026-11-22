package fssoloworld

import (
	"encoding/binary"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// sendRoleSave 0x04 存档（gm save 命令 / 下线 entryClientClosed 两源）：
// roleState 内存态（GM 热改后的权威）Marshal 成 RST1 json 发 store。
// store 端 merge 库内 meta 后落 nx_roles（role_state_json 覆盖）。
// V1 不做 PlSaving 状态流转与周期 ticker（手动+下线先行）。
func (s *Server) sendRoleSave(pl *Player) bool {
	rst1, err := rolestate.MarshalRoleState(pl.roleState)
	if err != nil {
		s.trace("gm save marshal failed client=%d: %v", pl.Conn1, err)
		return false
	}
	payload := buildSaveStore(pl.Conn1, pl.Conn2, pl.Account, pl.Name, pl.roleState.Position, []byte(rst1))
	return s.inner.SendToStore(0, 0, 0x04, payload)
}

// storeSaveReply 0x03 回执 {client_id, serial, result}（官方
// MessageSave → 0x03 0x58B）：簿记面，trace 即可（玩家可能已离线）。
func (s *Server) storeSaveReply(id1 uint32, p []byte) bool {
	if len(p) < 12 {
		return false
	}
	clientID := binary.LittleEndian.Uint32(p[0:])
	result := binary.LittleEndian.Uint32(p[8:])
	s.trace("(SoloWorld::StoreMsgSave)save reply client=%d result=%d", clientID, result)
	return true
}
