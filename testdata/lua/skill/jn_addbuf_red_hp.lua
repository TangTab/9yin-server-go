--流血,近身外功伤害,几率向目标添加"buf_red_hp"
--buff期间HP持续减少
--LifeTime = 5 + Level
--buff每秒触发1次,每次减少1%MaxHP
--1hits

require("define")
require("command_define")
require("share\\server_custom_define")

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
	local level = nx_query_property(skill, "Level")
	
	local damage = (1.5 + level * 0.1) * math.random(nx_query_property(self, "MinMeleeDamage"), nx_query_property(self, "MaxMeleeDamage"))
	local meleepowerprob = nx_number(nx_query_property(self, "MeleePowerProb")) + nx_number(nx_query_property(self, "MeleePowerProbAdd"))
	
	damage = damage * (meleepowerprob + 100) * 0.01

	add_buffer(self, target, level)

	return damage, 0, 0, 0, 0
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_bleed")
	
	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	local reduce_hp = -0.01 * (nx_query(self, "MaxHP") + nx_query(self, "MaxHPAdd"))
	nx_set_property(buffer, "Level", level)
		
	nx_set_property(buffer, "ReplaceLevel", level)
	nx_set_property(buffer, "ReplaceType", 0)
	--生命期
	nx_set_property(buffer, "LifeTime", nx_number(5 + level))
	--buffer在X秒内有效
	nx_set_property(buffer, "EffectType", 3)
	nx_set_property(buffer, "LifeTimeType", 0)
	nx_set_property(buffer, "EffectTimer", 1000)
	--触发几率
	nx_set_property(buffer, "EffectRatio", 100)
	
	--添加到DirectPropertyRec
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(reduce_hp), 0)
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
 		nx_destroy_self(buffer)
		return false
	end
	
	return true	
end