-- [警告]：本技能脚本由脚本生成器自动生成, 手动修改有被覆盖的可能.
-- jn_poison_08,
--[[Level1:外功伤害=(100~200)+敏捷*0.3
]]

require("define")

MAX_HIT = 1

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
	return 1
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
function skill_finish(self, skill)
end

--计算技能伤害(每一次命中都会调用一次), 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
	-- 获取技能的级别
	local level = nx_query_property(skill, "Level")
	local hit = nx_query_property(skill, "Hits")
	local PhyDamage = 0
	local Dex = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")

	if level == 1 then
		PhyDamage = math.random(100,200) + Dex*0.30
	end

	return PhyDamage,0,0,0,0
end


