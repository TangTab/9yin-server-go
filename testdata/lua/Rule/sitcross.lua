require("share\\logicstate_define")
require("share\\server_custom_define")
require("define")

-- 特殊技能 打坐调息 的ConfigID
function sitcross_skill_configId()
	return "zs_default_01"
end

--是否可以打坐
function can_sitcross(self)

	local skill_id = sitcross_skill_configId()

	local sitcross_skill = nx_find_skill(self, skill_id)

	-- 如果没有打坐调息则返回失败
	if not sitcross_skill then
		return 0
	end

	return 1
end

--开始打坐
function begin_sitcross(self)

	--结束其他所有的流程
	nx_break_any_flow(self)

	--[[local skill_id = sitcross_skill_configId()

	local sitcross_skill = nx_find_skill(self, skill_id)

	-- 如果没有打坐调息则返回失败
	if sitcross_skill then
		--local perpareAction = nx_query_property(sitcross_skill, "PrepareAction")

		--local stateAction = nx_query_property(sitcross_skill, "LeadAction")

		-- 改变动作状态
		--nx_set_property(self, "State", stateAction)

		--做动作
		--nx_custom_byken(self, SERVER_CUSTOMMSG_ACTION, self, perpareAction)
	end]]


	--设置逻辑状态
	nx_set_property(self, "LogicState", LS_SITCROSS)

	--开始打坐
	nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9400")

	return 1
end

--结束打坐
function end_sitcross(self)

	-- 变回原来的动作状态
	--nx_set_property(self, "State", "")

	--[[local skill_id = sitcross_skill_configId()

	local sitcross_skill = nx_find_skill(self, skill_id)

	-- 如果没有打坐调息则返回失败
	if sitcross_skill then
		local finishAction = nx_query_property(sitcross_skill, "FinishAction")

		--做动作
		nx_custom_byken(self, SERVER_CUSTOMMSG_ACTION, self, finishAction)
	end]]

	--还原逻辑状态
	nx_set_property(self, "LogicState", LS_NORMAL)
	--结束打坐
	nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9401")

	return 1
end

-- 结束回调
function on_finish(self)
	restore(self)
end

function restore(self)
	local skill_id = sitcross_skill_configId()

	local sitcross_skill = nx_find_skill(self, skill_id)

	-- 如果没有打坐调息则返回失败
	if not sitcross_skill then
		return false
	end

	local skill_level = nx_query_property(sitcross_skill, "Level")
	local maxhp = nx_query_property(self, "MaxHP") + nx_query_property(self, "MaxHPAdd")
	local maxmp = nx_query_property(self, "MaxMP") + nx_query_property(self, "MaxMPAdd")

	local hp = nx_query_property(self, "HP")
	local mp = nx_query_property(self, "MP")

	local hp_add = maxhp * (4 + skill_level) * 0.01
	local mp_add = maxmp * (4 + skill_level) * 0.01

	if hp_add <= 0 then
		hp_add = 1
	end

	if mp_add <= 0 then
		mp_add = 1
	end

	if hp + hp_add > maxhp then
		hp_add = maxhp - hp;
	end

	if mp + mp_add > maxmp then
		mp_add = maxmp - mp;
	end

	if hp_add == 0 and mp_add == 0 then
		return false
	end

	--执行打坐的效果
	if hp_add > 0 then
		nx_inc_property(self, "HP", hp_add)
		nx_custom(self, SERVER_CUSTOMMSG_EFFECT, self, "ADDHP", hp_add)
	end

	if mp_add > 0 then
		nx_inc_property(self, "MP", mp_add)
		nx_custom(self, SERVER_CUSTOMMSG_EFFECT, self, "ADDMP", mp_add)
	end

	return true
end

--修炼内功
function practice_neigong(self)
	local neigong_id = nx_query_property(self, "CurNeiGong")
	if string.len(neigong_id) == 0 then
		return
	end

	--增加当前内功经验
	local level = nx_query_property(self, "Level")
	local exp = level
	nx_add_neigong_exp(self, neigong_id, exp)

	nx_custom(self, SERVER_CUSTOMMSG_EFFECT, self, "ADDEXP", exp);
end
