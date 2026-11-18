-- 天朝威武

require("define")
require("share\\server_custom_define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

function buffer_timer(self, buffer)
	-- TYPE_NPC
	local NearbyList = nx_get_around_list(self, 20, 4, 50, 1)

	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		nx_custom_byken(tar, SERVER_CUSTOMMSG_NPC_TALK, tar, "天朝威武!")
	end

	-- TYPE_ITEM
	local NearbyList = nx_get_around_list(self, 20, 8, 50, 1)

	for i = 1, table.getn(NearbyList) do
		tar = nx_object(NearbyList[i])
		nx_custom_byken(tar, SERVER_CUSTOMMSG_NPC_TALK, tar, "天朝威武!")
	end

	return 1
end

--状态被打断
function buffer_break(self, buffer)
end
