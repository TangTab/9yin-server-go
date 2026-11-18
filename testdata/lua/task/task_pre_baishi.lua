-----------------------------------------
-- @ 脚本名称: task_pre_baishi.lua
-- @ 脚本功能: 拜师入门任务特殊处理逻辑
-- @ 创建者:   yinxiang
-- @ 创建日期: 2010.8.19
-----------------------------------------
local g_tab_task = {[1]={id = 40, sence_id = 1, x = 645, z = 671, o = 1,},
					[2]={id = 49, sence_id = 3, x = 463, z = 705, o = 1,},
					--[3]={id = 50, sence_id = 2, x = 645, z = 671, o = 1,},
				}

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	local trace_info = "interact_"..nx_string(taskid)
	-- 任务进度表增加一行,用于与门派指引人交互
	nx_add_taskrecord_row_values(player, nx_int(taskid), nx_int(8),
			nx_int(0), 1, 0, nx_int(1), "", "", trace_info)
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

function on_task_giveup(player, taskid)
--[[
	local len = table.getn(g_tab_task)
	for i=1, len do

		local task_id = 0
		local baishi = g_tab_task[i]
		task_id = baishi.id

		if task_id ~= 0 then
		local complete_times = 0
			complete_times = nx_get_completed_task_times(player, task_id)

			if complete_times > 0 then
				--若放弃任务则自动承接拜师入门任务
				nx_accept_task(player, task_id)

				nx_switch_locate(player, nx_int(baishi.sence_id), nx_float(baishi.x),
				nx_float(0), nx_float(baishi.z), nx_float(baishi.o))
			end
		end
	end
]]
	return 1
end
