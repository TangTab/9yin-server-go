--[[
buf_control_02
禁止行动
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_control_02")

	if not nx_exists(buffer) then
		return false
	end

	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "CantMove", nx_double(1), 0,0)

	--添加到目标
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
