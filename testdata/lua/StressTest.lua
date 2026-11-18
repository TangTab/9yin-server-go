require("define")
function StressTest(self,...)
		local msgid = nx_int(arg[1]);
		local msg_type = nx_string(arg[2])--消息类型
		if msg_type == "npc战斗测试"or msg_type == "npcfight" then
			local x,y,z,o = nx_get_position(self);
			local npc1 = nx_create_obj("monc023700", 0, nx_number(x)+2, y,  nx_number(z)+2, o);
			nx_add_record_row_values(npc1, "EnemyNpcRec", "monc023600")
			nx_add_record_row_values(npc1, "EnemyNpcRec", "monc023500")
			--nx_add_record_row_values(npc1, "EnemyNpcRec", "monrm054100")
			local npc2 = nx_create_obj("monc023600", 0, nx_number(x)+2, y,  nx_number(z)-2, o);
			nx_add_record_row_values(npc2, "EnemyNpcRec", "monc023700")		
			nx_add_record_row_values(npc2, "EnemyNpcRec", "monc023500")
		--	nx_add_record_row_values(npc2, "EnemyNpcRec", "monrm054100")
			local npc3 = nx_create_obj("monc023500", 0, nx_number(x)-2, y,  nx_number(z)+2, o);
			nx_add_record_row_values(npc3, "EnemyNpcRec", "monc023700")		
			nx_add_record_row_values(npc3, "EnemyNpcRec", "monc023600")
			--nx_add_record_row_values(npc3, "EnemyNpcRec", "monrm054100")
		--	local npc4 = nx_create_obj("monrm054100", 0, nx_number(x)-2, y,  nx_number(z)-2, o);
		--	nx_add_record_row_values(npc4, "EnemyNpcRec", "monv030500")		
		--	nx_add_record_row_values(npc4, "EnemyNpcRec", "monv030600")
		--	nx_add_record_row_values(npc4, "EnemyNpcRec", "monrm054200")
		elseif msg_type == "buff测试" or msg_type == "bufftest" then
			add_buffer(self, self)
		elseif msg_type == "创建副本" or msg_type == "createclone"then
			nx_clone_scene(self, 27)--孔雀山庄
		elseif msg_type == "切换场景" or msg_type == "switch_born"then
			local sceneid = nx_int(arg[3])
			local curSceneid = nx_get_scene_id()
			if curSceneid == 1 then
				nx_switch_born(self, nx_int(12))
			end
			if curSceneid == 12 then
				nx_switch_born(self, nx_int(9))
			end
			if curSceneid == 9 then
				nx_switch_born(self, nx_int(1))
			end
		elseif msg_type == "random位置" or msg_type == "randomMove" then
			local x = nx_int(arg[3])
			local z = nx_int(arg[4])
			local y = nx_get_height(x, z);
			local scene = nx_get_scene_id()
			--nx_msgbox(nx_string(x).."/"..nx_string(y).."/"..nx_string(z).."/"..nx_string(scene))
			nx_moveto(self, x, y, z, 0)
		elseif msg_type == "初始化离线行为" or msg_type == "InitOffLineJob" then

			nx_clear_record(self,"OffLine_Job");
			nx_clear_record(self,"OffLine_Job_Log");

			init_offlinejob_id(self)
		end

		return 0;
end

function add_buffer(self, target)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_newrole_test")
	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end

	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end

--初始化离线行为
function init_offlinejob_id(self)
	nx_set_property(self, "CantMove", 0)
end
