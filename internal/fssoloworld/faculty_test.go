package fssoloworld

// faculty_test.go 修炼族（C2S 1006）+ 历练层 tick + 潜能赎回（C2S 235）
// 单测：选书行级增量帧、演武选卡升级通知（m191）、整分钟结算、100s
// 步进累积/转化、询价。

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// newFacultyFixture 在线玩家 + 帧捕获（itemUseFixture 同款观察缝）。
func newFacultyFixture(t *testing.T) (*Server, *Player) {
	t.Helper()
	s, pl := newCustomTestServer(t)
	pl.State = PlOnline
	return s, pl
}

// facultyBook 测试书：ng_fixture_001（不在 StarterSet/曲线表——Total 走
// 自然曲线 +200，断言确定；StaticData=0 无 varprop 档）。槽位 = 末行。
func facultyBook() rolestate.NeiGongBook {
	return rolestate.NeiGongBook{
		ConfigID: "ng_fixture_001", StaticData: 0, ItemType: 1100,
		Level: 1, MaxLevel: 3, Total: 200, MaxPower: 100,
	}
}

// TestFacultySelectAndBegin 1006/1 选书 → View43 行级增量帧（0x10
// isView=1 view=43 owner=槽位）+ 资源批；1006/11 → 资源批携 FacultyState=2。
func TestFacultySelectAndBegin(t *testing.T) {
	s, pl := newFacultyFixture(t)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, facultyBook())
	slot := uint16(len(st.NeiGong))
	frames := captureFrames(t)
	s.dispatchCustom(pl, customMsg(1006, customInt(1), customStr("ng_fixture_001")))
	got := *frames
	if len(got) != 2 || got[0][0] != 0x10 || got[1][0] != 0x10 {
		t.Fatalf("选书帧数/形态: ops=% x", opOf(got))
	}
	// 行级增量帧：0x10 且 isView=1、obj=43、own=槽位。
	if got[0][1] != 1 {
		t.Fatalf("行级增量帧 isView=%d, want 1", got[0][1])
	}
	if obj, own := binary.LittleEndian.Uint32(got[0][2:]), binary.LittleEndian.Uint32(got[0][6:]); obj != 43 || own != uint32(slot) {
		t.Fatalf("行级增量帧 obj/own=%d/%d, want 43/%d", obj, own, slot)
	}
	if !frameContains([][]byte{got[1]}, []byte("ng_fixture_001")) {
		t.Fatalf("资源批未携 FacultyName: % x", got[1])
	}
	if st.FacultyName != "ng_fixture_001" {
		t.Fatalf("FacultyName=%q", st.FacultyName)
	}
	// 普通内修开始：资源批 FacultyState=2（值 2 出现在批内）。
	*frames = nil
	s.dispatchCustom(pl, customMsg(1006, customInt(11)))
	begin := *frames
	if len(begin) != 1 || begin[0][0] != 0x10 {
		t.Fatalf("begin 帧形态: % x", opOf(begin))
	}
	if !frameContains(begin, mustHex(t, "02000000")) {
		t.Fatalf("begin 资源批未见 FacultyState=2: % x", begin[0])
	}
	if st.FacultyState != 2 || st.FacultyStyle != 1 || st.FacultyFillSpeed != 200 {
		t.Fatalf("begin 状态落点: %+v", st)
	}
	// 停：回空闲。
	s.dispatchCustom(pl, customMsg(1006, customInt(12)))
	if st.FacultyState != 0 || st.FacultyFillSpeed != 0 {
		t.Fatalf("exit 状态落点: %+v", st)
	}
}

// TestFacultyActPayLevelUp 演武选卡：满级链 191(14,1)+191(23,0)；
// 书 fill 清零、等级顶满。
func TestFacultyActPayLevelUp(t *testing.T) {
	s, pl := newFacultyFixture(t)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, facultyBook())
	s.dispatchCustom(pl, customMsg(1006, customInt(1), customStr("ng_fixture_001")))
	s.dispatchCustom(pl, customMsg(1006, customInt(21)))
	frames := captureFrames(t)
	// 现网客户端选卡以 double 发枚举值。
	s.dispatchCustom(pl, customMsg(1006, customInt(23), customDouble(0), customDouble(1)))
	got := *frames
	book := &st.NeiGong[len(st.NeiGong)-1]
	// 自然曲线：L2=400、L3=600 → 20000-200-400=19400 ≥ 600 但级顶满即停；
	// 满级 fill=0。
	if book.Level != book.MaxLevel || book.Fill != 0 || book.Total != 600 {
		t.Fatalf("选卡后书态: %+v", book)
	}
	// 帧序：行级增量 + MaxSP 专帧 + 资源批 + 面板族 + 191(14) + 191(23)。
	if len(got) < 6 {
		t.Fatalf("选卡帧数=%d: ops=% x", len(got), opOf(got))
	}
	var sawLevelUp, sawExit bool
	for _, f := range got {
		if f[0] == 0x1E && binary.LittleEndian.Uint32(f[4:]) == 191 {
			sub := int32(binary.LittleEndian.Uint32(f[9:]))
			switch sub {
			case 14:
				sawLevelUp = true
			case 23:
				sawExit = true
			}
		}
	}
	if !sawLevelUp || !sawExit {
		t.Fatalf("升级通知缺失（14=%v 23=%v）: ops=% x", sawLevelUp, sawExit, opOf(got))
	}
}

// TestFacultyTickAdvance 整分钟结算：2 分钟 → fill+400、修为-200；
// 升级时行级增量帧 + 资源批。
func TestFacultyTickAdvance(t *testing.T) {
	s, pl := newFacultyFixture(t)
	st := &pl.roleState
	st.NeiGong = append(st.NeiGong, facultyBook())
	st.Faculty = 1000
	st.FacultyName = "ng_fixture_001"
	st.FacultyState = facultyStateConvert
	st.FacultyStyle = facultyStyleNormal
	st.FacultyFillSpeed = normalFacultyFillPerMinute
	st.FacultyLastAdvance = time.Now().Add(-2 * time.Minute)
	frames := captureFrames(t)
	s.facultyTick()
	book := &st.NeiGong[len(st.NeiGong)-1]
	// fill 400 ≥ total 200 → 升一级、fill=200、total 取自然曲线 +200。
	if book.Level != 2 || book.Fill != 200 || book.Total != 400 {
		t.Fatalf("结算书态: %+v", book)
	}
	if st.Faculty != 800 {
		t.Fatalf("修为扣减=%d, want 800", st.Faculty)
	}
	got := *frames
	if len(got) != 2 || got[0][0] != 0x10 || got[1][0] != 0x10 {
		t.Fatalf("tick 帧形态: ops=% x", opOf(got))
	}
	if got[0][1] != 1 || binary.LittleEndian.Uint32(got[0][2:]) != 43 {
		t.Fatalf("tick 行级增量帧形态: % x", got[0])
	}
}

// TestLivepointTick 历练层 100s 步进：累积 +1000 毫 → 转化 10 毫
//（70% 修为 / 30% 潜能）→ LiveGroove_1..5 单属性帧 + 资源批。
func TestLivepointTick(t *testing.T) {
	s, pl := newFacultyFixture(t)
	st := &pl.roleState
	st.LivepointLast = time.Now().Add(-livepointStepSeconds * time.Second)
	frames := captureFrames(t)
	s.livepointTick()
	if st.LiveGrooves[1] != 990 {
		t.Fatalf("槽 1 余量=%d, want 990（1000-10）", st.LiveGrooves[1])
	}
	if st.Faculty != 7 || st.HoldFaculty != 3 {
		t.Fatalf("转化落点 Faculty=%d Hold=%d, want 7/3", st.Faculty, st.HoldFaculty)
	}
	got := *frames
	if len(got) != 5+1 { // 5 槽帧 + 资源批
		t.Fatalf("tick 帧数=%d, want 6: ops=% x", len(got), opOf(got))
	}
	for i := 0; i < 5; i++ {
		if got[i][0] != 0x10 {
			t.Fatalf("槽帧 %d 非 0x10: % x", i, got[i])
		}
	}
}

// TestLivepointPrice 询价回 [235,1,500]；无黄金赎回静默。
func TestLivepointPrice(t *testing.T) {
	s, pl := newFacultyFixture(t)
	frames := captureFrames(t)
	s.dispatchCustom(pl, customMsg(235, customInt(1)))
	got := *frames
	if len(got) != 1 || got[0][0] != 0x1E || binary.LittleEndian.Uint32(got[0][4:]) != 235 {
		t.Fatalf("询价帧形态: % x", got)
	}
	if !frameContains(got, mustHex(t, "f4010000")) { // 500
		t.Fatalf("询价未携价 500: % x", got[0])
	}
	// 赎回：黄金余额 0 → 静默零帧。
	*frames = nil
	st := &pl.roleState
	st.HoldFaculty = 1000
	s.dispatchCustom(pl, customMsg(235, customInt(2), customInt(2)))
	if extra := *frames; len(extra) != 0 || st.Faculty != 0 || st.HoldFaculty != 1000 {
		t.Fatalf("无黄金赎回应静默: frames=%d faculty=%d hold=%d", len(extra), st.Faculty, st.HoldFaculty)
	}
}
