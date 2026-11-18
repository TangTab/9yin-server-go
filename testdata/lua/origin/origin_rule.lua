-----------------------------------------
-- @ 脚本名称: origin_rule.lua
-- @ 脚本功能: 身份规则脚本文件
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.12.11
-----------------------------------------

-- 身份类型
local ORIGIN_TYPE_BASE		= 0		-- 基础
local ORIGIN_TYPE_BORN		= 1		-- 出生
local ORIGIN_TYPE_JIANGHU	= 2		-- 江湖
local ORIGIN_TYPE_MENPAI	= 3		-- 门派
local ORIGIN_TYPE_LIFE		= 4		-- 生活
local ORIGIN_TYPE_BANGPAI 	= 5		-- 帮派

-- 获得身份的同时加入门派的身份表
local school_origin_table = {
							 	-- 配置方法 代表身份名，代表对应的门派名
								 "sta_slin0101", "school_shaolin",
										
								 "sta_emei0101", "school_emei"
							}


-- 能否接受一个身份挑战
function can_accept_origin(player, origin_id)

	-- 根据策划的要求，当前内部测试版本只开放门派身份
	-- 只有门派身份能够接受， dReAmSun-2009-12-11
	local type = nx_get_origin_type(origin_id)

	if type ~= ORIGIN_TYPE_MENPAI then
		return 0
	end

	return 1
end


-- 接受身份挑战后
function after_accept_origin(player, origin_id)

	return 1
end


-- 能否获得一个身份
function can_get_origin(player, origin_id)

	-- 同样做只有门派可获得的限制
	local type = nx_get_origin_type(origin_id)

	if type ~= ORIGIN_TYPE_MENPAI then
		return 0
	end

	return 1
end


-- 获得身份后
function after_get_origin(player, origin_id)

	-- 任务进度处理 有些任务需要获得身份
	need_origin_task(player, origin_id)
	
	-- 判断是否需要置响应的门派属性
	need_change_school_property(player, origin_id)
	
	-- 1032号任务特殊处理
	do_special_task(player, origin_id)
	
	-- 获得身份后有些任务可以承接，刷新任务标记
	nx_refresh_task_flag(player)

	return 1
end


-- 需求特定身份的任务处理
function need_origin_task(player, origin_id)

	local rows = nx_get_record_rows(player, "Task_Record")

	for count = 1, rows do
		local row = count - 1

		local task_need_origin = nx_query_record_value(player, "Task_Record", nx_int(row), nx_int(4))
		
		-- 获得身份任务进度刷新
		if tostring(task_need_origin) == tostring(origin_id) then
			local need_num = nx_query_record_value(player, "Task_Record", nx_int(row), nx_int(3))
			nx_set_record_value(player, "Task_Record", nx_int(row), nx_int(2), nx_int(need_num))

			--刷新任务进度
			nx_refresh_task_flag(player)
			break
		end
	end
	
	return 1
	
end


-- 判断是否需要变更门派属性
function need_change_school_property(player, origin_id)
	
	-- 判断身份是否在表中
	local count = table.getn(school_origin_table) / 2
	
	for i = 1, count do
		
		-- 身份配置
		local origin_name = school_origin_table[i*2 - 1]
		
		-- 匹配
		if origin_name == origin_id then
		
			-- 设置门派属性
			nx_set_property(player, "School", school_origin_table[i*2])
			return 1
			
		end
		
	end
	
end



-- 一些任务特殊处理
function do_special_task(player, origin_id)

	-- 判断是不是门派初始身份
	local count = table.getn(school_origin_table) / 2
	
	for i = 1, count do
		
		-- 身份配置
		local origin_name = school_origin_table[i*2 - 1]
		
		-- 匹配
		if origin_name == origin_id then
				
			-- 判断是否有1032号任务
			local findrow = nx_find_record_value(player, "Task_Record", nx_int(0), nx_int(1032))
			if findrow < 0 then
				return
			end
			
			-- 判断是否在进度要求范围内
			local nowNum = nx_query_record_value(player, "Task_Record", findrow, nx_int(2))
		
			if nowNum > 0 then
				return
			end
			
			-- 设置完成任务
			nx_set_record_value(player, "Task_Record", findrow, nx_int(2), nx_int(1))
			
			--刷新任务进度
			nx_refresh_task_flag(player)
			
		end
		
	end

end