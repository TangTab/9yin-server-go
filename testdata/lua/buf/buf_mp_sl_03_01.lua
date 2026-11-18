-- 增益buff
-- lv1:每次收到攻击时，对攻击方造成20点外功伤害，10次有效
-- lv2:每次收到攻击时，对攻击方造成25点外功伤害，12次有效
-- lv3:每次收到攻击时，对攻击方造成30点外功伤害，14次有效
-- lv4:每次收到攻击时，对攻击方造成35点外功伤害，16次有效
-- lv5:每次收到攻击时，对攻击方造成40点外功伤害，18次有效

require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

function buffer_timer(self, buffer, sender, cmdid, damage, damagetype)
	if damage >0 and damagetype == DAMAGE_TYPE_PHY then
		level = nx_query_property(buffer, "Level")
		if level == 1 then
			nx_damage(self, target,20)
		elseif level == 2 then
			nx_damage(self, target,35)
		elseif level == 3 then
			nx_damage(self, target,30)
		elseif level == 4 then
			nx_damage(self, target,35)
		elseif level == 5 then
			nx_damage(self, target,40)
		end
		return 1
	end
	return 0
end

--状态被打断
function buffer_break(self, buffer)
end
