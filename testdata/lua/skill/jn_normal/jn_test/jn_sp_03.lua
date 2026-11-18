require("define")
require("skill\\AddBuff\\jn_normal\\jn_test\\jn_sp_03a")
require("skill\\AddBuff\\jn_normal\\jn_test\\jn_sp_03b")

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
	if not nx_add_data(skill, "Phase", VTYPE_INT) then
		return 0
	end
	nx_set_data(skill, "Phase", 0)
	return 1
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
end

--技能命中之前
function skill_before_hit(self, skill)
end

--技能命中对象
function skill_after_hit(self, skill, target)
	local target = nx_query_property(self, "LastObject")
	local phase = nx_query_data(skill, "Phase")

	if phase == 1 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 1)		-- 给对方加增益
		AddBuff_b(self, self, 1)		-- 给自己加减益
	elseif phase == 2 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 2)		-- 给对方加增益
		AddBuff_b(self, self, 2)		-- 给自己加减益
	elseif phase == 3 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 3)		-- 给对方加增益
		AddBuff_b(self, self, 3)		-- 给自己加减益
	elseif phase == 4 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 4)		-- 给对方加增益
		AddBuff_b(self, self, 4)		-- 给自己加减益
	elseif phase == 5 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 5)		-- 给对方加增益
		AddBuff_b(self, self, 5)		-- 给自己加减益
	elseif phase == 6 then
	--[[
		if not DecMP(self, 30) then
			return 0
		end
	]]
		AddBuff_a(self, target, 6)		-- 给对方加增益
		AddBuff_b(self, self, 6)		-- 给自己加减益
	end

	phase = phase + 1

	nx_set_data(skill, "Phase", phase)

	return 1
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--耗蓝
function DecMP(object, value)
	local currMP = nx_query_property(object, "MP")
	if currMP < value then
		return false
	end

	nx_inc_property(object, "MP", -value)

	return true
end
