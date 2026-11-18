package charge

import (
	"bufio"
	gonet "net"
	"testing"
	"time"
)

// startTestServer 起一个随机端口 charge 服务（FreeBackend）。
func startTestServer(t *testing.T, cfg Config) (*Server, string) {
	t.Helper()
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1"
	}
	cfg.Port = 0
	s := NewServer(cfg, nil)
	s.Logf = func(string, ...any) {}
	addr, err := s.Listen()
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	t.Cleanup(func() { s.Close() })
	return s, addr.String()
}

// dialClient 模拟官方 CChargeSockHook：连上、按行收发。
func dialClient(t *testing.T, addr string) (gonet.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := gonet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, bufio.NewReader(conn)
}

func sendLine(t *testing.T, conn gonet.Conn, line string) {
	t.Helper()
	if _, err := conn.Write([]byte(line + "\r\n")); err != nil {
		t.Fatal(err)
	}
}

func readLine(t *testing.T, r *bufio.Reader) VarList {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	vl, err := DecodeLine(trimCRLF(line))
	if err != nil {
		t.Fatal(err)
	}
	return vl
}

func trimCRLF(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// 全链路方言测试：register → login → get_points → renew → query_gift →
// logout → unregister，逐应答核对官方 Publish* 元素表。
func TestChargeFullDialect(t *testing.T) {
	cfg := DefaultConfig()
	_, addr := startTestServer(t, cfg)
	conn, r := dialClient(t, addr)

	// register：[echo,"register",serial,serverID,name,ident,port]
	sendLine(t, conn, "$gw1 $register 1 101 $server1 $5D41402A... 9600")
	resp := readLine(t, r)
	if resp.StringVal(0) != "gw1" || resp.StringVal(1) != "register" || resp.Int(2) != 1 {
		t.Fatalf("register 应答: %q", resp.EncodeLine())
	}

	// login：[echo,"login",?(2),account,passwd,ip]
	sendLine(t, conn, "$gw1 $login 0 $TEST001 $68B329DA9893E34099C7A8C296E971E9 $192.168.1.5")
	resp = readLine(t, r)
	if resp.StringVal(1) != "login" || resp.Int(4) != 1 {
		t.Fatalf("login 应答: %q", resp.EncodeLine())
	}
	if len(resp) != 14 {
		t.Fatalf("login 应答应 14 元素，得 %d: %q", len(resp), resp.EncodeLine())
	}
	if resp.StringVal(3) != "TEST001" {
		t.Errorf("账号大写化: %q", resp.StringVal(3))
	}
	userIdx := 1 // AddUser 首个

	// keep：无应答（刷新活跃）。
	sendLine(t, conn, "$gw1 $keep")
	// get_points：[echo,"get_points",useridx]
	sendLine(t, conn, "$gw1 $get_points 1")
	resp = readLine(t, r)
	if resp.StringVal(1) != "get_points" || resp.Int(2) != userIdx {
		t.Fatalf("get_points 应答: %q", resp.EncodeLine())
	}

	// renew：FreeBackend 类型 2 → 回发。
	sendLine(t, conn, "$gw1 $renew 1 $sess 0 $order")
	resp = readLine(t, r)
	if resp.StringVal(1) != "renew" || resp.Int(3) != 2 {
		t.Fatalf("renew 应答: %q", resp.EncodeLine())
	}

	// query_gift：空目录 N=0。
	sendLine(t, conn, "$gw1 $query_gift 1")
	resp = readLine(t, r)
	if resp.StringVal(1) != "query_gift" || resp.Int(3) != 0 {
		t.Fatalf("query_gift 应答: %q", resp.EncodeLine())
	}

	// item_info：空目录。
	sendLine(t, conn, "$gw1 $item_info $TEST001")
	resp = readLine(t, r)
	if resp.StringVal(1) != "item_info" || resp.Int(5) != 0 {
		t.Fatalf("item_info 应答: %q", resp.EncodeLine())
	}

	// validate：无连接前置也可用。
	sendLine(t, conn, "$web $validate $TEST001 $68B329... $10.0.0.1 0 0")
	resp = readLine(t, r)
	if resp.StringVal(0) != "web" || resp.StringVal(1) != "validate" || resp.Int(4) != 1 {
		t.Fatalf("validate 应答: %q", resp.EncodeLine())
	}

	// logout：[echo,"logout",useridx,sess,reason,account,passwd]
	sendLine(t, conn, "$gw1 $logout 1 $sess 0 $TEST001 $68B329...")
	resp = readLine(t, r)
	if resp.StringVal(1) != "logout" || resp.Int(2) != userIdx || resp.Int(3) != 1 {
		t.Fatalf("logout 应答: %q", resp.EncodeLine())
	}

	// logout 后用户已删：再按账号 logout → 官方 user not found 无应答。
	sendLine(t, conn, "$gw1 $logout 0 $sess 0 $TEST001 $x")
	conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := r.ReadString('\n'); err == nil {
		t.Fatal("已删用户的 logout 不应有应答")
	}
	conn.SetReadDeadline(time.Time{})

	// unregister
	sendLine(t, conn, "$gw1 $unregister")
	resp = readLine(t, r)
	if resp.StringVal(1) != "unregister" || resp.Int(2) != 1 {
		t.Fatalf("unregister 应答: %q", resp.EncodeLine())
	}
}

// 重复 register：官方 Unknown 态 FindServer 查重回 [echo,"register",0]
// （须第二条连接——首连接已 Online，官方对 Online 态 event1 走 error
// 无应答）。
func TestRegisterDuplicate(t *testing.T) {
	_, addr := startTestServer(t, DefaultConfig())
	conn, r := dialClient(t, addr)
	sendLine(t, conn, "$gw1 $register 1 101 $s1 $id 9600")
	if resp := readLine(t, r); resp.Int(2) != 1 {
		t.Fatalf("首次 register: %q", resp.EncodeLine())
	}
	conn2, r2 := dialClient(t, addr)
	sendLine(t, conn2, "$gw2 $register 1 101 $s1 $id 9600")
	resp := readLine(t, r2)
	if resp.Int(2) != 0 {
		t.Fatalf("重复 register 应回 0: %q", resp.EncodeLine())
	}
}

// 未 register 直接 login：官方 "Server state not online" 丢弃（无应答）。
func TestLoginBeforeRegisterNoReply(t *testing.T) {
	_, addr := startTestServer(t, DefaultConfig())
	conn, r := dialClient(t, addr)
	sendLine(t, conn, "$gw1 $login 0 $A $B $1.2.3.4")
	conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := r.ReadString('\n'); err == nil {
		t.Fatal("未注册连接的 login 不应有应答")
	}
}

// reset：清本服用户、无应答；随后 get_points 应无应答（user not found）。
func TestResetClearsUsers(t *testing.T) {
	_, addr := startTestServer(t, DefaultConfig())
	conn, r := dialClient(t, addr)
	sendLine(t, conn, "$gw1 $register 1 101 $s1 $id 9600")
	readLine(t, r)
	sendLine(t, conn, "$gw1 $login 0 $TEST001 $PWD $1.1.1.1")
	readLine(t, r)
	sendLine(t, conn, "$gw1 $reset")
	// reset 无应答；直接探测 get_points 应无应答
	sendLine(t, conn, "$gw1 $get_points 1")
	conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := r.ReadString('\n'); err == nil {
		t.Fatal("reset 后用户应已清除（get_points 无应答）")
	}
}

// 顶号：同账号第二次 login，旧连接收到空回显 renew 踢人帧。
func TestUniqueAccountKick(t *testing.T) {
	_, addr := startTestServer(t, DefaultConfig())
	conn1, r1 := dialClient(t, addr)
	sendLine(t, conn1, "$gw1 $register 1 101 $s1 $id 9600")
	readLine(t, r1)
	sendLine(t, conn1, "$gw1 $login 0 $TEST001 $PWD $1.1.1.1")
	readLine(t, r1)

	conn2, r2 := dialClient(t, addr)
	// conn2 需先 register（另一逻辑服）
	sendLine(t, conn2, "$gw2 $register 1 102 $s2 $id 9600")
	readLine(t, r2)
	sendLine(t, conn2, "$gw2 $login 0 $TEST001 $PWD $2.2.2.2")
	readLine(t, r2) // conn2 的 login 应答

	// conn1 应收到空回显 renew 踢人帧
	conn1.SetReadDeadline(time.Now().Add(time.Second))
	kick := readLine(t, r1)
	if kick.StringVal(0) != "" || kick.StringVal(1) != "renew" || kick.Int(3) != 2 {
		t.Fatalf("踢人帧形态: %q", kick.EncodeLine())
	}
}

// keep 超时：注入时钟推进 6 分钟后连接被服务端关闭。
func TestKeepTimeout(t *testing.T) {
	cfg := DefaultConfig()
	s, addr := startTestServer(t, cfg)
	base := time.Now()
	cur := base
	s.nowFunc = func() time.Time { return cur } // 注册即用注入钟
	conn, r := dialClient(t, addr)
	sendLine(t, conn, "$gw1 $register 1 101 $s1 $id 9600")
	readLine(t, r)

	cur = base.Add(6 * time.Minute) // 越过 5 分钟 keep 阈值
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, err := r.ReadByte() // 期待服务端断连（EOF），而非读超时
	if err == nil {
		t.Fatal("不应有数据")
	}
	if te, ok := err.(interface{ Timeout() bool }); ok && te.Timeout() {
		t.Fatalf("keep 超时未生效（读超时而非断连）: %v", err)
	}
}
