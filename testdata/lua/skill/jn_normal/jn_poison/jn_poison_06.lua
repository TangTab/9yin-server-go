--使用得到buff，专门抵抗悲酥清风的效果。

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
	add_buffer(self, self)
	return 1
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

function add_buffer(self, target)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_poison_06")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",20000)

	--添加免疫表
	nx_add_record_row_values(buffer, "ImmunityRec", 3, "buf_poison_02")

	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)

	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end
