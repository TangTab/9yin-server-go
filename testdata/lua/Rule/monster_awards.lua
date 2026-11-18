----------------------------------------------------
--说明:		怪物击杀后的奖励逻辑
----------------------------------------------------
require("define")

local TEAM_REC_NAME	= "team_rec"
--[[
内功	功能	内功和怪物之间级别差带来的惩罚
若（玩家激活内功级别-怪物的级别）=1 经验*0.8
若（玩家激活内功级别-怪物的级别）=2 经验*0.65
若（玩家激活内功级别-怪物的级别）=3 经验*0.5
若（玩家激活内功级别-怪物的级别）=4 经验*0.35
若（玩家激活内功级别-怪物的级别）=5 经验*0.25
若（玩家激活内功级别-怪物的级别）=6 经验*0.15

若（怪物的级别-玩家激活内功级别）=0 经验不变
若（怪物的级别-玩家激活内功级别）=1 经验不变
若（怪物的级别-玩家激活内功级别）=2 经验不变
若（怪物的级别-玩家激活内功级别）=3 经验不变
若（怪物的级别-玩家激活内功级别）=4 经验*0.8
若（怪物的级别-玩家激活内功级别）=5 经验*0.65
若（怪物的级别-玩家激活内功级别）=6 经验*0.5
若（怪物的级别-玩家激活内功级别）=7 经验*0.35
若（怪物的级别-玩家激活内功级别）=8 经验*0.25
若（怪物的级别-玩家激活内功级别）=9 经验*0.15
  其它所有情况，得到经验为 1
  
内功	功能	组队打怪，获得修为的修正计算
数值讨论
参数1：队伍中队员的数量
N个队员每个PLAYER得到的经验为(1+5%*N)/N
]]--

--值相等
function lua_number_equal(a, b, dist)

	if (math.abs(a-b) < dist) then
		return true
	else
		return false
	end

end

--根据玩家和NPC之间的以及玩家内功与npc之间的关系计算内功奖励
--参数1: object owner		玩家
--参数2: object npc			被杀NPC
--参数3: object neigong		当前内功ID
--参数4: int inc_exp		原内内功奖励经验
function neigong_awards(owner, npc, inc_exp)
	
	--nx_msgbox("neigong_awards1")
	
	if not nx_exists(owner) then
		return false
	end
	
	if not nx_exists(npc) then
		return false
	end
	
	--奖励内功值最小为1
	if inc_exp < 1 then
		inc_exp = 1
	end
	
	--是否正在队伍里,只有自已一个人队伍不算
	if is_in_team(owner) then
	
		add_team_members_neigong_exp(owner, npc, inc_exp)
	
	else
		--获取自已当前内功应得的经验
		local neigong_config_id = nx_query_property(owner, "CurNeiGong")
		
		--当前内功
		local neigong_obj = nx_find_neigong(owner, neigong_config_id)
		
		if nx_exists(neigong_obj) then
			
			local my_inc_exp = get_my_inc_exp(neigong_obj, npc, inc_exp)
		
			--nx_msgbox("my_inc_exp=" .. nx_string(my_inc_exp))
			--直接添加玩家内功经验
			nx_add_neigong_exp(owner, neigong_config_id, my_inc_exp)
			
			--实际获得的经验小于应得的，则提示惩罚
			if my_inc_exp < inc_exp then
				--内功等级和怪物等级差距导致处罚
				nx_sysinfo(owner, TIPSTYPE_SYSTEMNO, "8007")
			end
		end
	end
	
end

--得到自已内功应得的经验
function get_my_inc_exp(neigong, npc, inc_exp)

	local neigong_level = nx_query_property(neigong, "Level");
	
	local npc_level = nx_query_property(npc, "Level");
	
	local final_inc_exp = 1
	
	--内功等级大于NPC等级
	if neigong_level > npc_level then
		
		local sub_level = neigong_level - npc_level
		
		if sub_level == 1 then
		
			final_inc_exp = nx_int(inc_exp * 0.8)
			
		elseif sub_level == 2 then
		
			final_inc_exp = nx_int(inc_exp * 0.65)
			
		elseif sub_level == 3 then
		
			final_inc_exp = nx_int(inc_exp * 0.5)
			
		elseif sub_level == 4 then
		
			final_inc_exp = nx_int(inc_exp * 0.35)
			
		elseif sub_level == 5 then
		
			final_inc_exp = nx_int(inc_exp * 0.25)
			
		elseif sub_level == 6 then
		
			final_inc_exp = nx_int(inc_exp * 0.15)
		
		end
		
	else --内功等级小于NPC等级
		
		local sub_level = npc_level - neigong_level
		
		if sub_level >= 0 and sub_level <= 3 then
		
			final_inc_exp = inc_exp
		
		elseif sub_level == 4 then
		
			final_inc_exp = nx_int(inc_exp * 0.85)
			
		elseif sub_level == 5 then
		
			final_inc_exp = nx_int(inc_exp * 0.65)
			
		elseif sub_level == 6 then
		
			final_inc_exp = nx_int(inc_exp * 0.55)
		
		elseif sub_level == 7 then
		
			final_inc_exp = nx_int(inc_exp * 0.35)
		
		elseif sub_level == 8 then
		
			final_inc_exp = nx_int(inc_exp * 0.25)
		
		elseif sub_level == 9 then
		
			final_inc_exp = nx_int(inc_exp * 0.15)
		
		end
	
	end
	
	return final_inc_exp
	
end

--检查玩家是否在队伍中
function is_in_team(player)

	local team_id = nx_query_property(player, "TeamID")
	
	if team_id > 0 then
		
		--这里只有一个人在队伍里时，不作组队分配
		local row_count = nx_get_record_rows(player, TEAM_REC_NAME)
		
		if row_count == 1 then
			return false
		end
		
		return true
		
	else
	
		return false
	
	end
	
end

--获取队伍中的所有玩家
function add_team_members_neigong_exp(player, npc, total_exp)

	--计算每个人分到的内功
	local row_count = nx_get_record_rows(player, TEAM_REC_NAME)
	
	local every_one_exp = nx_int((1 + 0.05 * row_count) * total_exp / row_count)
	
	--最少一点
	if every_one_exp < 1 then
		every_one_exp = 1
	end
		
	for i = 1, row_count do
	
		local team_member_name =  nx_query_record_value(player, TEAM_REC_NAME, i-1, 0);
	
		local player_obj = nx_find_player(team_member_name)
	
		--nx_msgbox(nx_string(team_member_name) .. " every_one_exp =" .. nx_string(every_one_exp))
		--检查玩家是否在同服
		if nx_exists(player_obj) then

			--直接添加玩家内功经验
			local neigong_config_id = nx_query_property(player_obj, "CurNeiGong")
			
			--获取组队队员当前内功
			local neigong_obj = nx_find_neigong(player_obj, neigong_config_id)
			
			if nx_exists(neigong_obj) then
				
				--跟居当前内功和NPC重算应该获得的内功经验
				local my_inc_exp = get_my_inc_exp(neigong_obj, npc, every_one_exp)
				
				local neigong_config_id = nx_get_config(neigong_obj)
				--最少一点
				if my_inc_exp < 1 then
					my_inc_exp = 1
				end
				--nx_msgbox("my_inc_exp=" .. nx_string(my_inc_exp))
				--添加内功经验
				nx_add_neigong_exp(player_obj, neigong_config_id, my_inc_exp)
				
				--实际获得的经验小于应得的，则提示惩罚
				if my_inc_exp < every_one_exp then
					--内功等级和怪物等级差距导致处罚
					nx_sysinfo(player_obj, TIPSTYPE_SYSTEMNO, "8007")
				end
			end
		end
	end
end

--击杀怪物后奖励修为值 self：玩家自身，npc：被击杀的怪物， inc_value：击杀后增加的修为值
--nx_inc_faculty（self， value， “” ）
function award_faculty( self, npc, inc_value )
	return nx_inc_faculty(self, inc_value, "kill npc" );
end