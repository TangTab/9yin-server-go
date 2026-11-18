--AOE演示00
--近距离获得巨大伤害，远距离小伤害

require("define")

--设定最大HIT值
MAX_HIT=17

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

	--取自己的连击数
	local hits = nx_query_property(self, "Hits")

	--获取武器伤害
	local MaxMeleeDamage = nx_query_property(self, "MaxMeleeDamage")
	local MinMeleeDamage = nx_query_property(self, "MinMeleeDamage")
	local MeleeDamage = math.random(MinMeleeDamage,MaxMeleeDamage)

	--外功伤害
	local phyDamage = 0

	--刚属性伤害
	local stiDamage = 0

	local distance = cal_distance(self, target)
	local pwrlevel = distance_to_power_level(distance)

	if level == 1 then
		phyDamage = math.random(200,400) + MeleeDamage*2 + hits*10
		stiDamage = math.random(100,300)
	elseif level == 2 then
		phyDamage = math.random(640,740) + MeleeDamage*2.5 + hits*12
		stiDamage = math.random(120,350)
	elseif level == 3 then
		phyDamage = math.random(680,780) + MeleeDamage*3 + hits*14
		stiDamage = math.random(140,400)
	elseif level == 4 then
		phyDamage = math.random(720,820) + MeleeDamage*3.5 + hits*16
		stiDamage = math.random(160,450)
	elseif level == 5 then
		phyDamage = math.random(760,860) + MeleeDamage*4 + hits*18
		stiDamage = math.random(180,500)
	end

	if hits ~= MAX_HIT then
		stiDamage = 0
	end

	phyDamage = (phyDamage / MAX_HIT)
	
	phyDamage = pwrlevel * phyDamage --	根据威力增加威力

	return phyDamage, stiDamage, 0, 0, 0
end

function cal_distance(self, target)
	local self_x,self_y,self_z,self_o = nx_get_position(self)
	local tar_x,tar_y,tar_z,tar_o = nx_get_position(target)
	self_x = nx_int(self_x)
	self_y = nx_int(self_y)
	self_z = nx_int(self_z)
	tar_x = nx_int(self_x)
	tar_y = nx_int(tar_y)
	tar_z = nx_int(tar_z)
	
	return nx_int(math.sqrt((self_x - tar_x)^2 + (self_y - tar_y)^2 + (self_z - tar_z)^2))
end

function distance_to_power_level(distance)
	if distance >0 and distance<=2 then
		return 5
	elseif distance >2 and distance<=5 then
		return 2
	elseif distance >5 and distance<=8 then
		return 1
	elseif distance >8 then
		return 1
	end
	return 0
end
