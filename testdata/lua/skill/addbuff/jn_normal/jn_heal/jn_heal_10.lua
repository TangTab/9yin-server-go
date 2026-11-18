--测试技能，立刻恢复生命，徐徐恢复生命

--[[
buf_heal_10
单体瞬时回复
--等级1，恢复生命，60+治疗等级*1，每3秒恢复生命=20+治疗等级，持续30秒
--等级2，恢复生命，120+治疗等级*1.5，每3秒恢复生命=40+治疗等级，持续30秒
--等级3，恢复生命，180+治疗等级*2，每3秒恢复生命=60+治疗等级，持续30秒
--等级4，恢复生命，240+治疗等级*2.5，每3秒恢复生命=80+治疗等级，持续30秒
--等级5，恢复生命，320+治疗等级*3，每3秒恢复生命=100+治疗等级，持续30秒
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local healpower = nx_query_property(self, "HealPower") + nx_query_property(self, "HealPowerAdd")

	local healpoint = 0
	local inc_value = 0

	if level == 1 then
		healpoint = 60 + healpower * 1
		inc_value = 20 + healpower

	elseif level == 2 then
		healpoint = 120 + healpower * 1.5
		inc_value = 40 + healpower

	elseif level == 3 then
		healpoint = 180 + healpower * 2
		inc_value = 60 + healpower

	elseif level == 4 then
		healpoint = 240 + healpower * 2.5
		inc_value = 80 + healpower

	elseif level == 5 then
		healpoint = 320 + healpower * 3
		inc_value = 100 + healpower
	end

	--创建Buffer, 瞬时回复
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_10a")
	if not nx_exists(buffer) then
		return false
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "HP", nx_double(healpoint), 0,1)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	--创建Buffer, 周期性回血
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_10b")
	if not nx_exists(buffer) then
		return false
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(inc_value), 0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	return true
end
