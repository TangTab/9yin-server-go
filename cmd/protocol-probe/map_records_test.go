package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// 官服 c.pcapng 0.128s 原始帧（tmp/pcap_analysis/pcaplib 重解码）：
// obj/owner 是官服对象号，本服帧差异仅在 2..10 字段；10 起的
// rec/起始行/行数/行数据必须逐字节一致。整串机器生成原样粘贴——
// 手工分段曾两次引入错位（行 6-8），勿再人工切分。
const officialOpenMapTail = "330000001800feff037a282a81e1f600000410fe0100000000000000000000eefe4c4052004e7b003d40002300002012041cc3f00080007d04000000be00000000000000000000ee3b1f8001289800005c782000085454010000000005000000e00701000000"

// rec=56 起始行0 行数1 | col0=366（scene22 全局索引）| col1="scene22"
//（u32 长含 NUL）| col2=1（迷雾已开）。
const officialMapMaskTail = "3800000001006e010000080000007363656e6532320001000000"

func hexBytes(t *testing.T, hex string) []byte {
	t.Helper()
	out := make([]byte, len(hex)/2)
	for i := 0; i+1 < len(hex); i += 2 {
		hi := unhexNibble(t, hex[i])
		lo := unhexNibble(t, hex[i+1])
		out[i/2] = hi<<4 | lo
	}
	return out
}

func unhexNibble(t *testing.T, c byte) byte {
	t.Helper()
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	t.Fatalf("bad hex nibble %q", c)
	return 0
}

// OpenMapRec 帧必须与官服 0.128s 抓包字节级同形（本服对象号段除外）。
func TestOpenMapRecFrameMatchesOfficial(t *testing.T) {
	frame := openMapRecFrame(officialOpenMapBits)
	if len(frame) != 112 { // 官服 len=112（16 头 + 24×4）
		t.Fatalf("frame length = %d, want 112", len(frame))
	}
	if got := frame[10:]; !bytes.Equal(got, hexBytes(t, officialOpenMapTail)) {
		t.Fatalf("OpenMapRec tail mismatch:\n got %x\nwant %s", got, officialOpenMapTail)
	}
	if rec := binary.LittleEndian.Uint16(frame[10:]); rec != recordOpenMap {
		t.Fatalf("record index = %d, want %d", rec, recordOpenMap)
	}
	if rows := binary.LittleEndian.Uint16(frame[14:]); rows != openMapRecRows {
		t.Fatalf("row count = %d, want %d", rows, openMapRecRows)
	}
}

// MapMaskRec 行（366, "scene22", 1）与官服帧逐字节一致。
func TestMapMaskRecFrameMatchesOfficial(t *testing.T) {
	frame := mustMapMaskFrame(366, "scene22")
	if len(frame) != 36 { // 官服 len=36
		t.Fatalf("frame length = %d, want 36", len(frame))
	}
	if got := frame[10:]; !bytes.Equal(got, hexBytes(t, officialMapMaskTail)) {
		t.Fatalf("MapMaskRec tail mismatch:\n got %x\nwant %s", got, officialMapMaskTail)
	}
}

// grantMapRecords 输出两帧（迷雾行在前、位图在后），位图叠加当前场景位。
// scenes.ini 资源树缺席时注入样本索引表（scene22→366 官服锚点）。
func TestGrantMapRecordsOrderAndOverlay(t *testing.T) {
	original := sceneGlobalIndexLookup
	sceneGlobalIndexLookup = func(resource string) (uint32, bool) {
		if resource == "scene22" {
			return 366, true
		}
		return 0, false
	}
	t.Cleanup(func() { sceneGlobalIndexLookup = original })

	rec := &frameRecorder{}
	if err := grantMapRecords(rec, "scene22"); err != nil {
		t.Fatalf("grantMapRecords: %v", err)
	}
	frames := rec.snapshot()
	if len(frames) != 2 {
		t.Fatalf("wrote %d frames, want 2", len(frames))
	}
	if frames[0][0] != 0x11 || frames[1][0] != 0x11 {
		t.Fatalf("frame ops = %d/%d, want 0x11/0x11", frames[0][0], frames[1][0])
	}
	if got := frames[0][10:]; !bytes.Equal(got, hexBytes(t, officialMapMaskTail)) {
		t.Fatalf("MapMaskRec frame tail mismatch:\n got %x", got)
	}
	// scene22 位（行11 bit14）在官服样本里已点亮，叠加后帧不变。
	if got := frames[1][10:]; !bytes.Equal(got, hexBytes(t, officialOpenMapTail)) {
		t.Fatalf("OpenMapRec frame tail mismatch after overlay:\n got %x", got)
	}
}

// 位图叠加：未点亮位点亮、已点亮位幂等，越界索引（≥768）安全忽略。
func TestOverlaySceneBit(t *testing.T) {
	bits := officialOpenMapBits
	overlaySceneBit(&bits, 0) // 行0 bit0：样本 0x7a03fffe 的 bit0 未点亮
	if bits[0] != 0x7a03ffff {
		t.Fatalf("overlay 0 -> row0 = %#x, want 0x7a03ffff", bits[0])
	}
	before := bits[0]
	overlaySceneBit(&bits, 5) // 行0 bit5：样本已含，幂等
	if bits[0] != before {
		t.Fatalf("overlay bit5 changed row0: %x -> %x", before, bits[0])
	}
	overlaySceneBit(&bits, 300) // 300 = 行9 bit12：样本行9=0x2300 无 bit12
	if bits[9] != 0x3300 {
		t.Fatalf("overlay 300 -> row9 = %#x, want 0x3300", bits[9])
	}
	overlaySceneBit(&bits, 768) // 越界（24 行×32）忽略
	overlaySceneBit(&bits, 99999)
}

// scenes.ini 解析：366=scene22 实测锚点 + 带后缀配置名截地形名 + 首现绑定。
func TestParseSceneIndexTable(t *testing.T) {
	table := parseSceneIndexTable("// 场景配置文件\r\n" +
		"[1]\r\nMember=1\r\nConfig=ini\\scene\\born04_QianDengZhen\r\n" +
		"[366]\r\nMember=14\r\nConfig=ini\\scene\\scene22\r\n" +
		"[367]\r\nConfig=ini/scene/city02_suzhou\r\n" +
		"[368]\r\nConfig=ini\\scene\\city02_chengdu\r\n" +
		"[bad]\r\nConfig=ini\\scene\\x\r\n")
	if got := table["scene22"]; got != 366 {
		t.Fatalf("scene22 index = %d, want 366", got)
	}
	if got := table["city02"]; got != 367 {
		t.Fatalf("city02 index = %d, want 367 (first occurrence)", got)
	}
	if got := table["born04"]; got != 1 {
		t.Fatalf("born04 index = %d, want 1", got)
	}
	if _, ok := table["x"]; ok {
		t.Fatalf("non-numeric section should be skipped")
	}
}

// 真实资源树冒烟（NINEYIN_SERVER_ROOT 未配置时跳过）：
// scenes.ini 全表解析 + scene22→366 锚点。
func TestSceneGlobalIndexSmoke(t *testing.T) {
	if index, ok := sceneGlobalIndex("scene22"); !ok {
		// 资源缺失时 loadSceneIndexTable 已打日志，索引表为空——跳过。
		t.Skipf("scenes.ini resource unavailable (NINEYIN_SERVER_ROOT unset)")
	} else if index != 366 {
		t.Fatalf("scene22 = %d, want 366", index)
	}
	for _, res := range []string{"city02", "city05", "scene14", "school01", "born03"} {
		if index, ok := sceneGlobalIndex(res); !ok {
			t.Errorf("resource %q missing from scenes.ini index table", res)
		} else if index >= 768 {
			t.Errorf("resource %q index %d exceeds OpenMapRec bitmap space (768)", res, index)
		}
	}
}
