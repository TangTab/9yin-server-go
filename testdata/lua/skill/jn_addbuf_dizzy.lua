--眩晕,有伤害(近身外功,或内功为主),并几率向目标添加"buf_dizzy"
--buff期间角色无法移动,进行攻击,被攻击无法回避,格挡
--基础几率50%,技能Level,自己str与对方sta差最大影响 3% * Level
--LifeTime = 2 + Level / 3
--buff效果作用时,播放debuff_01动作
--1hit

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
	local damage = (2 + level * 0.1) * math.random(nx_query_property(self, "MinMeleeDamage"), nx_query_property(self, "MaxMeleeDamage"))
	local meleepowerprob = nx_number(nx_query_property(self, "MeleePowerProb")) + nx_number(nx_query_property(self, "MeleePowerProbAdd"))
	
	damage = damage * (meleepowerprob + 100) * 0.01

	local str_self = nx_query_property(self, "Str") + nx_query_property(self, "StrAdd")
	local sta_target = nx_query_property(target, "Sta") + nx_query_property(target, "StaAdd")

	--检定buff是否命中	
	local rate = math.random(0,100)
	if str_self - sta_target > 50 then
		rate = rate + 3 * level
	elseif str_self - sta_target > 30 then
		rate = rate + 2 * level
	elseif str_self - sta_target > 0 then
		rate = rate + level
	end
	
	if rate >= 50 then
		add_buffer(self, target, level)
	end

	return damage,0,0,0,0
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_dizzy")
	
	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	nx_set_property(buffer, "Level", level)
		
	nx_set_property(buffer, "ReplaceLevel", level)
	nx_set_property(buffer, "ReplaceType", 0)
	--生命期
	nx_set_property(buffer, "LifeTime", nx_number(2 + level / 3))
	--buffer在X秒内有效
	nx_set_property(buffer, "EffectType", 1)
	nx_set_property(buffer, "LifeTimeType", 1)
	
	--添加到DirectPropertyRec
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "CantMove", 1, 0, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "CantAttack", 1, 0, 0)	
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "DodgeProb", -100, 1, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "DodgeProbAdd", -100, 1, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "PhyParryProb", -100, 1, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "PhyParryProbAdd", -100, 1, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "MagicParryProb", -100, 1, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "MagicParryProbAdd", -100, 1, 0)
	nx_set_property(self, "State", debuff_01)
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
 		nx_destroy_self(buffer)
		return false
	end
	
	return true	
end