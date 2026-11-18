-----------------------------------------
-- @ 脚本名称: task2302.lua
-- @ 脚本功能: 为民除恶后，师兄出现
-- @ 创建者:   yrc
-- @ 创建日期: 2010.4.19
-----------------------------------------
--播放剧情id
local movie_id = 609;


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
    nx_beigin_movie(player, nx_null_obj(), nx_int(movie_id));
	return 1
end

