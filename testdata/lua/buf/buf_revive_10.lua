-- 死后原地复活，满血满状态

require("define")
require("share\\logicstate_define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

function buffer_timer(self, buffer, sender, cmdid)
	nx_set_property(self, "Dead", 0)
	nx_set_property(self, "LogicState", LS_NORMAL)

	local max_hp = nx_query_property(self, "MaxHP") + nx_query_property(self, "MaxHPAdd")
	local max_mp = nx_query_property(self, "MaxMP") + nx_query_property(self, "MaxMPAdd")

	nx_set_property(self, "HP", max_hp)
	nx_set_property(self, "MP", max_mp)

	return 1
end

--状态被打断
function buffer_break(self, buffer)
end
