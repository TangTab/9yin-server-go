-----------------------------------------
-- @ 脚本名称: task2406.lua
-- @ 脚本功能: 2406号任务执行脚本
-- @ 创建者:   huangmw
-- @ 创建日期: 2009.4.14
-----------------------------------------

--可承接任务的道具
local NEED_ITEM_ID = "TaskItemClone009001"
--表示完成任务的临时属性名
local TASK_COMPLETE_DATA = "Task2406Complete"

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	--有其他ID的任务也用这个函数判断承接条件
	--承接条件:背包中是否有NEED_ITEM_ID
	local ToolBox = nx_get_child(player, nx_widestr("ToolBox"))
	
	if nx_exists(ToolBox) then
		local need_item_num = nx_container_get_itemcount(ToolBox, NEED_ITEM_ID)
		
		if need_item_num <= 0 then
			return 0
		end
	end

	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	--这个完成任务的条件只有2406用
	if nx_int(taskid) ~= nx_int(2406) then
		return 0
	end

	if not nx_exists(npc) then
		return 0
	end
	
	if not nx_find_data(npc, TASK_COMPLETE_DATA) then
		return 0
	end
	
	--删除临时数据
	--nx_remove_data(npc, TASK_COMPLETE_DATA)
	
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	return 1
end

