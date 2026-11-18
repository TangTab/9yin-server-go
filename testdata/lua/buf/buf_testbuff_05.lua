	--等级1：触发限制BUFF，15秒内出架招，损失一半MP
	--等级2：触发限制BUFF，20秒内出架招，损失一半MP
	--等级3：触发限制BUFF，25秒内出架招，损失一半MP
	--等级4：触发限制BUFF，30秒内出架招，损失一半MP
	--等级5：触发限制BUFF，35秒内出架招，损失一半MP
	
require("define")
require("share\\server_custom_define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer)

    --判断获取角色当前使用技能的行为
	local cur_skill = nx_query_property(self, "CurSkill")
	
	--如果没有使用技能，则返回
	if not nx_exists(cur_skill) then
		return 0
	end
	
	--如果没有使用架招，则返回
	if nx_query_property(cur_skill, "EffectType") ~= 3 then
		return 0
	end
	
	--使用了架招，扣除当前HP的50%
	local cur_mp = nx_query_property(self, "MP")
	nx_set_property(self, "MP", cur_mp / 2)
	
	--发送自定义消息，损失HP
	nx_custom_byken(self, SERVER_CUSTOMMSG_EFFECT, self, "MP", cur_mp / 2)
	
	return 1
end

--状态被打断
function buffer_break(self, buffer)
end