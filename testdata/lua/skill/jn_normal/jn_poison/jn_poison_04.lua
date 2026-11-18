--向目标单体施放毒buff，buff的效果会影响到中buff的目标周围一定范围的所有角色。
--持续30s，每5s触发一次，每次对自己伤害200，对附近10米范围内的单位造成100点伤害

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
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_poison_04")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",30000)

	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 5000)

	--BUFFER连续改变的属性（连续扣血）
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(-200), 0)

	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)

	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
