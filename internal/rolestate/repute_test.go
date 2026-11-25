package rolestate

// repute_test.go 声望/历练累加与登录簇行推。

import (
	"testing"
)

func TestAddReputeAndLiveGroove(t *testing.T) {
	var s RoleState
	row, total, added := s.AddRepute("repute_jianghu", 10)
	if row != 0 || total != 10 || !added {
		t.Fatalf("首次声望应加行: row=%d total=%d added=%v", row, total, added)
	}
	row, total, added = s.AddRepute("repute_jianghu", 20)
	if row != 0 || total != 30 || added {
		t.Fatalf("已有声望应累加: row=%d total=%d added=%v", row, total, added)
	}
	row, total, added = s.AddRepute("repute_school", 5)
	if row != 1 || total != 5 || !added {
		t.Fatalf("另一种声望应另起一行: row=%d total=%d added=%v", row, total, added)
	}
	if total, ok := s.AddLiveGroove(5, 5000); !ok || total != 5000 {
		t.Fatalf("历练累加 = %d %v", total, ok)
	}
	if total, ok := s.AddLiveGroove(5, 10000); !ok || total != 15000 {
		t.Fatalf("历练再累加 = %d %v", total, ok)
	}
	if _, ok := s.AddLiveGroove(0, 1); ok {
		t.Fatal("下标 0 不应累加")
	}
	if _, ok := s.AddLiveGroove(6, 1); ok {
		t.Fatal("下标 6 不应累加")
	}
}
