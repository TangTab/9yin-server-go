package login

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/net"
)

// fakeAccountStore 记录取证：registered 路径直通、fail 路径回错。
type fakeAccountStore struct {
	identity AccountIdentity
	secret   string
	calls    int
	fail     error
}

func (f *fakeAccountStore) Verify(identity AccountIdentity, secret string) (bool, error) {
	f.calls++
	f.identity, f.secret = identity, secret
	return f.fail == nil, f.fail
}

// TestFlowAccountStoreVerifyPassedCredentials Flow 应把解析出的
// 身份/密码摘要原样交给 AccountStore（通过=照常回 0x04）。
func TestFlowAccountStoreVerifyPassedCredentials(t *testing.T) {
	flight := capturedFirstFlight(t)
	accounts := &fakeAccountStore{}
	flow := &Flow{
		Clock:    func() time.Time { return time.Date(2026, 11, 18, 20, 30, 15, 0, time.Local) },
		Accounts: accounts,
	}
	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	conn, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	reply, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if want := noRoleLoginSuccessFrame(flow.Clock()); !bytes.Equal(reply, want) {
		t.Fatalf("0x04 应答 %x want %x", reply, want)
	}

	if accounts.calls != 1 {
		t.Fatalf("Verify 调用 %d 次 want 1", accounts.calls)
	}
	if got := accounts.identity.String(); got != capturedIdentity {
		t.Fatalf("交给仓的身份 %s want %s", got, capturedIdentity)
	}
	_, wantSecret, err := ParseLoginCredentials(flight)
	if err != nil {
		t.Fatal(err)
	}
	if accounts.secret != wantSecret {
		t.Fatalf("交给仓的密码摘要 %s want %s", accounts.secret, wantSecret)
	}
}

// TestFlowAccountStoreRejectClosedCredentials 仓报错（密码不符/存储
// 故障）→ 拒绝整条连接：无 0x04 应答，读侧即断。
func TestFlowAccountStoreRejectClosedCredentials(t *testing.T) {
	flight := capturedFirstFlight(t)
	accounts := &fakeAccountStore{fail: errors.New("password mismatch")}
	flow := &Flow{Accounts: accounts}
	g := &Gateway{Addr: "127.0.0.1:0", OnFrame: flow.Handle}
	addr, err := g.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Close() })
	go g.Serve()

	conn, err := net.Dial(addr.String(), net.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteMsg(flight); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ReadMsg(); err == nil {
		t.Fatal("凭据不符应收连接而不是回帧")
	}
	if accounts.calls != 1 {
		t.Fatalf("Verify 调用 %d 次 want 1", accounts.calls)
	}
}
