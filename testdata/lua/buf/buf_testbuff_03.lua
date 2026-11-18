	--等级1：触发限制BUFF，15秒内出实招，力量降低50点，持续30秒
	--等级2：触发限制BUFF，15秒内出实招，力量降低100点，持续30秒
	--等级3：触发限制BUFF，15秒内出实招，力量降低150点，持续30秒
	--等级4：触发限制BUFF，15秒内出实招，力量降低200点，持续30秒
	--等级5：触发限制BUFF，15秒内出实招，力量降低250点，持续30秒
	
require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer)
	local sender = nx_query_property(buffer, "Sender")
	
	nx_damage_target(sender, self, buffer)

	return 1
end

--状态被打断
function buffer_break(self, buffer)
end

--计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, buffer, target)
	local buffer_level = nx_query_property(buffer, "Level")
	
	return 0, 20 + buffer_level * 2, 0, 0, 0
end