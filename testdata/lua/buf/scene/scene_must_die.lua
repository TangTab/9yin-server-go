require("define")
require("share\\server_custom_define")

function OnTimer(tar, buffer)
	if nx_type(tar) == TYPE_PLAYER then
		nx_set_property(tar, "HP", 0)
		nx_set_property(tar, "Dead", 1)
	end

end

