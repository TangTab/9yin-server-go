package gmapi

import (
	"fmt"
	"math"
	"strings"
)

// Validate 是 GM 命令输入校验的唯一权威实现：body 的形状与界值
// 检查、预设/目的地字段的闭名单转换，全部集中在这里。它从
// cmd/protocol-probe/gm_panel.go 的 HTTP 校验大 switch 原样抽取，
// 错误文案逐字保留（面板提示与测试断言依赖这些字符串）。
//
// 分工不变：这里只做输入形状校验（HTTP 侧 400 的来源）；运行态
// 约束（如怒气按当前内功上限饱和）仍由游戏会话循环的执行器处理。
func Validate(body CommandBody) (Command, error) {
	command := Command{Action: body.Action, Silver: body.Silver, SP: body.SP, Value: body.Value, Faction: body.Faction}
	switch body.Action {
	case "set_silver":
		if body.Silver < 0 {
			return command, fmt.Errorf("silver must be non-negative")
		}
	case "set_sp":
		// 校验侧读不到会话内的当前运行内功，按最大可达上限校验；
		// 实际写入仍由执行器按当前有效上限饱和。
		if bound := MaxPlayerSP + MaxNeiGongRageCapAdd; body.SP < 0 || body.SP > bound {
			return command, fmt.Errorf("SP must be between 0 and %d", bound)
		}
	case "add_sp":
		if bound := MaxPlayerSP + MaxNeiGongRageCapAdd; body.SP < 1 || body.SP > bound {
			return command, fmt.Errorf("SP addition must be between 1 and %d", bound)
		}
	case "fill_sp":
	case "set_rage_recovery":
		// 怒气自动回复速率（1s ticker；会话级，重登归零）。
		// 0=关闭；上界 1000 只是护栏，官方满档 5。
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) ||
			float64(body.Value) != float64(int32(body.Value)) || body.Value < 0 || body.Value > 1000 {
			return command, fmt.Errorf("rage recovery must be an integer between 0 and 1000")
		}
	case "set_move_speed":
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) || body.Value < 0.1 || body.Value > 100 {
			return command, fmt.Errorf("move speed must be between 0.1 and 100")
		}
	case "set_gravity":
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) || body.Value < 0.1 || body.Value > 100 {
			return command, fmt.Errorf("gravity must be between 0.1 and 100")
		}
	case "set_respawn_interval":
		// 批量刷新间隔（秒）：0=关闭批量，回到每怪独立 RefreshTime 计时。
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) ||
			float64(body.Value) != float64(int32(body.Value)) || body.Value < 0 || body.Value > 86400 {
			return command, fmt.Errorf("respawn interval must be an integer between 0 and 86400 seconds")
		}
	case "set_corpse_ttl":
		// 尸体清除延迟（秒）：0=尸体保留到重生。
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) ||
			float64(body.Value) != float64(int32(body.Value)) || body.Value < 0 || body.Value > 3600 {
			return command, fmt.Errorf("corpse ttl must be an integer between 0 and 3600 seconds")
		}
	case "set_lootbox_ttl":
		// 杀怪掉落宝箱在场时长（秒）：未拾超时自动消失（0x0E）；0=保留
		// 到拾空。进程级，改值对在场宝箱即刻生效。
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) ||
			float64(body.Value) != float64(int32(body.Value)) || body.Value < 0 || body.Value > 86400 {
			return command, fmt.Errorf("lootbox ttl must be an integer between 0 and 86400 seconds")
		}
	case "set_npc_chase":
		// NPC 自动追击总开关（0=关 1=开，缺省视为 0）。进程级（初值=
		// 启动 env NINEYIN_NPC_CHASE，重启回落）；关闭后仇恨照常持有但
		// 追击位移整段静默，怪坐标恒停出生点。
		if body.Enabled != 0 && body.Enabled != 1 {
			return command, fmt.Errorf("npc chase enabled must be 0 or 1")
		}
		command.Enabled = body.Enabled
	case "spawn_chest":
		// GM 世界宝箱：在当前角色位置生成一只独立宝箱实体（stormbox_10
		// 方言），掉落走回落阶梯一包，拾空/超时消失。
	case "set_autosave_interval":
		// 在线玩家内功/经脉快照的自动存盘周期（秒）：0=关闭（仅断线
		// 与 Ctrl+C 关服时存）。进程级，改动即刻对全部在线会话生效。
		if math.IsNaN(float64(body.Value)) || math.IsInf(float64(body.Value), 0) ||
			float64(body.Value) != float64(int32(body.Value)) || body.Value < 0 || body.Value > 86400 {
			return command, fmt.Errorf("autosave interval must be an integer between 0 and 86400 seconds")
		}
	case "switch_scene":
		destination, parseErr := ParseDestination(body.Destination)
		if parseErr != nil {
			return command, parseErr
		}
		command.Destination = destination
	case "switch_scene_preset":
		destination, presetErr := ScenePresetDestination(body.Preset)
		if presetErr != nil {
			return command, presetErr
		}
		// 会话循环拥有既有的 switch_scene 执行路径。
		// 预设只是校验侧的输入转换，必须在命令
		// 到达该循环之前完成转换。
		command.Action = "switch_scene"
		command.Destination = destination
	case "teleport":
		position, parseErr := ParsePosition(body.Position)
		if parseErr != nil {
			return command, parseErr
		}
		command.Position = position
	case "set_faction":
		if _, factionErr := FactionPresetByKey(body.Faction); factionErr != nil {
			return command, factionErr
		}
	case "set_showequip_type":
		if body.Type < 0 || body.Type > 2 {
			return command, fmt.Errorf("show equip type must be 0 (江湖装), 1 (身份装) or 2 (时装)")
		}
		command.ShowEquipType = body.Type
	case "set_gold":
		if body.Gold < 0 {
			return command, fmt.Errorf("gold must be non-negative")
		}
		command.Gold = body.Gold
	case "set_guanyin":
		if body.GuanYin < 0 {
			return command, fmt.Errorf("guanYin must be non-negative")
		}
		command.GuanYin = body.GuanYin
	case "set_power_level":
		if body.Value < 0 {
			return command, fmt.Errorf("power level must be non-negative")
		}
		command.Value = body.Value
	case "set_alignment":
		// 前端 +v||0 折叠：0=跳过该值，三者全 0 拒绝（Java 同口径）。
		if body.Justice < 0 || body.Evil < 0 || body.FreeBase < 0 {
			return command, fmt.Errorf("alignment values must be non-negative")
		}
		if body.Justice == 0 && body.Evil == 0 && body.FreeBase == 0 {
			return command, fmt.Errorf("set_alignment requires at least one of justice/evil/freeBase")
		}
		command.Justice, command.Evil, command.FreeBase = body.Justice, body.Evil, body.FreeBase
	case "set_pk_mode":
		if body.PkMode < 0 || body.PkMode > 8 {
			return command, fmt.Errorf("pkMode must be between 0 and 8")
		}
		command.PkMode = body.PkMode
	case "set_vitals":
		// maxEneAdd/maxSatAdd 官方表无此二属性，忽略（Java applyVitals 同）。
		values := []int32{body.Ene, body.MaxEne, body.ResumeEne, body.Sat, body.MaxSat, body.PkValue, body.DrunkLevel}
		for _, value := range values {
			if value < 0 {
				return command, fmt.Errorf("vital values must be non-negative")
			}
		}
		if body.Ene == 0 && body.MaxEne == 0 && body.ResumeEne == 0 && body.Sat == 0 &&
			body.MaxSat == 0 && body.PkValue == 0 && body.DrunkLevel == 0 {
			return command, fmt.Errorf("set_vitals requires at least one non-zero vital")
		}
		command.Ene, command.MaxEne, command.ResumeEne = body.Ene, body.MaxEne, body.ResumeEne
		command.Sat, command.MaxSat, command.PkValue, command.DrunkLevel = body.Sat, body.MaxSat, body.PkValue, body.DrunkLevel
	case "set_faculty":
		if body.Value < 0 || body.Value > 2_000_000_000 {
			return command, fmt.Errorf("faculty value must be between 0 and 2000000000")
		}
		command.Value = body.Value
	case "set_five_attributes":
		attrs := []int32{body.AttrStr, body.AttrSta, body.AttrDex, body.AttrIng, body.AttrSpi}
		anySet := false
		for _, attr := range attrs {
			if attr < 0 || attr > BaseAttributeCap {
				return command, fmt.Errorf("attributes must be between 0 and 1000000")
			}
			if attr > 0 {
				anySet = true
			}
		}
		if !anySet {
			return command, fmt.Errorf("set_five_attributes requires at least one non-zero attribute")
		}
		command.AttrStr, command.AttrSta, command.AttrDex, command.AttrIng, command.AttrSpi = body.AttrStr, body.AttrSta, body.AttrDex, body.AttrIng, body.AttrSpi
	case "set_fill_speed":
		if math.IsNaN(float64(body.Multiplier)) || math.IsInf(float64(body.Multiplier), 0) {
			return command, fmt.Errorf("multiplier must be a finite number")
		}
		if body.Base < 1 || body.Base > 1_000_000 {
			return command, fmt.Errorf("fill speed base must be between 1 and 1000000")
		}
		if body.Multiplier < 0.1 || body.Multiplier > 100 {
			return command, fmt.Errorf("fill speed multiplier must be between 0.1 and 100")
		}
		command.FacultyBase, command.FillMultiplier = body.Base, body.Multiplier
	case "set_act_day_limit":
		if body.Value < 0 || body.Value > 999_999 {
			return command, fmt.Errorf("act day limit must be between 0 and 999999 (0 = unlimited)")
		}
		command.Value = body.Value
	case "add_livepoint":
		// 历练发放（×1000 毫按序填槽）：总历练上限 999 点（Java 同界）。
		if body.Points < 1 || body.Points > 999 {
			return command, fmt.Errorf("livepoint points must be between 1 and 999")
		}
		command.Points = body.Points
	case "set_hold_faculty":
		// 潜能修为直设（HoldFaculty@800）。
		if body.Value < 0 || body.Value > 2_000_000_000 {
			return command, fmt.Errorf("hold faculty must be between 0 and 2000000000")
		}
		command.Value = body.Value
	case "set_live_speed_power":
		// 历练转化速度加成 %（LiveSpeedPower@809，卦象效果等价物）。
		if body.Value < 0 || body.Value > 1000 {
			return command, fmt.Errorf("live speed power must be between 0 and 1000")
		}
		command.Value = body.Value
	case "set_zhenqi_cap":
		// 经脉每日真气硬闸（Java GmRemote.validateCommand 同域）：
		// 是否开启 0/1 + 每日阈值 1..10⁷（走 Value 键，与 set_act_day_limit 同模式）。
		if body.Enabled != 0 && body.Enabled != 1 {
			return command, fmt.Errorf("zhenqi cap enabled must be 0 or 1")
		}
		if body.Value < 1 || body.Value > 10_000_000 {
			return command, fmt.Errorf("zhenqi cap value must be between 1 and 10000000")
		}
		command.Enabled = body.Enabled
		command.Value = body.Value
	case "set_jingmai_tier":
		// 经脉周天直设（面板档 36..216；游戏侧按目录上限逐条钳制）。
		if strings.TrimSpace(body.JingMaiId) == "" {
			return command, fmt.Errorf("jingMaiId is required")
		}
		if body.JingMaiTier < 1 || body.JingMaiTier > 216 {
			return command, fmt.Errorf("jingMaiTier must be between 1 and 216")
		}
		command.JingMaiID = strings.TrimSpace(body.JingMaiId)
		command.JingMaiTier = body.JingMaiTier
	case "set_weather":
		if _, presetErr := WeatherPresetByKey(body.Weather); presetErr != nil {
			return command, presetErr
		}
		command.Weather = body.Weather
		command.FadeSec = 10
		if body.FadeSec != 0 {
			if body.FadeSec < 0 || body.FadeSec > 600 {
				return command, fmt.Errorf("weather fade must be between 0 and 600 seconds")
			}
			command.FadeSec = body.FadeSec
		}
	case "ninjiuzhi_set":
		// 九阴志全局门控：目标阶段界值内即受理；「阶段只增不减」
		// 的拒绝语义在游戏侧判定并把原因文本回给面板。
		if body.Stage < JyzStageMin || body.Stage > JyzStageMax {
			return command, fmt.Errorf("stage must be between %d and %d", JyzStageMin, JyzStageMax)
		}
		command.Stage = body.Stage
	case "ninjiuzhi_event":
		if kind := body.EventKind; kind != "kill" && kind != "task" {
			return command, fmt.Errorf("eventKind must be kill or task")
		}
		command.EventKind = body.EventKind
	case "ninjiuzhi_reset", "ninjiuzhi_status":
		// 无负载动作：全服阶段状态经 worldInfo + 会话 LastAction 回显。
	case "popup_gm_panel":
	case "tersure_wash_open":
		// 宝物洗炼界面开窗（on_custom_show_tersure）：new=899000021 新界面
		//（主武学洗）/old=899000008 老界面（属性洗+武学洗）。复用 itemConfig
		// 字段承载形态参数。
		form := strings.TrimSpace(body.ItemConfig)
		if form != "new" && form != "old" {
			return command, fmt.Errorf("itemConfig must be new or old")
		}
		command.ItemConfig = form
	case "custom_probe":
		if body.MsgId < 1 || body.MsgId > 764 {
			return command, fmt.Errorf("probe msgId must be between 1 and 764")
		}
		command.MsgId, command.SubType = body.MsgId, body.SubType
	case "grant_item", "add_item":
		if strings.TrimSpace(body.ItemConfig) == "" {
			return command, fmt.Errorf("itemConfig is required")
		}
		if body.Amount < 1 {
			body.Amount = 1
		}
		command.ItemConfig, command.Amount, command.WearNow = strings.TrimSpace(body.ItemConfig), body.Amount, body.WearNow
	case "equip_card", "equip_hair":
		if body.CardId <= 0 {
			return command, fmt.Errorf("cardId must be a positive skin id")
		}
		command.CardId = body.CardId
	case "grant_identity_suits", "grant_all_fashion", "unequip_card", "grant_all_cards":
		// 无负载动作：目录/存档校验在会话循环内完成。
	case "learn_all_taolu", "forget_all_taolu", "learn_all_neigong":
		// 武学三动作（面板「门派/武学套路」区块）：已学集/目录空态由
		// 会话循环判并回执（如「尚未学会任何武学」），这里无负载可校验。
	case "npc_bubble", "drama_prompt", "red_super_armor",
		"action_state_fishing", "action_state_hsqs05", "action_state_sgmd",
		"action_native_fishing", "action_state_stop":
		// 只测试已注册的客户端原生剧情路由。负载由服务器
		// 持有，因此面板无法提交格式错误的自定义值。
	case "lua_eval":
		// Lua 源码在会话循环内经沙盒执行（超时+指令预算），
		// 这里只做长度闸门，16KB 与 internal/luascript 一致。
		if strings.TrimSpace(body.Code) == "" {
			return command, fmt.Errorf("lua_eval requires non-empty code")
		}
		if len(body.Code) > 16<<10 {
			return command, fmt.Errorf("lua_eval code exceeds 16KB")
		}
		command.Code = body.Code
	case "lua_reload", "task_rebuild":
	case "authored_task_start":
		// 任务调试四动作的数据侧写入口（task_records.go）：接取必填
		// 任务 ID；其余动作 0=最早活跃任务（面板文案同口径）。
		if body.TaskId < 1 {
			return command, fmt.Errorf("task id must be at least 1")
		}
		command.TaskId = body.TaskId
	case "authored_task_submit_ready", "authored_task_remove", "authored_task_reward":
		if body.TaskId < 0 {
			return command, fmt.Errorf("task id must not be negative")
		}
		command.TaskId = body.TaskId
	default:
		return command, fmt.Errorf("unsupported action")
	}
	return command, nil
}
