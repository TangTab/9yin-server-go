	--等级1：触发限制BUFF，30秒内使用技能，损失生命=技能消耗的MP*3
	--等级2：触发限制BUFF，30秒内使用技能，损失生命=技能消耗的MP*6
	--等级3：触发限制BUFF，30秒内使用技能，损失生命=技能消耗的MP*9
	--等级4：触发限制BUFF，30秒内使用技能，损失生命=技能消耗的MP*12
	--等级5：触发限制BUFF，30秒内使用技能，损失生命=技能消耗的MP*15
	
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
	--获取BUFF等级
	local level = nx_query_property(buffer, "Level")
    --判断获取角色当前使用技能的行为
	local cur_skill = nx_query_property(self, "CurSkill")
	--如果没有使用技能，则返回
	if not nx_exists(cur_skill) then
		return 0
	end

	--使用技能，使用技能，损失生命=技能消耗的MP*1
	local consume_mp = nx_query_property(cur_skill, "AConsumeMP")
	local percent_mp = nx_query_property(cur_skill, "PConsumeMP")

	if percent_mp > 0  then
		local max_mp = nx_query_property(self, "MaxMP") + nx_query_property(self, "MaxMPAdd")
		consume_mp = consume_mp + max_mp * percent_mp / 100
	end

	if level == 1 then
		curDamage = consume_mp * 3
	elseif level == 2 then
		curDamage = consume_mp * 6
	elseif level == 3 then
		curDamage = consume_mp * 9
	elseif level == 4 then
		curDamage = consume_mp * 12
	elseif level == 5 then
		curDamage = consume_mp * 15	
	end
	
	if curDamage > 0 then
		local sender = nx_query_property(buffer, "Sender")
		nx_damage(sender, self, curDamage)
	end
	
	return 1
end

--状态被打断
function buffer_break(self, buffer)
end