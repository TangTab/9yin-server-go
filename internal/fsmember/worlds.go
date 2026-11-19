package fsmember

import (
	"encoding/binary"
	"fmt"
)

// world 消息 payload 布局（§5 定案；名字/账号均为官方定长字段，
// Go 侧按「u32 域 + 名字区」读取，名字区 UTF-16LE 到零止）。
const (
	worldPlayerNameOff = 0x0C // CommandPlayerAdd 名字区（wchar[36]）
	worldPlayerAcctOff = 0x54 // CommandPlayerAdd 账号区（char）
	worldPlayerNameSz  = 0x48
)

// decodeUTF16Z 定长 UTF-16LE 区取零结尾串。
func decodeUTF16Z(b []byte) string {
	var out []rune
	for i := 0; i+1 < len(b); i += 2 {
		v := binary.LittleEndian.Uint16(b[i:])
		if v == 0 {
			break
		}
		out = append(out, rune(v))
	}
	return string(out)
}

// ProcessWorld 复刻 ProcessWorldMsg 跳表（§5 十七型；未知型静默丢弃，
// 官方 default 无操作）。
func (m *Member) ProcessWorld(ident, typ uint32, payload []byte) {
	switch typ {
	case WorldCreateScene:
		m.worldCreateScene(payload)
	case WorldCreateClone:
		m.worldCreateClone(payload)
	case WorldDeleteClone:
		m.worldDeleteClone(payload)
	case WorldCheckScene:
		m.worldCheckScene(payload)
	case WorldSceneAll:
		m.worldSceneAll()
	case WorldSceneAdd:
		m.worldSceneReport(payload, "SceneAdd")
	case WorldSceneRemove:
		m.worldSceneReport(payload, "SceneRemove")
	case WorldPlayerAll:
		m.worldPlayerAll()
	case WorldPlayerAdd:
		m.worldPlayerAdd(payload)
	case WorldPlayerRemove:
		m.worldPlayerRemove(payload)
	case WorldPlayerChange:
		m.worldPlayerChange(payload)
	case WorldRoleInfo, WorldRoleAdd, WorldRoleRemove:
		m.worldRole(typ, payload)
	case WorldItemInfo:
		m.logf("(GameMember::CommandItemInfo)item info %d bytes", len(payload))
	case WorldCloseMember:
		m.BeginClose()
	case WorldDumpPerf:
		m.logf("Dump member logic module performance succeed")
	default:
		// 官方 default：无操作
	}
}

// worldCreateScene 复刻 CommandCreateScene：{sceneID@0, type@4, …}。
func (m *Member) worldCreateScene(p []byte) {
	if len(p) < 8 {
		m.logf("(GameMember::CommandCreateScene)scene invalid")
		return
	}
	id := binary.LittleEndian.Uint32(p[0:])
	typ := binary.LittleEndian.Uint32(p[4:])
	if _, err := m.Scenes.CreateScene(id, SceneType(typ), fmt.Sprintf("scene_%d", id), 0); err != nil {
		m.logf("%v", err)
		return
	}
	m.logf("create scene %d type %d", id, typ)
}

// worldCreateClone 复刻 CommandCreateClone：{protoID@0, cloneID@4}。
func (m *Member) worldCreateClone(p []byte) {
	if len(p) < 8 {
		m.logf("(GameMember::CommandCreateScene)scene invalid")
		return
	}
	proto := binary.LittleEndian.Uint32(p[0:])
	clone := binary.LittleEndian.Uint32(p[4:])
	if _, err := m.Scenes.CloneScene(proto, clone, fmt.Sprintf("clone_%d_of_%d", clone, proto), 0); err != nil {
		m.logf("%v", err)
		return
	}
	m.logf("clone scene succeed: %d <- %d", clone, proto)
}

// worldDeleteClone 复刻 CommandDeleteClone：{sceneID@0}。
func (m *Member) worldDeleteClone(p []byte) {
	if len(p) < 4 {
		return
	}
	id := binary.LittleEndian.Uint32(p[0:])
	if err := m.Scenes.DeleteClone(id); err != nil {
		m.logf("%v", err)
	}
}

// worldCheckScene 复刻 CommandCheckScene：存在性应答回 world。
func (m *Member) worldCheckScene(p []byte) {
	if len(p) < 4 {
		return
	}
	id := binary.LittleEndian.Uint32(p[0:])
	ok := m.Scenes.CanCreateScene(id)
	reply := make([]byte, 8)
	binary.LittleEndian.PutUint32(reply[0:], id)
	if ok {
		binary.LittleEndian.PutUint32(reply[4:], 1)
	}
	if m.SendToWorld != nil {
		_ = m.SendToWorld(WorldCheckScene, reply)
	}
}

// worldSceneAll 复刻 CommandSceneAll：全量场景上报
//（{count@0} + count×{sceneID, type, players}）。
func (m *Member) worldSceneAll() {
	scenes := m.Scenes.Snapshot()
	body := make([]byte, 4+12*len(scenes))
	binary.LittleEndian.PutUint32(body[0:], uint32(len(scenes)))
	for i, s := range scenes {
		off := 4 + i*12
		binary.LittleEndian.PutUint32(body[off:], s.ID)
		binary.LittleEndian.PutUint32(body[off+4:], uint32(s.Type))
		binary.LittleEndian.PutUint32(body[off+8:], uint32(s.CurPlayers))
	}
	if m.SendToWorld != nil {
		_ = m.SendToWorld(WorldSceneAll, body)
	}
}

// worldSceneReport 增量场景上报（SceneAdd/SceneRemove 同构）。
func (m *Member) worldSceneReport(p []byte, who string) {
	if len(p) < 4 {
		m.logf("(GameMember::Command%s)load scene info failed", who)
		return
	}
	id := binary.LittleEndian.Uint32(p[0:])
	body := make([]byte, 4)
	binary.LittleEndian.PutUint32(body[0:], id)
	if m.SendToWorld != nil {
		_ = m.SendToWorld(WorldSceneAdd, body)
	}
}

// worldPlayerAll 复刻 CommandPlayerAll：全量玩家上报
//（{count@0} + count×{ident, serial, sceneID}）。
func (m *Member) worldPlayerAll() {
	players := m.Players.Snapshot()
	body := make([]byte, 4+12*len(players))
	binary.LittleEndian.PutUint32(body[0:], uint32(len(players)))
	for i, pl := range players {
		off := 4 + i*12
		binary.LittleEndian.PutUint32(body[off:], pl.Ident)
		binary.LittleEndian.PutUint32(body[off+4:], pl.Serial)
		binary.LittleEndian.PutUint32(body[off+8:], pl.SceneID)
	}
	if m.SendToWorld != nil {
		_ = m.SendToWorld(WorldPlayerAll, body)
	}
}

// worldPlayerAdd 复刻 CommandPlayerAdd：
// {sceneID@0, ident@4, serial@8, name wchar[36]@0xC, account char@0x54}。
func (m *Member) worldPlayerAdd(p []byte) {
	if len(p) < worldPlayerAcctOff {
		m.logf("(GameMember::CommandPlayerAdd)data error")
		return
	}
	sceneID := binary.LittleEndian.Uint32(p[0:])
	ident := binary.LittleEndian.Uint32(p[4:])
	serial := binary.LittleEndian.Uint32(p[8:])
	name := decodeUTF16Z(p[worldPlayerNameOff : worldPlayerNameOff+worldPlayerNameSz])
	account := cstring(p[worldPlayerAcctOff:])
	if err := m.Players.AddPlayer(sceneID, ident, serial, name, account); err != nil {
		m.logf("(GameMember::CommandPlayerAdd)add failed: %v", err)
		return
	}
	if s, ok := m.Scenes.Get(sceneID); ok {
		m.Scenes.SetPlayerCount(sceneID, s.CurPlayers+1)
	}
}

// worldPlayerRemove 复刻 CommandPlayerRemove：{sceneID@0, ?@4, name@8}。
func (m *Member) worldPlayerRemove(p []byte) {
	if len(p) < 8+4 {
		m.logf("(GameMember::CommandPlayerRemove)player not found")
		return
	}
	sceneID := binary.LittleEndian.Uint32(p[0:])
	name := decodeUTF16Z(p[8:])
	if err := m.Players.RemovePlayer(sceneID, name); err != nil {
		m.logf("%v", err)
		return
	}
	if s, ok := m.Scenes.Get(sceneID); ok && s.CurPlayers > 0 {
		m.Scenes.SetPlayerCount(sceneID, s.CurPlayers-1)
	}
}

// worldPlayerChange 复刻 CommandPlayerChange：{name@…, newScene}。
// 官方语义 ChangePlayerSceneId；名字取区 0（可变长实现：前段 u32 场景 +
// 后段名，按 CommandPlayerRemove 同构处理）。
func (m *Member) worldPlayerChange(p []byte) {
	if len(p) < 8 {
		m.logf("(GameMember::CommandPlayerChange)player not found")
		return
	}
	newScene := binary.LittleEndian.Uint32(p[0:])
	name := decodeUTF16Z(p[4:])
	if err := m.Players.ChangePlayerSceneId(name, newScene); err != nil {
		m.logf("(GameMember::CommandPlayerChange)player not found: %s", name)
	}
}

// worldRole 角色档案三类（RoleInfo/RoleAdd/RoleRemove）——数据面记录，
// 载荷细节待 persist 角色存档合流后对拍细化。
func (m *Member) worldRole(typ uint32, p []byte) {
	if len(p) == 0 {
		m.logf("(GameMember::CommandRoleInfo)load role info failed")
		return
	}
	m.logf("world role message %#x (%d bytes)", typ, len(p))
}

// cstring C 风格定长区取零结尾串。
func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
