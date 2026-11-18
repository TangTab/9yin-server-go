--AI模块接口,buff消失或移除时调用
function on_buffer_remove(self, buffer)
	local isRideSpurt = nx_query_property(self, "IsRideSpurt")
	if isRideSpurt == 1 then
		nx_end_ride_spurt(self)
	end
end