require("define")
require("share\\skill_define")
require("share\\server_custom_define")
require("share\\skill_static_data_define")

--玩家自动还击技能所在的快捷栏
auto_counter_attack_index = 10

function on_load(self)
	local skill_container = nx_get_child(self, nx_widestr("NormalAttackContainer"))
	if not nx_exists(skill_container) then
		return
	end
	
--[[
	--添加默认的普通攻击技能
	for _, skill_id in pairs(normal_attack_action_id) do
		local skill = nx_create_from_config(skill_container, skill_id)
		if nx_exists(skill) then
			nx_add_data(skill, "NormalAttack", VTYPE_INT)
		end
	end
]]

    for _, skill_table in pairs( normal_attack_action_id ) do
        for _, skill_id in pairs( skill_table ) do
            local skill = nx_create_from_config(skill_container, skill_id)
            if nx_exists(skill) then
                if not nx_find_data( skill, "NormalAttack" ) then
			        nx_add_data(skill, "NormalAttack", VTYPE_INT)
                end
            end
        end
    end
	
	--添加基础技能
	local skill_id_lst = {"zs_zpqf_01", "zs_zpqf_02", "zs_zpqf_03"}
	
	for _, skill_id in pairs(skill_id_lst) do
		local skill = nx_find_skill(self, skill_id)
		if not nx_exists(skill) then
			skill = nx_create_from_config(nx_get_scene(), skill_id)
			if nx_exists(skill) then
				if not nx_add_skill(self, skill) then
					nx_destroy_self(skill)
				end
			end
		end
	end
end

--正式使用伤害类技能前的判断
--返回0表示不能对此对象使用指定的技能
function before_skill_effecttype_compare(self, target, skill)
	
	if not nx_exists(skill) then
		return 1
	end


	-- boss技能释放流程中,玩家对boss使用拼招技能时，boss不受影响。
	if nx_type(self) == TYPE_PLAYER and nx_type(target) == TYPE_NPC then
		-- 判断是否是boss
		local npcscript = nx_string(nx_get_script(target))
		
		if npcscript == "BossNpc" then
				return 2	
		end
		
	end
		
		
	local static_data = nx_query_property(skill, "StaticData")
	
	--技能类型, 0:无类别,1:实招,2:虚招,3:架招
	local effect_type = nx_query_skill_int(skill, static_data, Skill_EffectType)

	--技能configid
	local self_skill_id = nx_get_config(skill)
	
	if nx_type(target) == TYPE_NPC then
		--NPC,特殊处理,如果AI暂停了，不处理
		local stopAi = nx_query_property(target, "StopAI")
		if stopAi > 0 then
			return 1
		end
		
		-- 怪物的技能选择
		local sel_skill = nx_query_property(target, "SelSkill")
		
		-- 先判断怪物是否需要作弊
		local bFraud = false
		
		if not nx_exists(sel_skill) then
			--发送技能相信判断结果的自定义消息
			send_skill_effecttype_compare_result(self, target,  
				self_skill_id, "no", effect_type, -1, 2)
			return 1
		end
		
		-- 得到比试结果
		local result = skill_effecttype_compare(self, target, skill, sel_skill)

		-- 得到NPC的脚本
		local lua_name = nx_query_property(target, "LuaScript")
		if lua_name ~= "" and lua_name ~= nil and lua_name ~= 0  then
			-- 将比试结果发送到AI模块供Boss使用
			nx_execute(nx_string(lua_name), "ai_skill_compare_result", target, sel_skill, nx_int(result))
		end
		
		-- 再发
		if nx_type(self) == TYPE_NPC then
			-- 得到NPC的脚本
			local self_lua_name = nx_query_property(self, "LuaScript")
			if self_lua_name ~= "" and self_lua_name ~= nil and self_lua_name ~= 0  then
				-- 将比试结果发送到AI模块供Boss使用
				nx_execute(nx_string(self_lua_name), "ai_skill_compare_result", self, skill, nx_int(2-result))
			end
		end
		
		-- 结果处理
		if result == 0 then
			-- npc获胜，使用技能
			nx_use_skill_to_target(target, sel_skill)
			return 0
		elseif result == 1 then
			-- 双方都不获胜
			nx_break_any_flow(self)
			nx_break_any_flow(target)
			return 0
		else
			-- 攻击方获胜
			-- 查询锁定时间, 非锁定技能不打断
			local lock_time = nx_number(nx_query_skill_int(skill, static_data, Skill_LockSrcTime))
			
			if lock_time > 0 then
				nx_break_any_flow(target)
			end
			
			return 1
		end
	else
		--玩家
		
		--发送技能相信判断结果的自定义消息
		send_skill_effecttype_compare_result(self, target, 
			self_skill_id, "no", effect_type, -1, 2)
			
		return 1
		
		--[[
		
		--自动还击技能编号
		local row = nx_find_record_value(target, "shortcut_rec", 0, auto_counter_attack_index)
		if row < 0 then
			--未定义
			--发送技能相信判断结果的自定义消息
			send_skill_effecttype_compare_result(self, target, self_skill_id, "no", effect_type, -1, 2)
			return 1
		end
		
		local auto_skill_id = nx_query_record_value(target, "shortcut_rec", row, 2)
		local auto_skill = nx_find_skill(target, auto_skill_id)
		if not nx_exists(auto_skill) then
			--技能未找到
			--发送技能相信判断结果的自定义消息
			send_skill_effecttype_compare_result(self, target, 
				self_skill_id, "no", effect_type, -1, 2)
			return 1
		end
		
		if nx_can_use_skill_to_target(target, self, auto_skill) ~= 1 then
			--target不能使用自动还击技能攻击攻击者
			--发送技能相信判断结果的自定义消息
			send_skill_effecttype_compare_result(self, target, 
				self_skill_id, "no", effect_type, -1, 2)
			return 1
		end
		
		--比较技能
		result = skill_effecttype_compare(self, target, skill, auto_skill)
		if result == 2 then
			--攻击方的技能比自动还击技能厉害, 攻击方的技能继续
			return 1
		else		
			if result == 0 then
				--target的自动还击技能获胜
				--target的当前选择对象切换
				nx_set_property(target, "LastObject", self)
				--target开始使用auto_skill技能
				nx_use_skill_to_target(target, auto_skill)
			elseif result == 1 then
				--与target的自动还击技能打成平手, 执行平手特效
				nx_execute("rull\\skill_effect.lua", "do_deuce_effect", self, target, skill)
			end
		
			--返回0, 终止self的当前技能流程
			return 0
		end
		
		--]]
		
	end
	
	return 1
end

--调用技能脚本修正技能相性相同时比较用的值
function effecttype_compare_value_add(skill, value)
	
	local static_data = nx_query_property(skill, "StaticData")
	
	local lua_script = nx_query_skill_string(skill, static_data, Skill_LuaScript)
	
	if string.len(lua_script) == 0 then
		return value
	end
	
	local added_value = nx_execute(lua_script, "effecttype_compare_value_add", skill, value)
	if added_value == nil then
		return value
	end
	
	return added_value[1]
end

--技能相性比较
--返回0, self输; 返回1, 双方都输; 返回2, self赢
function skill_effecttype_compare(self, target, self_skill, target_skill)
	local result = 2
	
	--技能相性类型
	local self_effect_type, target_effect_type = 0, 0
	local self_static_data = nx_query_property(self_skill, "StaticData")
	local self_effect_type = nx_query_skill_int(self_skill, self_static_data, Skill_EffectType)

	local target_static_data = nx_query_property(target_skill, "StaticData")
	local target_effect_type = nx_query_skill_int(self_skill, target_static_data, Skill_EffectType)
	
	--0:无类别,1:实招,2:虚招,3:架招
	if self_effect_type == target_effect_type and self_effect_type > 0 then
		--相性相等且有相性类别, 比较角色属性
		local self_value, target_value = 0, 0
		
		if self_effect_type == 1 then
			--实招比力量
			self_value = nx_query_property(self, "Str") + nx_query_property(self, "StrAdd")
			target_value = nx_query_property(target, "Str") + nx_query_property(target, "StrAdd")
		elseif self_effect_type == 2 then
			--虚招比敏捷
			self_value = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")
			target_value = nx_query_property(target, "Dex") + nx_query_property(target, "DexAdd")
		elseif self_effect_type == 3 then
			--双方都是架招, 比较力量与敏捷的乘积
			self_value = (nx_query_property(self, "Str") + nx_query_property(self, "StrAdd"))
						* (nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd"))
			target_value = (nx_query_property(target, "Str") + nx_query_property(target, "StrAdd"))
						* nx_query_property(target, "Dex") + nx_query_property(target, "DexAdd")
		end
		
		--调用技能脚本修正对比用的值
		self_value = effecttype_compare_value_add(self_skill, self_value)
		target_value = effecttype_compare_value_add(target_skill, target_value)
		
		if self_value > target_value then
			result = 2
		elseif self_value == target_value then
			result =1
		else
			result =0
		end
	elseif target_effect_type == 0 then
		--所有的类别克制0, 赢
		result = 2
	elseif (self_effect_type == 1 and target_effect_type == 2) or
			(self_effect_type == 2 and target_effect_type == 3) or
			(self_effect_type == 3 and target_effect_type == 1) then
		--1克2, 2克3, 3克1, 赢
		result = 2
	else
		--其他情况, 输
		result = 0
	end

	local self_skill_id = nx_get_config(self_skill)
	local target_skill_id = nx_get_config(target_skill)
	
	--发送技能相信判断结果的自定义消息
	send_skill_effecttype_compare_result(self, target,
				self_skill_id, target_skill_id,
				self_effect_type, target_effect_type, result)
	
	--赢了对方的什么招式,记录下来,计算伤害时可能会用到
	if result == 0 then
		--self输
		nx_set_property(target, "TargetSkill", self_skill)
	elseif result == 2 then
		--self赢
		nx_set_property(self, "TargetSkill", target_skill)
	end
	
	return result
end

--发送技能相信判断结果的自定义消息
--result:0,self输; 1,平手; 2,self赢
--xxx_effect_type:0:无类别,1:实招,2:虚招,3:架招
function send_skill_effecttype_compare_result(self, target,
				self_skill_id, target_skill_id,
				self_effect_type, target_effect_type, result)

	if nx_type(self) == TYPE_PLAYER then
		nx_custom(self, SERVER_CUSTOMMSG_EFFECT, target, "SKILLCOMPARE",
				self_skill_id, target_skill_id, self_effect_type, target_effect_type, result)
	end
	
	if result == 0 then
		result = 2
	elseif result == 2 then
		result = 0
	end
	
	if nx_type(target) == TYPE_PLAYER then
		nx_custom(target, SERVER_CUSTOMMSG_EFFECT, self, "SKILLCOMPARE",
				target_skill_id, self_skill_id, target_effect_type, self_effect_type, result)
	end
end