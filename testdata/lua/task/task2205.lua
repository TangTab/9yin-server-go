-----------------------------------------
-- @ 脚本名称: task.lua
-- @ 脚本功能: 任务默认执行脚本
-- @ 创建者:   yinxiang
-- @ 创建日期: 2010.5.27
-----------------------------------------
require("define")
require("share\\server_custom_define")
-- 每个单独的任务脚本可以复制此模板，修改逻辑


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
    --开始小游戏
	nx_start_game(player)
	
	--设置player是否正在游戏的临时数据
	if not nx_find_data(player, "Gameing") then
	   nx_add_data(player, "Gameing", VTYPE_INT)
	end
    nx_set_data(player, "Gameing", 1)
    
    --设置玩家正在进行的任务，用于游戏流程中停止游戏时调用任务失败接口
    if not nx_find_data(player, "Tasking_Id") then
       nx_add_data(player, "Tasking_Id", VTYPE_INT)
    end
     nx_set_data(player, "Tasking_Id", taskid)      
    
    --接受任务后播放特效
	if not nx_find_data(player, "Effect") then
	   nx_add_data(player, "Effect", VTYPE_STRING)
	end
    nx_set_data(player, "Effect", "npcitem019")
	local effectname = nx_query_data(player, "Effect")
    nx_custom_effect(player, nx_string(effectname)) 
    return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	--停止小游戏
    nx_stop_game(player) 
    
    if nx_find_data(player, "Gameing") then
        nx_remove_data(player, "Gameing")
    end
    
    if nx_find_data(player, "Tasking_Id") then
        nx_remove_data(player, "Tasking_Id")
    end
  
    --消除特效
    if nx_find_data(player, "Effect") then
       local effectname = nx_query_data(player, "Effect")
       nx_custom_destory_effect(player, nx_string(effectname))
       nx_remove_data(player, "Effect") 
    end
	return 1
end

--放弃任务
function on_task_giveup(player, task_id)
	--停止小游戏
    nx_stop_game(player)    

   if nx_find_data(player, "Gameing") then
        nx_remove_data(player, "Gameing")
    end
    
    if nx_find_data(player, "Tasking_Id") then
        nx_remove_data(player, "Tasking_Id")
    end
    
 --任务放弃后，停止拎水特效
    if nx_find_data(player, "Effect") then
       local effectname = nx_query_data(player, "Effect")    
       nx_custom_destory_effect(player, effectname)
       nx_remove_data(player, "Effect") 
    end
	return 1
end


--任务失败之后的处理
function on_task_failed(player, task_id)
	--停止小游戏
    nx_stop_game(player)    

   if nx_find_data(player, "Gameing") then
        nx_remove_data(player, "Gameing")
    end
    
    if nx_find_data(player, "Tasking_Id") then
        nx_remove_data(player, "Tasking_Id")
    end
    
 --任务放弃后，停止拎水特效
    if nx_find_data(player, "Effect") then
       local effectname = nx_query_data(player, "Effect")
       nx_custom_destory_effect(player, nx_string(effectname))
       nx_remove_data(player, "Effect") 
    end
	return 1
end