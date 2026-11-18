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

--计能计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
  --伤害初始化
  local hits = nx_query_property(self, "Hits")

	local damage = 999999
	
	if hits == 1 then
		damage=250
		
		local targer_hp =  nx_query_property(target, "HP")
		-- 目标HP小于800就秒杀
		if targer_hp < 10000 then
			damage = 999999
		end
		
	elseif hits == 2 then
	
		damage=300
		
	elseif hits == 3 then
	
		damage=350
		
	end

	--默认是外功攻击
	return damage, 0, 0, 0, 0
end
