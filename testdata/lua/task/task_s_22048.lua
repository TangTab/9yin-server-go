-----------------------------------------
-- @ 脚本名称: task_s_24056.lua
-- @ 脚本功能: 24056号任务添加一行进度，最大进度为1
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.3.18
-----------------------------------------
require("task\\task_define")
-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	local scene = nx_get_scene()
	local scene_id = nx_query_property(scene, "ConfigID")
	--local origin_id = "sta_start"
	local config_id = "tasklua_22048"
	local school_value = nx_string(nx_query_property(player, "School"))

	if school_value ~= "" then
		nx_add_taskrecord_row_values(player, nx_int(taskid), 8, 0, 0, 1, 1, config_id, nx_string(scene_id), config_id);
		--nx_add_record_row_values(player, REC_TASK_PLAN, nx_int(taskid), 8, 0, 0, 1, 1, config_id, nx_string(scene_id), config_id, 0)
	else
		nx_add_taskrecord_row_values(player, nx_int(taskid), 8, 0, 0, 0, 1, config_id, nx_string(scene_id), config_id);
		--nx_add_record_row_values(player, REC_TASK_PLAN, nx_int(taskid), 8, 0, 0, 0, 1, config_id, nx_string(scene_id), config_id, 0)
	end
	return 1
end

-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	return 1
end

