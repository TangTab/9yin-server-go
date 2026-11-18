-----------------------------------------
-- @ 脚本名称: task40.lua
-- @ 脚本功能: 40任务特殊处理逻辑
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.4.30
-----------------------------------------

-- 每个单独的任务脚本可以复制此模板，修改逻辑

--该任务的后续任务列表
local g_task = {
		["WorldNpc00212"] = 41,	--少林
		["WorldNpc04401"] = 42,	--武当
		["WorldNpc02604"] = 43,	--峨嵋
		["WorldNpc03003"] = 44,	--丐帮
		["WorldNpc03305"] = 45,	--君子堂
		["WorldNpc02201"] = 46,	--唐门
		["WorldNpc03816"] = 47,	--极乐谷
		["WorldNpc04101"] = 48,	--锦衣卫
};

-- 门派招收弟子限制
local g_task_limit = {
	["WorldNpc00212"] = { sex = 0, ui = "ui_shaolin_sex_limit_info",},
	["WorldNpc02604"] = { sex = 1, ui = "ui_emei_sex_limit_info",},
};
-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 开启新手帮助
	nx_set_property(player, "HelpStepID", "workhelp_00")
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	local config = nx_query_property(npc, "ConfigID");
	local nextTask = g_task[config];
	if nextTask ~= nil then
		nx_accept_task(player, nextTask);
	end
	return 1
end

-- 提交任务之前自定义菜单，如果返回1则会阻止提交任务。
function complete_menu(player, npc, id)
	-- 安全判定
	if not nx_exists(player) or not nx_exists(npc) or id < 0 then
		--nx_msgbox("not obj")
		return 0
	end

	-- 检查npc配置
	local config = nx_query_property(npc, "ConfigID")
	local cfg = g_task_limit[config]
	if cfg == nil then
		return 0
	end

	-- 性别判定
	local sex = nx_query_property(player, "Sex");
	if sex == cfg.sex then
		return 0
	else
		nx_begin_menu(player)
		nx_add_title(player, cfg.ui)
		nx_add_menu(player, 2, "ui_trans_back")
		nx_end_menu(player, npc)
		return 1
	end
end
