package fssoloauction

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/auction"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// ---- 帧累积器：半包/粘包/多帧/坏长度 ----

func TestInnerFrameAccumulator(t *testing.T) {
	f1, _ := encodeInner(0x18, 5, 2, []byte{1, 2, 3})
	f2, _ := encodeInner(4, 0, 0x29, []byte("store"))
	var got [][4]any
	acc := &frameAcc{}
	onFrame := func(comm, id1, id2 uint32, payload []byte) {
		got = append(got, [4]any{comm, id1, id2, append([]byte(nil), payload...)})
	}
	// 全量一次喂入
	if err := acc.feed(append(append([]byte(nil), f1...), f2...), onFrame); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0][0] != uint32(0x18) || string(got[1][3].([]byte)) != "store" {
		t.Fatalf("两帧粘包解析失败: %+v", got)
	}
	// 逐字节半包
	got = nil
	acc = &frameAcc{}
	whole := append(append([]byte(nil), f1...), f2...)
	for i, b := range whole {
		if err := acc.feed([]byte{b}, onFrame); err != nil {
			t.Fatal(err)
		}
		if len(got) > 0 && i < len(f1)-1 {
			t.Fatalf("第 %d 字节不应成帧", i)
		}
	}
	if len(got) != 2 {
		t.Fatalf("逐字节喂入应得 2 帧: %d", len(got))
	}
	// N < 12：损坏报错（官方 "message size error"）
	acc = &frameAcc{}
	if err := acc.feed([]byte{4, 0, 0, 0, 0, 0, 0, 0}, onFrame); err == nil {
		t.Fatal("N<12 应报错")
	}
}

// ---- forw（auction_forw_message_t）往返 ----

func TestForwRoundTrip(t *testing.T) {
	vl := (&proto.VarList{}).AddInt32(1).AddString("uid-x").AddInt64(42).AddFloat32(0.5)
	body := encodeForw(77, vl)
	p2, got, err := decodeForw(body)
	if err != nil || p2 != 77 || len(got.Values) != 4 || got.Values[1].Text != "uid-x" {
		t.Fatalf("forw 往返失败: %v %d %+v", err, p2, got)
	}
	if _, _, err := decodeForw(body[:6]); err == nil {
		t.Fatal("<8 字节应报错（官方断言语义）")
	}
}

// ---- 广播 announce 编解码 ----

func TestAnnounceCodec(t *testing.T) {
	pkt := encodeAnnounce("192.168.1.5", 16003, 1, 11)
	if len(pkt) != announceSize {
		t.Fatalf("announce 应为 0x24 字节: %d", len(pkt))
	}
	if binary.LittleEndian.Uint32(pkt[0:]) != announceServer ||
		binary.LittleEndian.Uint32(pkt[4:]) != announceMagic {
		t.Fatal("type/magic 头不符")
	}
	a, ok := decodeAnnounce(pkt)
	if !ok || a.Addr != "192.168.1.5" || a.Port != 16003 || a.District != 1 || a.Server != 11 {
		t.Fatalf("自宣解码失败: %+v", a)
	}
	// world 自宣（0x48）：[0x24]=序号
	w := make([]byte, 0x48)
	binary.LittleEndian.PutUint32(w[0:], announceWorld)
	binary.LittleEndian.PutUint32(w[4:], announceMagic)
	copy(w[8:], "10.0.0.1")
	binary.LittleEndian.PutUint32(w[0x18:], 15001)
	binary.LittleEndian.PutUint32(w[0x1C:], 1)
	binary.LittleEndian.PutUint32(w[0x20:], 11)
	binary.LittleEndian.PutUint32(w[0x24:], 3)
	a, ok = decodeAnnounce(w)
	if !ok || a.Type != announceWorld || a.Index != 3 || a.Addr != "10.0.0.1" {
		t.Fatalf("world 自宣解码失败: %+v", a)
	}
	// magic 不符 / 长度不足
	bad := append([]byte(nil), pkt...)
	binary.LittleEndian.PutUint32(bad[4:], 0x1234)
	if _, ok := decodeAnnounce(bad); ok {
		t.Fatal("magic 不符应忽略")
	}
	if _, ok := decodeAnnounce(pkt[:10]); ok {
		t.Fatal("长度不足应忽略")
	}
}

// ---- 状态机全流转 ----

func TestStateMachineFlow(t *testing.T) {
	var reg, loadReq, closed, openedTick, closingTick, onLoaded int
	exitSaved := -1
	m := &StateMachine{
		DistrictID: 1, ServerID: 11,
		Logf:         func(string, ...any) {},
		SendReg:      func() bool { reg++; return true },
		SendLoadReq:  func() bool { loadReq++; return loadReq <= 1 }, // 首发成功后续失败（trace 路径）
		SendClosed:   func() bool { closed++; return true },
		OnOpenedTick: func(int) { openedTick++ },
		OnClosingTick: func(int) bool {
			closingTick++
			return closingTick == 2 // 第二轮才扫完（验证重试路径）
		},
		OnServerClose: func() {},
	}
	if m.State() != StateUnknown || m.State().String() != "Unknown" {
		t.Fatal("初始态应 Unknown")
	}
	m.Event(evTick, 1) // Unknown：tick 无事
	if m.State() != StateUnknown || reg != 0 {
		t.Fatal("Unknown 态 tick 不应有动作")
	}
	m.Event(evStoreConnected, 0) // → 0x3F 注册 → LoadAuctionHall
	if reg != 1 || m.State() != StateLoadAuctionHall {
		t.Fatalf("store 连接应转 LoadAuctionHall: reg=%d state=%v", reg, m.State())
	}
	m.Event(evStoreConnected, 0) // 一次性闩锁
	if reg != 1 {
		t.Fatal("0x3F 只发一次（this+0x13E 闩锁）")
	}
	m.Event(evTick, 1) // LoadAuctionHall：每 tick 重发 0x40
	m.Event(evTick, 1)
	if loadReq != 2 { // 两拍两发（转换本身只发 0x3F，不发 0x40）
		t.Fatalf("每 tick 应重发加载请求: %d", loadReq)
	}
	m.LoadOK(false, func() { onLoaded++ }) // result=0：失败不转
	if m.State() != StateLoadAuctionHall || onLoaded != 0 {
		t.Fatal("LoadOK(false) 不应转 Opened")
	}
	m.LoadOK(true, func() { onLoaded++ }) // 0x2B → Opened + OnAuctionDataLoaded
	if m.State() != StateOpened || onLoaded != 1 {
		t.Fatalf("0x2B 应转 Opened 并回调: %v %d", m.State(), onLoaded)
	}
	m.Event(evTick, 1)
	if openedTick != 1 {
		t.Fatal("Opened 态 tick 应驱动 OnOpenedTick")
	}
	m.Event(evClose, 0) // → Closing
	if m.State() != StateClosing {
		t.Fatal("evClose 应转 Closing")
	}
	m.Event(evTick, 1) // 第一轮未扫完
	if m.State() != StateClosing || closed != 0 {
		t.Fatal("退存未完不应关播")
	}
	m.Event(evTick, 1) // 第二轮扫完 → SendToWorld → Closed
	if m.State() != StateClosed || closed != 1 {
		t.Fatalf("退存完应转 Closed: %v closed=%d", m.State(), closed)
	}
	_ = exitSaved
}

// ---- PUB4 记录块往返 + Row 互转 ----

func TestStoreBlocksRoundTrip(t *testing.T) {
	rows := [][]proto.Value{
		rowValues(auction.Row{UniqueID: "u1", ConfigID: "c1", ItemString: "IT1", SellerUid: "s1",
			StartPrice: 100, FixedPrice: 500, TransFeeCoe: 0.15, OverTime: 1_700_000_000,
			CurPrice: 120, State: 2, ItemType: 9, SellerName: "卖家甲", Level: 30, Quality: 5}),
		rowValues(auction.Row{UniqueID: "u2", ConfigID: "c2", ItemString: "IT2", SellerUid: "s2",
			StartPrice: 10, CurPrice: 10, State: 1, ItemType: 3, SellerName: "卖家乙", Level: 5, Quality: 1}),
	}
	for i := 0; i < 600; i++ { // 600 行验证 0x100(256) 分片边界
		rows = append(rows, rowValues(auction.Row{UniqueID: fmt.Sprintf("bulk-%d", i),
			ItemString: "B", CurPrice: int64(i), ItemType: 1, SellerName: "批量", State: 1}))
	}
	blocks := encodeRecordBlocks("Main", auction.MaxTotalRows, mainCols, rows)
	if len(blocks) != 3 { // 600+2=602 → 256/256/90
		t.Fatalf("602 行应切 3 片: %d", len(blocks))
	}
	var decoded []auction.Row
	for _, b := range blocks {
		name, start, rs, err := decodeRecordBlock(b)
		if err != nil {
			t.Fatal(err)
		}
		if name != "Main" {
			t.Fatalf("记录名不符: %s", name)
		}
		if start != len(decoded) {
			t.Fatalf("分片行号应连续: start=%d 已解码=%d", start, len(decoded))
		}
		for _, vs := range rs {
			r, err := valuesRow(vs)
			if err != nil {
				t.Fatal(err)
			}
			decoded = append(decoded, r)
		}
	}
	if len(decoded) != len(rows) {
		t.Fatalf("行数不符: %d ≠ %d", len(decoded), len(rows))
	}
	for i := range rows {
		a, b := rows[i], decoded[i]
		r1, _ := valuesRow(a)
		if r1 != b {
			t.Fatalf("第 %d 行往返不一致:\n%+v\n%+v", i, r1, b)
		}
	}
	// attr 块
	blk := encodeAttrBlock("soloauction", "Main", []storeAttr{
		{Name: "DistrictID", Value: proto.Value{Type: proto.VarInt32, I64: 1}},
	})
	uid, name, attrs, err := decodeAttrBlock(blk)
	if err != nil || uid != "soloauction" || name != "Main" || len(attrs) != 1 || attrs[0].Value.I64 != 1 {
		t.Fatalf("attr 块往返失败: %v %+v", err, attrs)
	}
}

// ---- world 注册表：定长布局解析 ----

func TestWorldRegistries(t *testing.T) {
	// scene add 0x90
	sb := make([]byte, sceneAddSize)
	binary.LittleEndian.PutUint32(sb[0:], 11)  // server
	binary.LittleEndian.PutUint32(sb[4:], 2)   // member
	binary.LittleEndian.PutUint32(sb[8:], 101) // scene
	copy(sb[0x10:], "扬州城")
	copy(sb[0x50:], "scene_yangzhou")
	srv, mem, sc, clone, name, res, ok := parseSceneAdd(sb)
	if !ok || srv != 11 || mem != 2 || sc != 101 || clone != 0 || name != "扬州城" || res != "scene_yangzhou" {
		t.Fatalf("scene add 解析失败: %+v", sb)
	}
	// player add 0x9C
	pb := make([]byte, playerAddSize)
	binary.LittleEndian.PutUint32(pb[0:], 7)
	binary.LittleEndian.PutUint32(pb[4:], 11)
	binary.LittleEndian.PutUint32(pb[8:], 555)
	binary.LittleEndian.PutUint32(pb[0xC:], 1)
	putWide(pb[0x10:], "玩家一")
	copy(pb[0x58:], "acct-1")
	con, psrv, pid, on, pname, acct, ok := parsePlayerAdd(pb)
	if !ok || con != 7 || psrv != 11 || pid != 555 || on != 1 || pname != "玩家一" || acct != "acct-1" {
		t.Fatalf("player add 解析失败")
	}
	// roles archive：{wide,str,bool}×2
	w := &storeWriter{}
	w.wide("角色甲")
	w.str("uid-a")
	w.byte(0)
	w.wide("角色乙")
	w.str("uid-b")
	w.byte(1)
	s := New(Options{ManualTick: true, Logf: func(string, ...any) {}})
	s.loadRoles(w.b)
	if s.nameuid.Count() != 2 || s.nameuid.GetUid("角色甲") != "uid-a" ||
		!s.nameuid.GetDeleted("角色乙") || s.nameuid.GetName("uid-a") != "角色甲" {
		t.Fatal("roles 全量重建失败")
	}
}

func putWide(dst []byte, s string) {
	units := utf16.Encode([]rune(s))
	for i, u := range units {
		if i*2+1 >= len(dst) {
			break
		}
		binary.LittleEndian.PutUint16(dst[i*2:], u)
	}
}

// ---- 全栈：member 连入 → 寄售 → 回执（真 TCP 回环）----

const testMemberComm = commMemberMin // member index 0

func newTestServer(t *testing.T, port int, ls *LocalStore) *Server {
	t.Helper()
	s := New(Options{
		DistrictID: 1, ServerID: 11,
		InnerAddr: "127.0.0.1", InnerPort: port,
		EchoAddr: "", EchoPort: -1, // 关 echo
		BroadPort:  0, // 关广播
		LocalStore: ls,
		ManualTick: true,
		Logf:       t.Logf,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	return s
}

// innerTestClient 是测试用 member 客户端（同帧协议）。
type innerTestClient struct {
	t    *testing.T
	c    net.Conn
	acc  frameAcc
	recv chan [4]any
}

func dialInner(t *testing.T, port int) *innerTestClient {
	t.Helper()
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cl := &innerTestClient{t: t, c: c, recv: make(chan [4]any, 8)}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := c.Read(buf)
			if n > 0 {
				data := append([]byte(nil), buf[:n]...)
				_ = cl.acc.feed(data, func(comm, id1, id2 uint32, payload []byte) {
					cl.recv <- [4]any{comm, id1, id2, payload}
				})
			}
			if err != nil {
				close(cl.recv)
				return
			}
		}
	}()
	return cl
}

func (cl *innerTestClient) send(comm, id1, id2 uint32, payload []byte) {
	cl.t.Helper()
	frame, err := encodeInner(comm, id1, id2, payload)
	if err != nil {
		cl.t.Fatal(err)
	}
	if _, err := cl.c.Write(frame); err != nil {
		cl.t.Fatal(err)
	}
}

func (cl *innerTestClient) waitFrame() (comm, id1, id2 uint32, payload []byte) {
	cl.t.Helper()
	select {
	case f, ok := <-cl.recv:
		if !ok {
			cl.t.Fatal("连接已断")
		}
		return f[0].(uint32), f[1].(uint32), f[2].(uint32), f[3].([]byte)
	case <-time.After(3 * time.Second):
		cl.t.Fatal("等帧超时")
		return 0, 0, 0, nil
	}
}

// consignArgs 构造 19 参寄售负载（类型序=官方 CheckConsignArgs 类型闸）。
func consignArgs(uid string, overTime int64) *proto.VarList {
	vl := &proto.VarList{}
	vl.AddInt64(9001)             // consign_id（FsGame gen_item_unique_id）
	vl.AddString(uid)             // UniqueID
	vl.AddString("cfg-42")        // ConfigID
	vl.AddInt32(0)                // SortType（透传）
	vl.AddString("ITEMDATA|42|1") // ItemString
	vl.AddString("seller-uid-1")  // SellerUid
	vl.AddInt64(100)              // StartPrice
	vl.AddInt64(500)              // FixedPrice
	vl.AddFloat32(0.15)           // TransFeeCoe
	vl.AddInt64(1)                // KeepingFee
	vl.AddInt64(overTime)         // OverTime
	vl.AddString("")              // CurBiderUid
	vl.AddInt64(100)              // CurPrice
	vl.AddInt32(auction.StateConsigned)
	vl.AddInt32(9)   // ItemType（无资源表 → Other 兜底）
	vl.AddWide("卖家") // SellerName
	vl.AddInt32(30)  // Level
	vl.AddInt32(5)   // Quality
	vl.AddString("") // ExtraStr
	return vl
}

func TestServerMemberFullStack(t *testing.T) {
	port := freePort(t)
	srv := newTestServer(t, port, nil)
	cl := dialInner(t, port)
	defer cl.c.Close()

	// identify：member index 0（comm 0x18）
	cl.send(testMemberComm, 0, innerIdentifyID2, nil)

	// 寄售：id1=5（目标透传）、id2=2（MemberMessage）、param2=77
	msg := (&proto.VarList{}).AddInt32(auction.MsgConsign)
	for _, v := range consignArgs("uid-full-1", time.Now().Add(time.Hour).Unix()).Values {
		msg.Values = append(msg.Values, v)
	}
	cl.send(testMemberComm, 0, 2, encodeForw(77, msg))

	comm, id1, id2, payload := cl.waitFrame()
	if comm != commSelf || id1 != 0 || id2 != 0xD {
		t.Fatalf("回执帧头不符: comm=%d id1=%d id2=%#x", comm, id1, id2)
	}
	p2, ack, err := decodeForw(payload)
	if err != nil || p2 != 77 {
		t.Fatalf("回执 forw 头不符: %v param2=%d", err, p2)
	}
	if len(ack.Values) != 3 || ack.Values[0].I64 != auction.MsgConsignAck ||
		ack.Values[1].I64 != 9001 || ack.Values[2].Text != "ITEMDATA|42|1" {
		t.Fatalf("寄售回执不符（[7, consign_id, itemstring]）: %+v", ack.Values)
	}
	if n := srv.mgr.GetItemNum(); n != 1 {
		t.Fatalf("入表行数应 1: %d", n)
	}
	srv.Stop()
}

// ---- 超时结算：300s 扫描（HCheckDueTime）→ 广播 target=0 ----

func TestServerOvertimeBroadcast(t *testing.T) {
	port := freePort(t)
	dir := t.TempDir()
	srv := newTestServer(t, port, &LocalStore{Dir: dir, Logf: t.Logf})
	cl := dialInner(t, port)
	defer cl.c.Close()
	cl.send(testMemberComm, 0, innerIdentifyID2, nil)

	// LocalStore 加载序：Start 已发 store-connected → LoadAuctionHall；
	// 第一拍完成 0x40→0x29/0x2A/0x2B → Opened
	srv.TimerCheck(1)
	if srv.sm.State() != StateOpened {
		t.Fatalf("首拍后应 Opened: %v", srv.sm.State())
	}

	// 寄售一件已过期物品（State=1 无出价 → 流拍 msg 12）
	msg := (&proto.VarList{}).AddInt32(auction.MsgConsign)
	for _, v := range consignArgs("uid-expired", time.Now().Add(-time.Minute).Unix()).Values {
		msg.Values = append(msg.Values, v)
	}
	cl.send(testMemberComm, 0, 2, encodeForw(9, msg))
	if _, _, _, _ = cl.waitFrame(); false {
	} // 消费寄售回执 [7]
	if n := srv.mgr.GetItemNum(); n != 1 {
		t.Fatalf("过期物品应先入表: %d", n)
	}

	// 驱动 299 拍（不足 300s）无广播；第 300 拍触发
	for i := 0; i < 299; i++ {
		srv.TimerCheck(1)
	}
	select {
	case f := <-cl.recv:
		t.Fatalf("300s 前不应广播: %+v", f)
	case <-time.After(100 * time.Millisecond):
	}
	srv.TimerCheck(1) // 第 300 拍

	comm, id1, id2, payload := cl.waitFrame()
	if comm != commSelf || id1 != 0 || id2 != 0xD {
		t.Fatalf("超时广播帧头不符: comm=%d id1=%d id2=%#x", comm, id1, id2)
	}
	p2, out, err := decodeForw(payload)
	if err != nil || p2 != 0 {
		t.Fatalf("超时广播 param2 应 0: %v %d", err, p2)
	}
	if out.Values[0].I64 != auction.MsgUnsoldReturn || len(out.Values) != 5 ||
		out.Values[1].Text != "uid-expired" {
		t.Fatalf("流播退回应为 [12, uid, configid, selleruid, itemstring]: %+v", out.Values)
	}
	if n := srv.mgr.GetItemNum(); n != 0 {
		t.Fatalf("结算后应移除: %d", n)
	}
	srv.Stop()
}

// ---- LocalStore 持久化：存盘 → 重启回放 ----

func TestServerLocalStorePersistence(t *testing.T) {
	dir := t.TempDir()
	port := freePort(t)
	srv := newTestServer(t, port, &LocalStore{Dir: dir, Logf: t.Logf})
	srv.TimerCheck(1) // → Opened
	if srv.sm.State() != StateOpened {
		t.Fatalf("应 Opened: %v", srv.sm.State())
	}
	msg := (&proto.VarList{}).AddInt32(auction.MsgConsign)
	for _, v := range consignArgs("uid-persist", time.Now().Add(time.Hour).Unix()).Values {
		msg.Values = append(msg.Values, v)
	}
	srv.processMemberMsg(1, 2, encodeForw(1, msg))
	if srv.mgr.GetItemNum() != 1 {
		t.Fatal("寄售应入表")
	}
	srv.innerSaveData(true)
	if _, err := os.Stat(filepath.Join(dir, "main.attr")); err != nil {
		t.Fatalf("attr 未落盘: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "main.record")); err != nil {
		t.Fatalf("record 未落盘: %v", err)
	}
	srv.Stop()

	// 重启同目录：回放后行数保持
	port2 := freePort(t)
	srv2 := newTestServer(t, port2, &LocalStore{Dir: dir, Logf: t.Logf})
	srv2.TimerCheck(1)
	if srv2.sm.State() != StateOpened {
		t.Fatalf("重启后应 Opened: %v", srv2.sm.State())
	}
	if n := srv2.mgr.GetItemNum(); n != 1 {
		t.Fatalf("重启后行数应 1: %d", n)
	}
	if uid := srv2.mgr.Main.Rows[srv2.mgr.Main.FindItem("uid-persist")].ItemString; uid != "ITEMDATA|42|1" {
		t.Fatalf("回放行内容不符: %s", uid)
	}
	srv2.Stop()
}

// ---- echo 行协议 ----

func TestEchoConsoleProtocol(t *testing.T) {
	port := freePort(t)
	s := New(Options{
		InnerAddr: "127.0.0.1", InnerPort: freePort(t),
		EchoAddr: "127.0.0.1", EchoPort: port,
		BroadPort: 0, ManualTick: true,
		Logf: func(string, ...any) {},
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(3 * time.Second))

	// 空行不分发；\r 与 \n 皆终止；命令应答带 \r\n
	if _, err := c.Write([]byte("\r\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte("state\r\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 512)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	got := string(buf[:n])
	if len(got) < 6 || got[:6] != "state=" {
		t.Fatalf("state 命令应答不符: %q", got)
	}
	// 未知命令
	if _, err := c.Write([]byte("bogus\n")); err != nil {
		t.Fatal(err)
	}
	n, err = c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if want := "[echo] 未知命令"; len(string(buf[:n])) < len(want) || string(buf[:n])[:len(want)] != want {
		t.Fatalf("未知命令应答不符: %q", string(buf[:n]))
	}
}

// ---- 内网来源路由：未知 comm 拒收 ----

func TestInnerSourceRouting(t *testing.T) {
	port := freePort(t)
	srv := newTestServer(t, port, nil)
	cl := dialInner(t, port)
	defer cl.c.Close()
	// comm 0x40（不在 world/store/member 任一命名空间）→ "message source error"
	cl.send(0x40, 0, 2, encodeForw(1, (&proto.VarList{}).AddInt32(auction.MsgConsign)))
	select {
	case f := <-cl.recv:
		t.Fatalf("未知来源不应回帧: %+v", f)
	case <-time.After(300 * time.Millisecond):
	}
	srv.Stop()
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// ---- 发现监督器：store 广播门 +4 勘误 + 就绪谓词 ----

func TestOnBroadcastStoreGateRawID(t *testing.T) {
	// 假 store 受面：收 auction identify 后回 identify comm 4
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	storePort := ln.Addr().(*net.TCPAddr).Port
	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			accepted <- c
		}
	}()

	s := New(Options{
		DistrictID: 1, ServerID: 11,
		InnerAddr: "127.0.0.1", InnerPort: freePort(t),
		EchoAddr: "", EchoPort: -1,
		BroadPort:  0, // 关广播口（直喂 onBroadcast）
		ManualTick: true,
		Logf:       t.Logf,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// store 自宣 +0x24 = 裸 StoreID 0（fssolostore 同构；官方 auction
	// 门要求完整 comm 4..7 属永不命中的失配——勘误后 +4 基址）
	pkt := make([]byte, 0x28)
	binary.LittleEndian.PutUint32(pkt[0:], announceStore)
	binary.LittleEndian.PutUint32(pkt[4:], announceMagic)
	copy(pkt[8:], "127.0.0.1")
	binary.LittleEndian.PutUint32(pkt[0x18:], uint32(storePort))
	binary.LittleEndian.PutUint32(pkt[0x1C:], 1)
	binary.LittleEndian.PutUint32(pkt[0x20:], 11)
	binary.LittleEndian.PutUint32(pkt[0x24:], 0)
	s.onBroadcast(pkt, "127.0.0.1", 0)

	var c net.Conn
	select {
	case c = <-accepted:
		defer c.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("裸 StoreID 0 自宣未触发外连（门勘误失效）")
	}
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	hdr := make([]byte, 16)
	if _, err := io.ReadFull(c, hdr); err != nil {
		t.Fatal(err)
	}
	if comm := binary.LittleEndian.Uint32(hdr[4:]); comm != commSelf {
		t.Fatalf("auction identify comm=%d（应 %d）", comm, commSelf)
	}
	if id2 := binary.LittleEndian.Uint32(hdr[12:]); id2 != innerIdentifyID2 {
		t.Fatalf("首帧应 identify，id2=%#x", id2)
	}
	frame, _ := encodeInner(commStoreMin, 0, innerIdentifyID2, nil)
	c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.Write(frame); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(s.DiscoveryPending()) == 0 && s.inner.Connected(commStoreMin) {
			return // identify 应答即就绪
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("store identify 后未就绪：pending=%v connected=%v",
		s.DiscoveryPending(), s.inner.Connected(commStoreMin))
}

func TestDiscoveryPendingLocalStore(t *testing.T) {
	// LocalStore 垫片在场即视作 store 已连（Start 内 evStoreConnected 同语义）
	s := New(Options{
		DistrictID: 1, ServerID: 11,
		InnerAddr: "127.0.0.1", InnerPort: freePort(t),
		EchoAddr: "", EchoPort: -1,
		BroadPort: 0, ManualTick: true,
		Logf:      func(string, ...any) {},
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	if p := s.DiscoveryPending(); len(p) != 1 || p[0] != "store0" {
		t.Fatalf("无 store 时缺口应为 [store0]：%v", p)
	}

	ls := &LocalStore{}
	s2 := New(Options{
		DistrictID: 1, ServerID: 11,
		InnerAddr: "127.0.0.1", InnerPort: freePort(t),
		EchoAddr: "", EchoPort: -1,
		BroadPort: 0, ManualTick: true,
		LocalStore: ls,
		Logf:      func(string, ...any) {},
	})
	if err := s2.Start(); err != nil {
		t.Fatal(err)
	}
	defer s2.Stop()
	if p := s2.DiscoveryPending(); len(p) != 0 {
		t.Fatalf("LocalStore 垫片应即就绪：%v", p)
	}
}
