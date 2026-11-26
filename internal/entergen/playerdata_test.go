package entergen

// 0x10 人物数据大批测试：布局（12B 头 + u16 n）、2433 表全量解码消费
// 到帧尾、官方口径锚点（MaxSP 全批仅一条/恢复公式/五维面板含书加成/
// BufferInfo 24 槽）、PreReady 实证序 0x0B→0x0D→0x1F→0x10 大批。

import (
	"encoding/binary"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

func f32At(b []byte, off int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[off:]))
}

func rune16Decode(units []uint16) []rune {
	out := make([]rune, 0, len(units))
	for _, u := range units {
		if u == 0 {
			break
		}
		out = append(out, rune(u))
	}
	return out
}

// decodeVitalProps 按 2433 表线型走 0x10 帧属性区（propanalyze 同口径），
// 返回 名→值 map + 出现次数（MaxSP 单次声明断言用）。
func decodeVitalProps(t *testing.T, frame []byte) (map[string]string, map[string]int) {
	t.Helper()
	props := make(map[string]string)
	counts := make(map[string]int)
	n := int(binary.LittleEndian.Uint16(frame[10:]))
	off := 12
	for i := 0; i < n; i++ {
		idx := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("prop %d not in table", idx)
		}
		var val string
		switch entry.TypeCode {
		case 1:
			val = strconv.Itoa(int(frame[off]))
			off++
		case 2:
			val = strconv.Itoa(int(binary.LittleEndian.Uint16(frame[off:])))
			off += 2
		case 3:
			val = strconv.Itoa(int(int32(binary.LittleEndian.Uint32(frame[off:]))))
			off += 4
		case 4:
			val = strconv.Itoa(int(int64(binary.LittleEndian.Uint64(frame[off:]))))
			off += 8
		case 5:
			val = strconv.FormatFloat(float64(f32At(frame, off)), 'g', -1, 32)
			off += 4
		case 7:
			ln := int(binary.LittleEndian.Uint32(frame[off:]))
			off += 4
			val = string(frame[off : off+ln-1])
			off += ln
		case 8:
			ln := int(binary.LittleEndian.Uint32(frame[off:]))
			off += 4
			units := make([]uint16, ln/2)
			for j := range units {
				units[j] = binary.LittleEndian.Uint16(frame[off+j*2:])
			}
			val = string(rune16Decode(units))
			off += ln
		case 9:
			v := binary.LittleEndian.Uint64(frame[off:])
			val = "obj"
			off += 8
			_ = v
		default:
			t.Fatalf("prop %q unencodable type %d", entry.Name, entry.TypeCode)
		}
		props[entry.Name] = val
		counts[entry.Name]++
	}
	if off != len(frame) {
		t.Fatalf("consumed %d of %d bytes", off, len(frame))
	}
	return props, counts
}

func TestPlayerVitalBatchFrame(t *testing.T) {
	ctx := testContext(t)
	state := ctx.State
	frame, err := PlayerVitalBatchFrame(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("op = %#x", frame[0])
	}
	if binary.LittleEndian.Uint32(frame[2:]) != 1 || binary.LittleEndian.Uint32(frame[6:]) != 1 {
		t.Fatalf("obj/own: %x", frame[2:10])
	}

	props, counts := decodeVitalProps(t, frame)

	// 官方恢复公式：基础位 ceil(总/10)、Add 位 round(0.07·Sta)/round(0.2·Spi)。
	book := state.NeiGong[0]
	wantHPUp := sitcrossRecovery(int32(state.HP) + int32(book.MaxHPAdd))
	if props["HPUpSpeed"] != strconv.Itoa(int(wantHPUp)) {
		t.Fatalf("HPUpSpeed = %q want %d", props["HPUpSpeed"], wantHPUp)
	}
	wantSta := int32(state.Sta) + int32(book.StaAdd)
	if props["HPUpSpeedAdd"] != strconv.Itoa(int(staHundredthsAdd(wantSta))) {
		t.Fatalf("HPUpSpeedAdd = %q (sta=%d)", props["HPUpSpeedAdd"], wantSta)
	}
	wantSpi := int32(state.Spi) + int32(book.SpiAdd)
	if props["MPUpSpeedAdd"] != strconv.Itoa(int(spiTenthsAdd(wantSpi))) {
		t.Fatalf("MPUpSpeedAdd = %q (spi=%d)", props["MPUpSpeedAdd"], wantSpi)
	}

	// MaxSP 全量批一次声明（基础 80 + 运行内功档位加成：首本书 ng_wd_001 是
	// 武当一内，+10）；MaxSPAdd 不出帧。
	if want := strconv.Itoa(int(RageCap(ctx))); props["MaxSP"] != want || want != "90" {
		t.Fatalf("MaxSP = %q, want %s", props["MaxSP"], want)
	}
	if counts["MaxSP"] != 1 {
		t.Fatalf("MaxSP appears %d times", counts["MaxSP"])
	}
	if _, ok := props["MaxSPAdd"]; ok {
		t.Fatal("MaxSPAdd must not ship in login batch")
	}

	// 五维面板 = 白板 + 首本书 varprop。
	if props["Str"] != strconv.Itoa(state.Str+book.StrAdd) {
		t.Fatalf("Str = %q want %d", props["Str"], state.Str+book.StrAdd)
	}
	// 攻击面板 Add 列 = 对应维官方公式分量。
	if props["MeleePowerAdd"] != strconv.Itoa(state.Str+book.StrAdd) {
		t.Fatalf("MeleePowerAdd = %q", props["MeleePowerAdd"])
	}
	// 货币（CapitalType1 int64——碎银）。
	if props["CapitalType1"] != strconv.Itoa(int(state.Money)) {
		t.Fatalf("CapitalType1 = %q want %d", props["CapitalType1"], state.Money)
	}
	// 格挡族登录补位四锚点 + ParryValue 推导和。
	if props["ParryRatio"] != "100" || props["ParryUpSpeed"] != "4" ||
		props["CantParry"] != "0" || props["ParryType"] != "0" {
		t.Fatalf("parry anchors: %+v", props)
	}
	if props["ParryValue"] != strconv.Itoa(state.Parry+book.MaxParryAdd) {
		t.Fatalf("ParryValue = %q", props["ParryValue"])
	}
	if props["ModifySkillLockTime"] != "200" {
		t.Fatalf("ModifySkillLockTime = %q", props["ModifySkillLockTime"])
	}

	// BufferInfo 24 槽全下（1/2/4 带登录 buff CSV）。
	for slot := 1; slot <= 24; slot++ {
		name := "BufferInfo" + strconv.Itoa(slot)
		if _, ok := props[name]; !ok {
			t.Fatalf("%s absent", name)
		}
	}
	if props["BufferInfo1"] == "" || props["BufferInfo24"] != "" {
		t.Fatalf("BufferInfo CSV slots: %q / %q", props["BufferInfo1"], props["BufferInfo24"])
	}

	// 门派三空串 + 经脉 8 件套（jmActCount 官方初始 3）。
	if props["School"] != "" || props["Force"] != "" || props["NewSchool"] != "" {
		t.Fatalf("faction: %+v", props)
	}
	if props["jmActCount"] != "3" || props["ZhenQiDayValue"] != "500" {
		t.Fatalf("jingmai anchors: %+v", props)
	}

	// 体量对拍：52626 #27 实机 n=160；本批建号子集 n≥140（BufferInfo×24
	// + 面板全集）。过小说明属性集漂移。
	n := int(binary.LittleEndian.Uint16(frame[10:]))
	if n < 140 {
		t.Fatalf("property count %d too small (official #27 = 160)", n)
	}
}

// TestPreReadyVitalOrder 52626 实证序：0x0B→0x0D→0x1F→0x10 大批→
// （物理流/挂载/0x28 等）。
func TestPreReadyVitalOrder(t *testing.T) {
	frames, err := PreReadyFrames(testContext(t), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) < 4 {
		t.Fatal("short pre-ready sequence")
	}
	ops := []byte{frames[0][0], frames[1][0], frames[2][0], frames[3][0]}
	want := []byte{0x0B, 0x0D, 0x1F, 0x10}
	for i := range want {
		if ops[i] != want[i] {
			t.Fatalf("prefix ops = % x want % x", ops, want)
		}
	}
	// 0x1F 坐标与 0x0D 同源（LocationFrame 25B）。
	if len(frames[2]) != 25 {
		t.Fatalf("0x1F len = %d", len(frames[2]))
	}
	// 大批 n≥140（结构锚点见 TestPlayerVitalBatchFrame）。
	if n := binary.LittleEndian.Uint16(frames[3][10:]); n < 140 {
		t.Fatalf("vital batch n = %d", n)
	}
}

// TestSPIsSessionValue 当前怒气（SP）取 Context.SP（会话值），不取存档
// State.SP；上限 MaxSP 恒为运行内功的 RageCap，与两者都无关。0x0D / 人物数据
// 大批 / 基础属性批三处同源。此前 SP 跟着建号初值 80 下发，而服务端战斗态从 0
// 起——登录后客户端满怒、放怒招却回「战意不足」（2026-11-26 定案：初始 0）。
func TestSPIsSessionValue(t *testing.T) {
	for _, tc := range []struct {
		name             string
		archive, session int
	}{
		{name: "旧档存着建号初值 80，登录从 0 起", archive: 80, session: 0},
		{name: "新建号", archive: 0, session: 0},
		{name: "切场景重入带当前怒气", archive: 80, session: 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testContext(t)
			ctx.State.SP, ctx.SP = tc.archive, tc.session
			want := strconv.Itoa(tc.session)

			appearance, err := AppearanceFrame(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got := decodeAppearanceProps(t, appearance)["SP"]; got != want {
				t.Errorf("0x0D SP = %q, want %q", got, want)
			}
			for name, build := range map[string]func(Context) ([][]byte, error){
				"人物数据大批": func(c Context) ([][]byte, error) {
					frame, err := PlayerVitalBatchFrame(c)
					return [][]byte{frame}, err
				},
				"基础属性批": BasePropertyFrames,
			} {
				frames, err := build(ctx)
				if err != nil || len(frames) != 1 {
					t.Fatalf("%s: %d 帧, err=%v", name, len(frames), err)
				}
				props, _ := decodeVitalProps(t, frames[0])
				if props["SP"] != want {
					t.Errorf("%s SP = %q, want %q", name, props["SP"], want)
				}
				if props["MaxSP"] != strconv.Itoa(int(RageCap(ctx))) {
					t.Errorf("%s MaxSP = %q, want %d（上限与存档 SP 无关）", name, props["MaxSP"], RageCap(ctx))
				}
			}
		})
	}
}
