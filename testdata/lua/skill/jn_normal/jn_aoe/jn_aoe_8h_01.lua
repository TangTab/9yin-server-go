--AOE灵蛇幻影
--等级1：外功伤害=220+武器伤害*3.0
--等级2：外功伤害=250+武器伤害*3.1
--等级3：外功伤害=280+武器伤害*3.2
--等级4：外功伤害=310+武器伤害*3.3
--等级5：外功伤害=340+武器伤害*3.4

require("define")

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
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
end

--技能被打断
function skill_break(self, skill)
end

--计算技能伤害(每一次命中都会调用一次), 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
	--取自己技能的级别
	local level = nx_query_property(skill, "Level")

	--取自己的连击数
	local hits = nx_query_property(self, "Hits")

	--角色当前内功，如果存在则获取内功级别
	local neigong_level = 0
	local cur_neigong = nx_get_current_neigong(self)
	if nx_exists(cur_neigong) then
		--角色存在当前内功
		--内功级别
		neigong_level = nx_query_property(cur_neigong, "Level")
	end

	--获取武器伤害
	local MaxMeleeDamage = nx_query_property(self, "MaxMeleeDamage")
	local MinMeleeDamage = nx_query_property(self, "MinMeleeDamage")
	local MeleeDamage = math.random(MinMeleeDamage,MaxMeleeDamage)

	--临时变量：物理伤害
	local PhyDamage = 0

	--等级1：外功伤害=220+武器伤害*3.0
	--等级2：外功伤害=250+武器伤害*3.1
	--等级3：外功伤害=280+武器伤害*3.2
	--等级4：外功伤害=310+武器伤害*3.3
	--等级5：外功伤害=340+武器伤害*3.4
	if level == 1 then
		PhyDamage = 220 + MeleeDamage*3.0
	elseif level == 2 then
		PhyDamage = 250 + MeleeDamage*3.1
	elseif level == 3 then
		PhyDamage = 280 + MeleeDamage*3.2
	elseif level == 4 then
		PhyDamage = 310 + MeleeDamage*3.3
	elseif level == 5 then
		PhyDamage = 340 + MeleeDamage*3.4
	end


	--输出伤害
	return PhyDamage, 0, 0, 0, 0
end
