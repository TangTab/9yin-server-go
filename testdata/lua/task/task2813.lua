-----------------------------------------
-- @ 脚本名称: task2813.lua
-- @ 脚本功能: 2813号任务执行脚本
-- @ 创建者:   luxu
-- @ 创建日期: 2010.6.21
-----------------------------------------
require("define")

--跟随NPC的ID
local CONVOY_NPC_ID = "Convoy2810"
--要删除的NPC ID
local DELETE_NPC_ID = "FunnpcClone017005"

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	--玩家接受2813号任务后,删除原npc,把跟随npc放到原位置上
	if not nx_exists(npc) then
		return
	end
	
	if nx_get_config(npc) ~= DELETE_NPC_ID then
		return
	end
	
	--1500毫秒后删除原npc
	if nx_type(npc) ~= 2 then
		if nx_find_property(npc, "LifeTime") then
			nx_set_property(npc, "LifeTime", nx_int(1500))
		end
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

