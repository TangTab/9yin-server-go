--麻痹,有伤害(远程外功为主),并几率向目标添加"buf_palsy"
--buff期间角色无法移动,进行攻击,被攻击无法回避,格挡;受到伤害时buff自动移除
--基础几率70%,技能Level,差最大影响 3% * Level
--LifeTime = 1 + Level / 5
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
	local damage = (1 + level * 0.1) * math.random(nx_query_property(self, "MinShotDamage"), nx_query_property(self, "MaxShotDamage"))
	local shotpowerprob = nx_number(nx_query_property(self, "shotPowerProb")) + nx_number(nx_query_property(self, "shotPowerProbAdd"))
	
	damage = damage * (shotpowerprob + 100) * 0.01

	local dex_self = nx_query_property(self, "Dex") + nx_query_property(self, "DexAdd")
	local sta_target = nx_query_property(target, "Sta") + nx_query_property(target, "StaAdd")
	
	--检定buff是否命中
	local rate = math.random(0,100)
	if dex_self - sta_target > 50 then
		rate = rate + 3 * level
	elseif dex_self - sta_target > 30 then
		rate = rate + 2 * level
	elseif dex_self - sta_target > 0 then
		rate = rate + level
	end
	
	if rate >= 30 then
		add_buffer(self, target, level)
	end

	return damage,0,0,0,0
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_palsy")
	
	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	nx_set_property(buffer, "Level", level)
		
	nx_set_property(buffer, "ReplaceLevel", level)
	nx_set_property(buffer, "ReplaceType", 0)
	--生命期
	nx_set_property(buffer, "LifeTime", nx_number(1 + level / 5))
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
	
	--添加到EventEffectRec
	nx_add_record_row_values(buffer, "EventEffectRec", COMMAND_BEDAMAGE)
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
 		nx_destroy_self(buffer)
		return false
	end
	
	return true	
end