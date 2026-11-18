-----------------------------------------
-- @ 脚本名称: task19.lua
-- @ 脚本功能: ID为19的任务执行脚本逻辑
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.3.31
-----------------------------------------
-- 定义全局变量
local funcNpc = "WorldNpc91004"
local createNpc = "Monster10001"


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- Invisible=1是隐身状态, 暂时先不写隐身逻辑
	-- 得到人物坐标位置
	local x,y,z,o = nx_get_position(player)
	nx_set_property(npc, "Invisible", 1)
	
	-- 在附近刷任务怪
	nx_create_obj(nx_string(createNpc), 0, nx_float(x), nx_float(y), nx_float(z), nx_float(o))
	
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
