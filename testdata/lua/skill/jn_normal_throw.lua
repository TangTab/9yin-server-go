--普通暗器技能

require("define")

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
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--计能计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)

	local minshotdamage = nx_query_property(self, "MinShotDamage")
	local maxshotdamage = nx_query_property(self, "MaxShotDamage")
	local shotpowerprob = nx_number(nx_query_property(self, "ShotPowerProb")) + nx_number(nx_query_property(self, "ShotPowerProbAdd"))

	local damage = math.random(minshotdamage, maxshotdamage)
	damage = damage * (shotpowerprob + 100) * 0.01
	
	--输出外功伤害
	return damage, 0, 0, 0, 0

end
