--[[
buf_heal_07
单体瞬时回复
--等级1：80+精神*2
--等级2：120+精神*2.2
--等级3：160+精神*2.4
--等级4：200+精神*2.6
--等级5：240+精神*3
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0

	if level == 1 then
		healpoint = 80 + Spi * 2
	elseif level == 2 then
		healpoint = 120 + Spi * 2.2
	elseif level == 3 then
		healpoint = 160 + Spi * 2.4
	elseif level == 4 then
		healpoint = 200 + Spi * 2.6
	elseif level == 5 then
		healpoint = 240 + Spi * 3
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_07")
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
