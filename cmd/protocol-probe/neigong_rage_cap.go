package main

import (
	"strconv"
	"strings"
)

// 怒气上限加成的运行内功分档。
//
// 权威口径 = docs/neigong-beginner-guide.md §六（2026-11-07 转写稿）：
//   宗派内功按当前【运行】的内功决定（非叠加）：一内 +10、二内 +20、
//   三内 +30、四内 +40、五内 +50、六内 +50、会意功 +50；
//   隐世门派 一内 +30、二内 +40；
//   势力 徐家庄 +10、金针沈家/万兽山庄 +30、天轮寺/青衣阁/移花宫/
//   无根门/丁氏山庄 +40；
//   江湖内功 坐忘功 +10、其它 +40。
//
// 书目分层（哪本是一内/二内、哪个 ConfigID 属哪一档）取自客户端
// resources/modern/ini/ui/wuxue/neigong_type.ini（2026-11-07 快照，本表
// 为数据冻结版；该文件为 UI 索引极少变动，若资源升级需人工复核）：
//   - 宗派 13 段（sl/wd/em/gb/tm/jl/jy/jz/mj/ts/kl/xt/wxp）r= 依次
//     _001.._006（=一内..六内）+_101（会意功），段内位置与后缀一致；
//   - 隐世 13 段各两本，档位按段内【位置】而非后缀——ng_xd/ng_gm 两段
//     首项是 _002（一内=ng_xd_002/ng_gm_002），其余段首项为 _001；
//   - 势力 8 段各一本：ini 有 徐家庄(xj)/金针沈家(shenjia)/万兽山庄
//     (wanshou)/天轮寺(tls)/青衣阁(qyg)/移花宫(yihua)/无根门(wugen)/
//     桃花岛(taohua)。指南势力表列"丁氏山庄"而 ini 无此段、恰多出
//     桃花岛段，其余 7 项名字逐一吻合，故桃花岛按该 +40 槽位处理；
//   - 江湖段 ng_jianghu：坐忘功=ng_jh_001（stringname.idres
//     "ng_jh_001=坐忘功" 实证），其余含 九阴真经(ng_jyzj)/九阳神功
//     (ng_jysg)/造化功(ng_task_001) 一律 +40；
//   - 指南未覆盖的段（三皇劲 ng_bh、气剑系 *_qj、ng_qfg 等）与未知
//     ConfigID 返回 0——分档无依据时不虚构加成。
//
// 数值来源说明：varprop 无 MaxSPAdd 键（neigong_varprop.ini 全文无此
// 名），Java 参考服 PlayerProgress 注释同断"无加成来源恒 0"；官方
// 服务器也从不在线上下发 583。本表即加成的唯一权威实现。
//
// 客户端口径已实证（2026-11-07 用户确认）：怒气球分母 = MaxSP(580) +
// MaxSPAdd(583)。

// maxNeiGongRageCapAdd 是任何内功能提供的最大怒气上限加成（五内/
// 六内/会意 +50）。GM HTTP 侧无法读到会话内的当前上限，用它做入参
// 上界（实际值仍由 setSP/addSP 按当前有效上限饱和）。
const maxNeiGongRageCapAdd int32 = 50

// schoolNeiGongPrefixes 是 12+1 个宗派段（neigong_type.ini）的学校
// 前缀；后缀 _001.._006 = 一内..六内、_101 = 会意功。
var schoolNeiGongPrefixes = map[string]bool{
	"sl": true, "wd": true, "em": true, "gb": true, "tm": true,
	"jl": true, "jy": true, "jz": true, "mj": true, "ts": true,
	"kl": true, "xt": true, "wxp": true,
}

// hiddenForceRageCapAdd 隐世门派 13 段全部书目（档位按段内位置冻结，
// 注意 ng_xd/ng_gm 段一内是 _002）。
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

// forceRageCapAdd 势力内功（各一本，固定值；桃花岛=指南"丁氏山庄"
// +40 槽位，见文件头注）。
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

// neiGongRageCapAdd 返回运行该内功时的怒气上限加成（0 表示无依据，
// 不加成）。configID 为 neigong.ini QName（如 ng_wd_001）。
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
	// 江湖段:ng_jh_*(坐忘功 +10,其余 +40)与两段式的九阴/九阳真经、
	// 造化功。
	if school == "jh" || configID == "ng_jyzj" || configID == "ng_jysg" || configID == "ng_task_001" {
		if configID == zuoWangGongID {
			return 10
		}
		return jianghuRageCapAdd
	}
	switch suffix {
	case "001", "002", "003", "004", "005", "006":
		if schoolNeiGongPrefixes[school] {
			tier, err := strconv.Atoi(suffix)
			if err != nil {
				return 0
			}
			add := int32(tier) * 10 // 一内10 .. 六内60 → 封顶50
			if add > maxNeiGongRageCapAdd {
				add = maxNeiGongRageCapAdd
			}
			return add
		}
	case "101": // 会意功
		if schoolNeiGongPrefixes[school] {
			return maxNeiGongRageCapAdd
		}
	}
	return 0
}
