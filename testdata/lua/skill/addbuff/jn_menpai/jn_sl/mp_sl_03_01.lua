-- 增益buff
-- lv1:每次收到攻击时，对攻击方造成20点外功伤害，10次有效
-- lv2:每次收到攻击时，对攻击方造成25点外功伤害，12次有效
-- lv3:每次收到攻击时，对攻击方造成30点外功伤害，14次有效
-- lv4:每次收到攻击时，对攻击方造成35点外功伤害，16次有效
-- lv5:每次收到攻击时，对攻击方造成40点外功伤害，18次有效

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_03_01")
	if not nx_exists(buffer) then
		return false
	end

	local count = 0
	local dmgAdd = 0

	if level == 1 then
		count = 10
	elseif level == 2 then
		count = 12
	elseif level == 3 then
		count = 14
	elseif level == 4 then
		count = 16
	elseif level == 5 then
		count = 18
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	--COMMAND_BEDAMAGE = 614
	nx_add_record_row_values(buffer, "EventEffectRec", 614)
	nx_set_property(buffer, "EffectTimer", count)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
