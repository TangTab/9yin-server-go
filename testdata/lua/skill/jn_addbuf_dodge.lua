--向目标添加buff,提高DodgeProbAdd

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
	local level = nx_query_property(self, "Level")
	local buff_value = 20 + math.floor(level / 6)
	local buff_lifetime = 15000
	
	add_buffer(self, target, buff_value, buff_lifetime)

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

function add_buffer(self, target, value, lifetime)
	local buffer = nx_create_from_config(nx_get_scene(), "buf_add_dodge")
	if not nx_exists(buffer) then
		return false
	end
	
	nx_set_property(buffer, "LifeTime", lifetime)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "DodgeProbAdd", nx_double(value), 0, 0)

	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true
	
end