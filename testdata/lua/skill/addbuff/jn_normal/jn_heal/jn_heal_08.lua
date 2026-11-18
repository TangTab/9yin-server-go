--[[
buf_heal_08
单体瞬时回复+回复buffer
--等级1：100+精神*2,40秒|每5秒回复lv*50气血
--等级2：200+精神*2.4,40秒|每5秒回复lv*50气血
--等级3：400+精神*2.8,40秒|每5秒回复lv*50气血
--等级4：800+精神*3.2,40秒|每5秒回复lv*50气血
--等级5：1200+精神*3.6,40秒|每5秒回复lv*50气血
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local healpower = nx_query_property(self, "HealPower") + nx_query_property(self, "HealPowerAdd")
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0
	local inc_value = 0

	if level == 1 then
		healpoint = 100 + Spi * 2
		inc_value = healpower*50
	elseif level == 2 then
		healpoint = 200 + Spi * 2.2
		inc_value = healpower*50
	elseif level == 3 then
		healpoint = 400 + Spi * 2.4
		inc_value = healpower*50
	elseif level == 4 then
		healpoint = 800 + Spi * 2.6
		inc_value = healpower*50
	elseif level == 5 then
		healpoint = 1200 + Spi * 3
		inc_value = healpower*50
	end

	--创建Buffer, 瞬时回复
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_08a")
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
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_08b")
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
