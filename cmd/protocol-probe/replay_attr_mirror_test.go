package main

// ATTR-SHADOW 回归闸（2026-11-15 真值优先/影子兜底）：
//  1. 方言零漂移：内嵌 Phase2 全部 self 0x10 帧在空 override 下
//     解析→重编码必须与原帧逐字节一致（decoder↔encoder round-trip）。
//  2. 换值正确性：override 单 idx 后仅该行值变化、帧长不变。
//  3. 非 self 帧拒绝：obj 不匹配 → ok=false（运行时透传）。
//  4. 排除表：481/624 不入 override（占位常量保留影子）。

import (
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/cmd/protocol-probe/replaydata"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/sim"
)

func unpackEmbeddedFrames(t *testing.T, raw []byte) [][]byte {
	t.Helper()
	var frames [][]byte
	for len(raw) >= 4 {
		fl := int(binary.LittleEndian.Uint32(raw))
		if fl <= 0 || fl > len(raw)-4 || fl > 0x10000 {
			t.Fatalf("bad frame len %d", fl)
		}
		frames = append(frames, raw[4:4+fl])
		raw = raw[4+fl:]
	}
	return frames
}

// selfAttrObjFromPhase2 统计 self 0x10 帧的 obj（出现频次最高者）。
func selfAttrObjFromPhase2(t *testing.T) (uint32, [][]byte) {
	t.Helper()
	frames := unpackEmbeddedFrames(t, replaydata.Phase2)
	counts := map[uint32]int{}
	for _, f := range frames {
		if len(f) >= 12 && f[0] == 0x10 && f[1] == 0 {
			counts[binary.LittleEndian.Uint32(f[2:])]++
		}
	}
	if len(counts) == 0 {
		t.Fatal("Phase2 无 0x10 帧")
	}
	best, bestN := uint32(0), 0
	for obj, n := range counts {
		if n > bestN {
			best, bestN = obj, n
		}
	}
	var self [][]byte
	for _, f := range frames {
		if len(f) >= 12 && f[0] == 0x10 && f[1] == 0 && binary.LittleEndian.Uint32(f[2:]) == best {
			self = append(self, f)
		}
	}
	t.Logf("self obj=%#x，self 0x10 帧 %d/%d", best, len(self), len(frames))
	return best, self
}

// TestReplayAttrShadowRoundTrip 方言零漂移闸：全部 self 0x10 帧
// 空覆盖重编码必须逐字节还原。
func TestReplayAttrShadowRoundTrip(t *testing.T) {
	obj, self := selfAttrObjFromPhase2(t)
	failed := 0
	for i, frame := range self {
		out, swapped, _, ok := replayAttrShadowSwap(frame, obj, map[uint16]clientdata.IndexedProperty{})
		if !ok {
			t.Errorf("self 0x10 #%d 解析失败（len=%d）", i, len(frame))
			failed++
			continue
		}
		if swapped != 0 {
			t.Errorf("self 0x10 #%d 空 override 却报 swapped=%d", i, swapped)
		}
		if string(out) != string(frame) {
			t.Errorf("self 0x10 #%d round-trip 字节漂移:\n orig=%x\n  out=%x", i, frame, out)
			failed++
		}
	}
	if failed > 0 {
		t.Fatalf("%d/%d 帧 round-trip 失败", failed, len(self))
	}
}

// TestReplayAttrShadowSwapSingleRow 换值正确性：仅覆盖行变化。
func TestReplayAttrShadowSwapSingleRow(t *testing.T) {
	obj, self := selfAttrObjFromPhase2(t)
	// 找一个含 478 MaxParry 的 self 帧
	var target []byte
	for _, frame := range self {
		p := 12
		cnt := int(binary.LittleEndian.Uint16(frame[10:]))
		for i := 0; i < cnt && p+2 <= len(frame); i++ {
			idx := binary.LittleEndian.Uint16(frame[p:])
			if idx == 478 {
				target = frame
			}
			entry, ok := clientdata.PropertyByIndex(int(idx))
			if !ok {
				break
			}
			_, used, valid := decodeAttrShadowValue(entry.Type, frame[p+2:])
			if !valid {
				break
			}
			p += 2 + used
		}
		if target != nil {
			break
		}
	}
	if target == nil {
		t.Skip("Phase2 self 帧无 478 MaxParry 行")
	}
	overrides := map[uint16]clientdata.IndexedProperty{
		478: {Index: 478, Name: "MaxParry", Value: clientdata.Int32Value(98765)},
	}
	out, swapped, kept, ok := replayAttrShadowSwap(target, obj, overrides)
	if !ok {
		t.Fatal("swap 失败")
	}
	if swapped != 1 || kept != int(binary.LittleEndian.Uint16(target[10:]))-1 {
		t.Fatalf("swapped=%d kept=%d", swapped, kept)
	}
	if len(out) != len(target) {
		t.Fatalf("帧长漂移 %d → %d", len(target), len(out))
	}
	// 解析输出：478 行值=98765，其余行字节与原帧一致
	pOld, pNew := 12, 12
	for i := 0; i < int(binary.LittleEndian.Uint16(target[10:])); i++ {
		idx := binary.LittleEndian.Uint16(target[pOld:])
		entry, _ := clientdata.PropertyByIndex(int(idx))
		_, usedOld, _ := decodeAttrShadowValue(entry.Type, target[pOld+2:])
		_, usedNew, _ := decodeAttrShadowValue(entry.Type, out[pNew+2:])
		if usedOld != usedNew {
			t.Fatalf("行 %d 宽度漂移", idx)
		}
		if idx == 478 {
			if got := int32(binary.LittleEndian.Uint32(out[pNew+2:])); got != 98765 {
				t.Fatalf("478 行值=%d 期望 98765", got)
			}
		} else if string(target[pOld:pOld+2+usedOld]) != string(out[pNew:pNew+2+usedNew]) {
			t.Fatalf("非覆盖行 %d 字节漂移", idx)
		}
		pOld += 2 + usedOld
		pNew += 2 + usedNew
	}
}

// TestReplayAttrShadowRejectsNonSelf 非 self 帧 ok=false。
func TestReplayAttrShadowRejectsNonSelf(t *testing.T) {
	obj, self := selfAttrObjFromPhase2(t)
	if len(self) == 0 {
		t.Fatal("无 self 帧")
	}
	if out, swapped, _, ok := replayAttrShadowSwap(self[0], obj^1, nil); ok || swapped != 0 || &out[0] != &self[0][0] {
		t.Fatalf("非 self obj 应拒绝：ok=%v", ok)
	}
}

// newAttrShadowTestPlayer 最小合法玩家（vitals 非零防除零：stateProperties
// 的比值计算以 MaxHP/MaxMP 为分母）。
func newAttrShadowTestPlayer() *playerActor {
	player := &playerActor{}
	player.vitals = sim.Vitals{HP: 1000, BaseMaxHP: 1000, MaxHP: 1000, MP: 500, BaseMaxMP: 500, MaxMP: 500}
	return player
}

// TestReplayAttrOverridesExclusions 排除表：测试玩家也绝不产出 481/624。
func TestReplayAttrOverridesExclusions(t *testing.T) {
	overrides := replayAttrOverrides(newAttrShadowTestPlayer())
	for _, idx := range []uint16{481, 624} {
		if _, bad := overrides[idx]; bad {
			t.Errorf("override 含排除 idx %d", idx)
		}
	}
	if len(overrides) == 0 {
		t.Fatal("测试玩家 override 为空（值源异常？）")
	}
	t.Logf("测试玩家 override=%d 项", len(overrides))
}

// TestReplayAttrShadowCoverage 换值覆盖率对账：零值玩家 override 下统计
// self 0x10 帧的换值/影子行数，并列出影子兜底 idx（与盘点文档 B 节
// 107 idx 清单互证；新 idx 获得我方值源后从此清单销项）。
func TestReplayAttrShadowCoverage(t *testing.T) {
	obj, self := selfAttrObjFromPhase2(t)
	overrides := replayAttrOverrides(newAttrShadowTestPlayer())
	framesSwapped, rowsSwapped, rowsShadow := 0, 0, 0
	shadowIdx := map[uint16]int{}
	for _, frame := range self {
		cnt := int(binary.LittleEndian.Uint16(frame[10:]))
		for i := 0; i < cnt; i++ {
			// 行起点直接由 swap 结果推：逐行重扫（帧均为单行，行首=@12）
			idx := binary.LittleEndian.Uint16(frame[12:])
			if _, ok := overrides[idx]; !ok {
				shadowIdx[idx]++
			}
		}
		_, swapped, kept, ok := replayAttrShadowSwap(frame, obj, overrides)
		if !ok {
			t.Errorf("帧解析失败 len=%d", len(frame))
			continue
		}
		if swapped > 0 {
			framesSwapped++
		}
		rowsSwapped += swapped
		rowsShadow += kept
	}
	var shadowList []uint16
	for idx := range shadowIdx {
		shadowList = append(shadowList, idx)
	}
	for i := 0; i < len(shadowList); i++ {
		for j := i + 1; j < len(shadowList); j++ {
			if shadowList[j] < shadowList[i] {
				shadowList[i], shadowList[j] = shadowList[j], shadowList[i]
			}
		}
	}
	t.Logf("覆盖对账: self 帧=%d 换值帧=%d 换值行=%d 影子行=%d override=%d 影子idx=%d",
		len(self), framesSwapped, rowsSwapped, rowsShadow, len(overrides), len(shadowList))
	for _, idx := range shadowList {
		if entry, ok := clientdata.PropertyByIndex(int(idx)); ok {
			t.Logf("  影子兜底 %d %s", idx, entry.Name)
		} else {
			t.Logf("  影子兜底 %d (?)", idx)
		}
	}
	if rowsSwapped == 0 {
		t.Fatal("零值玩家覆盖率为 0（override 与 bin idx 无交集？）")
	}
}
