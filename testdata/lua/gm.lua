require("define")
require("share\\server_custom_define")
require("share\\static_data_type")
require("gm_util")

--获得当前属性操作对象
function get_prop_target(gm)
	if not nx_find_data(gm, "GMPropTarget") then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not exist property operate target!"));
		return nx_null();
	end

	local target = nx_query_data(gm, "GMPropTarget");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not exist property operate target!"));
		return nx_null();
	end

	return target
end

function ss(gm, ...)
	if table.getn(arg) == 0 then
		local rows = nx_get_pubdata_record_rows("domain", nx_widestr("Domain_SceneList"), "scene_list_rec");

        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("scene list begin ..."));
		for i = 0, rows - 1 do
			local values = nx_query_pubdata_record_row_values("domain", nx_widestr("Domain_SceneList"), "scene_list_rec", i);

			local proto_scene_id = nx_get_proto_scene_id(gm,nx_int(values[1]))
            if proto_scene_id <= 0 then
                nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("scene_id: " .. nx_string(values[1]) .. ", scene_config: " .. nx_string(values[2])));
            else
                nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("scene_id: " .. nx_string(values[1]) .. nx_string("(") .. nx_string(proto_scene_id) ..nx_string(")") .. ", scene_config: " .. nx_string(values[2])));
            end
		end
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("scene list end ..."));
	else
		local id = nx_int(arg[1]);
		nx_switch_born(gm, id);
	end
end

-- 查询指定场景id的玩家信息
function scene_player_info(gm, ...)
    if table.getn(arg) ~= 1 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "arg num error ...");
        return;
    end

    local proto_scene_id = nx_int(arg[1]);

    local scene_exists = nx_int(nx_scene_exists(proto_scene_id));
    if 0 == scene_exists then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "scene not exists ...");
        return;
    end

    local scene_list = nx_get_clone_scene_list(proto_scene_id);
    local scene_count = table.getn(scene_list);
    local player_count = nx_get_scene_player_count(proto_scene_id);

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "====================");

    if scene_count > 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "proto_scene_id: " .. nx_string(proto_scene_id) .. "; clone_scene_count: " .. nx_string(scene_count));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "proto_scene_id: " .. nx_string(proto_scene_id) .. "; clone_scene_count: " .. nx_string(scene_count) .. "; player_count: " .. nx_string(player_count));
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "");

    for i = 1, scene_count do
        local scene_id = scene_list[i];
        local player_count = nx_get_scene_player_count(scene_id);
        local online_count = nx_get_scene_online_count(scene_id);

        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_string(i) .. " --- scene_id: " .. nx_string(scene_id) .. "; player_count: " .. nx_string(player_count) .. "; online_count: " .. nx_string(online_count));
    end
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "====================");
end

-- 查询当前场景id
function get_cur_scene_id(gm, ...)
	local scene_id = nx_get_scene_id();

    local scene_class = nx_get_scene_class();
    if 2 == scene_class then
        local proto_id = nx_get_proto_scene_id(gm, nx_int(scene_id));
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "scene_id: " .. nx_string(scene_id) .. "; proto_id: " .. nx_string(proto_id));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "scene_id: " .. nx_string(scene_id));
    end
end

function setobjmoney(gm,...)
    --[[
	local var = nx_int(arg[1])
	if var > 0 then
    	nx_inc_capital(gm, nx_int64(1), var)
    else
    	nx_dec_capital(gm, nx_int64(1), -var)
    end
    --]]

    if nx_int64_large_int64(nx_int64(arg[1]), nx_int64(0)) then
    	nx_inc_capital(gm, 1, 1, nx_int64(arg[1]))
    else
    	nx_dec_capital(gm, 1, 1, nx_int64_dec_int64(nx_int64(0), nx_int64(arg[1])))
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set money sucess "))
end

function setchangeindex(gm,...)

	local newVar = nx_int(arg[1])
	nx_set_property(gm, "ChangeIndex", newVar)

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set ChangeIndex sucess "))

end

function setunderwaternocost(gm,...)

    if not nx_find_data(gm, "UnderWaterNoCost") then
		nx_add_data(gm, "UnderWaterNoCost", VTYPE_INT);
	end
	nx_set_data(gm, "UnderWaterNoCost", nx_int(arg[1]))
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set UnderWaterNoCost sucess "))

    return true

end

--属性操作	设置当前属性操作的对象	setobj [对象] [可选项]
--					setobj			设置自身为属性设置操作的对象
--					setobj gm		设置自身为属性设置操作的对象
--					setobj select		设置当前选中的对象为属性操作对象
--					setobj scene		设置当前场景为属性操作的对象
--					setobj child XXXXX	设置当前属性操作对象的指定名称的子对象为当前属性操作的对象
--					setobj index XXXXX	设置当前属性操作对象指定位置的子对象为当前属性操作的对象
--					setobj id XXXXX		设置当前属性操作对象为指定PERSISTID的对象
--					setobj config XXXXX	设置当前属性操作对象为指定CONFIGID的对象
--					setobj script XXXXX	设置当前属性操作对象为指定Script的对象

function setobj(gm, ...)
	if not nx_find_data(gm, "GMPropTarget") then
		nx_add_data(gm, "GMPropTarget", VTYPE_OBJECT);
	end
	if table.getn(arg) == 0 then
		nx_set_data(gm, "GMPropTarget", gm);
		--设置属性操作对象成功！
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set object property succeed!"));
		return;
	end

	local para0 = nx_string(arg[1]);
	if para0 == "gm" then
		nx_set_data(gm, "GMPropTarget", gm);
	elseif para0 == "select" then
		nx_set_data(gm, "GMPropTarget", nx_query_property(gm, "LastObject"));
	elseif para0 == "scene" then
		nx_set_data(gm, "GMPropTarget", nx_get_scene());
	elseif para0 == "id" then
		if table.getn(arg) < 2 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setobj id <objectid>"));--格式：setobj id 序列号(xx.xx)
			return;
		end
		local child = nx_object(arg[2]);
		if not nx_exists(child) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found object"));--未找到对象[{@0:名称}]
			return;
		end
		nx_set_data(gm, "GMPropTarget", child);
    elseif para0 == "name" then
		if table.getn(arg) < 2 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setobj name <player_name>"));
			return;
		end

        local player = nx_find_player(nx_widestr(arg[2]));
		if not nx_exists(player) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found player"));
			return;
		end
        nx_set_data(gm, "GMPropTarget", player);
	else
		local target = get_prop_target(gm)
		if not nx_exists(target) then
			return
		end

		if para0 == "child" then
			if table.getn(arg) < 2 then
				--格式：setobj child 子对象名称
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format：setobj child name"));
				return;
			end
			local child = nx_get_child(target, nx_widestr(arg[2]))
			if not nx_exists(child) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("not found child[{@0:名称}]"));
				return;
			end
			nx_set_data(gm, "GMPropTarget", child);
		elseif para0 == "index" then
			if table.getn(arg) < 2 then
				--格式：setobj index 序号
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setoj index <no>"));
				return;
			end
			local child = nx_get_item(target, nx_int(arg[2]))
			if not nx_exists(child) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("No object at index:" .. nx_string(arg[2])));--未找到子对象[{@0:名称}]
				return;
			end
			nx_set_data(gm, "GMPropTarget", child);
		elseif para0 == "parent" then
			local parent = nx_parent(target);
			if not nx_exists(parent) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("No parent"));--未找到父对象
				return;
			end
			nx_set_data(gm, "GMPropTarget", parent);
		elseif para0 == "config" then
			if table.getn(arg) < 2 then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setobj config <configd>"));--格式：setobj config configid
				return;
			end
			local configid = nx_int(arg[2]);
			local childs = nx_get_child_list(target, 0);
			local proptarget = nx_null();
			for i, child in pairs(childs) do
				if nx_get_config(child) == configid then
					proptarget = child;
					break;
				end
			end
			if not nx_exists(proptarget) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found child object configid is:" .. configid));--未找到子对象
				return;
			end
			nx_set_data(gm, "GMPropTarget", proptarget);
		elseif para0 == "script" then
			if table.getn(arg) < 2 then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setobj script <sctipt>"));--格式：setobj script <script>
				return;
			end
			local script = nx_string(arg[2]);
			local childs = nx_get_child_list(target, 0);
			local proptarget = nx_null();
			for i, child in pairs(childs) do
				if nx_get_script(child) == script then
					proptarget = child;
					break;
				end
			end
			if not nx_exists(proptarget) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found child object script is:" .. script));--未找到子对象
				return;
			end
			nx_set_data(gm, "GMPropTarget", proptarget);
		else
			--直接根据名称查找子对象
			local child = nx_get_child(target, nx_widestr(arg[1]))
			if not nx_exists(child) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found child object name:" .. nx_string(arg[1])));--未找到子对象[{@0:名称}]
				return;
			end
			nx_set_data(gm, "GMPropTarget", child);
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Set object property succeed!");
end

--查询当前属性操作的对象的所有子对象
function child(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local type = TYPE_PLAYER + TYPE_NPC + TYPE_ITEM;
	if table.getn(arg) > 0 then
		type = nx_int(arg[1]);
	end

	local script = "";
	if table.getn(arg) > 1 then
		script = nx_string(arg[2]);
	end

	local childlist = nx_get_child_list(target, type);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(nx_query_property(target, "Name")) .. " Child Object List:"));
	for i = 1, table.getn(childlist) do
		local child_script = nx_get_script(childlist[i])
		if string.len(script) == 0 or child_script == script then
			local child_index = nx_get_index(childlist[i])
			local child_name = nx_query_property(childlist[i], "Name")
			local config_name = nx_get_config(childlist[i])
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(child_index) .. ":" .. nx_string(child_name) .. "(" .. nx_string(childlist[i]) .. ")" .. child_script..","..config_name))
		end
	end
end

function capacity(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local cap = nx_get_capacity(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(nx_query_property(target, "Name")) .. " Capacity:" .. cap));--[{@0:名称}]容量:{@1:数值}
end

--查询当前属性操作的对象的所有属性
function property(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local proplist = nx_get_property_list(target);
	local total  = 0;
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Property List Begin:" .. "====="));
	for i = 1, table.getn(proplist) do
		--[{@0:属性}]={@1:数值}
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(proplist[i] .. "=" .. nx_string(nx_query_property(target, proplist[i]))));
		total = i;
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Property List End.total:" ..nx_string(total) .."======"));
end

--设置当前属性操作的对象的属性值
--set 属性名 属性值
function set(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set <property> <value>"));--格式：set 属性名称 属性值
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local prop = nx_string(arg[1]);
	--不能设置的属性列表
	local cant_set_prop_lst = {"GameMaster"};
	for i, prop_temp in pairs(cant_set_prop_lst) do
		if prop == prop_temp then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Can not set this property.");--不能设置此属性
			return;
		end
	end

	if not nx_find_property(target, prop) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found property:" .. prop));--未找到属性：{@0:属性名}
		return;
	end

	if table.getn(arg) == 1 then
		local type = nx_get_property_type(target, prop);
		if type == VTYPE_STRING then
			nx_set_property(target, prop, "");
		elseif type == VTYPE_WIDESTR then
			nx_set_property(target, prop, nx_widestr(""));
		else
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set <property> <value>"));--格式：set 属性名称 属性值
			return
		end
	else
		local value = arg[2];
		local type = nx_get_property_type(target, nx_string(prop));
		if type == VTYPE_INT then
			nx_set_property(target, prop, nx_int(value));

			if nx_string("Dead") == nx_string(prop) then
				nx_command(target,target,616,0)
			end

		elseif type == VTYPE_INT64 then
			nx_set_property(target, prop, nx_int64(value));
		elseif type == VTYPE_FLOAT then
			nx_set_property(target, prop, nx_float(value));
		elseif type == VTYPE_DOUBLE then
			nx_set_property(target, prop, nx_number(value));
		elseif type == VTYPE_STRING then
			nx_set_property(target, prop, nx_string(value));
		elseif type == VTYPE_WIDESTR then
			nx_set_property(target, prop, nx_widestr(value));
		elseif type == VTYPE_OBJECT then
			nx_set_property(target, prop, nx_object(value));
		end
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set " .. prop .. " succeed."));--设置属性{@0:属性名}成功
end

--查询当前属性操作的对象的属性值
--query 属性名
function query(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:query <property>"));--格式：query 属性名称
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local prop = nx_string(arg[1]);

	if not nx_find_property(target, prop) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(target) .. " --- Not found property:" .. prop));--未找到属性：{@0:属性名}
		return;
	end

	local value = nx_query_property(target, prop);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(target) .. " --- " .. prop .. "=" .. nx_string(value)));--[{@0:属性}]={@1:数值}
end

-- 查询npc的静态属性
function query_npc_prop(gm, ...)
    if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: query_npc_prop <property>"));
		return;
    end
	
    local target = get_prop_target(gm);
	if not nx_exists(target) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_query_npc_prop(target, prop);
    if value then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_get_config(target) .. " --- " .. prop .. "=" .. nx_string(value)));
    else
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("query npc prop fail ..."));
    end
end

-- 设置npc的静态属性
function set_npc_prop(gm, ...)
    if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set_npc_prop <property> <value>"));
		return;
    end
	
    local target = get_prop_target(gm);
	if not nx_exists(target) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_string(arg[2]);
    local ret = nx_set_npc_prop(target, prop, value);
    if ret == true then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set npc prop succeed: "));
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_get_config(target) .. " --- " .. prop .. "=" .. value));
    else
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set npc prop fail ..."));
    end
end

--设置当前属性操作的对象的属性值
--setdata 属性名 属性值
function setdata(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setdata <dataname> <value>"));--格式：set 属性名称 属性值
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local prop = nx_string(arg[1]);
	local value = arg[2];

	if not nx_find_data(target, prop) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found data named:" .. prop));--未找到属性：{@0:属性名}
		return;
	end

	local type = nx_get_data_type(target, nx_string(prop));
	if type == VTYPE_INT then
		nx_set_data(target, prop, nx_int(value));
	elseif type == VTYPE_INT64 then
		nx_set_data(target, prop, nx_int64(value));
	elseif type == VTYPE_FLOAT then
		nx_set_data(target, prop, nx_float(value));
	elseif type == VTYPE_DOUBLE then
		nx_set_data(target, prop, nx_number(value));
	elseif type == VTYPE_STRING then
		nx_set_data(target, prop, nx_string(value));
	elseif type == VTYPE_WIDESTR then
		nx_set_data(target, prop, nx_widestr(value));
	elseif type == VTYPE_OBJECT then
		nx_set_data(target, prop, nx_object(value));
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set data " .. prop .. " succeed."));--设置Data{@0:属性名}成功
end

--查询当前属性操作的对象的属性值
--querydata 属性名
function querydata(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:querydata <dataname>"));--格式：querydata Data名称
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local prop = nx_string(arg[1]);
	if not nx_find_data(target, prop) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found data named:" .. prop));--未找到属性：{@0:属性名}
		return;
	end

	local value = nx_query_data(target, prop);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(prop.. "=" .. nx_string(value)));--[{@0:属性}]={@1:数值}
end

--查询当前属性操作的对象的所有表格名
function record(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local total = 0
	local reclist = nx_get_record_list(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Record List Begin:" .. "====="));
	for i = 1, table.getn(reclist) do
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(reclist[i]));
		total = i
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Record List End.total:" ..nx_string(total).. "======"));
end

--设置当前属性操作的对象的表格值
--recordset 表格名 行号 列号 值
function recordset(gm, ...)
	if table.getn(arg) < 4 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordset <recname> <row> <col> <value>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	local row = nx_int(arg[2]);
	local col = nx_int(arg[3]);
	local value = arg[4];

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname));
		return;
	end

	local record_rows = nx_get_record_rows(target,recname)
	if row >= nx_int(record_rows) or row < 0 then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record rows: " .. recname));
	end

	local type = nx_get_record_coltype(target, recname, col);
	if type == VTYPE_INT then
		nx_set_record_value(target, recname, row, col, nx_int(value));
	elseif type == VTYPE_INT64 then
		nx_set_record_value(target, recname, row, col, nx_int64(value));
	elseif type == VTYPE_FLOAT then
		nx_set_record_value(target, recname, row, col, nx_float(value));
	elseif type == VTYPE_DOUBLE then
		nx_set_record_value(target, recname, row, col, nx_double(value));
	elseif type == VTYPE_STRING then
		nx_set_record_value(target, recname, row, col, nx_string(value));
	elseif type == VTYPE_WIDESTR then
		nx_set_record_value(target, recname, row, col, nx_widestr(value));
	elseif type == VTYPE_OBJECT then
		nx_set_record_value(target, recname, row, col, nx_object(value));
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set record " .. recname .. " value succeed!"));
end

--查询当前属性操作的对象的表格内容
--recordquery 表格名
function recordquery(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordquery <recname>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname));
		return;
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. nx_string(nx_query_property(target, "Name")) .. ":" .. recname .. "================"));
	local cols = nx_get_record_cols(target, recname);
	local rows = nx_get_record_rows(target, recname);
	for row = 0, rows - 1 do
		local text = nx_string(row) .. "|";
		for col = 0, cols - 1 do
			text = text .. nx_string(nx_query_record_value(target, recname, row, col)) .. "|";
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text));
	end
end

-- 格式化当前属性操作的对象的表格内容
-- recordshow 表格名
function recordshow(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordshow <recname>"))
		return
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1])
	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname))
		return
	end

	-- 打印表名
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================"
	 .. nx_string(nx_query_property(target, "Name")) .. ":" .. recname .. "================"))

	-- 宽度表
	local width_table = {}

	-- 遍历取得每列的最大宽度
	local rows = nx_get_record_rows(target, recname)
	local cols = nx_get_record_cols(target, recname)
	local total_width = cols + 1

	for i = 0, cols - 1 do
		local max_width = 1

		for j = 0, rows - 1 do
			local width = string.len(tostring(nx_query_record_value(target, recname, j, i)))
			if width > max_width then
				max_width = width
			end
		end

		width_table[i] = max_width
		total_width = total_width + max_width
	end

	-- 行分割线
	local row_split_text = ""
	for e = 1, total_width do
		row_split_text = row_split_text .. "-"
	end

	-- 格式化输出表内容
	for row = 0, rows - 1 do
		-- 分割线
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(row_split_text))

		local text = "|"

		for col = 0, cols - 1 do
			local value = nx_string(nx_query_record_value(target, recname, row, col))
			text = text .. value
			local len = string.len(value)
			local need_space = width_table[col] - len
			for k = 1, need_space do
				text = text .. " "
			end
			text = text .. "|"
		end

		-- 输出
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text))
	end

	-- 分割线
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(row_split_text))
end

--添加一行
--recordaddrow 表格名 ...
function recordaddrow(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Format: recordaddrow recname <row> ...");
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record " .. recname));
		return;
	end

	local cols = nx_get_record_cols(target, recname);
	if table.getn(arg) < (1 + cols) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Record " .. recname .. " columns is " .. cols));
		return;
	end

	local row = nx_add_record_row(target, recname);

	for col = 0, cols - 1 do
		local coltype = nx_get_record_coltype(target, recname, col);
		if coltype == VTYPE_INT then
			nx_set_record_value(target, recname, row, col, nx_int(arg[col + 2]));
		elseif coltype == VTYPE_INT64 then
			nx_set_record_value(target, recname, row, col, nx_int64(arg[col + 2]));
		elseif coltype == VTYPE_FLOAT then
			nx_set_record_value(target, recname, row, col, nx_float(arg[col + 2]));
		elseif coltype == VTYPE_DOUBLE then
			nx_set_record_value(target, recname, row, col, nx_number(arg[col + 2]));
		elseif coltype == VTYPE_STRING then
			nx_set_record_value(target, recname, row, col, nx_string(arg[col + 2]));
		elseif coltype == VTYPE_WIDESTR then
			nx_set_record_value(target, recname, row, col, nx_widestr(arg[col + 2]));
		elseif coltype == VTYPE_OBJECT then
			nx_set_record_value(target, recname, row, col, nx_object(arg[col + 2]));
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Add row succeed!"));
end

--删除表格的指定行的数据
--recordremoverow 表格名 行号
function recordremoverow(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordremoverow recname row"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	local row = nx_int(arg[2]);

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Not found record " .. recname);
		return;
	end

	if not nx_remove_record_row(target, recname, row) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Remove record " .. recname .. " row " .. row .. " failed!");
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Remove record " .. recname .. " row " .. row .. " succeed!");
	end
end

--清除指定表格的内容
--recordclear 表格名
function recordclear(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Format: recordclear recname");
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Not found record " .. recname);
		return;
	end

	if not nx_clear_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Clear record " .. recname .. " failed!");
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Clear record " .. recname .. " succeed!");
	end
end

--设置表格标记位
--recordsetflag 表格名 位数
function recordsetflag(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordsetflag <recname> <pos>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	local pos = nx_int(arg[2]);

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname));
		return;
	end

	if not nx_set_record_flag(target, recname, pos) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Set record " .. recname .. " flag failed!");
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Set record " .. recname .. " flag succeed!");
	end
end

--清空表格标记位
--recordclearflag 表格名 位数
function recordclearflag(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordsetflag <recname> <pos>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	local pos = nx_int(arg[2]);

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname));
		return;
	end

	if not nx_clear_record_flag(target, recname, pos) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Clear record " .. recname .. " flag failed!");
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Clear record " .. recname .. " flag succeed!");
	end
end

--测试表格标记位
--recordtestflag 表格名 位数
function recordtestflag(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: recordtestflag <recname> <pos>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local recname = nx_string(arg[1]);
	local pos = nx_int(arg[2]);

	if not nx_find_record(target, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. recname));
		return;
	end

	if not nx_test_record_flag(target, recname, pos) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Test record " .. recname .. " flag failed!");
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Test record " .. recname .. " flag succeed!");
	end
end

--查询当前属性操作的对象的创建脚本和配置文件名称
function see(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Type: " .. nx_type(target) .. " Script: " .. nx_get_script(target) .. " Config:" .. nx_get_config(target)));
end

--选择当前属性操作对象为选中对象
--select 对象名称
function select(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	nx_set_property(gm, "LastObject", target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Selected " .. nx_string(nx_query_property(target, "Name")) .. " succeed!"));
end

--对当前选中对象执行doselect操作
function doselect(gm, funcid)
	local target = nx_query_property(gm, "LastObject");
	if nx_exists(target) then
		nx_select(gm, target, nx_int(funcid))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("failed, last object not exists..."));
	end
end

--在当前属性操作对象中，创建一个子游戏对象
--cobjchild 对象ID
function cobjchild(gm, ...)

	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: cobjchild <id> [<amount>]"));--格式：cobjchild id
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local id = nx_string(arg[1]);
	local amount = 1;
	if table.getn(arg) > 1 then
		amount = nx_int(arg[2]);
	end

	local item = nx_create_from_config(nx_get_scene(), nx_string(id))
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"));
		return
	end
	if nx_find_property(item, "Amount") then
	   nx_set_property(item, "Amount",amount)
	end

	if nx_container_can_placeitem(target,item) > 0 then
	    nx_container_placeitem(target,item)
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));
	else
	    nx_destroy_self(item)
	end
end

--添加附带打孔等信息的装备
function cobjequip(gm, ...)
    if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: cobjequip <id> [<amount>]"));--格式：cobjchild id
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local id = nx_string(arg[1]);

	local amount = 1;
	local hole = 0--打孔数量
	local star = 0--强化等级

	if table.getn(arg) > 1 then
	   amount = nx_int(arg[2]);
	end

	if table.getn(arg) > 2 then
	   hole = nx_int(arg[3]);
	end

	if table.getn(arg) > 3 then
	   star = nx_int(arg[4]);
	end

	for i=1,amount do
        local item = nx_create_from_config(nx_get_scene(), id)


        if nx_container_can_placeitem(target,item) > 0 then
            nx_container_placeitem(target,item)
            nx_add_ranpro(item)

            nx_set_property(item,"CurHoleCount",hole)
            nx_strengthen(item,nx_int(star) )
            nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));
        else
            nx_inc_property(item, "Amount", -1)
            return
        end
	end

end

--添加可叠加的ITEM
function cobjadditem(gm, ...)
    if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: cobjequip <id> [<amount>]"));--格式：cobjchild id
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local id = nx_string(arg[1]);
	local amount = 1;
	local count=1

	if table.getn(arg) > 1 then
	   amount = nx_int(arg[2]);
	end

	local item = nx_create_from_config(nx_get_scene(), id)

	local maxamount    = nx_query_property(item, "MaxAmount")
	local singleamount = amount--nx_query_property(item, "Amount")

	count = amount/maxamount

	for i=1,count do
	    local item = nx_create_from_config(nx_get_scene(), id)
	    if nx_container_can_placeitem(target,item) > 0 then
	        nx_set_property(item,"Amount",maxamount)
	        nx_container_placeitem(target,item)
	    end
	end

	nx_set_property(item,"Amount",singleamount%maxamount)
	nx_container_placeitem(target,item)

end

--添加随机属性道具
function cobjchild1(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: cobjchild <id> [<amount>]"));--格式：cobjchild id
		return
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local id = nx_string(arg[1])
	local amount = 1;
	if table.getn(arg) > 1 then
	   amount = nx_int(arg[2])
	end

	local item = nx_create_from_config(nx_get_scene(), nx_string(id))
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"))
		return
	end
	if nx_find_property(item, "Amount") then
	   nx_set_property(item, "Amount",amount)
	end
	nx_add_ranpro(item)
	if nx_container_placeitem(target, item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"))
	else
	   nx_destroy_self(item)
	end
end

--创建一个游戏对象
--cobj 对象ID
function cobj(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:cobj <ID> [<amount>]"));--格式：cobj 编号
		return;
	end

	local x,y,z,o = nx_get_position(gm);
	local id = nx_string(arg[1]);
	local amount = 1;
	if table.getn(arg) > 1 then
	   amount = nx_int(arg[2]);
	end

	local item = nx_create_obj(gm, id, 0, x, y, z, o);
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"));
	else
		if nx_find_property(item, "Amount") then
			nx_set_property(item, "Amount",amount);
	   end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));
	end
end

--于目标处创建对象
function cobjthere(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:cobjthere <ID> [<amount>]"));--格式：cobj 编号
		return;
	end

	local target = nx_query_property(gm, "LastObject")
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not selected any object"));
		return;
	end

	local x, y, z, orient = nx_get_position(target);
	local id = nx_string(arg[1])
	local amount = 1;
	if table.getn(arg) > 1 then
	   amount = nx_int(arg[2]);
	end
	local item = nx_create_obj(id, 0, x, y, z, orient);
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"));
	else
		if nx_find_property(item, "Amount") then
			nx_set_property(item, "Amount",amount);
	   end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));
	end
end

--删除当前选中的对象
function del(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end
	if target ~= gm then
		if nx_destroy_self(target) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Delete selected object succeed!");
		else
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "Delete selected object failed!");
		end
	end
end

--将当前选中对象移动到GM自己这里
function here(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local x, y, z, orient = nx_get_position(gm);
	nx_moveto(target, x, y, z, orient);
end

function motionhere(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local x, y, z, orient = nx_get_position(gm);
	local runspeed = nx_query_property(target, "RunSpeed");
	local anglespeed = nx_float(6.28);
	nx_motion(target, runspeed, anglespeed, x, z);
end

function motionthere(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local x, y, z, orient = nx_get_position(target);
	local runspeed = nx_query_property(gm, "RunSpeed");
	local anglespeed = nx_float(6.28);
	nx_motion(gm, runspeed, anglespeed, x, z);
end

--自己瞬移到选定目标的坐标: there
function there(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not selected any object"));
		return;
	end

	local x, y, z, orient = nx_get_position(target);
	nx_moveto(gm, x, y, z, orient);
end

--指定玩家瞬移到自己的坐标: herebyname [玩家名]
function herebyname(gm, ...)
	if table.getn(arg) == 0 then
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, 0, 1949);--格式：herebyname 玩家名
		--return;
		arg[1] = getcurobjname(gm);
	end
	local target = nx_find_player(arg[1]);
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Please use [setobj] command first."));
		return;
	end

	local x,y,z,o = nx_get_position(gm);
	nx_moveto(target, x, y, z, orient);
end

--自己瞬移到指定玩家的坐标: therebyname [玩家名]
function therebyname(gm, ...)
	if table.getn(arg) == 0 then
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, 0, 1950);--格式：therebyname 玩家名
		--return;
		arg[1] = getcurobjname(gm);
	end
	local target = nx_find_player(arg[1]);
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Please use [setobj] command first."));
		return;
	end
	local x,y,z,o = nx_get_position(target);
	nx_moveto(gm, x, y, z, orient);
end

--瞬移到某个点: goto [x] [z]
function goto(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: goto <x> <z>"));
		return;
	end

	local x, y, z, orient = nx_get_position(gm);
	x = nx_float(arg[1]);
	z = nx_float(arg[2]);
	y = nx_get_height(x, z);

	if nx_moveto(gm, x, y, z, orient) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("goto to " .. nx_int(x) .. " " .. nx_int(z) .. " succeed." ));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("goto to " .. nx_int(x) .. " " .. nx_int(z) .. " failed." ));
	end
end

--移动到某个点: motionto [x] [z]
function motionto(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: motionto <x> <z>"));
		return;
	end

	local x = nx_float(arg[1]);
	local z = nx_float(arg[2]);

	local runspeed = nx_query_property(gm, "MoveSpeed");
	local anglespeed = nx_float(math.pi * 2);

	if nx_motion(gm, runspeed, anglespeed, x, z) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("motion to " .. nx_int(x) .. " " .. nx_int(z) .. " succeed." ));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("motion to " .. nx_int(x) .. " " .. nx_int(z) .. " failed." ));
	end
end

--前移一段距离: goahead [距离]
function goahead(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: goahead <distance>"));
		return;
	end

	local dist = nx_float(arg[1]);
	local x, y, z, orient = nx_get_position(gm);

	local destx = nx_float(nx_number(x) + nx_number(dist)*math.sin(nx_number(orient)));
	local destz = nx_float(nx_number(z) + nx_number(dist)*math.cos(nx_number(orient)));
	y = nx_get_height(destx, destz);

	--在移动之前先检查目标点可否走(待添加)
	nx_moveto(gm, destx, y, destz, orient);
end

--当前选中对象前移一段距离: goaheadobj [距离]
function goaheadobj(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: goaheadobj <distance>"));
		return;
	end

	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local dist = nx_float(arg[1]);
	local x, y, z, orient = nx_get_position(target);

	local destx = nx_float(nx_number(x) + nx_number(dist)*math.sin(nx_number(orient)));
	local destz = nx_float(nx_number(z) + nx_number(dist)*math.cos(nx_number(orient)));
	y = nx_get_height(destx, destz);

	--在移动之前先检查目标点可否走(待添加)
	nx_moveto(target, destx, y, destz, orient);
end

--查询GM当前位置信息
function pos(gm)
	local x, y, z, orient = nx_get_position(gm);
	local height = nx_get_height(x, z);

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("X:" .. nx_number(x) .. " Y:" .. nx_number(y) .. " Z:" .. nx_number(z) .. " Orient:" .. nx_number(orient)));
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Map Height:" .. nx_number(height)));
end

--查询选中对象的位置信息
function posobj(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local x, y, z, orient = nx_get_position(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("X:" .. nx_number(x) .. " Y:" .. nx_number(y) .. " Z:" .. nx_number(z) .. " Orient:" .. nx_number(orient)));
end

--查询与当前选中对象的距离
function dist(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local distance = nx_distance(gm, target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Distance:" .. nx_number(distance)));
end

--查询与当前选中对象的方向
function angle(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local angle = nx_angle(gm, target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Angle:" .. nx_number(angle)));
end

--满HP: hp
function hp(gm)
	nx_set_property( gm, "HP", nx_query_property(gm,"MaxHP") +  nx_query_property(gm,"MaxHPAdd"));
	return 1
end

--满MP: mp
function mp(gm)
	nx_set_property( gm, "MP", nx_query_property(gm,"MaxMP") + nx_query_property(gm,"MaxMPAdd"));
	return 1
end

--满RC: rc
function rc(gm)
	nx_set_property( gm, "RC", nx_query_property(gm,"MaxRC") + nx_query_property(gm,"MaxRCAdd"));
	return 1
end

--满EY: ey
function ey(gm)
	nx_set_property( gm, "EY", nx_query_property(gm,"MaxEY") + nx_query_property(gm,"MaxEYAdd"));
	return 1
end

--满AV: av
function av(gm)
	nx_set_property( gm, "AV", nx_query_property(gm,"MaxAV") + nx_query_property(gm,"MaxAVAdd"));
	return 1
end

--满FS: fs
function fs(gm)
	nx_set_property( gm, "FS", nx_query_property(gm,"MaxFS") + nx_query_property(gm,"MaxFSAdd"));
	return 1
end

--满DP: dp
function dp(gm)
	nx_set_property( gm, "DP", nx_query_property(gm,"MaxDP") + nx_query_property(gm,"MaxDPAdd"));
	return 1
end

--设置指定属性值
function propValue(gm,...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set <property> <value>"));--格式：set 属性名称 属性值
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local prop_name = nx_string(arg[1])
	local type = nx_get_property_type(target, prop_name)
	if type == VTYPE_INT then
		local value = nx_int(arg[2])
		nx_set_property( gm, prop_name, value );
	elseif type == VTYPE_FLOAT then
		local value = nx_float(arg[2])
		nx_set_property( gm, prop_name, value );
	end

	return 1
end

--移到某个玩家所在坐标: goplayer [玩家名]
function goplayer(gm, ...)
	if table.getn(arg) == 0 then
		--格式：
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, 0, 1943);--格式：goplayer 玩家名称
		--return;
		arg[1] = getcurobjname(gm);
	end

	player = nx_find_player( arg[1] );
	if not nx_exists(player) then
		scene_id = nx_get_player_scene(arg[1]); --取玩家场景号
		if scene_id > 0 then
			--[{@0:名称}]在{@1:数值}号场景中，请先切换到该场景
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(arg[1]) .. " is in scene:" .. scene_id));
		else
			local serverid = nx_get_player_serverid(arg[1]);
			if (serverid > 0) then
				local servername = nx_get_servername(serverid);
				--[{@0:名称}]在[{@1:服务器名}]
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(arg[1]) .. " is in server:" .. nx_string(servername)));
			else
				--找不到名称为[{@0:名称}]的玩家
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("can't find player:" .. nx_string(arg[1])));
			end
		end
	else
		local x, y, z, orient = nx_get_position(player);
		nx_moveto(gm, x, y, z, orient);
	end
end

--查询在线玩家数目: playernum
function playernum(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("player count:" .. nx_number(nx_get_player_count())));
end

--把选择目标移动到某个场景
function movescene(gm, ...)
    local target = nx_query_property(gm, "LastObject")

    if table.getn(arg) > 0 then
        local id = nx_int(arg[1])
        nx_switch_born(target, id);
	end
end
--把选择目标移动到某个点
function moveto(gm, ...)
    local target = nx_query_property(gm, "LastObject")

    if table.getn(arg) > 1 then
        local x = nx_int(arg[1])
        local z = nx_int(arg[2])
	    local y = nx_get_height(x, z);

        if nx_moveto(target, x, y, z, 1) then
        end

     end
end

--隐身: hide
function hide(gm)
	--if nx_query_property(gm,"Invisible") == 0 then
	--	nx_set_property(gm,"Invisible", 1);
	if nx_query_property(gm,"RoleHide") == 0 then
		nx_set_property(gm,"RoleHide", 1);
		--开始隐身
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Begin hide."));
	else
		--已经处于隐身状态中
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Begin hide."));
	end
end

--现身: show
function show(gm)
	--if nx_query_property(gm,"Invisible") == 1 then
	--	nx_set_property(gm,"Invisible", 0);
	if nx_query_property(gm,"RoleHide") == 1 then
		nx_set_property(gm,"RoleHide", 0);
		--开始现身
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("End hide."));
	else
		--已经处于现身状态中
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("End hide."));
	end
end

--踢人: kick [玩家名] <XXX分钟>
function kick(gm, ...)
	if table.getn(arg) == 0 then
		--格式：
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, 0, 1945);--格式：kick 玩家名称 分钟
		--return;
		arg[1] = getcurobjname(gm);
	end

	local minute = 0;
	if table.getn(arg) == 2 then
		minute = nx_int(arg[2]);
	end

	if minute > 0 then
		nx_block_player(arg[1], minute);
	end

	if nx_break_byname(arg[1]) then
		--将[{@0:名称}]踢下线了
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("kick off " .. nx_string(arg[1])));
	end
end

--踢账号：logout [账号]
function logout(gm, ...)
	if table.getn(arg) == 0 then
		--格式：
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: logout account"));
		return;
	end

	if nx_break_account(nx_string(arg[1])) then
		--将[{@0:名称}]踢下线了
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("kick out account " .. nx_string(arg[1]) .. " succeed."));
	else
		--失败
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("kick out account " .. nx_string(arg[1]) .. " failed."));
	end
end

--广播消息: broadcast [消息]
function broadcast(gm, ...)
	if table.getn(arg) == 0 then
		--格式：
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("broadcast info"));--格式：broadcast 内容（不能包含空格）
		return;
	end

	local info = arg[1];
	nx_sysinfo_broadcast(nx_int(TIPSTYPE_IMPORTANTINFO), "0", info);
end

--广播消息: broadcast [消息]
function bc(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("bc info"));--格式：bc 内容（不能包含空格）
		return;
	end

	local info = arg[1];
	nx_sysinfo_broadcast(nx_int(TIPSTYPE_IMPORTANTINFO), "0", info);
end

--广播消息: bctest [消息]
function bctest(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format bctest contet(can't include space)"));--格式：bctest 内容（不能包含空格）
		return;
	end

	local info = arg[1];
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", info);
end

--广播消息: bchaha [消息]
function bchaha(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format bchaha stringid para1, ..., paran"));--格式：bctest 内容（不能包含空格）
		return;
	end

	local stringid = nx_string(arg[1]);
	if table.getn(arg) == 1 then
		nx_sysinfo_broadcast(TIPSTYPE_TIPSINFO, stringid);
	elseif table.getn(arg) == 2 then
		nx_sysinfo_broadcast(TIPSTYPE_TIPSINFO, stringid, arg[2]);
	elseif table.getn(arg) == 3 then
		nx_sysinfo_broadcast(TIPSTYPE_TIPSINFO, stringid, arg[2], arg[3]);
	elseif table.getn(arg) == 4 then
		nx_sysinfo_broadcast(TIPSTYPE_TIPSINFO, stringid, arg[2], arg[3], arg[4]);
	elseif table.getn(arg) == 5 then
		nx_sysinfo_broadcast(TIPSTYPE_TIPSINFO, stringid, arg[2], arg[3], arg[4], arg[5]);
	end
end

--自己做动作: action [动作名]
function action(gm, action_name)
	nx_custom_byken(gm, SERVER_CUSTOMMSG_ACTION, gm, nx_string(action_name));
end

--选择目标做动作
function actionobj(gm, action_name)
	local target = nx_query_property(gm, "LastObject");
	nx_custom_byken(target, SERVER_CUSTOMMSG_ACTION, target, nx_string(action_name));
end

--禁言玩家：silence [玩家名] [分钟]
function silence(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:silence <rolename> <minute>"));--格式：silence 玩家名 分钟
		return;
	elseif table.getn(arg) < 2 then
		arg[2] = arg[1];
		arg[1] = getcurobjname(gm);
	end
	local playername = arg[1];
	local minute = arg[2];
	local tick = nx_int(minute) * 60 * 1000;

	local COMMAND_CHAT = 40;
	local CHATTYPE_SILENCE = 0;
	nx_command_byname(playername, COMMAND_CHAT, CHATTYPE_SILENCE, tick);
	if minute == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("[".. nx_string(playername) .. "] has released from silence."));--[{@0:名称}]已被解除禁言
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("[".. nx_string(playername) .. "] has silenced " .. nx_number(minute) .. " minute."));--[{@0:名称}]已被禁言{@1:分钟}分钟
	end
end

--测试某点是否可走
function canwalk( gm, x, z )
	if nx_can_walk( nx_float(x), nx_float(z) ) then
		nx_gminfo( gm, "0", nx_widestr("Can walk."));
	else
		nx_gminfo( gm, "0", nx_widestr("Can not walk."));
	end
end

--测试到某点之间是否可走
function canwalkto(gm, ...)
	if table.getn(arg) ~= 2 then
		nx_gminfo( gm, "0", nx_widestr("canwalkto: linecanwalk x z."));
		return;
	end

	local x = nx_float(arg[1]);
	local z = nx_float(arg[2]);

	if nx_obj_line_can_walk(gm, x, z) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Can walk."));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Can not walk."));
	end
end

--重新载入当前属性操作对象的配置文件
function reload(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if not nx_load_config(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Reload config file failed!"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Reload config file succeed!"));
	end
end

function state(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("state = " .. nx_get_state(target)));
	else
		if nx_set_state(target, nx_string(arg[1])) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(nx_query_property(target, "Name")) .. " set new state is " .. nx_string(arg[1])));
		else
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set state failed!"));
		end

	end
end

--查询公会所有的属性名称
--guildproperty 公会名
function guildproperty(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: guildproperty <guildname>"));
		return;
	end

	local guild = nx_widestr(arg[1]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild: " .. nx_string(guild)));
		return;
	end

	local result = nx_get_guild_property_list(guild);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. nx_string(guild) .. " property list" .. "================"));
	for i = 1, table.getn(result) do
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(result[i]));
	end
end

--查询公会所有的表格名称
--guildrecord 公会名
function guildrecord(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: guildrecord <guildname>"));
		return;
	end

	local guild = nx_widestr(arg[1]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild:" .. nx_string(guild)));
		return;
	end

	local result = nx_get_guild_rec_list(guild);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. nx_string(guild) .. "record list" .. "================"));
	for i = 1, table.getn(result) do
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(result[i]));
	end
end

--查询公会表格的内容
--guildrecordquery 公会名 表格名
function guildrecordquery(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: guildrecordquery <guildname> <recname>"));
		return;
	end

	local guild = nx_widestr(arg[1]);
	local recname = nx_string(arg[2]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild: " .. nx_string(guild)));
		return;
	end
	if not nx_find_guild_rec(guild, recname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild record: " .. nx_string(recname)));
		return;
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. nx_string(guild) .. ":" .. recname .. "================"));
	for i = 0, nx_get_guild_rec_rows(guild, recname) - 1 do
		local text = nx_string(i) .. "|";
		local result = nx_query_guild_rec_row_values(guild, recname, i);
		for j = 1, table.getn(result) do
			text = text .. nx_string(result[j]) .. "|";
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text));
	end
end

function guildquery(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: guildquery <guildname> <property>"));
		return;
	end

	local guild = nx_widestr(arg[1]);
	local prop = nx_string(arg[2]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild: " .. nx_string(guild)));
		return;
	end

	local value = nx_query_guild_property(guild, prop);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(prop .. ":" .. nx_string(value)));
end

--查询公会的UID
function queryguildid(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: queryguildid <guildname>"));
		return;
	end

	local guild = nx_widestr(arg[1]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild: " .. nx_string(guild)));
		return;
	end

	local id = nx_guild_getid(guild);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(guild) .. " uid:" .. nx_string(id)));
end

function guildinc(gm, ...)
	if table.getn(arg) < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: guildinc <guildname> <property> <inc>"));
		return;
	end

	local guild = arg[1];
	local prop = nx_string(arg[2]);
	local value = arg[3];

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found guild: " .. nx_string(guild)));
		return;
	end

	local type = nx_get_guild_property_type(guild, prop);
	local val;
	if type == 2 then
		val = nx_int(value);
	elseif type == 3 then
		val = nx_float(value);
	elseif type == 4 then
		val = nx_string(value);
	elseif type == 5 then
		val = nx_widestr(value);
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "error");
		return;
	end

	local res = nx_inc_guild_property(guild, nx_query_property(gm,"Name"), prop, val, 13081);
end

function guildset(gm, ...)
	if table.getn(arg) ~= 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "format:guildset prop val");
		return;
	end
	local guild = arg[1];
	local prop = nx_string(arg[2]);
	local value = arg[3];
	local type = nx_get_guild_property_type(guild, prop);
	local val;
	if type == 2 then
		val = nx_int(value);
	elseif type == 3 then
		val = nx_float(value);
	elseif type == 4 then
		val = nx_string(value);
	elseif type == 5 then
		val = nx_widestr(value);
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "error");
		return;
	end

	local res = nx_set_guild_property(guild, nx_query_property(gm,"Name"), prop, val);
	--nx_sysinfo(gm, TIPSTYPE_GMINFO,1,nx_string(res));
end

function guildrec_removerow(gm, ...)
	if table.getn(arg) ~= 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "format:guildremoverecrow guild recname row");
		return;
	end

	local guild = arg[1];
	local rec = nx_string(arg[2]);
	local row = nx_int(arg[3]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this guild");
		return;
	end

	if not nx_find_guild_rec(guild, rec) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this record");
		return;
	end

	if row < 0 or row >= nx_get_guild_rec_rows(guild, rec) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "the row exceed range");
		return;
	end

	local res = nx_remove_guild_rec_row(guild, nx_query_property(gm,"Name"), rec, row);
	--nx_sysinfo(gm, TIPSTYPE_GMINFO,1,nx_string(res));
end

function guildrec_addrow(gm, ...)
	if table.getn(arg) < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "format:guildrec_addrow guild recname ...");
		return;
	end

	local guild = arg[1];
	local rec = nx_string(arg[2]);

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this guild");
		return;
	end

	if not nx_find_guild_rec(guild, rec) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this record");
		return;
	end

	local cols = nx_get_guild_rec_cols(guild, rec);
	if cols > 15 or (table.getn(arg) < 2 + cols) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "arg num not match the record cols");
		return;
	end

	local v = {};
	for i = 0, cols - 1 do
		local coltype = nx_get_guild_rec_coltype(guild, rec, i);
		if coltype == VTYPE_INT then
			v[i + 1] = nx_int(arg[3 + i]);
		elseif coltype == VTYPE_FLOAT then
			v[i + 1] = nx_float(arg[3 + i]);
		elseif coltype == VTYPE_STRING then
			v[i + 1] = nx_string(arg[3 + i]);
		elseif coltype == VTYPE_WIDESTR then
			v[i + 1] = arg[3 + i];
		elseif coltype == VTYPE_OBJECT then
			v[i + 1] = nx_object(arg[3 + i]);
		end
	end

	local name = nx_query_property(gm, "Name");
	if cols == 1 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1]);
	elseif cols == 2 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2]);
	elseif cols == 3 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3]);
	elseif cols == 4 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4]);
	elseif cols == 5 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5]);
	elseif cols == 6 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6]);
	elseif cols == 7 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7]);
	elseif cols == 8 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8]);
	elseif cols == 9 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9]);
	elseif cols == 10 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10]);
	elseif cols == 11 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11]);
	elseif cols == 12 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11], v[12]);
	elseif cols == 13 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11], v[12], v[13]);
	elseif cols == 14 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11], v[12], v[13], v[14]);
	elseif cols == 15 then
		nx_add_guild_rec_row_values(guild, name, rec, v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9], v[10], v[11], v[12], v[13], v[14], v[15]);
	end

	--nx_sysinfo(gm, TIPSTYPE_GMINFO,1,nx_string(res));
end

function guildrecset(gm, ...)
	if table.getn(arg) < 5 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "format:guildrec_addrow guild recname ...");
		return;
	end

	local guild = arg[1];
	local rec = nx_string(arg[2]);
	local row = nx_int(arg[3]);
	local col = nx_int(arg[4]);
	local value = arg[5];

	if not nx_find_guild(guild) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this guild");
		return;
	end

	if not nx_find_guild_rec(guild, rec) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "not found this record");
		return;
	end

	local rows = nx_get_guild_rec_rows(guild, rec);
	local cols = nx_get_guild_rec_cols(guild, rec);

	if row < 0 or row >= rows then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "the row not exists!");
		return;
	end

	if col < 0 or col >= cols then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "the col not exists!");
		return;
	end

	local coltype = nx_get_guild_rec_coltype(guild, rec, col);
	if coltype == VTYPE_INT then
		v = nx_int(value);
	elseif coltype == VTYPE_FLOAT then
		v = nx_float(value);
	elseif coltype == VTYPE_STRING then
		v = nx_string(value);
	elseif coltype == VTYPE_WIDESTR then
		v = value;
	elseif coltype == VTYPE_OBJECT then
		v = nx_object(value);
	end

	local name = nx_query_property(gm, "Name");
	nx_set_guild_rec_value(guild, name, rec, row, col, v);
end

function testradomint(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format : testradomint <scale> <times>"));
		return;
	end
	local scale = nx_int(arg[1]);
	local timer = nx_int(arg[2]);
	local count_0 = 0;
	for i = 1, timer do
		if (nx_randint(scale) == 0) then
			count_0 = count_0 + 1;
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("testradomint end, generate " .. timer .. " numbers"));
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("scale : " .. scale .. ". mininum number appear times : " .. count_0));
end

--给当前属性操作对象加一个心跳
function addhb(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format : addhb <function name> <interval>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local funcname = nx_string(arg[1]);
	local sep = nx_int(arg[2]);

	if nx_add_heartbeat(target, funcname, sep) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add heartbeat " .. funcname .. "success !"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add heartbeat " .. funcname .. "failed !"));
	end
end

--删除当前属性操作对象的一个心跳
function removehb(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format : removehb <function name>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local funcname = nx_string(arg[1]);
	if nx_remove_heartbeat(target, funcname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove heartbeat " .. funcname .. "success !"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove heartbeat " .. funcname .. "failed !"));
	end
end

--
function servertime(gm)
	local time = os.time();
	local timeTab = os.date("*t",time)

	local sInfo = "time: ";
	sInfo = sInfo .. timeTab.year .. "." .. timeTab.month .. "." .. timeTab.day .. " ";
	sInfo = sInfo .. timeTab.hour .. ":" .. timeTab.min .. ":" .. timeTab.sec
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(sInfo));
end

function pubdataquery(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubdataquery <pubname> <recname>"));
		return;
	end

	local pubname = arg[1];
	local dataname = nx_string(arg[2]);

	if not nx_find_pub_data(pubname, dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found data: " .. dataname));
		return;
	end

	local value = nx_query_pub_data(pubname, dataname);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(value));
end

function pubdataset(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubdataquery <pubname> <recname>"));
		return;
	end

	local pubname = arg[1];
	local dataname = nx_string(arg[2]);

	if not nx_find_pub_data(pubname, dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found data: " .. dataname));
		return;
	end

	local value;
	local data_type = nx_get_pub_data_type(pubname, dataname);
	if data_type == VTYPE_INT then
		value = nx_int(arg[3]);
	elseif data_type == VTYPE_FLOAT then
		value = nx_float(arg[3]);
	elseif data_type == VTYPE_STRING then
		value = nx_string(arg[3]);
	elseif data_type == VTYPE_WIDESTR then
		value = arg[3];
	elseif data_type == VTYPE_OBJECT then
		value = nx_ex_object(arg[3]);
	end

	nx_set_pub_data(pubname, dataname, value);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(value));
end

--公共数据表查询
function pubrecquery(gm, ...)
	if table.getn(arg) < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubrecquery <pub_space_name> <pub_data_name> <rec_name>"));
		return;
	end

	local pub_space_name = nx_string(arg[1]);
	local pub_data_name = nx_widestr(arg[2]);
	local rec_name = nx_string(arg[3]);

	if not nx_find_pubdata_record(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. rec_name));
		return;
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. rec_name .. "================"));

	local rows = nx_get_pubdata_record_rows(pub_space_name, pub_data_name, rec_name);
	for row = 0, rows - 1 do
		local text = nx_string(row) .. " | ";
		local value = nx_query_pubdata_record_row_values(pub_space_name, pub_data_name, rec_name, row);
		for i = 1, table.getn(value) do
			text = text .. nx_string(value[i]) .. " | ";
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text));
	end
end

function pubrecclear(gm, ...)
	if table.getn(arg) < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubrecclear <pub_space_name> <pub_data_name> <rec_name>"));
		return;
	end

	local pub_space_name = nx_string(arg[1]);
	local pub_data_name = nx_widestr(arg[2]);
	local rec_name = nx_string(arg[3]);

	if not nx_find_pubdata_record(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. rec_name));
		return;
	end

	if not nx_clear_pub_rec(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Clear record: " .. rec_name .. " fail!") );			return;
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Clear record: " .. rec_name .. " success!") );
end

--添加一行
--recordaddrow 表格名 ...
function pubrecaddrow(gm, ...)
	if table.getn(arg) < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubrecaddrow <pub_space_name> <pub_data_name> <rec_name>"));
		return;
	end

	local pub_space_name = nx_string(arg[1]);
	local pub_data_name = nx_widestr(arg[2]);
	local rec_name = nx_string(arg[3]);

	if not nx_find_pubdata_record(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found pub record " .. recname));
		return;
	end

	local cols = nx_get_pubdata_record_cols(pub_space_name, pub_data_name, rec_name);
	if table.getn(arg) < (3 + cols) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("pub record " .. rec_name .. " columns is " .. cols));
		return;
	end

    local row_value_str = nx_string(arg[4]);
    for col = 1, cols - 1 do
        row_value_str = row_value_str .. ";" .. nx_string(arg[4 + col]);
    end

    nx_add_pub_rec_row_value(pub_space_name, pub_data_name, rec_name, nx_widestr(row_value_str));

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Add row succeed!"));
end

-- 删除一行
function pubrecremoverow(gm, ...)
	if table.getn(arg) < 4 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubrecremoverow <pub_space_name> <pub_data_name> <rec_name> <row>"));
		return;
	end

	local pub_space_name = nx_string(arg[1]);
	local pub_data_name = nx_widestr(arg[2]);
	local rec_name = nx_string(arg[3]);
	local row = nx_int(arg[4]);

	if not nx_find_pubdata_record(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found pub record " .. recname));
		return;
	end

    nx_remove_pub_rec_row(pub_space_name, pub_data_name, rec_name, row);

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Remove row succeed!"));
end

--设置公共数据的表格值
--pubrecset 空间名称 表格名称 行号 列号 值
function pubrecset(gm, ...)
	if table.getn(arg) < 6 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: pubrecset <pub_space_name> <pub_data_name> <rec_name> <row> <col> <value>"));
		return;
	end

    local pub_space_name = nx_string(arg[1]);
	local pub_data_name = nx_widestr(arg[2]);
	local rec_name = nx_string(arg[3]);
	local row = nx_int(arg[4]);
	local col = nx_int(arg[5]);
	local value = arg[6];

    if not nx_find_pubdata_record(pub_space_name, pub_data_name, rec_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. rec_name));
		return;
	end

	local type = nx_get_pubdata_record_coltype(pub_space_name, pub_data_name, rec_name, col);

	if type == VTYPE_INT then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_int(value));
    elseif type == VTYPE_INT64 then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_int64(value));
	elseif type == VTYPE_FLOAT then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_float(value));
	elseif type == VTYPE_STRING then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_string(value));
	elseif type == VTYPE_WIDESTR then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_widestr(value));
	elseif type == VTYPE_OBJECT then
		nx_set_pub_rec_value(pub_space_name, pub_data_name, rec_name, row, col, nx_object(value));
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set record " .. rec_name .. " value succeed!"));
end

function addskill(gm, ...)
	local id = nx_string(arg[1])
	local skill = nx_create_from_config(nx_get_scene(), id)
	if not nx_exists(skill) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("create skill " .. id .. " failed"))
		return
	end

	if not nx_add_skill(gm, skill) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add skill " .. id .. " failed"))
		nx_destroy_self(skill)
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add skill " .. id .. " succeed"))
end

function addallskill(gm, ...)
    if table.getn(arg) ~= 0 and table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: addallskill [<startindex> <endindex>]"));
		return;
	end

    local startindex = 0
    local endindex = -1

    if table.getn(arg) == 2 then
        startindex = nx_int(arg[1])
        endindex = nx_int(arg[2])
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("begin add skill...") )

    local succeed = nx_add_all_skill(gm, startindex, endindex)
    if succeed == 0 then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("addallskill failed"))
	    return
    end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("addallskill succeed, ".. succeed .. " skills add"))
end

function removeallskill(gm, ...)
    if table.getn(arg) ~= 0 and table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: removeallskill [<startindex> <endindex>]"));
		return;
	end

    local startindex = 0
    local endindex = -1

    if table.getn(arg) == 2 then
        startindex = nx_int(arg[1])
        endindex = nx_int(arg[2])
    end

    local succeed = nx_remove_all_skill(gm, startindex, endindex)

    if succeed == 0 then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("removeallskill failed"))
	    return
    end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("removeallskill succeed, ".. succeed .. " skills removed"))
end

function useskill(gm, ...)
	local target = nx_query_property(gm, "LastObject");
	local id = nx_string(arg[1]);

	local skill = nx_find_skill(gm, id);
	if not nx_exists(skill) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("not found skill " .. id));
		return
	end
	nx_use_skill_to_target(gm, skill);
end

--删除技能
function delskill(gm, ...)

	local id = nx_string(arg[1]);

	local skill = nx_find_skill(gm, id);

	if not nx_exists(skill) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("delskill not found skill " .. id));
		return
	end

	nx_remove_skill(gm,skill)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("delskill " .. id .. " success"));
end

function addshoufa(gm, ...)
	local id = nx_string(arg[1])

	if not nx_add_shoufa(gm, id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add shoufa " .. id .. " failed"))
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add shoufa " .. id .. " succeed"))
end

--选择目标放技能
function useskillobj(gm, ...)
	local target = nx_query_property(gm, "LastObject");
	local id = nx_string(arg[1]);
	local skilltype = nx_int(arg[2])

    if not nx_exists(target) then
        return
    end
    local skill = nx_find_skill(target, id);
    if not nx_exists(skill) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("not found skill " .. id));
        return
    end
    nx_use_skill_to_target(target, skill);
end

function useitem(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:useitem index"));
		return;
	end

	local index = nx_int(arg[1]);
	local toolbox = nx_get_child(gm, nx_widestr("ToolBox"));
	local item = nx_get_item(toolbox, index);
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found item at " .. index .. " in toolbox."));
		return;
	end

	if not nx_find_property(item, "SkillID") then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("The item at " .. index .. " in toolbox isn't a tool item."));
		return;
	end

	--nx_use(gm, item)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("use item at " .. index .. " in toolbox succeed!"));
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("use item at " .. index .. " in toolbox failed!"));
end

function addexp(gm, ...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: addexp exp."));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if nx_type(target) ~= TYPE_PLAYER then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Target must be a player."));
		return ;
	end

	local exp = nx_int(arg[1]);
	if nx_addexp(target, exp, 2) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Add exp succeed."));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Add exp failed."));
	end
end

function getconfigprop(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:getconfigprop configid prop."));
		return;
	end
	local configid = nx_int(arg[1]);
	local prop = nx_string(arg[2]);

	local value = nx_get_config_property(configid, prop);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("ConfigID:" .. configid .. " " .. prop .. "=" .. value));
end

--测试NPC凋落物品
--       testdrop value1 value2        value1表示NPC的ConfigID; value2表示NPC的数量
function testdrop(gm, ...)
	local id = nx_string(arg[1]);
	local times = nx_int(arg[2]);
	local item_count = { };
	local bosslevel = 0--nx_get_config_property(id, "BossLevel");
	for nn = 1, times do
		if nx_int(bosslevel) <= 0 then
		--nx_msgbox(nx_string(id))
		    local dropnormalitems = nx_get_normal_dropitems(nx_get_scene(), id);
		    --显示普通掉落物品
		    for i = 1, table.getn(dropnormalitems) do
                local name_id = nx_get_config(dropnormalitems[i]);
                local cur_count = item_count[name_id];
                if cur_count == nil then
                	cur_count = 0;
                end
		    	item_count[name_id] = cur_count + 1;
		    end
		else
			--Bosss掉落属性
			local proplevel = nx_get_config_property(id, "PropLevel");
			local dropbossitems = nx_get_boss_dropitems(nx_get_scene(), id, nx_int(bosslevel), nx_int(proplevel));
			--显示BOSS掉落物品
		    for i = 1, table.getn(dropbossitems) do
                local name_id = nx_get_config(dropbossitems[i]);
                local cur_count = item_count[name_id];
                if cur_count == nil then
                	cur_count = 0;
                end
		    	item_count[name_id] = cur_count + 1;
			end
		end
	end

	local total = 0;
	--显示物品信息
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. times .. "个" .. nx_string(id) .. "的" .. " Items List Begin:" .. "====="));
	for name_id, count in pairs(item_count) do
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(name_id .. "-- amount : " .. count));
		total = total + count;
	end
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("====" .. "total amount : " .. total .. " =====Items List end:" .. "====="));
end

function testdroptime(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:testdroptime dropid [times]"));
		return;
	end

	local tick = nx_get_ticks();

	local dropid = nx_string(arg[1]);--掉落编号
	local times = 0;--测试次数
	if table.getn(arg) > 1 then
		times = nx_int(arg[2]);
	end
	if times <= 0 then
		times = 1;
	end

	local scene = nx_get_scene();--当前场景
	local x,y,z,o = nx_get_position(gm);

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====Begin Test NormalDrop " .. dropid .. " ====="));

	local bosslevel = 0--nx_int(nx_get_config_property(dropid, "BossLevel"));
	local proplevel = nx_int(nx_get_config_property(dropid, "PropLevel"));
	local notionsatisfy = 0;
	local dropsatisfy = {};
	for i = 1, times do
		local items = {};
		if bosslevel == 0 then
			items = nx_get_normal_dropitems(scene, dropid);
		else
			items = nx_get_boss_dropitems(scene, dropid, bosslevel, proplevel);
		end

		local drops = table.getn(items);
		if drops > 0 then
			if dropsatisfy[drops] == nil then
				dropsatisfy[drops] = 0;
			end
			dropsatisfy[drops] = dropsatisfy[drops] + 1;
		else
			notionsatisfy = notionsatisfy + 1;
		end

		for j, item in pairs(items) do
			nx_destroy_self(item);
		end
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("drop 0 times:" .. notionsatisfy));
	for i = 1, table.getn(dropsatisfy) do
		if dropsatisfy[i] == nil then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("drop " .. i .. " times:" .. 0));
		else
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("drop " .. i .. " times:" .. dropsatisfy[i]));
		end
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Ticks:" .. nx_get_ticks() - tick));
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====Test NormalDrop Ended " .. dropid .. " ====="));
end

function findmodule(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:findmodule modulename"));
		return;
	end

	local modulename = nx_string(arg[1]);

	if nx_findmodule(modulename) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Module [" .. modulename .. "] loaded succeed!"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Module [" .. modulename .. "] had not be loaded!"));
	end

end

function test_arrangecontainer(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
	nx_arrange_container(target);
end

function test_repair_item(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
	nx_repair_item(gm, target);
end

function test_repair_all(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
	nx_repair_all(gm, target);
end

function test_move_item(gm, ...)
	local srcviewid = nx_int(arg[1]);
	local srcpos = nx_int(arg[2]);
	local destviewid = nx_int(arg[3]);
	local destpos = nx_int(arg[4]);

	nx_test_move_item(gm, srcviewid, srcpos, destviewid, destpos);
end

-- 创建副本场景
-- clone_scene 场景序列号
function clone_scene(gm, ...)
    if table.getn(arg) < 1 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: clonescene scene_serial"));
        return;
    end

    local serial = nx_int(arg[1]);
    local clone_scene_id = nx_clone_scene(gm, serial);
    if clone_scene_id <= 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clone scene fail!"));
        return;
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(clone_scene_id));
end

-- 获取副本通关奖励物品权值
function get_pass_award(gm,...)
    if table.getn(arg) < 4 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0",nx_widestr("Format: get_pass_award <clone_id> <clone_difficulty> <passLevel> <awardTimes>"));
        return;
    end
    nx_get_pass_award(gm, nx_int(arg[1]),nx_int(arg[2]),nx_int(arg[3]),nx_int(arg[4]))
end

--重置副本次数
function clear_clone_times(gm, ...)
    nx_clear_clone_times(gm, nx_int(arg[1]));
end

--通关副本
function clone_pass(gm, ...)
    nx_clone_pass(gm);
end

-- 关闭当前副本
-- close_clone
function close_clone(gm, ...)
    nx_close_clone();
end

function getclonenum(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format：getclonenum <sceneid>"));
		return;
	end

	local scene = nx_int(arg[1]);
	local num   = nx_get_clone_num(scene);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clonenum=" .. num));
end

function test_curse(gm, ...)
	local ticks = nx_int(arg[1])
	nx_custom(gm, SERVER_CUSTOMMSG_BEGIN_CURSE, ticks)
end

--设置副本进入次数
function set_clone_enter_count(gm, ...)
	if table.getn(arg) < 3 then
		return
	end

	local protoscene = nx_int(arg[1])
	local difficuty = nx_int(arg[2])
	local count = nx_int(arg[3])

	nx_set_clone_enter_count(gm, protoscene, difficuty, count)
end

--设置副本评分
function set_clone_pass_score(gm, ...)
	if table.getn(arg) < 3 then
		return
	end

	local protoscene = nx_int(arg[1])
	local difficuty = nx_int(arg[2])
	local score = nx_int(arg[3])

	nx_set_clone_pass_score(gm, protoscene, difficuty, score)
end

function set_reward(gm, ...)
	if table.getn(arg) < 1 then
		return
	end

	local count = nx_int(arg[1])

	nx_set_reward_rand_count(gm, count)
end

-- 转职
function setjob(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	nx_trans_job(target)

	--if table.getn(arg) == 0 then
	--	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: job_id"))
	--	return
	--end

	--local job_id = nx_int(arg[1])

	-- 强制接
	--if nx_trans_job(gm, job_id) then
	--	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setjob " .. nx_string(job_id) .. "succeed."))
	--else
	--	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setjob " .. nx_string(job_id) .. "failed."))
	--end
end

-- 强制接受某任务 accepttask [task_id]
function accepttask(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: accepttask <taskid>"))
		return
	end

	local taskid = nx_int(arg[1])

	-- 强制接
	if nx_accept_task(gm, taskid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Accept task " .. nx_string(taskid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Accept task " .. nx_string(taskid) .. "failed."))
	end
end

function settasksumbit(gm, ...)
	if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: sumbittask <taskid>"))
		return
	end

	local taskid = nx_int(arg[1])

	if nx_set_task_sumbit(gm, taskid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Sumbit task " .. nx_string(taskid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Sumbit task " .. nx_string(taskid) .. "failed."))
	end
end

function addforce(gm, ...)

	local force = nx_int(arg[1])

	if nx_add_force(gm, force) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add force" .. nx_string(force) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add force " .. nx_string(force) .. "failed."))
	end
end

function addrepute(gm, ...)

	local force = nx_int(arg[1])
	local repute = nx_int(arg[2])

	if nx_add_repute(gm, force, repute) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add repute " .. nx_string(force) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add repute " .. nx_string(force) .. "failed."))
	end
end

function settaskstep(gm, ...)
    if table.getn(arg) == 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: accepttask <taskid>"))
		return
	end

	if table.getn(arg) == 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: accepttask <subfunction>"))
		return
	end

	local taskid = nx_int(arg[1])
	local sub_type = nx_int(arg[2])
	local sub_index = nx_int(arg[3])
	local num = nx_int(arg[4])

	if nx_set_task_step(gm, taskid, sub_type, sub_index, num) then
	    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set task " .. nx_string(taskid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set task " .. nx_string(taskid) .. "failed."))
	end
end

function linkto(gm)
	local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	nx_link_to_obj(gm, target)
end

function unlink(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
	--nx_player_unlink(gm)
	nx_fly_unlink(gm)
end

function activeairport(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: activeairport <airport_id>"));
		return;
	end

	local airport_id = nx_int(arg[1])

	nx_active_airport(gm, airport_id)
end

function reload_trans_path(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_trans_path "  .. " begin."))
	nx_reload_trans_path(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_trans_path "  .. " end."))
end

function switch_locate_npc(gm,...)
    local target = nx_query_property(gm, "LastObject");
	if not nx_exists(target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("LastObject not exists!"));
		return;
	end

	local scene = nx_int(arg[1])
	local x = nx_float(arg[2])
	local y = nx_float(arg[3])
	local z = nx_float(arg[4])
	local orient = nx_float(arg[5])

	nx_switch_locate_npc(target, scene,x, y, z, orient);
end

function inc_job_level(gm,...)
    if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: inc_job_level <jobid> <value>"));
		return;
	end

	local jobid  = nx_string(arg[1]);
	local value  = nx_float(arg[2]);

	nx_inc_job_level(gm, jobid, value)
end

--创建一个游戏对象
--cobj 对象ID
function cAdventureNpc(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:cobj <ID>"));--格式：cobj 编号
		return;
	end

	local x,y,z,o = nx_get_position(gm);
	local id = nx_string(arg[1]);

	local item = nx_create_adventurenpc(id,x, y, z, o);
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));

	end
end

function SetOfflinePrizeTime(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:cobj <ID>"));--格式：cobj 编号
		return;
	end

	local id = nx_string(arg[1]);
	local times = nx_int(arg[2]);


	nx_set_prize_times(gm, id, times)

end


-- 修改声望
-- setrepute repute_id repute_value
function changerepute(gm, ...)

	-- 检查参数数量
	if table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:changerepute <name> <value>"))
		return
	end

	-- 得到参数
	local name = nx_string(arg[1])
	local value = nx_int(arg[2])

	-- 判断是否是声望名
	if not nx_is_repute(name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not a repute name!"))
		return
	end

	-- 执行规则
	nx_modify_repute(gm, name, value)

end


-- 书写剧本
-- writedrama drama_id
function writedrama(gm, ...)

	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:writedrama <drama_name>"))
		return
	end

	-- 得到参数
	local drama_id = nx_string(arg[1])

	-- 申请写剧本
	nx_require_write_drama(gm, drama_id)

end

function reload_config(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:reload_config <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	if nx_reload_config(file_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " failed."))
	end
end

function reload_skill_res(gm, ...)
    if nx_reload_skill_res(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload skill resource succeed."))
    else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload skill resource failed."))
    end
end

function set_skill_radius(gm, ...)
    if nx_set_radius_add(nx_int(arg[1])) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set skill radius_add succeed."))
    else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set skill radius_add   failed."))
    end
end

function reload_buffer_res(gm, ...)
    if nx_reload_buffer_res(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload buffer resource succeed."))
    else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload buffer resource failed."))
    end
end

--给事件拥有者添加一个buff
function add_buff(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: add_buff buffid"));--格式：set 属性名称 属性值
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
	local buff_id = nx_string(arg[1]);
	local bufferobj = nx_create_from_config(nx_get_scene(), nx_string(buff_id))
	if not nx_exists(bufferobj) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add buffer " .. buff_id .. " failed."))
		return
	end
	if not nx_add_buffer(target, gm, bufferobj) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add buffer " .. buff_id .. " failed."))
		nx_destroy_self(bufferobj)
		return
	end
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add buffer " .. buff_id .. " succeed."))
end

--移除一指定的BUFF
function remove_buff(gm, ...)

	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: remove_buff buffid"))
		return
	end

	local target = get_prop_target(gm)

	if not nx_exists(target) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove buffer " .. buff_id .. " failed."))

		return
	end

	local buff_id = nx_string(arg[1])

	if nx_remove_buffer(target, gm, buff_id) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove buffer " .. buff_id .. " succeed."))
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove buffer " .. buff_id .. " failed."))
    end

end

--重新加载副本进入规则
function reload_clone_config(gm, ...)
	nx_reload_clone_config()
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_clone_config "  .. " ok."))
end

--重载NPC, 重载NPC，需要新创建的NPC才能起效果
function reload_normal_npc_config(gm, ...)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_npc_config "  .. " begin."))

	nx_reload_static_data(STATIC_DATA_NPC)
	nx_reload_npcconfig()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_npc_config "  .. " ok."))
end

--重载装备和物品配置, 玩家物品需重新登陆才能起效
function reload_item_config(gm, ...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_item_config "  .. " begin."))
--	nx_reload_static_data(STATIC_DATA_EQUIP_ACTIVATE)
--	nx_reload_static_data(STATIC_DATA_ITEM_ART)
--	nx_reload_static_data(STATIC_DATA_ITEM_LOGIC)
--	nx_reload_static_data(STATIC_DATA_ITEM_FUNC)
--	nx_reload_static_data(STATIC_DATA_ITEM_COPY)		-- 副本道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_INGREDIENT)	-- 配方道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_TASK)		-- 任务道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_USESKILL)	-- 使用技能道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_LEARNSKILL)	-- 学习技能道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_SEED)		-- 种子道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_MATERIAL)	-- 生活材料道具专用包
--	nx_reload_static_data(STATIC_DATA_ITEM_PROPMODIFY)
    nx_reload_item_xml()
--	nx_reload_config("ini\\item\\Equipment.xml")
--	nx_reload_config("ini\\item\\toolitem.xml")
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_item_config "  .. " ok."))
end

--重载BUFFER配置,  新BUFFER起效
function reload_buffer_config(gm, ...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_buffer_config "  .. " begin."))
	nx_reload_static_data(STATIC_DATA_BUFF)
	nx_reload_static_data(STATIC_DATA_FORMULA)
	--nx_reload_config("ini\\skill\\buffer_new.xml")
	nx_reload_buffer_resource()
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_buffer_config "  .. " ok."))
end

--重载技能配置，需重新登陆
function reload_skill_config(gm, ...)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_skill_config "  .. " begin."))

	nx_reload_static_data(STATIC_DATA_SKILL_STATIC)
	nx_reload_static_data(STATIC_DATA_SKILL_LOCK_DATA)
	nx_reload_static_data(STATIC_DATA_SKILL_FLOW)
	nx_reload_static_data(STATIC_DATA_SKILL_CONSUME)
	nx_reload_static_data(STATIC_DATA_SKILL_TARGETSHAPE)
	nx_reload_static_data(STATIC_DATA_SKILL_HITSHAPE)
	nx_reload_static_data(STATIC_DATA_SKILL_TARGET_SELECT)
	nx_reload_static_data(STATIC_DATA_FORMULA)
	nx_load_skill_resource()
	nx_reload_skill_uselimit_data()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_skill_config "  .. " ok."))
end

--重新加载路点配置
function reload_transport_roadpoint(gm, ...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_transport_roadpoint "  .. " begin."))

	nx_reload_static_data(STATIC_DATA_TRANS_PATH_PROP)
	nx_reload_transport_roadpoint()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_transport_roadpoint "  .. " end."))

end

--删除玩家身上的零时变量
function del_owner_data(gm, ...)
	local dataname = nx_string(arg[1])
	if  nx_find_data(gm, dataname) then
		nx_remove_data(gm, dataname)
	end
end

--显示当前场景离线打工状态
function offlinejob(gm, ...)
	local scene = nx_get_scene_id()

	local table_name = nx_get_scene_players(nx_int(scene))

	local totalnum = table.getn(table_name)
	local offnum = 0
	local workstate = ""

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("========================offlinejob========================"));
	for i = 1, totalnum do
		local name = table_name[i]
		local obj = nx_find_player(nx_widestr(name))
		if nx_exists(obj) then
			local type = nx_query_property(obj, "LogicState")
			if nx_int(type) == nx_int(106) then
				offnum = offnum + 1
				local id = nx_query_property(obj, "OfflineJobId")
				local path = nx_query_data(obj, "CurOffLineJobPath")
				local account = nx_query_property(obj, "Account")

				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(offnum).."|"..nx_string(name).."|"..nx_string(id).."|"..nx_string(path).."|"..nx_string(account).."|"))
			end
		end
	end
	local text = nx_string("scene:".. nx_string(scene).." total:" .. nx_string(totalnum).." offlinejob:"..nx_string(offnum))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text))
end

--新增离线行为
function addofflinejob(gm, ...)
	if table.getn(arg) == 1 then

		--清空表格
		nx_clear_record(gm, "OffLineJob")
		nx_clear_record(gm, "OffLineJob_Log")

		--添加行为
		nx_get_offline_job(gm, nx_string(arg[1]))

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Add OffLineJob succeed"))
	end
end

--初始化路径
function initoffLinepath(gm, ...)

	if table.getn(arg) == 1 then

		--初始化路径
		if not nx_find_data(gm, "PATH_ID") then
			nx_add_data(gm, "PATH_ID", VTYPE_STRING)
		end
		nx_set_data(gm, "PATH_ID", nx_string(arg[1]))

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Init OffLineJob Path succeed"))
	end
end

-- 重新载入离线资源
function reload_offline_resource(gm,...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_offline_resource "  .. " begin."))

	if table.getn(arg) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: reload_offline_resource"));
		return;
	end

	nx_reload_offline_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_offline_resource "  .. " end."))
end

--设置任务标记为完成
function finishtask(gm, ...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: finishtask task_id"));
		return;
	end

  local taskId = nx_int(arg[1])
	nx_set_task_complete(gm, taskId)
end

--设置任务标记为完成
function resettask(gm, ...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: finishtask task_id"));
		return;
	end

  local taskId = nx_int(arg[1])
	nx_reset_task(gm, taskId)
end

function test_music(gm, ...)
	local flag = nx_int(arg[1])
	local music = nx_string(arg[2])
	nx_custom(gm, SERVER_CUSTOMMSG_PLAY_MUSIC, flag,music)
end

function get_player_count(gm, ...)
	local num = nx_get_player_count()
	local text2 = "Total Players : " .. nx_string(num)
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text2));
	local scene = nx_get_scene()
	--local childlist = nx_get_child_list(scene, 2)
	local scene = nx_get_scene_id()
	local table_name = nx_get_scene_players(nx_int(scene))
	local text = "Total Players : " .. nx_string(table.getn(table_name))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text));
	for i=1, table.getn(table_name) do
		local name = table_name[i]
		local obj = nx_find_player(nx_widestr(name))

		local x,y,z,o = nx_get_position(obj);
		text1 = "p="..nx_string(obj).."/".."x="..nx_string(x).."/".."y="..nx_string(y).."/".."z="..nx_string(z)
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text1));
	end
end

--改变当前副本进度为
function set_clone_progress(gm,...)
	if table.getn(arg) < 1 then
		return
	end
	local value = nx_int(arg[1])
	local scene = nx_get_scene()
	if nx_find_property(scene, "Progress") then
		nx_set_property(scene, "Progress", value)
	end
end

--清除表
function clear_record(gm,...)
	if table.getn(arg) < 1 then
		return
	end
	local name = nx_string(arg[1])
	nx_clear_record(gm, name)
end

--播放电影
function play_movie(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: play_movie <movie_id>"))
		return
	end

	local movie_id = nx_int(arg[1])

	nx_beigin_movie(gm, gm, movie_id)
end



--[[
别参数
//1 装备
//2 招式
//3 内功
//4 心法
//5 轻功
//6 技能
//7 道具
//8 任务
//9 活动
//14经脉
//15穴位
]]

function remove_obj(gm, ...)

	local obj = nx_query_data(gm, "GMPropTarget");

	if nx_exists(obj) then

		if nx_find_property(obj, "QTypeID") then

			local qtypeid = nx_query_property(obj, "QTypeID")

			if qtypeid == 2 then

				nx_destroy_self(obj)
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove_obj success"));

			end

			return
		end
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove_obj failed"));
end

function reload_award_resource(gm,...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_award_resource "  .. " begin."))

	if table.getn(arg) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: reload_award_resource"));
		return;
	end

	nx_reload_award_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_award_resource "  .. " end."))
end
--
-- 重新载入任务资源
function reload_task_resource(gm,...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_task_resource "  .. " begin."))

	if table.getn(arg) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: reload_task_resource"));
		return;
	end

	nx_reload_task_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_task_resource "  .. " end."))
end

-- 重新载入筛选资源
function reload_option_resource(gm,...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_option_resource "  .. " begin."))

	if table.getn(arg) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: reload_option_resource"));
		return;
	end

	nx_reload_option_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_option_resource "  .. " end."))
end

--放弃任务
function giveuptask(gm,...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: giveuptask taskid"));
		return;
	end
	local task_id = nx_int(arg[1])

	nx_giveup_task(gm, task_id);
end

--参数说明
-- ? 或者 h 表示帮助,列出所有参数说明
-- g 或者 表示放弃任务 任务编号
-- a 或者 表示接受任务 任务编号
-- p 或者 表示显示任务表 1 任务承接表 2 任务进度表 3 任务完成表
-- s
-- c 或者 表示清空任务表
local g_task_rec =
{
	"Task_Accepted",
	"Task_Record",
	"Task_Completed",
	"TaskRepeatRec",
	"Task_Hide_Rec",
	"CmpltCloneTaskRec",
}
function task(gm, ...)
	local cmd = "";
	if table.getn(arg) < 1 then cmd = "?"; else cmd = nx_string(arg[1]) end

	if string.find(cmd, "?") ~= nil or string.find(cmd, "h") then
		-- 打印帮助
		task_help(gm, TIPSTYPE_GMINFO);
		return
	elseif string.find(cmd, "g") then
		if table.getn(arg) < 2 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("警告: 正在删除所有任务......"));
			local rows = nx_get_record_rows(gm, "Task_Accepted");
			for i = 0, rows-1 do
				local id = nx_query_record_value(gm, "Task_Accepted", i, 0);
				nx_giveup_task(gm, id);
			end
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功放弃所有任务"));
		else
			local id = nx_int(arg[2])
			nx_giveup_task(gm, id)
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功放弃任务[......"..nx_string(id).."]"));
		end
	elseif string.find(cmd, "a") then
		if table.getn(arg) < 2 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("参数太少,详见帮助...."));
			task_help(gm, TIPSTYPE_GMINFO);
			return
		end
		local id = nx_int(arg[2]);
		nx_accept_task(gm, id)
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功承接任务[......"..nx_string(id).."]"));
	elseif string.find(cmd, "p") then
		if table.getn(arg) < 2 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("参数太少,详见帮助...."));
			task_help(gm, TIPSTYPE_GMINFO);
			return
		end
		local id = nx_int(arg[2]);
		local rec = g_task_rec[id]
		if rec ~= nil then
			if not nx_find_record(gm, rec) then
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. rec));
				return;
			end
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("================" .. nx_string(nx_query_property(gm, "Name")) .. ":" .. rec .. "================"));
			local cols = nx_get_record_cols(gm, rec);
			local rows = nx_get_record_rows(gm, rec);
			for row = 0, rows - 1 do
				local text = nx_string(row) .. "|";
				for col = 0, cols - 1 do
					text = text .. nx_string(nx_query_record_value(gm, rec, row, col)) .. "|";
				end
				nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(text));
			end
		else
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("表格id编号错误,详见帮助...."));
			task_help(gm, TIPSTYPE_GMINFO);
			return
		end
	elseif string.find(cmd, "s") then
		if table.getn(arg) < 5 then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("参数太少,详见帮助...."));
			task_help(gm, TIPSTYPE_GMINFO);
			return
		end
		local id = nx_int(arg[2]);
		local rec = g_task_rec[id]
		local row = nx_int(arg[3])
		local col = nx_int(arg[4]);
		local value = arg[5]

		if not nx_find_record(gm, rec) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Not found record: " .. rec));
			return;
		end
		if row >= nx_get_record_rows(gm, rec) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("错误 行号超出表格最大行: " .. rec));
			return
		end
		if col >= nx_get_record_cols(gm, rec) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("错误 列号超出表格最大列: " .. rec));
			return
		end

		local type = nx_get_record_coltype(gm, rec, col);
		if type == VTYPE_INT then
			nx_set_record_value(gm, rec, row, col, nx_int(value));
		elseif type == VTYPE_INT64 then
			nx_set_record_value(gm, rec, row, col, nx_int64(value));
		elseif type == VTYPE_FLOAT then
			nx_set_record_value(gm, rec, row, col, nx_float(value));
		elseif type == VTYPE_DOUBLE then
			nx_set_record_value(gm, rec, row, col, nx_double(value));
		elseif type == VTYPE_STRING then
			nx_set_record_value(gm, rec, row, col, nx_string(value));
		elseif type == VTYPE_WIDESTR then
			nx_set_record_value(gm, rec, row, col, nx_widestr(value));
		elseif type == VTYPE_OBJECT then
			nx_set_record_value(gm, rec, row, col, nx_object(value));
		end

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("设置表格成功: " .. rec));
	elseif string.find(cmd, "c") then
		-- 清空进度表
		local bret = nx_clear_record(gm, "Task_Record");
		if bret then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功清空任务进度表"));
		end

		bret = nx_clear_record(gm, "Task_Accepted");
		if bret then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功清空任务接受表"));
		end

		-- 清空任务记录表
		bret = nx_clear_record(gm, "Task_Completed");
		if bret then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("成功清空任务完成表"));
		end
	end
end

--清空所有任务表格
function clearalltask(gm,...)
	local bClearCompleted = false;
	if table.getn(arg) > 0 then
		local flag = nx_string(arg[1]);
		if flag == nx_string("-help") then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clearalltask 命令后面可以附加一个参数: "));
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("参数为[-help], 则打印本帮助信息"));
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("参数为[-record], 则删除所有任务的同时删除任务记录"));
			return
		elseif flag == nx_string("-record") then
			bClearCompleted = true;
		end
	end

	-- 先放弃所有任务
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Warning: trying to giveup all task......"));
	local rows = nx_get_record_rows(gm, "Task_Accepted");
	for i = 0, rows-1 do
		local id = nx_query_record_value(gm, "Task_Accepted", i, 0);
		nx_giveup_task(gm, id);
	end

	-- 清空进度表
	--local bret = nx_clear_record(gm, "Task_Record");
	--if bret then
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear task_record succeed"));
	--end

	--bret = nx_clear_record(gm, "Task_Accepted");
	--if bret then
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear task_accepted succeed"));
	--end

	-- 清空任务记录表
	if bClearCompleted then
		local bret = nx_clear_record(gm, "Task_Completed");
		if bret then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear task_completed succeed"));
		end
	end
end


--重新载入生活技能相关ini
function reload_lifejob_config(gm,...)

	local res = nx_reload_lifejob_config()

	if res then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_lifejob_config OK"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_lifejob_config faild"))
	end

end

--重新载入静态属性资源文件
function reload_static_data(gm,...)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_static_data "  .. "begin ..."))

	nx_reload_all_static_data()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_static_data "  .. "ok."))
end

--重载交通系统相关配置,如果路点信息改变则要重启客户端
function reload_trans_config(gm, ...)
	nx_reload_config("ini\\Npc\\TransObj\\TransPathRec.xml")
	nx_reload_trans_func_id();
	nx_reload_transport_roadpoint();
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_trans_config "  .. " ok."))
end

--重新载入技能事件配置相关ini
function reload_skill_event(gm,...)

	local res = nx_reload_skill_event()

	if res then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_skill_event OK"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_skill_event faild"))
	end

end

--查询当前属性操作的对象的所有保存属性
function property_save(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local total = 0
	local proplist = nx_get_property_list(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Save Property List Begin:" .. "====="));
	for i = 1, table.getn(proplist) do
		if nx_get_attr_saving(target, proplist[i]) then
			--[{@0:属性}]={@1:数值}
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(proplist[i] .. "=" .. nx_string(nx_query_property(target, proplist[i]))));
			total = total + 1
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Save Property List End .total:" ..nx_string(total).. "======"));
end

--查询当前属性操作的对象的所有可视属性
function property_visible(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local total = 0
	local proplist = nx_get_property_list(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Visible Property List Begin:" .. "====="));
	for i = 1, table.getn(proplist) do

		if nx_get_attr_visible(target, proplist[i]) then
			--[{@0:属性}]={@1:数值}
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(proplist[i] .. "=" .. nx_string(nx_query_property(target, proplist[i]))));
			total = total + 1
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Visible Property List End.total:" ..nx_string(total).. "======"));
end

--查询当前属性操作的对象的所有保存表格名
function record_save(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local total = 0
	local reclist = nx_get_record_list(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Save Record List Begin:" .. "====="));
	for i = 1, table.getn(reclist) do
		if nx_get_record_saving(target, reclist[i]) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(reclist[i]));
			total = total + 1
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Save Record List End.total:" ..nx_string(total).. "======"));
end

--查询当前属性操作的对象的所有可视表格名
function record_visible(gm)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local total = 0
	local reclist = nx_get_record_list(target);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. nx_string(nx_query_property(target, "Name")) .. " Visible Record List Begin:" .. "====="));
	for i = 1, table.getn(reclist) do
		if nx_get_record_visible(target, reclist[i]) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(reclist[i]));
			total = total + 1
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("======" .. nx_string(nx_query_property(target, "Name")) .. " Visible Record List End.total:" ..nx_string(total).. "======"));
end

--传送到某个分组子场景
function switch_to_sub_secne(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:switch_to_sub_secne <subsceneid>"));--格式：silence 玩家名 分钟
		return;
	end
	local subsceneid = arg[1];
	nx_switch_to_subscene(gm,nx_int(subsceneid))
end

-- 查询某ID的NPC具体位置信息
function gps(gm,...)
	if table.getn(arg) < 1 then

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: gps <ConfigID>"));

	end

	local npcid = nx_string(arg[1])

	local scene = nx_get_scene()

	local npclist = nx_find_child_list_byconfig(scene,npcid,TYPE_NPC)

	local num = nx_int(table.getn(npclist))

	if num ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Searching for "..nx_string(npcid).." as follows:"))
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Num:"..nx_string(num)))
	else nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("After automatic searching,we make the conclusion:Object doesn't exist"))
	end

	for i = 1,num do
		local npc = nx_object(npclist[i])

		local name = nx_query_property(npc,"Name")

		local x,y,z,o = nx_get_position(npc)

		local info = nx_string("Name:"..nx_string(name).." ".. "Location:"..nx_string(x).." ".. nx_string(y).." ".. nx_string(z).." ".. nx_string(o))

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(info))

	end

end

--创建宝箱
function get_box(gm, ...)

	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:get_box <ID> <ownername>"));--格式：get_box 编号 拥有者
		return;
	end

	local x,y,z,o = nx_get_position(gm);
	local boxid = nx_string(arg[1]);
	local ownername = nx_widestr(arg[2]);


	local item = nx_create_box(boxid, ownername, x, y, z, o);
	if not nx_exists(item) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .. " failed!"));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Create object " .. id .." succeed!"));
	end
end

function fight_test(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:fight_test <mount> "));--格式：get_box 编号 拥有者
		return;
	end
	local num = nx_int(arg[1])
	local x,y,z,o = nx_get_position(gm);
	for i = 0, num do
		local npc1 = nx_create_obj("monv030500", 0, nx_number(x)+2, y,  nx_number(z)+2, o);
		nx_add_record_row_values(npc1, "EnemyNpcRec", "monv030600")
		nx_add_record_row_values(npc1, "EnemyNpcRec", "monrm054200")
		nx_add_record_row_values(npc1, "EnemyNpcRec", "monrm054100")
		local npc2 = nx_create_obj("monv030600", 0, nx_number(x)+2, y,  nx_number(z)-2, o);
		nx_add_record_row_values(npc2, "EnemyNpcRec", "monv030500")
		nx_add_record_row_values(npc2, "EnemyNpcRec", "monrm054200")
		nx_add_record_row_values(npc2, "EnemyNpcRec", "monrm054100")
		local npc3 = nx_create_obj("monrm054200", 0, nx_number(x)-2, y,  nx_number(z)+2, o);
		nx_add_record_row_values(npc3, "EnemyNpcRec", "monv030500")
		nx_add_record_row_values(npc3, "EnemyNpcRec", "monv030600")
		nx_add_record_row_values(npc3, "EnemyNpcRec", "monrm054100")
		local npc4 = nx_create_obj("monrm054100", 0, nx_number(x)-2, y,  nx_number(z)-2, o);
		nx_add_record_row_values(npc4, "EnemyNpcRec", "monv030500")
		nx_add_record_row_values(npc4, "EnemyNpcRec", "monv030600")
		nx_add_record_row_values(npc4, "EnemyNpcRec", "monrm054200")
	end
end

--给场景添加临时数据
function add_scenedata(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:add_scenedata <ID> "))--格式：add_scenedata 数据ID
		return 0
	end
	local dataname = nx_string(arg[1])
	local scene = nx_get_scene()
	if nx_find_data(scene,dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." already exists"))
	else
		if nx_add_data(scene,dataname,VTYPE_INT) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." successfully added"))
		end
	end
end

--删除场景临时数据
function del_scenedata(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:del_scenedata <ID> "));--格式：del_scenedata 数据ID
	end
	local dataname = nx_string(arg[1])
	local scene = nx_get_scene()
	if not nx_find_data(scene,dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." doesn't exist"))
	else
		if nx_remove_data(scene,dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." successfully removed"))
		end
	end
end

--删除GM身上临时数据
function del_playerdata(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:del_playerdata <ID> "));--格式：del_playerdata 数据ID
	end
	local dataname = nx_string(arg[1])
	local scene = nx_get_scene()
	if not nx_find_data(gm,dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." doesn't exist"))
	else
		if nx_remove_data(gm,dataname) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(dataname).." successfully removed"))
		end
	end
end


function get_npc_count(gm)
	local scene = nx_get_scene()
	local childlist = nx_get_child_list(scene, 4)
	local num = table.getn(childlist)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Npc: " .. nx_string(num)));
end

function init_player_data(gm)
	local list = nx_get_record_list(gm)
	for i = 1, table.getn(list) do
		local recname = list[i]
		if nx_find_record(gm, recname) then
			if nx_get_record_saving(gm, recname) then
				local cols = nx_get_record_cols(gm, recname)
				local maxrow = nx_get_record_max_rows(gm,recname)
				local rows = nx_get_record_rows(gm,recname)
				for j = 1,maxrow-rows do
					local temprow = nx_add_record_row(gm,recname)
					for k = 0, cols-1 do
						local coltype = nx_get_record_coltype(gm, recname, k)
						if coltype == VTYPE_STRING then
							nx_set_record_value(gm,recname,temprow, k, "我顶" )
						elseif coltype == VTYPE_WIDESTR then
							nx_set_record_value(gm,recname,temprow, k, nx_widestr("我顶") )
						end
					end
				end
			end
		end
	end

end

function init_player_data1(gm)
	--技能
	local skillbox = nx_get_child(gm, nx_widestr("SkillContainer"))
	for i = 5, 200 do
		nx_create_from_config(skillbox,"zs_default_01");
	end
end

function AddPosionToWeapon(gm, ...)

	if table.getn(arg) < 4 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:AddPosionToWeapon <BuffID> <times> <ratio> <type>"));--AddPosionToWeapon buffID 次数 几率（0-100） 类型（0 武器 1暗器）
		return
	end
	local buffid = nx_string(arg[1])
	local time   = nx_int(arg[2])
	local ratio  = nx_int(arg[3])
	local ntype  = nx_int(arg[4])


	local equipbox = nx_get_child(gm, nx_widestr("EquipBox"))

	if not nx_exists(equipbox) then
		return;
	end
	if ntype == 0 then
		weapon = nx_get_item(equipbox,22)
	else
		weapon = nx_get_item(equipbox,23)
	end

	if not nx_exists(weapon) then
		return;
	end

	if nx_get_record_rows(weapon, "PoisonRec") >= nx_get_record_max_rows(weapon, "PoisonRec") then
		nx_remove_record_row(weapon, "PoisonRec", 0)
	end
	nx_add_record_row_values(weapon, "PoisonRec", buffid, time, ratio)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Posion"))
end

function fishing(gm)
	nx_begin_fishing(gm)
end

function studyjob(gm, ...)
	local jobid = nx_string(arg[1])
	nx_study_job(gm,jobid)
end


-- Unit test entry
-- flying add
function runtest( gm, ... )
	nx_run_test( gm, nx_query_property(gm, "LastObject") );
end

-- 清空背包
function cleartoolbox(gm)
	nx_clear_container(gm)
end

--重新加载道具整理排序规则文件: item_arrange_order.xml
function reload_item_arrange(gm, ...)
	if nx_reload_item_arrange_xml(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("item_arrange_order.xml reload succeed "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("item_arrange_order.xml reload failed "))
	end
end


--加载钓鱼数据
function reload_fishing_data(gm)
	nx_reload_fishing_config(gm)
end

--添加空中技能点
function add_airskillpoint( gm, ... )

	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: add_airskillpoint <value>"));
		return;
	end

	local value   = nx_int(arg[1])

	if nx_add_airskillpoint(gm, value) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add_airskillpoint ok"))
	end

end

--开始一个小游戏
function start_game(gm,...)
	local game = nx_string(arg[1])
	local level = nx_int(arg[2])
	nx_start_game(gm, game, level)
end

function set_friend_value(gm, ...)
	local friendname = nx_widestr(arg[1])
	local friend_value = nx_int(arg[2])

	if friend_value < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "friend value must be more than 0")
	else
		local target = get_prop_target(gm)
		if not nx_exists(target) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "please select valid object!")
			return
		end
		nx_set_friend_value(target, friendname, friend_value)
	end
end

function plant(gm,...)
	local farmnpc = nx_query_property(gm, "LastObject")
	local seedname = nx_string(arg[1])

end

function effecttest(gm,...)
	local obj = nx_query_property(gm, "LastObject")
	local name = nx_string(arg[1])
	nx_custom_effect(obj, name)

end

--增加称号
function addtitle(gm, ...)
	local id = nx_int(arg[1])
	if not nx_add_playertitle(gm, id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add addtitle " .. id .. " failed"))
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add addtitle " .. id .. " succeed"))
end

--删除称号
function removetitle(gm, ...)
	local id = nx_int(arg[1])
	if not nx_remove_playertitle(gm, id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove title " .. id .. " failed"))
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("remove title " .. id .. " succeed"))
end

function neigpkmem(gm,...)
	local str = nx_get_neigpk_data();
	local tb = nx_split_string(str, "|");
	local len = table.getn(tb);
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "====================内功比拼内存数据:=======================")
	for i = 1, len do
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", tb[i])
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "====================/内功比拼内存数据=======================")
end

function publish_news(gm, ...)
	local argnum = table.getn(arg)
	if argnum < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "发布新闻需要一个字符串参数")
		return
	end

	local info = nx_string(arg[1])
	local num = table.getn(arg);
	local ty = 1
	local count = 1;
	if num > 1 then ty = nx_int(arg[2]) end
	if num > 2 then count = nx_int(arg[3]) end

	if nx_publish_news(ty, count, info) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "发布新闻成功")
	end
end

function create_trans(gm,...)
	local obj = nx_query_data(gm, "GMPropTarget");
	local transpathid = nx_number(arg[1])
	if nx_exists(obj) then
		nx_create_transtool(obj,transpathid)
	end
end


function destroy_trans(gm,...)
	local obj = nx_query_data(gm, "GMPropTarget");
	if nx_exists(obj) then
		nx_destroy_transtool(obj)
	end
end


function create_world_trans(gm,...)
	local obj = nx_query_data(gm, "GMPropTarget");
	local sz_to = nx_string(arg[1])
	if nx_exists(obj) then
		nx_create_world_transtool(obj,sz_to)
	end
end

-- 清除关卡里所有npc生成器
function clear_all_npc(gm,...)
	nx_clear_allnpc()
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "清除关卡里所有npc成功")
end

--设置攻击者
function SetAttacker(gm,...)
	if not nx_find_data(gm, "GMAttacker") then
		nx_add_data(gm, "GMAttacker", VTYPE_OBJECT);
	end

	nx_set_data(gm, "GMAttacker", gm);

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set Attacker succeed"));
end

--设置被攻击者
function SetAttackTarget(gm,...)
	if not nx_find_data(gm, "GMAttackTarget") then
		nx_add_data(gm, "GMAttackTarget", VTYPE_OBJECT);
	end

	nx_set_data(gm, "GMAttackTarget", nx_query_property(gm, "LastObject"));

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Set AttackTarget succeed"));

end


--暴击
function  Fight_Effect(gm,...)

	local para0 = nx_string(arg[1]);

	local hp = nx_int(arg[2]);


	local attacker  = nx_query_data(gm, "GMAttacker");

	local attacktarget = nx_query_data(gm,"GMAttackTarget");

	if para0 == "VAHP" then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("VAHP"));
		nx_send_fight_vahp(attacker,attacktarget,hp);
	elseif para0 == "FAILING" then
		nx_send_fight_failing(attacker,attacktarget,hp);
	elseif para0 == "SPITE" then
		nx_send_fight_spite(attacker,attacktarget,hp);
	elseif para0 == "MISS" then
		nx_send_fight_miss(attacker,attacktarget);
	elseif para0 == "PARRY" then
		nx_send_fight_parry(attacker,attacktarget);
	elseif para0 == "DODGE" then
		nx_send_fight_dodge(attacker,attacktarget);
	end


end

--变身操作
function  chagemodel_role(gm,...)
    local bchange = nx_int(arg[1]);

	local path = nx_string(arg[2]);
	nx_changemodel_role(gm,bchange,path);
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("chagemodel_role succeed"))
end

--召唤宠物操作
function  showpet(gm,...)
	local name = nx_string(arg[1]);
	if nx_show_pet(gm,name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("showpet succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("showpet failed"))
	end
end

--回收宠物操作
function  destroypet(gm,...)
	local para0 = nx_string(arg[1]);
	if para0 == "select" then
		if not nx_find_data(gm, "GMPropTarget") then
			nx_add_data(gm, "GMPropTarget", VTYPE_OBJECT);
		end
		nx_set_data(gm, "GMPropTarget", nx_query_property(gm, "LastObject"));
		local target = get_prop_target(gm)
		if not nx_exists(target) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("destroypet failed"))
			return
		end
		if nx_destroy_pet(gm,target) then
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("destroypet succeed"))
			return
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("destroypet failed"))
end

--检查技能
function  checkSkill(gm,...)
	local name = nx_string(arg[1]);
	if nx_show_pet(gm,name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("showpet succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("showpet failed"))
	end
end

-- 创建一个战斗载具
function  showbc(gm,...)
	local name = nx_string(arg[1]);
	if nx_show_battlecarrier(gm,name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("show battle carrier succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("show battle carrier failed"))
	end
end
-- 创建一个战斗载具， 提供坐标点
function  showbc_at_pos(gm,...)
	local name = nx_string(arg[1]);
	x = nx_float(arg[2]);
	z = nx_float(arg[3]);
	if nx_show_battlecarrier_at_pos(gm,name,x,z) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("show battle carrier succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("show battle carrier failed"))
	end
end
-- 跳下战斗载具
function  getoffbc(gm)
	if nx_getoff_bc(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("getoff battle carrier succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("getoffba ttle carrier failed"))
	end
end
-- 重新装载战斗载具路点资源
function  reload_bc_path(gm)
	if nx_reload_bc_roadpoint(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload battle carrier load path poiint succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload battle carrier load path poiint failed"))
	end
end

-- 自动行走的战斗载具返回
function  bc_return(gm)
	if nx_return_start_point(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set battle carrier return succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set battle carrier return failed"))
	end
end
-- 变身为一个战斗载具
function  changerole_to_bc(gm,...)
	local name = nx_string(arg[1]);
	if nx_battlecarrier_change_role(gm,name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change role to battle carrier succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change role to battle carrier failed"))
	end
end


--清空战场排队表
function  clear_battlewar_rec(gm)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_clear_battlewar_recg ok"))
	nx_clear_battlewar_rec(gm)

end

-- 骑马，参数为技能ID
function  getonride(gm,...)
	local skill_id = nx_string(arg[1]);
	if get_on_ride(gm,skill_id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("get on ride succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("get on ride failed"))
	end
end

-- 执行Qte，参数为Qte事件的编号
function qte_start(gm, ...)
	local event_id = nx_string(arg[1]);
	if qte_excute_event(gm, event_id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start qte succeed " ..event_id))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start qte failed " ..event_id))
	end
end

-- 执行Qte，参数为Qte事件的编号
--(测试由CLIENT_CUSTOMMSG_QTEEVENT_START发来的数据)
--此函数参数和 qte_start形式是等同
function qte_cstart(gm, ...)
	local event_id = nx_string(arg[1]);
	if qte_custom_excute_event(gm, event_id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("cstart qte succeed " ..event_id))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("cstart qte failed " ..event_id))
	end
end

-- 停止Qte
function qte_stop(gm, ...)
	if qte_stop_event(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("stop qte succeed "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("stop qte failed "))
	end
end

-- Qte 重新加载资源
function qte_reload(gm, ...)
	if qte_reload_resource(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("qte reload succeed "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("qte reload failed "))
	end
end

-- 重新加载执行器文件并刷新场景
function reload_scene_trigger(gm, ...)
	if nx_reload_processor_resource(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload processor files succeed "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload processor files failed "))
	end
end

-- 重新加载执行器文件并刷新场景
function reload_processor_resource(gm, ...)
	if nx_reload_processor_resource(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload processor files succeed "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload processor files failed "))
	end
end

-- 原地复活
function relive(gm, ...)
    if nx_relive(gm, nx_int(1)) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("player relive succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("player relive fail ..."));
    end
end

-- 开始场景触发器事件
function start_es(gm, ...)
    if nx_start_es(gm, nx_int(arg[1])) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start event spring succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start event spring fail ..."));
    end
end

-- 结束场景触发器事件
function end_es(gm, ...)
    if nx_end_es(gm, nx_int(arg[1])) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("end event spring succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("end event spring fail ..."));
    end
end

-- 重载场景触发器资源
function reload_es(gm, ...)
    if nil == nx_reload_es(gm) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload event spring fail ..."));
    end
end

-- 重载scenes.ini
function reload_scene_base_info(gm, ...)
    if nx_reload_scene_base_info(gm) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload scene base info succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload scene base info fail ..."));
    end
end

-- 获取场景最大人数
function get_scene_max_player(gm, ...)
    local max_player = nx_get_scene_max_player(gm, nx_int(arg[1]));
    if max_player then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "scene_id: " .. nx_string(arg[1]) .. "; MaxPlayer: " .. nx_string(max_player));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "get scene max player error ...");
    end
end

-- 设置场景最大人数
function set_scene_max_player(gm, ...)
    if nx_set_scene_max_player(gm, nx_int(arg[1]), nx_int(arg[2])) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set scene max plyaer succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set scene max player error ..."));
    end
end

-- 查询刷怪器int属性
function query_creator_prop_int(gm, ...)
	local npc = get_prop_target(gm);
	if not nx_exists(npc) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_query_creator_prop_int(gm, npc, prop);
    if value then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "query succeed: " .. nx_string(prop) .. " = " .. nx_string(value));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "query fail: " .. nx_string(prop));
    end
end

-- 设置刷怪器int属性
function set_creator_prop_int(gm, ...)
	local npc = get_prop_target(gm);
	if not nx_exists(npc) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_set_creator_prop_int(gm, npc, prop, nx_int(arg[2]));
    if value then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "set succeed: " .. nx_string(prop) .. " = " .. nx_string(value));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "set fail: " .. nx_string(prop));
    end
end

-- 查询刷怪器float属性
function query_creator_prop_float(gm, ...)
	local npc = get_prop_target(gm);
	if not nx_exists(npc) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_query_creator_prop_float(gm, npc, prop);
    if value then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "query succeed: " .. nx_string(prop) .. " = " .. nx_string(value));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "query fail: " .. nx_string(prop));
    end
end

-- 设置刷怪器float属性
function set_creator_prop_float(gm, ...)
	local npc = get_prop_target(gm);
	if not nx_exists(npc) then
		return;
	end

    local prop = nx_string(arg[1]);
    local value = nx_set_creator_prop_float(gm, npc, prop, nx_float(arg[2]));
    if value then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "set succeed: " .. nx_string(prop) .. " = " .. nx_string(value));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "set fail: " .. nx_string(prop));
    end
end

-- 得到当前的区域id
function get_area_id(gm, ...)
    local area_id = nx_get_area_id(gm);
    if area_id then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "area_id: " .. nx_string(area_id));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "get area id error ...");
    end
end

-- 测试对象直线是否可行走
function trace_line_walk(gm, ...)
	local x,y,z,o = nx_get_position(gm);

    local step = nx_float(arg[1]);
    local dest_x = nx_float(arg[2]);
    local dest_z = nx_float(arg[3]);

    local ret,new_x,new_y,new_z = nx_trace_line_walk(step, x, y, z, dest_x, dest_z);

    if ret then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "trace succeed ...");
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "trace fail, new pos: " .. nx_string(new_x) .. ", " .. nx_string(new_y) .. ", " .. nx_string(new_z));
    end
end

--修改货币
function setcapital(gm,...)
    local sequence = arg[1] --货币类型
    local mount = arg[2]    --货币量

    if mount==nil then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setcapital  money_type  money_count"))
        return
    end

    if nx_int64_large_int64(nx_int64(arg[2]), nx_int64(0)) then
    	nx_inc_capital(gm, 1, nx_int(arg[1]), nx_int64(arg[2]))
    else
    	nx_dec_capital(gm, 1, nx_int(arg[1]), nx_int64_dec_int64(nx_int64(0), nx_int64(arg[2])))
    end
end

--修改货币(不受每日限制)
function setcapital_nolimit(gm,...)
    local sequence = arg[1] --货币类型
    local mount = arg[2]    --货币量

    if mount==nil then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setcapital  money_type  money_count"))
        return
    end

    if nx_int64_large_int64(nx_int64(arg[2]), nx_int64(0)) then
    	nx_inc_capital(gm, 2, nx_int(arg[1]), nx_int64(arg[2]))
    else
    	nx_dec_capital(gm, 2, nx_int(arg[1]), nx_int64_dec_int64(nx_int64(0), nx_int64(arg[2])))
    end
end

function clear_all_skill_cd(gm, ...)
	if table.getn(arg) ~= 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("format: clear_all_skill_cd"));
    end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear_all_skill_cd invalid target"));
		return
	end

    nx_clear_all_skill_cooldown(target)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear_all_skill_cd succeed"));
end

--清除所有cd
function clear_all_cd(gm, ...)
    nx_clear_all_cooldown(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear_all_cd succeed"));
end

function delcompletedquest(gm, ...)

	if table.getn(arg) == 0 then
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: delcompletedquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_del_completed_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Del Completed quest " .. nx_string(questid) .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Del Completed quest " .. nx_string(questid) .. " failed."))
	end

end

function search_player_in_scene(gm, ...)

	local custom_scene_id;

	if table.getn(arg) == 0 then
		custom_scene_id = -1
	else
		custom_scene_id = nx_int(arg[1])
	end

	--返回结果定义
	local SUCCESS = 0
	local OBJ_NOT_EXIST = 1
	local SCENE_ID_NOT_EXIST = 2

	local result = nx_search_player_in_scene(gm, custom_scene_id)

	local list_count = nx_number(table.getn(result))
	local index = 1
	local loop_num = 0
	local loop_num_max = 1000

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(""));

	while (index < list_count) and (loop_num < loop_num_max) do

		local player_name = nx_string(result[index])
		local account = nx_string(result[index + 1])
		local pos_x = nx_int(result[index + 2])
		local pos_y = nx_int(result[index + 3])
		local pos_z = nx_int(result[index + 4])

		local info = string.format("[%s] [%s] [%s,%s]", player_name, account, nx_string(pos_x), nx_string(pos_z))

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(info));

		index = index + 5
		loop_num = loop_num + 1
	end

	local count_info = string.format("[total: %s player]", nx_string(loop_num))

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(count_info));

end

function delallquests(gm,...)

	-- 先放弃所有任务
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Warning: trying to giveup all quest......"));
	nx_clear_quest_data(gm)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("giveup all quest successed"));
end

function acceptquest(gm,...)
    if table.getn(arg) == 0 then
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: acceptquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_accept_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Accept quest " .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Accept quest " .. nx_string(questid) .. "failed."))
	end
end

function submitquest(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: submitquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_submit_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Submit quest " .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Submit quest " .. nx_string(questid) .. "failed."))
	end
end

function giveupquest(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: giveupquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_giveup_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Submit quest " .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Submit quest " .. nx_string(questid) .. "failed."))
	end
end

function showacceptpage(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: showacceptpage <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_show_accept_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Show accept quest " .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Show accept quest " .. nx_string(questid) .. "failed."))
	end
end

function showsubmitpage(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: showsubmitpage <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	-- 强制接
	if nx_show_submit_quest(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Show Submit quest " .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Show Submit quest " .. nx_string(questid) .. "failed."))
	end
end

function reload_quest_resource(gm,...)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_quest_resource "  .. " begin."))

	nx_reload_quest_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_quest_resource "  .. " end."))
end

function setquestfail(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: submitquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_quest_fail "  .. " begin."))

	if nx_set_quest_fail(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_quest_fail" .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_quest_fail" .. nx_string(questid) .. "failed."))
	end
end

function setquestcansubmit(gm,...)
    if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: submitquest <questid>"))
        return
	end

	local questid = nx_int(arg[1])

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquestcansubmit "  .. " begin."))

	if nx_set_quest_cansubmit(gm, questid) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquestcansubmit" .. nx_string(questid) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquestcansubmit" .. nx_string(questid) .. "failed."))
	end
end

function setcriteriaprogress(gm,...)
    if table.getn(arg) ~= 2 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setcriteriaprogress <criteria_id, proress>"))
        return
	end

	local criteria_id = nx_int(arg[1])
	local progress = nx_int(arg[2])

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setcriteriaprogress "  .. " begin."))

	if nx_set_criteria_progress(gm, criteria_id, progress) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setcriteriaprogress" .. nx_string(criteria_id) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setcriteriaprogress" .. nx_string(criteria_id) .. "failed."))
	end
end

function addcriteriaprogress(gm,...)
    if table.getn(arg) ~= 2 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: addcriteriaprogress <criteria_id, proress>"))
        return
	end

	local criteria_id = nx_int(arg[1])
	local progress = nx_int(arg[2])

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("addcriteriaprogress "  .. " begin."))

	if nx_add_criteria_progress(gm, criteria_id, progress) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("addcriteriaprogress" .. nx_string(criteria_id) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("addcriteriaprogress" .. nx_string(criteria_id) .. "failed."))
	end
end

function setquesttrack(gm,...)
    if table.getn(arg) ~= 2 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setquesttrack <quest_id>"))
        return
	end

	local quest_id = nx_int(arg[1])

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquesttrack "  .. " begin."))

	if nx_set_quest_track(gm, criteria_id, progress) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquesttrack" .. nx_string(quest_id) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("setquesttrack" .. nx_string(quest_id) .. "failed."))
	end
end

function getallquests(gm)
    local quests_list = nx_get_all_quests(gm)
    local quests_str = ""

    for k,v in pairs(quests_list) do
		local quest_title = nx_get_quest_title(nx_int(v))
		quests_str = "quest_id:".." "..nx_string(v).." "
		nx_sysinfo(gm, TIPSTYPE_GMINFO, nx_string(quest_title), nx_widestr(quests_str))
    end
end

function getcriterias(gm, ...)
    if table.getn(arg) ~= 1 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: getcriterias <quest_id>"))
        return
	end

    local quest_id = nx_int(arg[1])

    local criterias_list = nx_get_criterias(gm, quest_id)
    local criterias_str = ""

    for k,v in pairs(criterias_list) do
        criterias_str = criterias_str .. " " .. nx_string(v)
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("criterias:" .. criterias_str))
end

function reload_skillpack(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:reload_config <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	if nx_reload_skillpack(file_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " failed."))
	end
end

function reload_petfight(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:reload_petfight <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	if nx_reload_petfight(file_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload " .. file_name .. " failed."))
	end
end

--检查技能是否正确
function check_skill(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_skill <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	if nx_check_skill(file_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("check_skill " .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("check_skill " .. " failed."))
	end
end

--检查npc配置是否正确
function check_npcConfig(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_npcConfig <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	if nx_check_npccreate(file_name) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("check_npcConfig " .. " succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("check_npcConfig " .. " failed."))
	end
end

--使用技能
function npc_use_skill(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:check_skill <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	nx_npc_use_skill(gm,file_name)
end

--技能CD
function npc_query_skillCD(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:npc_query_skillCD <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	nx_npc_query_skillCD(gm,file_name)
end

-- 查询npc的技能列表
function npc_query_skill_list(gm, ...)
    local target = get_prop_target(gm)
	local skill_list = nx_npc_query_skill_list(target);
    if skill_list == nil then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("npc query skill list error ..."));
        return;
    end

	local skill_count = table.getn(skill_list) - 1;
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "=====");
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "npc_id: " .. nx_string(skill_list[1]) .. ", skill_count: " .. nx_string(skill_count));
    for i = 1, skill_count do
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_string(i) .. ": " .. skill_list[i + 1]);
    end
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "=====");
end


--重新载入ini资源文件。
function reload_path_resource(gm,...)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_path_resource "  .. " begin."))

	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: reload_path_resource"));
		return;
	end


	nx_reload_path_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_path_resource "  .. " end."))
end

function reload_npc_path(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format:reload_npc_path <config file>"))
		return
	end

	local file_name = nx_string(arg[1])
	nx_reload_npc_path(gm,file_name)

end

--查询事件中不存在的条件，行为等
function check_event_spring(gm, ...)
	-- 检查参数数量
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: <es_id> 0 to check all events"))
		return
	end
	local es_id = nx_string(arg[1])

	local es_list  = nx_get_event_by_id(gm,es_id)
	--打开事件LOG文件
	nx_open_event_log()
	for i = 1, table.getn(es_list) do
		--检查behavior_id是否存在
		local beh_list = nx_not_exist_behavior_id(gm,es_list[i])
		for j = 1, table.getn(beh_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " behavior_id " .. nx_string(beh_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " behavior_id " .. nx_string(beh_list[j]) .. " not exist\n")
		end

		--检查condition_id是否存在
		local con_list = nx_not_exist_condition_id(gm,es_list[i])
		for j = 1, table.getn(con_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " condition_id " .. nx_string(con_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " condition_id " .. nx_string(con_list[j]) .. " not exist\n")
		end

		--检查notice_id是否存在
		local notice_list = nx_not_exist_notice_id(gm,es_list[i])
		for j = 1, table.getn(notice_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " notice_id " .. nx_string(notice_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " notice_id " .. nx_string(notice_list[j]) .. " not exist\n")
		end

		--检查reward_id是否存在
		local reward_list = nx_not_exist_reward_id(gm,es_list[i])
		for j = 1, table.getn(reward_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " reward_id " .. nx_string(reward_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " reward_id " .. nx_string(reward_list[j]) .. " not exist\n")
		end

		--检查switch_id是否存在
		local switch_list = nx_not_exist_switch_id(gm,es_list[i])
		for j = 1, table.getn(switch_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " switch_id " .. nx_string(switch_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " switch_id " .. nx_string(switch_list[j]) .. " not exist\n")
		end

		--检查condition_id是否存在
		local task_list = nx_not_exist_task_id(gm,es_list[i])
		for j = 1, table.getn(task_list) do
			--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "event_id " .. nx_string(es_list[i]) .. " task_id " .. nx_string(task_list[j]) .. " not exist")
			nx_write_event_log(gm,"event_id " .. nx_string(es_list[i]) .. " task_id " .. nx_string(task_list[j]) .. " not exist\n")
		end
	end
	--关闭事件LOG文件
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "check event error succeed !")

	nx_close_event_log()
end

--清除玩家身上BUFF表
function clear_all_buff(gm, ...)
	nx_clear_all_buff(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear buff ok !")
end

--清除玩家第二职业
function clear_life_occupation(gm, ...)
	nx_clear_life_occupation(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear life occupation ok !")
end

--清除玩家好友列表
function clear_friend_list(gm, ...)
	nx_clear_friend_list(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear friend list ok !")
end

--清除玩家黑名单列表
function clear_black_list(gm, ...)
	nx_clear_black_list(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear black list ok !")
end

--清除玩家仇人列表
function clear_enemy_list(gm, ...)
	nx_clear_enemy_list(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear enemy list ok !")
end

--清除玩家队伍信息
function clear_team_info(gm, ...)
	nx_clear_team_info(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear team info ok !")
end

--清除副本进度
function clear_clone_progress(gm, ...)
	--nx_clear_clone_progress(gm)
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear black list ok !")
end

--清除竞技场信息
function clear_arena_info(gm, ...)
	--nx_clear_arena_info(gm)
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear black list ok !")
end

--清除战场信息
function clear_battlewar_info(gm, ...)
	--nx_clear_battlewar_info(gm)
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear black list ok !")
end

--设置事件CD
function set_event_cd(gm, ...)

	if table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: <es_id> <cd_time> to check all events"))
		return
	end

	local es_id = nx_int(arg[1])
	local cd_time = nx_int(arg[2])

	nx_set_event_cd(gm,es_id,cd_time)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "set event CD ok !")
	--if nx_add_repute(gm, force, repute) then
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add repute " .. nx_string(force) .. "succeed."))
	--else
		--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add repute " .. nx_string(force) .. "failed."))
	--end
end

--设置PVP任务CD
function set_pvp_task_cd(gm, ...)

	if table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: <pvp_id> <cd_time> to check all events"))
		return
	end

	local pvp_id = nx_int(arg[1])
	local cd_time = nx_int(arg[2])

	nx_set_pvp_task_cd(gm,pvp_id,cd_time)
	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", "clear black list ok !")
end

function reload_arenaplane_resource(gm,...)
    nx_arenaplane_reload()
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload arenaplane_resource succeed."))
end

-- 开始重置角色数据
function start_revert(gm, ...)
    if nx_start_revert(gm) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start revert spring succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("start revert spring fail ..."));
    end
end

-- 解除角色武器装备限制
function removelimit(gm)
	nx_set_property( gm, "RemoveLimit", 1 )
	return 1
end

-- 打开测试机器人界面
function open_test_robot_form(gm)
    nx_open_test_robot_form(gm)
end

-- 重建拍卖行表
function rebuild_ah_recs(gm)
    nx_rebuild_ah_recs(gm)
end

-- 清空拍卖行数据
function clear_ah_data(gm)
    nx_clear_ah_data(gm)
end

-- 重建求购行表
function rebuild_asktobuy_recs(gm)
    nx_rebuild_asktobuy_recs(gm)
end

-- 清空求购行数据
function clear_asktobuy_data(gm)
    nx_clear_asktobuy_data(gm)
end

-- 能量井免费制造载具（无需权限，不走完整流程）
function produce_carrier_free(gm, ...)
    if table.getn(arg) ~= 3 then
        nx_sysinfo(
            gm, TIPSTYPE_GMINFO, "0",
            nx_widestr("Format: <ew_id> <carrier_id>")
        )

        return
    end

    local ew_id = nx_int(arg[1])
    local camp = nx_int(arg[2])
    local car_id = nx_int(arg[3])

    nx_creat_ew_carrier(gm, ew_id, camp, car_id)
end

-- 能量井免费开启制造载具（需要权限，走正常流程）
function begin_carrier_produce(gm, ...)
    if table.getn(arg) ~= 2 then
        nx_sysinfo(
            gm, TIPSTYPE_GMINFO, "0",
            nx_widestr("Format: <ew_id> <carrier_id>")
        )

        return
    end

    local ew_id = nx_int(arg[1])
    local car_id = nx_int(arg[2])

    nx_begin_carrier_produce(gm, ew_id, car_id)
end

-- 能量井免费加速制造载具（需要权限，走正常流程）
function speed_up_carrier_produce(gm, ...)
    if table.getn(arg) ~= 1 then
        nx_sysinfo(
            gm, TIPSTYPE_GMINFO, "0",
            nx_widestr("Format: <ew_id>")
        )

        return
    end

    local ew_id = nx_int(arg[1])

    nx_speed_up_carrier_produce(gm, ew_id)
end

-- 能量井免费制造停车位（无需权限，不走完整流程）
function produce_parkingspace_free(gm, ...)
    if table.getn(arg) ~= 3 then
        nx_sysinfo(
            gm, TIPSTYPE_GMINFO, "0",
            nx_widestr("Format: <ew_id> <camp> <parkingspace_id>")
        )

        return
    end

    local ew_id = nx_int(arg[1])
    local camp = nx_int(arg[2])
    local parkingspace_id = nx_int(arg[3])

    nx_creat_ew_parkingspace(ew_id, camp, parkingspace_id)
end

function get_move_mode(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end
    local move_mode = nx_get_move_mode( target)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(target) .. " move_mode:" .. move_mode))
end

function get_rotate_speed(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

    local rotate_speed = nx_get_rotate_speed( target)

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(nx_string(target) .. " rotate_speed:" .. nx_string(rotate_speed)))
end

function setexplored(gm, ...)
	if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setexplored <area_id>"))
        return
	end

	local area_id = nx_string(arg[1])

	-- 强制接
	if nx_set_explored(gm, area_id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set explored " .. nx_string(area_id) .. "succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set explored " .. nx_string(area_id) .. "failed."))
	end
end

function clearexplored(gm)
	nx_clear_explored(gm)
end

function completeachieve(gm,...)
	if table.getn(arg) < 1 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: completeachieve <achievement_id>"))
        return
	end

	local achievement_id = nx_int(arg[1])

	nx_complete_achievement(gm, achievement_id)
end

function resetachieve(gm)
	nx_reset_achievement(gm)
end

function clear_sole_achievement(gm)
	nx_clear_sole_achievement()
end

function reloadachieve(gm)
	nx_reload_achieve_resource(gm)
end

--设置指定ID的pvp任务完成次数
function setpvptaskcount(gm,...)
	if table.getn(arg) < 2 then
		return
	end

	local taskid = nx_string(arg[1])
	local count = nx_int(arg[2])

	nx_set_pvptask_complete_count(gm,taskid,count )

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_pvptask_complete_count complete "))
end

--设置指定ID的pvp活动状态
function setpvpactivitystate(gm,...)
	if table.getn(arg) < 5 then
		return
	end

	local activityid = nx_string(arg[1])
	local opentime_hour = nx_int(arg[2])
	local opentime_min = nx_int(arg[3])
	local closetime_hour = nx_int(arg[4])
	local closetime_min = nx_int(arg[5])

	if nx_set_pvptask_activity_state(gm,activityid,opentime_hour,opentime_min, closetime_hour, closetime_min) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_pvptask_activity_state succeed "))
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("nx_set_pvptask_activity_state fail "))
    end
end

--重置pvp任务完成次数为0
function reset_pvptask_complete_count(gm)
	nx_reset_pvptask_complete_count(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reset_pvptask_complete_count succeed "))
end

--清空pvp任务(次数，当前接受的任务)
function clear_pvptask(gm)
	nx_clear_pvptask(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear_pvptask succeed "))
end

--重载pvp任务资源
function reload_pvptask(gm)
	nx_reload_pvptask(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_pvptask succeed "))
end

function battlewarbegin(gm)
    nx_battlewar_begin()
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("begin battlewar succeed."))
end

--修复背包里面装备耐久
function repair_hardiness(gm, ...)
    repairall_hardiness(gm,2)
end
--修复装备栏装备耐久
function repair_equipbox(gm,...)
    repairall_hardiness(gm,1)
end

function battlewarend(gm,...)
	if table.getn(arg) == 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: battlewarend <camp>"))
        return
	end

	local camp = nx_int(arg[1])

	if  nx_battlewar_end(camp) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("end battlewar <win camp: " .. nx_string(camp) .. ">succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("end battlewar <win camp: " .. nx_string(camp) .. ">failed."))
	end
end

function battlewarnum(gm,...)
	if table.getn(arg) < 3 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: battlewarnum <bwid> <minnum> <maxnum> "))
        return
	end

	local bwid = nx_int(arg[1])
	local minnum = nx_int(arg[2])
	local maxnum = nx_int(arg[3])

	if nx_battlewar_change_queuenum(bwid,minnum,maxnum) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change battlewar queuenum succeed."))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change battlewar queuenum failed."))
	end
end
--模拟杀死NPC凋落物品
--npcdropn value1 value2        value1表示NPC的ConfigID; value2表示杀死的次数
function npcdropn(gm, ...)
	local id
	local num
	if table.getn(arg) == 1 then
		id = get_prop_target(gm)
		num = arg[1];

	elseif table.getn(arg) == 2 then
		id = arg[1];
		num = arg[2];
	else
	--输入不正确
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("npcdropn failed."));
	end
	--显示物品信息
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  List Begin:" .. "====="));

	--特殊掉落 简单规则
	local dropspecialitems = nx_get_special_npcdropn(nx_get_scene(),nx_string(id),nx_int(num))
	if  table.getn(dropspecialitems) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  specialDropID  ItemList Begin:" .. "====="));
		--显示掉落物品
		for i = 1, table.getn(dropspecialitems) do
		--获得格式 名称  num
			local name_id = dropspecialitems[i];
			nx_sysinfo(gm, TIPSTYPE_GMINFO,"drop", nx_widestr(name_id));
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  specialDropID  ItemList end" .. "====="));
	end

	--world掉落 复杂规则
	local dropworlditems = nx_get_world_npcdropn(nx_get_scene(),nx_string(id),nx_int(num))
	if table.getn(dropworlditems) ~= 0 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  WorldDropID  ItemList Begin:" .. "====="));
		--显示掉落物品
		for i = 1, table.getn(dropworlditems) do
		--获得格式 名称  num
			local name_id = dropworlditems[i];
			nx_sysinfo(gm, TIPSTYPE_GMINFO,"drop", nx_widestr(name_id));
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  WorldDropID  ItemList end" .. "====="));
	end
   nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "npcdropn  List end" .. "====="));

end


--模拟某个掉落包凋落物品
--DropIDdropn value1 value2        value1表示掉落包; value2表示杀死的次数
function DropIDdropn(gm, ...)
	local id
	local num
	if table.getn(arg) == 2 then
		id = arg[1];
		num = arg[2];
	else
	--输入不正确
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("DropIDdropn failed."));
	end
	--显示物品信息
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  List Begin:" .. "====="));

	--特殊掉落 简单规则
	local dropspecialitems = nx_get_special_DropIDdropn(nx_get_scene(),nx_string(id),nx_int(num))
	if table.getn(dropspecialitems) ~= 0  then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  specialDropID  ItemList Begin:" .. "====="));
		--显示掉落物品
		for i = 1, table.getn(dropspecialitems) do
		--获得格式 名称  num
			local name_id = dropspecialitems[i];
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "drop", nx_widestr(name_id));
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  specialDropID  ItemList end" .. "====="));
	end

	--world掉落 复杂规则
	local dropworlditems = nx_get_world_DropIDdropn(nx_get_scene(),nx_string(id),nx_int(num))
	if table.getn(dropworlditems) ~= 0  then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  WorldDropID  ItemList Begin:" .. "====="));
		--显示掉落物品
		for i = 1, table.getn(dropworlditems) do
		--获得格式 名称  num
			local name_id = dropworlditems[i];

			nx_sysinfo(gm, TIPSTYPE_GMINFO, "drop", nx_widestr(name_id));
		end
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  WorldDropID  ItemList end" .. "====="));
	end
   nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "DropIDdropn  List end" .. "====="));

end

--清理道具监控公服数据
function cleardropitem(gm, ...)
	local b_success =  nx_clear_dropitem_Manage(nx_get_scene())
	if b_success then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "cleardropitem  success" .. "====="));
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "cleardropitem  failed" .. "====="));
	end
end

-- 创建公会
function createguild(gm, ...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("arguments error"))
		return
	end
	local guild_name = nx_widestr(arg[1])
	if nx_create_guild(gm, guild_name, nx_widestr("GM Create Guild")) then
		local info = nx_widestr("create guild " .. nx_string(guild_name) .. " succeed")
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", info)
	else
		local info = nx_widestr("create guild " .. nx_string(guild_name) .. " failed")
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", info)
	end
end
-- 清除公会信息
function clearguild(gm, ...)
	if nx_clear_guild(gm) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear guild data succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear guild data failed"))
	end
end
-- 设置公会成员人数上限
function setguildtoplimit(gm, ...)
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("arguments error"))
		return
	end
	local maxnumstr = nx_string(arg[1])
	if nx_set_guild_toplimit(gm, maxnumstr) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set guild member toplimit succeed"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set guild member toplimit failed"))
	end
end

--设置副本进度
function set_clonescene_progress(gm,...)
    nx_set_clonescene_progress(gm,nx_int(arg[1]))
	--nx_msgbox(nx_string(arg[1]))
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set clonescene progress ok ! "));
end

--学习生活技能
function studyjob(gm,...)
    nx_learn_life_occupation(gm,nx_int(arg[1]))
	--nx_msgbox(nx_string(arg[1]))
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("learn life occupation ok ! "));
end

--升级生活技能
function life_occup_level(gm,...)
	nx_life_occup_level(gm,nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set life occupation level ok ! "));
end

--设置最大体力值
function set_max_physical(gm,...)
	nx_set_max_physical(gm,nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set max physical ok ! "));
end

--设置体力值
function set_physical(gm,...)
	nx_set_physical(gm,nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set max physical ok ! "));
end

--设置生活职业熟练度
function set_job_exp(gm,...)
	nx_set_job_exp(gm,nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set job Exp ok! "));
end

--接受PVP任务
function setPVPtask(gm,...)
	local id = nx_string(arg[1]);

	if nx_accept_pvptask(gm, id) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add pvptask task ok! " .. id));
		return
	end

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add pvptask failed! ".. id));
end

--竞技场竞技属性
function arena_update_property(gm,...)
	nx_arena_update_property(gm, nx_int(arg[1]), nx_int(arg[2]), nx_widestr(arg[3]), nx_int(arg[4]), nx_int(arg[5]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_update_property ok! "));
end

--竞技场荣耀值
function arena_honour(gm,...)

	nx_arena_honour(gm, nx_widestr(arg[1]), nx_int(arg[2]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_honour ok! "));
end

--竞技场赛季时间
function arena_match_time(gm,...)
	nx_arena_match_time(gm, nx_string(arg[1]), nx_int(arg[2]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_match_time ok! "));
end

--竞技场强制胜利
function arena_force_win(gm,...)
	nx_arena_force_win(gm, nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_force_win ok! "));
end

--竞技场道具限制
function arena_limit_item(gm,...)
	nx_arena_limit_item(gm, nx_string(arg[1]), nx_int(arg[2]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_limit_item ok! "));
end

--竞技场等级限制
function arena_limit_level(gm,...)
	nx_arena_limit_level(gm, nx_int(arg[1]))
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set arena_limit_level ok! "));
end

-- 设置vip
function addvip(gm, ...)
	-- nx_add_vip_time(gm, nx_int(arg[1]) )
	-- nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set vip time ok! "));
end

-- 查询当前游戏世界资金情况
function sgmoney(gm, ...)
	nx_show_total_capidal(gm)
end

-- 设置当前游戏世界资金情况 这个gm命令仅供测试
function setgmoney(gm, ...)
	local num = table.getn(arg)
	if num < 3 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "para too less! Format: setgmoney capital_type type value");
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "type只能是 1, 2, 3, 4, 10, 11");
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "1表示调整系统资金总量, 2表示将可出售资金划分到可奖励资金中");
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "3表示设置兑换开关, 4表示设置奖励开关");
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "11表示设置离线奖励数量, 10表示设置离线奖励开关");
		return
	end

	local capital_type = nx_int(arg[1])
	local type = nx_number(arg[2])
	local value = nx_int64(arg[3])

	local can_op = {[1] = 1, [2] = 1, [3] = 1, [4] = 1, [10] = 1, [11] = 1}

	if can_op[type] == nil then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "type只能是 1, 2, 3, 4, 10, 11");
		return
	end

	nx_set_total_money(gm, capital_type, nx_int(type), value)
end

--重置升级技能
function ResetSkill(gm,...)
	nx_reset_skill_level(gm);

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reset skill succeed"));
end

--设置当前属性操作的对象的身份为X级犯人和封停时间
--setexilestatus 身份 时间
function setexilestatus(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: setexilestatus <status> <time>"));
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local status = nx_int(arg[1]);
	local time = nx_int(arg[2])

	nx_set_status(target,status,time)

	--nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("SetExileStatus succeed."));
end

--技能无消耗
function noconsume(gm,...)
    nx_skill_noconsume(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change consume succeed"));
end

--技能无CD
function nocooldown(gm,...)
    nx_skill_nocooldown(gm)
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("change cooldowntime succeed"));
end

--查询事件最大收益
function query_es_reward(gm, ...)
	--没有则查询所有事件
	if table.getn(arg) == 0 then
		nx_open_event_log()
		local reward_list = nx_query_es_reward(gm,0)
		for i = 1, table.getn(reward_list)/4 do
			local offset = (i - 1)*4
			nx_write_event_log(gm,"es: " .. nx_string(reward_list[offset + 1]) .. "  exp: " .. nx_string(reward_list[offset + 2]) ..
		"  money: " .. nx_string(reward_list[offset + 3]) .. "  hunt: " .. nx_string(reward_list[offset + 4]) .. "\n")
		end
		nx_close_event_log()
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("all ok"))
		return
	end
	--查询某个事件
	if table.getn(arg) == 1 then

		local reward_list = nx_query_es_reward(gm,nx_int(arg[1]))

		if table.getn(reward_list) < 4 then
			return
		end

		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0",
		nx_widestr("es: " .. nx_string(reward_list[1]) .. "  exp: " .. nx_string(reward_list[2]) ..
		"  money: " .. nx_string(reward_list[3]) .. "  hunt: " .. nx_string(reward_list[4])))

		return
	end

end

-- 查询当前场景的所有预加载npc
function query_preload_npc(gm, ...)
    local ret = nx_query_preload_npc(gm);

    local total_npc_num = nx_int(ret[1]);
    local total_npc_type = nx_int(ret[2]);

    if total_npc_num <= 0 then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("no preload npc exists ..."));
        return;
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("===================="));

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("total preload npc num: " .. total_npc_num));
    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(""));

    for i = 1, total_npc_type do
        local npc_id = nx_string(ret[2 * i + 1]);
        local npc_num = nx_string(ret[2 * i + 2]);

        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(npc_id  .. ": " .. npc_num));
    end

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("===================="));
end

--设置pvp任务实力值愤怒值
function add_pvp_prop(gm, ...)
    local prop_name = nx_string(arg[1]);
    local camp = nx_int(arg[2])
    local val = nx_int(arg[3])

    nx_add_pvp_prop(prop_name, camp, val)

    nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("add pvp prop value"))

end

-- 设置当前角色能量井居民身份等级
function setEnergyWellStatus(gm, ...)
	local well_id = nx_string(arg[1])
	local level = nx_int(arg[2])
	nv_set_wellResidentLevel(gm, well_id, level)
end

--清除能量井申请表
function ew_clear_apply_rec(gm,...)
	nx_ew_clear_apply_rec(gm)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set ew_clear_apply_rec ok! "));
end

--
function ew_set_guildwar_state(gm,...)
	local ew_id = nx_string(arg[1])
	local state = nx_int(arg[2])
	nx_ew_set_guildwar_state(gm,ew_id,state)
end

--
function ew_set_well_own(gm,...)
	local ew_id = nx_string(arg[1])
	local state = nx_widestr(arg[2])
	nx_ew_set_well_own(gm,ew_id,state)
end

function ew_clear_well_own(gm,...)
	local ew_id = nx_string(arg[1])
	nx_ew_clear_well_own(gm,ew_id)
end
--强制能量井战场结束
function ew_force_fight_end(gm,...)
	local ew_id = nx_string(arg[1])
	nx_ew_force_fight_end(gm, ew_id)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set ew_force_fight_end ok! "));
end

--强制能量井战场结束
function ew_force_fight_open(gm,...)
	local ew_id = nx_string(arg[1])
	nx_ew_force_fight_open(gm, ew_id)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set ew_force_fight_open ok! "));
end

function revertprop(gm)
	nx_revert_prop(gm)
end

-- 清除排行榜
function clear_rank(gm, ...)
    if nx_clear_rank(gm) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear rank succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("clear rank fail ..."));
    end
end


--设置ITEM的剩余有效时间
function set_leftime(gm,...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: nx_set_leftime  <value>"));--格式：set 属性名称 属性值
		return;
	end

	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	local leftime = nx_int(arg[1])
	nx_set_leftime(target,nx_int(leftime) )
end

function openprompt_clearstate(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if nx_openprompt_clearstate(gm, target) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("openprompt_clearstate ok! "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("openprompt_clearstate error! "))
	end
end

function openprompt_setstate(gm, ...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if nx_openprompt_setstate(gm, target, nx_int(arg[1]), nx_int(arg[2])) then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("openprompt_setstate ok! "))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("openprompt_setstate error! "))
	end
end

--设置强化等级
function set_star_level(gm, ...)
    if table.getn(arg) < 1 then
        return  0
    end

    local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

    local star_level = arg[1]
    nx_strengthen(target,nx_int(star_level) )
end

--treasureboxranpro param1 param2   param1:宝箱id  param2:测试打开次数
function treasureboxranpro(gm, ...)
	local id
	local num
	if table.getn(arg) == 2 then
		id = arg[1];
		num = arg[2];
	else
	--输入不正确
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("test_treasurebox failed."));
	end
	--显示物品信息
	local lst_items = nx_treasurebox_ranpro(nx_string(id),nx_int(num))
	if table.getn(lst_items) ~= 0  then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "test_treasurebox  ItemList Begin:" .. "====="));
		--显示宝箱中物品
		for i = 1, table.getn(lst_items) do
		    --获得格式 名称  num
			local name_id = lst_items[i];
			nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr(name_id));
		end
	end
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("=====" .. "test_treasurebox   ItemList end" .. "====="));
end

--设置战场积分
function set_battlewar_point(gm, ...)
    local camp = nx_int(arg[1])
    local point = nx_int(arg[2])

    nx_set_battlewar_point(camp, point)
end

--设置每日战场次数
function set_day_battlewar_count(gm, ...)
    local count = nx_int(arg[1])

    nx_set_day_battlewar_count(gm, count)
end

-- 填充player身上的存档表(机器人测试用)
function fill_records(gm, ...)
    if nx_fill_records(gm) then
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("fill records succeed ..."));
    else
        nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("fill records fail ..."));
    end
end

--add by hutian 2013-3-29
-- 设置活动开关
function set_switch(gm, ...)
	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set_switch <id> <bopen>"))
		return
	end

	local id = nx_int(arg[1])
	local open = nx_int(arg[2])

	local flag = nx_set_switch(id, open)

	if flag then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_switch success"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_switch failed"))
	end
end

-- 查询活动开关
function query_switch(gm, ...)
	if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: query_switch <id>"))
		return
	end

	local id = nx_int(arg[1])

	local flag = nx_query_switch(id)

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("query_switch :"..nx_string(flag)))
end

--重载开关资源 \ini\Rule\SwitchManager.xml
function reload_switch_resource(gm, ...)
	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_switch_resource "  .. " begin."))

	nx_reload_switch_resource()

	nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("reload_switch_resource "  .. " ok."))
end


--开始记录某个玩家上发的消息
function begin_logmsg(gm,...)
	local target = get_prop_target(gm)
	if not nx_exists(target) then
		return
	end

	if not nx_exists(target) or nx_type(target) ~= 2 then --TYPE_PLAYER	= 2
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("cur obj not player object"));--当前选取对象不是玩家对象
		return
	end


	nx_begin_logmsg(target, gm)
end

--停止记录玩家上发
function stop_logmsg(gm,...)
	nx_stop_logmsg()
end

--购买一个档案
function buy_charge_order(gm,...)
    if table.getn(arg) < 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("param "))
		return
	end

	local id = nx_int(arg[1])
	nx_buy_charge_mode(gm,nx_int(id))
end

--add by huanglh 2013-5-15
-- 设置日志开关
function set_log_switch(gm, ...)
	if table.getn(arg) ~= 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set_log_switch <index> <bopen>"))
		return
	end

	local index = nx_int(arg[1])
	local open  = nx_int(arg[2])

	local flag = nx_set_log_switch(index, open)

	if flag then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_log_switch success"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_log_switch failed"))
	end
end

--设置记录log心跳事件
function set_time_money(gm, ...)  --时间单位——分钟
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set_time_money <value>"))
		return
	end

	local value = nx_int(arg[1])

	local flag = nx_set_time_money(value)

	if flag then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_time_money success"))
	else
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("set_time_money failed"))
	end
end
--清除能量井数据
function clear_domain_ew_data(gm, ...)  --时间单位——分钟
	nx_clear_domain_ew_data()
end

--捐献能量井材料
function contribute_ew_mats(gm, ...)  --时间单位——分钟
	if table.getn(arg) ~= 1 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: contribute_ew_mats <sum> "))
		return
	end

	local sum = nx_int(arg[1])
	nx_contribute_ew_mats(sum)
end

function set_scene_cant_relive(gm, ...)

	if table.getn(arg) < 2 then
		nx_sysinfo(gm, TIPSTYPE_GMINFO, "0", nx_widestr("Format: set_scene_cant_relive <scene id> <0/1>"));
		return;
	end

    local scene = nx_int(arg[1])
    local cant_relive = nx_int(arg[2])

    nx_set_scene_cant_relive( scene, cant_relive)

end

-- end of file gm.lua
