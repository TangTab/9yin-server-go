--狂风迅雷
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
		
		if not nx_exists(target) then
			return
		end

		local buf = nx_find_buffer(target, "buf_clone001_haiqikuo_jd")
		
		-- 如果有buff 倦怠，则替换成buffer狂风迅雷
    if nx_exists(buf) then
    	nx_remove_buffer(target, buf)
	    add_buffer(self, target, level)
	  end
	  
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--计能计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)

	return 0, 0, 0, 0, 500
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_clone001_haiqikuo_kfxlg")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end

