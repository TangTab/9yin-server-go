--[[
buf_heal_03
单体瞬时回复
Lv1:60+精神*1
Lv2:120+精神*2
Lv3:240+精神*3
Lv4:480+精神*4
Lv5:960+精神*5
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0

	if level == 1 then
		healpoint = 60 + Spi * 1
	elseif level == 2 then
		healpoint = 120 + Spi * 2
	elseif level == 3 then
		healpoint = 240 + Spi * 3
	elseif level == 4 then
		healpoint = 480 + Spi * 4
	elseif level == 5 then
		healpoint = 960 + Spi * 5
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_03")
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
