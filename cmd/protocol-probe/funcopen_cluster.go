package main

import "fmt"

// 功能开放簇（native-menu 9/19 构建实证方言，62311 抓包逐帧镜像）：
// 登录链上与九阴志/功能入口配套的三块补集。
//
//   - CollectSkillRec（记录表下标 67，[7,3]）：204 项采集技能全 1 级，
//     native grantCollectSkillRec 在 Task_Accepted 23 行之后、
//     Task_Completed 位图之前铺下（#548-#751）；
//   - quest49 簇：Task_Accepted 第 24 行 task 49（Transbron03A 传音
//     公告任务，28 列全行）+ Task_Record（下标 30）追踪行 + NewRideRec
//     （下标 310）两行公共坐骑（9513/41523）——native #1680-#1682，
//     紧跟 Task_Completed 位图、先于 419/data_record；
//   - 帧序铁律：quest49 簇 → 419 步骤 → data_record → switch sync →
//     414 'gamestep' 收尾（gamestep 必须在 sync 之后，native #1743→
//     #1744；sync 会重置 414 域状态，先发的 gamestep 会被抹掉）。
// collectSkillIDs 是 native-menu grantCollectSkillRec 的 204 项采集技能
// 白名单（62311/49840 两会话逐值一致，全部 1 级）。含 CS_by_xhmdl01..
// 生活采集族与 zs_normal_05..08 等通用族；功能开放链的一环——
// 缺行则生活/采集相关入口不解锁。
var collectSkillIDs = []string{
		"CS_by_xhmdl01", "CS_by_xhmdl02", "CS_by_xhmdl03", "CS_by_xhmdl04", "CS_by_xhmdl05",
		"CS_by_xhmdl06", "CS_by_xhmdl07", "CS_by_xhmdl08", "CS_gb_dgbf01", "CS_gb_dgbf02",
		"CS_gb_dgbf03", "CS_gb_dgbf04", "CS_gb_dgbf05", "CS_gb_dgbf06", "CS_gb_dgbf07",
		"CS_gb_dgbf08", "CS_jh_chz01", "CS_jh_chz02", "CS_jh_chz03", "CS_jh_chz04",
		"CS_jh_chz05", "CS_jh_chz06", "CS_jh_chz07", "CS_jh_cqgf01", "CS_jh_cqgf04",
		"CS_jh_cqgf06", "CS_jh_dgjj01", "CS_jh_dgjj02", "CS_jh_dgjj03", "CS_jh_dgjj04",
		"CS_jh_dgjj05", "CS_jh_dgjj06", "CS_jh_dgjj07", "CS_jh_dgjj08", "CS_jh_dgjj09",
		"CS_jh_hyjf01", "CS_jh_hyjf02", "CS_jh_hyjf03", "CS_jh_hyjf04", "CS_jh_hyjf05",
		"CS_jh_hyjf06", "CS_jh_hyjf07", "CS_jh_hyjf08", "CS_jh_jybgz01", "CS_jh_jybgz02",
		"CS_jh_jybgz03", "CS_jh_jybgz04", "CS_jh_jybgz05", "CS_jh_jybgz06", "CS_jh_jybgz07",
		"CS_jh_jybgz08", "CS_jh_jysg01", "CS_jh_jysg02", "CS_jh_jysg03", "CS_jh_jysg04",
		"CS_jh_jysg05", "CS_jh_jysg06", "CS_jh_jysg07", "CS_jh_jysg07_hide", "CS_jh_jysg08",
		"CS_jh_jysg09", "CS_jh_jysg09_hide", "CS_jh_lbwb02", "CS_jh_mlyqj01_f", "CS_jh_mlyqj01_m",
		"CS_jh_mlyqj02_f", "CS_jh_mlyqj02_m", "CS_jh_mlyqj03_f", "CS_jh_mlyqj03_m", "CS_jh_mlyqj04_f",
		"CS_jh_mlyqj04_m", "CS_jh_mlyqj05_f", "CS_jh_mlyqj05_m", "CS_jh_mlyqj06_f", "CS_jh_mlyqj06_m",
		"CS_jh_mlyqj07_f", "CS_jh_mlyqj07_m", "CS_jh_mlyqj08_f", "CS_jh_mlyqj08_m", "CS_jh_myjf01",
		"CS_jh_myjf02", "CS_jh_myjf03", "CS_jh_myjf04", "CS_jh_myjf05", "CS_jh_myjf06",
		"CS_jh_myjf07", "CS_jh_myjf08", "CS_jh_sfgp01", "CS_jh_sfgp02", "CS_jh_sfgp03",
		"CS_jh_sfgp04", "CS_jh_sfgp05", "CS_jh_sfgp06", "CS_jh_sfgp06_hide", "CS_jh_sfgp07",
		"CS_jh_sfgp07_hide", "CS_jh_sfgp08", "CS_jh_tmby01", "CS_jh_tmby02", "CS_jh_tmby03",
		"CS_jh_tmby04", "CS_jh_tmby05", "CS_jh_tmby06", "CS_jh_tmby06_hide", "CS_jh_tmby07",
		"CS_jh_tmby08", "CS_jh_tslyz01", "CS_jh_tslyz02", "CS_jh_tslyz03", "CS_jh_tslyz04",
		"CS_jh_tslyz05", "CS_jh_tslyz06", "CS_jh_tslyz07", "CS_jh_xlsbz03", "CS_jh_xlsbz04",
		"CS_jh_xlsbz05", "CS_jh_xlsbz06", "CS_jh_xlsbz07", "CS_jh_xlsbz10", "CS_jh_xlsbz13",
		"CS_jh_xlsbz15", "CS_jh_xlsbz18", "CS_jh_xlsbz18_sky", "CS_jh_yqq01", "CS_jh_yqq02",
		"CS_jh_yqq03", "CS_jh_yqq04", "CS_jh_yqq05", "CS_jh_yqq06", "CS_jh_yydf01",
		"CS_jh_yydf02", "CS_jh_yydf03", "CS_jh_yydf04", "CS_jh_yydf05", "CS_jh_yydf06",
		"CS_jh_yydf06_1", "CS_jh_yydf06_hide", "CS_jh_yydf06_hide_1", "CS_jh_yydf07", "CS_jhgp_gxfhj01",
		"CS_jhgp_gxfhj02", "CS_jhgp_gxfhj03", "CS_jhgp_gxfhj04", "CS_jhgp_gxfhj05", "CS_jhgp_gxfhj06",
		"CS_jhgp_gxfhj07", "CS_jhgp_wjj01", "CS_jhgp_wjj02", "CS_jhgp_wjj03", "CS_jhgp_wjj04",
		"CS_jhgp_wjj05", "CS_jhgp_wjj06", "CS_jhgp_wjj07", "CS_jhgp_wjj08", "CS_jhgp_wjj09",
		"CS_jhgp_wjj09_hide", "CS_light_rad_81", "CS_ng_jh_306_01", "CS_ng_jh_306_02", "CS_ng_jh_306_03",
		"CS_ng_jh_507_01", "CS_ng_jh_507_02", "CS_ng_jh_507_03", "CS_sl_lzs01", "CS_sl_lzs02",
		"CS_sl_lzs03", "CS_sl_lzs04", "CS_sl_lzs05", "CS_sl_lzs06", "CS_sl_lzs07",
		"CS_wd_tjq01", "CS_wd_tjq02", "CS_wd_tjq03", "CS_wd_tjq04", "CS_wd_tjq05",
		"CS_wd_tjq06", "CS_wd_tjq07", "CS_wd_tjq08", "CS_xy_shl01", "CS_xy_shl02",
		"CS_xy_shl03", "CS_xy_shl04", "CS_xy_shl05", "CS_xy_shl06", "CS_xy_shl07",
		"CS_xy_shl07_hide", "CS_xy_shl08", "CS_yhwq_hsqs01", "CS_yhwq_hsqs02", "CS_yhwq_hsqs03",
		"CS_yhwq_hsqs04", "CS_yhwq_hsqs05", "CS_yhwq_hsqs06", "CS_yhwq_hsqs07", "riding_start",
		"zs_normal_00", "zs_normal_01", "zs_normal_02", "zs_normal_03", "zs_normal_04",
		"zs_normal_05", "zs_normal_06", "zs_normal_07", "zs_normal_08",
}

const (
	// recordCollectSkill 官方 447 schema：CollectSkillRec [7,3]。
	recordCollectSkill uint16 = 67
	// recordTaskTrace 官方 447 schema：Task_Record（任务追踪行）。
	recordTaskTrace uint16 = 30
)

// grantCollectSkills 铺 native grantCollectSkillRec 的 204 项采集技能
// （全部 1 级）。行形 = ('CS_by_xhmdl01', 1) 窄串 + int。
func grantCollectSkills(conn sceneMessageConnection) error {
	for _, id := range collectSkillIDs {
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordCollectSkill, []recordCell{
			recordString(id), recordInt(1),
		})
		if err != nil {
			return fmt.Errorf("encode CollectSkillRec %s: %w", id, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add CollectSkillRec %s: %w", id, err)
		}
	}
	return nil
}

// pushJyzQuestCluster native #1680-#1682 的 quest49 簇：传音公告任务 49
// 的 Task_Accepted 全行 + Task_Record 追踪行 + NewRideRec 两行公共坐骑。
// 逐列照录 62311 抓包（serverRecordAddCells 对尾列补零与 native 全行
// 逐字节一致）。ownedPacks=玩家 NewRideRec 快照已有的坐骑包（ mounts
// 域已先行整刷），公共坐骑跳过避免重复行。
func pushJyzQuestCluster(conn sceneMessageConnection, ownedPacks map[uint32]bool) error {
	task49, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
		recordInt(49), recordInt(9), recordString("Server"), recordInt(3),
		recordString("Transbron03A"), recordString("title_49"),
		recordInt(0), recordInt(1), recordInt(0), recordInt64(0), recordInt(0), recordByte(0),
		recordString("target_49"), recordString("context_49"),
	})
	if err != nil {
		return fmt.Errorf("encode task 49 accepted row: %w", err)
	}
	if err := conn.WriteFrame(task49); err != nil {
		return fmt.Errorf("add task 49 accepted row: %w", err)
	}
	trace, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskTrace, []recordCell{
		recordInt(49), recordInt(9), recordInt(0), recordInt(1), recordInt(0), recordInt(1),
		recordString("Transbron03A"), recordString(""), recordString("task_TrackInfo_49"),
		recordInt(0), recordString("49_9_Transbron03A"), recordInt64(0), recordInt(0), recordInt(0), recordString(""),
	})
	if err != nil {
		return fmt.Errorf("encode task 49 trace row: %w", err)
	}
	if err := conn.WriteFrame(trace); err != nil {
		return fmt.Errorf("add task 49 trace row: %w", err)
	}
	for _, ride := range []struct {
		packID   uint32
		usedDate uint32
		selected bool
	}{
		{9513, 30, false},
		{41523, 30, true},
	} {
		if ownedPacks[ride.packID] {
			continue
		}
		frame, rideErr := serverRecordAddCells(playerObjectID, playerOwnerID, recordNewRide, []recordCell{
			recordInt(ride.packID), recordInt(ride.usedDate), recordFloat64(0), recordByte(uint32(boolByte(ride.selected))),
		})
		if rideErr != nil {
			return fmt.Errorf("encode NewRideRec %d: %w", ride.packID, rideErr)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("add NewRideRec %d: %w", ride.packID, err)
		}
	}
	return nil
}
