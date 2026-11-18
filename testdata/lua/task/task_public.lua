-----------------------------------------
-- @ 脚本名称: task_public.lua
-- @ 脚本功能: 任务公共脚本
-- @ 创建者:   yinxiang
-- @ 创建日期: 2010.7.5
-----------------------------------------
--判断是否有强制区域任务
function check_limit_task_region(self)
	--屏蔽该接口，需要时再还原
	--[[
	--定义强制区域任务的任务编号，后续若还有别的任务在此增加
	local g_table_limit_task = {1,1001, 24001}
	local rows = nx_get_record_rows(self, "Task_Accepted")

	if rows <= 0 then
		return 0
	end

	for i = 1, table.getn(g_table_limit_task) do
		for j = 0, rows-1 do
			if g_table_limit_task[i] == nx_query_record_value(self, "Task_Accepted", j, 0) then
				return 1
			end
		end
	end
	]]
	return 0
end
