require("define")
--双手刺普通攻击（身法）

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

	--外功强度
	local meleePowerProb = nx_number(nx_query_property(self, "MeleePowerProb")) + nx_number(nx_query_property(self, "MeleePowerProbAdd"))

	--获得角色当前武器最大攻击
	local maxDamage = nx_query_property(self, "MaxMeleeDamage")
	
	--获得角色当前武器最小攻击
	local minDamage = nx_query_property(self, "MinMeleeDamage")

	--当前武器攻击（随机取值）
	local curDamage = math.random(minDamage,maxDamage)

	--刀普通攻击(外功)=(1+外功强度/100)*(武器攻击+身法)
	damage = (1 + meleePowerProb * 0.01) * (curDamage + dex)

	if hits == 1 then
		--第一击
		damage = damage * 0.4 + 1
	elseif hits == 2 then
		--第二击
		damage = damage * 0.6 + 2
	else
		--其他情况
		damage = damage * 0.5
	end
	
	--输出外功伤害
	return damage, 0, 0, 0, 0


end
