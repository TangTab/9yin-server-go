require("define")
require("share\\server_custom_define")
require("command_define")

--选定一个目标，交换血量比

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
	local target_hp_ratio = nx_query_property(target, "HPRatio")/100
	local target_max_hp = nx_query_property(target, "MaxHP")
	local self_hp_ratio = nx_query_property(self, "HPRatio")/100
	local self_max_hp = nx_query_property(self, "MaxHP")
	nx_set_property(self, "HP", self_max_hp*target_hp_ratio)
	nx_set_property(target, "HP", target_max_hp*self_hp_ratio)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--计能计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)

  --伤害初始化
	local damage = 0
	--添加BUFF
	--add_buffer(self, target, skill)
	return damage, 0, 0, 0, 0

end


