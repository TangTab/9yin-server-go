-- buff的效果会影响到中buff的目标周围一定范围的所有角色。

require("define")

--Buffer被激活
function buffer_active(self, buffer)
	self.Color = "255,0,255,0"
end

--Buffer未被激活
function buffer_deactive(self, buffer)
	self.Color = "0,255,255,255"
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer)
	-- TYPE_NPC | TYPE_PLAYER =6
	local NearbyList = nx_get_around_list(self, 10, 6, 50, 1)
	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		nx_damage_target(self, tar, buffer)
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
