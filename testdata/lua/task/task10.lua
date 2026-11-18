-----------------------------------------
-- @ 脚本名称: task10.lua
-- @ 脚本功能: ID为10的任务执行脚本逻辑
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.3.11
-----------------------------------------


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 自动弹出该包裹界面
	nx_custom(player, nx_string("open_tool_box"))
	
	-- -- 获得人物包裹
	local toolbox = nx_get_child(player, nx_widestr("ToolBox"))
	if not nx_exists(toolbox) then
		return 0
	end
	
	-- 得到要显示的物品taskitem_4在装备栏中的位置,找个空闲位置
	local pos = nx_get_free_pos(toolbox)
	local realpos = pos - 1
		
	-- 如果得到正确位置则高亮显示
	if nx_int(pos) > nx_int(0) then
		nx_custom(player, nx_string("show_animation"), nx_string("BoxShow"), nx_int(realpos))
	end

	-- 当前场景
	local scene = nx_get_scene()
	local scene_id = nx_query_property(scene, "ConfigID")
	local scene_name = nx_query_property(scene, "Name")
	
	local npc_name = nx_string("食用怀抱鲤")
	
    -- 任务进度表增加一行
	local row = nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
	               nx_int(0), nx_int(1), npc_name, nx_string(scene_id), nx_string(scene_name))
				   
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

