-----------------------------------------
-- @ 脚本名称: task_2803.lua
-- @ 脚本功能: 任务默认执行脚本
-- @ 创建者:   
-- @ 创建日期: 2010.6.25
-----------------------------------------
require("define")
require("share\\server_custom_define")

local buffer_id = "buf_useitemclone017001_01"
local npc_camp = 91

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
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


-- 任务完成后，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	remove_buffer(player)
	return 1
end

--放弃任务
function on_task_giveup(player, task_id)
	remove_buffer(player)
	return 1
end


--任务失败之后的处理
function on_task_failed(player, task_id)
	return 1
end

function remove_buffer(player)
	local buffer = nx_find_buffer(player, buffer_id)
	if nx_exists(buffer) then
		nx_remove_buffer(player, buffer)
	end

	--重置服装
	nx_set_property(player, "WorkCloth", "")

	--恢复敌对关系
	local row = nx_find_record_value(player, "ForceRec", 0, nx_int(npc_camp))
	if row > 0 then	
		nx_remove_record_row(player, "ForceRec", row)
	end
	nx_add_record_row_values(player, "ForceRec", nx_int(npc_camp), 0)

	nx_custom(player, SERVER_CUSTOMMSG_TASK_MSG, 8)
end

