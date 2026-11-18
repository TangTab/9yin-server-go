-----------------------------------------
-- @ 脚本名称: task1012.lua
-- @ 脚本功能: ID为1012的任务执行脚本逻辑
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.2.4
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
	
	-- 分别对3个NPC使用道具,NPC分别为
	local npc001 = "WorldNpc00629"
	local npc002 = "WorldNpc00630"
	
	-- 追踪信息1，2
	local trackInfo1 = "useitem1012_1"
	local trackInfo2 = "useitem1012_2"

	-- 任务进度表增加2行
	nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
	               nx_int(0), nx_int(1), npc001, nx_string(scene_id), trackInfo1)
	
	nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(8),
	               nx_int(0), nx_int(1), npc002, nx_string(scene_id), trackInfo2)

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