-----------------------------------------
-- @ 脚本名称: task10147.lua
-- @ 脚本功能: 10147任务逻辑
-- @ 创建者:   wwl
-- @ 创建日期: 2009.10.27
-----------------------------------------


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	   -- 使用物品
    local npc_name = nx_string("使用毛笔宣纸")
	function create_obj_to_player(player, configid)

		local ToolBox = nx_get_child(player, nx_widestr("ToolBox"))
		
		if not nx_exists(ToolBox) then
			return 0
	    end
		local pos = nx_get_free_pos(toolbox)
		local realpos = pos - 1	
		
		
		if not nx_container_placeitem(ToolBox, item) then
			nx_destroy_self(item)
			-- 提示空间不足
			nx_sysinfo(target, 1, "9627")
			nx_sysinfo(target, 2, "9627")
		end		
end	


        -- 增加标记记录表一行
	local row = nx_add_record_row_values(player, "StateTaskRec", nx_int(taskid), nx_int(2))
	
	if row < 0 then
		return 0
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

