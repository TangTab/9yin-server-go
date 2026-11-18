--大慈大悲	指定区域范围内瞬回
--等级1：80+精神*2
--等级2：120+精神*2.2
--等级3：160+精神*2.4
--等级4：200+精神*2.6
--等级5：240+精神*3


require("define")
require("skill\\AddBuff\\jn_normal\\jn_heal\\jn_heal_07")

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
