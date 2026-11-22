package entergen

// maprecords.go 大地图开图/迷雾/自动寻路三帧（b-line map_records.go 移植，
// 外服 c.pcapng 2026-11-10 原始字节定版）。三件缺一不可——全缺时大地图
// 图层为空、点击无反应、msg266 都不发起（「无法点击地图自动跑」根因）：
//
//   - 0x11 OpenMapRec#51 +24 行 int32 开图位图（PreReady 段，0x0D 后、
//     ClientReady 前）：行 i 位 j = 客户端 scenes.ini 全局场景段号 i*32+j，
//     form_map_scene.lua update_scene_list_by_OpenMapRec 逐位点亮城市标签；
//   - 0x11 MapMaskRec#56 +1 行 [段号, resource, 1]（同窗口）：当前场景
//     迷雾已开（canclosemask() 按 col1=场景名、col2=1 判定）；
//   - 0x10 AutoTrace=1（属性 idx 234，Ready 段 m52 前）：引擎侧地图点击
//     寻路闸门。
//
// 0x0C 切图清玩家对象后须以新场景 resource 重推（b-line stable re-entry
// 同款）——本仓切图链未落地，登录/重登窗口先行。

import (
	"encoding/binary"
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

const (
	recordOpenMap = uint16(51) // OpenMapRec（单列 int32）
	recordMapMask = uint16(56) // MapMaskRec（int32/窄串/int32）

	// openMapRecRows 开图位图行数（24×32=768 个场景索引空间）。
	openMapRecRows = 24
)

// officialOpenMapBits 外服 c.pcapng 0.128s 实测开图位图基线（24 行原值，
// 字节级一致下发）。行 11=0x8000f0c3 含 bit14=索引 366=scene22，与同簇
// MapMaskRec 行 col0=366 互证；叠加当前场景位保证所在地必开。
var officialOpenMapBits = [openMapRecRows]uint32{
	0x7a03fffe, 0xe1812a28, 0x040000f6, 0x0001fe10,
	0x00000000, 0x00000000, 0x4cfeee00, 0x4e005240,
	0x403d007b, 0x00002300, 0x1c041220, 0x8000f0c3,
	0x00047d00, 0x00be0000, 0x00000000, 0x00000000,
	0x1f3bee00, 0x98280180, 0x785c0000, 0x54080020,
	0x00000154, 0x00050000, 0x07e00000, 0x00000001,
}

// EffectiveOpenMapBits 生效开图位图：存档位（roleState.OpenMaps，玩家级
// 累积）优先，无档回落外服基线；再叠加当前场景位（所在地必开）。进场
// 下发与 fssoloworld 回写共用本口径（nil OpenMaps 首登=基线+出生场景）。
func EffectiveOpenMapBits(state rolestate.RoleState, sceneIndex int) [openMapRecRows]uint32 {
	bits := officialOpenMapBits
	if len(state.OpenMaps) == openMapRecRows {
		copy(bits[:], state.OpenMaps)
	}
	if sceneIndex >= 0 {
		overlaySceneBit(&bits, uint32(sceneIndex))
	}
	return bits
}

// MapRecordsFrames 开图双记录帧（PreReady 段；sceneIndex<0 = scenes.ini
// 缺席，只发位图、跳过迷雾行）。序：迷雾行先、位图后（外服同序）。
func MapRecordsFrames(ctx Context, sceneIndex int) ([][]byte, error) {
	frames := make([][]byte, 0, 2)
	bits := EffectiveOpenMapBits(ctx.State, sceneIndex)
	if sceneIndex >= 0 {
		overlaySceneBit(&bits, uint32(sceneIndex))
		mask, err := RecordAddCellsFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID,
			recordMapMask, []RecordCell{RecordInt(uint32(sceneIndex)), RecordString(ctx.State.Resource), RecordInt(1)})
		if err != nil {
			return nil, fmt.Errorf("map mask rec: %w", err)
		}
		frames = append(frames, mask)
	}
	frames = append(frames, openMapRecFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, bits))
	return frames, nil
}

// AutoTraceFrame 自动寻路开关单属性帧（Ready 段；0x10 idx234=1）。
func AutoTraceFrame(ctx Context) ([]byte, error) {
	return ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0,
		[]IndexedProperty{NamedProperty("AutoTrace", Int32Value(1))})
}

// openMapRecFrame OpenMapRec 帧：0x11 + 24 行 int32 位图一帧连发
// （外服 len=112，不拆单行帧）。多行形态与 RecordAddCellsFrame 的单行
// （行数=1）方言不同——头同构，行区按 schema 列型 ×24。
func openMapRecFrame(objectID, ownerID uint32, bits [openMapRecRows]uint32) []byte {
	recordIndex := clientdata.MustRecordIndexByName("OpenMapRec")
	msg := make([]byte, 16, 16+openMapRecRows*4)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 起始行 0
	binary.LittleEndian.PutUint16(msg[14:], openMapRecRows)
	for _, bits := range bits {
		msg = binary.LittleEndian.AppendUint32(msg, bits)
	}
	return msg
}

// overlaySceneBit 把全局场景段号对应位点亮（基线位图叠加当前场景位）。
func overlaySceneBit(bits *[openMapRecRows]uint32, index uint32) {
	row, bit := int(index)/32, uint(index%32)
	if row >= 0 && row < openMapRecRows {
		bits[row] |= 1 << bit
	}
}
