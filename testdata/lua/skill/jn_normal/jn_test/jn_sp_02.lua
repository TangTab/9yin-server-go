require("define")
require("share\\server_custom_define")
-- 人物消失，瞬移至目标位置

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
	--隐身
--	if nx_query_property(self,"Invisible") == 0 then
--		nx_set_property(self,"Invisible", 1)
--	end

	--播放动作
	--nx_custom_byken(self, SERVER_CUSTOMMSG_ACTION, gm, nx_string("2h_qg_swim_run")
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
end

--技能命中之前
function skill_before_hit(self, skill)
	return 1
end

--技能命中对象
function skill_after_hit(self, skill, target)
	local x,y,z,o = nx_get_position(target)
	o = get_orient(self)

	nx_motion(self, x, y, z, o)

	if nx_query_property(self,"Invisible") == 1 then
		nx_set_property(self,"Invisible", 0)
	end

	nx_custom_byken(self, SERVER_CUSTOMMSG_ACTION, self, "2h_qg_jump03")
end

--技能使用结束
function skill_finish(self, skill)

end

--技能被打断
function skill_break(self, skill)
end

function get_orient(self)
	local self_x, self_y, self_z, self_o = nx_get_position(self)

	return self_o
end
