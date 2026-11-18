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
end

--调整自身命中值
function adjust_hit(self, skill, target, hit)
	hit = nx_number(hit)
	return nx_float(hit)
end

--调整闪避值
function adjust_dodge(self, skill, src, src_skill, dodge)
	dodge = nx_number(dodge)
	return nx_float(dodge)
end

--调整抗性值
function adjust_defend_prob(self, skill, src, src_skill, defend_prob)
	defend_prob = nx_number(defend_prob)
	return nx_float(defend_prob)
end

--调整格档值
function adjust_parry(self, skill, src, src_skill, parry)
	parry = nx_number(parry)
	return nx_float(parry)
end