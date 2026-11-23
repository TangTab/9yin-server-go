package fssoloworld

// 技能受理链测试：帧序与取值按官服女号抓包定版
//（docs/official/技能服务端XML构建链与m211受理链.md），技能取值来自
// tools/skillgen 构建产物（仓内 testdata/ini/Skill，缺席则跳过）。
//
// 断言面是「可读化的帧序」：describeFrames 把每个下行帧转写为
// 属性/自定义消息/冷却表格/位置的紧凑文本，与期望串按 token 比对
//（期望串里用 " | " 分行只是为了可读）。冷却表起止折算为相对首次施放
// 时刻的毫秒偏移——会话时钟由夹具拨动。

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// skillFixture 一个在线玩家 + 可拨动的会话时钟 + 下行帧捕获。
type skillFixture struct {
	s      *Server
	pl     *Player
	out    *[][]byte
	clock  *time.Time
	origin time.Time // 时间偏移基准（首次施放时刻）
}

func newSkillFixture(t *testing.T) *skillFixture {
	t.Helper()
	if _, err := os.Stat(filepath.Join("..", "..", "testdata", "ini", "Skill", "skill_static.xml")); err != nil {
		t.Skipf("技能构建产物缺席（go run ./tools/skillgen）: %v", err)
	}
	s := newNpcTestServer(t)
	// 登记进玩家表（combatTick 按表驱动）；存档水位推到远期，位移收编
	// 不触发 store 存档（测试内网无连接，徒增日志）。
	pl := s.players.AddPlayer(7, 0, "")
	pl.State, pl.haveRoleState = PlOnline, true
	pl.roleState.MP = 55
	pl.posSaveAt = time.Now().Add(time.Hour)
	var frames [][]byte
	prevOut, prevNow := clientSendObserver, combatNow
	clientSendObserver = func(_ *Player, frame []byte) { frames = append(frames, frame) }
	now := time.Unix(1_700_000_000, 0)
	combatNow = func() time.Time { return now }
	t.Cleanup(func() { clientSendObserver, combatNow = prevOut, prevNow })
	return &skillFixture{s: s, pl: pl, out: &frames, clock: &now, origin: now}
}

func (f *skillFixture) advance(d time.Duration) { *f.clock = f.clock.Add(d) }

// take 取走已捕获的下行帧并清空。
func (f *skillFixture) take() [][]byte {
	frames := *f.out
	*f.out = nil
	return frames
}

// finish 手动触发收招（定时器走真实时钟，测试直接驱动），丢弃收招帧。
func (f *skillFixture) finish() {
	f.pl.combat.mu.Lock()
	seq, locked := f.pl.combat.castSeq, f.pl.combat.locked
	f.pl.combat.mu.Unlock()
	f.s.finishCast(f.pl, seq, locked)
	f.take()
}

// combatStateText 会话态的可打印摘要（结构体含锁，不能整体 %+v）。
func (f *skillFixture) combatStateText() string {
	st := &f.pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	return fmt.Sprintf("cur=%q rows=%d mp=%d qgp=%d staged=%t locked=%t",
		st.curSkill, len(st.cooldowns), st.mp, st.qinggong, st.staged != nil, st.locked)
}

// frameTokens 断言串归一：分隔符与空白折叠为单空格——期望串里可用 " | "
// 按帧分行书写，比对只看 token 序列。
func frameTokens(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, " | ", " ")), " ")
}

// assertFrames 整批帧与期望串比对。
func (f *skillFixture) assertFrames(t *testing.T, frames [][]byte, want string) {
	t.Helper()
	got := strings.Join(describeFrames(t, frames, f.origin), " | ")
	if frameTokens(got) != frameTokens(want) {
		t.Errorf("帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

// assertPrefix 只比对开头一段（尾部含未钉死的动作名等）。
func (f *skillFixture) assertPrefix(t *testing.T, frames [][]byte, want string) {
	t.Helper()
	got := strings.Join(describeFrames(t, frames, f.origin), " | ")
	gotTokens, wantTokens := strings.Fields(frameTokens(got)), strings.Fields(frameTokens(want))
	if len(gotTokens) < len(wantTokens) || strings.Join(gotTokens[:len(wantTokens)], " ") != strings.Join(wantTokens, " ") {
		t.Errorf("帧序前缀不符:\n got: %s\nwant: %s", got, want)
	}
}

// castMessage 构造 m211 值表。
func castMessage(skill string, x, y, z, orient float32, target *uint64, tx, ty, tz float32) entergen.CustomInbound {
	values := []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomUseSkill}, {Type: 6, Text: skill},
		{Type: 4, Float32: x}, {Type: 4, Float32: y}, {Type: 4, Float32: z}, {Type: 4, Float32: orient},
		{Type: 4}, {Type: 4}, {Type: 4}, {Type: 6},
	}
	if target != nil {
		var raw [8]byte
		binary.LittleEndian.PutUint32(raw[:], uint32(*target))
		binary.LittleEndian.PutUint32(raw[4:], uint32(*target>>32))
		values = append(values, entergen.CustomInboundValue{Type: 8, Raw: raw},
			entergen.CustomInboundValue{Type: 4, Float32: tx},
			entergen.CustomInboundValue{Type: 4, Float32: ty},
			entergen.CustomInboundValue{Type: 4, Float32: tz})
	}
	return entergen.CustomInbound{Opcode: 0x0A, Values: values}
}

// moveMessage 构造 m212/m220 值表（[号, x, y, z, 朝向, 可选尾]）。
func moveMessage(id int32, x, y, z, orient float32, tail *int32) entergen.CustomInbound {
	values := []entergen.CustomInboundValue{
		{Type: 2, Int32: id},
		{Type: 4, Float32: x}, {Type: 4, Float32: y}, {Type: 4, Float32: z}, {Type: 4, Float32: orient},
	}
	if tail != nil {
		values = append(values, entergen.CustomInboundValue{Type: 2, Int32: *tail})
	}
	return entergen.CustomInbound{Opcode: 0x0A, Values: values}
}

// 常用对象键：目标 obj=5 own=0x10CD。
var (
	testTarget  = uint64(5) | uint64(0x10CD)<<32
	testTargetA = uint32(5)
	testTargetO = uint32(0x10CD)
)

const (
	testTargetText = "o:5/4301"
	areaSkill      = "CS_jh_cqgf04" // 自身范围技：射程 5、CD 3000、耗 MP 3、套路 CS_jh_cqgf
	areaSkill2     = "CS_jh_cqgf06" // 同套路：CD 2000、耗 MP 3
	otherSetSkill  = "wx_sl_tzcq02" // 别的套路 wx_sl_tzcq：CD 3000、耗 MP 10、公共 3000
	rushSkill      = "CS_jh_cqgf01" // 锁定冲锋技：射程 3、CD 6000、耗 MP 3 + 轻功 10
	basicAttack    = "zs_normal_00" // 普攻：扇形射程 2、CD 1200、组内公共 2000
	casterLoc      = "loc(100.000,5.000,200.000,1.000)"
	casterSync     = "m337(o:1/1,f:100.000,f:5.000,f:200.000,f:1.000,i:0)"
	aimLoc         = "loc(100.000,5.000,200.000,1.571)"
	aimSync        = "m337(o:1/1,f:100.000,f:5.000,f:200.000,f:1.571,i:0)"
)

// describeFrames 下行帧 → 可读串；冷却表起止折算为相对基准时刻的毫秒偏移。
func describeFrames(t *testing.T, frames [][]byte, origin time.Time) []string {
	t.Helper()
	base := filetimeMS(origin)
	le16 := func(b []byte) int { return int(binary.LittleEndian.Uint16(b)) }
	le32 := func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
	f32 := func(b []byte) float32 { return math.Float32frombits(le32(b)) }
	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		if len(frame) == 0 {
			out = append(out, "<空帧>")
			continue
		}
		switch frame[0] {
		case 0x10:
			for i, offset := 0, 12; i < le16(frame[10:]); i++ {
				entry, _ := clientdata.PropertyByIndex(le16(frame[offset:]))
				offset += 2
				var value string
				switch entry.Type {
				case clientdata.WireByte:
					value = fmt.Sprint(frame[offset])
					offset++
				case clientdata.WireInt32:
					value = fmt.Sprint(int32(le32(frame[offset:])))
					offset += 4
				case clientdata.WireFloat32:
					value = fmt.Sprintf("%.3f", f32(frame[offset:]))
					offset += 4
				case clientdata.WireString:
					length := int(le32(frame[offset:]))
					value = string(frame[offset+4 : offset+4+length-1])
					offset += 4 + length
				case clientdata.WireObject:
					value = fmt.Sprintf("o:%d/%d", le32(frame[offset:]), le32(frame[offset+4:]))
					offset += 8
				default:
					t.Fatalf("未覆盖的属性线型 %v（%s）", entry.Type, entry.Name)
				}
				out = append(out, entry.Name+"="+value)
			}
		case 0x1E:
			message, err := entergen.ParseCustomMessage(frame)
			if err != nil || len(message.Values) == 0 {
				t.Fatalf("自定义帧解码失败: %v", err)
			}
			parts := make([]string, 0, len(message.Values))
			for _, value := range message.Values[1:] {
				switch value.Type {
				case 2:
					parts = append(parts, fmt.Sprintf("i:%d", value.Int32))
				case 4:
					parts = append(parts, fmt.Sprintf("f:%.3f", value.Float32))
				case 6:
					parts = append(parts, "s:"+value.Text)
				case 8:
					parts = append(parts, fmt.Sprintf("o:%d/%d",
						binary.LittleEndian.Uint32(value.Raw[:]), binary.LittleEndian.Uint32(value.Raw[4:])))
				default:
					parts = append(parts, fmt.Sprintf("tag%d", value.Type))
				}
			}
			out = append(out, fmt.Sprintf("m%d(%s)", message.Values[0].Int32, strings.Join(parts, ",")))
		case 0x11:
			out = append(out, fmt.Sprintf("cd+行%d", le16(frame[12:])))
		case 0x12:
			out = append(out, fmt.Sprintf("cd-行%d", le16(frame[12:])))
		case 0x13:
			row, column := le16(frame[14:]), frame[16]
			var value string
			switch column {
			case 0:
				value = fmt.Sprint(le32(frame[17:]))
			case 3:
				value = fmt.Sprint(le16(frame[17:]))
			default:
				value = fmt.Sprintf("%+dms", int64(binary.LittleEndian.Uint64(frame[17:]))-base)
			}
			out = append(out, fmt.Sprintf("cd%d.%d=%s", row, column, value))
		case 0x1F:
			out = append(out, fmt.Sprintf("loc(%.3f,%.3f,%.3f,%.3f)", f32(frame[9:]), f32(frame[13:]), f32(frame[17:]), f32(frame[21:])))
		default:
			out = append(out, fmt.Sprintf("op%02X", frame[0]))
		}
	}
	return out
}

// 冷却三行（个人行 → 窗口行 → 公共行）的可读形；%s = 行号，时间另拼。
func cdInsert(row, category, start, end, team string) string {
	return "cd+行" + row + " | cd" + row + ".0=" + category + " | cd" + row + ".1=" + start +
		" | cd" + row + ".2=" + end + " | cd" + row + ".3=" + team
}

const (
	catWindow = "1"
	teamNone  = "65535"
)

// 自身范围技首放：帧序 = 抓包 209.389 簇（CurSkillID → 冷却三行 → MP → 0x1F
// → m337 → m20）。新号 CurSkillTarget 本就是 None，不发。
func TestSkillCastAreaSkillFirstCast(t *testing.T) {
	f := newSkillFixture(t)
	f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1, nil, 0, 0, 0))
	f.assertFrames(t, f.take(),
		"CurSkillID="+areaSkill+" | "+cdInsert("0", "5830", "+0ms", "+3000ms", teamNone)+" | "+
			cdInsert("1", catWindow, "+0ms", "+1000ms", teamNone)+" | "+
			cdInsert("2", "50000", "+0ms", "+7000ms", "50124")+" | MPRatio=94 | MP=52 | "+casterLoc+" | "+
			casterSync+" | m20(o:1/1,s:"+areaSkill+",o:0/0,i:0,f:0.000)")
	if state := f.combatStateText(); state != `cur="CS_jh_cqgf04" rows=3 mp=52 qgp=150 staged=false locked=false` {
		t.Errorf("受理后会话态不符: %s", state)
	}
}

// 普攻带目标：三行（个人 2012 / 窗口 / 组内公共 50001）、无 MP 帧、朝向指向
// 目标、m20 取流程包连招动作；冷却中零应答，个人行过期被清扫（单发删除，
// 抓包 397.170 同形）而组内公共行仍在时依旧静默（抓包 397.08/397.45）。
func TestSkillCastBasicAttackTargetsAndCools(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	cast := func() { f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1, &target, 101, 5, 200)) }
	action := func(step string) string {
		return aimLoc + " | " + aimSync + " | m20(o:1/1,s:zs_normal_00_" + step + "," + testTargetText + ",i:0,f:0.000)"
	}
	cast()
	f.assertFrames(t, f.take(),
		"CurSkillID="+basicAttack+" | CurSkillTarget="+testTargetText+" | "+
			cdInsert("0", "2012", "+0ms", "+1200ms", teamNone)+" | "+
			cdInsert("1", catWindow, "+0ms", "+1000ms", teamNone)+" | "+
			cdInsert("2", "50001", "+0ms", "+2000ms", teamNone)+" | "+action("01"))

	f.finish()
	f.advance(500 * time.Millisecond)
	cast()
	f.assertFrames(t, f.take(), "") // 个人冷却 1200ms 内

	f.advance(800 * time.Millisecond)
	cast()
	f.assertFrames(t, f.take(), "cd-行0") // 个人行过期被清扫；公共行 2000ms 仍在 → 静默

	f.advance(800 * time.Millisecond)
	cast()
	f.assertFrames(t, f.take(),
		"cd-行1 | CurSkillID="+basicAttack+" | "+cdInsert("1", "2012", "+2100ms", "+3300ms", teamNone)+" | "+
			"cd0.1=+2100ms | cd0.2=+3100ms | "+cdInsert("2", "50001", "+2100ms", "+4100ms", teamNone)+" | "+action("02"))
}

// 套路规则（用户定案 + 抓包 384~388s）：同套路只按招式自身冷却，公共行只拦
// 换套路；同套路受理仍刷新公共行（原地改写起止）。普攻是另一套机制
// （CoolEffectType=2 组内公共），不受套路公共行约束（抓包 216.188/390.880）。
func TestSkillCooldownFollowsTaoluRule(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	cast := func(skill string, withTarget bool) {
		var aim *uint64
		if withTarget {
			aim = &target
		}
		f.s.handleSkillCast(f.pl, castMessage(skill, 100, 5, 200, 1, aim, 101, 5, 200))
	}
	cast(areaSkill2, false)
	f.take()
	f.finish()

	// +2.5s 同套路另一招：受理。本技能类别无过期行可清（5682 是别的招，
	// 官服同样不清）；公共行未过期 → 原地改写起止。
	f.advance(2500 * time.Millisecond)
	cast(areaSkill, false)
	f.assertFrames(t, f.take(),
		"CurSkillID="+areaSkill+" | "+cdInsert("3", "5830", "+2500ms", "+5500ms", teamNone)+" | "+
			"cd1.1=+2500ms | cd1.2=+3500ms | cd2.1=+2500ms | cd2.2=+9500ms | MPRatio=89 | MP=49 | "+
			casterLoc+" | "+casterSync+" | m20(o:1/1,s:"+areaSkill+",o:0/0,i:0,f:0.000)")
	f.finish()

	// +0.5s 换套路（CoolEffectType=1）：套路公共行未过期 → 零应答。
	f.advance(500 * time.Millisecond)
	cast(otherSetSkill, false)
	f.assertFrames(t, f.take(), "")

	// 同一时刻普攻（CoolEffectType=2）：不受套路公共行约束 → 受理。
	cast(basicAttack, true)
	f.assertFrames(t, f.take(),
		"CurSkillID="+basicAttack+" | CurSkillTarget="+testTargetText+" | "+
			cdInsert("4", "2012", "+3000ms", "+4200ms", teamNone)+" | cd1.1=+3000ms | cd1.2=+4000ms | "+
			cdInsert("5", "50001", "+3000ms", "+5000ms", teamNone)+" | "+aimLoc+" | "+aimSync+
			" | m20(o:1/1,s:zs_normal_00_01,"+testTargetText+",i:0,f:0.000)")
	f.finish()

	// +6.6s：套路公共行（9.5s 止）已过期 → 换套路放行；过期公共行被清扫，
	// 新公共行带本技能的分组。上一个技能带过目标，此处补发
	// CurSkillTarget=None。
	f.advance(6600 * time.Millisecond)
	cast(otherSetSkill, false)
	f.assertPrefix(t, f.take(),
		"cd-行2 | CurSkillID="+otherSetSkill+" | CurSkillTarget=o:0/0 | "+
			cdInsert("5", "4060", "+9600ms", "+12600ms", teamNone)+" | "+
			"cd1.1=+9600ms | cd1.2=+10600ms | "+cdInsert("6", "50000", "+9600ms", "+12600ms", "50185")+
			" | MPRatio=70 | MP=39")
}

// 超距：回 m24 目标 80 + 目标位 + 朝向（抓包 205.644）。目标不在 NPC 目录
// 时用 m211 自报位。
func TestSkillCastOutOfRangeRejects(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.25, &target, 105, 5.2, 200))
	f.assertFrames(t, f.take(), "m24("+testTargetText+",i:80,f:105.000,f:5.200,f:200.000,f:1.250)")
	if state := f.combatStateText(); state != `cur="" rows=0 mp=55 qgp=150 staged=false locked=false` {
		t.Errorf("拒绝不应留下技能态: %s", state)
	}
}

// 进行中再按 → 9033；需目标而无目标 → 9030；MP 不足 → 9021。
func TestSkillCastRejections(t *testing.T) {
	f := newSkillFixture(t)
	cast := func() { f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1, nil, 0, 0, 0)) }
	cast()
	f.take()
	cast()
	f.assertFrames(t, f.take(), "m707(i:9033)")
	f.finish()

	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1, nil, 0, 0, 0))
	f.assertFrames(t, f.take(), "m707(i:9030)")

	f.advance(3500 * time.Millisecond) // 过个人冷却（3000ms）
	f.pl.combat.mu.Lock()
	f.pl.combat.mp = 1
	f.pl.combat.mu.Unlock()
	cast()
	// 被拒的按键同样先清扫过期行（个人行 3000ms 已过期），再回 MP 不足。
	f.assertFrames(t, f.take(), "cd-行0 | m707(i:9021)")
}

// 选靶回落：需目标的技能在 m211 不带目标时用最近一次 msg431 的目标。
func TestSkillCastFallsBackToSelectedTarget(t *testing.T) {
	f := newSkillFixture(t)
	var raw [8]byte
	binary.LittleEndian.PutUint32(raw[:], testTargetA)
	binary.LittleEndian.PutUint32(raw[4:], testTargetO)
	f.s.handleNPCSelectTarget(f.pl, entergen.CustomInbound{Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomSelectTarget}, {Type: 8, Raw: raw},
	}})
	f.take()
	// 目标不在 NPC 目录且 m211 无自报位 → 目标位回落 (0,0,0) → 超距拒绝，
	// m24 携带回落到的选靶对象。
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1, nil, 0, 0, 0))
	f.assertFrames(t, f.take(), "m24("+testTargetText+",i:80,f:0.000,f:0.000,f:0.000,f:1.000)")
}

// 冲锋技全链：暂存 → 进带授予 c61 → m212 落地释放 → 收招清位。
func TestSkillRushStagedGrantLanding(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	const (
		casterX, targetX = 100.0, 106.0 // 相距 6m：超直收带（3.5）、在冲锋带（6.5）内
		approachX        = 104.5        // 进带上报位
	)
	// 暂存（抓包 206.558）：冷却/MP 此刻不结。
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, casterX, 5, 200, 1, &target, targetX, 5, 200))
	f.assertFrames(t, f.take(),
		"CurSkillID="+rushSkill+" | CurSkillTarget="+testTargetText+" | m25(i:30000,s:skill_pause) | "+
			"PauseSkill="+rushSkill+" | m707(i:15)")
	if state := f.combatStateText(); state != `cur="CS_jh_cqgf01" rows=0 mp=55 qgp=150 staged=true locked=false` {
		t.Errorf("暂存不应结算冷却/MP: %s", state)
	}

	// 授予（抓包 207.415）：落点 = 目标身前 1.5m、y+1；此刻结冷却/MP/轻功。
	f.s.handleRushApproach(f.pl, moveMessage(220, approachX, 5, 200, 0, nil))
	f.assertFrames(t, f.take(),
		"m24(o:1/1,i:61,f:104.500,f:6.000,f:200.000,i:1,s:qg_fastrun_0h_01) | CantMove=1 | CantParry=1 | "+
			cdInsert("0", "5828", "+0ms", "+6000ms", teamNone)+" | "+
			cdInsert("1", catWindow, "+0ms", "+1000ms", teamNone)+" | "+
			cdInsert("2", "50000", "+0ms", "+7000ms", "50124")+" | MPRatio=94 | MP=52 | QingGongPoint=140")
	if state := f.combatStateText(); state != `cur="CS_jh_cqgf01" rows=3 mp=52 qgp=140 staged=true locked=true` {
		t.Errorf("授予后会话态不符: %s", state)
	}

	// 落地（抓包 207.777）：锚 = m212 自报位；m19 打击点沿攻击方向。
	tail := int32(1)
	f.s.handleDashLanding(f.pl, moveMessage(212, approachX, 5, 200, 0, &tail))
	landing := "loc(104.500,5.000,200.000,1.571)"
	f.assertFrames(t, f.take(),
		"CantMove=0 | CantParry=0 | "+landing+" | CurBreathState=0 | PauseSkill= | m26() | "+landing+" | "+
			"m337(o:1/1,f:104.500,f:5.000,f:200.000,f:1.571,i:0) | LastWGHit=100.000 | "+
			"m24("+testTargetText+",i:40,i:1,o:1/1) | "+
			"m19(o:1/1,"+testTargetText+",s:"+rushSkill+",i:1500,i:1500,f:105.370,f:5.000,f:200.000,f:1.571,"+
			"f:0.870,f:106.900,f:5.000,f:200.000,f:2.617,f:0.900,i:0,f:0.000,i:1,i:1,f:0.000,"+
			"f:106.000,f:5.000,f:200.000) | CantMove=1 | CantParry=1 | "+landing)
	if state := f.combatStateText(); state != `cur="CS_jh_cqgf01" rows=3 mp=52 qgp=140 staged=false locked=true` {
		t.Errorf("落地后应保持出招锁到收招: %s", state)
	}

	// 收招（锁定包 LockSrcTime）：清授予对 + CurSkillID。
	f.pl.combat.mu.Lock()
	seq, locked := f.pl.combat.castSeq, f.pl.combat.locked
	f.pl.combat.mu.Unlock()
	f.s.finishCast(f.pl, seq, locked)
	f.assertFrames(t, f.take(), "CantMove=0 | CantParry=0 | CurSkillID=")

	// 无暂存的 m212：仅位置收编，零帧（官服零回应）。
	f.s.handleDashLanding(f.pl, moveMessage(212, 104, 5, 200, 0, &tail))
	f.assertFrames(t, f.take(), "")
}

// 暂存中被新按键覆盖：三清后按新技能受理（2433 参考 rearm 同序）。暂存
// 期已下发过目标，换放无目标技能按抓包 198.879 语义补发 CurSkillTarget=None。
func TestSkillRushRearmByNewCast(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, 100, 5, 200, 1, &target, 106, 5, 200))
	f.take()
	f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1, nil, 0, 0, 0))
	f.assertFrames(t, f.take(),
		"CurSkillID= | m26() | PauseSkill= | CurSkillID="+areaSkill+" | CurSkillTarget=o:0/0 | "+
			cdInsert("0", "5830", "+0ms", "+3000ms", teamNone)+" | "+
			cdInsert("1", catWindow, "+0ms", "+1000ms", teamNone)+" | "+
			cdInsert("2", "50000", "+0ms", "+7000ms", "50124")+" | MPRatio=94 | MP=52 | "+
			casterLoc+" | "+casterSync+" | m20(o:1/1,s:"+areaSkill+",o:0/0,i:0,f:0.000)")
	if state := f.combatStateText(); !strings.Contains(state, "staged=false") {
		t.Errorf("换技后暂存态应清除: %s", state)
	}
}

// 暂存窗到期未被授予：三清收尾（直接驱动到期回调）。
func TestSkillRushExpiryClears(t *testing.T) {
	f := newSkillFixture(t)
	target := testTarget
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, 100, 5, 200, 1, &target, 106, 5, 200))
	f.take()
	f.pl.combat.mu.Lock()
	seq := f.pl.combat.staged.seq
	f.pl.combat.mu.Unlock()
	f.s.expireStaged(f.pl, seq)
	f.assertFrames(t, f.take(), "CurSkillID= | m26() | PauseSkill=")
	if state := f.combatStateText(); state != `cur="" rows=0 mp=55 qgp=150 staged=false locked=false` {
		t.Errorf("到期后应无暂存/无进行中技能: %s", state)
	}
}

// MP 回复：每秒 ceil(上限/10)/2（55 → 3），变化即推 MP 帧，回满即停。
func TestCombatTickRegeneratesMP(t *testing.T) {
	f := newSkillFixture(t)
	f.pl.combat.mu.Lock()
	f.pl.combat.mp, f.pl.combat.mpReady = 40, true
	f.pl.combat.mu.Unlock()
	var got []string
	for i := 0; i < 7; i++ {
		f.s.combatTick()
		got = append(got, frameTokens(strings.Join(describeFrames(t, f.take(), f.origin), " | ")))
	}
	want := []string{"MP=43", "MP=46", "MP=49", "MP=52", "MP=55", "", ""}
	if strings.Join(got, ";") != strings.Join(want, ";") {
		t.Errorf("MP 回复帧 = %v, want %v", got, want)
	}
}
