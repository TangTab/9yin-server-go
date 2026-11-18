-----------------------------------------
-- @ 脚本名称: task2818.lua
-- @ 脚本功能: 2818获得门派身份脚本
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.4.17
-----------------------------------------
-- 每个单独的任务脚本可以复制此模板，修改逻辑
local g_pre_task = 2817
local movie_id = 909

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	if nx_is_doing_task(player, g_pre_task) or
		nx_test_record_flag(player, "Task_Completed", g_pre_task) then
		return 1;
	end

	return 0
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	nx_beigin_movie(player, nx_null_obj(), nx_int(movie_id));
	return 1;
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	
	return 1
end

