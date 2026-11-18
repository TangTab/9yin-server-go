--[[
buf_revive_10
60分钟内如果死亡，死后原地复活满血满状态:)
]]

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_revive_10")
	if not nx_exists(buffer) then
		return false
	end

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
