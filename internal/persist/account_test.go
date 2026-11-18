package persist

import (
	"context"
	"encoding/binary"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/login"
)

// testDSN 集成测试开关：不设则跳过（CI/无库环境不误红）。
// 本机实跑示例：
//
//	NINEYIN_MYSQL_TEST_DSN=nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin \
//	  go test ./internal/persist/
const testDSNEnv = "NINEYIN_MYSQL_TEST_DSN"

// buildTestLoginFrame 按 auth.go 钉死的 op02 布局造最小合法帧
// （header 25B：opcode + 20B 填充 + u32 accountLen；三槽均 16B）。
func buildTestLoginFrame(t *testing.T, acct, pw [16]byte) []byte {
	t.Helper()
	msg := make([]byte, 0, 25+4+16+4+16+4+4+16+4)
	msg = append(msg, login.OpClientLogin)
	msg = append(msg, make([]byte, 20)...) // [0x01:0x15) 填充（u32 333 语义未证）
	var u32 [4]byte
	put := func(v uint32) { binary.LittleEndian.PutUint32(u32[:], v); msg = append(msg, u32[:]...) }
	put(uint32(len(acct)))
	msg = append(msg, acct[:]...)
	put(uint32(len(pw)))
	msg = append(msg, pw[:]...)
	put(1)  // client-type
	put(16) // login-string
	msg = append(msg, make([]byte, 16)...)
	put(1) // login-type（必须恰好收尾）
	return msg
}

func TestAccountStoreRegisterAndVerify(t *testing.T) {
	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("设 %s 才跑 MySQL 集成测试", testDSNEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := OpenAccountStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	// 唯一账号槽（时间源散列）→ 每次运行独立账号，不惊动既有行。
	var acct, pwA, pwB [16]byte
	binary.LittleEndian.PutUint64(acct[:8], uint64(time.Now().UnixNano()))
	copy(pwA[:], "secret-a")
	copy(pwB[:], "secret-b")

	identity, secretA, err := login.ParseLoginCredentials(buildTestLoginFrame(t, acct, pwA))
	if err != nil {
		t.Fatal(err)
	}
	identityAgain, secretB, err := login.ParseLoginCredentials(buildTestLoginFrame(t, acct, pwB))
	if err != nil {
		t.Fatal(err)
	}
	if identity.String() != identityAgain.String() {
		t.Fatal("同账号槽异密码帧应得同身份")
	}
	t.Cleanup(func() {
		// 先删行再关池（两动作并作一个 Cleanup，避免 defer Close 抢先
		// 把池关掉、DELETE 落在已关连接上静默失败——首轮实测踩过）。
		if _, err := store.db.Exec(
			"DELETE FROM `nx_district`.`dc_accounts` WHERE `name`=?", identity.String()); err != nil {
			t.Logf("清理测试行失败: %v", err)
		}
		store.Close()
	})

	if registered, err := store.Verify(identity, secretA); err != nil || !registered {
		t.Fatalf("首登应自动注册：registered=%t err=%v", registered, err)
	}
	if registered, err := store.Verify(identity, secretA); err != nil || registered {
		t.Fatalf("二次同密应通过且不再注册：registered=%t err=%v", registered, err)
	}
	if _, err := store.Verify(identity, secretB); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("异密应拒绝并报 mismatch，得 err=%v", err)
	}

	// 落库形态：官方快照默认值齐备、lastlog_time 已刷新
	//（datetime 比对放 SQL 侧——DSN 未开 parseTime，驱动回 []byte）。
	var password string
	var userType, status int
	var touchedInOrder bool
	err = store.db.QueryRow(
		"SELECT `password`,`user_type`,`status`,`lastlog_time`>=`create_time` "+
			"FROM `nx_district`.`dc_accounts` WHERE `name`=?", identity.String()).
		Scan(&password, &userType, &status, &touchedInOrder)
	if err != nil {
		t.Fatal(err)
	}
	if password != secretA || userType != 51 || status != 1 {
		t.Fatalf("行形态 password=%q user_type=%d status=%d", password, userType, status)
	}
	if !touchedInOrder {
		t.Fatal("lastlog_time 早于 create_time")
	}
}
