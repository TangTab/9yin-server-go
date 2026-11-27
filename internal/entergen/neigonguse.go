package entergen

// neigonguse.go 运行内功切换（C2S 215）的帧编码（2433 线
// player_progress.go case 215 / writeEquippedNeiGongBuffFrame 口径）：
//
//	MaxSP 专帧   0x10 单属性（换书改怒气上限；先刷分母再发资源批——
//	             与登录 BasePropertyFrames 尾追 MaxSP 同源）
//	资源批       0x10 大批：CurNeiGong/CurNeiGongWuXing/FacultyName/
//	             BufferInfo4（动态合成，旧书 buff 图标随批刷新消失）
//	战斗面板     单属性 0x10 逐帧（CombatAttributeValues 同一取值面）
//	m169 两值帧  [169, buff名, obj] 挂新书可见 buff（buff 栏图标只由
//	             m169 驱动——属性帧不足以换图标）

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// S2CFacultyMessage 修炼族原生通知 m191 [191, sub, arg2]（faculty.lua：
// 14=升级、22=演武开始、23=演武退出、24=继续演武）。
const S2CFacultyMessage int32 = 191

// FacultyMessageFrame m191 两整值帧。
func FacultyMessageFrame(sub, arg2 int32) ([]byte, error) {
	return CustomMessageFrame(S2CFacultyMessage, CustomInt(sub), CustomInt(arg2))
}

// BookUpdateFrame View43 行级增量帧（0x10 isView=1 view=43 owner=行号）：
// 已存在行的等级/进度/内力变化必须走属性增量——裸 0x18 VIEW_ADD 对已
// 存在行号不触发客户端 databinder 的 updateitem，武学面板(K)内功页会停
// 在旧等级/旧进度（2433 线 bookUpdateFrame：2026-11-08 50514 抓包实证）。
// neiGongLevel≤0 时省略 NeiGongLevel（varprop 档缺失，bookViewRow 同口径）。
func BookUpdateFrame(slot uint16, book rolestate.NeiGongBook, neiGongLevel int32) ([]byte, error) {
	props := []IndexedProperty{
		NamedProperty("Level", ByteValue(byte(clampU8(book.Level)))),
		NamedProperty("MaxLevel", Int32Value(int32(book.MaxLevel))),
		NamedProperty("CurFillValue", Int32Value(int32(book.Fill))),
		NamedProperty("TotalFillValue", Int32Value(int32(book.Total))),
		NamedProperty("PowerValue", Int32Value(int32(book.Power))),
		NamedProperty("MaxPowerValue", Int32Value(int32(book.MaxPower))),
	}
	if neiGongLevel > 0 {
		props = append(props, NamedProperty("NeiGongLevel", Int32Value(neiGongLevel)))
	}
	return ObjectPropertyFrame(uint32(ViewportNeiGong), uint32(slot), 1, props)
}

// RageCapOf 运行指定内功时的怒气上限（RageCap 的 configID 直取面：
// 215 换书后运行时重算用）。
func RageCapOf(configID string) int32 {
	return baseMaxSP + neiGongRageCapAdd(configID)
}

// EquipBuffFrame m169 两值帧 [169, buff名, obj]：装备内功可见 buff 的
// 登记（LoginBuffFrames 同形；buff 栏图标只由此帧驱动）。
func EquipBuffFrame(id Identity, buffID string) ([]byte, error) {
	return CustomMessageFrame(MsgBuff,
		CustomString(buffID), CustomObject(id.ObjectID, id.OwnerID))
}

// BufferInfo4CSV 装备内功 buff 的 BufferInfo4 槽串（"staticHex,obj,
// own,level,0,0"；无 buff 空串）。static=buff_new.ini 的 StaticData，
// obj/own 大写十六进制（appearance.go 登录批同格式）。
func BufferInfo4CSV(id Identity, static uint32, buffLevel int32) string {
	if static == 0 {
		return ""
	}
	return fmt.Sprintf("%X,%X,%X,%d,0,0", static, id.ObjectID, id.OwnerID, buffLevel)
}
