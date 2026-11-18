package entry

import (
	"fmt"
	"sync"
)

// MaxClientNum 官方 MAX_CLIENT_NUM（ProcSightSend 断言 index < 0x2000）。
const MaxClientNum = 0x2000

// MaxMemberNum 官方 interconnect.h 断言上限：member 编号 0..15（内网
// ident 0x18..0x27）。
const MaxMemberNum = 16

// ClientState 官方 client_t 关键域（CClients 0x4C 字节槽位）。
type ClientState struct {
	Inuse   bool   // +0x00 bInuse：槽位占用（连接存活）
	Online  bool   // +0x01 bOnline：已通过 member 绑定上线
	Ident   uint32 // +0x0C：entry 侧标识 = entryID<<16 | 槽位
	Serial  uint32 // +0x10：递增序号（防陈旧消息错绑）
	Member  int32  // 绑定的 member 编号（-1 = 未绑定 → 消息发 world）
	MemberIdent uint32 // member 侧标识（set_client_online 第 4 域）
	Scene   uint32 // 当前场景（0 = 无）
	X, Y    int32  // +0x44/+0x48 世界坐标（网格放置依据）
	next    int    // +0x3C 场景格链表下一跳
	cell    int    // 当前所在格（-1 = 未入格）
	addr    string
}

// Scene 网关侧场景登记（set_client_posi 驱动懒建）。
type Scene struct {
	ID   uint32
	Grid *sceneGrid
}

// Clients 是 CClients 复刻：固定 0x2000 槽 + 场景表 + 频道表。
type Clients struct {
	mu      sync.Mutex
	clients [MaxClientNum]ClientState
	scenes  map[uint32]*Scene
	// channels: channelID → 客户端槽集合
	channels map[uint32]map[int]bool
	// sight 螺旋偏移表（懒建，全场景共享）
	sight []sightOffset
	// serial 计数器（官方从 1 起且跳过 0）
	serial uint32
	// 统计：连接数 / 在线数
	connectCount int
	inuseCount   int
	onlineCount  int
}

func NewClients() *Clients {
	return &Clients{
		scenes:   make(map[uint32]*Scene),
		channels: make(map[uint32]map[int]bool),
	}
}

// Alloc 分配槽位（OnAccept）：ident = entryID<<16|槽位，serial 单调递增。
// 返回槽位下标；池满返回 -1（官方 "client array too big"）。
func (c *Clients) Alloc(entryID uint16, addr string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 0; i < MaxClientNum; i++ {
		if !c.clients[i].Inuse {
			c.serial++
			if c.serial == 0 { // 官方跳过 0 序号
				c.serial = 1
			}
			st := &c.clients[i]
			st.Inuse = true
			st.Online = false
			st.Ident = uint32(entryID)<<16 | uint32(i)
			st.Serial = c.serial
			st.Member = -1
			st.MemberIdent = 0
			st.Scene = 0
			st.X, st.Y = 0, 0
			st.next = -1
			st.cell = -1
			st.addr = addr
			c.connectCount++
			c.inuseCount++
			return i
		}
	}
	return -1
}

// Lookup 按标识找活跃槽；serial 不匹配返回 nil（官方 "serial not match"）。
func (c *Clients) Lookup(ident, serial uint32) *ClientState {
	idx := int(ident & 0xFFFF) // 官方收包即 ident&0xFFFF
	if idx < 0 || idx >= MaxClientNum {
		return nil
	}
	st := &c.clients[idx]
	if !st.Inuse || st.Serial != serial || st.Ident != ident {
		return nil
	}
	return st
}

// Release 释放槽位（OnClose）：摘格、摘频道。
func (c *Clients) Release(idx int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := &c.clients[idx]
	if !st.Inuse {
		return
	}
	c.ungridLocked(idx)
	c.channelRemoveAllLocked(idx)
	if st.Online {
		c.onlineCount--
	}
	st.Inuse = false
	st.Online = false
	c.connectCount--
	c.inuseCount--
}

// SetOnline 复刻 set_client_online：标记上线并绑定 member。
func (c *Clients) SetOnline(ident, serial uint32, member int32, memberIdent uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("client not in use: %#x/%d", ident, serial)
	}
	if !st.Online {
		st.Online = true
		c.onlineCount++
	}
	st.Member = member
	st.MemberIdent = memberIdent
	return nil
}

// SetInuse 复刻 set_client_inuse（member 侧确认占用/回收）。
func (c *Clients) SetInuse(ident, serial uint32, inuse bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("client not in use: %#x/%d", ident, serial)
	}
	st.Inuse = inuse
	if inuse {
		c.inuseCount++
	} else {
		c.inuseCount--
	}
	return nil
}

// ungridLocked 摘除格链表（调用方持锁）。
func (c *Clients) ungridLocked(idx int) {
	st := &c.clients[idx]
	if st.cell < 0 {
		return
	}
	sc := c.scenes[st.Scene]
	if sc != nil && sc.Grid != nil && st.cell < len(sc.Grid.cells) {
		head := sc.Grid.cells[st.cell]
		if head == idx {
			sc.Grid.cells[st.cell] = st.next
		} else {
			for p := head; p >= 0; p = c.clients[p].next {
				if c.clients[p].next == idx {
					c.clients[p].next = st.next
					break
				}
			}
		}
	}
	st.cell = -1
	st.next = -1
}

// gridLocked 放入场景格（调用方持锁）；场景/网格懒建。
func (c *Clients) gridLocked(idx int, sceneID, x0, y0, width, height uint32, x, y int32) {
	st := &c.clients[idx]
	sc := c.scenes[sceneID]
	if sc == nil {
		sc = &Scene{ID: sceneID, Grid: newSceneGrid(int32(x0), int32(y0), int32(width), int32(height))}
		c.scenes[sceneID] = sc
	} else if !sc.Grid.sameDims(int32(x0), int32(y0), int32(width), int32(height)) {
		// 官方：维度变化 → 释放旧网格整场重建
		old := sc.Grid
		sc.Grid = newSceneGrid(int32(x0), int32(y0), int32(width), int32(height))
		if old != nil {
			for i := 0; i < MaxClientNum; i++ {
				if c.clients[i].Inuse && c.clients[i].Scene == sceneID {
					c.ungridLocked(i)
				}
			}
		}
	}
	c.ungridLocked(idx)
	st.Scene = sceneID
	st.X, st.Y = x, y
	if ci := sc.Grid.cellIndex(x, y); ci >= 0 {
		st.cell = ci
		st.next = sc.Grid.cells[ci]
		sc.Grid.cells[ci] = idx
	}
}

// SetPosi 复刻 set_client_posi（40B）：落格 + 坐标更新。
func (c *Clients) SetPosi(ident, serial uint32, sceneID, x0, y0, width, height uint32, x, y int32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("client not in use: %#x/%d", ident, serial)
	}
	c.gridLocked(idxOf(ident), sceneID, x0, y0, width, height, x, y)
	return nil
}

// Move 复刻 set_client_move：批量更新同场景客户端坐标（16B/条
// {ident,serial,x,y}；场景不符跳过 —— 官方 "client scene id not match"）。
func (c *Clients) Move(sceneID uint32, entries [][4]uint32) (moved int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range entries {
		st := c.Lookup(e[0], e[1])
		if st == nil || st.Scene != sceneID {
			continue
		}
		idx := idxOf(e[0])
		sc := c.scenes[sceneID]
		if sc == nil {
			continue
		}
		c.ungridLocked(idx)
		st.X, st.Y = int32(e[2]), int32(e[3])
		if ci := sc.Grid.cellIndex(st.X, st.Y); ci >= 0 {
			st.cell = ci
			st.next = sc.Grid.cells[ci]
			sc.Grid.cells[ci] = idx
		}
		moved++
	}
	return moved
}

// ChannelAdd/ChannelRemove/ClearChannel/ChannelMembers 复刻频道面。
func (c *Clients) ChannelAdd(ident, serial, channel uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("client index error: %#x/%d", ident, serial)
	}
	set := c.channels[channel]
	if set == nil {
		set = make(map[int]bool)
		c.channels[channel] = set
	}
	set[idxOf(ident)] = true
	return nil
}

func (c *Clients) ChannelRemove(ident, serial, channel uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("client index error: %#x/%d", ident, serial)
	}
	if set := c.channels[channel]; set != nil {
		delete(set, idxOf(ident))
		if len(set) == 0 {
			delete(c.channels, channel)
		}
	}
	return nil
}

func (c *Clients) channelRemoveAllLocked(idx int) {
	for id, set := range c.channels {
		if set[idx] {
			delete(set, idx)
			if len(set) == 0 {
				delete(c.channels, id)
			}
		}
	}
}

func (c *Clients) ClearChannel(channel uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channel)
}

func (c *Clients) ChannelMembers(channel uint32) []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	set := c.channels[channel]
	if len(set) == 0 {
		return nil
	}
	out := make([]int, 0, len(set))
	for idx := range set {
		out = append(out, idx)
	}
	return out
}

// SightMembers 复刻 ProcSightSend 的螺旋遍历：中心 (x,y)，半径 radius
// （均为官方坐标单位）。返回视野内客户端槽列表（不含中心客户端自身）。
func (c *Clients) SightMembers(ident uint32, sceneID uint32, x, y, radius int32) []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	sc := c.scenes[sceneID]
	if sc == nil || sc.Grid == nil {
		return nil // 官方 "no scene grid"
	}
	g := sc.Grid
	if c.sight == nil {
		c.sight = buildSightTable()
	}
	// cell 级上限 = (radius+cell)²，与官方一致（格内任意点最远补一格）
	lim := float64(radius) + float64(g.cell)
	lim2 := lim * lim
	self := idxOf(ident)
	seen := make(map[int]bool)
	var out []int
	for _, off := range c.sight {
		if float64(off.dist)*float64(g.cell)*float64(g.cell) > lim2 {
			break // 升序早退
		}
		ci := g.cellIndex(x+off.dx*g.cell, y+off.dy*g.cell)
		if ci < 0 {
			continue
		}
		for p := g.cells[ci]; p >= 0; p = c.clients[p].next {
			if p == self || seen[p] {
				continue
			}
			seen[p] = true
			st := &c.clients[p]
			if !st.Inuse || !st.Online {
				continue // 官方断言 bInuse/bOnline
			}
			dx := float64(st.X) - float64(x)
			dy := float64(st.Y) - float64(y)
			if dx*dx+dy*dy <= lim2 {
				out = append(out, p)
			}
		}
	}
	return out
}

// OnlineMembers / AllMembers 返回目标槽列表（send_online / send_all）。
func (c *Clients) OnlineMembers() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []int
	for i := 0; i < MaxClientNum; i++ {
		if c.clients[i].Inuse && c.clients[i].Online {
			out = append(out, i)
		}
	}
	return out
}

func (c *Clients) AllMembers() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []int
	for i := 0; i < MaxClientNum; i++ {
		if c.clients[i].Inuse {
			out = append(out, i)
		}
	}
	return out
}

// Stats 返回 (connect, inuse, online) 计数。
func (c *Clients) Stats() (int, int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectCount, c.inuseCount, c.onlineCount
}

// Client 暴露槽位快照（维护面查询用）。
func (c *Clients) Client(idx int) ClientState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clients[idx]
}

func idxOf(ident uint32) int { return int(ident & 0xFFFF) }
