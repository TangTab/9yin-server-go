
function on_buffer_remove(self, buffer)
	if nx_is_riding(self) then
		nx_get_down_ride(self);
	end
end