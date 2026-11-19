package fssolopublic

import (
	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// processWorldMsg 官方 ProcessWorldMsg（@0x100C7170 跳表，13 处器）。
func (s *Server) processWorldMsg(typ uint32, p []byte) {
	switch typ {
	case worldServerState:
		s.worldServerState(p)
	case worldClosePublic:
		s.stateMachine(evWorldClose, 0)
	case worldSceneAll:
		s.worldSceneAll(p)
	case worldSceneAdd:
		s.worldSceneAdd(p)
	case worldSceneRemove:
		s.worldSceneRemove(p)
	case worldPlayerAll:
		s.worldPlayerAll(p)
	case worldPlayerAdd:
		s.worldPlayerAdd(p)
	case worldPlayerRemove:
		s.worldPlayerRemove(p)
	case worldPlayerChange:
		s.worldPlayerChange(p)
	case worldRoleInfo:
		s.worldRoleInfo(p)
	case worldRoleAdd:
		s.worldRoleMutate(p, false, false)
	case worldRoleRemove:
		s.worldRoleMutate(p, true, false)
	case worldRoleRecreate:
		s.worldRoleMutate(p, true, true)
	default:
		s.logf("(SoloPublic::ProcessWorldMsg)unknown type %d", typ)
	}
}

// worldServerState 官方 type1：public_res_server_state_t 0x24B → Callee
// OnServerState（集群同源：{char[16]@0, 5×int@0x10}）。
func (s *Server) worldServerState(p []byte) {
	if len(p) != sizeServerState {
		s.logf("(SoloPublic::WorldServerState)len == sizeof(public_res_server_state_t)=0x24，得 %d", len(p))
		return
	}
	state := fspublic.ServerState{
		Name:  cstr(p[0:0x10]),
		Props: [5]int32{i32at(p, 0x10), i32at(p, 0x14), i32at(p, 0x18), i32at(p, 0x1C), i32at(p, 0x20)},
	}
	s.callee.OnServerState(nil, state)
}

// worldSceneAll 官方 type3：{int count@0, 项 0x90×count}（member_forw_
// scene_all_t，len ≥ sizeof(头)+count×0x90）。逐项 AddScene（幂等）。
func (s *Server) worldSceneAll(p []byte) {
	const hdr = 4
	if len(p) < hdr {
		s.logf("(SoloPublic::WorldSceneAll)len >= sizeof(member_forw_scene_all_t): %d", len(p))
		return
	}
	count := int(u32at(p, 0))
	if len(p) < hdr+count*sizeSceneAdd {
		s.logf("(SoloPublic::WorldSceneAll)receive size error: %d", len(p))
		return
	}
	for i := 0; i < count; i++ {
		s.applySceneAdd(p[hdr+i*sizeSceneAdd : hdr+(i+1)*sizeSceneAdd])
	}
}

// worldSceneAdd 官方 type4：0x90B（len 断言）。
func (s *Server) worldSceneAdd(p []byte) {
	if len(p) != sizeSceneAdd {
		s.logf("(SoloPublic::WorldSceneAdd)len == sizeof(member_forw_scene_add_t)=0x90，得 %d", len(p))
		return
	}
	if !s.applySceneAdd(p) {
		s.logf("(SoloPublic::WorldSceneAdd)scene not found: %s", cstr(p[0x10:0x50]))
	}
}

// applySceneAdd 解析 {int scene@0,int member@4,int ident@8,int clone@0xC,
// char name[0x40]@0x10, char resName[0x40]@0x50} → CPubScenes::AddScene。
func (s *Server) applySceneAdd(p []byte) bool {
	s.Scenes.AddScene(u32at(p, 0), u32at(p, 4), u32at(p, 8), u32at(p, 0xC),
		cstr(p[0x10:0x50]), cstr(p[0x50:0x90]))
	_, ok := s.Scenes.GetScene(u32at(p, 0))
	return ok
}

// worldSceneRemove 官方 type5：{int scene@0}（len 断言 8）。
func (s *Server) worldSceneRemove(p []byte) {
	if len(p) != 8 {
		s.logf("(SoloPublic::WorldSceneRemove)len == sizeof(member_forw_scene_remove_t)=8，得 %d", len(p))
		return
	}
	if !s.Scenes.RemoveScene(u32at(p, 0)) {
		s.logf("(SoloPublic::WorldSceneRemove)scene not exists: %d", u32at(p, 0))
	}
}

// worldPlayerAll 官方 type6：{int count@0, 项 0x9C×count}。
func (s *Server) worldPlayerAll(p []byte) {
	const hdr = 4
	if len(p) < hdr {
		s.logf("(SoloPublic::WorldPlayerAll)len >= sizeof(member_forw_player_all_t): %d", len(p))
		return
	}
	count := int(u32at(p, 0))
	if len(p) < hdr+count*sizePlayerAdd {
		s.logf("(SoloPublic::WorldPlayerAll)receive size error: %d", len(p))
		return
	}
	for i := 0; i < count; i++ {
		s.applyPlayerAdd(p[hdr+i*sizePlayerAdd : hdr+(i+1)*sizePlayerAdd])
	}
}

// worldPlayerAdd 官方 type7：0x9CB {int member@0, int scene@4, int ident@8,
// int@0xC, wstr name[0x48]@0x10, char account[0x44]@0x58}。
func (s *Server) worldPlayerAdd(p []byte) {
	if len(p) != sizePlayerAdd {
		s.logf("(SoloPublic::WorldPlayerAdd)len == sizeof(member_forw_player_add_t)=0x9C，得 %d", len(p))
		return
	}
	s.applyPlayerAdd(p)
}

func (s *Server) applyPlayerAdd(p []byte) {
	s.Players.AddPlayer(u32at(p, 0), u32at(p, 4), u32at(p, 8),
		fixedWide(p[0x10:0x58]), cstr(p[0x58:0x9C]))
}

// worldPlayerRemove 官方 type8：0x50B {int member@0, int scene@4,
// wstr name[0x48]@8}。
func (s *Server) worldPlayerRemove(p []byte) {
	if len(p) != sizePlayerRemove {
		s.logf("(SoloPublic::WorldPlayerRemove)len == sizeof(member_forw_player_remove_t)=0x50，得 %d", len(p))
		return
	}
	if !s.Players.RemovePlayer(u32at(p, 0), u32at(p, 4), fixedWide(p[8:0x50])) {
		s.logf("(SoloPublic::WorldPlayerRemove)player not found: %s", fixedWide(p[8:0x50]))
	}
}

// worldPlayerChange 官方 type9：0x54B {int member@0, int scene@4, ?@8,
// wstr name[0x48]@0xC} → CPubPlayers::CleanPlayerConnect（换线清理）。
func (s *Server) worldPlayerChange(p []byte) {
	if len(p) != sizePlayerChange {
		s.logf("(SoloPublic::WorldPlayerChange)len == sizeof(member_forw_player_change_t)=0x54，得 %d", len(p))
		return
	}
	if !s.Players.CleanPlayerConnect(fixedWide(p[0xC:0x54])) {
		s.logf("(SoloPublic::WorldPlayerChange)player not found: %s", fixedWide(p[0xC:0x54]))
	}
}

// worldRoleInfo 官方 type10（0x34B member_forw_role_info_t）：+0x176=0 →
// CNameUid::Clear → 逐项 Add → +0x176=1（LoadTrades 双门之一）。
// 线格式：{int@0, int@4, int count@8, 项@0xC}。
func (s *Server) worldRoleInfo(p []byte) {
	if len(p) != sizeRoleInfo {
		s.logf("(SoloPublic::WorldRoleInfo)len >= sizeof(member_forw_role_info_t)=0x34，得 %d", len(p))
		return
	}
	if u32at(p, 4) == 0 {
		s.logf("(SoloPublic::WorldRoleInfo)load failed")
		return
	}
	s.nameuids.Store(false)
	s.Names.Clear()
	count := int(u32at(p, 8))
	// 项区 0x28 字节/项（0x34 头区 - 0xC）：wstr name[0x20] + char uid[8]
	const entry = 0x28
	if len(p) < 0xC+count*entry {
		s.logf("(SoloPublic::WorldRoleInfo)receive size error")
		return
	}
	for i := 0; i < count; i++ {
		e := p[0xC+i*entry : 0xC+(i+1)*entry]
		s.Names.Add(fixedWide(e[0:0x20]), cstr(e[0x20:0x28]))
	}
	s.nameuids.Store(true)
}

// worldRoleMutate 官方 type11/12/13（0x6CB member_forw_role_{add,remove,
// recreate}_t）：{char name[0x24]@0, char newname[0x24]@0x48} →
// NameUid Remove(+Add)。
func (s *Server) worldRoleMutate(p []byte, remove, recreate bool) {
	if len(p) != sizeRoleMutate {
		s.logf("(SoloPublic::WorldRoleMutate)len == sizeof(member_forw_role_t)=0x6C，得 %d", len(p))
		return
	}
	name := cstr(p[0:0x24])
	newname := cstr(p[0x48:0x6C])
	if remove {
		s.Names.Remove(name, "")
		if recreate {
			s.Names.Add(newname, name)
		}
		return
	}
	s.Names.Add(name, newname)
}
