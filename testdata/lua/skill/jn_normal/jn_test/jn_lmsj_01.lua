--jn_lmsj_01	六脉神剑	少冲剑
--阴属性伤害	几率附加额外伤害
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
	
	--获取自己的智力（悟性）
	local ing = nx_query_property(self, "Ing") + nx_query_property(self, "IngAdd")
	
	--获得自己的内功威力
	local magicPower = nx_query_property(self, "MagicPower") + nx_query_property(self, "MagicPowerAdd")

	--临时变量：伤害初始化
	local negDamage = 0
	
	--临时变量：BUFF效果数值初始化	
	local mul_value = 0
	
--等级1：阴伤害=100+智力*1，15％几率获得额外阴伤害=内功威力*1~2
--等级2：阴伤害=200+智力*1.2，15％几率获得额外阴伤害=内功威力*2~3
--等级3：阴伤害=300+智力*1.5，15％几率获得额外阴伤害=内功威力*3~5
--等级4：阴伤害=400+智力*1.7，15％几率获得额外阴伤害=内功威力*5~7
--等级5：阴伤害=500+智力*2，15％几率获得额外阴伤害=内功威力*7~10
	if math.random(0,100) <= 15 then
		mul_value = 1
	end
	
	if level == 1 then
		negDamage = 800+math.random(240,360)

	elseif level == 2 then
		negDamage = 800+math.random(320,440)

	elseif level == 3 then
		negDamage = 800+math.random(400,560)

	elseif level == 4 then
		negDamage = 800+math.random(480,600)

	elseif level == 5 then
		negDamage = 800+math.random(560,800)
	end
	
	--输出外功伤害
	return 0, 0, 0, negDamage, 0
end

