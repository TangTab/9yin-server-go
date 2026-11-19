package fssolomember

import (
	"encoding/binary"

	"github.com/lizhanyin/9yin-server-go/internal/fsmember"
)

// processWorldMsg 官方 ProcessWorldMsg（@0x102A2E50，两级跳表 0x48 槽；§六）。
// 表内未列 id2 → 默认分支：遍历场景表 SendToScene(scene, id2 原样)。
func (s *Server) processWorldMsg(id1, id2 uint32, payload []byte) {
	switch id2 {
	case 0x00:
		s.worldCreateScene(payload)
	case 0x01:
		s.worldCreateClone(payload)
	case 0x02:
		s.worldDeleteClone(payload)
	case 0x03:
		s.worldCheckScene(id1, payload)
	case 0x04:
		s.worldSceneAll()
	case 0x05:
		s.worldSceneAdd(payload)
	case 0x06:
		s.worldSceneRemove(payload)
	case 0x07:
		s.worldPlayerAll()
	case 0x08:
		s.worldPlayerAdd(payload)
	case 0x09:
		s.worldPlayerRemove(payload)
	case 0x0A:
		s.worldPlayerChange(payload)
	case 0x0B:
		s.worldRoleInfo(payload)
	case 0x0C, 0x0D, 0x0E:
		s.worldRole(id2, payload)
	case 0x33, 0x34:
		s.worldItemInfo(id2)
	case 0x3E:
		s.StateMachine(evClose, 0, nil) // CommandCloseMember
	case 0x3F:
		s.opt.Logf("dump logic module performance...")
	case 0x44:
		s.opt.Logf("clear logic module performance")
	case 0x47:
		s.worldMemoryRole()
	default:
		for _, sc := range s.Scenes.Snapshot() {
			s.sendToScene(sc.ID, id2, payload)
		}
	}
}

// worldCreateScene CommandCreateScene：{scene u32, type u32, name/配置…}
// （集群 GameMember 布局；创建器来自逻辑模块——Go 侧静态注册表）。
func (s *Server) worldCreateScene(p []byte) {
	if len(p) < 8 {
		return
	}
	id := le32(p, 0)
	typ := le32(p, 4)
	name := cstringAt(p, 8)
	if _, err := s.Scenes.CreateScene(id, fsmemberSceneType(typ), name, 0); err != nil {
		s.opt.Logf("(SoloMember::CommandCreateScene)%v", err)
	}
}

func (s *Server) worldCreateClone(p []byte) {
	if len(p) < 8 {
		return
	}
	proto, clone := le32(p, 0), le32(p, 4)
	if _, err := s.Scenes.CloneScene(proto, clone, cstringAt(p, 8), 0); err != nil {
		s.opt.Logf("(SoloMember::CommandCreateClone)%v", err)
	}
}

func (s *Server) worldDeleteClone(p []byte) {
	if len(p) < 4 {
		return
	}
	if err := s.Scenes.DeleteClone(le32(p, 0)); err != nil {
		s.opt.Logf("(SoloMember::CommandDeleteClone)%v", err)
	}
}

// worldCheckScene CommandCheckScene：场景存在性应答回 world（0x3 型）。
func (s *Server) worldCheckScene(id1 uint32, p []byte) {
	if len(p) < 4 {
		return
	}
	sceneID := le32(p, 0)
	_, ok := s.Scenes.Get(sceneID)
	reply := make([]byte, 8)
	binary.LittleEndian.PutUint32(reply[0:], sceneID)
	if ok {
		binary.LittleEndian.PutUint32(reply[4:], 1)
	}
	s.inner.SendToWorld(id1, 0x03, reply)
}

func (s *Server) worldSceneAll() {
	scenes := s.Scenes.Snapshot()
	buf := make([]byte, 4+len(scenes)*8)
	binary.LittleEndian.PutUint32(buf[0:], uint32(len(scenes)))
	for i, sc := range scenes {
		off := 4 + i*8
		binary.LittleEndian.PutUint32(buf[off:], sc.ID)
		binary.LittleEndian.PutUint32(buf[off+4:], uint32(sc.CurPlayers))
	}
	s.inner.SendToWorld(0, 0x04, buf)
}

func (s *Server) worldSceneAdd(p []byte) {
	if len(p) < 8 {
		return
	}
	s.worldCreateScene(p)
}

func (s *Server) worldSceneRemove(p []byte) {
	if len(p) < 4 {
		return
	}
	s.Scenes.Remove(le32(p, 0))
}

func (s *Server) worldPlayerAll() {
	players := s.Players.Snapshot()
	buf := make([]byte, 4+len(players)*12)
	binary.LittleEndian.PutUint32(buf[0:], uint32(len(players)))
	for i, pl := range players {
		off := 4 + i*12
		binary.LittleEndian.PutUint32(buf[off:], pl.SceneID)
		binary.LittleEndian.PutUint32(buf[off+4:], pl.Ident)
		binary.LittleEndian.PutUint32(buf[off+8:], pl.Serial)
	}
	s.inner.SendToWorld(0, 0x07, buf)
}

// worldPlayerAdd CommandPlayerAdd：{scene@0, ident@4, serial@8,
// name[36]wchar@0xC, account@0x54}（集群实证布局；solo 同构）。
func (s *Server) worldPlayerAdd(p []byte) {
	if len(p) < 0x58 {
		return
	}
	sceneID, ident, serial := le32(p, 0), le32(p, 4), le32(p, 8)
	name := utf16ZAt(p, 0xC, 36)
	account := cstringAt(p, 0x54)
	if err := s.Players.AddPlayer(sceneID, ident, serial, name, account); err != nil {
		s.opt.Logf("(SoloMember::CommandPlayerAdd)%v", err)
	}
}

func (s *Server) worldPlayerRemove(p []byte) {
	if len(p) < 8 {
		return
	}
	sceneID := le32(p, 0)
	name := utf16ZAt(p, 8, (len(p)-8)/2)
	if err := s.Players.RemovePlayer(sceneID, name); err != nil {
		s.opt.Logf("(SoloMember::CommandPlayerRemove)%v", err)
	}
}

func (s *Server) worldPlayerChange(p []byte) {
	if len(p) < 4 {
		return
	}
	if err := s.Players.ChangePlayerSceneId(utf16ZAt(p, 4, (len(p)-4)/2), le32(p, 0)); err != nil {
		s.opt.Logf("(SoloMember::CommandPlayerChange)%v", err)
	}
}

// worldRoleInfo CommandRoleInfo：角色档案全量重建 + **+0x43C 闩**
// （Loading 门之一，@0x102A1B4D 写点）。
func (s *Server) worldRoleInfo(p []byte) {
	s.mu.Lock()
	s.roleReady = true
	s.mu.Unlock()
}

// worldRole RoleAdd(0xC)/RoleRemove(0xD)/RoleRecreate(0x0E)。
func (s *Server) worldRole(id2 uint32, p []byte) {
	s.opt.Logf("world role message 0x%X (%d bytes)", id2, len(p))
}

func (s *Server) worldItemInfo(id2 uint32) {
	s.opt.Logf("world item info 0x%X", id2)
}

func (s *Server) worldMemoryRole() {
	s.opt.Logf("world memory role")
}

func le32(p []byte, off int) uint32 { return binary.LittleEndian.Uint32(p[off:]) }

func cstringAt(p []byte, off int) string {
	if off >= len(p) {
		return ""
	}
	b := p[off:]
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func utf16ZAt(p []byte, off, maxChars int) string {
	if off >= len(p) {
		return ""
	}
	b := p[off:]
	var runes []rune
	for i := 0; i+1 < len(b) && len(runes) < maxChars; i += 2 {
		u := uint16(b[i]) | uint16(b[i+1])<<8
		if u == 0 {
			break
		}
		runes = append(runes, rune(u))
	}
	return string(runes)
}

// fsmemberSceneType 桥接集群 SceneType（0=prototype/1=clone 语义）。
func fsmemberSceneType(t uint32) fsmember.SceneType { return fsmember.SceneType(t) }
