-----------------------------------------
-- @ 脚本名称: task2106.lua
-- @ 脚本功能: 2106任务逻辑
-- @ 创建者:   huangmw
-- @ 创建日期: 2010.11.8
-----------------------------------------
require("define")

local NPC_ID = "funnpcclone003002"

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


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	return 1
end

--完成任务某一步后的处理
function on_complete_step(player, task_id, subFuncId, subFuncType, subFuncOrder)
	--完成任务2106的第二步后的处理
	if nx_number(task_id) == 2106 then
		if nx_number(subFuncOrder) == 2 then
			--让NPC开始巡逻，15秒后删除
			local npc = get_obj_around_target(player, NPC_ID, 5)
			
			if npc ~= nil 
				and nx_exists(npc) then
				
				nx_set_property(npc, "LifeTime", 15000)
			end
		end
	end
end

--在目标的周围取得一个指定configid的目标
function get_obj_around_target(target, config_id, range)
	if not nx_exists(target) then
		return
	end
	
	if (not config_id) or (config_id == "") then
		return
	end

	local list = nx_get_around_list(target, nx_int(range), TYPE_NPC, 10, 1)
	
	for i = 1, table.getn(list) do
		local obj = nx_object(list[i])
		
		if nx_exists(obj) then
			if nx_get_config(obj) == nx_string(config_id) then	
				return obj
			end
		end
	end
	
	return
end
