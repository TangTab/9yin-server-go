--实现下马时于player身边创建马匹NPC功能
function on_buffer_remove(self, buffer)
	
		local dn_result = nx_dn_special_ride(self)
		local del_result = nx_del_special_mount(self)

		if dn_result and del_result then
		
			--创建NPC
			local x,y,z,o = nx_get_position(self)
			local X,Y,Z,O = nx_number(x)+math.sin(nx_number(o)+math.pi/2)*0.5,y,nx_number(z)+math.cos(nx_number(o)+math.pi/2)*0.5,o
			nx_create_obj("carriernpc002005",0,X,Y,Z,O)

		end	
end