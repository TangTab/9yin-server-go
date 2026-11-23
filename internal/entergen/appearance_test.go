package entergen

// 0x0D 玩家物化帧测试：布局（63B 头 + u16 属性数）、身体渲染模型集
// 定版（布衣三件/Hat=Hair/Photo·Face·Hair 取档）、按 2433 表全量
// 解码消费到帧尾（与 captures/local 52626 #25 同一解析口径）。

import (
	"encoding/binary"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// decodeAppearanceProps 按 2433 表线型走帧属性区（对拍工具同口径），
// 返回 名→串值 map（重复键后写覆盖——真机帧 State/Parry 族尾段重复）。
func decodeAppearanceProps(t *testing.T, frame []byte) map[string]string {
	t.Helper()
	props := make(map[string]string)
	n := int(binary.LittleEndian.Uint16(frame[61:]))
	off := 63
	for i := 0; i < n; i++ {
		idx := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		entry, ok := clientdata.PlayerProperty(appearanceTestName(idx))
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
			val = strconv.FormatFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(frame[off:]))), 'g', -1, 32)
			off += 4
		case 6:
			val = "d"
			off += 8
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
			val = string(utf16Decode(units))
			off += ln
		case 9:
			val = "obj"
			off += 8
		default:
			t.Fatalf("prop %d unknown type %d", idx, entry.TypeCode)
		}
		props[entry.Name] = val
	}
	if off != len(frame) {
		t.Fatalf("consumed %d/%d", off, len(frame))
	}
	return props
}

func utf16Decode(units []uint16) []rune {
	out := make([]rune, 0, len(units))
	for i := 0; i < len(units); i++ {
		if units[i] >= 0xD800 && units[i] < 0xDC00 && i+1 < len(units) &&
			units[i+1] >= 0xDC00 && units[i+1] < 0xE000 {
			out = append(out, rune(units[i]-0xD800)<<10|rune(units[i+1]-0xDC00)+0x10000)
			i++
		} else {
			out = append(out, rune(units[i]))
		}
	}
	for len(out) > 0 && out[len(out)-1] == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// appearanceTestName 测试内索引→名（全表扫描足够小）。
func appearanceTestName(idx uint16) string {
	for _, entry := range clientdata.OfficialPropertyEntries() {
		if entry.Index == idx {
			return entry.Name
		}
	}
	return "?"
}

// TestAppearanceFrameLayout 63B 头 + 布衣三件 + Hat=Hair + 全量解码消费。
func TestAppearanceFrameLayout(t *testing.T) {
	ctx := testContext(t)
	// 建角自定义槽位：[0]=book [1]=photo [2]=face [4]=hair（定版）。
	ctx.State.Appearance = []string{"book2", `icon\players\boy01_create.png`,
		"e/330$33,'51\x1a4>=3333333333?5=<==\x19=33!F!1'6F3\t\x01", "", `obj\char\b_hair\b_hair3`}
	frame, err := AppearanceFrame(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x0D {
		t.Fatalf("op = %#x", frame[0])
	}
	if binary.LittleEndian.Uint32(frame[1:]) != 1 || binary.LittleEndian.Uint32(frame[5:]) != 1 {
		t.Fatalf("obj/own: %x", frame[1:9])
	}
	pos := ctx.State.Position
	for i, want := range []float64{pos[0], pos[1], pos[2], pos[3]} {
		got := math.Float32frombits(binary.LittleEndian.Uint32(frame[9+i*4:]))
		if math.Abs(float64(got)-want) > 0.01 {
			t.Fatalf("pos[%d] = %v want %v", i, got, want)
		}
	}
	for i := 25; i < 61; i++ {
		if frame[i] != 0 {
			t.Fatalf("reserved byte %d nonzero: %#x", i, frame[i])
		}
	}

	props := decodeAppearanceProps(t, frame)

	// 身体渲染模型集（2026-11-23 官服勘误：无装备=jianghu000 裸身族，
	// Hat=Hair 回填；穿装见 TestEquipSlotModel）。
	if props["Cloth"] != `obj\char\b_jianghu000\b_cloth000` {
		t.Fatalf("Cloth = %q", props["Cloth"])
	}
	if props["Pants"] != `obj\char\b_jianghu000\b_pants000` {
		t.Fatalf("Pants = %q", props["Pants"])
	}
	if props["Shoes"] != `obj\char\b_jianghu000\b_shoes000` {
		t.Fatalf("Shoes = %q", props["Shoes"])
	}
	if props["PhyDef"] != "0" {
		t.Fatalf("PhyDef = %q want 0（无装备 0 基线）", props["PhyDef"])
	}
	if props["Hat"] != props["Hair"] {
		t.Fatalf("Hat %q != Hair %q", props["Hat"], props["Hair"])
	}
	if props["Hair"] != `obj\char\b_hair\b_hair3` { // 档内外观槽[4]覆盖默认
		t.Fatalf("Hair = %q", props["Hair"])
	}
	if props["Photo"] != "icon\\players\\boy01_create.png" { // 档内外观槽[1]覆盖默认
		t.Fatalf("Photo = %q", props["Photo"])
	}
	if props["Face"] != "e/330$33,'51\x1a4>=3333333333?5=<==\x19=33!F!1'6F3\t\x01" {
		t.Fatalf("Face = %q", props["Face"])
	}
	if props["State"] != "stand" || props["ActionSet"] != "0h" {
		t.Fatalf("State/ActionSet = %q/%q", props["State"], props["ActionSet"])
	}
	if props["Name"] != ctx.RoleName {
		t.Fatalf("Name = %q", props["Name"])
	}
	if props["CurNeiGong"] != ctx.State.CurNeiGong {
		t.Fatalf("CurNeiGong = %q", props["CurNeiGong"])
	}
}

// TestAppearanceFrameFemale 女号：镜像 g_ 裸身族 + Sex=1。
func TestAppearanceFrameFemale(t *testing.T) {
	ctx := testContext(t)
	ctx.State.Sex = 2
	ctx.State.Appearance = nil
	frame, err := AppearanceFrame(ctx)
	if err != nil {
		t.Fatal(err)
	}
	props := decodeAppearanceProps(t, frame)
	if props["Sex"] != "1" {
		t.Fatalf("Sex = %q", props["Sex"])
	}
	if props["Cloth"] != `obj\char\g_jianghu000\g_cloth000` ||
		props["Pants"] != `obj\char\g_jianghu000\g_pants000` ||
		props["Shoes"] != `obj\char\g_jianghu000\g_shoes000` {
		t.Fatalf("female cloth set: %q %q %q", props["Cloth"], props["Pants"], props["Shoes"])
	}
	if props["Hair"] != appearanceHairG || props["Hat"] != appearanceHairG {
		t.Fatalf("female hair/hat: %q %q", props["Hair"], props["Hat"])
	}
}

// TestPreReadyFramesContainsAppearance 0x0B 之后紧跟 0x0D（52626 实证序）。
func TestPreReadyFramesContainsAppearance(t *testing.T) {
	frames, err := PreReadyFrames(testContext(t), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) < 2 || frames[0][0] != 0x0B || frames[1][0] != 0x0D {
		t.Fatalf("enter prefix = %#x %#x", frames[0][0], frames[1][0])
	}
}
