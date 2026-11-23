package persist

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
	"github.com/lizhanyin/9yin-server-go/internal/dbconn/dbconntest"
	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// ---- dbconntest 假库：库重启窗口与首登 INSERT 丢回包 ----

// retryTestIdentity 取一个首登身份（账号/密码槽取时间源散列，只走解析
// 不落库）。
func retryTestIdentity(t *testing.T) (login.AccountIdentity, string) {
	t.Helper()
	var acct, pw [16]byte
	copy(acct[:], "retry-acct")
	copy(pw[:], "retry-pw-")
	acct[9] = byte(time.Now().UnixNano())
	acct[10] = byte(time.Now().UnixNano() >> 8)
	identity, secret, err := login.ParseLoginCredentials(buildTestLoginFrame(t, acct, pw))
	if err != nil {
		t.Fatal(err)
	}
	return identity, secret
}

// TestVerifySurvivesRestart 库重启窗口内到达的登录校验挂起重发，库起后
// 照常首登注册。
func TestVerifySurvivesRestart(t *testing.T) {
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name: "persist-test", RetryTimeout: 3 * time.Second, PingTimeout: 100 * time.Millisecond,
		Backoff: 2 * time.Millisecond, BackoffMax: 4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	defer db.Close()
	store := &MySQLAccountStore{db: db}
	identity, secret := retryTestIdentity(t)

	f.Down()
	type result struct {
		registered bool
		err        error
	}
	done := make(chan result, 1)
	go func() {
		registered, err := store.Verify(identity, secret)
		done <- result{registered, err}
	}()
	select {
	case r := <-done:
		t.Fatalf("库停期间不得返回: %+v", r)
	case <-time.After(40 * time.Millisecond):
	}
	f.Up()
	if r := <-done; !r.registered || r.err != nil {
		t.Fatalf("库起后首登应注册成功: registered=%v err=%v", r.registered, r.err)
	}
}

// TestVerifyLostAckRegister 首登注册 INSERT 已落库、回包丢失：重发撞主键
// 后回读命中同密码 → 登录照常放行，而不是报「存储故障」断连。（此路
// registered 报 false——与并发首登竞态同一出口，该位只进日志。）
func TestVerifyLostAckRegister(t *testing.T) {
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name: "persist-test", RetryTimeout: 3 * time.Second,
		Backoff: 2 * time.Millisecond, BackoffMax: 4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	defer db.Close()
	store := &MySQLAccountStore{db: db}
	identity, secret := retryTestIdentity(t)

	f.On("SELECT `password` FROM", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, nil // 首登：账号不存在
		}
		return &dbconntest.Rows{Cols: []string{"password"}, Vals: [][]driver.Value{{secret}}}, nil
	})
	f.On("INSERT INTO `nx_district`.`dc_accounts`", func(c *dbconntest.Call) (*dbconntest.Rows, error) {
		if c.N == 1 {
			return nil, mysql.ErrInvalidConn // 已落库，回包丢
		}
		return nil, dbconntest.SQLError(1062, "Duplicate entry") // 重发撞主键
	})
	registered, err := store.Verify(identity, secret)
	if err != nil {
		t.Fatalf("丢回包的首登不该被拒（账号已在、密码相符）: registered=%v err=%v", registered, err)
	}
}
