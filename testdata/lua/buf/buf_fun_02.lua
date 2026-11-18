-- 所到之处寸草不生人畜皆死

require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

function buffer_timer(self, buffer)
	-- TYPE_NPC | TYPE_PLAYER =6
	local NearbyList = nx_get_around_list(self, 20, 6, 50, 1)

	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		nx_damage(self, tar, 10000000)
	end

	-- TYPE_ITEM
	local NearbyList = nx_get_around_list(self, 20, 8, 50, 1)

	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		nx_damage(self, tar, 10000000)
		nx_destroy_self(tar)
	end

	return 1
end

--状态被打断
function buffer_break(self, buffer)
end
