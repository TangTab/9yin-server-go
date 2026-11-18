--攻击目标并移除目标buf_testbuff_06(hp持续回复buff)

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
	local ing= nx_query_property(self, "Ing") + nx_query_property(self, "IngAdd")
	local magicpower = nx_query_property(self, "MagicPower") + nx_query_property(self, "MagicPowerAdd")
	local magicpowerprob = nx_number(nx_query_property(self, "MagicPowerProb")) + nx_number(nx_query_property(self, "MagicPowerProbAdd"))
	
	local damage = (ing * math.random(0.5, 1) + magicpower)*(magicpowerprob + 100) * 0.01

	add_buffer(self, target)
	return damage, 0, 0, 0, 0

end

function add_buffer(self, target)
	local buffer = nx_find_buffer(target, "buf_testbuff_06")
	
	if nx_exists(buffer) then
		nx_remove_buffer(target, buffer)
		return true
	end
	
	return false
end
