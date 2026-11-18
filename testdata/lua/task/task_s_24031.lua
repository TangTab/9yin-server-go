-----------------------------------------
-- @ 脚本名称: task_s_24031.lua
-- @ 脚本功能: 24031号任务添加一行进度，最大进度为2
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.3.18
-----------------------------------------

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	-- 当前场景
	local scene = nx_get_scene()
	local scene_id = nx_query_property(scene, "ConfigID")
	local scene_name = nx_query_property(scene, "Name")
	local track_info = "useitem_2f"

	-- 任务进度表增加一行
	local row = nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
	               nx_int(0), nx_int(2), track_info, nx_string(scene_id), track_info)
				   
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

