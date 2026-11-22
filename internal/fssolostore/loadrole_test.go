package fssolostore

// LoadRole account 空串 = name-only 匹配（GM 离线邮件 world 合成 clientID
// 查档路径）；非空时账户门行为不变。

import (
	"testing"
)

func TestMemoryLoadRoleNameOnly(t *testing.T) {
	m := NewMemoryStore("")
	t.Cleanup(func() { m.Close() })
	meta := RoleMeta{Account: "acctA", ServerID: 1, Location: "1,2,3,4"}
	if ok, err := m.CreateRole("acctA", "离线侠客", "uid1", 1, meta, nil); err != nil || !ok {
		t.Fatalf("create: ok=%v err=%v", ok, err)
	}

	// name-only：account 空串命中。
	_, got, ok, err := m.LoadRole("", "离线侠客", 1)
	if err != nil || !ok {
		t.Fatalf("name-only load: ok=%v err=%v", ok, err)
	}
	if got.Account != "acctA" {
		t.Fatalf("meta account = %q, want acctA（回写 0x04 需要真实账户）", got.Account)
	}

	// 非空 account 语义不变：错账户拒。
	if _, _, ok, _ := m.LoadRole("acctB", "离线侠客", 1); ok {
		t.Fatal("wrong account should not match")
	}
	// 正确账户照常。
	if _, _, ok, _ := m.LoadRole("acctA", "离线侠客", 1); !ok {
		t.Fatal("correct account should match")
	}
}
