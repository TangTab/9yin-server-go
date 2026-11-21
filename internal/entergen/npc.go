package entergen

// npc.go NPC 物化帧族（m229 c130 大族）——外服 B 线方言：
// ClientReady 应答最前连发，每条 0x1E TVarList 16 槽（ready_golden.bin
// 帧 #2-63 实证 62 条，26×npc\Guildqizhi01 旗帜 + 9×npc\itemnpc057 复活
// 特效 + 28×npc\1701_* 建筑）：
//
//	[int 229, int 130, obj(对), 窄串资源, f32 x, f32 y(高), f32 z,
//	 f32 yaw, 窄串Scale("1.100,1.100,1.100"，空=不缩放), int 层(301|302),
//	 int 组号(旗帜12/特效6/余0), int aux1(多0), int aux2(多0),
//	 宽串名(物件空/旗帜"Yaochi"), 窄串图标("7.png#...#r,g,b,r"，
//	 空默认), 窄串朝向("0.0,3.141,0.0" 三轴欧拉)]
//
// 定版与待验项见 docs/official/外服NPCm229物化方言定版.md。

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NpcEntry 一条待物化 NPC（m229 帧 16 槽的高层投影）。
type NpcEntry struct {
	ObjectID uint32 // wire objectID（0x03 段）
	OwnerID  uint32 // wire ownerID（0x20xx 段）
	Resource string // 模型资源路径（npc\Guildqizhi01 形；客户端按路径装载）
	X, Y, Z  float32
	Yaw      float32 // 槽7：朝向 wrap 到 [0,2π)（golden 62/62 命中）
	OrientText string // 槽15 三轴欧拉串（"0.698,-1.571,0.0"——pitch 可非零；
	// 原始域值与 f32 wrap 后互推丢精度，独立承载；空=由 Yaw 生成）

	Scale string // 缩放串（NpcConfig Scale 原样直传；空=不缩放）
	Layer int32  // 槽9 层号：golden 观察 301/302 双发，众数 302
	Group int32  // 槽10 组号小枚举：旗帜 12 / 复活特效 6 / 余 0（语义待真机定案）
	Aux1  int32  // 槽11（golden 多 0）
	Aux2  int32  // 槽12（golden 多 0）
	Name  string // 宽串显示名：物件空、人物填模板名（人物名行为待真机验证）
	Icon  string // 旗帜公会图标串（空默认）
}

// NpcObjectID 场景内 NPC wire objectID（idx 1 起）：0x03 段——golden
// 观察 0x03002091..，与玩家 0x11 段隔离。
func NpcObjectID(index int) uint32 { return 0x03000000 | uint32(index) }

// NpcOwnerID 场景内 NPC wire ownerID：0x20xx 段（golden 观察 0x209e..）。
// 避开 owner=1（外服客户端解析为「1 号玩家化身」的 fork 方言坑）与
// 玩家 ownerID 0x7A5C0B。
func NpcOwnerID(index int) uint32 { return 0x2000 + uint32(index) }

// NpcListFrames 每条 NPC 一帧 m229 c130。
func NpcListFrames(entries []NpcEntry) ([][]byte, error) {
	frames := make([][]byte, 0, len(entries))
	for i, e := range entries {
		if e.ObjectID == 0 || e.OwnerID == 0 {
			return nil, fmt.Errorf("npc entry %d: zero ident (%x,%x)", i, e.ObjectID, e.OwnerID)
		}
		if e.Resource == "" {
			return nil, fmt.Errorf("npc entry %d: empty resource", i)
		}
		frame, err := CustomMessageFrame(229,
			CustomInt(130),
			CustomObject(e.ObjectID, e.OwnerID),
			CustomString(e.Resource),
			CustomFloat32(e.X),
			CustomFloat32(e.Y),
			CustomFloat32(e.Z),
			CustomFloat32(e.Yaw),
			CustomString(e.Scale),
			CustomInt(e.Layer),
			CustomInt(e.Group),
			CustomInt(e.Aux1),
			CustomInt(e.Aux2),
			CustomWideString(e.Name),
			CustomString(e.Icon),
			CustomString(OrientTripletText(e.OrientText, e.Yaw)),
		)
		if err != nil {
			return nil, fmt.Errorf("npc entry %d: %w", i, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// OrientTriplet 朝向三轴欧拉窄串（golden 62 条格式实证）：%.3f 四舍五入
// 去尾零、保底 1 位小数（"0.0,3.141,0.0"/"0.698,-1.571,0.0"）。
// 本仓摆放数据只有 yaw（ay），pitch/roll 恒 0。
func OrientTriplet(yaw float32) string {
	return "0.0," + orientScalar(yaw) + ",0.0"
}

// OrientTripletText 带原始三轴文本的朝向串（text 非空时直用——原始域
// 朝向与 wrap 后的 Yaw 互推丢精度）。
func OrientTripletText(text string, yaw float32) string {
	if text == "" {
		return OrientTriplet(yaw)
	}
	return text
}

// WrapYaw 朝向 wrap 到 [0,2π)（golden 槽7 float 全量命中）。
func WrapYaw(v float32) float32 {
	const tau = 6.2831855
	if v >= 0 && v < tau {
		return v
	}
	w := float32(math.Floor(float64(v)/tau)) * tau
	r := v - w
	if r < 0 || r >= tau {
		return 0
	}
	return r
}

// orientScalar %.3f → 去尾零 → 整数化时保 ".0"。
func orientScalar(v float32) string {
	s := strconv.FormatFloat(float64(v), 'f', 3, 32)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		if strings.HasSuffix(s, ".") {
			s += "0"
		}
	}
	return s
}
