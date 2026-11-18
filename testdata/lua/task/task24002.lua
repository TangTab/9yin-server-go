-----------------------------------------
-- @ 脚本名称: task515.lua
-- @ 脚本功能: 515任务逻辑
-- @ 创建者:   fengqw
-- @ 创建日期: 2009.12.14
-----------------------------------------
require("define")

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	--对自己隐藏npc
	--if not nx_find_data(player, "TargetNpc") then
	--	nx_add_data(player, "TargetNpc", VTYPE_OBJECT)
	--	nx_set_data(player, "TargetNpc", npc)
	--	nx_hide_npc_to_self(player, npc, 1)
	--end


	local movie_cfg = "born04_QianDengZhen\\qdz001"
	--播放动画ini
	nx_custom(player, "npc_fight_ani_start", movie_cfg)
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)

	--对自己显示npc
	--if nx_find_data(player, "TargetNpc") then
	--	local show_npc = nx_query_data(player, "TargetNpc")
	--	nx_remove_data(player, "TargetNpc")
	--	nx_hide_npc_to_self(player, show_npc, 0)
	--end
	return 1
end

function hide_npc(player, npc, taskid)
	return 1
end

function show_hide_npc(player, npc, taskid)
	return 1
end

