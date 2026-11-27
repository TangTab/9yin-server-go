package fssoloworld

// currency_test.go 货币三币单测：GM 设置热发（gm set money/official/gold
// 各一帧 CapitalTypeN Int64）、gm state 三币余额行、任务奖励官银入账
//（prize_base[1] → CapitalType2 余额帧）。

import (
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// TestGMSetCurrencies gm set 三币：白名单受理、0x10 单属性帧携
// CapitalTypeN（官银 2 / 黄金 0）、gm state 展示三币余额。
func TestGMSetCurrencies(t *testing.T) {
	_, pl, frames, gm := newGmDebugFixture(t)
	*frames = nil
	if got := gm("set", "acct", "official", "1234"); !strings.Contains(got, "official=1234") {
		t.Fatalf("gm set official 应答异常: %q", got)
	}
	if got := gm("set", "acct", "gold", "56"); !strings.Contains(got, "gold=56") {
		t.Fatalf("gm set gold 应答异常: %q", got)
	}
	if pl.roleState.OfficialSilver != 1234 || pl.roleState.Gold != 56 {
		t.Fatalf("余额落点: official=%d gold=%d", pl.roleState.OfficialSilver, pl.roleState.Gold)
	}
	// 两帧：CapitalType2 → CapitalType0（各一帧单属性 Int64）。
	if len(*frames) != 2 {
		t.Fatalf("应 2 帧, got %d: % x", len(*frames), opOf(*frames))
	}
	cap2 := mustCapIdx(t, "CapitalType2")
	cap0 := mustCapIdx(t, "CapitalType0")
	if !frameContains(*frames, uint16Bytes(cap2)) || !frameContains(*frames, uint16Bytes(cap0)) {
		t.Fatalf("帧未携对应 CapitalType 索引: % x", *frames)
	}
	// gm state 三币余额行。
	if got := gm("state", "acct"); !strings.Contains(got, "money=") ||
		!strings.Contains(got, "official=1234") || !strings.Contains(got, "gold=56") {
		t.Fatalf("gm state 应含三币余额: %q", got)
	}
}

// TestTaskRewardOfficialSilver 完成奖励官银入账：prize_base[1] > 0 →
// OfficialSilver 累加 + CapitalType2 余额帧（碎银对之后）。
func TestTaskRewardOfficialSilver(t *testing.T) {
	s, pl := newCustomTestServer(t)
	st := &pl.roleState
	st.Money, st.OfficialSilver = 0, 100
	frames := captureFrames(t)
	task := taskdata.Task{Title: "官银奖励", Prize: "200,50"} // 碎银 200 + 官银 50
	out := &taskFrames{}
	s.npcTaskCompleteReward(pl, out, task)
	s.sendTaskFrames(pl, out)
	if st.OfficialSilver != 150 {
		t.Fatalf("官银入账=%d, want 150", st.OfficialSilver)
	}
	if st.Money != 200 {
		t.Fatalf("碎银入账=%d, want 200", st.Money)
	}
	cap2 := uint16Bytes(mustCapIdx(t, "CapitalType2"))
	if !frameContains(*frames, cap2) {
		t.Fatalf("未发官银余额帧: ops=% x", opOf(*frames))
	}
	// 官银 150 = 0x96 应出现在某帧值里。
	if !frameContains(*frames, []byte{0x96, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("官银余额帧未携 150: % x", *frames)
	}
}

// mustCapIdx 属性名 → 索引（CapitalTypeN 表内名）。
func mustCapIdx(t *testing.T, name string) uint16 {
	t.Helper()
	return clientdata.MustPlayerProperty(name).Index
}

// uint16Bytes 小端两字节。
func uint16Bytes(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }
