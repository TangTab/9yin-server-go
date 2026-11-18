----------------------------------------
-- @ 名称: gather.lua
-- @ 功能: 执行采集物品的脚本逻辑
-- @ 作者: dReAmSun
-- @ 日期: 2009-3-11
----------------------------------------

-- 采集NPC的类型
local GATHER_TYPE_TASK = 1
local GATHER_TYPE_LIFE = 2
local GATHER_TYPE_OTHER = 3
local GATHER_TYPE_DESTROY = 4;		--采集任务(隐身某物体)
local GATHER_TYPE_TASK_NO_RECORD = 5;
local SERVER_CUSTOMMSG_SHENGHUO_ACTION = "shenghuo_action"

local GATHER_RULE_SIZE = 8
local gather_rule = {
	[1] = 0,
	[2] = 49,
	[3] = 99,
	[4] = 149,
	[5] = 199,
	[6] = 249,
	[7] = 274,
	[8] = 300,
}

-- 是否能采集
function can_gather(player, npc)
	--[[
	-- 保护
	if not nx_exists(npc) then
		return 0
	end

	-- 正在回复中,不能采集
	if nx_find_heartbeat(npc, "GatherModule::H_EndEff") then
		return 0
	end

	-- 判断NPC是不是正在被采集中
	local is_using = nx_int(nx_query_property(npc, "IsUsing"))
	if nx_int(is_using) > nx_int(0) then
		nx_sysinfo(player, 2, "9688")
		return 0
	end

	--在骑马时不能采集
	if nx_find_property(player, "Mount") then
		local mount = nx_query_property(player, "Mount");
		if mount ~= nil and string.len(mount) > 0 then
			--出提示 正在骑马，无法进行采集
			nx_sysinfo(player, 2, "1147");
			return 0;
		end
	end

--	-- 判断玩家是否装备了需要的工具
--	local need_tool = nx_query_property(npc, "NeedTool")
--	local ToolType  = nx_query_property(npc, "ToolType")
--	if need_tool ~= nil and need_tool ~= "" then
--	--统一标准生活采集道具全部检测背包栏
--	-- 获得人物装备栏
--		local ToolBox = nx_get_child(player, nx_widestr("ToolBox"))
--
--		-- 判断玩家装备栏里的物品是否是该工具
--		local need_item_num = nx_container_get_itemcount(ToolBox, need_tool)
--		if need_item_num <= 0 then
--			-- 必须装备工具提示
--			nx_sysinfo(player, 2, "1144")
--			return 0
--		end
--	end

	-- 判断采集NPC的类型
	local npctype = nx_query_property(npc, "GatherType")
	local npc_config = nx_query_property(npc, "ConfigID")
	local dropstr = nx_query_property(npc, "DropID")

	if npctype == GATHER_TYPE_TASK then
		-- 任务id, 进度 列表
		local str_taskids = nx_query_property(npc, "GatherPara1")
		local str_orders  = nx_query_property(npc, "GatherPara2")

		-- 防止没有配这个属性
		if "" == str_orders then
			str_orders = "0"
		end

		local taskids_table = nx_split_string(str_taskids, ";")
		local orders_table  = nx_split_string(str_orders, ";")

		if nil == taskids_table or nil == orders_table then
			return 0
		end

		local task_num = table.getn(taskids_table)
		if task_num > table.getn(orders_table) then
			task_num = table.getn(orders_table)
		end

		local b_find = false

		for count = 1, task_num do

			-- 任务物品采集
			local taskid = nx_int(taskids_table[count])
			local order = nx_int(orders_table[count])

			-- 遍历任务记录表,查找是否有该类型任务,找到第一个满足条件的
			local rows = nx_get_record_rows(player, "Task_Record");
			--local findrow = nx_find_record_value(player, "Task_Record", 0, taskid)
			for i = 0, rows-1 do
				local id = nx_query_record_value(player, "Task_Record", i, 0);
				if id == taskid then
					-- 如果有进度要求，不到进度不能采集
					if nx_int(order) > nx_int(0) then
						local cur_order = nx_query_record_value(player, "Task_Record", i, 2)

						if nx_int(cur_order + 1) == nx_int(order) then
							-- 说明有任务当前可以采集
							b_find = true

							break
						end
					else -- 0 为没有进度要求

						local config = nx_query_record_value(player, "Task_Record", i, 4)
						if config == dropstr or npc_config == dropstr then
							local now_num = nx_query_record_value(player, "Task_Record", i, 2)
							local max_num = nx_query_record_value(player, "Task_Record", i, 3)

							-- 说明有任务当前可以采集
							if now_num < max_num then
								b_find = true

								break
							end
						end
					end
				end
			end

			if b_find then
				break;
			end
		end -- for

		if false == b_find then
			return 0
		end
	elseif npctype == GATHER_TYPE_LIFE then

		-- 生活技能采集
		local skill_type = nx_query_property(npc, "GatherPara1")
		local skill_level = nx_int(nx_query_property(npc, "GatherPara2"))

		-- 查找有没有这个技能
		local therow = nx_find_record_value(player, "job_rec", 0, nx_string(skill_type))
		if therow < 0 then
			--nx_sysinfo(player, 2, "11714", nx_string(skill_type))
			if skill_type == "sh_kg" then
				nx_sysinfo(player, 2, "11717")
			elseif skill_type == "sh_qf" then
				nx_sysinfo(player, 2, "11718")
			elseif skill_type == "sh_ys" then
				nx_sysinfo(player, 2, "11719")
			elseif skill_type == "sh_ds" then
				nx_sysinfo(player, 2, "11720")
			elseif skill_type == "sh_yf" then
				nx_sysinfo(player, 2, "11721")
			elseif skill_type == "sh_nf" then
				nx_sysinfo(player, 2, "11722")
			end
			return 0
		end

		-- 查询级别够不够
		local has_level = nx_query_record_value(player, "job_rec", therow, 1)
		local job_name = nx_query_record_value(player, "job_rec", therow, 0)

		local max_level = gather_rule[GATHER_RULE_SIZE]

		for j = 1, GATHER_RULE_SIZE do
			if has_level <= gather_rule[j] then
				max_level = gather_rule[j]
				break
			end
		end

		if max_level < skill_level then
			--nx_sysinfo(player, 2, "11711", job_name, skill_level)
			if skill_type == "sh_kg" then
				nx_sysinfo(player, 2, "11723")
			elseif skill_type == "sh_qf" then
				nx_sysinfo(player, 2, "11724")
			elseif skill_type == "sh_ys" then
				nx_sysinfo(player, 2, "11725")
			elseif skill_type == "sh_ds" then
				nx_sysinfo(player, 2, "11726")
			elseif skill_type == "sh_yf" then
				nx_sysinfo(player, 2, "11727")
			elseif skill_type == "sh_nf" then
				nx_sysinfo(player, 2, "11728")
			end
			return 0
		end

		-- 判断数量
		local left_count = nx_int(nx_query_property(npc, "LeftCount"))
		if left_count <= 0 then
			return 0
		end

		-- 体力限制
		local need_ene = nx_query_property(npc, "NeedEne")
		local have_ene = nx_query_property(player, "Ene")

		if have_ene < need_ene then
			-- 提示体力不足
			nx_sysinfo(player, 2, "1145")
			return 0
		end
	elseif npctype == GATHER_TYPE_OTHER then
		-- 其他类型暂时不设置限制
	elseif npctype == GATHER_TYPE_DESTROY then

		-- 任务id, 进度 列表
		local str_taskids = nx_query_property(npc, "GatherPara1")
		local str_orders  = nx_query_property(npc, "GatherPara2")

		-- 防止没有配这个属性
		if "" == str_orders then
			str_orders = "0"
		end

		local taskids_table = nx_split_string(str_taskids, ";")
		local orders_table  = nx_split_string(str_orders, ";")
		if nil == taskids_table or nil == orders_table then
			return 0
		end

		local task_num = table.getn(taskids_table)
		if task_num > table.getn(orders_table) then
			task_num = table.getn(orders_table)
		end

		local b_find = false

		for count = 1, task_num do

			-- 任务物品采集
			local taskid = nx_int(taskids_table[count])
			local order = nx_int(orders_table[count])

			-- 查找是否有该任务
			local findrow = nx_find_record_value(player, "Task_Record", 0, taskid)

			if nx_int(findrow) >= nx_int(0) then

				-- 如果有进度要求，不到进度不能采集
				if nx_int(order) > nx_int(0) then
					local cur_order = nx_query_record_value(player, "Task_Record", findrow, 2)

					if nx_int(cur_order + 1) == nx_int(order) then
						-- 说明有任务当前可以采集
						b_find = true

						break
					end
				else -- 0 为没有进度要求

					local config = nx_query_record_value(player, "Task_Record", findrow, 4)
					if config == npc_config then
						local now_num = nx_query_record_value(player, "Task_Record", findrow, 2)
						local max_num = nx_query_record_value(player, "Task_Record", findrow, 3)

						-- 说明有任务当前可以采集
						if now_num < max_num then
							b_find = true

							break
						end
					end
				end
			end

		end -- for

		if false == b_find then

			return 0
		end
	elseif npctype == GATHER_TYPE_TASK_NO_RECORD then
		-- 任务id, 进度 列表
		local str_taskids = nx_query_property(npc, "GatherPara1")
		local str_orders  = nx_query_property(npc, "GatherPara2")

		-- 防止没有配这个属性
		if "" == str_orders then
			str_orders = "0"
		end

		local taskids_table = nx_split_string(str_taskids, ";")
		local orders_table  = nx_split_string(str_orders, ";")

		if nil == taskids_table or nil == orders_table then
			return 0
		end

		local task_num = table.getn(taskids_table)
		if task_num > table.getn(orders_table) then
			task_num = table.getn(orders_table)
		end

		local b_find = false

		for count = 1, task_num do
			-- 任务物品采集
			local taskid = nx_int(taskids_table[count])
			-- 如果没有配置或者配置为一个负数，则总是可以采集
			if taskid <= 0 then
				return 1;
			end

			-- 玩家任务表中有该编号的任务，则可以采集
			local row = nx_find_record_value(player, "Task_Record", 0, taskid);
			if row >= 0 then
				return 1;
			end
		end
	end
	--]]
	-- 返回成功
	return 1

end

-- 开始采集
function begin_gather(player)

	-- 播放采集动作
	local npc = nx_query_property(player, "LastObject")

	if not nx_exists(npc) then
		return 0
	end

	-- 得到动作名
	local action_name = nx_query_property(npc, "GatherAction")

	if action_name ~= "" and action_name ~= nil then
	   nx_custom_stateaction(player, action_name)
	end

	return 1
end


-- 结束采集
function end_gather(player)

	-- 将流程名字改变为空
	nx_set_property(player, "FlowCategory", "")

	-- 得到采集NPC
	local npc = nx_query_data(player, "GatherNpc")
	if not nx_exists(npc) then
		return 1
	end

	--动作复原
	nx_custom_stateaction(player, "")

	--手中道具清空
	nx_set_property(player, "ToolEffect", "")

	return 1
end

-- 准备结束处理逻辑
function on_prepare(player)
end

-- 命中后
function on_hit(player)

--[[
	--动作复原
	nx_custom_stateaction(player, "")

	--手中道具清空
	nx_set_property(player, "LifeJobTool", 0)

	-- 得到采集NPC
	local npc = nx_query_data(player, "GatherNpc")
	if not nx_exists(npc) then
		return 0
	end

	-- 设置NPC不在被开采中
	local is_use = nx_int(nx_query_property(npc, "IsUsing"))
	if is_use > 0 then
		nx_set_property(npc, "IsUsing", 0)
	end

	-- 得到采集类型
	local type = nx_int(nx_query_property(npc, "GatherType"))
	local npc_config = nx_query_property(npc, "ConfigID")
	local dropstr = nx_query_property(npc, "DropID")

	if type == GATHER_TYPE_TASK then

		-- 任务物品采集
		local invisible = nx_int(nx_query_property(npc, "Invisible"))
		if invisible > 0 then
			nx_end_gather(player)
			return 0
		end

		-- 得到包裹
		local bag = nx_get_child(player, nx_widestr("ToolBox"))
		if not nx_exists(bag) then
			nx_end_gather(player)
			return 0
		end

		-- 检查能否放下收集物品
		local get_item = nx_query_property(npc, "DropID")
		local item = 0

		if get_item ~= nil and get_item ~= "" then
			item = nx_create_from_config(nx_get_scene(), get_item)
			if not nx_exists(item) then
				nx_end_gather(player)
				return 0
			end

			if nx_container_can_placeitem(bag, item) <= 0 then
				-- 删除
				nx_destroy_self(item)
				nx_sysinfo(player, 2, "9627")
				nx_end_gather(player)
				return 0
			end
		end

		-- 判断需要上缴的物品是否足够
		local items_str = nx_query_property(npc, "LostItems")

		local str_table = nx_split_string(items_str, "|")
		local item_count = table.getn(str_table)

		-- 依次判断是否要上缴的物品足够
		local bEnough = true

		for n = 1, item_count do
			local formal_table = nx_split_string(str_table[n], "-")

			-- 属性
			local configid = formal_table[1]
			local need_amount = nx_int(formal_table[2])

			local have_amount = nx_container_get_itemcount(bag, configid)

			if have_amount < need_amount then
				bEnough = false
			end
		end

		-- 如果不足就结束
		if not bEnough then
			nx_sysinfo(player, 2, "9630")
			nx_end_gather(player)
			return 0
		end

		-- 删除上缴物品
		for n = 1, item_count do
			local formal_table = nx_split_string(str_table[n], "-")

			-- 属性
			local configid = formal_table[1]
			local need_amount = nx_int(formal_table[2])

			-- 删除
			nx_container_remove_itemcount(bag, configid, need_amount)
		end

		-- 给予采集物品
		if item ~= 0 then
			if nx_exists(item) then
				nx_container_placeitem(bag, item)

				-- 提示
				nx_sysinfo(player, 2, "9645", get_item, 1)
			end
		end

		-- 任务id, 进度 列表
		local str_taskids = nx_query_property(npc, "GatherPara1")
		local str_orders  = nx_query_property(npc, "GatherPara2")

		-- 防止没有配这个属性
		if "" == str_orders then
			str_orders = "0"
		end

		local taskids_table = nx_split_string(str_taskids, ";")
		local orders_table  = nx_split_string(str_orders, ";")

		if nil == taskids_table or nil == orders_table then
			nx_end_gather(player)
			return 0
		end


		local task_num = table.getn(taskids_table)
		if task_num > table.getn(orders_table) then
			task_num = table.getn(orders_table)
		end

		local b_find = false

		for count = 1, task_num do
			-- 任务采集的参数意义
			local taskid = nx_int(taskids_table[count])
			local order = nx_int(orders_table[count])
			local findrow = nx_find_record_value(player, "Task_Record", 0, taskid)

			-- 如果角色身上没有这个任务就不能采集
			if nx_int(findrow) >= nx_int(0) then

				if nx_int(order) > nx_int(0) then

					b_find = true

					-- 当前进度
					local now_num = nx_query_record_value(player, "Task_Record", findrow, 2)

					-- 刷新进度目标
					nx_refresh_order_target(player, taskid, findrow, order)

					-- 增加进度
					nx_set_record_value(player, "Task_Record", findrow, 2, nx_int(now_num + 1))

					-- 刷新标记
					nx_refresh_task_flag(player)

				else
					local config = nx_query_record_value(player, "Task_Record", findrow, 4)

					if npc_config == config or dropstr == config then
						local now_num = nx_query_record_value(player, "Task_Record", findrow, 2)
						local max_num = nx_query_record_value(player, "Task_Record", findrow, 3)

						if now_num >= max_num then

						else
							b_find = true

							-- 刷新标记
							nx_refresh_task_flag(player)
						end
					end
				end
			end
		end

		local cur_num = nx_int(nx_query_property(npc, "LeftCount"))
		local max_num = nx_int(nx_query_property(npc, "MaxCount"))
		if cur_num > max_num or cur_num <= 0 then
			nx_end_gather(player)
			return 0
		end

		-- 次数减1
		nx_set_property(npc, "LeftCount", cur_num - 1)
		if cur_num - 1 == 0 then

			local eff = nx_query_property(npc, "EffectsTime")
			if eff > 0 then
				nx_add_countbeat(npc, "GatherModule::H_EndEff", eff, 1)
			else
				nx_destroy_self(npc)
			end
		end

	elseif type == GATHER_TYPE_LIFE then

		-- -- 体力限制
		local need_ene = nx_query_property(npc, "NeedEne")
		local have_ene = nx_query_property(player, "Ene")

		if have_ene < need_ene then
			-- 提示体力不足
			nx_sysinfo(player, 2, "1145")
			-- 结束流程
			nx_end_gather(player)
			return 0
		end

		-- 减少体力
		have_ene = have_ene - need_ene
		nx_set_property(player, "Ene", have_ene)
		nx_sysinfo(player, 2, "11712", need_ene)


		-- 参数意义
		local skill_type = nx_query_property(npc, "GatherPara1")
		local skill_level = nx_int(nx_query_property(npc, "GatherPara2"))
		local skill_shexp = nx_int(nx_query_property(npc, "ShExp"))

		-- 生活技能采集
		local cur_num = nx_int(nx_query_property(npc, "LeftCount"))
		local max_num = nx_int(nx_query_property(npc, "MaxCount"))

		if cur_num > max_num or cur_num <= 0 then
			nx_end_gather(player)
			return 0
		end
--
--		-- 掉落设置
--		local drop_id = nx_string(nx_query_property(npc, "DropID"))
--		if nil == drop_id or 0 == string.len(drop_id) then
--			nx_end_gather(player)
--			return 0
--		end
--
--		-- 创建掉落包
--		local drop_box = nx_get_child(npc, nx_widestr("NpcDropBox"))
--		if nx_exists(drop_box) then
--			nx_destroy_self(drop_box)
--		end
--
--		drop_box = nx_create_container(npc, "NpcDropBox", 16)
--		if not nx_exists(drop_box) then
--			nx_end_gather(player)
--			return 0
--		end
--
--		nx_set_property(drop_box, "Name", nx_widestr("NpcDropBox"))
--
--		local dropitems
--		dropitems = nx_get_normal_dropitems(drop_box, drop_id)
--
--		if 0 == nx_get_child_count(drop_box) then
--			nx_destroy_self(drop_box)
--		end
--
--		-- 拾取框生成
--		-- 这里的 80 是 VIEWPORT_DROP_BOX 枚举 wl
--		if nx_find_viewport(player, 80) then
--			nx_remove_viewport(player, 80)
--		end
--
--		nx_add_viewport(player, 80, drop_box)

		-- 次数减1
		nx_set_property(npc, "LeftCount", cur_num - 1)
		if cur_num - 1 == 0 then
			nx_destroy_self(npc)
		end

	-- 生活掉落添加
	nx_process_life_drop_package( player, npc )

	-- 增加相关职业经验等级
--	nx_inc_job_level(player, skill_type, skill_level, skill_shexp)
	nx_process_life_exp_package( player, npc )


	elseif type == GATHER_TYPE_OTHER then

		-- 宝箱类采集NPC，直接调用掉落库
		local cur_num = nx_int(nx_query_property(npc, "LeftCount"))
		local max_num = nx_int(nx_query_property(npc, "MaxCount"))

		if cur_num > max_num or cur_num <= 0 then
			nx_end_gather(player)
			return 0
		end

		-- 掉落设置
		local drop_id = nx_string(nx_query_property(npc, "DropID"))
		if nil == drop_id or 0 == string.len(drop_id) then
			nx_end_gather(player)
			return 0
		end

		-- 创建掉落包
		local drop_box = nx_get_child(npc, nx_widestr("NpcDropBox"))
		if nx_exists(drop_box) then
			nx_destroy_self(drop_box)
		end

		drop_box = nx_create_container(npc, "NpcDropBox", 16)
		if not nx_exists(drop_box) then
			nx_end_gather(player)
			return 0
		end

		nx_set_property(drop_box, "Name", nx_widestr("NpcDropBox"))

		-- 获得掉落列表
		local dropitems
		dropitems = nx_get_normal_dropitems(drop_box, drop_id)

		if 0 == nx_get_child_count(drop_box) then
			nx_destroy_self(drop_box)
		end

		-- 拾取框生成
		if nx_find_viewport(player, 80) then
			nx_remove_viewport(player, 80)
		end

		nx_add_viewport(player, 80, drop_box)

		-- 次数减1
		nx_set_property(npc, "LeftCount", cur_num - 1)

		if cur_num - 1 == 0 then
			-- 删除
			nx_destroy_self(npc)
		end

	elseif type == GATHER_TYPE_DESTROY then

		-- 任务id, 进度 列表
		local str_taskids = nx_query_property(npc, "GatherPara1")
		local str_orders  = nx_query_property(npc, "GatherPara2")

		-- 防止没有配这个属性
		if "" == str_orders then
			str_orders = "0"
		end

		local taskids_table = nx_split_string(str_taskids, ";")
		local orders_table  = nx_split_string(str_orders, ";")

		if nil == taskids_table or nil == orders_table then
			nx_end_gather(player)
			return 0
		end

		local task_num = table.getn(taskids_table)
		if task_num > table.getn(orders_table) then
			task_num = table.getn(orders_table)
		end

		local b_find = false

		for count = 1, task_num do
			-- 任务采集的参数意义
			local taskid = nx_int(taskids_table[count])
			local order = nx_int(orders_table[count])
			local findrow = nx_find_record_value(player, "Task_Record", 0, taskid)

			-- 如果角色身上没有这个任务就不能采集
			if nx_int(findrow) >= nx_int(0) then

				if nx_int(order) > nx_int(0) then

					b_find = true

					-- 当前进度
					local now_num = nx_query_record_value(player, "Task_Record", findrow, 2)

					-- 刷新进度目标
					nx_refresh_order_target(player, taskid, findrow, order)

					-- 增加进度
					nx_set_record_value(player, "Task_Record", findrow, 2, nx_int(now_num + 1))

					-- 刷新标记
					nx_refresh_task_flag(player);
				else
					local config = nx_query_record_value(player, "Task_Record", findrow, 4)

					if npc_config == config then
						local now_num = nx_query_record_value(player, "Task_Record", findrow, 2)
						local max_num = nx_query_record_value(player, "Task_Record", findrow, 3)

						if now_num >= max_num then

						else
							b_find = true

							-- 增加进度
							nx_set_record_value(player, "Task_Record", findrow, 2, nx_int(now_num + 1))
							-- 刷新标记
							nx_refresh_task_flag(player)
						end
					end
				end
			end
		end

		if false == b_find then
			nx_end_gather(player)
			return 0
		end

		local cur_num = nx_int(nx_query_property(npc, "LeftCount"))
		local max_num = nx_int(nx_query_property(npc, "MaxCount"))
		if cur_num > max_num or cur_num <= 0 then
			nx_end_gather(player)
			return 0
		end

		-- 次数减1
		nx_set_property(npc, "LeftCount", cur_num - 1)
		if cur_num - 1 == 0 then
			local eff = nx_query_property(npc, "EffectsTime");
			if eff > 0 then
				nx_add_countbeat(npc, "GatherModule::H_EndEff", eff, 1);
			else
				-- 刷新心跳
				nx_destroy_self(npc)
			end
		end
	elseif type == GATHER_TYPE_TASK_NO_RECORD then

		-- 任务物品采集
		local invisible = nx_int(nx_query_property(npc, "Invisible"))
		if invisible > 0 then
			nx_end_gather(player)
			return 0
		end

		-- 得到包裹
		local bag = nx_get_child(player, nx_widestr("ToolBox"))
		if not nx_exists(bag) then
			nx_end_gather(player)
			return 0
		end

		-- 检查能否放下收集物品
		local get_item = nx_query_property(npc, "DropID")
		local item = 0

		if get_item ~= nil and get_item ~= "" then
			item = nx_create_from_config(nx_get_scene(), get_item)
			if not nx_exists(item) then
				nx_end_gather(player)
				return 0
			end

			if nx_container_can_placeitem(bag, item) <= 0 then
				-- 删除
				nx_destroy_self(item)
				nx_sysinfo(player, 2, "9627")
				nx_end_gather(player)
				return 0
			end
		end

		-- 判断需要上缴的物品是否足够
		local items_str = nx_query_property(npc, "LostItems")

		local str_table = nx_split_string(items_str, "|")
		local item_count = table.getn(str_table)

		-- 依次判断是否要上缴的物品足够
		local bEnough = true

		for n = 1, item_count do
			local formal_table = nx_split_string(str_table[n], "-")

			-- 属性
			local configid = formal_table[1]
			local need_amount = nx_int(formal_table[2])

			local have_amount = nx_container_get_itemcount(bag, configid)

			if have_amount < need_amount then
				bEnough = false
			end
		end

		-- 如果不足就结束
		if not bEnough then
			nx_sysinfo(player, 2, "9630")
			nx_end_gather(player)
			return 0
		end

		-- 删除上缴物品
		for n = 1, item_count do
			local formal_table = nx_split_string(str_table[n], "-")

			-- 属性
			local configid = formal_table[1]
			local need_amount = nx_int(formal_table[2])

			-- 删除
			nx_container_remove_itemcount(bag, configid, need_amount)
		end

		-- 给予采集物品
		if item ~= 0 then
			if nx_exists(item) then
				nx_container_placeitem(bag, item)

				-- 提示
				nx_sysinfo(player, 2, "9645", get_item, 1)
			end
		end

		local cur_num = nx_int(nx_query_property(npc, "LeftCount"))
		local max_num = nx_int(nx_query_property(npc, "MaxCount"))
		if cur_num > max_num or cur_num <= 0 then
			nx_end_gather(player)
			return 0
		end

		-- 次数减1
		nx_set_property(npc, "LeftCount", cur_num - 1)
		if cur_num - 1 == 0 then
			local eff = nx_query_property(npc, "EffectsTime");
			if eff > 0 then
				nx_add_countbeat(npc, "GatherModule::H_EndEff", eff, 1);
			else
				nx_destroy_self(npc)
			end
		end
	end

	-- 结束流程
	nx_end_gather(player)
--]]
	-- 成功返回
	return 1

end

--查找符合要求(任务编号和任务类型)第一个任务行号
function get_gather_task_row(self, id, type)

	local findrow = nx_find_record_value(player, "Task_Record", 0, taskid)

	local rows = nx_get_record_rows(self, "Task_Record");

	for i = 0, rows-1 do

		local task_type = nx_query_record_value(self, "Task_Record", i, 1);

		local task_id = nx_query_record_value(self, "Task_Record", i, 0);

		if task_id == id and task_type == type then
			return i
		end

	end

	return -1
end
