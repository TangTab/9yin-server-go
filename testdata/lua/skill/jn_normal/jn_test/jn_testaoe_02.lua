--气功	气功掌	伤害周围敌人

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
	
	--获取自己的意志（SPI）
	local spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")
	
	--获得自己的内功威力
	local magicPower = nx_query_property(self, "MagicPower") + nx_query_property(self, "MagicPowerAdd")

	--临时变量：伤害初始化
	local masDamage = 0

--等级1：阳伤害=100+内功威力+意志*1
	if level == 1 then
		masDamage = (math.random(50,150) + magicPower + spi * 1)

--等级2：阳伤害=150+内功威力+意志*1.5	
	elseif level == 2 then
		masDamage = (math.random(100,200) + magicPower + spi * 1.5)

--等级3：阳伤害=200+内功威力+意志*2
	elseif level == 3 then
		masDamage = (math.random(150,250) + magicPower + spi * 2)
		
--等级4：阳伤害=250+内功威力+意志*2.5
	elseif level == 4 then
		masDamage = (math.random(200,300) + magicPower + spi * 2.5)
		
--等级5：阳伤害=300+内功威力+意志*3	
	elseif level == 5 then
		masDamage = (math.random(250,350) + magicPower + spi * 3)
	end
	
	--输出外功伤害
	return 0, 0, 0, 0, masDamage
end

