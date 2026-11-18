--向目标添加1个攻击时打击内力buff
--buff生命期内,目标进行攻击时,根据伤害值减少目标一定量mp

require("define")
require("command_define")

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
	local level = nx_query_property(skill, "Level")
	
	add_buffer(self, target, level)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--计算技能伤害(每一次命中都会调用一次), 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_dam_mp")
	
	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	nx_set_property(buffer, "Level", level)
		
	nx_set_property(buffer, "ReplaceLevel", level)
	nx_set_property(buffer, "ReplaceType", 0)
	--生命期
	nx_set_property(buffer, "LifeTime", 30000)
	--buffer在X秒内有效
	nx_set_property(buffer, "EffectType", 1)
	nx_set_property(buffer, "LifeTimeType", 1)
	--触发几率
	nx_set_property(buffer, "EffectRatio", 100)
	
	--添加到EventEffectRec
	nx_add_record_row_values(buffer, "EventEffectRec", COMMAND_DAMAGETARGET)
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true	
end