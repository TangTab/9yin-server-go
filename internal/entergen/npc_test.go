package entergen

// npc_test.go m229 c130 族 golden 对拍：ready_golden.bin 帧 #2-63（62 条）
// 全量解码 → NpcEntry → 重编码逐字节还原。62 条构成：26×公会旗帜 +
// 9×复活特效 + 27×场景建筑（docs/official/外服NPCm229物化方言定版.md）。

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// npcGoldenFrames 读 fssoloworld 的 ready_golden.bin（[u32 len][frame]×N）。
func npcGoldenFrames(t *testing.T) [][]byte {
	t.Helper()
	raw, err := os.ReadFile("../fssoloworld/ready_golden.bin")
	if err != nil {
		t.Skipf("golden bins 缺席: %v", err)
	}
	var frames [][]byte
	for off := 0; off+4 <= len(raw); {
		n := int(binary.LittleEndian.Uint32(raw[off:]))
		off += 4
		if n <= 0 || off+n > len(raw) {
			break
		}
		frames = append(frames, raw[off:off+n])
		off += n
	}
	return frames
}

// npcGoldenEntries 过滤 m229 c130 帧并解码为 NpcEntry。
func npcGoldenEntries(t *testing.T) []NpcEntry {
	t.Helper()
	var entries []NpcEntry
	for _, f := range npcGoldenFrames(t) {
		if f[0] != 0x1E {
			continue
		}
		e, ok := decodeNpcFrame(f)
		if !ok {
			continue
		}
		entries = append(entries, e)
	}
	return entries
}

// decodeNpcFrame 解一条 m229 c130 物化帧；非该族返回 ok=false。
func decodeNpcFrame(f []byte) (NpcEntry, bool) {
	if len(f) < 8 || f[0] != 0x1E {
		return NpcEntry{}, false
	}
	argc := int(binary.LittleEndian.Uint16(f[1:]))
	if argc != 16 {
		return NpcEntry{}, false
	}
	p := 3
	nextInt := func() int32 {
		v := int32(binary.LittleEndian.Uint32(f[p+1:]))
		p += 5
		return v
	}
	if f[p] != 2 || nextInt() != 229 {
		return NpcEntry{}, false
	}
	if f[p] != 2 || nextInt() != 130 {
		return NpcEntry{}, false
	}
	var e NpcEntry
	// 槽 2 obj
	if f[p] != 8 {
		return NpcEntry{}, false
	}
	e.ObjectID = binary.LittleEndian.Uint32(f[p+1:])
	e.OwnerID = binary.LittleEndian.Uint32(f[p+5:])
	p += 9
	// 槽 3 资源窄串
	s, ok := takeStr(f, &p)
	if !ok {
		return NpcEntry{}, false
	}
	e.Resource = s
	// 槽 4-7 四 float
	var fl [4]float32
	for i := range fl {
		if f[p] != 4 {
			return NpcEntry{}, false
		}
		fl[i] = float32frombits(binary.LittleEndian.Uint32(f[p+1:]))
		p += 5
	}
	e.X, e.Y, e.Z, e.Yaw = fl[0], fl[1], fl[2], fl[3]
	// 槽 8 Scale 窄串
	if e.Scale, ok = takeStr(f, &p); !ok {
		return NpcEntry{}, false
	}
	// 槽 9-12 四 int
	for _, slot := range []*int32{&e.Layer, &e.Group, &e.Aux1, &e.Aux2} {
		if f[p] != 2 {
			return NpcEntry{}, false
		}
		*slot = nextInt()
	}
	// 槽 13 宽串名（len 含 NUL 码元；文本 = len-2 字节）
	if f[p] != 7 {
		return NpcEntry{}, false
	}
	ln := int(binary.LittleEndian.Uint32(f[p+1:]))
	p += 5
	for i := 0; i+2 <= ln-2; i += 2 {
		e.Name += string(rune(binary.LittleEndian.Uint16(f[p+i:])))
	}
	p += ln
	// 槽 14 图标窄串 / 槽 15 三轴朝向窄串（原样透传）
	if e.Icon, ok = takeStr(f, &p); !ok {
		return NpcEntry{}, false
	}
	if e.OrientText, ok = takeStr(f, &p); !ok {
		return NpcEntry{}, false
	}
	if p != len(f) {
		return NpcEntry{}, false
	}
	return e, true
}

// takeStr 取 tag6 窄串（len 含 NUL）。
func takeStr(f []byte, p *int) (string, bool) {
	if *p >= len(f) || f[*p] != 6 {
		return "", false
	}
	ln := int(binary.LittleEndian.Uint32(f[*p+1:]))
	*p += 5
	if ln <= 0 || *p+ln > len(f) {
		return "", false
	}
	s := string(f[*p : *p+ln-1])
	*p += ln
	return s, true
}

func float32frombits(b uint32) float32 { return math.Float32frombits(b) }

func TestNpcGoldenRoundTrip(t *testing.T) {
	entries := npcGoldenEntries(t)
	if len(entries) != 62 {
		t.Fatalf("m229 c130 帧数 = %d，want 62", len(entries))
	}
	frames, err := NpcListFrames(entries)
	if err != nil {
		t.Fatalf("NpcListFrames: %v", err)
	}
	// 逐字节往返还原。
	golden := npcGoldenFrames(t)
	var goldenNpc [][]byte
	for _, f := range golden {
		if e, ok := decodeNpcFrame(f); ok {
			_ = e
			goldenNpc = append(goldenNpc, f)
		}
	}
	if len(frames) != len(goldenNpc) {
		t.Fatalf("重编码帧数 %d ≠ golden %d", len(frames), len(goldenNpc))
	}
	for i := range frames {
		if !bytes.Equal(frames[i], goldenNpc[i]) {
			t.Fatalf("帧 %d 往返不一致:\n got  %x\n want %x", i, frames[i], goldenNpc[i])
		}
	}
}

func TestNpcGoldenComposition(t *testing.T) {
	entries := npcGoldenEntries(t)
	counts := map[string]int{}
	seen := map[uint32]bool{}
	for _, e := range entries {
		res := e.Resource
		switch {
		case res == `npc\Guildqizhi01`, res == `npc\itemnpc057`:
		case strings.HasPrefix(res, `npc\1701_`):
			res = `npc\1701_*`
		}
		counts[res]++
		// ident 段位与唯一性（owner 观察跨 0x20..0x2Cxx 段）。
		if e.ObjectID>>24 != 0x03 {
			t.Fatalf("objectID %x 不在 0x03 段", e.ObjectID)
		}
		if e.OwnerID < 0x2000 || e.OwnerID >= 0x3000 {
			t.Fatalf("ownerID %x 不在 0x2xxx 段", e.OwnerID)
		}
		if seen[e.ObjectID] {
			t.Fatalf("objectID %x 重复", e.ObjectID)
		}
		seen[e.ObjectID] = true
	}
	want := map[string]int{`npc\Guildqizhi01`: 26, `npc\itemnpc057`: 9, `npc\1701_*`: 27}
	for res, n := range want {
		if counts[res] != n {
			t.Errorf("资源 %s 条数 = %d，want %d", res, counts[res], n)
		}
	}
	// 层号观察：301/302 双发（众数 302）。
	layers := map[int32]int{}
	for _, e := range entries {
		layers[e.Layer]++
	}
	if layers[301] != 31 || layers[302] != 31 {
		t.Errorf("层号分布 = %v，want 301×31/302×31", layers)
	}
	// WrapYaw：原始域朝向文本 wrap 后 == float 槽（62/62）。
	for i, e := range entries {
		parts := strings.Split(e.OrientText, ",")
		if len(parts) != 3 {
			t.Fatalf("帧 %d 朝向串 %q 非 三轴", i, e.OrientText)
		}
		orig, err := strconv.ParseFloat(parts[1], 32)
		if err != nil {
			t.Fatalf("帧 %d 朝向串 %q 解析: %v", i, e.OrientText, err)
		}
		w := WrapYaw(float32(orig))
		if math.Abs(float64(w-e.Yaw)) > 0.001 {
			t.Errorf("帧 %d wrap(%v)=%v ≠ float 槽 %v", i, orig, w, e.Yaw)
		}
	}
}

func TestOrientTriplet(t *testing.T) {
	cases := []struct {
		yaw  float32
		want string
	}{
		{0, "0.0,0.0,0.0"},
		{3.1415927, "0.0,3.142,0.0"},
		{1.5708, "0.0,1.571,0.0"},
		{-1.5708, "0.0,-1.571,0.0"},
		{-0.012, "0.0,-0.012,0.0"},
	}
	for _, c := range cases {
		if got := OrientTriplet(c.yaw); got != c.want {
			t.Errorf("OrientTriplet(%v) = %q, want %q", c.yaw, got, c.want)
		}
	}
}

func TestNpcIdentFormula(t *testing.T) {
	if got := NpcObjectID(1); got != 0x03000001 {
		t.Errorf("NpcObjectID(1) = %x", got)
	}
	if got := NpcOwnerID(1); got != 0x2001 {
		t.Errorf("NpcOwnerID(1) = %x", got)
	}
	// 与玩家 ident（0x11 段 / 0x7A5C0B）不冲突。
	if NpcObjectID(200)&0xFF000000 == 0x11000000 {
		t.Error("NPC objectID 段与玩家 0x11 段冲突")
	}
}
