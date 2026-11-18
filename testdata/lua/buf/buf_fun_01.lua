-- 速度由非常慢变得非常快，逐渐加速

require("define")
require("share\\logicstate_define")

--Buffer被激活
function buffer_active(self, buffer)
	local RatioChange = -120

	local oldRatio = nx_query_property(self, "SpeedRatio")

	if not nx_find_data(buffer, "IncSpeedRatio") then
		nx_add_data(buffer, "IncSpeedRatio", VTYPE_INT)
	end

	nx_set_data(buffer, "IncSpeedRatio", nx_int(RatioChange))

	local newRatio = oldRatio + RatioChange
	nx_set_property(self, "SpeedRatio", newRatio)

end

--Buffer未被激活
function buffer_deactive(self, buffer)
	local RatioChange = nx_query_data(buffer , "IncSpeedRatio")
	local oldRatio = nx_query_property(self, "SpeedRatio")
	local newRatio = oldRatio - nx_int(RatioChange)
	nx_set_property(self, "SpeedRatio", newRatio)
end

function buffer_timer(self, buffer)
	local RatioChange = nx_query_data(buffer , "IncSpeedRatio")
	local ChangeUnit = 30
	RatioChange = nx_int(RatioChange) + 30
	nx_set_data(buffer, "IncSpeedRatio", nx_int(RatioChange))

	local oldRatio = nx_query_property(self, "SpeedRatio")
	local newRatio = oldRatio + ChangeUnit
	nx_set_property(self, "SpeedRatio", newRatio)
	return 1
end

--状态被打断
function buffer_break(self, buffer)
end
