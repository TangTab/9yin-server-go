-----------------------------------------
-- @ 脚本名称: task2208.lua
-- @ 脚本功能: 任务默认执行脚本
-- @ 创建者:   yinxiang
-- @ 创建日期: 2010.5.27
-----------------------------------------

-- 每个单独的任务脚本可以复制此模板，修改逻辑


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
	return 1
end

--新增接口，用于放弃任务时特定逻辑的处理
function on_task_giveup(self, task_id)
    local lastObject
    local target_config_id = "NpcClone005051"
      
    --判断player是否绑定了正在治疗的目标npc
    if  nx_find_data(self, "ItemTarget") then
        lastObject = nx_query_data(self, "ItemTarget")
    else
        lastObject = nx_query_property(self, "LastObject")
    end
    if not nx_exists(lastObject) then
        return 1
    end
    
    --设置player状态为普通状态
    nx_set_property(self, "State", "")
    
    --判断该目标npc是否为受伤的村民
    local config_id_now = nx_query_property(lastObject, "ConfigID")
    if config_id_now ~= target_config_id then
         return 0
    end
    
    --判断该目标npc是否正在播放特效
    if nx_find_data(lastObject, "Effect") then
        local effectname = nx_query_data(lastObject, "Effect")
        nx_custom_destory_effect(lastObject, nx_string(effectname))
        
        --删除npc临时数据
        nx_remove_data(lastObject, "Effect")
    end
    
    --删除player身上临时数据
    if nx_find_data(self, "ItemTarget") then
        nx_remove_data(self, "ItemTarget")
    end
    
    return 1 
end
