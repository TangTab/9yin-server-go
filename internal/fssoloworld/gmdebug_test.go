package fssoloworld

// gmdebug_test.go GM 调试工具（gmdebug.go）：gm rage 设置 / 上限 / 自动回复与
// 每拍回复、gm move 速度 / 重力帧与取值范围、缺省值与进场帧同源、切场景
// 重入重推、命令方言。

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// newGmDebugFixture 在线玩家 + 下行帧捕获 + gm 命令直调（不开监听）。
func newGmDebugFixture(t *testing.T) (*Server, *Player, *[][]byte, func(fields ...string) string) {
	t.Helper()
	s, pl := newCustomTestServer(t)
	pl.State = PlOnline
	frames := captureFrames(t)
	gm := func(fields ...string) string {
		t.Helper()
		client, server := net.Pipe()
		defer client.Close()
		done := make(chan struct{})
		go func() {
			// gm 命令族挂在维护控制台上（echo.go）；不开监听，直接调用。
			(&EchoConsole{srv: s}).gmCommand(server, fields)
			server.Close()
			close(done)
		}()
		out, _ := io.ReadAll(client)
		<-done
		return string(out)
	}
	return s, pl, frames, gm
}

// sentProps 已捕获帧里的 0x10 属性帧 → 每帧的 属性名→值（其余帧丢弃）。
func sentProps(frames [][]byte) []map[string]float64 {
	out := make([]map[string]float64, 0, len(frames))
	for _, frame := range frames {
		if len(frame) > 0 && frame[0] == 0x10 {
			out = append(out, decodePropertyFrame(frame, 10))
		}
	}
	return out
}

// rageState 玩家怒气会话态快照（结构体可整体比对）。
type rageState struct{ sp, maxSP, regen int }

func TestGmRageCommand(t *testing.T) {
	_, pl, frames, gm := newGmDebugFixture(t)
	live := func() rageState {
		st := &pl.combat
		st.mu.Lock()
		defer st.mu.Unlock()
		return rageState{st.live.sp, st.live.maxSP, st.rageRegen}
	}

	// 裸查询：首次进入战斗态（生存态按登录面板口径初始化），怒气 0、回复关。
	got := gm("rage", "acct")
	if !strings.HasPrefix(got, ">gm rage acct: sp=0/") || !strings.HasSuffix(got, " regen=0/s\r>end\r") {
		t.Fatalf("查询应答: %q", got)
	}
	state := live()
	maxSP := state.maxSP
	if maxSP <= 0 || state != (rageState{0, maxSP, 0}) {
		t.Fatalf("生存态初始化: %+v", state)
	}

	// 设 50 + 回复 5/秒：SP 单属性帧热发（挂玩家 wire ident）。
	*frames = nil
	got = gm("rage", "acct", "50", "5")
	if want := fmt.Sprintf(">gm rage acct: sp=50/%d regen=5/s sent (0x10 SP)\r", maxSP); !strings.HasPrefix(got, want) {
		t.Fatalf("设置应答 %q, want 前缀 %q", got, want)
	}
	if props := sentProps(*frames); len(props) != 1 || len(props[0]) != 1 || props[0]["SP"] != 50 {
		t.Fatalf("应只出一条 SP=50 属性帧, got %v", props)
	}
	if f := (*frames)[0]; binary.LittleEndian.Uint32(f[2:6]) != forkPlayerObjectID ||
		binary.LittleEndian.Uint32(f[6:10]) != forkPlayerOwnerID {
		t.Fatalf("属性帧 ident 应为玩家身份, got % x", f[:12])
	}
	if state := live(); state != (rageState{50, maxSP, 5}) {
		t.Fatalf("落值 %+v, want 50/%d/5", state, maxSP)
	}

	// 超出上限：钳到上限（= 回满），回复速率不动。
	*frames = nil
	got = gm("rage", "acct", "999999")
	if want := fmt.Sprintf(">gm rage acct: sp=%d/%d regen=5/s sent (0x10 SP)\r", maxSP, maxSP); !strings.HasPrefix(got, want) {
		t.Fatalf("钳顶应答 %q, want 前缀 %q", got, want)
	}
	if props := sentProps(*frames); len(props) != 1 || props[0]["SP"] != float64(maxSP) {
		t.Fatalf("钳顶帧 %v", props)
	}
	if state := live(); state.sp != maxSP {
		t.Fatalf("钳顶落值 %+v, want sp=%d", state, maxSP)
	}
	*frames = nil
	gm("rage", "acct", "0")
	if state := live(); state.sp != 0 || len(sentProps(*frames)) != 1 {
		t.Fatalf("清零 %+v frames=%d", state, len(*frames))
	}
	gm("rage", "acct", "max")
	if state := live(); state.sp != maxSP {
		t.Fatalf("max 应回满, %+v", state)
	}

	// 只改回复速率：怒气不动、不出帧。
	*frames = nil
	got = gm("rage", "acct", "regen", "2")
	if want := fmt.Sprintf(">gm rage acct: sp=%d/%d regen=2/s regen set\r", maxSP, maxSP); !strings.HasPrefix(got, want) {
		t.Fatalf("只改回复应答 %q, want 前缀 %q", got, want)
	}
	if len(*frames) != 0 {
		t.Fatalf("只改回复不应出帧, got %d", len(*frames))
	}
	if state := live(); state != (rageState{maxSP, maxSP, 2}) {
		t.Fatalf("只改回复的落值 %+v, want sp=%d regen=2", state, maxSP)
	}

	// 参数不合法：回用法，且不落值（含「怒气合法、速率非法」的整条拒绝）。
	for _, args := range [][]string{
		{"abc"}, {"-1"}, {"1.5"}, {"50", "-1"}, {"50", "1.5"}, {"50", "abc"},
		{"regen"}, {"regen", "-1"}, {"regen", "x"}, {"regen", "99999999999"},
		{"50", "1", "x"}, {"max", "x"},
	} {
		before := live()
		if got := gm(append([]string{"rage", "acct"}, args...)...); !strings.HasPrefix(got, ">usage: gm rage <who>") {
			t.Errorf("参数 %v 应回用法, got %q", args, got)
		}
		if after := live(); after != before {
			t.Errorf("参数 %v 不应落值: %+v → %+v", args, before, after)
		}
	}
}

// 自动回复每拍 +速率至上限（速率不设上限）；关闭后停；未开回复的玩家这一
// 拍零帧。
func TestGmRageRegenTick(t *testing.T) {
	s, pl, frames, gm := newGmDebugFixture(t)
	spFrames := func() []float64 {
		var out []float64
		for _, p := range sentProps(*frames) {
			if v, ok := p["SP"]; ok {
				out = append(out, v)
			}
		}
		return out
	}

	*frames = nil
	s.combatTick()
	if got := spFrames(); len(got) != 0 {
		t.Fatalf("未开回复不应出 SP 帧, got %v", got)
	}

	gm("rage", "acct", "10", "5")
	st := &pl.combat
	st.mu.Lock()
	maxSP := st.live.maxSP
	st.mu.Unlock()
	if maxSP <= 10 || (maxSP-10)%5 != 0 {
		t.Fatalf("夹具上限 %d 须大于 10 且与 10 的差是 5 的整倍", maxSP)
	}
	*frames = nil
	for want := 15; want <= maxSP; want += 5 {
		s.combatTick()
		if got := spFrames(); len(got) != 1 || got[0] != float64(want) {
			t.Fatalf("回复一拍 SP=%v, want [%d]", got, want)
		}
		*frames = nil
	}
	// 回满后停涨、不再出帧。
	for i := 0; i < 6; i++ {
		s.combatTick()
	}
	if got := spFrames(); len(got) != 0 {
		t.Fatalf("回满后不应再出 SP 帧, got %v", got)
	}
	st.mu.Lock()
	sp := st.live.sp
	st.mu.Unlock()
	if sp != maxSP {
		t.Fatalf("回满 sp=%d, want %d", sp, maxSP)
	}

	// 关闭回复：怒气未满也不涨。
	gm("rage", "acct", "regen", "0")
	gm("rage", "acct", "20")
	*frames = nil
	s.combatTick()
	if got := spFrames(); len(got) != 0 {
		t.Fatalf("关闭回复后不应出 SP 帧, got %v", got)
	}

	// 速率不设上限：大于剩余量的一拍直接回满，不出越界值。
	gm("rage", "acct", "10", "1000000")
	*frames = nil
	s.combatTick()
	if got := spFrames(); len(got) != 1 || got[0] != float64(maxSP) {
		t.Fatalf("大速率一拍应回满 %d, got %v", maxSP, got)
	}

	// 不在线的玩家不走这一拍。
	gm("rage", "acct", "10", "5")
	pl.State = PlOffline
	*frames = nil
	s.combatTick()
	if got := spFrames(); len(got) != 0 {
		t.Fatalf("离线玩家不应出 SP 帧, got %v", got)
	}
}

func TestGmMoveCommand(t *testing.T) {
	s, pl, frames, gm := newGmDebugFixture(t)

	// 裸查询：未改 = 进场缺省。
	got := gm("move", "acct")
	if !strings.HasPrefix(got, ">gm move acct: speed=3.9 gravity=9.8\r") {
		t.Fatalf("查询应答: %q", got)
	}

	// 移动速度：MoveSpeed + RunSpeed 两条单属性帧（先后序）。
	*frames = nil
	got = gm("move", "acct", "speed", "25")
	if !strings.HasPrefix(got, ">gm move acct: speed=25 sent (0x10 MoveSpeed+RunSpeed)\r") {
		t.Fatalf("移速应答: %q", got)
	}
	if props := sentProps(*frames); len(props) != 2 ||
		props[0]["MoveSpeed"] != 25 || len(props[0]) != 1 ||
		props[1]["RunSpeed"] != 25 || len(props[1]) != 1 {
		t.Fatalf("移速帧应为 MoveSpeed=25、RunSpeed=25 各一条, got %v", props)
	}

	// 下落重力：Gravity 单属性帧。
	*frames = nil
	got = gm("move", "acct", "gravity", "30")
	if !strings.HasPrefix(got, ">gm move acct: gravity=30 sent (0x10 Gravity)\r") {
		t.Fatalf("重力应答: %q", got)
	}
	if props := sentProps(*frames); len(props) != 1 || props[0]["Gravity"] != 30 {
		t.Fatalf("重力帧应为 Gravity=30 一条, got %v", props)
	}
	if got := gm("move", "acct"); !strings.HasPrefix(got, ">gm move acct: speed=25 gravity=30\r") {
		t.Fatalf("查询应答: %q", got)
	}
	// 小数原样（float32 最短定形）。
	gm("move", "acct", "speed", "12.345")
	if got := gm("move", "acct"); !strings.HasPrefix(got, ">gm move acct: speed=12.345 gravity=30\r") {
		t.Fatalf("小数应答: %q", got)
	}

	// 取值范围两端可用。
	for _, args := range [][]string{{"speed", "0.1"}, {"speed", "100"}, {"gravity", "0.1"}, {"gravity", "100"}} {
		if got := gm(append([]string{"move", "acct"}, args...)...); !strings.Contains(got, "sent (0x10") {
			t.Errorf("边界值 %v 应可用, got %q", args, got)
		}
	}
	// 越界 / 非数 / 参数不齐：回用法，不落值、不出帧。
	*frames = nil
	for _, args := range [][]string{
		{"speed"}, {"speed", "0"}, {"speed", "0.05"}, {"speed", "100.5"}, {"speed", "abc"},
		{"speed", "25", "x"}, {"gravity", "-1"}, {"gravity", "NaN"}, {"gravity", "Inf"}, {"fly", "3"},
	} {
		if got := gm(append([]string{"move", "acct"}, args...)...); !strings.HasPrefix(got, ">usage: gm move <who>") {
			t.Errorf("参数 %v 应回用法, got %q", args, got)
		}
	}
	if len(*frames) != 0 {
		t.Fatalf("用法错误不应出帧, got %d", len(*frames))
	}
	if got := gm("move", "acct"); !strings.HasPrefix(got, ">gm move acct: speed=100 gravity=100\r") {
		t.Fatalf("用法错误不应落值: %q", got)
	}

	// reset：会话覆写清空，两项回落缺省重发。
	*frames = nil
	got = gm("move", "acct", "reset")
	if !strings.HasPrefix(got, ">gm move acct: reset speed=3.9 gravity=9.8 sent (0x10)\r") {
		t.Fatalf("reset 应答: %q", got)
	}
	// 线上是 float32：按 float32 取值比对（3.9 → 3.9000000953674316）。
	defSpeed, defGravity := float64(gmDefaultMoveSpeed), float64(gmDefaultGravity)
	if props := sentProps(*frames); len(props) != 3 ||
		props[0]["MoveSpeed"] != defSpeed || props[1]["RunSpeed"] != defSpeed || props[2]["Gravity"] != defGravity {
		t.Fatalf("reset 帧应为 3.9/3.9/9.8 三条, got %v", props)
	}
	if got := gm("move", "acct"); !strings.HasPrefix(got, ">gm move acct: speed=3.9 gravity=9.8\r") {
		t.Fatalf("reset 后查询: %q", got)
	}
	if extra := s.gmMoveFrames(pl); len(extra) != 0 {
		t.Fatalf("reset 后不应再有会话覆写帧, got %d", len(extra))
	}
}

// 面板「调试工具」生成的命令行（与 gmpanel/jstest/debug_subpage.mjs 钉的
// 是同一份清单）原样过 echo 分发器：两端方言各写一份，这里钉对。
func TestGmDebugAcceptsPanelLines(t *testing.T) {
	s, pl, _, _ := newGmDebugFixture(t)
	s.players.SetupPlayerName(pl.Conn1, pl.Conn2, "岳绮丝")
	for _, line := range []string{
		"gm rage 岳绮丝 100 0", "gm rage 岳绮丝 50 5", "gm rage 岳绮丝 50", "gm rage 岳绮丝 max 3",
		"gm rage 岳绮丝 max", "gm rage 岳绮丝", "gm rage 岳绮丝 50 12", "gm rage 岳绮丝 regen 20",
		"gm move 岳绮丝 speed 25", "gm move 岳绮丝 gravity 30", "gm move 岳绮丝 reset",
	} {
		client, server := net.Pipe()
		done := make(chan struct{})
		go func() {
			(&EchoConsole{srv: s}).command(server, line)
			server.Close()
			close(done)
		}()
		out, _ := io.ReadAll(client)
		client.Close()
		<-done
		// 应答以「>gm rage <who>: 」开头（回执目标 + 当前值），不是用法行。
		head := ">" + strings.Join(strings.Fields(line)[:3], " ") + ": "
		if got := string(out); !strings.HasPrefix(got, head) || !strings.HasSuffix(got, "\r>end\r") {
			t.Errorf("%s 应回执 %q 开头, got %q", line, head, got)
		}
	}
}

// gm set sp 与 gm state 的 sp= 已删（怒气是会话值，走 gm rage）：存档字段
// roleState.SP 不再有 GM 读写面——`gm set sp` 按白名单外拒绝、不出帧。
func TestGmSetSPRetired(t *testing.T) {
	_, pl, frames, gm := newGmDebugFixture(t)
	*frames = nil
	got := gm("set", "acct", "sp", "50")
	if !strings.Contains(got, `unknown prop "sp"`) || strings.Contains(got, " sp ") {
		t.Fatalf("gm set sp 应按白名单外拒绝（用法行的白名单也不列它）: %q", got)
	}
	if len(*frames) != 0 {
		t.Fatalf("gm set sp 不应出帧, got %d", len(*frames))
	}
	if got := gm("state", "acct"); strings.Contains(got, " sp=") {
		t.Fatalf("gm state 不应再带存档怒气: %q", got)
	}
	// 怒气只经 gm rage 落会话态；存档字段不动。
	gm("rage", "acct", "50")
	if pl.roleState.SP != 0 {
		t.Fatalf("gm rage 不应写存档字段, got %d", pl.roleState.SP)
	}
}

// decodePropertyFrame 认 int64 线型：登录大批的 CapitalType1 是 int64，此前
// 在它处收尾，排在其后的 MaxSP 解不出来（怒气上限回退 50）。
func TestDecodePropertyFrameInt64(t *testing.T) {
	frame, err := entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("HP", entergen.Int32Value(70)),
		entergen.NamedProperty("CapitalType1", entergen.Int64Value(99999)),
		entergen.NamedProperty("MaxSP", entergen.Int32Value(80)),
	})
	if err != nil {
		t.Fatal(err)
	}
	props := decodePropertyFrame(frame, 10)
	if len(props) != 3 || props["HP"] != 70 || props["CapitalType1"] != 99999 || props["MaxSP"] != 80 {
		t.Fatalf("int64 之后的属性应照常解出, got %v", props)
	}
}

// 怒气上限 = 登录批下发给客户端的 MaxSP（同源）：GM「回满」回到的就是
// 客户端血条的上限，而不是回退值。
func TestGmRageCapIsPanelMaxSP(t *testing.T) {
	s, pl, _, gm := newGmDebugFixture(t)
	vital, err := entergen.PlayerVitalBatchFrame(s.enterContext(pl, false))
	if err != nil {
		t.Fatal(err)
	}
	sent := decodePropertyFrame(vital, 10)["MaxSP"]
	if sent != 80 {
		t.Fatalf("登录批 MaxSP = %v, want 80", sent)
	}
	if got := gm("rage", "acct", "max"); !strings.HasPrefix(got, ">gm rage acct: sp=80/80 ") {
		t.Fatalf("回满应到登录批的 MaxSP: %q", got)
	}
	st := &pl.combat
	st.mu.Lock()
	maxSP := st.live.maxSP
	st.mu.Unlock()
	if maxSP != int(sent) {
		t.Fatalf("战斗态怒气上限 %d, want 登录批 MaxSP %v", maxSP, sent)
	}
}

// rageProps 帧序里玩家对象上的 SP / MaxSP 取值：0x0D 外观帧与 0x10 属性帧
// （人物数据大批、基础属性批都是多属性帧）。
func rageProps(frames [][]byte) (sp, maxSP []float64) {
	for _, frame := range frames {
		if len(frame) == 0 || (frame[0] != 0x0D && frame[0] != 0x10) {
			continue
		}
		props := decodePropertyFrame(frame, 10)
		if frame[0] == 0x0D {
			props = decodePropertyFrame(frame, 61)
		}
		if v, ok := props["SP"]; ok {
			sp = append(sp, v)
		}
		if v, ok := props["MaxSP"]; ok {
			maxSP = append(maxSP, v)
		}
	}
	return sp, maxSP
}

// assertRage 帧序里的 SP 全为 want（至少出现一次）；出现的 MaxSP 全为登录批
// 上限 80（0x0D 不带 MaxSP，不强求出现）。
func assertRage(t *testing.T, frames [][]byte, want int) {
	t.Helper()
	sp, maxSP := rageProps(frames)
	if len(sp) == 0 {
		t.Fatalf("帧序里应带 SP（ops % x）", opOf(frames))
	}
	for _, v := range sp {
		if v != float64(want) {
			t.Errorf("SP = %v, want %d（全部 SP: %v）", v, want, sp)
			break
		}
	}
	for _, v := range maxSP {
		if v != 80 {
			t.Errorf("MaxSP = %v, want 80（全部 MaxSP: %v）", v, maxSP)
			break
		}
	}
}

// 进场/重入帧下发的怒气 = 战斗会话态，不取存档：登录（没进过战斗）与切场景
// 重入都不再出现存档里的建号初值 80，上限 MaxSP 照发。
func TestEnterFramesCarrySessionRage(t *testing.T) {
	s, pl, frames, gm := newGmDebugFixture(t)
	pl.roleState.SP = 80 // 旧档：存着建号初值
	preReady := func() [][]byte {
		t.Helper()
		out, err := entergen.PreReadyFrames(s.enterContext(pl, false), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	// 登录：没进过战斗，0x0D / 人物数据大批 / 基础属性批的 SP 全为 0。
	login := preReady()
	assertRage(t, login, 0)
	if sp, maxSP := rageProps(login); len(sp) != 3 || len(maxSP) != 2 {
		t.Fatalf("登录帧应带 3 处 SP、2 处 MaxSP, got %d / %d", len(sp), len(maxSP))
	}

	// 持 combatState.mu 取面板不得死锁：playerPanel 走 roleContext——会话
	// 怒气只在 enterContext 里取，而那把锁在这里已被持有。
	pl.panel = nil
	done := make(chan struct{})
	go func() {
		st := &pl.combat
		st.mu.Lock()
		defer st.mu.Unlock()
		s.ensureLiveLocked(st, pl)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("持 combatState.mu 取面板死锁（playerPanel 不得再走 enterContext）")
	}
	// 进过战斗（怒气 0）：仍然下发 0，不是存档值。
	assertRage(t, preReady(), 0)

	// 设了怒气：进场帧跟着会话态走。
	gm("rage", "acct", "30")
	assertRage(t, preReady(), 30)

	// 真切场景：就绪屏障前的 0x0D 与重入簇里的人物数据大批 / 基础属性批
	// 都带当前怒气（客户端怒气条不闪回 0）。
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })
	dest, ok := resolveSceneDestination("born01")
	if !ok {
		t.Fatal("born01 解析失败")
	}
	*frames = nil
	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.pendingSceneSwitch(pl) == nil {
		t.Fatal("awaitReady not armed after gap")
	}
	assertRage(t, *frames, 30)
	mark := len(*frames)
	s.clientReady(pl)
	reentry := (*frames)[mark:]
	assertRage(t, reentry, 30)
	if sp, maxSP := rageProps(reentry); len(sp) != 2 || len(maxSP) != 2 {
		t.Fatalf("重入簇应带 2 处 SP、2 处 MaxSP, got %d / %d", len(sp), len(maxSP))
	}
}

// 旧档回灌（0x02 RST1）：存档里的建号初值 SP=80 不再随档——载入即归零，
// 登录帧下发的怒气是 0。
func TestStoreLoadEnterZeroesArchiveSP(t *testing.T) {
	s, pl, frames, _ := newGmDebugFixture(t)
	state := pl.roleState
	state.SP = 80
	rst1, err := rolestate.MarshalRoleState(state)
	if err != nil {
		t.Fatal(err)
	}
	// 伪 0x02 回执 {client, serial, ok} + 尾段（空 para 宽串 + RST1）。
	p := make([]byte, 12)
	binary.LittleEndian.PutUint32(p[0:], pl.Conn1)
	binary.LittleEndian.PutUint32(p[4:], pl.Conn2)
	binary.LittleEndian.PutUint32(p[8:], 1)
	p = append(p, 2, 0, 0, 0, 0, 0)
	p = binary.LittleEndian.AppendUint32(p, 0x31545352)
	p = binary.LittleEndian.AppendUint32(p, uint32(len(rst1)))
	p = append(p, rst1...)
	*frames = nil
	if !s.storeLoadEnter(0, p) {
		t.Fatal("storeLoadEnter rejected")
	}
	if pl.roleState.SP != 0 {
		t.Fatalf("旧档 SP=%d 应回灌时归零", pl.roleState.SP)
	}
	if pl.State != PlOnline {
		t.Fatalf("进场未完成: %v", pl.State)
	}
	assertRage(t, *frames, 0)
	if sp, _ := rageProps(*frames); len(sp) != 3 {
		t.Fatalf("登录帧应带 3 处 SP, got %d", len(sp))
	}
}

// gm move 的缺省值与进场 0x0D 帧同源：两处漂了，reset 就回不到进场值。
func TestGmMoveDefaultsMatchAppearanceFrame(t *testing.T) {
	s, pl, _, _ := newGmDebugFixture(t)
	frame, err := entergen.AppearanceFrame(s.enterContext(pl, false))
	if err != nil {
		t.Fatal(err)
	}
	props := decodePropertyFrame(frame, 61)
	for name, want := range map[string]float64{
		"MoveSpeed": float64(gmDefaultMoveSpeed), "RunSpeed": float64(gmDefaultMoveSpeed),
		"Gravity": float64(gmDefaultGravity),
	} {
		if props[name] != want {
			t.Errorf("进场帧 %s=%v, gm move 缺省 %v", name, props[name], want)
		}
	}
}

// gmMoveIndex 帧序里 GM 移动参数帧（单属性，名为 MoveSpeed / RunSpeed /
// Gravity）的下标。人物数据大批（PlayerVitalBatchFrame）也带 Gravity 9.8——
// 多属性帧不算。
func gmMoveIndex(frames [][]byte) []int {
	var out []int
	for i, frame := range frames {
		if len(frame) == 0 || frame[0] != 0x10 {
			continue
		}
		p := decodePropertyFrame(frame, 10)
		if len(p) != 1 {
			continue
		}
		for _, name := range []string{"MoveSpeed", "RunSpeed", "Gravity"} {
			if _, ok := p[name]; ok {
				out = append(out, i)
			}
		}
	}
	return out
}

// 会话内跨场景保持：切场景重入会重发带缺省值的 0x0D 与人物数据大批（内含
// Gravity 9.8），重入簇里重推覆写；未改 / reset 后重入簇不带 GM 移动参数帧。
func TestGmMoveReappliedAfterSceneSwitch(t *testing.T) {
	s, pl, frames, gm := newGmDebugFixture(t)
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })
	reenter := func(key string) [][]byte {
		t.Helper()
		*frames = nil
		dest, ok := resolveSceneDestination(key)
		if !ok {
			t.Fatalf("目的地 %s 解析失败", key)
		}
		if err := s.transferPlayer(pl, dest, "test"); err != nil {
			t.Fatalf("transfer: %v", err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if s.pendingSceneSwitch(pl) == nil {
			t.Fatal("awaitReady not armed after gap")
		}
		// 就绪屏障前只有缺省值的 0x0D，没有任何 0x10 移动参数帧。
		if got := gmMoveIndex(*frames); len(got) != 0 {
			t.Fatalf("就绪屏障前不应有移动参数帧: %v", got)
		}
		mark := len(*frames)
		s.clientReady(pl)
		return (*frames)[mark:]
	}
	// 重入簇里 GM 帧的取值序列（按帧序）。
	reapplied := func(reentry [][]byte) (values []float64, first int) {
		idx := gmMoveIndex(reentry)
		if len(idx) == 0 {
			return nil, -1
		}
		for _, i := range idx {
			for _, v := range decodePropertyFrame(reentry[i], 10) {
				values = append(values, v)
			}
		}
		return values, idx[0]
	}

	if reentry := reenter("born01"); len(gmMoveIndex(reentry)) != 0 {
		t.Fatalf("未改移动参数，重入簇不应重推: %v", gmMoveIndex(reentry))
	}

	gm("move", "acct", "speed", "25")
	gm("move", "acct", "gravity", "20")
	values, first := reapplied(reenter("born03"))
	if first < 1 || len(values) != 3 || values[0] != 25 || values[1] != 25 || values[2] != 20 {
		t.Fatalf("重入簇应在 0x1F 之后重推 MoveSpeed/RunSpeed/Gravity = 25/25/20, got first=%d values=%v", first, values)
	}

	gm("move", "acct", "reset")
	if reentry := reenter("born02"); len(gmMoveIndex(reentry)) != 0 {
		t.Fatalf("reset 后重入簇不应重推: %v", gmMoveIndex(reentry))
	}
}
