--[[
buf_heal_04
单体瞬时回复
--Lv1:30+精神*1.2
--Lv2:60+精神*1.6
--Lv3:90+精神*2.0
--Lv4:180+精神*2.4
--Lv5:360+精神*2.8
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0

	if level == 1 then
		healpoint = 30 + Spi * 1.2
	elseif level == 2 then
		healpoint = 60 + Spi * 1.6
	elseif level == 3 then
		healpoint = 90 + Spi * 2.0
	elseif level == 4 then
		healpoint = 180 + Spi * 2.4
	elseif level == 5 then
		healpoint = 360 + Spi * 2.8
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_04")
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
	return true
end
