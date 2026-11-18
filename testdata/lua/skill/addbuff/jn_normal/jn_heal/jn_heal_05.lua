--[[
buf_heal_05
单体瞬时回复+回复buffer
--Lv1:20+精神*1,30秒|每5秒回复lv*20气血
--Lv2:40+精神*2,30秒|每5秒回复lv*20气血
--Lv3:80+精神*3,30秒|每5秒回复lv*20气血
--Lv4:160+精神*4,30秒|每5秒回复lv*20气血
--Lv5:320+精神*5,30秒|每5秒回复lv*20气血
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local healpower = nx_query_property(self, "HealPower") + nx_query_property(self, "HealPowerAdd")
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0
	local inc_value = 0

	if level == 1 then
		healpoint = 20 + Spi * 1
		inc_value = healpower * 20
	elseif level == 2 then
		healpoint = 40 + Spi * 2
		inc_value = healpower * 20
	elseif level == 3 then
		healpoint = 80 + Spi * 3
		inc_value = healpower * 20
	elseif level == 4 then
		healpoint = 160 + Spi * 4
		inc_value = healpower * 20
	elseif level == 5 then
		healpoint = 320 + Spi * 5
		inc_value = healpower * 20
	end

	--创建Buffer, 瞬时回复
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_05a")
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
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_05b")
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
