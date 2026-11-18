-----------------------------------------
-- @ 脚本名称: task2102.lua
-- @ 脚本功能: 2102任务逻辑
-- @ 创建者:   huangmw
-- @ 创建日期: 2010.11.8
-----------------------------------------
require("define")

local SEARCH_RANGE = 5

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end

-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	--修改NPC：STEP_1_ID为可攻击
	--local list = nx_get_point_list(npc, 581, 367, 
		--SEARCH_RANGE, TYPE_NPC, 20, 1)
	local group_id = nx_query_property(player, "GroupID")
			
	local list = nx_group_child_list(group_id, TYPE_NPC)
	
	for i = 1, table.getn(list) do
		local obj = nx_object(list[i])
		
		if nx_exists(obj) then
			if nx_get_config(obj) == "funnpcclone003014" then

			end
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
	--1、NPC：funnpcclone003102沿路径groupscene002_001行走1秒后删除
	--local npc = get_obj_around_point(npc, 586, 367, "funnpcclone003102", SEARCH_RANGE)
	local npc = get_group_npc(player, "funnpcclone003102")
			
	if npc ~= nil 
		and nx_exists(npc) then

		nx_set_property(npc, "LifeTime", 1000)
	end
	
	--2、调用接口，将玩家从分组场景中传送出，至苏州的(608.708,2.388,417,1.67)
	nx_switch_locate(player, nx_int(14), 573.802, 2.388, 420.28, 5.581)
	
	return 1
end

--完成任务某一步后的处理
function on_complete_step(player, task_id, subFuncId, subFuncType, subFuncOrder)
	if nx_number(task_id) == 2102 then
		if nx_number(subFuncOrder) == 1 then
			--完成任务2102的第一步后的处理
			
			--播放动画481
			nx_beigin_movie(player, nx_null(), 481)
			
			--删除坐标(586 10 367)附近1M范围内的NPC
			--local npc = get_obj_around_point(player, 586, 367, "funnpcclone003013", SEARCH_RANGE)
			local npc = get_group_npc(player, "funnpcclone003013")
			
			if npc ~= nil 
				and nx_exists(npc) then
				
				nx_set_property(npc, "LifeTime", 1000)
			end
		elseif nx_number(subFuncOrder) == 3 then
			--完成任务2102的第二步后的处理
			
			--1、NPC：funnpcclone003101沿路径groupscene002_002，行走6秒后删除
			--local npc = get_obj_around_point(player, 586, 367, "funnpcclone003101", SEARCH_RANGE)
			local npc = get_group_npc(player, "funnpcclone003101")
			
			if npc ~= nil 
				and nx_exists(npc) then

				nx_set_property(npc, "LifeTime", 6000)
			end
		end
	end
end

--在分组NPC中取得一个指定configid的目标
function get_group_npc(player, config_id)
	if (not config_id) or (config_id == "") then
		return
	end
	
	local group_id = nx_query_property(player, "GroupID")
			
	local list = nx_group_child_list(group_id, TYPE_NPC)
	
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

--在点周围取得一个指定configid的目标
function get_obj_around_point(npc, x, z, config_id, range)
	if (not config_id) or (config_id == "") then
		return
	end

	local list = nx_get_point_list(npc, x, z, nx_int(range), TYPE_NPC, 20, 1)
	
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