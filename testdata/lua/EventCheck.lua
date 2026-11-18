require("define")
require("share\\server_custom_define")
require("share\\logicstate_define")

function check_rate_hp(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0;
	end

	local rate    = nx_number(arg[1]);
	local op      = nx_string(arg[2])
	local HP = nx_query_property(Owner, "HP")
	local MaxHp = nx_query_property(Owner, "MaxHP")
	local temprate = HP/MaxHp
	if op == ">" then
		if temprate > rate then
			return 1
		end
	elseif op == "=" then
		if temprate == rate then
			return 1
		end
	elseif op == "<" then
		if temprate < rate then
			return 1
		end
	elseif op == "!=" then
		if temprate ~= rate then
			return 1
		end
	end
	return 0
end

function check_rate_mp(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0;
	end

	local rate    = nx_number(arg[1]);
	local op      = nx_string(arg[2])
	local MP = nx_query_property(Owner, "MP")
	local MaxMp = nx_query_property(Owner, "MaxMP")
	local temprate = MP/MaxMp
	if op == ">" then
		if temprate > rate then
			return 1
		end
	elseif op == "=" then
		if temprate == rate then
			return 1
		end
	elseif op == "<" then
		if temprate < rate then
			return 1
		end
	elseif op == "!=" then
		if temprate ~= rate then
			return 1
		end
	end
	return 0
end

function check_prop_less(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0;
	end
	local PropName = nx_string(arg[1]);
	local Value    = nx_int(arg[2]);
	local curValue = nx_query_property(Owner, PropName)

	if tonumber(curValue) < tonumber(Value) then
		return 1
	end
	return 0
end

function check_prop_less_byid(Owner,sender, ...)
	if table.getn(arg) < 3 then
		return 0;
	end
	local npcid    = nx_string(arg[1]);
	local PropName = nx_string(arg[2]);
	local Value    = nx_int(arg[3]);

    --取当前场景对象
	local scene = nx_get_scene()
	--去当前场景内npc列表
	local list = nx_get_child_list(scene, 0)

	for n = 1, table.getn(list) do
	    if nx_get_config(list[n]) == npcid then
            local curValue = nx_query_property(list[n], PropName)

            if tonumber(curValue) < tonumber(Value) then
                return 1
            end

            return 0
	    end
	end

	return 0
end

function check_prop_big(Owner, sender,...)
	if table.getn(arg) < 2 then
		return 0;
	end
	local PropName = nx_string(arg[1]);
	local Value    = nx_int(arg[2]);
	local curValue = nx_query_property(Owner, PropName)

	if tonumber(curValue) > tonumber(Value) then
		return 1
	end

	return 0
end

function check_prop_equal(Owner,sender, ...)

	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_prop_equal <PropName> <Value>"));--格式：cobj 编号
		return 0;
	end
	local PropName = nx_string(arg[1]);

	local nType = nx_get_property_type(Owner, PropName)
	local Value = nil
	if nType == 2 then
		Value = nx_int(arg[2]);
	elseif nType == 6 then
		Value = nx_string(arg[2]);
	elseif nType == 7 then
		Value = nx_string(arg[2]);
	else
		return 0;
	end

	local curValue = nx_query_property(Owner, PropName)

	if nType == 2 then

		if nx_int(curValue) == nx_int(Value) then
			return 1
		end

	elseif nType == 6 or nType == 7 then

		if nx_string(curValue) == nx_string(Value) then
			return 1
		end

	else
		return 0
	end

	return 0
end

function check_cur_complete_task_id(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0;
	end

	if not nx_find_data(Owner, "complete_task_id") then
		return 0;
	end

	local taskid = nx_int(arg[1]);
	if taskid == nx_query_data(Owner, "complete_task_id") then
		return 1
	end
	return 0;
end

function check_cur_use_skill(Owner,sender, ...)

	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_cur_use_skill <skillid>"));--格式：cobj 编号
		return 0;
	end

	if not nx_find_data(Owner, "use_skill_id") then
		return 0;
	end

	local skillid = nx_string(arg[1]);
	if skillid == nx_query_data(Owner, "use_skill_id") then
		return 1
	end

	return 0;
end


--判断某config id的机关npc是否都被触发
function check_alltype_MachineNpc(Owner,sender, ...)

	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_cur_use_skill <skillid>"));--格式：cobj 编号
		return 0
	end

	local npcid = nx_string(arg[1]);
	
	-- 取当前场景对象
	local scene = nx_get_scene()
	
	-- 取npc列表
	local npclist = nx_find_child_list_byconfig(nx_get_scene(),npcid,TYPE_NPC)
	
	for i = 1, table.getn(npclist) do
		npc = nx_object(npclist[i])
		if nx_find_property(npc,"SpringFlag") then
			local flag = nx_query_property(npc, "SpringFlag")
			if nx_int(flag) ~= nx_int(1) then
				return 0
			end
		end
	end
	
	-- 取npc列表
	local npclist2 = nx_find_child_list_byconfig(nx_get_scene(),npcid,TYPE_ITEM)
	
	for i = 1, table.getn(npclist2) do
		npc = nx_object(npclist2[i])
		if nx_find_property(npc,"SpringFlag") then
			local flag = nx_query_property(npc, "SpringFlag")
			if nx_int(flag) ~= nx_int(1) then
				return 0
			end
		end
	end
	
	return 1
end

--判断某config id的机关npc是否没有都被触发
function check_not_all_machine_spring(Owner,sender, ...)

	if table.getn(arg) < 1 then
		return 0
	end

	local npcid = nx_string(arg[1]);
	
	--取当前场景对象
	local scene = nx_get_scene()
	
	--去当前场景内npc列表
	local list = nx_get_child_list(scene, 0)
	
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		
		if npcid == nx_get_config(npc) then
		
			if nx_find_property(npc,"SpringFlag") then
			
				local flag = nx_query_property(npc, "SpringFlag")
				
				if nx_int(flag) ~= nx_int(1) then
					return 1
				end

			end
			
		end
	end
	return 0
end

--检测触发范围内玩家数量是否够一定数目
function check_spring_players(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local num = nx_int(arg[1]);
	local range = nx_query_property(Owner, "SpringRange")
	local list = nx_get_around_list(Owner, range, TYPE_PLAYER, 100, 1)
	if table.getn(list) >= num then
		return 1
	end

	return 0
end

--检测触发范围内有某个npc
function check_spring_npc_by_id(Owner,sender, ...)
	if table.getn(arg) < 3 then
		return 0
	end

	local npcid = nx_string(arg[1])
	local num = nx_int(arg[2]);
	local range = nx_int(arg[3])

	local list = nx_get_around_list(Owner, range, TYPE_NPC, 100, 1)
	local count = 0
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		local invisible = nx_query_property(npc, "Invisible")
			
		if npcid == nx_get_config(npc) and
			nx_int(invisible) == nx_int(0) then
			count = count + 1
			if count >= num then
				return 1
			end
		end
	end

	return 0
end

--检测触发范围内数量不足某个npc
function check_spring_npc_by_id_less(Owner,sender, ...)
	if table.getn(arg) < 3 then
		return 0
	end

	local npcid = nx_string(arg[1])
	local num = nx_int(arg[2]);
	local range = nx_int(arg[3])

	local list = nx_get_around_list(Owner, range, TYPE_NPC, 100, 1)
	local count = 0
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		local invisible = nx_query_property(npc, "Invisible")
			
		if npcid == nx_get_config(npc) and
		nx_int(invisible) == nx_int(0) then
			count = count + 1
			if count >= num then
				return 0
			end
		end
	end

	return 1
end

--检测宿主是否存活
function check_owner_is_live(Owner,sender, ...)

	if not nx_exists(Owner) then
		return
	end
	
	if nx_find_property(Owner,"Dead") then
	
		local dead = nx_query_property(Owner,"Dead") 
		
		-- 判断对象Dead属性是否为存活
		if nx_int(dead) == nx_int(0) then
			return 1		
		end
		
	end
	
	return 0
end


--检测范围内某个id的npc存活
function check_range_npc_live(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0
	end

	local npcid = nx_string(arg[1])
	local range = nx_int(arg[2])

	-- 判断npc
	local list = nx_get_around_list(Owner, range, TYPE_NPC, 100, 1)
	
	local count = 0
	
	for i = 1, table.getn(list) do
	
		npc = nx_object(list[i])
		
		if npcid == nx_get_config(npc) then

			local flag = nx_query_property(npc, "Dead")
			local invisible = nx_query_property(npc, "Invisible")
			
			if nx_int(flag) == nx_int(0) and nx_int(invisible) == nx_int(0) then
				return 1
			end
			
			
		end
		
	end

	-- 判断item物件
	local list = nx_get_around_list(Owner, range, TYPE_ITEM, 100, 1)
	
	local count = 0
	
	for i = 1, table.getn(list) do
	
		npc = nx_object(list[i])
		
		if npcid == nx_get_config(npc) then

			local flag = nx_query_property(npc, "Dead")
			local invisible = nx_query_property(npc, "Invisible")
			
			if nx_int(flag) == nx_int(0) and nx_int(invisible) == nx_int(0) then
				return 1
			end
			
		end
		
	end


	return 0
end



--检测 触发者id
function check_sender_id(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local npcid = nx_string(arg[1])
	if npcid == nx_get_config(sender) then
		return 1
	end
	return 0
end

--检测玩家背包是否不够n个道具
function check_item_less(Owner, sender,...)
	if table.getn(arg) < 2 then
		return 0
	end
	if not nx_exists(Owner) then
		return 0
	end
	local toolbox = nx_get_child(sender, nx_widestr("ToolBox"))
	if not nx_exists(toolbox) then
		return 0
	end

	local itemid = nx_string(arg[1])
	local num = nx_int(arg[2])
	local tempnum = nx_container_get_itemcount(toolbox, itemid)
	if tempnum < num then
		return 1
	end

	return 0
end

--检测玩家背包是否多于等于n个道具
function check_item_more(Owner, sender,...)
	if table.getn(arg) < 2 then
		return 0
	end
	if not nx_exists(Owner) then
		return 0
	end
	local toolbox = nx_get_child(sender, nx_widestr("ToolBox"))
	if not nx_exists(toolbox) then
		return 0
	end

	local itemid = nx_string(arg[1])
	local num = nx_int(arg[2])
	local tempnum = nx_container_get_itemcount(toolbox, itemid)

	if tempnum >= num then
		return 1
	end

	return 0
end

function check_in_range(Owner, sender,...)
	if table.getn(arg) < 1 then
		return 0
	end

	if not nx_exists(Owner) then
		return 0
	end
	if not nx_exists(sender) then
		return 0
	end
	local range = nx_int(arg[1])
	local dis = nx_distance(Owner, sender)

	if nx_number(dis) < nx_number(range) then
		return 1
	end
	return 0
end

function check_scene_player_num_more(Owner, sender,...)
	if table.getn(arg) < 1 then
		return 0
	end
	local num = nx_string(arg[1])
	local curid = nx_get_scene_id()
	local tempnum = nx_get_scene_player_count(curid)
	if nx_int(tempnum) >= nx_int(num) then
		return 1
	end
	return 0
end

function check_scene_player_num_less(Owner, sender,...)
	if table.getn(arg) < 1 then
		return 0
	end
	local num = nx_string(arg[1])
	local curid = nx_get_scene_id()
	local tempnum = nx_get_scene_player_count(curid)
	if nx_int(tempnum) < nx_int(num) then
		return 1
	end
	return 0
end

--检测 触发者是否有某个buff
function check_sender_buff(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local buffid = nx_string(arg[1])
	local buff   = nx_find_buffer(sender, buffid)

	if nx_exists(buff) then
		return 1
	end

	return 0
end

--检测 触发者是否没有某个buff
function check_sender_no_buff(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local buffid = nx_string(arg[1])
	local buff   = nx_find_buffer(sender, buffid)

	if not nx_exists(buff) then
		return 1
	end

	return 0
end

--检测 宿主是否有某个buff
function check_owner_buff(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local buffid = nx_string(arg[1])
	local buff   = nx_find_buffer(Owner, buffid)
	if nx_string(buff) ~= "0" then
		return 1
	end
	return 0
end

--检测 宿主是否没有某个buff
function check_owner_no_buff(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local buffid = nx_string(arg[1])
	local buff   = nx_find_buffer(Owner, buffid)
	if nx_string(buff) == "0" then
		return 1
	end
	return 0
end

--判断某config id的npc是否都已经死亡
function check_all_npcdead_byid(Owner,sender, ...)

	if table.getn(arg) < 1 then
		return 0
	end

	local npcid = nx_string(arg[1]);
	--取当前场景对象
	local scene = nx_get_scene()
	--去当前场景内npc列表
	local list = nx_get_child_list(scene, 0)
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		--检测npc“Dead”属性是否满足条件
		if npcid == nx_get_config(npc) then
			if nx_find_property(npc,"Dead") and nx_find_property(npc, "Invisible") then
				if nx_query_property(npc, "Dead") == 0 and
					nx_query_property(npc, "Invisible") == 0 then
					-- nx_msgbox(npcid)
					return 0
				end
			end
		end
	end
	return 1
end

--判断某config id的npc是否有活的
function check_has_npcalive_byid(Owner,sender, ...)

	if table.getn(arg) < 1 then
		return 0
	end

	local npcid = nx_string(arg[1]);
	--取当前场景对象
	local scene = nx_get_scene()
	--去当前场景内npc列表
	local list = nx_get_child_list(scene, 0)
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		--检测npc“Dead”属性是否满足条件
		if npcid == nx_get_config(npc) then
			if nx_find_property(npc,"Dead") and nx_find_property(npc, "Invisible") then
				if nx_query_property(npc, "Dead") == 0 and
					nx_query_property(npc, "Invisible") == 0 then
					return 1
				end
			end
		end
	end
	return 0
end

--判断  某个范围 某configid 的npc是否都已经死亡
function check_around_npcdead_byid(Owner,sender, ...)

	if table.getn(arg) < 2 then
		return 0
	end

	local npcid = nx_string(arg[1]);
	local range = nx_float(arg[2]);
	
	local list = nx_get_around_list(Owner, range, TYPE_NPC, 100, 1)

	for i = 1, table.getn(list) do
	
		npc = nx_object(list[i])
		
		if npcid == nx_get_config(npc) then
		
			local flag = nx_query_property(npc, "Dead")
			if nx_int(flag) == nx_int(0) then

				return 0
			end

		end
		
	end

	local list = nx_get_around_list(Owner, range, TYPE_ITEM, 100, 1)

	for i = 1, table.getn(list) do
	
		npc = nx_object(list[i])
		
		if npcid == nx_get_config(npc) then
			local flag = nx_query_property(npc, "Dead")
			if nx_int(flag) == nx_int(0) then
				return 0
			end
		end
		
	end

	return 1
end


--检测系统时间是否大于一个时间点
function check_time_after(Owner,sender, ...)
		if table.getn(arg) < 6 then
				return 0
		end
		local year = nx_int(arg[1]);
		local month = nx_int(arg[2]);
		local day = nx_int(arg[3]);
		local hour = nx_int(arg[4]);
		local minu = nx_int(arg[5]);
		local sec = nx_int(arg[6]);


		local list1 = nx_get_year_month_day()
		local tyear = list1[1]
		local tmonth = list1[2]
		local tday = list1[3]
		local list2 = nx_get_hour_minute_second()
		local thour = list2[1]
		local tminu = list2[2]
		local tsec = list2[3]


		local time = year*10000000000 + month*100000000 + day*1000000 + hour*10000 + minu*100 + sec
		local ttime = tyear*10000000000 + tmonth*100000000 + tday*1000000 + thour*10000 + tminu*100 + tsec

		if ttime > time then
			return 1
		else
			return 0
		end

end


--检测系统时间是否小于一个时间点
function check_time_before(Owner,sender, ...)
		if table.getn(arg) < 6 then
				return 0
		end
		local year = nx_int(arg[1]);
		local month = nx_int(arg[2]);
		local day = nx_int(arg[3]);
		local hour = nx_int(arg[4]);
		local minu = nx_int(arg[5]);
		local sec = nx_int(arg[6]);

		local list1 = nx_get_year_month_day()
		local tyear = list1[1]
		local tmonth = list1[2]
		local tday = list1[3]
		local list2 = nx_get_hour_minute_second()
		local thour = list2[1]
		local tminu = list2[2]
		local tsec = list2[3]

		local time = year*10000000000 + month*100000000 + day*1000000 + hour*10000 + minu*100 + sec
		local ttime = tyear*10000000000 + tmonth*100000000 + tday*1000000 + thour*10000 + tminu*100 + tsec

		if ttime < time then
			return 1
		else
			return 0
		end

		return 0
end

--检测是一天中的一个时间点（精确到秒）
function check_time(Owner,sender, ...)
		if table.getn(arg) < 4 then
				return 0
		end

		local hour = nx_int(arg[1]);
		local minu = nx_int(arg[2]);
		local sec = nx_int(arg[3]);
		local op  = nx_string(arg[4])

		local list2 = nx_get_hour_minute_second()
		local thour = list2[1]
		local tminu = list2[2]
		local tsec = list2[3]

		local time1 = hour*3600 + minu*60 + sec
		local time2 = thour*3600 + tminu*60 + tsec

		if op == ">" then
			if time2 > time1 then
				return 1
			end
		elseif op == "=" then
			if time2 == time1 then
				return 1
			end
		elseif op == "<" then
			if time2 < time1 then
				return 1
			end
		elseif op == "!=" then
			if time2 ~= time1 then
				return 1
			end
		end

		return 0
end

--比较日
function check_date(Owner,sender, ...)
		if table.getn(arg) < 2 then
				return 0
		end

		local day = nx_int(arg[1]);
		local op = nx_string(arg[2])

		local list1 = nx_get_year_month_day()
		local tyear = list1[1]
		local tmonth = list1[2]
		local tday = list1[3]

		if op == ">" then
			if tday > day then
				return 1
			end
		elseif op == "=" then
			if tday == day then
				return 1
			end
		elseif op == "<" then
			if tday < day then
				return 1
			end
		elseif op == "!=" then
			if tday ~= day then
				return 1
			end
		end

		return 0
end

--比较周日（0为星期天）
function check_day(Owner,sender, ...)
		if table.getn(arg) < 2 then
				return 0
		end

		local day = nx_int(arg[1]);
		local op = nx_string(arg[2])

		local tday = nx_get_day_of_week()

		if op == ">" then
			if tday > day then
				return 1
			end
		elseif op == "=" then
			if tday == day then
				return 1
			end
		elseif op == "<" then
			if tday < day then
				return 1
			end
		elseif op == "!=" then
			if tday ~= day then
				return 1
			end
		end

		return 0
end

--比较月份
function check_month(Owner,sender, ...)
		if table.getn(arg) < 2 then
				return 0
		end

		local month = nx_int(arg[1]);
		local op = nx_string(arg[2])

		local list1 = nx_get_year_month_day()
		local tyear = list1[1]
		local tmonth = list1[2]
		local tday = list1[3]

		if op == ">" then
			if tmonth > month then
				return 1
			end
		elseif op == "=" then
			if tmonth == month then
				return 1
			end
		elseif op == "<" then
			if tmonth < month then
				return 1
			end
		elseif op == "!=" then
			if tmonth ~= month then
				return 1
			end
		end

		return 0
end


--判断使用的道具的id是否符合
function check_use_item_id(Owner,sender, ...)
		if table.getn(arg) < 1 then
				return 0
		end

		if not nx_exists(Owner) then
			return 0
		end

		local check_item_id = nx_string(arg[1]);

		local use_item_id =  nx_query_data(Owner, "NewlyItem")

		if nx_string(use_item_id) == nx_string(check_item_id) then
			return 1
		end

		return 0
end

--检测触发者是否是队长
function check_sender_captain(Owner,sender, ...)
	
	if not nx_exists(sender) then
		return 0
	end

	if not nx_type(sender) == TYPE_PLAYER then
		return 0
	end

	if nx_is_team_captain(sender) then
		return 1
	else
		return 0
	end
	
end

--检测事件拥有者是否是队长
function check_owner_captain(Owner,sender, ...)

	if not nx_exists(Owner) then
		return 0
	end

	if not nx_type(Owner) == TYPE_PLAYER then
		return 0
	end

	if nx_is_team_captain(Owner) then
		return 1
	else
		return 0
	end
	
end


--检测触发者能否从事件拥有者身上接到某个任务
function check_sender_can_accept_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
				return 0
	end
	local taskid = nx_int(arg[1]);
	return nx_test_accept_task(sender, Owner, taskid)
end

--检测触发者是否正在进行某个任务
function check_sender_is_doing_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
				return 0
	end
	local taskid = nx_int(arg[1]);
	if nx_is_doing_task(sender, taskid) then
		return 1
	else
		return 0
	end
end

--检测触发者是否没有进行某个任务
function check_sender_is_not_doing_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
				return 0
	end
	local taskid = nx_int(arg[1]);
	if nx_is_doing_task(sender, taskid) then
		return 0
	else
		return 1
	end
end

--检测触发者是否完成过某个任务
function check_sender_done_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
				return 0
	end
	local taskid = nx_int(arg[1]);
	local times  = nx_get_completed_task_times(sender, taskid)
	if times > 0 then
		return 1
	else
		return 0
	end
end

--检测触发者是否没完成过某个任务
function check_sender_not_done_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
				return 0
	end
	local taskid = nx_int(arg[1]);
	local times  = nx_get_completed_task_times(sender, taskid)
	if times > 0 then
		return 0
	else
		return 1
	end
end

--检测所有玩家是否有完成过某个任务
function check_players_done_task(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end
	
	local taskid = nx_int(arg[1])
	local sceneid = nx_get_scene_id()
	local player_name_list = nx_get_scene_players(sceneid)
-- nx_msgbox(nx_string(table.getn(player_name_list)))
	
	for n = 1, table.getn(player_name_list) do
		local player = nx_find_player(nx_widestr(player_name_list[n]))
		
		if player ~= nil then
			local times  = nx_get_completed_task_times(player, taskid)
-- nx_msgbox(nx_string(times))
			if times > 0 then
				return 1
			end
		end
	end
	
	return 0
end

--检测宿主被选择的对话框id
function check_select_funcid(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0;
	end

	if not nx_find_data(sender, "select_id") then
        --nx_sysinfo (sender, TIPSTYPE_GMINFO, "0", nx_widestr("not"));
		return 0;
	end

	local funcid = nx_int(arg[1]);
	local tempid = nx_query_data(sender, "select_id")

	--// 移除临时数据，防止后来的判断出现错乱
	nx_remove_data (sender, "select_id")

	if funcid == tempid then
        --nx_sysinfo (sender, TIPSTYPE_GMINFO, "0", nx_widestr("exist"));
		return 1
	end

	return 0;
end

--检测宿主的一个临时属性是否等于
function check_owner_data_equal(Owner,sender, ...)
	if not nx_exists(Owner) then
		return
	end

	if table.getn(arg) < 3 then
		return
	end
	local dataname = nx_string(arg[1])
	local datatype = nx_int(arg[2])

	if not nx_find_data(Owner, dataname) then
		return 0
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[3])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[3])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[3])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[3])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[3])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[3])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[3])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[3])
	end
	if nx_query_data(Owner, dataname) == value then
		return 1
	else
		return 0
	end
end

--检测宿主的一个临时属性是否不等于
function check_owner_data_not_equal(Owner,sender, ...)
	if not nx_exists(Owner) then
		return
	end
	
	if table.getn(arg) < 3 then
		return
	end
	local dataname = nx_string(arg[1])
	local datatype = nx_int(arg[2])

	if not nx_find_data(Owner, dataname) then
		return 1
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[3])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[3])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[3])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[3])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[3])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[3])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[3])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[3])
	end
	if nx_query_data(Owner, dataname) == value then
		return 0
	else
		return 1
	end
end

--检测触发者的一个临时属性是否等于
function check_sender_data_equal(Owner,sender, ...)
	if not nx_exists(sender) then
		return
	end
	
	if table.getn(arg) < 3 then
		return
	end
	local dataname = nx_string(arg[1])
	local datatype = nx_int(arg[2])

	if not nx_find_data(sender, dataname) then
		return 0
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[3])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[3])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[3])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[3])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[3])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[3])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[3])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[3])
	end
	if nx_query_data(sender, dataname) == value then
		return 1
	else
		return 0
	end
end

--检测触发者的一个临时属性是否不等于
function check_sender_data_not_equal(Owner,sender, ...)
	if not nx_exists(sender) then
		return
	end
	
	if table.getn(arg) < 3 then
		return
	end
	local dataname = nx_string(arg[1])
	local datatype = nx_int(arg[2])

	--没有这个属性,相当于不等于
	if not nx_find_data(sender, dataname) then
		return 1
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[3])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[3])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[3])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[3])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[3])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[3])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[3])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[3])
	end
	if nx_query_data(sender, dataname) == value then
		return 0
	else
		return 1
	end
end

--检测npc的一个临时属性是否不等于
function check_npc_data_not_equal(Owner,sender, ...)
	if table.getn(arg) < 4 then
		return
	end
	local npcid = nx_string(arg[1]);
	local dataname = nx_string(arg[2])
	local datatype = nx_int(arg[3])
	--取当前场景对象
	local scene = nx_get_scene()
	--去当前场景内npc列表
	local list = nx_get_child_list(scene, 0)
	local npc = nil
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		--检测npc“Dead”属性是否满足条件
		if npcid == nx_get_config(npc) then
            break
		end
	end

    if npc == nil then return 0 end
	--没有这个属性,相当于不等于
	if not nx_find_data(npc, dataname) then
		return 1
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[4])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[4])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[4])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[4])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[4])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[4])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[4])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[4])
	end
	if nx_query_data(npc, dataname) == value then
		return 0
	else
		return 1
	end
end

--检测某ID的NPC一个临时数据是否相等
function check_npc_data_equal(Owner,sender, ...)
	if table.getn(arg) < 4 then
		return 0
	end
	local npcid = nx_string(arg[1]);
	local dataname = nx_string(arg[2])
	local datatype = nx_int(arg[3])
	--取当前场景内npc列表
	local list = nx_get_child_list(nx_get_scene(), 0)
	local npc = nil
	for i = 1, table.getn(list) do
		npc = nx_object(list[i])
		--检测npc configid是否相同
		if npcid == nx_get_config(npc) then
    	break
		end
	end
	
	--不存在此configid对象，返回不相等
  if npc == nil then 
  	return 0 
  end
	--没有这个属性,相当于不等于
	if not nx_find_data(npc, dataname) then
		return 0
	end
	if datatype == VTYPE_BOOL then
		value = nx_boolean(arg[4])
	elseif datatype == VTYPE_INT then
		value = nx_int(arg[4])
	elseif datatype == VTYPE_INT64 then
		value = nx_int64(arg[4])
	elseif datatype == VTYPE_FLOAT then
		value = nx_float(arg[4])
	elseif datatype == VTYPE_DOUBLE then
		value = nx_double(arg[4])
	elseif datatype == VTYPE_STRING then
		value = nx_string(arg[4])
	elseif datatype == VTYPE_WIDESTR then
		value = nx_widestr(arg[4])
	elseif datatype == VTYPE_OBJECT then
		value = nx_object(arg[4])
	end
	if nx_query_data(npc, dataname) == value then
		return 1
	else
		return 0
	end
end

--检测宿主的一个临时属性是否等于触发者ID
function check_owner_data_equal_sender_id(Owner,sender, ...)

	if table.getn(arg) < 3 then
		return
	end

	local dataname = nx_string(arg[1])

	if not nx_find_data(Owner, dataname) then
		return 0
	end

	if nx_string(nx_query_data(Owner, dataname)) == nx_string(sender) then
		return 1
	else
		return 0
	end

end

--检测当前副本进度
function check_clone_progress_by_op(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0;
	end

	local value    = nx_int(arg[1]);
	local op      = nx_string(arg[2])

	local scene = nx_get_scene()
	progress = nx_query_property(scene, "Progress")
	if op == ">" then
		if progress > value then
			return 1
		end
	elseif op == "=" then
		if progress == value then
			return 1
		end
	elseif op == "<" then
		if progress < value then
			return 1
		end
	elseif op == "!=" then
		if progress ~= value then
			return 1
		end
	end
	return 0
end


--测试能否将n个同一id的道具放进触发者背包
function check_sender_can_placeitem(Owner,sender, ...)
	if table.getn(arg) < 2 then
		return 0;
	end
	local itemid = nx_string(arg[1])
	local num = nx_int(arg[2])
	local toolbox = nx_get_child(sender,nx_widestr("ToolBox"));
	if nx_exists(toolbox) then
		local item = nx_create_from_config(nx_get_scene(),itemid);
		if not nx_exists(item) then
			nx_destroy_self(item)
			return 0
		end
		nx_set_property(item, "Amount", num);
		if nx_container_can_placeitem(toolbox,item) > 0 then
			return 1
		else
			return 0
		end
	end
	return 0
end

-- 检测拥有者高度是否满足条件
function check_owner_height(Owner, sender, ...)
	if table.getn(arg) < 2 then
		return 0
	end

	-- 比较符号
	local compare = tostring(arg[1])
	-- 比较高度
	local height  = nx_int(arg[2])

	-- owner位置
	local old_x, old_y, old_z, old_o = nx_get_position(Owner)

	if compare == ">" then
		if nx_int(old_y) > height then
			return 1
		end
	elseif compare == ">=" then
		if nx_int(old_y) >= height then
			return 1
		end
	elseif compare == "<" then
		if nx_int(old_y) < height then
			return 1
		end
	elseif compare == "<=" then
		if nx_int(old_y) <= height then
			return 1
		end
	elseif compare == "==" then
		if nx_int(old_y) == height then
			return 1
		end
	end

	return 0
end

-- 检测触发者某属性是否等于
function check_sender_prop_equal(Owner,sender, ...)

	if table.getn(arg) < 2 then
		return 0;
	end
	local PropName = nx_string(arg[1]);

	local nType = nx_get_property_type(sender, PropName)
	local Value = nil
	if nType == 2 then
		Value = nx_int(arg[2]);
	elseif nType == 6 then
		Value = nx_string(arg[2]);
	elseif nType == 7 then
		Value = nx_string(arg[2]);
	else
		return 0;
	end

	local curValue = nx_query_property(sender, PropName)

	if nType == 2 then

		if nx_int(curValue) == nx_int(Value) then
			return 1
		end

	elseif nType == 6 or nType == 7 then

		if nx_string(curValue) == nx_string(Value) then
			return 1
		end

	else
		return 0
	end

	return 0
end

--判断某个范围内的玩家是否都已经死亡
function check_range_player_dead(Owner,sender, ...)

	if table.getn(arg) < 1 then
		return 0
	end

	local range = nx_float(arg[1]);

	local player_list = nx_get_around_list(Owner, range, TYPE_PLAYER, 100, 1)


	for i = 1, table.getn(player_list) do

		player = nx_object(player_list[i])

		if nx_find_property(player,"Dead") then

			local flag = nx_query_property(player, "Dead")

			if nx_number(flag) == nx_number(0) then
				return 0
			end

		end

	end

	return 1
end

-- 判断触发者是否处于存活状态
function check_sender_not_dead(Owner,sender, ...)

	if not nx_exists(sender) then
		return 0
	end

	if not nx_find_property(sender,"Dead") then
		return 0
	end

	if nx_query_property(sender, "Dead") > 0 then
		return 0
	end

	return 1
end

--检测触发者是否存在临时属性
function check_sender_data_exist(Owner,sender,...)
	
	if table.getn(arg) < 1 then
		return 0
	end
	
	if not nx_exists(sender) then
		return 0
	end
	
	local data = nx_string(arg[1])
	
	if nx_find_data(sender,data) then
		return 1
	end

	return 0
end

--检测触发者是否存在临时属性
function check_sender_data_not_exist(Owner,sender,...)
	
	if table.getn(arg) < 1 then
		return 0
	end
	
	if not nx_exists(sender) then
		return 0
	end
	
	local data = nx_string(arg[1])
	
	if not nx_find_data(sender,data) then
		return 1
	end

	return 0
end

--检测场景是否存在临时属性
function check_scene_data_exist(Owner,sender,...)

	if table.getn(arg) < 1 then
		return 0
	end
	
	local data = nx_string(arg[1])
	local scene = nx_get_scene()
	
	if nx_find_data(scene,data) then
		return 1
	end

	return 0
end

--检测场景是否不存在临时属性
function check_scene_data_not_exist(Owner,sender,...)

	if table.getn(arg) < 1 then
		return 0
	end
	
	local data = nx_string(arg[1])
	local scene = nx_get_scene()
	
	if not nx_find_data(scene,data) then
		return 1
	end

	return 0
end


--检测触发者和宿主是否处于战斗关系
function check_owner_hostile_to_player(Owner,sender,...)
	
	return 0

end


-- 检测owner是否是在正在播放剧情
function owner_is_in_movie(Owner, sender,...)

	-- 保护
	if not nx_exists(Owner) then
		return 0
	end

	if nx_type(Owner) ~= TYPE_PLAYER then
		return 0
	end
	
	if nx_find_property(Owner, "InMovie") then
		local in_movie = nx_query_property(Owner, "InMovie")
		if nx_number(in_movie) ~= nx_number(0) then
			return 1		
		end
	end			

	return 0
end

-- 检测sender是否是在正在播放剧情
function sender_is_in_movie(Owner, sender,...)

	-- 保护
	if not nx_exists(sender) then
		return 0
	end

	if nx_type(sender) ~= TYPE_PLAYER then
		return 0
	end
	
	if nx_find_property(sender, "InMovie") then
		local in_movie = nx_query_property(sender, "InMovie")
		if nx_number(in_movie) ~= nx_number(0) then
			return 1		
		end
	end			

	return 0
end

--检测触发者id是否在ID表中
function check_sender_id_inarray(Owner,sender, ...)
	if table.getn(arg) < 1 then
		return 0
	end

	local str_npcid = nx_string(arg[1])
	local str_id_arr = nx_split_comma_string(str_npcid)
	local id_num = table.getn(str_id_arr) 
	
	if id_num <= 0 then
		return
	end
	
	for i = 1, id_num do
		if str_id_arr[i] == nx_get_config(sender) then
			return 1
		end
	end

	return 0
end
