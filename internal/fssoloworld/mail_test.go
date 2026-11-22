package fssoloworld

// 邮件链单测：在线投递/读信移尾/领附件入包+幂等/容量上限/离线回执路由
// （帧字节级断言在 entergen/mailrecords_test.go；此处经 clientSendObserver
// 观察帧序与 roleState 落点）。

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// captureFrames 观察缝装订：收集全部下行帧 + 还原。
func captureFrames(t *testing.T) *[][]byte {
	t.Helper()
	var frames [][]byte
	prev := clientSendObserver
	clientSendObserver = func(pl *Player, frame []byte) { frames = append(frames, frame) }
	t.Cleanup(func() { clientSendObserver = prev })
	return &frames
}

func opOf(frames [][]byte) []byte {
	ops := make([]byte, len(frames))
	for i, f := range frames {
		ops[i] = f[0]
	}
	return ops
}

// recordIndexOf 0x11/0x12 帧的记录下标（offset 10 u16）。
func recordIndexOf(frame []byte) uint16 { return binary.LittleEndian.Uint16(frame[10:]) }

func TestDeliverLetterOnline(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)
	letter := NewGMLetter("测试", "欢迎", entergen.AppendixXML("tool_an_0001", 2))
	if err := s.deliverLetterOnline(pl, letter); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if len(pl.roleState.Letters) != 1 || pl.roleState.Letters[0].Serial != letter.Serial {
		t.Fatalf("letter not stored: %+v", pl.roleState.Letters)
	}
	if len(*frames) != 1 || (*frames)[0][0] != 0x11 || recordIndexOf((*frames)[0]) != entergen.RecordRecvLetter {
		t.Fatalf("deliver frame = % x, want single 0x11 rec=%d", opOf(*frames), entergen.RecordRecvLetter)
	}

	// clientReady 邮箱重放：0x11(RecvLetterRec) 行在就绪段尾部。
	*frames = nil
	s.clientReady(pl)
	replayed := false
	for _, f := range *frames {
		if f[0] == 0x11 && recordIndexOf(f) == entergen.RecordRecvLetter {
			replayed = true
		}
	}
	if !replayed {
		t.Fatalf("clientReady did not replay letter rows (%d frames)", len(*frames))
	}
}

func TestReadLetterReorder(t *testing.T) {
	s, pl := newCustomTestServer(t)
	first := NewGMLetter("一", "c1", "")
	first.Read = true
	second := NewGMLetter("二", "c2", "")
	pl.roleState.Letters = []rolestate.Letter{first, second}

	frames := captureFrames(t)
	s.handleReadLetter(pl, customMsg(clientCustomPostRead, customStr(second.Serial)))

	if len(*frames) != 2 || (*frames)[0][0] != 0x12 || (*frames)[1][0] != 0x11 {
		t.Fatalf("read frames ops = % x, want 0x12+0x11", opOf(*frames))
	}
	del, add := (*frames)[0], (*frames)[1]
	if recordIndexOf(del) != entergen.RecordRecvLetter || recordIndexOf(add) != entergen.RecordRecvLetter {
		t.Fatal("record index mismatch on read reorder")
	}
	if binary.LittleEndian.Uint16(del[12:]) != 1 { // 原行号=1（第二封）
		t.Fatalf("del row = %d, want 1", binary.LittleEndian.Uint16(del[12:]))
	}
	st := pl.roleState.Letters
	if len(st) != 2 || !st[1].Read || st[1].Serial != second.Serial || st[0].Serial != first.Serial {
		t.Fatalf("reorder state wrong: %+v", st)
	}
}

func TestClaimAppendixGrantsBagItem(t *testing.T) {
	s, pl := newCustomTestServer(t)
	letter := NewGMLetter("带附件", "领我", entergen.AppendixXML("tool_an_0001", 3))
	pl.roleState.Letters = []rolestate.Letter{letter}

	frames := captureFrames(t)
	s.handleGetAppendix(pl, customMsg(clientCustomPostGetAppendix, customStr(letter.Serial)))

	// 帧序钉死：0x19（入包预清）→0x18（行）→0x12（删行）→0x11（重推）。
	if len(*frames) != 4 {
		t.Fatalf("claim frames = %d ops % x, want 4 (0x19 0x18 0x12 0x11)", len(*frames), opOf(*frames))
	}
	for i, want := range []byte{0x19, 0x18, 0x12, 0x11} {
		if (*frames)[i][0] != want {
			t.Fatalf("claim frame[%d] op=%#x want %#x (all: % x)", i, (*frames)[i][0], want, opOf(*frames))
		}
	}
	if len(pl.roleState.BagItems) != 1 || pl.roleState.BagItems[0].ConfigID != "tool_an_0001" ||
		pl.roleState.BagItems[0].Amount != 3 {
		t.Fatalf("bag item not granted: %+v", pl.roleState.BagItems)
	}
	got := pl.roleState.Letters[0]
	if got.Appendix != "" || !got.Read {
		t.Fatalf("letter not settled after claim: %+v", got)
	}

	// 幂等：重复领取零帧零包。
	*frames = nil
	s.handleGetAppendix(pl, customMsg(clientCustomPostGetAppendix, customStr(letter.Serial)))
	if len(*frames) != 0 || len(pl.roleState.BagItems) != 1 {
		t.Fatalf("double claim not idempotent: frames=%d bag=%d", len(*frames), len(pl.roleState.BagItems))
	}
}

func TestMailBoxCap(t *testing.T) {
	s, pl := newCustomTestServer(t)
	for i := 0; i < mailBoxCap; i++ {
		pl.roleState.Letters = append(pl.roleState.Letters, NewGMLetter("t", "c", ""))
	}
	if err := s.deliverLetterOnline(pl, NewGMLetter("溢出", "c", "")); err == nil || !strings.Contains(err.Error(), "full") {
		t.Fatalf("cap not enforced: err=%v", err)
	}
	if len(pl.roleState.Letters) != mailBoxCap {
		t.Fatalf("mailbox mutated past cap: %d", len(pl.roleState.Letters))
	}
}

func TestOfflineMailReplyPatchesRST1(t *testing.T) {
	s, _ := newCustomTestServer(t)
	state, err := rolestate.BuildRoleState(rolestate.DefaultBirthScene, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	rst1, _ := rolestate.MarshalRoleState(state)

	// 合成 clientID 登记 + 伪 0x02 回执尾段（para 宽串占位 + RST1 + ACC1；
	// parseRST1/parseACC1 按 magic 扫描，不依赖精确前缀）。
	letter := NewGMLetter("离线", "补投", "")
	ctx := &offlineMailCtx{name: "测试角色", letter: letter}
	id := offlineSynthClientBase + 9999
	offlineMailMu.Lock()
	offlineMailByClient[id] = ctx
	offlineMailMu.Unlock()

	rest := []byte{2, 0, 0, 0, 0, 0} // 空 para 宽串
	rest = binary.LittleEndian.AppendUint32(rest, 0x31545352)
	rest = binary.LittleEndian.AppendUint32(rest, uint32(len(rst1)))
	rest = append(rest, rst1...)
	rest = binary.LittleEndian.AppendUint32(rest, 0x31434341)
	rest = binary.LittleEndian.AppendUint32(rest, uint32(len("acct99")))
	rest = append(rest, "acct99"...)

	s.offlineMailReply(id, rest)

	offlineMailMu.Lock()
	_, still := offlineMailByClient[id]
	offlineMailMu.Unlock()
	if still {
		t.Fatal("offline ctx not consumed")
	}
	if parseACC1(rest) != "acct99" {
		t.Fatalf("parseACC1 = %q", parseACC1(rest))
	}
}

func TestParseAppendixXML(t *testing.T) {
	config, amount := parseAppendixXML(entergen.AppendixXML("tool_x", 7))
	if config != "tool_x" || amount != 7 {
		t.Fatalf("parse = %q %d", config, amount)
	}
	if c, _ := parseAppendixXML("<Root/>"); c != "" {
		t.Fatalf("malformed should yield empty config, got %q", c)
	}
}
