--攻击目标并降低其移动速度
--降低速度 等级*7%
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

	if not nx_exists(skill) then
		return
	end

	local str = nx_query_property(self, "Str") + nx_query_property(self, "StrAdd")
	local dex = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")
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
	
	local buff_lifetime = 10000
	local level = nx_query_property(skill, "Level")
	local reduce_value = -(level * 7)
	
	--nx_msgbox(tostring(reduce_value))

	add_buffer(self, target, reduce_value, buff_lifetime)
	
	return damage, 0, 0, 0, 0
end


function add_buffer(self, target, buff_value, buff_lifetime)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_sub_movespeed")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--Buffer的生命期
	nx_set_property(buffer, "LifeTime", buff_lifetime)
	
	--BUFFER改变的属性(添加到Buffer的DirectPropertyRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "MoveSpeed", nx_double(buff_value), 1, 0)
		
	--nx_msgbox(tostring(buff_value))
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true
end
