package main

import "fmt"

// switchSyncIDs 是参考服 pushSwitchSync 的功能开关白名单（Java
// StarterGrants.SWITCH_SYNC_IDS 逐值移植，源自 exe main.switchSyncIDs
// @0x140a2e6a0 与 61653 抓包帧 #1815 互验，203 项升序）。登录 ClientReady
// 收尾一次性 0x1E 414("sync", id×203) 批量开启客户端引擎 SwitchManager。
//
// 2919 = ST_FUNCTION_NEW_C_FORM 在列——人物/装备栏「新旧界面切换」按钮
// 的显示门槛（lua is_show_new_type = CheckSwitchEnable(2919) && 本地
// c_form_style==0；switch_define.lua）。1223（技能动作开关）不在此表：
// "sync" 会清空整棵开关树后重设，1223 由 enableSkillActionSwitch 走
// "reset" 单开，顺序必须 sync 在前。
var switchSyncIDs = []int32{
	0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 13,
	15, 21, 24, 26, 27, 30, 32, 39, 40, 41, 42, 46,
	47, 50, 52, 53, 70, 76, 77, 78, 92, 93, 94, 95,
	96, 97, 98, 102, 104, 106, 107, 109, 112, 113, 144, 148,
	151, 158, 161, 166, 167, 169, 171, 185, 210, 219, 232, 233,
	234, 235, 247, 249, 252, 253, 254, 256, 272, 304, 347, 362,
	363, 385, 387, 500, 501, 502, 503, 504, 505, 509, 520, 613,
	625, 637, 659, 661, 689, 690, 691, 692, 696, 758, 775, 779,
	783, 784, 789, 790, 791, 792, 793, 794, 818, 899, 930, 951,
	970, 976, 980, 981, 986, 997, 1000, 1001, 1002, 1003, 1004, 1005,
	1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1017, 1018, 1020,
	1022, 1027, 1031, 1038, 1045, 1046, 1082, 1086, 1090, 1102, 1112, 1116,
	1161, 1178, 1186, 1193, 1194, 1195, 1196, 1204, 1258, 1262, 1265, 1269,
	1271, 1293, 1301, 1325, 1327, 1344, 1345, 1347, 1348, 1351, 1352, 1353,
	1363, 1368, 1373, 1374, 1379, 1387, 1388, 1389, 2000, 2001, 2002, 2003,
	2004, 2019, 2090, 2091, 2092, 2111, 2146, 2861, 2873, 2880, 2881, 2895,
	2901, 2903, 2906, 2908, 2909, 2911, 2917, 2918, 2919, 2923, 2925,
}

// switchSyncFrame 生成 0x1E 414("sync", id×203) 批量开关帧（Java
// StarterGrants.switchSyncFrame 同形：旧版 0x1E 外壳，首值为消息号，
// 次值为窄字符串 "sync"，其后为开关号列表）。
func switchSyncFrame() ([]byte, error) {
	values := make([]serverCustomValue, 0, len(switchSyncIDs)+1)
	values = append(values, customString("sync"))
	for _, id := range switchSyncIDs {
		values = append(values, customInt(id))
	}
	return serverCustomIntMessage(serverSwitchControlMessage, values...)
}

// pushSwitchSync 登录收尾一次性下发（连接级，0x0C 切图后不重放——
// SwitchManager 是引擎全局状态，不随场景代际清空；Java LoginFlow
// switchSyncPushed 同口径）。缺此帧则 CheckSwitchEnable(2919) 恒假，
// 人物面板「切换新版」按钮不出现、is_show_new_type 恒走旧版界面。
func pushSwitchSync(link sceneMessageConnection) error {
	frame, err := switchSyncFrame()
	if err != nil {
		return fmt.Errorf("encode switch sync: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return fmt.Errorf("write switch sync: %w", err)
	}
	return nil
}
