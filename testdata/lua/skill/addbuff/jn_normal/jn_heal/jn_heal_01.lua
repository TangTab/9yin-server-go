--[[
buf_heal_01
单体瞬时回复
等级1：40+精神*1
等级2：80+精神*2
等级3：160+精神*3
等级4：320+精神*4
等级5：640+精神*5
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local Spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")

	local healpoint = 0

	if level == 1 then
		healpoint = 40 + Spi * 1
	elseif level == 2 then
		healpoint = 80 + Spi * 2
	elseif level == 3 then
		healpoint = 160 + Spi * 3
	elseif level == 4 then
		healpoint = 320 + Spi * 4
	elseif level == 5 then
		healpoint = 640 + Spi * 5
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_heal_01")

	if not nx_exists(buffer) then
		return false
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "HP", nx_double(healpoint), 0,1)

	--添加到目标
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	return true
end
