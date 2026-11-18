--BUFF持续过程中，如果玩家使用了实招，则损失当前HP的50%

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
	
	--如果没有使用实招，则返回
	if nx_query_property(cur_skill, "EffectType") ~= 1 then
		return 0
	end
	
	--使用了实招，扣除当前HP的50%
	local curDamage = nx_query_property(self, "HP") * 0.5
	if curDamage >= 1 then
		local sender = nx_query_property(buffer, "Sender")
		nx_damage(sender, self, curDamage)
	end
	
	return 1
end

--状态被打断
function buffer_break(self, buffer)
end
