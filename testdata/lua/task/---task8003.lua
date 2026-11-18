-----------------------------------------
-- @ 脚本名称: task8003.lua
-- @ 脚本功能: ID为8003的任务执行脚本逻辑
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.3.31
-----------------------------------------


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 增加标记记录表一行
	local row = nx_add_record_row_values(player, "StateTaskRec", nx_int(taskid), nx_int(0))
	
	if row < 0 then
		return 0
	end
	
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)

	-- 判断是否已经选择过
	if not nx_find_record(player, "StateTaskRec") then
		return 0
	end
	
	local row = nx_find_record_value(player, "StateTaskRec", 0, nx_int(taskid))

	if row < 0 then
		return 0
	end
	
	-- 判断是否已经触发并选择过, 为0代表没选择过， 1正， 2邪， 3中立
	local flag = nx_query_record_value(player, "StateTaskRec", nx_int(row), 1)
	

	if nx_int(flag) == 0 then
		return 0
	end
	
	return 1
	
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	-- 根据标记给予相应的任务物品
	if not nx_find_record(player, "StateTaskRec") then
		return 0
	end
	
	local row = nx_find_record_value(player, "StateTaskRec", 0, nx_int(taskid))

	if row < 0 then
		return 0
	end
	
	-- 读取标记
	local select_index = nx_query_record_value(player, "StateTaskRec", row, 1)
	
	if nx_int(select_index) == 1 then
		create_obj_to_player(player, nx_string("taskitem_23"))
	elseif nx_int(select_index) == 2 then
		create_obj_to_player(player, nx_string("taskitem_25"))
	elseif nx_int(select_index) == 3 then
		create_obj_to_player(player, nx_string("taskitem_26"))
	end
	
	-- 删除标记行
	nx_remove_record_row(player, "StateTaskRec", row)
	
	return 1
end


-- 创建物品
function create_obj_to_player(player, configid)

		local ToolBox = nx_get_child(player, nx_widestr("ToolBox"))
		
		if not nx_exists(ToolBox) then
			return 0
		end
		
		local item = nx_create_from_config(nx_get_scene(), nx_string(configid))
		
		if not nx_exists(item) then
			return 0
		end
		
		if not nx_container_placeitem(ToolBox, item) then
			nx_destroy_self(item)
			-- 提示空间不足
			nx_sysinfo(target, 1, "9627")
			nx_sysinfo(target, 2, "9627")
		end		
end