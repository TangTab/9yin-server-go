function skill_finish(self, skill)
	-- 骑马
	if not nx_is_riding(self) then
		if nx_go_ride(self) then
			add_buffer(self)
		end
	end
end


function add_buffer(self)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local mount_box = nx_get_child(self, nx_widestr("MountBox"))

	if nx_exists(mount_box) then
		local mount_item = nx_get_item(mount_box, 1)

		if nx_exists(mount_item) then
			buffer_id = nx_query_property(mount_item, "RideBuffer")
			local buffer = nx_create_from_config(nx_get_scene(), buffer_id)
		
			--如果创建失败则返回
			if not nx_exists(buffer) then
				return false
			end
			
			
			--添加到目标身上,同时记录结果
			local result = nx_add_buffer(self, self, buffer)

			--如果添加失败,则删除创建的Buffer对象
			if not result then
				nx_destroy_self(buffer)
				return false
			end
		end
	end
end