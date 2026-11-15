package main

import (
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/cmd/protocol-probe/replaydata"
	"github.com/local/9yin-go-server/internal/role"
)

// unpackReplayPackFrames 解 [u32 len][frame]×N 打包文件（replayFramesPack
// 同口径）。
func unpackReplayPackFrames(t *testing.T, path string) [][]byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("replay pack %s unavailable: %v", path, err)
	}
	var frames [][]byte
	for len(raw) >= 4 {
		fl := int(binary.LittleEndian.Uint32(raw))
		if fl <= 0 || fl > len(raw)-4 || fl > 0x10000 {
			break
		}
		frames = append(frames, raw[4:4+fl])
		raw = raw[4+fl:]
	}
	return frames
}

// TestOfficialP2ViewCreateSetByteIdentical 帧替换档的 30 个 0x15 视图容器
// 必须与外服 Phase B bin 帧逐字节同形（容量/BaseCap 结构常量定案的回归
// 闸——bin 帧或结构表任一漂移都会在这里暴露）。
func TestOfficialP2ViewCreateSetByteIdentical(t *testing.T) {
	binFrames := unpackReplayPackFrames(t, `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p2_slim.bin`)
	set := officialP2ViewCreateSet()
	viewIdx := 0
	for i, frame := range binFrames {
		if frame[0] != 0x15 {
			continue
		}
		if viewIdx >= len(set) {
			t.Fatalf("bin 0x15 #%d 超出结构表（表长 %d）", i, len(set))
		}
		spec := set[viewIdx]
		viewIdx++
		if got := binary.LittleEndian.Uint16(frame[1:]); got != spec.id {
			t.Errorf("bin 0x15 #%d view=%d，表=%d（顺序错位）", i, got, spec.id)
			continue
		}
		var props []serverViewProperty
		if spec.baseCap {
			props = append(props, viewInt(propBaseCap, 18))
		}
		out, err := serverCreateViewWithProperties(officialToServerViewSpec(spec), props)
		if err != nil {
			t.Fatalf("view %d encode: %v", spec.id, err)
		}
		if string(out) != string(frame) {
			t.Errorf("bin 0x15 #%d view=%d 字节不同形:\n bin=%x\ngen=%x", i, spec.id, frame, out)
		}
	}
	if viewIdx != len(set) {
		t.Errorf("bin 0x15 共 %d 帧，结构表 %d 项", viewIdx, len(set))
	}
}

// TestEmbeddedReplayBinsMatchSource 内嵌 bin 与 tmp 源文件逐字节一致
// （防「只改一头」漂移：tmp 源演进后必须重拷 replaydata/）。
func TestEmbeddedReplayBinsMatchSource(t *testing.T) {
	for _, c := range []struct {
		name string
		path string
		emb  []byte
	}{
		{"p1", `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p1_slim.bin`, replaydata.Phase1},
		{"p2", `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p2_slim.bin`, replaydata.Phase2},
	} {
		src, err := os.ReadFile(c.path)
		if err != nil {
			t.Skipf("source bin %s unavailable: %v", c.path, err)
		}
		if len(c.emb) != len(src) {
			t.Fatalf("%s embedded=%dB source=%dB 长度漂移", c.name, len(c.emb), len(src))
		}
		for i := range src {
			if c.emb[i] != src[i] {
				t.Fatalf("%s 首个差异 @%d: emb=%02x src=%02x（重拷 replaydata/）", c.name, i, c.emb[i], src[i])
			}
		}
	}
}

// TestP1DispositionCoversSlimBin P1 分派表按 slim bin 13 帧逐序位核验：
// 原生帧 op 形必须对上；透传帧必须全部有 UNKNOWN 标注（无漏标）。
func TestP1DispositionCoversSlimBin(t *testing.T) {
	binFrames := unpackReplayPackFrames(t, `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p1_slim.bin`)
	if len(binFrames) != 13 {
		t.Skipf("p1 slim 帧数 %d ≠ 13（bin 已演化，分派表须重走查）", len(binFrames))
	}
	player := newPlayerActor("probe", 0)
	for idx, frame := range binFrames {
		_, _, _, isNative := nativeP1Disposition(idx, frame, player, role.Position{X: 928.3, Y: 48.7, Z: 998.9, Orient: 0.906}, nil)
		if isNative {
			continue
		}
		if m := p1UnknownMarker(idx, frame); m == "UNKNOWN-P1-op" {
			t.Errorf("P1#%d op=0x%02X 无专属标注（漏标）", idx, frame[0])
		}
	}
	// 原生三位的形状闸：#5 必须 0x0B、#11 必须单属性帧。
	if _, _, _, ok := nativeP1Disposition(5, append([]byte{0x0B}, make([]byte, 82)...), player, role.Position{}, nil); !ok {
		t.Error("P1#5 0x0B 应为原生替换")
	}
	qgFrame := sceneObjectIntProperty(1, 2, 694, 150)
	if len(qgFrame) != 18 || qgFrame[0] != 0x10 {
		t.Errorf("QingGongPoint 单属性帧形状异常: %x", qgFrame)
	}
}

// TestP1AvatarNativeSwap 帧替换档 #6 换代闸：nativeAvatar 非空时必须
// 直接采用我方生成档（不再透传外服档案、零改写）。
func TestP1AvatarNativeSwap(t *testing.T) {
	binFrames := unpackReplayPackFrames(t, `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p1_slim.bin`)
	if len(binFrames) != 13 {
		t.Skipf("p1 slim 帧数 %d 不等于 13（bin 已演化）", len(binFrames))
	}
	avatar := binFrames[6]
	mine := []byte{0x28, 0x46, 0xEE}
	frames, _, note, isNative := nativeP1Disposition(6, avatar, nil, role.Position{}, mine)
	if !isNative || len(frames) != 1 || string(frames[0]) != string(mine) {
		t.Fatalf("换代路径未生效: native=%v frames=%d", isNative, len(frames))
	}
	if !strings.Contains(note, "我方档案") {
		t.Errorf("标注缺「我方档案」: %q", note)
	}
}

// TestP1AvatarPosARewriteNative 化身 0x28（P1#6）在 native 路径内的
// posA 改写回归闸——SWAP_SELF 撤参后这里是「重进不在退出位置」的唯一
// 防线：除 posA 16B 外必须与 bin 帧逐字节相等。
func TestP1AvatarPosARewriteNative(t *testing.T) {
	binFrames := unpackReplayPackFrames(t, `G:\pojie\ai\9yin-server-go\tmp\op28_analysis\replay_p1_slim.bin`)
	if len(binFrames) != 13 {
		t.Skipf("p1 slim 帧数 %d ≠ 13（bin 已演化）", len(binFrames))
	}
	avatar := binFrames[6]
	if avatar[0] != 0x28 {
		t.Fatalf("P1#6 非 0x28: 0x%02X", avatar[0])
	}
	// SELF_IDS 借用身份：官方对 0x07020fa4/0x003b3cc3（bin 内化身帧实值）。
	savedObj, savedOwn := playerObjectID, playerOwnerID
	playerObjectID, playerOwnerID = 0x07020fa4, 0x003b3cc3
	defer func() { playerObjectID, playerOwnerID = savedObj, savedOwn }()

	pos := role.Position{X: 11.5, Y: 22.5, Z: 33.5, Orient: 0.25}
	frames, _, note, isNative := nativeP1Disposition(6, avatar, nil, pos, nil)
	if !isNative {
		t.Fatal("P1#6 化身 0x28 应由 native 路径接管")
	}
	if len(frames) != 1 || len(frames[0]) != len(avatar) {
		t.Fatalf("化身帧数/长度异常: %d 帧 %d B（原 %d B）", len(frames), len(frames[0]), len(avatar))
	}
	if posOff, ok := officialFramePosAOffset(frames[0], playerObjectID, playerOwnerID); !ok {
		t.Fatal("改写帧头部形不符（posA 定位失败）")
	} else {
		f32 := func(off int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(frames[0][off:])) }
		x := f32(posOff)
		y := f32(posOff + 4)
		z := f32(posOff + 8)
		if x != pos.X || y != pos.Y || z != pos.Z {
			t.Errorf("posA 未对齐: (%v,%v,%v) 期望 (%v,%v,%v)", x, y, z, pos.X, pos.Y, pos.Z)
		}
	}
	if !strings.Contains(note, "UNKNOWN-4") {
		t.Errorf("化身帧标注缺失 UNKNOWN-4: %q", note)
	}
}
