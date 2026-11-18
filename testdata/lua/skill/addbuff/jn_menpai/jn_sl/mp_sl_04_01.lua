-- 增益buff
-- lv1:外功攻击增加18点, 持续30分钟
-- lv2:外功攻击增加20点, 持续30分钟
-- lv3:外功攻击增加24点, 持续30分钟
-- lv4:外功攻击增加30点, 持续30分钟
-- lv5:外功攻击增加38点, 持续30分钟

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_04_01")
	if not nx_exists(buffer) then
		return false
	end

	local dmg_add = 0


	if level == 1 then
		dmg_add = 18
	elseif level == 2 then
		dmg_add = 20
	elseif level == 3 then
		dmg_add = 24
	elseif level == 4 then
		dmg_add = 30
	elseif level == 5 then
		dmg_add = 38
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "PhyDamageAdd", nx_double(dmg_add), 0,0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
