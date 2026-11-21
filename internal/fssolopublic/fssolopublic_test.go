package fssolopublic

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

// ---- 线格式 ----

func TestInnerFrameRoundTrip(t *testing.T) {
	frame, err := encodeInner(CommSelf, 0, 0x26, []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) != 16+3 {
		t.Fatalf("frame len = %d", len(frame))
	}
	if n := binary.LittleEndian.Uint32(frame); n != 15 {
		t.Fatalf("N = %d, want 15", n)
	}
	var got []payloadMsg
	acc := &frameAcc{}
	if err := acc.feed(frame[:7], func(c, i1, i2 uint32, p []byte) {
		got = append(got, payloadMsg{c, i1, i2, p})
	}); err != nil {
		t.Fatal(err)
	}
	if err := acc.feed(frame[7:], func(c, i1, i2 uint32, p []byte) {
		got = append(got, payloadMsg{c, i1, i2, p})
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].comm != CommSelf || got[0].id2 != 0x26 || len(got[0].payload) != 3 {
		t.Fatalf("got %+v", got)
	}
}

type payloadMsg struct {
	comm, id1, id2 uint32
	payload        []byte
}

func TestInnerFrameTooSmall(t *testing.T) {
	acc := &frameAcc{}
	err := acc.feed([]byte{4, 0, 0, 0}, func(uint32, uint32, uint32, []byte) {})
	if err == nil {
		t.Fatal("N<12 应报 message size error")
	}
}

func TestAnnounceCodec(t *testing.T) {
	pkt := encodeAnnounce("127.0.0.1", 16003, 1, 11)
	if len(pkt) != 0x24 {
		t.Fatalf("len = %#x, want 0x24", len(pkt))
	}
	if u32at(pkt, 0) != announcePublic || u32at(pkt, 4) != 0x3001 {
		t.Fatalf("kind/magic = %#x/%#x", u32at(pkt, 0), u32at(pkt, 4))
	}
	a, ok := decodeAnnounce(pkt)
	if !ok || a.Type != announcePublic || a.Addr != "127.0.0.1" || a.Port != 16003 || a.District != 1 || a.Server != 11 {
		t.Fatalf("decode = %+v ok=%v", a, ok)
	}
	// store 自宣（≥0x28，[0x24]=comm 5）
	store := make([]byte, 0x28)
	copy(store, pkt)
	binary.LittleEndian.PutUint32(store[0:], announceStore)
	binary.LittleEndian.PutUint32(store[0x24:], 5)
	a, ok = decodeAnnounce(store)
	if !ok || a.Type != announceStore || a.Index != 5 {
		t.Fatalf("store decode = %+v ok=%v", a, ok)
	}
	// magic 不符忽略
	bad := append([]byte(nil), store...)
	binary.LittleEndian.PutUint32(bad[4:], 0x300)
	if _, ok := decodeAnnounce(bad); ok {
		t.Fatal("magic 0x300 应忽略（solo 方言 0x3001）")
	}
	// world type0 RoleProtectBase(double@0x7C)
	wex := make([]byte, 0x84)
	copy(wex, pkt)
	binary.LittleEndian.PutUint32(wex[0:], announceWorldEx)
	binary.LittleEndian.PutUint64(wex[0x7C:], 0x4059000000000000) // 100.0
	if v, ok := worldRoleProtectBase(wex); !ok || v != 100.0 {
		t.Fatalf("roleProtectBase = %v ok=%v", v, ok)
	}
}

func TestFixedWideHelpers(t *testing.T) {
	buf := make([]byte, 0x10)
	putFixedWide(buf, "角色")
	if got := fixedWide(buf); got != "角色" {
		t.Fatalf("wide round-trip = %q", got)
	}
	long := make([]byte, 4)
	putFixedWide(long, "abcdef") // 超长截断保终结
	if got := fixedWide(long); got != "a" {
		t.Fatalf("truncated = %q", got)
	}
}

// ---- 配置 ----

func TestParseGamePublicINI(t *testing.T) {
	cfg, err := ParseGamePublicINI(defaultGamePublicINI)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MainEntity != "SoloPublic" {
		t.Fatalf("MainEntity = %q", cfg.MainEntity)
	}
	if cfg.DistrictID != 1 || cfg.ServerID != 11 {
		t.Fatalf("district/server = %d/%d", cfg.DistrictID, cfg.ServerID)
	}
	if cfg.SaveMinutes != 20 {
		t.Fatalf("SaveMinutes = %d", cfg.SaveMinutes)
	}
	if cfg.BroadPort != 15000 || cfg.EchoPort != 15003 {
		t.Fatalf("broad/echo port = %d/%d", cfg.BroadPort, cfg.EchoPort)
	}
	if len(cfg.Modules) != 3 || cfg.Modules[2] != "fssolopublic.dll" {
		t.Fatalf("modules = %v", cfg.Modules)
	}
}

// ---- 注册表/关系/寄售 模型 ----

func TestNameUidBidirectional(t *testing.T) {
	n := NewNameUid()
	if !n.Add("张三", "uid-1") || !n.Add("李四", "uid-2") {
		t.Fatal("add")
	}
	if uid, _ := n.GetUid("张三"); uid != "uid-1" {
		t.Fatalf("GetUid = %q", uid)
	}
	if name, _ := n.GetName("uid-2"); name != "李四" {
		t.Fatalf("GetName = %q", name)
	}
	if !n.Remove("张三", "") || n.GetCount() != 1 {
		t.Fatal("remove")
	}
	if !n.GetDeleted("张三") {
		t.Fatal("deleted 墓碑")
	}
}

func TestRelationFeedFlow(t *testing.T) {
	r := NewRelation()
	r.SetFeedLevels(2)
	r.SetFeedLiveTime(0, 60)
	r.SetFeedLiveTime(1, 3600)
	if r.GetFeedLevels() != 2 || r.GetFeedLiveTime(1) != 3600 {
		t.Fatal("feed config")
	}
	if !r.AddRelation("a", "b", 1, 60) || !r.FindRelation("a", "b") {
		t.Fatal("relation")
	}
	id := r.NewFeed("a", "b", 1, 0, int32(time.Now().Unix()), "hello", "k1")
	f, ok := r.FindFeed(id)
	if !ok || f.Text != "hello" || f.Owner != "a" {
		t.Fatalf("feed = %+v", f)
	}
	r.AddComment(id, FeedComment{Owner: "c", Text: "hi"})
	r.IncFeedGood(id)
	if g, b, _ := r.GetFeedGoodBad(id); g != 1 || b != 0 {
		t.Fatalf("good/bad = %d/%d", g, b)
	}
	list := r.SyncRecentFeeds("b", 0, 10)
	if len(list) != 1 || len(list[0].Comments) != 1 {
		t.Fatalf("recent = %+v", list)
	}
	// 装载路径（0x23 语义）
	if !r.LoadFeed("store-feed-1", "a", "b", 2, 1, 100, "from store", 3, 4, "desc", 60) {
		t.Fatal("LoadFeed")
	}
	if r.FeedCount() != 2 {
		t.Fatalf("feeds = %d", r.FeedCount())
	}
}

func TestTradeLifecycle(t *testing.T) {
	tr := NewTrade()
	id := tr.Sell("seller1", "卖家甲", "item_key_9", "屠龙刀", 1, 5000, "银两", "九阶神兵", 3600)
	if tr.GetStatus(id) != 1 {
		t.Fatal("status")
	}
	list := tr.Search("屠龙刀", "")
	if len(list) != 1 || list[0].Price != 5000 {
		t.Fatalf("search = %+v", list)
	}
	ok, price, seller := tr.Buy("buyer", "买家乙", id)
	if !ok || price != 5000 || seller != "seller1" || tr.Count() != 0 {
		t.Fatalf("buy = %v/%d/%s", ok, price, seller)
	}
	// 超时
	tr.Sell("seller1", "卖家甲", "k2", "倚天剑", 1, 300, "银两", "", -1) // 无超时（永不）
	tr.Load("id-old", "seller9", "卖家丙", "k3", "玄铁重剑", 1, 99, "银两", "", "", 1, 0, -1, 0, 0)
	if tr.Count() != 2 {
		t.Fatalf("count = %d", tr.Count())
	}
	if dead := tr.CheckTimeout(time.Now().Add(-time.Second)); len(dead) != 0 {
		t.Fatalf("无超时单被误判：%d", len(dead))
	}
}

// ---- 白盒：store/world/member 处理器 ----

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg, err := ParseGamePublicINI(defaultGamePublicINI)
	if err != nil {
		t.Fatal(err)
	}
	// echo 动态口（0=系统分配）：测试与在跑服务（合一服 echo 15003）
	// 常驻冲突，端口写死会互相挤兑。
	cfg.EchoPort = 0
	return New(cfg, Options{})
}

func storeBack(result int32, count uint32, body []byte) []byte {
	out := make([]byte, 0x10, 0x10+len(body))
	binary.LittleEndian.PutUint32(out[0:], uint32(result))
	binary.LittleEndian.PutUint32(out[0xC:], count)
	return append(out, body...)
}

func TestStoreRelationFeedTradeBack(t *testing.T) {
	s := newTestServer(t)
	// 0x22 关系表
	arc := fspublic.NewStoreArchive(64)
	arc.WriteString("a")
	arc.WriteString("b")
	arc.WriteInt(1)
	arc.WriteDouble(60)
	s.processStoreMsg(storeRelationBack, storeBack(1, 1, arc.Bytes()))
	if !s.relations.Load() || !s.Relation.FindRelation("a", "b") {
		t.Fatal("relations flag/数据")
	}
	// 0x23 动态流（3 str + 3 int + wstr + 2 int + str + double）
	arc = fspublic.NewStoreArchive(128)
	arc.WriteString("f1") // id
	arc.WriteString("a")  // owner
	arc.WriteString("b")  // receiver
	arc.WriteInt(1)
	arc.WriteInt(0)
	arc.WriteInt(100)
	arc.WriteWideStr("hello")
	arc.WriteInt(0)
	arc.WriteInt(0)
	arc.WriteString("desc")
	arc.WriteDouble(60)
	s.processStoreMsg(storeFeedBack, storeBack(1, 1, arc.Bytes()))
	if !s.feeds.Load() || s.Relation.FeedCount() != 1 {
		t.Fatal("feeds flag/数据")
	}
	// 0x28 寄售（19 字段）
	arc = fspublic.NewStoreArchive(256)
	arc.WriteString("t1")   // id
	arc.WriteWideStr("卖家甲") // wstr
	arc.WriteString("seller1")
	arc.WriteWideStr("屠龙刀")
	arc.WriteString("k9")
	arc.WriteInt(1)
	arc.WriteInt(5000)
	arc.WriteWideStr("银两")
	arc.WriteWideStr("desc")
	arc.WriteWideStr("x1")
	arc.WriteWideStr("x2")
	arc.WriteWideStr("x3")
	arc.WriteInt(1)
	arc.WriteInt(0)
	arc.WriteInt(0)
	arc.WriteInt(0)
	arc.WriteDouble(-1)
	arc.WriteDouble(0)
	arc.WriteDouble(0)
	s.processStoreMsg(storeTradeBack, storeBack(1, 1, arc.Bytes()))
	if !s.trades.Load() || s.Trade.Count() != 1 {
		t.Fatal("trades flag/数据")
	}
	if it, _ := s.Trade.GetItemInfo("t1"); it.ItemName != "屠龙刀" {
		t.Fatalf("item = %+v", it)
	}
}

func TestWorldRegistries(t *testing.T) {
	s := newTestServer(t)
	// SceneAdd 0x90
	sa := make([]byte, sizeSceneAdd)
	binary.LittleEndian.PutUint32(sa[0:], 7)
	binary.LittleEndian.PutUint32(sa[4:], 3)
	binary.LittleEndian.PutUint32(sa[8:], 0x18)
	binary.LittleEndian.PutUint32(sa[0xC:], 0)
	putFixedStr(sa[0x10:0x50], "scene7")
	putFixedStr(sa[0x50:0x90], "res7")
	s.processWorldMsg(worldSceneAdd, sa)
	if sc, _ := s.Scenes.GetScene(7); sc == nil || sc.Name != "scene7" {
		t.Fatal("scene add")
	}
	if mid, _ := s.Scenes.GetSceneMemberId(7); mid != 3 {
		t.Fatalf("member id = %d", mid)
	}
	// PlayerAdd 0x9C
	pa := make([]byte, sizePlayerAdd)
	binary.LittleEndian.PutUint32(pa[0:], 3)
	binary.LittleEndian.PutUint32(pa[4:], 7)
	binary.LittleEndian.PutUint32(pa[8:], 0x18)
	putFixedWide(pa[0x10:0x58], "张三")
	putFixedStr(pa[0x58:0x9C], "acc1")
	s.processWorldMsg(worldPlayerAdd, pa)
	if pl, _ := s.Players.GetPlayer("张三"); pl == nil || pl.SceneID != 7 {
		t.Fatal("player add")
	}
	// RoleInfo 0x34（count@8，项 0x28：wstr name[0x20]+uid[8]）
	ri := make([]byte, sizeRoleInfo)
	binary.LittleEndian.PutUint32(ri[4:], 1)
	binary.LittleEndian.PutUint32(ri[8:], 1)
	putFixedWide(ri[0xC:0xC+0x20], "张三")
	putFixedStr(ri[0xC+0x20:0xC+0x28], "uid-9")
	s.processWorldMsg(worldRoleInfo, ri)
	if !s.nameuids.Load() {
		t.Fatal("nameuids flag")
	}
	if uid, _ := s.Names.GetUid("张三"); uid != "uid-9" {
		t.Fatalf("uid = %q", uid)
	}
	// SceneRemove
	sr := make([]byte, 8)
	binary.LittleEndian.PutUint32(sr[0:], 7)
	s.processWorldMsg(worldSceneRemove, sr)
	if _, ok := s.Scenes.GetScene(7); ok {
		t.Fatal("scene remove")
	}
}

// ---- 真 TCP 全链：store identify → 0x2D → 装载链 → Opened → member ----

// testPeer 是内网对端桩（identify + 收发帧）。
type testPeer struct {
	t      *testing.T
	conn   net.Conn
	comm   uint32
	frames chan payloadMsg
}

func dialInner(t *testing.T, port int, comm uint32) *testPeer {
	t.Helper()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("dial inner: %v", err)
	}
	p := &testPeer{t: t, conn: conn, comm: comm, frames: make(chan payloadMsg, 64)}
	// identify 自报 comm
	f, _ := encodeInner(comm, 0, innerIdentifyID2, nil)
	conn.Write(f)
	go p.readLoop()
	return p
}

func (p *testPeer) readLoop() {
	buf := make([]byte, 4096)
	acc := &frameAcc{}
	for {
		n, err := p.conn.Read(buf)
		if n > 0 {
			if err := acc.feed(buf[:n], func(c, i1, i2 uint32, pl []byte) {
				p.frames <- payloadMsg{c, i1, i2, pl}
			}); err != nil {
				return
			}
		}
		if err != nil {
			close(p.frames)
			return
		}
	}
}

func (p *testPeer) send(id2 uint32, payload []byte) {
	f, _ := encodeInner(p.comm, 0, id2, payload)
	p.conn.Write(f)
}

func (p *testPeer) recv(timeout time.Duration) (payloadMsg, bool) {
	select {
	case m, ok := <-p.frames:
		return m, ok
	case <-time.After(timeout):
		return payloadMsg{}, false
	}
}

func TestBootChainOverTCP(t *testing.T) {
	s := newTestServer(t)
	s.cfg.SaveMinutes = 0 // 关周期存盘
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	port := s.InnerPort()
	if port == 0 {
		t.Fatal("inner port 未回填")
	}
	store := dialInner(t, port, CommStoreMin)
	member := dialInner(t, port, CommMemberMin)

	// store identify → 状态机事件1 → 0x2D {ServerId=11}
	m, ok := store.recv(2 * time.Second)
	if !ok || m.id2 != storeRelationReq {
		t.Fatalf("首包 = %#x（want 0x2D）", m.id2)
	}
	if u32at(m.payload, 0) != 11 {
		t.Fatalf("0x2D payload = %v", m.payload)
	}
	if s.State() != StateLoadRelations {
		t.Fatalf("state = %v", s.State())
	}
	// 回 0x22 关系（空表）
	arc := fspublic.NewStoreArchive(16)
	s.processStoreTick(store, storeRelationBack, storeBack(1, 0, arc.Bytes()))
	// 下一 tick → 0x30 feed 配置 {ServerId; double×levels}
	s.Tick()
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storeFeedConfig {
		t.Fatalf("次包 = %#x（want 0x30）", m.id2)
	}
	if len(m.payload) != 0x108 || u32at(m.payload, 0) != 11 {
		t.Fatalf("0x30 payload len=%d id=%d", len(m.payload), u32at(m.payload, 0))
	}
	// 回 0x23 动态（空）→ 0x3A
	s.processStoreTick(store, storeFeedBack, storeBack(1, 0, arc.Bytes()))
	s.Tick()
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storeFeedReq {
		t.Fatalf("三包 = %#x（want 0x3A）", m.id2)
	}
	// 回 0x28 寄售（空）
	store.send(storeTradeBack, storeBack(1, 0, arc.Bytes()))
	// world RoleInfo 满 count=0 → +0x176
	world := dialInner(t, port, CommWorld)
	ri := make([]byte, sizeRoleInfo)
	binary.LittleEndian.PutUint32(ri[4:], 1)
	world.send(worldRoleInfo, ri)
	time.Sleep(150 * time.Millisecond)
	// 双门满足 → 首空间 0x24 {domain, nx_domains}
	s.Tick()
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storePubInfoReq {
		t.Fatalf("四包 = %#x（want 0x24）", m.id2)
	}
	if cstr(m.payload[0:0x24]) != "domain" || cstr(m.payload[0x24:0x48]) != "nx_domains" {
		t.Fatalf("0x24 = %q/%q", cstr(m.payload[0:0x24]), cstr(m.payload[0x24:0x48]))
	}
	if s.State() != StateLoadSpaces {
		t.Fatalf("state = %v", s.State())
	}
	// 回 0x19：domain 空间一条数据名
	nameArc := fspublic.NewStoreArchive(64)
	nameArc.WriteWideStr("张三的公会")
	info := make([]byte, 0x50, 0x50+nameArc.Len())
	putFixedStr(info[0:0x24], "domain")
	putFixedStr(info[0x24:0x48], "nx_domains")
	binary.LittleEndian.PutUint32(info[0x48:], 1)
	binary.LittleEndian.PutUint32(info[0x4C:], 1)
	info = append(info, nameArc.Bytes()...)
	store.send(storePubInfoBack, info)
	// → 0x25 拉取 {domain, nx_domains, 张三的公会}
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storePubLoadReq {
		t.Fatalf("五包 = %#x（want 0x25）", m.id2)
	}
	if cstr(m.payload[0:0x24]) != "domain" {
		t.Fatalf("0x25 space = %q", cstr(m.payload[0:0x24]))
	}
	// 造 PUB4 blob
	d := fspublic.NewPubData("张三的公会", "")
	d.SetAttr("level", fspublic.VTypeInt, fspublic.IntValue(5), true)
	blob, err := d.StoreArchiveBlob(false)
	if err != nil {
		t.Fatal(err)
	}
	load := make([]byte, sizeStoreLoadHdr, sizeStoreLoadHdr+len(blob))
	binary.LittleEndian.PutUint32(load[0:], 1)
	putFixedStr(load[8:8+0x24], "domain")
	putFixedWide(load[8+0x24+0x24:8+0x24+0x24+0x42], "张三的公会")
	putFixedStr(load[0x9C:0x9C+0x24], "guid-123")
	load = append(load, blob...)
	store.send(storePubLoadBack, load)
	// domain 完 → 第二空间 0x24 {guild, nx_guilds}
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storePubInfoReq || cstr(m.payload[0:0x24]) != "guild" {
		t.Fatalf("六包 = %#x %q（want 0x24 guild）", m.id2, cstr(m.payload[0:0x24]))
	}
	// guild 空 → 全空间完 → 事件2
	info2 := make([]byte, 0x50)
	putFixedStr(info2[0:0x24], "guild")
	putFixedStr(info2[0x24:0x48], "nx_guilds")
	binary.LittleEndian.PutUint32(info2[0x48:], 1)
	store.send(storePubInfoBack, info2)
	// 空间尽 → 事件2 → member 清单广播 + Opened
	m, ok = member.recv(2 * time.Second)
	if !ok || m.id2 != pushSpaceList {
		t.Fatalf("member 首包 = %#x", m.id2)
	}
	la := fspublic.NewLoadArchive(m.payload)
	n1, _ := la.ReadString()
	n2, _ := la.ReadString()
	cnt, _ := la.ReadInt()
	if n1 != "domain" || n2 != "nx_domains" || cnt != 1 {
		t.Fatalf("清单 = %q/%q/%d", n1, n2, cnt)
	}
	la2 := fspublic.NewLoadArchive(la.Rest())
	n3, _ := la2.ReadString()
	if n3 != "guild" {
		t.Fatalf("第二空间 = %q", n3)
	}
	if s.State() != StateOpened {
		t.Fatalf("state = %v（want Opened）", s.State())
	}
	// member register（8B）→ 定向 type0 清单
	reg := make([]byte, 8)
	member.send(0, reg)
	m, ok = member.recv(2 * time.Second)
	if !ok || m.id2 != 0 {
		t.Fatalf("register 应答 id2=%#x", m.id2)
	}
	// member data（0x74：prev 空）→ type1 快照
	req := make([]byte, sizeMemberDataReq)
	binary.LittleEndian.PutUint32(req[4:], 1) // count
	putFixedStr(req[8:8+0x21], "domain")
	member.send(1, req)
	m, ok = member.recv(2 * time.Second)
	if !ok || m.id2 != 1 {
		t.Fatalf("data 应答 id2=%#x", m.id2)
	}
	if len(m.payload) < 0x90 || u32at(m.payload, 4) != 0 {
		t.Fatalf("data 头 index = %d", u32at(m.payload, 4))
	}
	if fixedWide(m.payload[0x29:0x29+0x42]) != "张三的公会" {
		t.Fatalf("data 名 = %q", fixedWide(m.payload[0x29:0x29+0x42]))
	}
	// 修改数据 → Modify → Closing 收尾批量回存 0x26 → SendToWorld(2)
	gsp := s.FindPubSpace("domain")
	dd, _ := gsp.Get("张三的公会")
	dd.Modify = true
	world.send(worldClosePublic, nil) // 事件3 → Closing
	s.Tick()                          // Closing 批量回存
	m, ok = store.recv(2 * time.Second)
	if !ok || m.id2 != storePubSave {
		t.Fatalf("回存包 = %#x（want 0x26）", m.id2)
	}
	if len(m.payload) < 0x90 || cstr(m.payload[0:0x24]) != "domain" {
		t.Fatalf("0x26 头 = %q", cstr(m.payload[0:0x24]))
	}
	if got := m.payload[0x90:0x94]; binary.LittleEndian.Uint32(got) != 0x50554234 {
		t.Fatalf("blob magic = %#x", binary.LittleEndian.Uint32(got))
	}
	m, ok = world.recv(2 * time.Second)
	if !ok || m.id2 != worldCloseDone || u32at(m.payload, 0) != 1 {
		t.Fatalf("world close done = %#x", m.id2)
	}
	if s.State() != StateClosed {
		t.Fatalf("state = %v（want Closed）", s.State())
	}
}

// processStoreTick 让 store 应答落地后推进（官方应答是异步帧，此处直接
// 注入处理器 + 手动触发下一状态迁移）。
func (s *Server) processStoreTick(p *testPeer, typ uint32, payload []byte) {
	p.send(typ, payload)
	time.Sleep(100 * time.Millisecond)
}

func TestEchoWelcomeAndCommands(t *testing.T) {
	s := newTestServer(t)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", s.EchoPort()), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	buf := make([]byte, 256)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buf)
	if got := string(buf[:n]); got != "welcome public server 11\r" {
		t.Fatalf("welcome = %q", got)
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	conn.Write([]byte("state\r"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn.Read(buf)
	if got := string(buf[:n]); len(got) == 0 || got[0] != '>' {
		// 内置命令回显（state 命中命令集）
		t.Logf("state reply = %q", got)
	}
	conn.Write([]byte("bogus\r"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn.Read(buf)
	if got := string(buf[:n]); got != ">undefined command: bogus\r" {
		t.Fatalf("undefined = %q", got)
	}
}

func TestBroadcastServerInfoPacket(t *testing.T) {
	s := newTestServer(t)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	// 自宣包布局：kind=4/magic=0x3001/ip[16]/port/DistrictId/ServerId
	pkt := encodeAnnounce(s.cfg.InnerAddr, uint32(s.InnerPort()), s.cfg.DistrictID, s.cfg.ServerID)
	if u32at(pkt, 0x18) != uint32(s.InnerPort()) {
		t.Fatalf("port = %d", u32at(pkt, 0x18))
	}
	if u32at(pkt, 0x1C) != 1 || u32at(pkt, 0x20) != 11 {
		t.Fatalf("district/server 域错位")
	}
}

// ---- 发现监督器谓词 ----

// markConn 白盒标记 comm 已连（识别入登记等价；谓词只看登记面）。
func markConn(s *Server, comm uint32) {
	s.inner.mu.Lock()
	s.inner.commConn[comm] = 1
	s.inner.mu.Unlock()
}

func TestDiscoveryPendingPublic(t *testing.T) {
	s := newTestServer(t) // 谓词直驱（不 Start：监督器不起）
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "store,entry0" {
		t.Fatalf("初始缺口 = %v", p)
	}
	markConn(s, CommStoreMin)
	if p := s.DiscoveryPending(); strings.Join(p, ",") != "entry0" {
		t.Fatalf("store 连入后缺口 = %v", p)
	}
	markConn(s, CommEntryMin)
	if p := s.DiscoveryPending(); len(p) != 0 {
		t.Fatalf("全连后应就绪 = %v", p)
	}
}
