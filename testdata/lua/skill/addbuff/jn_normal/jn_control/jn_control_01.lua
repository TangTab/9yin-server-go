--[[
buf_control_01
减速60%
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_control_01")

	if not nx_exists(buffer) then
		return false
	end
	--设置Buffer
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "SpeedRatio", nx_double(-60), 0,0)

	--添加到目标
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
