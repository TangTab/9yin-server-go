package main

import (
	"encoding/binary"
	"strings"
	"sync"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

// frameRecorder 是 sceneMessageConnection 的测试替身：收集会话循环写出
// 的全部 S2C 帧（main.go processGMCommand 的 link.WriteFrame 去处）。
type frameRecorder struct {
	mu     sync.Mutex
	frames [][]byte
}

func (r *frameRecorder) WriteFrame(frame []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := make([]byte, len(frame))
	copy(clone, frame)
	r.frames = append(r.frames, clone)
	return nil
}

func (r *frameRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

func (r *frameRecorder) last() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.frames) == 0 {
		return nil
	}
	return r.frames[len(r.frames)-1]
}

// gmEffectHarness 组装「真实执行器 + 假连接」：playerActor / gmSession /
// frameRecorder 与 main.go handle() 里的对象同款，processGMCommand 闭包
// 各分支的调用序列在用例里逐字照抄（闭包自身无独立逻辑，只是把这些
// 调用串起来）。断言三层：玩家内存值、写给客户端的帧、面板状态回读。
func gmEffectHarness() (*playerActor, *gmSession, *frameRecorder) {
	hub := newGMHub()
	session := hub.attach(7, "127.0.0.1:5555", &role.RoleSnapshot{
		RoleSummary: role.RoleSummary{Name: "测试侠"},
		Location: role.Location{
			Scene:    role.Scene{Config: `ini\scene\city04_LuoYang`, Resource: "city04"},
			Position: role.Position{X: 243.4, Y: 43.3, Z: 403.8, Orient: 5.0},
		},
	}, "少林", 50, 0, 0, 10, 9_999_999)
	player := newPlayerActor("测试侠", 0)
	return player, session, &frameRecorder{}
}

// TestGMEffectSetSilver set_silver 全链路：内存值、CapitalType1 帧、
// 面板快照（顶栏碎银 + 最近操作）三层同时生效。
func TestGMEffectSetSilver(t *testing.T) {
	player, session, link := gmEffectHarness()

	// —— main.go processGMCommand case "set_silver" 原样调用序列 ——
	player.setSilver(12345)
	update, err := player.silverUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	session.setSilver(12345)

	if got := player.silverBalance(); got != 12345 {
		t.Fatalf("player silver=%d want 12345（内存值未生效）", got)
	}
	frame := link.last()
	if len(frame) != 22 || frame[0] != 0x10 {
		t.Fatalf("frame=% x（CapitalType1 瘦帧形破坏）", frame)
	}
	if index := binary.LittleEndian.Uint16(frame[12:]); index != propCapitalType1 {
		t.Fatalf("frame prop=%d want CapitalType1(%d)", index, propCapitalType1)
	}
	if value := int32(binary.LittleEndian.Uint64(frame[14:])); value != 12345 {
		t.Fatalf("frame value=%d want 12345（客户端帧未生效）", value)
	}
	state := session.snapshot()
	if state.Silver != 12345 || !strings.Contains(state.LastAction, "碎银设为 12345") {
		t.Fatalf("panel state=%+v（面板状态未生效）", state)
	}
}

// TestGMEffectSetSP set_sp/add_sp/fill_sp 家族：合法值原样生效，超上限
// 饱和到有效上限（80+内功加成，官方满怒语义），帧带钳制后的值，面板
// 快照带回执。
func TestGMEffectSetSP(t *testing.T) {
	player, session, link := gmEffectHarness()
	cap := player.rageCap()

	// 合法区间内的值原样生效。
	sp, update, err := player.setSP(50)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	session.setSP(sp, "怒气设为 50")
	if sp != 50 {
		t.Fatalf("setSP returned %d want 50", sp)
	}
	frame := link.last()
	if len(frame) < 14 || frame[0] != 0x10 {
		t.Fatalf("sp frame=% x", frame)
	}
	state := session.snapshot()
	if state.SP != 50 || !strings.Contains(state.LastAction, "怒气设为 50") {
		t.Fatalf("panel state=%+v", state)
	}

	// add_sp：增量语义（50+20）。
	sp, update, err = player.addSP(20)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	if sp != 70 {
		t.Fatalf("addSP result=%d want 70", sp)
	}

	// fill_sp / 超上限 set_sp：饱和到有效上限（新角色 80+内功加成）。
	sp, update, err = player.setSP(cap)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	if sp != cap {
		t.Fatalf("fill_sp result=%d want cap=%d", sp, cap)
	}
	sp, update, err = player.setSP(99999)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	if sp != cap {
		t.Fatalf("oversized setSP=%d want clamp to cap=%d", sp, cap)
	}
	if link.count() != 4 {
		t.Fatalf("frames=%d want 4", link.count())
	}
}

// TestGMEffectCurrencyAndFaculty 官银/黄金/修为：面板最常用的三类数值。
func TestGMEffectCurrencyAndFaculty(t *testing.T) {
	player, session, link := gmEffectHarness()

	// set_guanyin（case 序列：setGuanYin → guanYinUpdate → 写帧 → 会话回写）。
	player.setGuanYin(888)
	update, err := player.guanYinUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	session.setGuanYin(888)
	if got := player.guanYinValue(); got != 888 {
		t.Fatalf("guanYin=%d want 888", got)
	}
	if index := binary.LittleEndian.Uint16(link.last()[12:]); index != propCapitalType2 {
		t.Fatalf("guanYin frame prop=%d want CapitalType2", index)
	}

	// set_gold（会话级）。
	player.setGold(42)
	update, err = player.goldUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	session.setGold(42)
	if got := player.goldBalance(); got != 42 {
		t.Fatalf("gold=%d want 42", got)
	}
	if state := session.snapshot(); state.Gold != 42 {
		t.Fatalf("panel gold=%d", state.Gold)
	}

	// set_faculty（修为：帧 + 面板 Faculty）。
	update, err = player.setFacultyValue(123456)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(update); err != nil {
		t.Fatal(err)
	}
	session.setFaculty(123456)
	if got := player.facultyValue(); got != 123456 {
		t.Fatalf("faculty=%d want 123456", got)
	}
	if state := session.snapshot(); state.Faculty != 123456 {
		t.Fatalf("panel faculty=%d", state.Faculty)
	}
}

// TestGMEffectFiveAttributes set_five_attributes：五维覆写（0=跳过该维）
// 后的全量帧携带五维属性，且内存值生效。
func TestGMEffectFiveAttributes(t *testing.T) {
	player, _, link := gmEffectHarness()

	frame, err := player.setBaseAttributes(100, 0, 80, 0, 90)
	if err != nil {
		t.Fatal(err)
	}
	if err := link.WriteFrame(frame); err != nil {
		t.Fatal(err)
	}
	// 内存值：非 0 维度写入，0 维度保留原值。
	player.mu.Lock()
	attrs := player.progress.attributes
	player.mu.Unlock()
	if attrs.str != 100 || attrs.dex != 80 || attrs.spi != 90 {
		t.Fatalf("attributes=%+v want str=100 dex=80 spi=90", attrs)
	}
	// 全量 0x10 帧必须携带全部五维索引（客户端整组刷新）。
	indices, err := propertyFrameIndices(link.last())
	if err != nil {
		t.Fatal(err)
	}
	present := map[uint16]bool{}
	for _, index := range indices {
		present[index] = true
	}
	for _, want := range []uint16{propStr, propSta, propDex, propIng, propSpi} {
		if !present[want] {
			t.Fatalf("frame indices=%v missing %d", indices, want)
		}
	}
}
