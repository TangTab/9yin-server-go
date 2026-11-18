-----------------------------------------
-- @ 脚本名称: task\\task_baishi.lua
-- @ 脚本功能: 拜师任务特殊处理逻辑
-- @ 创建者:   liang
-- @ 创建日期: 2010.5.8
-----------------------------------------
require("task\\task_define")

-- 每个单独的任务脚本可以复制此模板，修改逻辑

--该任务的后续任务列表
local g_menpai = {
		[41] = "school_shaolin", --少林
		[42] = "school_wudang",  --武当
		[43] = "school_emei",    --峨嵋
		[44] = "school_gaibang", --丐帮
		[45] = "school_juzitang",--君子堂
		[46] = "school_tangmen", --唐门
		[47] = "school_jilegu",  --极乐谷
		[48] = "school_jinyiwei",--锦衣卫
};
-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	for id, cfg in pairs(g_menpai) do
		if nx_is_doing_task(player, nx_int(id)) then
			return 0;
		end
	end
	local school_value = nx_string(nx_query_property(player, "School"))
		if school_value ~= "" then
			return 0;
		end
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	local table_preid = {24048,22048}
	local rows = table.getn(table_preid)
	for i=1, rows do
		if table_preid[i] ~= nil then

			local row = nx_find_record_value(player, REC_TASK_PLAN, task_rec_id, table_preid[i]);
			if row >= 0 then
				nx_set_record_value(player, "Task_Record", row, task_rec_curnum, 1);
			end
		end
	end

	local menp = g_menpai[taskid];
	if menp ~= nil then
		nx_join_school(player, menp)
	end

	--  自动承接第二章第一个任务
	--local houxu_id = 24049
	--if nx_test_accept_task(player, nx_null_obj(), houxu_id) > 0 then
		--nx_accept_task(player, houxu_id)
	--end
-- 改变进度


	--  自动提交任务
	--local id = 500;
	--if nx_is_doing_task(player, id) then
		--nx_auto_submit_task(player, npc, id)
	--end

	return 1
end
