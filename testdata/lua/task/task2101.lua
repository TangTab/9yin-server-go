-----------------------------------------
-- @ 脚本名称: task2101.lua
-- @ 脚本功能: 2101任务逻辑
-- @ 创建者:   huangmw
-- @ 创建日期: 2010.11.16
-----------------------------------------
require("define")

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	-- 修改接受表的顺序字段
	local row = nx_find_record_value(player, "Task_Accepted", 0, taskid)
	if row >= 0 then
		nx_set_record_value(player, "Task_Accepted", row, 8, 1)
	end

	--调用接口将玩家传送至分组场景groupscene002，即56号场景
	nx_switch_to_subscene(player, nx_int(1))

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

--完成任务某一步后的处理
function on_complete_step(player, task_id, subFuncId, subFuncType, subFuncOrder)
end
