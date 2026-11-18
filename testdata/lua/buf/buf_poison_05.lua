-- 每次触发血量减半，并传染周围3米内的单位

require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer)
	--自己血量减半
	local max_hp = nx_query_property(self, "MaxHP") + nx_query_property(self, "MaxHPAdd")
	local dmg = max_hp / 2

	--感染周围单位

	-- TYPE_NPC | TYPE_PLAYER =6
	local NearbyList = nx_get_around_list(self, 3, 6, 50, 1)
	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		add_buffer(self, tar)
	end
	return 1
end

--状态被打断
function buffer_break(self, buffer)
end

--计算伤害, 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, buffer, target)
	return 100,0,0,0,0
end



function add_buffer(self, target)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_poison_05")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",30000)

	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 5000)

	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)

	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end
