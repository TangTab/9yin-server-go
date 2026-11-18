-- 增益buff
-- lv1:增加目标外功防御40点, 力量2点， 持续30分钟
-- lv2:增加目标外功防御50点, 力量4点， 持续30分钟
-- lv3:增加目标外功防御60点, 力量6点， 持续30分钟
-- lv4:增加目标外功防御70点, 力量8点， 持续30分钟
-- lv5:增加目标外功防御80点, 力量10点， 持续30分钟

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_01_01")
	if not nx_exists(buffer) then
		return false
	end

	local str_inc = 0
	local phy_def = 0

	if level == 1 then
		phy_def = 40
		str_inc = 2
	elseif level == 2 then
		phy_def = 50
		str_inc = 4
	elseif level == 3 then
		phy_def = 60
		str_inc = 6
	elseif level == 4 then
		phy_def = 70
		str_inc = 8
	elseif level == 5 then
		phy_def = 80
		str_inc = 10
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "PhyDefAdd", nx_double(phy_def), 0,0)
	nx_add_record_row_values(buffer, "RolePropRec", "StrAdd", nx_double(str_inc))

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
