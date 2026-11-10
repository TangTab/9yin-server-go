package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/clientdata"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 大地图开图/迷雾/自动寻路三帧（官服 c.pcapng 2026-11-10 原始字节定版）。
//
// 官服在两处下发，缺一不可（本服此前全缺——大地图图层为空、点击无反应，
// msg266 都不发起）：
//
//   - 0.128s（阶段C 记录簇，选角链 0x0D 后、ClientReady 前）：
//     0x11 OpenMapRec#51 +24 行 int32 开图位图——行 i 位 j = 全局场景索引
//     i*32+j（客户端 ini\scenes.ini 段号；form_map_scene.lua
//     update_scene_list_by_OpenMapRec 按 ext_int_to_byte 逐位点亮城市标签）；
//     0x11 MapMaskRec#56 +1 行 [场景索引, resource, 1]——当前场景迷雾已开
//     （canclosemask() 按 col1=场景名、col2=1 判定，form_map_scene.lua:1920）。
//   - 4.81s（ClientReady 簇，m144 后 m52 前）：
//     0x10 AutoTrace=1（属性表 idx 234）——自动寻路开关，引擎侧地图点击
//     寻路闸门。
//
// 记录帧布局与 serverRecordAddCells 同构（0x11 | obj u32 | owner u32 |
// rec u16 | 起始行 u16 | 行数 u16 | 行数据×N）；OpenMapRec 单列 int32 官服
// 一帧 24 行连发（len=112），不拆单行帧。
const (
	recordOpenMap uint16 = 51 // OpenMapRec
	recordMapMask uint16 = 56 // MapMaskRec

	// openMapRecRows 官服位图行数（24×32=768 个场景索引空间）。
	openMapRecRows = 24
)

// officialOpenMapBits 官服 c.pcapng 0.128s 实测开图位图（24 行原值）。
// 行 11 = 0x8000f0c3 含 bit14 = 索引 366 = scene22，与同簇 MapMaskRec
// 行 col0=366 互证。作为基线样本下发（官服方言字节级一致），再叠加
// 当前场景位保证所在地必开。
var officialOpenMapBits = [openMapRecRows]uint32{
	0x7a03fffe, 0xe1812a28, 0x040000f6, 0x0001fe10,
	0x00000000, 0x00000000, 0x4cfeee00, 0x4e005240,
	0x403d007b, 0x00002300, 0x1c041220, 0x8000f0c3,
	0x00047d00, 0x00be0000, 0x00000000, 0x00000000,
	0x1f3bee00, 0x98280180, 0x785c0000, 0x54080020,
	0x00000154, 0x00050000, 0x07e00000, 0x00000001,
}

// ---- ini\scenes.ini 全局场景索引表 ----

var (
	sceneIndexOnce  sync.Once
	sceneIndexByRes map[string]uint32
	// sceneGlobalIndexLookup 是测试注入口（资源树缺席时用样本表替换）。
	sceneGlobalIndexLookup = sceneGlobalIndex
)

// sceneGlobalIndex 取场景 resource（如 "scene22"/"city02"）在客户端
// ini\scenes.ini 的全局段号——OpenMapRec 位图与 MapMaskRec col0 共用
// 这套编号。表缺失/未收录返回 false（调用方降级：只发官服样本位图）。
func sceneGlobalIndex(resource string) (uint32, bool) {
	sceneIndexOnce.Do(loadSceneIndexTable)
	if sceneIndexByRes == nil {
		return 0, false
	}
	index, ok := sceneIndexByRes[strings.ToLower(strings.TrimSpace(resource))]
	return index, ok
}

// loadSceneIndexTable 解析 resources/modern/ini/scenes.ini（GBK）：
// 段号即全局场景索引，Config=ini\scene\<resource> 的文件基名即地形名
// （与 resourceForDoorScene 推导同名，366=scene22 实测锚点）。
func loadSceneIndexTable() {
	path := filepath.Join(runtimeProjectRoot, "resources", "modern", "ini", "scenes.ini")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("map records: scenes.ini unavailable (%s): %v — OpenMapRec falls back to official sample, MapMaskRec row skipped", path, err)
		return
	}
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
	if err != nil {
		log.Printf("map records: decode GBK scenes.ini: %v", err)
		return
	}
	sceneIndexByRes = parseSceneIndexTable(string(decoded))
}

// parseSceneIndexTable 纯解析（测试入口）：段号 → Config 基名首现绑定。
func parseSceneIndexTable(text string) map[string]uint32 {
	table := make(map[string]uint32, 300)
	section := -1
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if id, err := strconv.Atoi(line[1 : len(line)-1]); err == nil {
				section = id
			} else {
				section = -1
			}
			continue
		}
		if section < 0 || !strings.HasPrefix(strings.ToLower(line), "config=") {
			continue
		}
		config := strings.ReplaceAll(strings.TrimSpace(line[len("config="):]), "/", `\`)
		base := strings.TrimSuffix(filepath.Base(config), filepath.Ext(config))
		if base == "" {
			continue
		}
		// 键口径与 resourceForDoorScene 一致：city/school/born/scene 前缀
		// 截到首个 '_'（city02_suzhou → city02），其余取全基名；同地形
		// 多段取首现。
		key := strings.ToLower(sceneResourceOfBase(base))
		if _, exists := table[key]; !exists {
			table[key] = uint32(section)
		}
	}
	return table
}

// sceneResourceOfBase 配置基名 → 地形名（resourceForDoorScene 的纯函数核）。
func sceneResourceOfBase(base string) string {
	lower := strings.ToLower(base)
	for _, prefix := range []string{"city", "school", "born", "scene"} {
		if strings.HasPrefix(lower, prefix) {
			if separator := strings.IndexByte(lower, '_'); separator > 0 {
				return lower[:separator]
			}
		}
	}
	return lower
}

// ---- 帧构造 ----

// openMapRecFrame 官服同形 OpenMapRec 帧：0x11 + 24 行 int32 位图一帧连发。
func openMapRecFrame(bits [openMapRecRows]uint32) []byte {
	msg := make([]byte, 16, 16+openMapRecRows*4)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], playerObjectID)
	binary.LittleEndian.PutUint32(msg[6:], playerOwnerID)
	binary.LittleEndian.PutUint16(msg[10:], recordOpenMap)
	binary.LittleEndian.PutUint16(msg[12:], 0) // 起始行 0
	binary.LittleEndian.PutUint16(msg[14:], openMapRecRows)
	for _, bits := range bits {
		msg = binary.LittleEndian.AppendUint32(msg, bits)
	}
	return msg
}

// mapMaskRecFrame MapMaskRec 单行 [全局索引, resource, 1]（迷雾已开）。
func mapMaskRecFrame(sceneIndex uint32, resource string) ([]byte, error) {
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordMapMask, []recordCell{
		recordInt(sceneIndex),
		recordString(resource),
		recordInt(1),
	})
}

// grantMapRecords 选角链（0x0D 后、ClientReady 前，官服 0.128s 同位）下发
// 开图位图 + 当前场景迷雾行。0x0C 切图清玩家对象后由 stable re-entry 分支
// 以新场景 resource 重推。
func grantMapRecords(conn sceneMessageConnection, resource string) error {
	bits := officialOpenMapBits
	resource = strings.ToLower(strings.TrimSpace(resource))
	if index, ok := sceneGlobalIndexLookup(resource); ok {
		overlaySceneBit(&bits, index)
		if err := conn.WriteFrame(mustMapMaskFrame(index, resource)); err != nil {
			return fmt.Errorf("write MapMaskRec row %s: %w", resource, err)
		}
	} else {
		log.Printf("map records: scene resource %q absent from scenes.ini — MapMaskRec row skipped", resource)
	}
	if err := conn.WriteFrame(openMapRecFrame(bits)); err != nil {
		return fmt.Errorf("write OpenMapRec bitmap: %w", err)
	}
	return nil
}

// overlaySceneBit 把全局场景索引对应位点亮（官服位图叠加当前场景位）。
func overlaySceneBit(bits *[openMapRecRows]uint32, index uint32) {
	row, bit := int(index)/32, uint(index%32)
	if row < openMapRecRows {
		bits[row] |= 1 << bit
	}
}

// mustMapMaskFrame 的编码只依赖 schema 列型（int/string/int），失败即
// 官方表漂移，编码期暴露——不放回线上静默吞。
func mustMapMaskFrame(sceneIndex uint32, resource string) []byte {
	frame, err := mapMaskRecFrame(sceneIndex, resource)
	if err != nil {
		panic(fmt.Sprintf("map records: encode MapMaskRec %s: %v", resource, err))
	}
	return frame
}

// pushAutoTrace ClientReady 簇单属性帧：AutoTrace=1（官服 4.81s，m144 后
// m52 前；属性挂玩家对象，0x0C 清对象后随 re-entry 重推）。
func pushAutoTrace(conn sceneMessageConnection) error {
	entry, ok := clientdata.PlayerProperty("AutoTrace")
	if !ok {
		return fmt.Errorf("AutoTrace absent from official property table")
	}
	frame, err := objectSingleProperty(playerObjectID, playerOwnerID, entry.Index, entry.Name, clientdata.Int32Value(1))
	if err != nil {
		return fmt.Errorf("encode AutoTrace: %w", err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("write AutoTrace: %w", err)
	}
	return nil
}
