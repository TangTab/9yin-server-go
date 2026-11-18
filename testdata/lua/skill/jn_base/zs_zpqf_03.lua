--杂牌拳法	3HIT 架招

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
	--取自己技能的级别
	local level = nx_query_property(skill, "Level")
	--获取技能的Hits数
	local hits = nx_query_property(self, "Hits")
	--获取角色的力量（臂力）
	local str = nx_query_property(self, "Str") + nx_query_property(self, "StrAdd")
	--获取自己的敏捷（身法）
	local dex = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")
	--临时变量：外功伤害初始化
	local phyDamage = 0
	
	--等级1：外功伤害=30+臂力*1+身法*1
	--等级2：外功伤害=60+臂力*1.1+身法*1.1
	--等级3：外功伤害=90+臂力*1.25+身法*1.25
	--等级4：外功伤害=120+臂力*1.35+身法*1.35
	--等级5：外功伤害=150+臂力*1.5+身法*1.5
	
	if level == 1 then
		phyDamage = math.random(25,35) + (str + dex) * 1

	elseif level == 2 then
		phyDamage = math.random(50,70) + (str + dex) * 1.1

	elseif level == 3 then
		phyDamage = math.random(80,100) + (str + dex) * 1.25

	elseif level == 4 then
		phyDamage = math.random(110,130) + (str + dex) * 1.35

	elseif level == 5 then
		phyDamage = math.random(135,165)+ (str + dex) * 1.5
	end

	phyDamage = phyDamage * 0.25

	
	--输出外功伤害
	return phyDamage, 0, 0, 0, 0
end

