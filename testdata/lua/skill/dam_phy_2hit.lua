--普通物理技能
--str为主要参数,dex影响稳定
--射击dex为主要参数,str影响稳定
--2 hit

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

--计算技能伤害(每一次命中都会调用一次), 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
	local atktype = nx_query_property(self, "AttackType")
	local str = nx_query_property(self, "Str") + nx_query_property(self, "StrAdd")
	local dex = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")
	local damage = 0
	--计算打击伤害
	if 0 == atktype then
		local minmeleedamage = nx_query_property(self, "MinMeleeDamage")
		local maxmeleedamage = nx_query_property(self, "MaxMeleeDamage")
		local meleepower = nx_query_property(self, "MeleePower") + nx_query_property(self, "MeleePowerAdd")
		local meleepowerprob = nx_number(nx_query_property(self, "MeleePowerProb")) + nx_number(nx_query_property(self, "MeleePowerProbAdd"))
	
		damage = math.random(minmeleedamage, maxmeleedamage) + meleepower

		--打击伤害,dex影响稳定
		if str > dex then
			damage = damage - math.random(0, str - dex)
		end
		
		damage = damage * (meleepowerprob + 100) * 0.01

	--计算射击伤害
	elseif 1 == atktype then
		local minshotdamage = nx_query_property(self, "MinShotDamage")
		local maxshotdamage = nx_query_property(self, "MaxShotDamage")
		local shotpower = nx_query_property(self, "ShotPower") + nx_query_property(self, "ShotPowerAdd")
		local shotpowerprob = nx_number(nx_query_property(self, "ShotPowerProb")) + nx_number(nx_query_property(self, "ShotPowerProbAdd"))

		damage = math.random(minshotdamage, maxshotdamage) + shotpower
		
		--射击伤害,str影响稳定
		if dex > str then
			damage = damage - math.random(0, dex - str)
		end
		
		damage = damage * (shotpowerprob + 100) * 0.01
	end

	--分Hit输出	
	local hits = nx_query_property(self, "Hits")
	if 1 == hits then
		damage = damage * 0.4 + 1
	elseif 2 == hits then
		damage = damage * 0.6 + 1
	end

	return damage, 0, 0, 0, 0

end
