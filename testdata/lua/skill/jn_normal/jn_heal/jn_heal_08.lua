--普渡众生	指定区域范围内添加回复buff
--等级1：100+精神*2,40秒|每5秒回复lv*50气血
--等级2：200+精神*2.4,40秒|每5秒回复lv*50气血
--等级3：400+精神*2.8,40秒|每5秒回复lv*50气血
--等级4：800+精神*3.2,40秒|每5秒回复lv*50气血
--等级5：1200+精神*3.6,40秒|每5秒回复lv*50气血


require("define")
require("skill\\AddBuff\\jn_normal\\jn_heal\\jn_heal_08")

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
