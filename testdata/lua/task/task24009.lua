-----------------------------------------
-- @ 脚本名称: task24009.lua
-- @ 脚本功能: 任务默认执行脚本
-- @ 创建者:   yinxiang
-- @ 创建日期: 2010.8.12
-----------------------------------------
require("define")
require("share\\server_custom_define")
require("task\\task_define")


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	--获得坐忘功等级
	local neigong_zwg = nx_find_neigong(player, "ng_jh_001")
	local neigong_level = nx_query_property(neigong_zwg, "Level")

	--若坐忘功等级小于2级，则向客户端发送修炼内功按钮提示
	if neigong_level < 2 then
		nx_custom(player, SERVER_CUSTOMMSG_WUXUE_FLASH)

		-- 任务进度表增加一行,进度为0
		--nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
		--		nx_int(0), 1, 0, nx_int(1), "TaskLimit", "", "interact24009", 0, "", 0)
		nx_add_taskrecord_row_values(player, nx_int(taskid), nx_int(8),
				nx_int(0), 1, 0, nx_int(1), "TaskLimit", "", "interact24009")

	else
		-- 任务进度表增加一行，进度为1
		--nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
		--		nx_int(0), 1, 1, nx_int(1), "TaskLimit", "", "interact24009", 0, "", 0)
		nx_add_taskrecord_row_values(player, nx_int(taskid), nx_int(8),
				nx_int(0), 1, 1, nx_int(1), "TaskLimit", "", "interact24009")

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

