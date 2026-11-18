require("define")
require("share\\logicstate_define")

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
end

--技能命中之前
function skill_before_hit(self, skill)
	return 1
end

--技能命中对象
function skill_after_hit(self, skill, target)
	if nx_query_property(target, "Dead") == 0 then
		return
	end
	
	nx_set_property(target, "Dead", 0)
	nx_set_property(self, "LogicState", LS_NORMAL)
	
	local level = nx_query_property(skill, "Level")
	
	local max_hp = nx_query_property(target, "MaxHP") + nx_query_property(target, "MaxHPAdd")
	local cur_hp = nx_query_property(target, "HP")

	local max_mp = nx_query_property(target, "MaxMP") + nx_query_property(target, "MaxMPAdd")
	local cur_mp = nx_query_property(target, "MP")
	
	local min_hp = max_hp * 0.2
	local min_mp = max_mp * 0.1
	
	if cur_hp < min_hp then
		nx_set_property(target, "HP", min_hp)
	end

	if cur_mp < min_mp then
		nx_set_property(target, "MP", min_mp)
	end
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
