package main

// 回放 0x10 影子属性帧「真值优先，影子兜底」（2026-11-15 用户拍板）：
// Phase B bin 里的官方抓包属性帧（外服账号影子值，如 MaxParry=10372）
// 逐行解析——有我方真值来源的 idx 就地换成我方值；我方还没有的
// （492-496 五维Add 聚合/516-527 分系增伤/540+ AI 族/1405 等）保留
// 影子字节当默认值，待值域定案后逐个销项。
//
// 数据字典：docs/official/回放bin-LZ明文结构与属性盘点.md B 节
// （119 帧/107 idx，值宽按 2433 表型码逐属性判定）。
//
// 安全纪律：
//   - 只处理 [0x10][isView==0][obj==self] 帧；视图行(0x10 isView=1)与
//     场景对象帧不碰（我方另有专通道）。
//   - 影子行重编码字节必须与原帧逐位一致（decoder↔encoder round-trip
//     校验），校验不过整帧透传——方言零漂移。
//   - MOTION-15 移动族已在上游就地改写（replay_native_replace.go），
//     本层处理其余 idx。

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// attrShadowExcluded 显式保留影子的 idx：我方值源存在但为未定案的
// 占位常量，与官方口径差距大——真值定案前不算「能用我们的」。
var attrShadowExcluded = map[uint16]string{
	481: "ParryUpSpeed（我方恒 4 vs 官方 104，公式未定位）",
	624: "MPHeartSpeedAdd（我方恒 0 vs 官方 -350）",
}

// replayAttrOverrides 我方真值 override 表（每连接 Phase B 开始时构建一次）。
// 值源=stateProperties 登录全量批（vitals/格挡族/回复速度/BufferInfo 槽/
// 物理档案/经脉 8 件套/progress 域）打底，combatPanelProperties（装备随机
// 属性折算+阴阳太极累计的战斗面板权威出口）覆盖同 idx。
func replayAttrOverrides(player *playerActor) map[uint16]clientdata.IndexedProperty {
	overrides := make(map[uint16]clientdata.IndexedProperty)
	if player == nil {
		return overrides
	}
	for _, prop := range player.stateProperties() {
		if prop.Value.Type == 0 {
			continue
		}
		overrides[prop.Index] = prop
	}
	for _, prop := range player.combatPanelProperties() {
		if prop.Value.Type == 0 {
			continue
		}
		overrides[prop.Index] = prop
	}
	for idx := range attrShadowExcluded {
		delete(overrides, idx)
	}
	return overrides
}

// decodeAttrShadowValue 按线型解码影子行值。返回 (值, 消耗字节, ok)。
func decodeAttrShadowValue(wire clientdata.WireType, b []byte) (clientdata.Value, int, bool) {
	switch wire {
	case clientdata.WireByte:
		if len(b) < 1 {
			return clientdata.Value{}, 0, false
		}
		return clientdata.ByteValue(b[0]), 1, true
	case clientdata.WireWord:
		if len(b) < 2 {
			return clientdata.Value{}, 0, false
		}
		return clientdata.WordValue(binary.LittleEndian.Uint16(b)), 2, true
	case clientdata.WireInt32:
		if len(b) < 4 {
			return clientdata.Value{}, 0, false
		}
		return clientdata.Int32Value(int32(binary.LittleEndian.Uint32(b))), 4, true
	case clientdata.WireInt64, clientdata.WireObject:
		if len(b) < 8 {
			return clientdata.Value{}, 0, false
		}
		return clientdata.Value{Type: wire, U64: binary.LittleEndian.Uint64(b)}, 8, true
	case clientdata.WireFloat32:
		if len(b) < 4 {
			return clientdata.Value{}, 0, false
		}
		return clientdata.Float32Value(math.Float32frombits(binary.LittleEndian.Uint32(b))), 4, true
	case clientdata.WireString:
		if len(b) < 4 {
			return clientdata.Value{}, 0, false
		}
		n := int(binary.LittleEndian.Uint32(b))
		if n < 1 || 4+n > len(b) {
			return clientdata.Value{}, 0, false
		}
		return clientdata.StringValue(string(b[4 : 4+n-1])), 4 + n, true
	case clientdata.WireWideString:
		if len(b) < 4 {
			return clientdata.Value{}, 0, false
		}
		n := int(binary.LittleEndian.Uint32(b))
		if n < 2 || n%2 != 0 || 4+n > len(b) {
			return clientdata.Value{}, 0, false
		}
		units := make([]uint16, n/2)
		for i := range units {
			units[i] = binary.LittleEndian.Uint16(b[4+2*i:])
		}
		return clientdata.WideStringValue(string(utf16.Decode(units[:len(units)-1]))), 4 + n, true
	default:
		return clientdata.Value{}, 0, false
	}
}

// replayAttrShadowSwap 解析影子 0x10 帧行并按 overrides 换值重建。
// 返回 (输出帧, 换值行数, 保留行数, ok)；ok=false 时调用方必须透传原帧。
// 命中 override 为 0 时返回原帧（零风险快路径）。
func replayAttrShadowSwap(frame []byte, obj uint32, overrides map[uint16]clientdata.IndexedProperty) (out []byte, swapped, kept int, ok bool) {
	// 帧形：[0x10][isView][obj u32][own u32][cnt u16][idx u16 + 值]×cnt
	if len(frame) < 12 || frame[0] != 0x10 || frame[1] != 0 {
		return frame, 0, 0, false
	}
	if binary.LittleEndian.Uint32(frame[2:]) != obj {
		return frame, 0, 0, false
	}
	own := binary.LittleEndian.Uint32(frame[6:])
	cnt := int(binary.LittleEndian.Uint16(frame[10:]))
	props := make([]clientdata.IndexedProperty, 0, cnt)
	p := 12
	for i := 0; i < cnt; i++ {
		if p+2 > len(frame) {
			return frame, 0, 0, false
		}
		idx := binary.LittleEndian.Uint16(frame[p:])
		p += 2
		entry, inTable := clientdata.PropertyByIndex(int(idx))
		if !inTable || entry.Type == 0 {
			return frame, 0, 0, false
		}
		value, used, valid := decodeAttrShadowValue(entry.Type, frame[p:])
		if !valid || p+used > len(frame) {
			return frame, 0, 0, false
		}
		p += used
		if override, hit := overrides[idx]; hit && override.Value.Type == entry.Type {
			props = append(props, override)
			swapped++
		} else {
			props = append(props, clientdata.IndexedProperty{Index: idx, Name: entry.Name, Value: value})
			kept++
		}
	}
	if p != len(frame) {
		return frame, 0, 0, false // 残差≠0：解析错位，透传
	}
	if swapped == 0 {
		return frame, 0, cnt, true // 纯影子帧：原字节透传
	}
	// 方言零漂移闸：影子行（未换值行）重编码必须与原帧逐位一致。
	shadow := make([]clientdata.IndexedProperty, 0, cnt)
	q := 12
	for i := 0; i < cnt; i++ {
		idx := binary.LittleEndian.Uint16(frame[q:])
		q += 2
		entry, _ := clientdata.PropertyByIndex(int(idx))
		value, used, _ := decodeAttrShadowValue(entry.Type, frame[q:])
		q += used
		shadow = append(shadow, clientdata.IndexedProperty{Index: idx, Name: entry.Name, Value: value})
	}
	roundTrip, rerr := sceneObjectProperties(obj, own, 0, shadow)
	if rerr != nil || !bytes.Equal(roundTrip, frame) {
		log.Printf("attr-shadow: frame round-trip mismatch (len %d vs %d, err=%v); passthrough", len(roundTrip), len(frame), rerr)
		return frame, 0, 0, false
	}
	rebuilt, berr := sceneObjectProperties(obj, own, 0, props)
	if berr != nil {
		log.Printf("attr-shadow: rebuild encode: %v; passthrough", berr)
		return frame, 0, 0, false
	}
	return rebuilt, swapped, kept, true
}
