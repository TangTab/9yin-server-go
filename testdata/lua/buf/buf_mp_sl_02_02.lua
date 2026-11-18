-- 增益buff
-- lv1:对目标造成的下10次普通外功伤害，附加额外10点伤害
-- lv2:对目标造成的下12次普通外功伤害，附加额外15点伤害
-- lv3:对目标造成的下14次普通外功伤害，附加额外20点伤害
-- lv4:对目标造成的下16次普通外功伤害，附加额外25点伤害
-- lv5:对目标造成的下18次普通外功伤害，附加额外30点伤害

require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

function buffer_timer(self, buffer, sender, cmdid, target, damage, damagetype)
	if damage >0 and damagetype == DAMAGE_TYPE_PHY then
		level = nx_query_property(buffer, "Level")
		if level == 1 then
			nx_damage(self, target,10)
		elseif level == 2 then
			nx_damage(self, target,15)
		elseif level == 3 then
			nx_damage(self, target,20)
		elseif level == 4 then
			nx_damage(self, target,25)
		elseif level == 5 then
			nx_damage(self, target,30)
		end
		return 1
	end

	return 0
end

--状态被打断
function buffer_break(self, buffer)
end
