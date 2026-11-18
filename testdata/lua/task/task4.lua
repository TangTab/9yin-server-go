-----------------------------------------
-- @ 脚本名称: task4.lua
-- @ 脚本功能: ID为4的任务执行脚本逻辑
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.3.10
-----------------------------------------

-- 要高亮显示的武器编号
local task_weapon_config = "sword_1001"


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 获得人物装备栏
	local EquipBox = nx_get_child(player, nx_widestr("EquipBox"))
	if not nx_exists(EquipBox) then
		return 0
	end
	
	-- 获得人物包裹
	local toolbox = nx_get_child(player, nx_widestr("ToolBox"))
	if not nx_exists(toolbox) then
		return 0
	end
	
	-- 22为武器在装备栏中的位置，见ini/rule/EquipBox.ini
	local weapon = nx_get_item(EquipBox, nx_int(22)) 
	
	-- 武器栏没有武器，自动弹出背包，高亮显示武器
	if not nx_exists(weapon) then
		nx_custom(player, nx_string("open_tool_box"))
		
		-- 得到要显示的武器在装备栏中的位置
		local pos = nx_find_item_pos(toolbox, nx_string(task_weapon_config))
		local realpos = nx_int(pos) - nx_int(1)
		
		-- 如果得到正确位置则高亮显示此装备
		if nx_int(pos) > nx_int(0) then
			nx_custom(player, nx_string("show_animation"), nx_string("BoxShow"), nx_int(realpos))
		end
		
	end
	
	-- 将标志位置1表示现在正在处理的任务
	local flag = nx_query_property(player, "NewPlayerFlag1")
	if flag == 0 then
		nx_set_property(player, "NewPlayerFlag1", 1)
	end

	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	-- 获得人物装备栏
	local EquipBox = nx_get_child(player, nx_widestr("EquipBox"))
	if not nx_exists(EquipBox) then
		return 0
	end
	
	-- 17为武器在装备栏中的位置，见ini/rule/EquipBox.ini
	local weapon = nx_get_item(EquipBox, nx_int(22)) 
	
	-- 武器栏没有武器, 则没有完成
	if not nx_exists(weapon) then
		return 0
	end
	
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	-- 将检测标志位置0
	nx_set_property(player, "NewPlayerFlag1", 0)
	
	return 1
end

