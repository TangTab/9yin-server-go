require("share\\itemtype_define")
require("define")
require("share\\pk_define")
require("share\\logicstate_define")
require("share\\item_static_data_define")

-- 能否使用一个物品的判断
function can_use(item, target)

	-- 保护
	if not nx_exists(item) then
		return 0
	end

	return 1
end

--使用物品的通用脚本
--返回0使用成功
--返回1使用失败
function on_use(item, target)
	-- 有效性检查：该物品是否存在
	if not nx_exists(item) then
		return 0
	end
		
    local selftype = nx_query_property(target,"Type")--自身类型,是NPC还是player
	local skill_id = nx_query_property(item, "Skill1")
    --主动触发技能
    local skill
	if selftype == TYPE_PLAYER  and skill_id~=nil and skill_id~=""then 
	    skill =get_temp_skillobj(target,skill_id)
	else
	    skill = nx_create_from_config(nx_get_scene(), skill_id)
	end
	
    if  nx_exists(skill) then
           local result =  nx_use_skill_to_target(target, skill)
           if result ~=1 then--使用失败
               return 0
           end
    end
	--物品类型
	local itemtype = nx_query_property(item, "ItemType")
	--[[
	elseif itemtype == ITEMTYPE_HUICHENG_TONGPIAO then	--旅店通票回家
		return useitem_return(item, target)--]]
	if itemtype == ITEMTYPE_COMPOSITE_FORMULA then --合成配方
		return useitem_formula(item, target)
	--elseif itemtype == ITEMTYPE_GOHOME then -- 使用回城道具
	--	return useitem_go_home(item, target)
	elseif itemtype == ITEMTYPE_ADD_TASK then -- 触发任务
		return useitem_trigger_task(item, target)
	elseif itemtype == ITEMTYPE_DEBUF then
        return useitem_remove_debuffer(item, target)
    elseif itemtype == ITEMTYPE_FRAGMENT_COMPOSE then --碎片合成
        return useitem_fragment_compose(item, target)
	end
	
	use_item_disappear(item)
	
	return 0
end

function useitem_fragment_compose(item, target)
    nx_fragment_compose(target, item)
    return 0
end

function useitem_remove_debuffer(item, target)

	local array = {}
	
   local buffer1 = nx_find_buffer(target, "buffer_dead01_01")
    if nx_string(buffer1) ~= "0" then
	    array[1] = buffer1
	end
	
	local buffer2 = nx_find_buffer(target,"buffer_dead02_01")
    if nx_string(buffer2) ~= "0" then
	    array[2] = buffer2
	end
	
	local buffer3 = nx_find_buffer(target, "buffer_dead03_01")
    if nx_string(buffer3) ~= "0" then
	    array[3] = buffer3
	end
	
	local buffer4 = nx_find_buffer(target,"buffer_dead04_01")
    if nx_string(buffer4) ~= "0" then
	    array[4] = buffer4
	end
	
	local buffer5 = nx_find_buffer(target, "buffer_dead05_01")
    if nx_string(buffer5) ~= "0" then
	    array[5] = buffer5
	end
	
	local buffer6 = nx_find_buffer(target, "buffer_dead06_01")
    if nx_string(buffer6) ~= "0" then
	    array[6] = buffer6
	end
	
	local buffer7 = nx_find_buffer(target, "buffer_PlayerExp01_01")
    if nx_string(buffer7) ~= "0" then
	    array[7] = buffer7
	end

	
	--清除
	local ncount = table.getn(array)
  	for k = 1, ncount do
 	    if nx_string(array[k]) ~= "" then
			if nx_exists(array[k]) then
				nx_remove_buffer(target, array[k])
				nx_inc_property(item, "Amount", -1)
				return 1
			end
		end
	end

	return 0
end

function useitem_trigger_task(item, target)
    local task_id = nx_query_property(item, "TriggerTaskID")
    
    if task_id ~= nil and task_id ~= "" then
        nx_trigger_task(target, nx_int(task_id))
    end
    
    return 0
end


function useitem_return(item, target)
	if not nx_find_property(item, "Para1") then
		return
	end
  local sencelocation = nx_query_property(item, "Para1")
  local senceid,x,y,z,orient = nx_get_location(sencelocation)
  nx_switch_locate(target,senceid,x,y,z,orient)
end

--添加配方
function useitem_formula(item, target)	
	--[[
	local formula_name = nx_item_static_query(item, Item_IngredientPack_FormulaID)
	if 1 == nx_add_formula_to_player(target, formula_name) then
		--添加成功,减少数量
		nx_inc_property(item, "Amount", -1)
	end
	--]]
	
	--2011-10-20 lihl
	if 1 == nx_learn_formula(target, item) then
		nx_inc_property(item, "Amount", -1);
		return true;
	end
	
	return false;
end

-- 能否使用回城道具
function can_useitem_go_home(item, player)
	local stall_state = nx_query_property(player, "StallState")
	-- 摆摊准备 或 摆摊中
	if nx_int(stall_state) == nx_int(1) or nx_int(stall_state) == nx_int(2) then
		return 0
	end

	-- 如果在战斗状态判断能不能使用该物品
	--[[
	if not nx_find_property(item, "Para4") then
		return 0
	end	
	local fight_state_use = nx_query_property(item, "Para4")
	if nx_int(fight_state_use) == nx_int(1) then
		local logic_state = nx_query_property(player, "LogicState")
		if nx_int(logic_state) == nx_int(LS_FIGHTING) then
			return 0
		end
	end
	--]]

	return 1
end

-- 使用回城道具
function useitem_go_home(item, player)
	nx_inc_property(item, "Amount", -1)
	nx_go_home(player)
	return 1
end

-- 使用回城道具开始冷却后
function after_cooldown_useitem_go_home(item, player)
	if not nx_find_property(item, "Para1") then
		return
	end	
	local ret_type = nx_query_property(item, "Para1")

	if nx_int(ret_type) == nx_int(0) then     -- 回城 最近绑定点
		if not nx_find_property(player, "RelivePositon") then
			return
		end

		local go_home_position = nx_query_property(player, "RelivePositon")

		local pos_table = nx_split_string(go_home_position, ",")

		if table.getn(pos_table) < 5 then
			return
		end

		local scene_id = nx_int(pos_table[1])
		local x = nx_float(pos_table[2])
		local y = nx_float(pos_table[3])
		local z = nx_float(pos_table[4])
		local orient = nx_float(pos_table[5])


		-- 2010-4-8 wanglei 这里要先减小物品的数量 然后在切换场景
		-- 本人认为 当切换场景时 玩家的数据会拷贝一份 传送到 另一个场景
		-- 所以应该在切换场景前 减小数据
		-- 如果在切换场景后 将数量减小 只会改变当前场景中的物品数据
		-- 而 新拷贝的数据 不会改变
		nx_inc_property(item, "Amount", -1)

		nx_switch_locate(player, scene_id, x, y, z, orient)

	elseif nx_int(ret_type) == nx_int(1) then -- 回出生点
		local scene_id = nx_int(nx_get_scene_id())

		-- 2010-4-8 wanglei 这里要先减小物品的数量 然后在切换场景
		-- 本人认为 当切换场景时 玩家的数据会拷贝一份 传送到 另一个场景
		-- 所以应该在切换场景前 减小数据
		-- 如果在切换场景后 将数量减小 只会改变当前场景中的物品数据
		-- 而 新拷贝的数据 不会改变
		nx_inc_property(item, "Amount", -1)

		nx_switch_born(player, scene_id)

	end

end

--是否是宝石
function accord_inlay(item)

    local itemtype = nx_query_property(item, "ItemType")
   
    if ITEMTYPE_INLAY1<=itemtype and itemtype<=ITEMTYPE_INLAY4 then -- 是宝石
        return 1
    end
    
    return 0
end

--创建获取临时道具产生的技能对象
function get_temp_skillobj(self,skill_id)

    local skill 
    local container = nx_get_container_fromconfigid(self,nx_widestr("OtherSkillContainer"))

    if nx_exists(container) then
         skill = nx_container_find_item(container,skill_id) --先找
         if  not nx_exists(skill) then
            skill = nx_create_from_config(container,skill_id)--找不到就插！!
         end
    end
    
    return skill
end

--道具的使用消失问题
function use_item_disappear(item)
    nx_inc_property(item,"CurrentUseNumber",1)--使用次数+1
    local disappear = nx_query_property(item,"Disappear")
    if nx_number(disappear)>0 then--要消失的
        nx_inc_property(item, "Amount", -1)
    end
end

