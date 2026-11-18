-- 增益buff
-- lv1:对目标造成的下10次普通外功伤害，附加额外10点伤害
-- lv2:对目标造成的下12次普通外功伤害，附加额外15点伤害
-- lv3:对目标造成的下14次普通外功伤害，附加额外20点伤害
-- lv4:对目标造成的下16次普通外功伤害，附加额外25点伤害
-- lv5:对目标造成的下18次普通外功伤害，附加额外30点伤害

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_02_02")
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
	--COMMAND_DAMAGETARGET = 613
	nx_add_record_row_values(buffer, "EventEffectRec", 613)
	nx_set_property(buffer, "EffectTimer", count)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
