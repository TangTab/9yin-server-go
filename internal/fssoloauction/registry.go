package fssoloauction

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// ---- CAuctionScenes（AuctionScenes.obj）----
//
// scene_t 表，按 scene_id 寻址；normal/clone 双计数
//（GetNormalSceneMax/Count、GetCloneSceneMax/Count）。

// SceneInfo 是 scene_t 投影（AddScene(server, member, scene, clone, name,
// resource)——WorldSceneAdd @0x100B30E6 实参序）。
type SceneInfo struct {
	ServerID int32
	MemberID int32
	SceneID  int32
	Clone    int32
	Name     string
	Resource string
}

// Scenes 复刻 CAuctionScenes。
type Scenes struct {
	byID  map[int32]*SceneInfo
	order []int32
}

// NewScenes 建空表。
func NewScenes() *Scenes { return &Scenes{byID: map[int32]*SceneInfo{}} }

// Add 官方 AddScene（已存在返回 false）。
func (s *Scenes) Add(serverID, memberID, sceneID, clone int32, name, resource string) bool {
	if _, ok := s.byID[sceneID]; ok {
		return false
	}
	s.byID[sceneID] = &SceneInfo{
		ServerID: serverID, MemberID: memberID, SceneID: sceneID,
		Clone: clone, Name: name, Resource: resource,
	}
	s.order = append(s.order, sceneID)
	return true
}

// Remove 官方 RemoveScene。
func (s *Scenes) Remove(sceneID int32) bool {
	if _, ok := s.byID[sceneID]; !ok {
		return false
	}
	delete(s.byID, sceneID)
	for i, id := range s.order {
		if id == sceneID {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// Get 官方 GetScene。
func (s *Scenes) Get(sceneID int32) (*SceneInfo, bool) {
	v, ok := s.byID[sceneID]
	return v, ok
}

// Clear 全量重建前清表（WorldSceneAll 语义）。
func (s *Scenes) Clear() {
	s.byID = map[int32]*SceneInfo{}
	s.order = nil
}

// Counts 返回 (normal, clone)。clone 标非 0 计入副本。
func (s *Scenes) Counts() (normal, clone int) {
	for _, sc := range s.byID {
		if sc.Clone != 0 {
			clone++
		} else {
			normal++
		}
	}
	return
}

// Dump 官方 Dump（文件改为日志呈现）。
func (s *Scenes) Dump() string {
	normal, clone := s.Counts()
	var b strings.Builder
	fmt.Fprintf(&b, "scenes %d (normal/clone %d/%d):\r\n", len(s.byID), normal, clone)
	for _, id := range s.order {
		sc := s.byID[id]
		fmt.Fprintf(&b, "  scene %d: server=%d member=%d clone=%d name=%s res=%s\r\n",
			sc.SceneID, sc.ServerID, sc.MemberID, sc.Clone, sc.Name, sc.Resource)
	}
	return b.String()
}

// ---- CAuctionPlayers（AuctionPlayers.obj）----

// PlayerInfo 是 player_t 投影（AddPlayer(connect, server, player, online,
// wname, account)——WorldPlayerAdd @0x100B374F 实参序）。
type PlayerInfo struct {
	ConnectID int32
	ServerID  int32
	PlayerID  int32
	Online    bool
	Name      string
	Account   string
}

// Players 复刻 CAuctionPlayers（名字/账号双索引 + 在离线计数）。
type Players struct {
	list   []*PlayerInfo
	byName map[string]*PlayerInfo
}

// NewPlayers 建空表。
func NewPlayers() *Players { return &Players{byName: map[string]*PlayerInfo{}} }

// Add 官方 AddPlayer（同名已存在返回 false）。
func (p *Players) Add(connectID, serverID, playerID, online int32, name, account string) bool {
	if _, ok := p.byName[name]; ok {
		return false
	}
	pi := &PlayerInfo{
		ConnectID: connectID, ServerID: serverID, PlayerID: playerID,
		Online: online != 0, Name: name, Account: account,
	}
	p.list = append(p.list, pi)
	p.byName[name] = pi
	return true
}

// Get 官方 GetPlayer(server, player, wname)：任一键命中即可。
func (p *Players) Get(serverID, playerID int32, name string) *PlayerInfo {
	if name != "" {
		if pi, ok := p.byName[name]; ok {
			return pi
		}
	}
	for _, pi := range p.list {
		if pi.ServerID == serverID && pi.PlayerID == playerID {
			return pi
		}
	}
	return nil
}

// Remove 官方 RemovePlayer。
func (p *Players) Remove(serverID, playerID int32, name string) bool {
	pi := p.Get(serverID, playerID, name)
	if pi == nil {
		return false
	}
	delete(p.byName, pi.Name)
	for i, v := range p.list {
		if v == pi {
			p.list = append(p.list[:i], p.list[i+1:]...)
			break
		}
	}
	return true
}

// SetOnline 官方 WorldPlayerChange 语义：改在线位。
func (p *Players) SetOnline(serverID, playerID int32, name string, online bool) bool {
	pi := p.Get(serverID, playerID, name)
	if pi == nil {
		return false
	}
	pi.Online = online
	return true
}

// CleanConnect 官方 CleanPlayerConnect：置离线。
func (p *Players) CleanConnect(name string) bool {
	pi := p.byName[name]
	if pi == nil {
		return false
	}
	pi.Online = false
	return true
}

// Counts 返回 (online, offline)。
func (p *Players) Counts() (online, offline int) {
	for _, pi := range p.list {
		if pi.Online {
			online++
		} else {
			offline++
		}
	}
	return
}

// Count 是总玩家数（GetPlayerCount）。
func (p *Players) Count() int { return len(p.list) }

// Dump 官方 Dump。
func (p *Players) Dump() string {
	online, offline := p.Counts()
	var b strings.Builder
	fmt.Fprintf(&b, "players %d (online/offline %d/%d):\n", len(p.list), online, offline)
	for _, pi := range p.list {
		fmt.Fprintf(&b, "  %s account=%s server=%d player=%d online=%v\n",
			pi.Name, pi.Account, pi.ServerID, pi.PlayerID, pi.Online)
	}
	return b.String()
}

// ---- CNameUid（NameUid.obj：角色名↔uid 双向哈希 + deleted 位）----

// NameUid 复刻 CNameUid。
type NameUid struct {
	nameToUid map[string]string
	uidToName map[string]string
	deleted   map[string]bool
}

// NewNameUid 建空表。
func NewNameUid() *NameUid {
	return &NameUid{nameToUid: map[string]string{}, uidToName: map[string]string{}, deleted: map[string]bool{}}
}

// Add 官方 Add(name, uid, deleted)（已有返回 false）。
func (n *NameUid) Add(name, uid string, isDeleted bool) bool {
	if _, ok := n.nameToUid[name]; ok {
		return false
	}
	n.nameToUid[name] = uid
	n.uidToName[uid] = name
	n.deleted[name] = isDeleted
	return true
}

// Remove 官方 Remove(name, uid)。
func (n *NameUid) Remove(name, uid string) {
	delete(n.nameToUid, name)
	delete(n.uidToName, uid)
	delete(n.deleted, name)
}

// FindName 官方 FindName(uid)。
func (n *NameUid) FindName(uid string) bool { _, ok := n.uidToName[uid]; return ok }

// FindUid 官方 FindUid(name)。
func (n *NameUid) FindUid(name string) bool { _, ok := n.nameToUid[name]; return ok }

// GetUid 官方 GetUid(name)（无则空）。
func (n *NameUid) GetUid(name string) string { return n.nameToUid[name] }

// GetName 官方 GetName(uid)。
func (n *NameUid) GetName(uid string) string { return n.uidToName[uid] }

// GetDeleted 官方 GetDeleted(name)。
func (n *NameUid) GetDeleted(name string) bool { return n.deleted[name] }

// Clear 官方 Clear（WorldRoleInfo 全量重建前清表 @0x100B3AD3）。
func (n *NameUid) Clear() {
	n.nameToUid = map[string]string{}
	n.uidToName = map[string]string{}
	n.deleted = map[string]bool{}
}

// Count 官方 GetCount。
func (n *NameUid) Count() int { return len(n.nameToUid) }

// ---- World 消息定长布局（反汇编实证）----

// member_forw_scene_add_t（0x90，WorldSceneAdd @0x100B306D 断言）：
//
//	+0x00 i32 server_id  +0x04 i32 member_id  +0x08 i32 scene_id
//	+0x0C i32 clone      +0x10 char name[0x40]  +0x50 char resource[0x40]
const sceneAddSize = 0x90

func parseSceneAdd(b []byte) (serverID, memberID, sceneID, clone int32, name, resource string, ok bool) {
	if len(b) != sceneAddSize {
		return 0, 0, 0, 0, "", "", false
	}
	serverID = int32(binary.LittleEndian.Uint32(b[0:]))
	memberID = int32(binary.LittleEndian.Uint32(b[4:]))
	sceneID = int32(binary.LittleEndian.Uint32(b[8:]))
	clone = int32(binary.LittleEndian.Uint32(b[0xC:]))
	return serverID, memberID, sceneID, clone, fixedStr(b[0x10 : 0x10+0x40]), fixedStr(b[0x50:]), true
}

// member_forw_player_add_t（0x9C，WorldPlayerAdd @0x100B36F3 断言）：
//
//	+0x00 i32 connect_id  +0x04 i32 server_id  +0x08 i32 player_id
//	+0x0C i32 online      +0x10 wchar name[0x24]  +0x58 char account[0x44]
const playerAddSize = 0x9C

func parsePlayerAdd(b []byte) (connectID, serverID, playerID, online int32, name, account string, ok bool) {
	if len(b) != playerAddSize {
		return 0, 0, 0, 0, "", "", false
	}
	connectID = int32(binary.LittleEndian.Uint32(b[0:]))
	serverID = int32(binary.LittleEndian.Uint32(b[4:]))
	playerID = int32(binary.LittleEndian.Uint32(b[8:]))
	online = int32(binary.LittleEndian.Uint32(b[0xC:]))
	return connectID, serverID, playerID, online, fixedWide(b[0x10 : 0x10+0x48]), fixedStr(b[0x58:]), true
}
