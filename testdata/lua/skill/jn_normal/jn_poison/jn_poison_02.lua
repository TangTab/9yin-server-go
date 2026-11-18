--悲酥清风	向角色周围范围内施放大规模毒术，毒术对自身有效，建议先用解药(触发抵抗buff)
--降低5点体质，60秒内每2秒减少50的血量

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

	-- TYPE_NPC | TYPE_PLAYER =6
	local NearbyList = nx_get_around_list(self, 10, 6, 50, 1)

	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		add_buffer(self, tar)
	end

	-- 自己也中招！
	add_buffer(self, self)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

function add_buffer(self, target)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_poison_02")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",60000)

	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 2000)

	--BUFFER持续改变的属性（降低命中）
	nx_add_record_row_values(buffer, "RolePropRec", "Sta", nx_double(-5))

	--BUFFER连续改变的属性（连续扣血）
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(-50),0)

	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)

	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end
