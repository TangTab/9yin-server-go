package fssoloworld

// neigonguse_test.go C2S 215 运行内功切换单测：未学书拒收、换书帧序
//（MaxSP 专帧 → 资源批 → 战斗面板单帧族 → m169 新书 buff）、状态落点。

import (
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// TestUseNeiGongRejectsUnlearned 未学书拒收：零应答、CurNeiGong 不动。
func TestUseNeiGongRejectsUnlearned(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)
	before := pl.roleState.CurNeiGong
	s.dispatchCustom(pl, customMsg(215, customStr("ng_sl_001")))
	if extra := *frames; len(extra) != 0 || pl.roleState.CurNeiGong != before {
		t.Fatalf("未学书应拒收: frames=%d cur=%q want %q", len(extra), pl.roleState.CurNeiGong, before)
	}
}

// TestUseNeiGongSwitchFrames 换书帧序（2433 线 case 215 定版）：
// MaxSP 单属性专帧（先刷分母）→ 资源批（CurNeiGong/五行/FacultyName/
// BufferInfo4）→ 战斗面板单帧族 → m169 两值帧（书带 BuffID 时）。
func TestUseNeiGongSwitchFrames(t *testing.T) {
	s, pl := newCustomTestServer(t)
	resetNeiGongCatalog()
	t.Cleanup(resetNeiGongCatalog)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, rolestate.NeiGongBook{
		ConfigID: "ng_wd_003", StaticData: 5459, ItemType: 1100,
		Level: 1, MaxLevel: 36, Total: 200, Power: 0, MaxPower: 100,
		BuffID: "buf_ng_wd_003_1", BuffLevel: 1,
		StrAdd: 3, IngAdd: 2, MaxMPAdd: 50,
	})
	wantStr := make([]byte, 4)
	binary.LittleEndian.PutUint32(wantStr, uint32(int32(st.Str)+3)) // 基值 + 书 StrAdd
	frames := captureFrames(t)
	s.dispatchCustom(pl, customMsg(215, customStr("ng_wd_003")))
	got := *frames
	if st.CurNeiGong != "ng_wd_003" {
		t.Fatalf("CurNeiGong=%q, want ng_wd_003", st.CurNeiGong)
	}
	if len(got) != 1+1+8+1 {
		t.Fatalf("换书帧数=%d, want 11（MaxSP+资源批+面板8+m169）: ops=% x", len(got), opOf(got))
	}
	// 1. MaxSP 专帧：单属性 0x10，值 = 80 + 三内 30 = 110 (0x6E)。
	if got[0][0] != 0x10 || !frameContains([][]byte{got[0]}, mustHex(t, "6e000000")) {
		t.Fatalf("MaxSP 专帧异常: % x", got[0])
	}
	// 2. 资源批：多属性 0x10 携 CurNeiGong 新书 ID。
	if got[1][0] != 0x10 || !frameContains([][]byte{got[1]}, []byte("ng_wd_003")) {
		t.Fatalf("资源批未携新书 ID: % x", got[1])
	}
	// 3. 战斗面板 8 帧单属性 0x10（CombatAttributeValues 子集）。
	for i := 2; i <= 9; i++ {
		if got[i][0] != 0x10 {
			t.Fatalf("战斗面板第 %d 帧非 0x10: % x", i-2, got[i])
		}
	}
	// 面板族应含书五维加成的面板值（StrAdd=3 叠进 Str 总值后逐帧下发）。
	if !frameContains(got[2:10], wantStr) {
		t.Fatalf("战斗面板族未见书五维加成（Str=%d）: % x", int32(st.Str)+3, got[2:10])
	}
	// 4. m169 两值帧：0x1E 首值 169 + "buf_ng_wd_003_1"。
	if got[10][0] != 0x1E || binary.LittleEndian.Uint32(got[10][4:]) != 169 ||
		!frameContains([][]byte{got[10]}, []byte("buf_ng_wd_003_1")) {
		t.Fatalf("m169 新书 buff 帧异常: % x", got[10])
	}
}
