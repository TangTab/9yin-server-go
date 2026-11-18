--jn_heal_04	慈悲疗法	给友方一定范围的数个单位回复气血
--Lv1:30+精神*1.2
--Lv2:60+精神*1.6
--Lv3:90+精神*2.0
--Lv4:180+精神*2.4
--Lv5:360+精神*2.8


require("define")
require("skill\\AddBuff\\jn_normal\\jn_heal\\jn_heal_04")

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
	local level = nx_query_property(skill, "Level")
	AddBuff(self, target, level)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
