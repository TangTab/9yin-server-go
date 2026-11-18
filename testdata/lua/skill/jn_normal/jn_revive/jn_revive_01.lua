-- 复活单体对象，并按照技能等级回复血量
-- lv1: 10%血量, 5%魔法
-- lv2: 20%血量，10%魔法
-- lv3: 40%血量，15%魔法
-- lv4: 60%血量，20%魔法
-- lv5: 80%血量，25%魔法

require("define")
require("share\\logicstate_define")
require("share\\server_custom_define")


--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
	--math.random(0,-1)
	target = nx_query_property(self, "LastObject")

	if not nx_exists(target) then
		return 0
	end

	if nx_query_property(target, "Dead") == 0 then
		nx_sysinfo(target, TIPSTYPE_TIPSINFO, "0", nx_widestr(nx_string("无效的目标.")))
		return 0
	end

	return 1
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
end

--技能命中之前
function skill_before_hit(self, skill)
	target = nx_query_property(self, "LastObject")

	if nx_query_property(target, "Dead") == 0 then
		nx_sysinfo(target, TIPSTYPE_TIPSINFO, "0", nx_widestr(nx_string("无效的目标.")))
		return 0
	end

	return 1
end

--技能命中对象
function skill_after_hit(self, skill, target)
--[[	if nx_query_property(target, "Dead") == 0 then
		nx_sysinfo(target, TIPSTYPE_TIPSINFO, "0", nx_widestr(nx_string("无效的目标.")))
		return
	end
]]
	nx_set_property(target, "Dead", 0)
	nx_set_property(self, "LogicState", LS_NORMAL)

	local level = nx_query_property(skill, "Level")

	local max_hp = nx_query_property(target, "MaxHP") + nx_query_property(target, "MaxHPAdd")

	local max_mp = nx_query_property(target, "MaxMP") + nx_query_property(target, "MaxMPAdd")

	-- lv1: 10%血量, 5%魔法
	-- lv2: 20%血量，10%魔法
	-- lv3: 40%血量，15%魔法
	-- lv4: 60%血量，20%魔法
	-- lv5: 80%血量，25%魔法
	if level == 1 then
		hp = max_hp * 0.1
		mp = max_mp * 0.05
	elseif level ==2 then
		hp = max_hp * 0.2
		mp = max_mp * 0.10
	elseif level ==3 then
		hp = max_hp * 0.4
		mp = max_mp * 0.15
	elseif level ==4 then
		hp = max_hp * 0.6
		mp = max_mp * 0.20
	elseif level ==5 then
		hp = max_hp * 0.8
		mp = max_mp * 0.25
	end

	nx_set_property(target, "HP", hp)
	nx_set_property(target, "MP", mp)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
