package fssolopublic

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// processStoreMsg 官方 ProcessStoreMsg（@0x100C56E0 跳表，9 处理器；
// solo type 空间 = 集群 +6，另含关系/动态/寄售三型）。
func (s *Server) processStoreMsg(typ uint32, payload []byte) {
	switch typ {
	case storeRoleOffline:
		s.storeRoleOffline(payload)
	case storePubInfoBack:
		s.storePubInfo(payload)
	case storePubLoadBack:
		s.storePubLoad(payload)
	case storePubSaveBack, storePubAddBack, storePubRemoveBack:
		s.storePubAck(typ, payload)
	case storeRelationBack:
		s.storeRelationInfo(payload)
	case storeFeedBack:
		s.storeFeedInfo(payload)
	case storeTradeBack:
		s.storeTradeInfo(payload)
	default:
		// 官方跳表外静默（trace 面）
		s.logf("(SoloPublic::ProcessStoreMsg)unknown type %d", typ)
	}
}

// storeBackHeader 是 store 应答公共头 {result@0 … count@0xC, archive@0x10}
// （0x22/0x23/0x28 同构，定长前缀 0x10）。
type storeBackHeader struct {
	Result int32
	F1     uint32
	F2     uint32
	Count  uint32
}

func parseStoreBackHeader(p []byte) (storeBackHeader, *fspublic.LoadArchive, error) {
	if len(p) < 0x10 {
		return storeBackHeader{}, nil, fmt.Errorf("len >= sizeof(store_back_header_t)=0x10，得 %d", len(p))
	}
	h := storeBackHeader{
		Result: i32at(p, 0), F1: u32at(p, 4), F2: u32at(p, 8), Count: u32at(p, 0xC),
	}
	return h, fspublic.NewLoadArchive(p[0x10:]), nil
}

// storeRelationInfo 官方 0x22：关系表装载 → CRelation::AddRelation×count
// → +0x174。
func (s *Server) storeRelationInfo(p []byte) {
	h, arc, err := parseStoreBackHeader(p)
	if err != nil {
		s.logf("(SoloPublic::StoreRelationInfo)len >= sizeof(store_back_relation_info_t)：%v", err)
		return
	}
	if h.Result == 0 {
		s.logf("(SoloPublic::StoreRelationInfo)get relation info failed")
		return
	}
	for i := uint32(0); i < h.Count; i++ {
		a, err1 := arc.ReadString()
		b, err2 := arc.ReadString()
		typ, err3 := arc.ReadInt()
		live, err4 := arc.ReadDouble()
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			s.logf("(SoloPublic::StoreRelationInfo)load relation failed: %v", err1)
			return
		}
		if !s.Relation.AddRelation(a, b, typ, live) {
			s.logf("(SoloPublic::StoreRelationInfo)add relation failed: %s %s", a, b)
		}
	}
	s.relations.Store(true)
}

// storeFeedInfo 官方 0x23：动态流装载 → CRelation::LoadFeed×count
// （ReadString×3+ReadInt×3+ReadWideStr+ReadInt×2+ReadString+ReadDouble）
// → +0x175。
func (s *Server) storeFeedInfo(p []byte) {
	h, arc, err := parseStoreBackHeader(p)
	if err != nil {
		s.logf("(SoloPublic::StoreFeedInfo)len >= sizeof(store_back_feed_info_t)：%v", err)
		return
	}
	if h.Result == 0 {
		s.logf("(SoloPublic::StoreFeedInfo)get feed info failed")
		return
	}
	for i := uint32(0); i < h.Count; i++ {
		var err error
		strs := make([]string, 0, 5)
		ints := make([]int32, 0, 5)
		var wstr string
		var dbl float64
		for k := 0; k < 3; k++ {
			var v string
			v, err = arc.ReadString()
			if err != nil {
				break
			}
			strs = append(strs, v)
		}
		if err == nil {
			for k := 0; k < 3; k++ {
				var v int32
				v, err = arc.ReadInt()
				if err != nil {
					break
				}
				ints = append(ints, v)
			}
		}
		if err == nil {
			wstr, err = arc.ReadWideStr()
		}
		if err == nil {
			for k := 0; k < 2; k++ {
				var v int32
				v, err = arc.ReadInt()
				if err != nil {
					break
				}
				ints = append(ints, v)
			}
		}
		var desc string
		if err == nil {
			desc, err = arc.ReadString()
		}
		if err == nil {
			dbl, err = arc.ReadDouble()
		}
		if err != nil {
			s.logf("(SoloPublic::StoreFeedInfo)load feed failed: %v", err)
			return
		}
		var i0, i1, i2, i3, i4 int32
		i0, i1, i2 = getInt(ints, 0), getInt(ints, 1), getInt(ints, 2)
		i3, i4 = getInt(ints, 3), getInt(ints, 4)
		s.Relation.LoadFeed(strs[0], strs[1], strs[2], i0, i1, i2, wstr, i3, i4, desc, dbl)
	}
	s.feeds.Store(true)
}

func getInt(ints []int32, i int) int32 {
	if i < len(ints) {
		return ints[i]
	}
	return 0
}

// storeTradeInfo 官方 0x28：寄售行装载 → CTrade::Load×count（20 字段
// 序：str,wstr,str,wstr,str,int,int,wstr×4,int×4,double×3）→ +0x177。
func (s *Server) storeTradeInfo(p []byte) {
	h, arc, err := parseStoreBackHeader(p)
	if err != nil {
		s.logf("(SoloPublic::StoreTradeInfo)len >= sizeof(store_back_trade_info_t)：%v", err)
		return
	}
	if h.Result == 0 {
		s.logf("(SoloPublic::StoreTradeInfo)load trade failed")
		return
	}
	for i := uint32(0); i < h.Count; i++ {
		var strs [3]string
		var wstrs [2]string
		var ints [2]int32
		var ws [5]string
		var ins [4]int32
		var ds [3]float64
		readErr := error(nil)
		strAt := func(i int) { strs[i], readErr = arc.ReadString() }
		wAt := func(i int) { wstrs[i], readErr = arc.ReadWideStr() }
		intAt := func(i int) { ints[i], readErr = arc.ReadInt() }
		// 官方 19 字段序：str,wstr,str,wstr,str,int,int,wstr×5,int×4,double×3
		for k := 0; k < 3 && readErr == nil; k++ {
			strAt(k)
			if k < 2 {
				wAt(k)
			}
		}
		for k := 0; k < 2 && readErr == nil; k++ {
			intAt(k)
		}
		for k := 0; k < 5 && readErr == nil; k++ {
			ws[k], readErr = arc.ReadWideStr()
		}
		for k := 0; k < 4 && readErr == nil; k++ {
			ins[k], readErr = arc.ReadInt()
		}
		for k := 0; k < 3 && readErr == nil; k++ {
			ds[k], readErr = arc.ReadDouble()
		}
		if readErr != nil {
			s.logf("(SoloPublic::StoreTradeInfo)load trade failed: %v", readErr)
			return
		}
		s.Trade.Load(strs[0], strs[1], strs[2], wstrs[0], wstrs[1],
			ints[0], ints[1], ws[0], ws[1], ws[2], ins[0], ins[1], ds[0], ds[1], ds[2])
	}
	s.trades.Store(true)
}

// storePubInfo 官方 0x19：{space[0x24],table[0x24],result@0x48,
// count@0x4C,archive{wstr}×} → space->Add(name,saveOnly) + 首条 0x25
// 请求 / 空间完 → 下一空间 0x24 / 全完 → 事件2。
func (s *Server) storePubInfo(p []byte) {
	if len(p) < 0x50 {
		s.logf("(SoloPublic::StorePubInfo)len >= sizeof(store_back_pub_info_t)=0x50，得 %d", len(p))
		return
	}
	space := cstr(p[0:0x24])
	if i32at(p, 0x48) == 0 {
		s.logf("(SoloPublic::StorePubInfo)get pub info failed: %s", space)
		return
	}
	sp := s.FindPubSpace(space)
	if sp == nil {
		s.logf("(SoloPublic::StorePubInfo)space not exists: %s", space)
		return
	}
	arc := fspublic.NewLoadArchive(p[0x50:])
	count := u32at(p, 0x4C)
	for i := uint32(0); i < count; i++ {
		name, err := arc.ReadWideStr()
		if err != nil {
			s.logf("(SoloPublic::StorePubInfo)load data name failed: %v", err)
			return
		}
		if _, err := sp.AddPubData(name, "", ""); err != nil {
			s.logf("(SoloPublic::StorePubInfo)add data failed: %s", name)
		}
	}
	s.mu.Lock()
	cursor := s.loadSpaceCursor
	s.mu.Unlock()
	if sp.Count() > 0 {
		s.requestPubLoad(sp, sp.All()[0].Name, 0)
		return
	}
	s.onPubSpaceLoaded(sp, cursor)
}

// requestPubLoad 官方 0x25 请求 {space[0x24],table[0x24],wname[0x42],
// index}（0x8C 头，链式拉取）。
func (s *Server) requestPubLoad(sp *fspublic.PubSpace, wname string, index uint32) {
	out := make([]byte, 0x8C)
	putFixedStr(out[0:0x24], sp.Name)
	putFixedStr(out[0x24:0x48], sp.TableName)
	putFixedWide(out[0x48:0x8A], wname)
	binary.LittleEndian.PutUint32(out[0x88:], index)
	if !s.inner.SendToStore(0, 0, storePubLoadReq, out) {
		s.logf("(SoloPublic::StorePubInfo)send pub load failed")
	}
}

// storePubLoad 官方 0x1A（前缀 0xC0）：SetUid + LoadArchive(blob) +
// SetModify(0) → 下一条 0x25 / 空间完 → onPubSpaceLoaded。
func (s *Server) storePubLoad(p []byte) {
	if len(p) < sizeStoreLoadHdr {
		s.logf("(SoloPublic::StorePubLoad)len >= sizeof(store_back_pub_load_t)=0xC0，得 %d", len(p))
		return
	}
	if i32at(p, 0) == 0 {
		s.logf("(SoloPublic::StorePubLoad)load pub data failed")
		return
	}
	space := cstr(p[8 : 8+0x24])
	sp := s.FindPubSpace(space)
	if sp == nil {
		s.logf("(SoloPublic::StorePubLoad)space not exists: %s", space)
		return
	}
	wname := fixedWide(p[8+0x24+0x24 : 8+0x24+0x24+0x42])
	uid := cstr(p[0x9C : 0x9C+0x24])
	d, ok := sp.Get(wname)
	if !ok {
		s.logf("(SoloPublic::StorePubLoad)data not exists: %s", wname)
		return
	}
	d.UID = uid
	if err := d.LoadArchiveBlob(p[sizeStoreLoadHdr:]); err != nil {
		s.logf("(SoloPublic::StorePubLoad)load data failed: %v", err)
		return
	}
	d.Modify = false
	// 链式取下一条
	idx := sp.Find(wname)
	if idx+1 < sp.Count() {
		s.requestPubLoad(sp, sp.All()[idx+1].Name, uint32(idx+1))
		return
	}
	s.mu.Lock()
	cursor := s.loadSpaceCursor
	s.mu.Unlock()
	s.onPubSpaceLoaded(sp, cursor)
}

// onPubSpaceLoaded 官方 OnPubSpaceLoaded：下一空间 0x24 / 全完 →
// StateMachine(2)。
func (s *Server) onPubSpaceLoaded(sp *fspublic.PubSpace, cursor int) {
	s.callee.OnPubSpaceLoaded(nil, sp.Name)
	s.mu.Lock()
	next := cursor + 1
	total := len(s.spaces)
	var done bool
	if next < total {
		s.loadSpaceCursor = next
	} else {
		done = true
	}
	s.mu.Unlock()
	if done {
		s.stateMachine(evSpacesDone, 0)
		return
	}
	if !s.requestPubInfo(next) {
		s.logf("(SoloPublic::StorePubInfo)send pub load failed")
	}
}

// storePubAck 官方 0x1B/0x1C/0x1D：{result, space[0x24], table[0x24],
// wname[0x42]} ==0x88（失败仅 trace）。
func (s *Server) storePubAck(typ uint32, p []byte) {
	name := map[uint32]string{
		storePubSaveBack: "StorePubSave", storePubAddBack: "StorePubAdd", storePubRemoveBack: "StorePubRemove",
	}[typ]
	if len(p) != 0x88 {
		s.logf("(%s)len == sizeof(store_back_pub_%s_t)=0x88，得 %d", name, name, len(p))
		return
	}
	if i32at(p, 0) == 0 {
		s.logf("(%s)save failed: %s", name, cstr(p[4:4+0x24]))
	}
}

// storeRoleOffline 官方 0x01：{result@0, count@4, 行(str,wstr,double,str)×@8}
// → Callee OnRoleOffline（集群同源）。
func (s *Server) storeRoleOffline(p []byte) {
	if len(p) < 8 {
		s.logf("(SoloPublic::StoreRoleOffline)len >= sizeof(store_back_role_offline_t): %d", len(p))
		return
	}
	if u32at(p, 0) != 1 {
		s.logf("(SoloPublic::StoreRoleOffline)load failed")
		return
	}
	count := int(u32at(p, 4))
	arc := fspublic.NewLoadArchive(p[8:])
	rows := make([][]fspublic.Value, 0, count)
	for i := 0; i < count; i++ {
		s1, err1 := arc.ReadString()
		w1, err2 := arc.ReadWideStr()
		d1, err3 := arc.ReadDouble()
		s2, err4 := arc.ReadString()
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			s.logf("(SoloPublic::StoreRoleOffline)retrieve failed: %v", err1)
			return
		}
		rows = append(rows, []fspublic.Value{
			fspublic.StringValue(s1), fspublic.WideValue(w1),
			fspublic.DoubleValue(d1), fspublic.StringValue(s2),
		})
	}
	s.callee.OnRoleOffline(nil, rows)
}

// RequestRoleOffline 官方 CPubKernel::RequestRoleOffline：广播全体
// store type1 {ServerId, roleID, Y,M,D,h,m,s, flag}。
func (s *Server) RequestRoleOffline(roleID uint64, flag bool) {
	out := make([]byte, 0x24)
	binary.LittleEndian.PutUint32(out[0:], s.cfg.ServerID)
	binary.LittleEndian.PutUint64(out[4:], roleID)
	now := time.Now()
	binary.LittleEndian.PutUint16(out[12:], uint16(now.Year()))
	out[14] = uint8(now.Month())
	out[15] = uint8(now.Day())
	out[16] = uint8(now.Hour())
	out[17] = uint8(now.Minute())
	out[18] = uint8(now.Second())
	if flag {
		out[19] = 1
	}
	s.inner.SendToAllStore(0, 1, out)
}

// RequestServerState 官方 CPubKernel::RequestServerState（→world 1B）。
func (s *Server) RequestServerState() {
	s.inner.SendToWorld(0, worldServerStateReq, []byte{0})
}

// LoadPlayer 官方 SoloPublic::LoadPlayer（→world type0 {account[0x21],
// wname[0x21], fromIdent} 0x67B）。
func (s *Server) LoadPlayer(account, wname string, fromIdent uint32) bool {
	out := make([]byte, 0x67)
	putFixedStr(out[0:0x21], account)
	putFixedWide(out[0x21:0x63], wname)
	binary.LittleEndian.PutUint32(out[0x63:], fromIdent)
	return s.inner.SendToWorld(0, worldLoadPlayerReq, out)
}
