require("define")
--暗器攻击（身法）

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
	--当前第几击
	local hits = nx_query_property(self, "Hits")

	--技能的级别
	local level = nx_query_property(skill, "Level")

        --伤害初始化
	local damage = 0

	--角色身法
	local dex = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")

	--射击强度
	local shotPowerProb = nx_number(nx_query_property(self, "ShotPowerProb")) + nx_number(nx_query_property(self, "ShotPowerProbAdd"))

	--获得角色当前暗器最大攻击
	local maxDamage = nx_query_property(self, "MaxShotDamage")
	
	--获得角色当前暗器最小攻击
	local minDamage = nx_query_property(self, "MinShotDamage")

	--当前武器攻击（随机取值）
	local curDamage = math.random(minDamage,maxDamage)

	--暗器普通攻击(外功)=(1+射击强度/100)*(武器攻击+敏捷)
	damage = (1 + shotPowerProb * 0.01) * (curDamage + dex)
	
	--输出外功伤害
	return damage, 0, 0, 0, 0


end
