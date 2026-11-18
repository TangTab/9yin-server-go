--[[
传功运气，虚弱自身
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff_b(self, target, level)
	if level<=1 and level>=6 then
		return false
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_sp_03a")

	if not nx_exists(buffer) then
		return false
	end

	local value = 0
	local lifetime = 0

	if level == 1 then
		lifetime = 1800000
		value = -15
	elseif level == 2 then
		lifetime = 2400000
		value = -20
	elseif level == 3 then
		lifetime = 3000000
		value = -25
	elseif level == 4 then
		lifetime = 3600000
		value = -30
	elseif level == 5 then
		lifetime = 4200000
		value = -35
	elseif level == 6 then
		lifetime = 4800000
		value = -40
	end

	--设置Buffer
	nx_set_property(buffer, "LifeTime", lifetime)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "StrAdd", nx_double(value), 1,0)

	--添加到目标
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
