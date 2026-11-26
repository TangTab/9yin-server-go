package entergen

import (
	"strconv"
	"strings"
)

// 怒气上限（MaxSP）= 基础 80 + 当前【运行】内功的档位加成（移植 2433 线
// cmd/protocol-probe/neigong_rage_cap.go）。分档权威 = docs/guides/
// neigong-beginner-guide.md §六：只看运行中的那一本，非叠加——
//
//	宗派内功  一内 +10、二内 +20、三内 +30、四内 +40、五内/六内/会意功 +50
//	隐世门派  一内 +30、二内 +40
//	势力      徐家庄 +10；金针沈家/万兽山庄 +30；天轮寺/青衣阁/移花宫/无根门 +40
//	江湖内功  坐忘功 +10；其它 +40
//
// 书目分层（哪个 ConfigID 属哪一档）取自客户端 ini/ui/wuxue/neigong_type.ini
//（UI 索引，极少变动；资源升级需人工复核）：宗派段后缀 _001.._006 = 一内..
// 六内、_101 = 会意功；隐世段档位按段内【位置】而非后缀（ng_xd/ng_gm 段首位
// 是 _002）。指南势力表列「丁氏山庄」而 ini 无此段、恰多出桃花岛段，其余 7 家
// 名字逐一吻合，故桃花岛按该 +40 槽位处理。指南未覆盖的段与未知 ConfigID
// 返回 0——分档无依据时不虚构加成。
//
// 线上口径：官方全程零下发 MaxSPAdd(583)，加成在服务端折进 MaxSP(580) 一次
// 给出；客户端怒气球分母 = MaxSP + MaxSPAdd（form_sp_bar.lua），583 不出帧时
// 按默认 0 参与，分母与本服上限一致。

// baseMaxSP 怒气上限的基础位：无内功 / 书目无分档依据时的上限。
const baseMaxSP int32 = 80

// maxRageCapAdd 内功能给的最大档位加成（五内/六内/会意功封顶）。
const maxRageCapAdd int32 = 50

// RageCap 本次进场的怒气上限：基础位 + 运行内功的档位加成。运行内功 = 人物
// 数据大批按其下发层级 / 实力的那本（vitalBook）。与当前怒气无关——那是会话
// 值，走 Context.SP。
func RageCap(ctx Context) int32 {
	return baseMaxSP + neiGongRageCapAdd(vitalBook(ctx).ConfigID)
}

// schoolNeiGongPrefixes 宗派段（neigong_type.ini）的门派前缀；后缀 _001.._006
// = 一内..六内、_101 = 会意功。
var schoolNeiGongPrefixes = map[string]bool{
	"sl": true, "wd": true, "em": true, "gb": true, "tm": true,
	"jl": true, "jy": true, "jz": true, "mj": true, "ts": true,
	"kl": true, "xt": true, "wxp": true,
}

// hiddenForceRageCapAdd 隐世门派各段书目（档位按段内位置冻结，注意
// ng_xd/ng_gm 段一内是 _002）。
var hiddenForceRageCapAdd = map[string]int32{
	// 一内 +30（ng_xd/ng_gm 段首位是 _002，次位 _001 即二内 +40）
	"ng_xd_001": 40, "ng_xd_002": 30,
	"ng_gm_001": 40, "ng_gm_002": 30,
	"ng_cf_001": 30, "ng_cf_002": 40,
	"ng_nl_001": 30, "ng_nl_002": 40,
	"ng_ssg_001": 30, "ng_ssg_002": 40,
	"ng_hs_001": 30, "ng_hs_002": 40,
	"ng_wx_001": 30, "ng_wx_002": 40,
	"ng_dm_001": 30, "ng_dm_002": 40,
	"ng_sjy_001": 30, "ng_sjy_002": 40,
	"ng_xmg_001": 30, "ng_xmg_002": 40,
	"ng_tyhg_001": 30, "ng_tyhg_002": 40,
	"ng_whz_001": 30, "ng_whz_002": 40,
	"ng_bg_001": 30, "ng_bg_002": 40,
}

// forceRageCapAdd 势力内功（各一本，固定值；桃花岛 = 指南「丁氏山庄」+40
// 槽位，见文件头注）。
var forceRageCapAdd = map[string]int32{
	"ng_xj_001": 10, // 徐家庄
	"ng_sj_001": 30, // 金针沈家
	"ng_ws_001": 30, // 万兽山庄
	"ng_tl_001": 40, // 天轮寺
	"ng_qy_001": 40, // 青衣阁
	"ng_yh_001": 40, // 移花宫（明玉神功）
	"ng_wg_001": 40, // 无根门
	"ng_th_001": 40, // 桃花岛（碧波心经）
}

// zuoWangGongID 坐忘功：江湖内功中唯一 +10 档。
const zuoWangGongID = "ng_jh_001"

// jianghuRageCapAdd 江湖内功的默认档（其它）。
const jianghuRageCapAdd int32 = 40

// neiGongRageCapAdd 运行该内功时的怒气上限加成（0 = 无依据，不加成）。
// configID 为 neigong.ini QName（如 ng_wd_001）。
func neiGongRageCapAdd(configID string) int32 {
	if configID == "" {
		return 0
	}
	if add, ok := forceRageCapAdd[configID]; ok {
		return add
	}
	if add, ok := hiddenForceRageCapAdd[configID]; ok {
		return add
	}
	parts := strings.Split(configID, "_")
	school, suffix := "", ""
	if len(parts) == 3 && parts[0] == "ng" {
		school, suffix = parts[1], parts[2]
	}
	// 江湖段：ng_jh_*（坐忘功 +10，其余 +40）与两段式的九阴/九阳神功、造化功。
	if school == "jh" || configID == "ng_jyzj" || configID == "ng_jysg" || configID == "ng_task_001" {
		if configID == zuoWangGongID {
			return 10
		}
		return jianghuRageCapAdd
	}
	if !schoolNeiGongPrefixes[school] {
		return 0
	}
	switch suffix {
	case "001", "002", "003", "004", "005", "006":
		tier, err := strconv.Atoi(suffix)
		if err != nil {
			return 0
		}
		return min(int32(tier)*10, maxRageCapAdd) // 一内 10 .. 六内 60 → 封顶 50
	case "101": // 会意功
		return maxRageCapAdd
	}
	return 0
}
