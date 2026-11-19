package fspublic

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// PublicState 本地状态机（PublicStateMachine 跳表 + GetStateName 双证）。
type PublicState int32

const (
	StateUnknown     PublicState = 0 // 启动中
	StateLoadSpaces  PublicState = 1 // 向 store 拉取空间数据
	StateOpened      PublicState = 2 // 开放服务
	StateClosing     PublicState = 3 // 批量回存
	StateClosed      PublicState = 4 // 终态
)

// String 状态名（官方 GetStateName 同表）。
func (s PublicState) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateLoadSpaces:
		return "LoadSpaces"
	case StateOpened:
		return "Opened"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	}
	return fmt.Sprintf("State(%d)", int32(s))
}

// 状态机事件（PublicStateMachine arg1）。
const (
	smEventTick          = 0 // 每秒 tick（DoWait）
	smEventStoreConn     = 1 // store 连接
	smEventSpacesLoaded  = 2 // 全部空间装载完
	smEventWorldClose    = 3 // world 关闭令
)

// 广播包定案（BroadcastServerInfo，0x24B）：kind=4/magic=0x300 +
// ip[16] + port + DistrictId@0x1C + ServerId@0x20。
const (
	broadcastKind  = 4
	broadcastMagic = 0x300
	broadcastSize  = 0x24
)

// msg 内网消息（队列条目）。
type msg struct {
	ident   uint32
	typ     uint32
	payload []byte
}

// Public 复刻 GamePublic 服务本体：状态机 + 四队列 + 三消息面 +
// store 装载链 + 周期广播。
type Public struct {
	State atomic.Int32 // PublicState

	DistrictID uint32
	ServerID   uint32

	// SaveBatch 官方 SavePubDataBatchSize（缺省 50，夹紧 1..1000）。
	SaveBatch int

	Spaces *Spaces
	Timer  *PubTimer

	// Callee 逻辑模块（FsPubLogic 等价；nil=空逻辑）。
	Callee Callee

	// SendFaces 外发面（Assembly 注入；nil 时外发丢弃并记日志）。
	SendToWorld    func(typ uint32, payload []byte) error
	SendToMember   func(ident, typ uint32, payload []byte) error
	SendToAllMem   func(typ uint32, payload []byte) error
	SendToStorage  func(typ uint32, payload []byte) error // store#0（ident 4）
	SendToAllStore func(typ uint32, payload []byte) error

	// Logf 日志面。
	Logf func(format string, args ...any)

	mu       sync.Mutex
	worldQ   []msg
	storeQ   []msg
	memberQ  []msg

	// store 装载链状态（+0x8C 已装完空间计数；当前空间的数据名清单
	// 由 0x13 应答填充、0x1F/0x14 链式游标推进）。
	loadedSpaces int
	loadNames    []string // 当前装载空间的数据名（wchar）
	loadIndex    int      // 0x1F 链式游标

	// Closing 批量回存双游标（+0x90/+0x94：空间下标/数据下标）。
	closeSpace int
	closeData  int

	innerAddr string
	innerPort int

	lastTickNano atomic.Int64
}

// New 服务构造（官方缺省：SaveBatch=50）。
func New(districtID, serverID uint32) *Public {
	p := &Public{
		DistrictID: districtID,
		ServerID:   serverID,
		SaveBatch:  50,
		Spaces:     NewSpaces(),
		Timer:      NewPubTimer(),
		Logf:       func(format string, args ...any) {},
	}
	p.lastTickNano.Store(-1)
	return p
}

// OwnIdent 官方 SetCommIdent(1)。
func (p *Public) OwnIdent() uint32 { return PublicIdent }

// SetInner 官方 Adapter 地址（广播包 ip/port 源）。
func (p *Public) SetInner(addr string, port int) {
	p.innerAddr, p.innerPort = addr, port
}

// --- 队列面（CMsgDeque2 等价）---

// PutWorldMessage/PutStoreMessage/PutMemberMessage 三队列入口。
func (p *Public) PutWorldMessage(ident, typ uint32, payload []byte) {
	p.mu.Lock()
	p.worldQ = append(p.worldQ, msg{ident, typ, payload})
	p.mu.Unlock()
}

func (p *Public) PutStoreMessage(ident, typ uint32, payload []byte) {
	p.mu.Lock()
	p.storeQ = append(p.storeQ, msg{ident, typ, payload})
	p.mu.Unlock()
}

func (p *Public) PutMemberMessage(ident, typ uint32, payload []byte) {
	p.mu.Lock()
	p.memberQ = append(p.memberQ, msg{ident, typ, payload})
	p.mu.Unlock()
}

func (p *Public) swap(q *[]msg) []msg {
	p.mu.Lock()
	out := *q
	*q = nil
	p.mu.Unlock()
	return out
}

// DoWait 单帧（官方 CThread WorkerProc 循环体）：泵三队列 → 秒分支
// （状态机 tick + 广播）。
func (p *Public) DoWait() {
	for _, m := range p.swap(&p.worldQ) {
		p.ProcessWorldMsg(m.ident, m.typ, m.payload)
	}
	for _, m := range p.swap(&p.storeQ) {
		p.ProcessStoreMsg(m.ident, m.typ, m.payload)
	}
	for _, m := range p.swap(&p.memberQ) {
		p.ProcessMemberMsg(m.ident, m.typ, m.payload)
	}
	if el, ok := p.elapseMillisec(); ok {
		p.PublicStateMachine(smEventTick, el)
	}
}

// elapseMillisec 秒节拍（官方 CTickTimer.GetElapseMillisec(1000)>=1000）。
func (p *Public) elapseMillisec() (int, bool) {
	now := nanoNow()
	last := p.lastTickNano.Load()
	if last < 0 {
		p.lastTickNano.Store(now)
		return 0, false
	}
	if now-last < int64(timeSecondNano) {
		return 0, false
	}
	p.lastTickNano.Store(now)
	return int((now - last) / int64(timeMilliNano)), true
}

const (
	timeSecondNano = 1000_000_000
	timeMilliNano  = 1000_000
)

// --- 状态机（PublicStateMachine）---

// PublicStateMachine 事件分派（§5 迁移图）。
func (p *Public) PublicStateMachine(event int, elapsed int) {
	switch PublicState(p.State.Load()) {
	case StateUnknown:
		if event == smEventStoreConn {
			p.stateUnknownStoreConn()
		}
	case StateLoadSpaces:
		if event == smEventSpacesLoaded {
			p.stateLoadSpacesDone()
		}
	case StateOpened:
		switch event {
		case smEventTick:
			if p.Callee != nil {
				p.Callee.OnFrameExecute(p.kernel(), elapsed)
			}
			p.Timer.Run(elapsed)
		case smEventWorldClose:
			if p.Callee != nil {
				p.Callee.OnServerClose(p.kernel())
			}
			p.closeSpace, p.closeData = 0, 0
			p.setState(StateClosing)
		}
	case StateClosing:
		if event == smEventTick {
			p.stateClosingTick()
		}
	}
}

func (p *Public) setState(s PublicState) {
	old := PublicState(p.State.Swap(int32(s)))
	if old != s {
		p.Logf("public state %s -> %s", old, s)
	}
}

// stateUnknownStoreConn 官方 Unknown(event1)：对第一个空间发 0x1E。
func (p *Public) stateUnknownStoreConn() {
	list := p.Spaces.List()
	if len(list) == 0 {
		// 无空间也直接完成装载（官方不会发生——main.lua 至少建 2 个）
		p.stateLoadSpacesDone()
		return
	}
	if err := p.requestPubInfo(list[0]); err != nil {
		p.Logf("(GamePublic::PublicStateUnknown)send error: %v", err)
		return
	}
	p.loadedSpaces = 0
	p.setState(StateLoadSpaces)
}

// requestPubInfo 官方 0x1E 请求 {space[33], table[33]} 0x42B。
func (p *Public) requestPubInfo(s *PubSpace) error {
	if p.SendToStorage == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	buf := make([]byte, 0x42)
	putFixedString(buf[0:], s.Name)
	putFixedString(buf[nameLen:], s.TableName)
	return p.SendToStorage(StorePubInfoReq, buf)
}

// stateLoadSpacesDone 官方 event2：组空间清单广播全体 member → Opened。
func (p *Public) stateLoadSpacesDone() {
	payload, err := p.BuildSpacesList()
	if err != nil {
		p.Logf("(GamePublic::PublicStateLoadSpaces)%v", err)
		return
	}
	if p.SendToAllMem != nil {
		_ = p.SendToAllMem(PushSpaces, payload)
	}
	if p.Callee != nil {
		p.Callee.OnPublicStart(p.kernel())
	}
	p.setState(StateOpened)
}

// BuildSpacesList 空间清单（MemberRegister 应答同构）：
// {int 空间数; per {str space, str table, int dataCount}}。
func (p *Public) BuildSpacesList() ([]byte, error) {
	list := p.Spaces.List()
	arc := NewStoreArchive(0x100)
	if err := arc.WriteInt(int32(len(list))); err != nil {
		return nil, err
	}
	for _, s := range list {
		if err := arc.WriteString(s.Name); err != nil {
			return nil, err
		}
		if err := arc.WriteString(s.TableName); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(s.Count())); err != nil {
			return nil, err
		}
	}
	return arc.Bytes(), nil
}

// stateClosingTick 官方 Closing(event0)：分批回存（GetSave&&GetModify，
// 每帧至多 SaveBatch 条）；全空间遍历完 → SendToWorld(2,{1}) → Closed。
func (p *Public) stateClosingTick() {
	list := p.Spaces.List()
	sent := 0
	for p.closeSpace < len(list) {
		space := list[p.closeSpace]
		datas := space.All()
		for p.closeData < len(datas) {
			d := datas[p.closeData]
			p.closeData++
			if !d.Save || !d.Modify {
				continue
			}
			if err := p.InnerSavePubData(space, d); err != nil {
				p.Logf("(GamePublic::InnerSavePubData)%v", err)
				continue
			}
			sent++
			if sent >= p.saveBatchClamp() {
				return
			}
		}
		p.closeSpace++
		p.closeData = 0
	}
	// 回存毕：通知 world 关闭完成
	var done [4]byte
	binary.LittleEndian.PutUint32(done[:], 1)
	if p.SendToWorld != nil {
		_ = p.SendToWorld(WorldCloseDone, done[:])
	}
	p.setState(StateClosed)
}

func (p *Public) saveBatchClamp() int {
	if p.SaveBatch < 1 {
		return 1
	}
	if p.SaveBatch > 1000 {
		return 1000
	}
	return p.SaveBatch
}

// InnerSavePubData 官方 0x20 保存包 {space[33], table[33], wname[33](0x42),
// blob@0x84}（blob=saveOnly 快照）。
func (p *Public) InnerSavePubData(space *PubSpace, d *PubData) error {
	if p.SendToStorage == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	blob, err := d.StoreArchiveBlob(true)
	if err != nil {
		return err
	}
	buf := make([]byte, 0x84+len(blob))
	putFixedString(buf[0:], space.Name)
	putFixedString(buf[nameLen:], space.TableName)
	putFixedWideStr(buf[nameLen*2:], d.Name)
	copy(buf[0x84:], blob)
	return p.SendToStorage(StorePubSave, buf)
}

// --- world 面（ProcessWorldMsg：0 WorldServerState / 1 WorldClosePublic）---

// ProcessWorldMsg 处理 world→public 消息。
func (p *Public) ProcessWorldMsg(ident, typ uint32, payload []byte) {
	switch typ {
	case WorldServerState:
		if len(payload) != 0x24 {
			p.Logf("(GamePublic::WorldServerState)len == sizeof(public_res_server_state_t): %d", len(payload))
			return
		}
		if p.Callee != nil {
			st := ServerState{
				Name:  getFixedString(payload[0:16]),
				Props: [5]int32{},
			}
			for i := range st.Props {
				st.Props[i] = int32(binary.LittleEndian.Uint32(payload[0x10+i*4:]))
			}
			p.Callee.OnServerState(p.kernel(), st)
		}
	case WorldClosePublic:
		p.PublicStateMachine(smEventWorldClose, 0)
	default:
		p.Logf("(GamePublic::ProcessWorldMsg)unknown type %d", typ)
	}
}

// ServerState 是 public_res_server_state_t {char[16], 5×int}。
type ServerState struct {
	Name  string
	Props [5]int32
}

// --- store 面（ProcessStoreMsg 跳表：1/0x13/0x14/0x15/0x16/0x17）---

// ProcessStoreMsg 处理 store→public 消息（装载链 + ACK）。
func (p *Public) ProcessStoreMsg(ident, typ uint32, payload []byte) {
	switch typ {
	case StoreBackRoleOffline:
		p.onStoreRoleOffline(payload)
	case StoreBackPubInfo:
		p.onStorePubInfo(payload)
	case StoreBackPubLoad:
		p.onStorePubLoad(payload)
	case StoreBackPubSave, StoreBackPubAdd, StoreBackPubRemove:
		p.onStoreAck(typ, payload)
	default:
		p.Logf("(GamePublic::ProcessStoreMsg)unknown type %#x", typ)
	}
}

// OnStoreConnected 内网 store#0 注册回调（官方 OnConnected→PutMessage(0,1)）。
func (p *Public) OnStoreConnected() {
	p.PublicStateMachine(smEventStoreConn, 0)
}

// OnWorldConnected 内网 world 注册回调（官方空操作）。
func (p *Public) OnWorldConnected() {}

// onStorePubInfo 官方 StorePubInfo（0x13）：{result, space[33], table[33],
// count, wname[33]×count}——记录数据名清单并发出首条 0x1F。
func (p *Public) onStorePubInfo(payload []byte) {
	const base = 0x4A
	if len(payload) < base {
		p.Logf("(GamePublic::StorePubInfo)len >= sizeof(store_back_pub_info_t): %d", len(payload))
		return
	}
	result := binary.LittleEndian.Uint32(payload[0:])
	if result != 1 {
		p.Logf("(GamePublic::StorePubInfo)get pub info failed")
		return
	}
	spaceName := getFixedString(payload[4:])
	table := getFixedString(payload[4+nameLen:])
	count := int(binary.LittleEndian.Uint32(payload[4+nameLen*2:]))
	if count < 0 || len(payload) < base+count*wnameLen {
		p.Logf("(GamePublic::StorePubInfo)load data name failed")
		return
	}
	space, ok := p.Spaces.Find(spaceName)
	if !ok {
		p.Logf("(GamePublic::StorePubInfo)space not exists: %s", spaceName)
		return
	}
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = getFixedWideStr(payload[base+i*wnameLen:])
	}
	// 空间元数据核对：表名不符报错（官方 add data failed 路径）
	if space.TableName != table {
		p.Logf("(GamePublic::StorePubInfo)add data failed: table %s != %s", table, space.TableName)
		return
	}
	p.loadNames = names
	p.loadIndex = 0
	p.requestNextPubLoad(space)
}

// requestNextPubLoad 官方 0x1F 链式请求 {count, index, space[33],
// table[33], wname[33]} 0x8CB。
func (p *Public) requestNextPubLoad(space *PubSpace) error {
	if p.loadIndex >= len(p.loadNames) {
		return p.onPubSpaceLoaded(space)
	}
	if p.SendToStorage == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	buf := make([]byte, 0x8C)
	binary.LittleEndian.PutUint32(buf[0:], uint32(len(p.loadNames)))
	binary.LittleEndian.PutUint32(buf[4:], uint32(p.loadIndex))
	putFixedString(buf[8:], space.Name)
	putFixedString(buf[8+nameLen:], space.TableName)
	putFixedWideStr(buf[8+nameLen*2:], p.loadNames[p.loadIndex])
	return p.SendToStorage(StorePubLoadReq, buf)
}

// onStorePubLoad 官方 StorePubLoad（0x14）：{result, index@8, space@0xC,
// table@0x2D, wname@0x4E, uid@0x90, blob@0xB1}——装载并链式取下一条。
func (p *Public) onStorePubLoad(payload []byte) {
	const base = 0xB1
	if len(payload) < base {
		p.Logf("(GamePublic::StorePubLoad)len >= sizeof(store_back_pub_load_t): %d", len(payload))
		return
	}
	result := binary.LittleEndian.Uint32(payload[0:])
	if result != 1 {
		p.Logf("(GamePublic::StorePubLoad)load pub data failed")
		return
	}
	index := int(binary.LittleEndian.Uint32(payload[8:]))
	spaceName := getFixedString(payload[0xC:])
	wname := getFixedWideStr(payload[0x4E:])
	uid := getFixedString(payload[0x90:])
	space, ok := p.Spaces.Find(spaceName)
	if !ok {
		p.Logf("(GamePublic::StorePubLoad)space not exists: %s", spaceName)
		return
	}
	d, ok := space.Get(wname)
	if !ok {
		d, _ = space.Add(wname, "")
		if d == nil {
			p.Logf("(GamePublic::StorePubLoad)data not exists: %s", wname)
			return
		}
	}
	d.UID = uid
	if err := d.LoadArchiveBlob(payload[base:]); err != nil {
		p.Logf("(GamePublic::StorePubLoad)load data failed: %v", err)
		return
	}
	d.Modify = false
	if p.Callee != nil {
		p.Callee.OnPubDataLoaded(p.kernel(), spaceName, wname)
	}
	p.loadIndex = index + 1
	if err := p.requestNextPubLoad(space); err != nil {
		p.Logf("(GamePublic::StorePubLoad)send error: %v", err)
	}
}

// onPubSpaceLoaded 空间装载完：游标推进；全齐 → event2。
func (p *Public) onPubSpaceLoaded(space *PubSpace) error {
	if p.Callee != nil {
		p.Callee.OnPubSpaceLoaded(p.kernel(), space.Name)
	}
	p.loadedSpaces++
	list := p.Spaces.List()
	if p.loadedSpaces >= len(list) {
		p.PublicStateMachine(smEventSpacesLoaded, 0)
		return nil
	}
	// 下一空间 0x1E
	return p.requestPubInfo(list[p.loadedSpaces])
}

// onStoreAck 官方 0x15/0x16/0x17 ACK：{result, space[33], table[33],
// wname[33]} ==0x88B。
func (p *Public) onStoreAck(typ uint32, payload []byte) {
	if len(payload) != 0x88 {
		p.Logf("(GamePublic::StorePubAck)len == sizeof(store_back_pub_ack_t): %d", len(payload))
		return
	}
	result := binary.LittleEndian.Uint32(payload[0:])
	wname := getFixedWideStr(payload[0x48:])
	switch typ {
	case StoreBackPubSave:
		if result != 1 {
			p.Logf("(GamePublic::StorePubSave)save failed: %s", wname)
		}
	case StoreBackPubAdd:
		if result != 1 {
			p.Logf("(GamePublic::StorePubAdd)add failed: %s", wname)
		}
	case StoreBackPubRemove:
		if result != 1 {
			p.Logf("(GamePublic::StorePubRemove)remove failed: %s", wname)
		}
	}
}

// onStoreRoleOffline 官方 StoreRoleOffline（0x1）：{result, count,
// 行 (str,wstr,double,str)×} → Callee OnRoleOffline。
func (p *Public) onStoreRoleOffline(payload []byte) {
	if len(payload) < 8 {
		p.Logf("(GamePublic::StoreRoleOffline)len >= sizeof(store_back_role_offline_t): %d", len(payload))
		return
	}
	result := binary.LittleEndian.Uint32(payload[0:])
	if result != 1 {
		p.Logf("(GamePublic::StoreRoleOffline)load failed")
		return
	}
	count := int(binary.LittleEndian.Uint32(payload[4:]))
	a := NewLoadArchive(payload[8:])
	rows := make([][]Value, 0, count)
	for i := 0; i < count; i++ {
		s1, err := a.ReadString()
		if err != nil {
			p.Logf("(GamePublic::StoreRoleOffline)retrieve failed: %v", err)
			return
		}
		w1, err := a.ReadWideStr()
		if err != nil {
			p.Logf("(GamePublic::StoreRoleOffline)retrieve failed: %v", err)
			return
		}
		d1, err := a.ReadDouble()
		if err != nil {
			p.Logf("(GamePublic::StoreRoleOffline)retrieve failed: %v", err)
			return
		}
		s2, err := a.ReadString()
		if err != nil {
			p.Logf("(GamePublic::StoreRoleOffline)retrieve failed: %v", err)
			return
		}
		rows = append(rows, []Value{StringValue(s1), WideValue(w1), DoubleValue(d1), StringValue(s2)})
	}
	if p.Callee != nil {
		p.Callee.OnRoleOffline(p.kernel(), rows)
	}
}

// --- member 面（ProcessMemberMsg：0 Register / 1 Data / 2 Message）---

// ProcessMemberMsg 处理 member→public 消息。
func (p *Public) ProcessMemberMsg(ident, typ uint32, payload []byte) {
	switch typ {
	case MemberForwRegister:
		p.onMemberRegister(ident, payload)
	case MemberForwData:
		p.onMemberData(ident, payload)
	case MemberForwMessage:
		p.onMemberMessage(ident, payload)
	default:
		p.Logf("(GamePublic::ProcessMemberMsg)unknown type %d", typ)
	}
}

// onMemberRegister 官方 MemberRegister：len==8 断言；State==2 门；
// 回包空间清单（定向，type 0）。
func (p *Public) onMemberRegister(ident uint32, payload []byte) {
	if len(payload) != 8 {
		p.Logf("len == sizeof(public_forw_register_t): %d", len(payload))
		return
	}
	if PublicState(p.State.Load()) != StateOpened {
		return // 官方静默 return 0
	}
	list, err := p.BuildSpacesList()
	if err != nil {
		p.Logf("(GamePublic::MemberRegister)%v", err)
		return
	}
	if p.SendToMember != nil {
		if err := p.SendToMember(ident, PushSpaces, list); err != nil {
			p.Logf("(GamePublic::MemberRegister)send error: %v", err)
		}
	}
}

// onMemberData 官方 MemberData：{?, index@4, space[33]@8}；回包 0x90 头 +
// 全量快照（定向，type 1）。
func (p *Public) onMemberData(ident uint32, payload []byte) {
	const reqLen = 0x29
	if len(payload) != reqLen {
		p.Logf("len == sizeof(public_forw_data_t): %d", len(payload))
		return
	}
	index := int(binary.LittleEndian.Uint32(payload[4:]))
	spaceName := getFixedString(payload[8:])
	space, ok := p.Spaces.Find(spaceName)
	if !ok {
		p.Logf("(GamePublic::MemberData)space not exists: %s", spaceName)
		return
	}
	d, err := space.GetByIndex(index)
	if err != nil {
		p.Logf("(GamePublic::MemberData)%v", err)
		return
	}
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		p.Logf("(GamePublic::MemberData)store data failed: %v", err)
		return
	}
	buf := make([]byte, 0x90+len(blob))
	binary.LittleEndian.PutUint32(buf[0:], uint32(space.Count()))
	binary.LittleEndian.PutUint32(buf[4:], uint32(index))
	putFixedString(buf[8:], spaceName)
	putFixedWideStr(buf[0x29:], d.Name)
	putFixedString(buf[0x6B:], d.Key)
	binary.LittleEndian.PutUint32(buf[0x8C:], 1)
	copy(buf[0x90:], blob)
	if p.SendToMember != nil {
		if err := p.SendToMember(ident, PushData, buf); err != nil {
			p.Logf("(GamePublic::MemberData)send error: %v", err)
		}
	}
}

// onMemberMessage 官方 MemberMessage：{msg_id, argc, 标签值列} →
// Callee OnPublicMessage。
func (p *Public) onMemberMessage(ident uint32, payload []byte) {
	const base = 8
	if len(payload) < base {
		p.Logf("len >= sizeof(public_forw_message_t): %d", len(payload))
		return
	}
	msgID := binary.LittleEndian.Uint32(payload[0:])
	argc := int(binary.LittleEndian.Uint32(payload[4:]))
	a := NewLoadArchive(payload[base:])
	args, err := GetVaryMessage(a, argc)
	if err != nil {
		p.Logf("(GamePublic::MemberMessage)load message failed: %v", err)
		return
	}
	if p.Callee != nil {
		p.Callee.OnPublicMessage(p.kernel(), ident, msgID, args)
	}
}

// --- 逻辑层 API（CPubKernel 面的 Go 等价）---

// kernel 返回 Callee 用的内核门面（this 自身实现 Kernel 接口）。
func (p *Public) kernel() *Kernel { return &Kernel{p: p} }

// Kernel 是 IPubKernel 的 Go 门面（逻辑模块观测槽：+0x10 FindPubData/
// +0x20 AddTimer/+0x30 SendPublicMessage）。
type Kernel struct{ p *Public }

// FindPubData 按空间名+数据名查（官方 +0x10 槽）。
func (k *Kernel) FindPubData(spaceName, dataName string) (*PubData, bool) {
	s, ok := k.p.Spaces.Find(spaceName)
	if !ok {
		return nil, false
	}
	return s.Get(dataName)
}

// AddTimer 官方 AddTimer(cb, space, name, interval, data)（60s=0xEA60）。
func (k *Kernel) AddTimer(space, name string, interval int, cb TimerFunc, data []Value) error {
	return k.p.Timer.Add(space+"/"+name, interval, cb, data)
}

// RemoveTimer 官方 RemoveTimer。
func (k *Kernel) RemoveTimer(space, name string) {
	k.p.Timer.Remove(space + "/" + name)
}

// SendPublicMessage 官方 +0x30 槽：组 {msg_id, argc, 值列} 经
// SendToAllMember 广播（op 0xD；State==2 门=SyncPublic）。
func (k *Kernel) SendPublicMessage(spaceName, dataName string, msgID uint32, args []Value) error {
	p := k.p
	if PublicState(p.State.Load()) != StateOpened {
		return fmt.Errorf("(GamePublic::SyncPublic)state not opened")
	}
	if p.SendToAllMem == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	arc := NewStoreArchive(0x40)
	if err := arc.WriteInt(int32(msgID)); err != nil {
		return err
	}
	if err := arc.WriteInt(int32(len(args))); err != nil {
		return err
	}
	if err := PutVaryMessage(arc, args); err != nil {
		return err
	}
	return p.SendToAllMem(PushMessage, arc.Bytes())
}

// RequestLoadPlayer 官方 LoadPlayer：{account[33], wname[33], fromIdent}
// 0x67B → world type 0。
func (k *Kernel) RequestLoadPlayer(account, roleName string, fromIdent uint32) error {
	if k.p.SendToWorld == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	buf := make([]byte, 0x67)
	putFixedString(buf[0:], account)
	putFixedWideStr(buf[nameLen:nameLen+wnameLen], roleName)
	binary.LittleEndian.PutUint32(buf[nameLen+wnameLen:], fromIdent)
	return k.p.SendToWorld(WorldLoadPlayer, buf)
}

// RequestServerState 官方：1B → world type 1。
func (k *Kernel) RequestServerState() error {
	if k.p.SendToWorld == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	return k.p.SendToWorld(WorldRequestState, []byte{0})
}

// RequestRoleOffline 官方：{ServerId, roleID, Y,M,D,h,m,s, flag} 0x24B →
// 全 store type 1。
func (k *Kernel) RequestRoleOffline(roleID int64, ts [7]int32, flag int32) error {
	if k.p.SendToAllStore == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	buf := make([]byte, 0x24)
	binary.LittleEndian.PutUint32(buf[0:], k.p.ServerID)
	binary.LittleEndian.PutUint64(buf[4:], uint64(roleID))
	for i, v := range ts {
		binary.LittleEndian.PutUint32(buf[0xC+i*4:], uint32(v))
	}
	binary.LittleEndian.PutUint32(buf[0x20:], uint32(flag))
	return k.p.SendToAllStore(StoreRoleOfflineReq, buf)
}

// SaveLog 官方 type 0xD：{wtitle[33], wname[0x100], wtext[0x100]} 0x44AB
//（禁 `'`）。
func (k *Kernel) SaveLog(title, name, text string) error {
	if strings.ContainsAny(title+name+text, "'") {
		return fmt.Errorf("(CPubKernel::SaveLog)log content error")
	}
	if k.p.SendToStorage == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	buf := make([]byte, 0x44A)
	putFixedWideStr(buf[0:], title)
	putFixedWideStr(buf[wnameLen:], name)
	putFixedWideStr(buf[wnameLen+0x200:], text)
	return k.p.SendToStorage(StoreSaveLog, buf)
}

// CustomLog 官方 type 0x11：{file[0x20], u32, varlist blob}。
func (k *Kernel) CustomLog(file string, args []Value) error {
	if k.p.SendToStorage == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	arc := NewStoreArchive(0x40)
	if err := PutVaryMessage(arc, args); err != nil {
		return fmt.Errorf("(CPubKernel::CustomLog)put fields error: %v", err)
	}
	buf := make([]byte, 0x24, 0x24+len(arc.Bytes()))
	putFixedString(buf[0:0x20], file)
	buf = append(buf, arc.Bytes()...)
	return k.p.SendToStorage(StoreCustomLog, buf)
}

// --- 数据变更推送（CPubData::Send\* 官方语义：权威侧变更 → 广播镜像）---

// SyncPush 广播一条推送 op（State==2 门=SyncPublic 语义）。
func (p *Public) SyncPush(op uint32, payload []byte) error {
	if PublicState(p.State.Load()) != StateOpened {
		return fmt.Errorf("(GamePublic::SyncPublic)state not opened")
	}
	if p.SendToAllMem == nil {
		return fmt.Errorf("(CInterConnect::SendMsg)not connected")
	}
	return p.SendToAllMem(op, payload)
}

// PushAddPubData 官方 CPubSpace::AddPubData：广播 op2（公共头+blob
// flag=0）+ store op 0x21 持久化（0xA5 头含 uid）。
func (p *Public) PushAddPubData(space *PubSpace, d *PubData) {
	hdr := BuildPushHeader(space.Name, d.Name, d.Key)
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		p.Logf("(CPubSpace::AddPubData)%v", err)
		return
	}
	_ = p.SyncPush(PushAdd, append(hdr, blob...))
	// store 0x21
	if p.SendToStorage != nil {
		sb, err := d.StoreArchiveBlob(true)
		if err != nil {
			p.Logf("(CPubSpace::AddPubData)%v", err)
			return
		}
		buf := make([]byte, 0xA5, 0xA5+len(sb))
		putFixedString(buf[0:], space.Name)
		putFixedString(buf[nameLen:], space.TableName)
		putFixedWideStr(buf[nameLen*2:], d.Name)
		putFixedString(buf[0x84:], d.UID)
		buf = append(buf, sb...)
		_ = p.SendToStorage(StorePubAdd, buf)
	}
}

// PushRemovePubData 官方 op3（头截 0x63）+ store 0x22。
func (p *Public) PushRemovePubData(space *PubSpace, d *PubData) {
	hdr := BuildPushHeader(space.Name, d.Name, "")
	_ = p.SyncPush(PushRemove, hdr[:nameLen+wnameLen])
	if p.SendToStorage != nil {
		buf := make([]byte, 0x84)
		putFixedString(buf[0:], space.Name)
		putFixedString(buf[nameLen:], space.TableName)
		putFixedWideStr(buf[nameLen*2:], d.Name)
		_ = p.SendToStorage(StorePubRemoveReq, buf)
	}
}

// PushAddAttr 官方 op4（AddAttr/SetAttr* 复用）：头+type@0x84+单值。
func (p *Public) PushAddAttr(spaceName, dataName, key string, v Value) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	arc := NewStoreArchive(0x20)
	_ = arc.WriteInt(int32(v.Type))
	_ = v.PutArchive(arc)
	_ = p.SyncPush(PushAddAttr, append(hdr, arc.Bytes()...))
}

// PushRemoveAttr 官方 op5。
func (p *Public) PushRemoveAttr(spaceName, dataName, key string) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	_ = p.SyncPush(PushRemoveAttr, hdr)
}

// PushAddRec 官方 op6：头+cols@0x84+列类型列。
func (p *Public) PushAddRec(spaceName, dataName, key, recName string, rc *Record) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	arc := NewStoreArchive(0x20)
	_ = arc.WriteInt(int32(rc.Cols))
	for _, t := range rc.ColType {
		_ = arc.WriteInt(int32(t))
	}
	_ = p.SyncPush(PushAddRec, append(hdr, arc.Bytes()...))
}

// PushRemoveRec 官方 op7。
func (p *Public) PushRemoveRec(spaceName, dataName, key string) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	_ = p.SyncPush(PushRemoveRec, hdr)
}

// PushAddRecRow 官方 op8：头+row@0x84+全列值。
func (p *Public) PushAddRecRow(spaceName, dataName, key string, row int, vals []Value) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	arc := NewStoreArchive(0x40)
	_ = arc.WriteInt(int32(row))
	for _, v := range vals {
		_ = v.PutArchive(arc)
	}
	_ = p.SyncPush(PushRecAddRow, append(hdr, arc.Bytes()...))
}

// PushRemoveRecRow 官方 op9：头+row@0x84。
func (p *Public) PushRemoveRecRow(spaceName, dataName, key string, row int) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(row))
	_ = p.SyncPush(PushRecRemoveRow, append(hdr, b[:]...))
}

// PushClearRecRow 官方 opA。
func (p *Public) PushClearRecRow(spaceName, dataName, key string) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	_ = p.SyncPush(PushRecClear, hdr)
}

// PushSetRecGrid 官方 opB：头+row@0x84+col@0x88+coltype@0x8C+单值。
func (p *Public) PushSetRecGrid(spaceName, dataName, key string, row, col int, v Value) {
	hdr := BuildPushHeader(spaceName, dataName, key)
	var b [12]byte
	binary.LittleEndian.PutUint32(b[0:], uint32(row))
	binary.LittleEndian.PutUint32(b[4:], uint32(col))
	b[8] = byte(v.Type)
	arc := NewStoreArchive(0x10)
	_ = v.PutArchive(arc)
	_ = p.SyncPush(PushRecChange, append(append(hdr, b[:]...), arc.Bytes()...))
}

// PushSetRecRow 官方 opC：头+row@0x84+全列值。
func (p *Public) PushSetRecRow(spaceName, dataName, key string, row int, vals []Value) {
	p.PushAddRecRow(spaceName, dataName, key, row, vals) // 同构线格式
}

// --- 广播包（BroadcastServerInfo）---

// BuildBroadcast 官方 0x24B UDP 包：kind=4/magic=0x300/ip[16]/port/
// DistrictId@0x1C/ServerId@0x20。
func (p *Public) BuildBroadcast(ip string, port int) []byte {
	buf := make([]byte, broadcastSize)
	binary.LittleEndian.PutUint32(buf[0:], broadcastKind)
	binary.LittleEndian.PutUint32(buf[4:], broadcastMagic)
	putFixedString(buf[8:], ip) // 官方 CopyString(ip16, 0x10, addr)——ASCII 串
	binary.LittleEndian.PutUint32(buf[0x18:], uint32(port))
	binary.LittleEndian.PutUint32(buf[0x1C:], p.DistrictID)
	binary.LittleEndian.PutUint32(buf[0x20:], p.ServerID)
	return buf
}

// String 状态摘要（echo state 命令）。
func (p *Public) String() string {
	return fmt.Sprintf("public server %d state %s spaces %d loaded %d",
		p.ServerID, PublicState(p.State.Load()), p.Spaces.Count(), p.loadedSpaces)
}
