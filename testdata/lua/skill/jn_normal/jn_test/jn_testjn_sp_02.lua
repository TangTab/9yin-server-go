require("define")
-- 人物消失，瞬移至目标位置

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
	--隐身
	if nx_query_property(self,"Invisible") == 0 then
		nx_set_property(self,"Invisible", 1)
    end
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
end

--技能使用结束
function skill_finish(self, skill)
	nx_moveto(self, x, y, z, orient)
	if nx_query_property(self,"Invisible") == 1 then
		nx_set_property(self,"Invisible", 0)
	end
	nx_custom_byken(self, SERVER_CUSTOMMSG_ACTION, gm, nx_string("2h_qg_jump03"))
end

--技能被打断
function skill_break(self, skill)

end

