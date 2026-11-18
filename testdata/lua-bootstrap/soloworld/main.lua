--游戏服务器主控脚本

--程序状态
local WAPS_UNKNOWN = 0	--未知
local WAPS_STARTING = 1	--正在启动
local WAPS_READY = 2	--就绪（但是未开放服务）
local WAPS_OPENING = 3	--正在开放服务
local WAPS_OPENED = 4	--已开放服务
local WAPS_CLOSING = 5	--正在关闭
local WAPS_CLOSED = 6	--已关闭服务
local WAPS_QUITING = 7	--正在退出

--入口函数
function main(world)
	nx_set_value("world", world)

	--快速日志系统
	world.UseQuickLog = true
	--应用进程记录
	world.UseAppRecord = true

	--屏幕输出字符颜色
	world:SetConsoleColor("LightGreen")

	--读取程序配置
	local ini = nx_create("IniDocument")

	--配置文件名
	--ini.FileName = "game_world.ini"
	ini.FileName = nx_main_config()

	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("can't read config file!")
		nx_destroy(ini)
		return 0
	end

	--游戏分区编号
	local district_id = ini:ReadInteger("Main", "DistrictID", 1)
	--服务器编号
	local server_id = ini:ReadInteger("Main", "ServerID", 1)
	--服务器名称
	local server_name = ini:ReadString("Main", "ServerName", "")
	--最大线人数
	local max_players = ini:ReadInteger("Main", "MaxPlayers", 8192)
	--保存玩家数据的间隔分钟数
	local save_minutes = ini:ReadInteger("Main", "SaveMinutes", 20)
	--刷新账户信息的间隔分钟数
	local flush_minutes = ini:ReadInteger("Main", "FlushMinutes", 0)
	--创建角色对角色名字长度的限制
	local rolenamesizelimit = ini:ReadInteger("Main", "RoleNameSizeLimit", 16)
	--是否需要登录排队
	local player_queue = nx_boolean(ini:ReadString("Main", "PlayerQueue", "true"))
	--是否开启VIP优先排队功能
	local vip_queue = nx_boolean(ini:ReadString("Main", "VipQueue", "true"))

	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 0
	--local inner_port = 16001

	inner_addr = ini:ReadString("Inner", "Addr", inner_addr)
	inner_port = ini:ReadInteger("Inner", "Port", inner_port)

	--广播地址
	local broad_addr = "127.255.255.255"
	local broad_port = 15000

	broad_addr = ini:ReadString("Broadcast", "Addr", broad_addr)
	broad_port = ini:ReadInteger("Broadcast", "Port", broad_port)

	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15001

	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)

	--计费服务器
	local charge_inuse = nx_boolean(ini:ReadString("Charge", "Inuse", ""))
	local charge_addr = ini:ReadString("Charge", "Addr", "")
	local charge_port = ini:ReadInteger("Charge", "Port", 0)

	--客服服务器
	local gmcc_inuse = nx_boolean(ini:ReadString("Gmcc", "Inuse", ""))
	local gmcc_addr = ini:ReadString("Gmcc", "Addr", "")
	local gmcc_port = ini:ReadInteger("Gmcc", "Port", 0)

	--管理服务器
	local manage_inuse = nx_boolean(ini:ReadString("ManageServer", "Inuse", ""))
	local manage_addr = ini:ReadString("ManageServer", "Addr", "")
	local manage_port = ini:ReadInteger("ManageServer", "Port", 0)

    --监控服务器
	local monitor_inuse = nx_boolean(ini:ReadString("Monitor", "Inuse", ""))
	local monitor_addr = ini:ReadString("Monitor", "Addr", "")
	local monitor_port = ini:ReadInteger("Monitor", "Port", 0)

    --转储服务器
	local dumpserver_inuse = nx_boolean(ini:ReadString("DumpServer", "Inuse", ""))
	local dumpserver_addr = ini:ReadString("DumpServer", "Addr", "")
	local dumpserver_port = ini:ReadInteger("DumpServer", "Port", 0)

	nx_destroy(ini)

	--绑定脚本
	nx_bind_script(world, "main", "echo_init")

	world.AppName = "world_" .. nx_string(server_id)
	world.DistrictId = district_id
	world.ServerId = server_id
	world.ServerName = nx_widestr(server_name)
	world.MaxPlayers = max_players
	world.SaveMinutes = save_minutes
	world.FlushMinutes = flush_minutes
	world.PlayerQueue = player_queue
	world.VipQueue = vip_queue
	--每个帐号可以创建的角色数量
	world.AccountRoles = 5
	--创建角色对角色名字长度的限制
	world.RoleNameSizeLimit = rolenamesizelimit
	world.InnerAddr = inner_addr
	world.InnerPort = inner_port
	world.BroadAddr = broad_addr
	world.BroadPort = broad_port
	world.EchoAddr = echo_addr
	world.EchoPort = echo_port
	--保证启动日志服务
	world.LogLauncher = "game_log game_log.ini daemon"

	--加载名字过滤文件
	world:LoadNameFilter(nx_resource_path() .. "filter.txt")

	--加载验证码字符图像
	load_char_image(world)

	--添加普通场景
	load_scene_config(world)

	--计费连接
	if charge_inuse then
		world:SetChargeAddress(charge_addr, charge_port)
	end

	--客服连接
	if gmcc_inuse then
		world:SetGmccAddress(gmcc_addr, gmcc_port)
	end

	--管理连接
	if manage_inuse then
		world:SetManageAddress(manage_addr, manage_port)
	end

	--启动游戏服务
	world:Start()

	--实际使用的内网侦听端口
	inner_port = world.InnerPort

	--显示应用程序名
	world:WriteLine("app name " .. world.AppName)
	--显示通讯地址和端口
	world:WriteLine("echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	world:WriteLine("inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	world:WriteLine("broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))

	if charge_inuse then
		world:WriteLine("charge socket " .. charge_addr .. ":" .. nx_string(charge_port))
	end

	if gmcc_inuse then
		world:WriteLine("gmcc socket " .. gmcc_addr .. ":" .. nx_string(gmcc_port))
	end

	if manage_inuse then
		world:WriteLine("manage socket " .. manage_addr .. ":" .. nx_string(manage_port))
	end

	if monitor_inuse then
	    world:SetExtraAddress(0, monitor_addr, monitor_port)
	end

	if dumpserver_inuse then
	    world:SetExtraAddress(2, dumpserver_addr, dumpserver_port)
	end

	--启动完成
	world:WriteLine("game server " .. nx_string(server_id) .. " starting...")

	--设置进程开启状态
	world:SetAppState(WAPS_OPENED);

	return 1
end

--加载验证码字符图像
function load_char_image(world)
	world:LoadCharImage("0", nx_resource_path() .. "bmp\\0.bmp")
	world:LoadCharImage("1", nx_resource_path() .. "bmp\\1.bmp")
	world:LoadCharImage("2", nx_resource_path() .. "bmp\\2.bmp")
	world:LoadCharImage("3", nx_resource_path() .. "bmp\\3.bmp")
	world:LoadCharImage("4", nx_resource_path() .. "bmp\\4.bmp")
	world:LoadCharImage("5", nx_resource_path() .. "bmp\\5.bmp")
	world:LoadCharImage("6", nx_resource_path() .. "bmp\\6.bmp")
	world:LoadCharImage("7", nx_resource_path() .. "bmp\\7.bmp")
	world:LoadCharImage("8", nx_resource_path() .. "bmp\\8.bmp")
	world:LoadCharImage("9", nx_resource_path() .. "bmp\\9.bmp")
	world:LoadCharImage("A", nx_resource_path() .. "bmp\\a.bmp")
	world:LoadCharImage("B", nx_resource_path() .. "bmp\\b.bmp")
	world:LoadCharImage("C", nx_resource_path() .. "bmp\\c.bmp")
	world:LoadCharImage("D", nx_resource_path() .. "bmp\\d.bmp")
	world:LoadCharImage("E", nx_resource_path() .. "bmp\\e.bmp")
	world:LoadCharImage("F", nx_resource_path() .. "bmp\\f.bmp")
	world:LoadCharImage("G", nx_resource_path() .. "bmp\\g.bmp")
	world:LoadCharImage(",", nx_resource_path() .. "bmp\\comma.bmp")

	return true
end

--加载场景配置文件
function load_scene_config(world)
	local ini = nx_create("IniDocument")

	--场景配置文件名
	ini.FileName = nx_resource_path() .. "scenes.ini"

	if not ini:LoadFromFile() then
		nx_destroy(ini)
		return false;
	end

	local sect_table = ini:GetSectionList()

	for i = 1, table.getn(sect_table) do
		local sect = sect_table[i]

		--场景编号
		local scene_id = nx_number(sect)
		--所在场景服务器编号
		local member_id = ini:ReadInteger(sect, "Member", 0)
		--逻辑类名
		local script = ini:ReadString(sect, "Script", "scene")
		--场景配置名
		local config = ini:ReadString(sect, "Config", "")
		--是否是副本场景原型
		local clonable = nx_boolean(ini:ReadInteger(sect, "Clonable", 0))

		if clonable then
			--副本原型场景
			world:AddPrototypeScene(scene_id, member_id, script, config)
		else
			--普通场景
			world:AddScene(scene_id, member_id, script, config)
		end
	end

	nx_destroy(ini)

	return true
end

--关闭
function close()
	local world = nx_value("world")

	if world.State == "Closed" then
		force_close()
	else
		--进入关闭状态后自动退出程序
		world.QuitOnClosed = true
		--发出关闭服务器命令
		world:CloseServer()
		world:WriteLine("waiting server close...")
	end

	return true
end

--强制关闭
function force_close()
	local world = nx_value("world")

	--关闭游戏服务
	world:WriteLine("world server shutdown...")
	world:Stop()

	--结束程序
	nx_quit()

	return true
end

--维护连接脚本

--初始化
function echo_init(self)
	nx_callback(self, "on_open", "echo_open")
	nx_callback(self, "on_close", "echo_close")
	nx_callback(self, "on_input", "echo_input")

	nx_callback(self, "on_input_getp", "echo_input_getp")
	nx_callback(self, "on_input_setp", "echo_input_setp")
	nx_callback(self, "on_input_getc", "echo_input_getc")
	nx_callback(self, "on_input_setc", "echo_input_setc")
	nx_callback(self, "on_input_method", "echo_input_method")
	nx_callback(self, "on_input_listp", "echo_input_listp")
	nx_callback(self, "on_input_listc", "echo_input_listc")
	nx_callback(self, "on_input_listm", "echo_input_listm")
	nx_callback(self, "on_input_listg", "echo_input_listg")
	nx_callback(self, "on_input_function", "echo_input_function")
	nx_callback(self, "on_input_debugger", "echo_input_debugger")

	return 1
end

--维护连接打开
function echo_open(self, addr, port)
	self:WriteLine("echo client " .. nx_string(addr)
		.. ":" .. nx_string(port) .. " connected")
	self:SendEchoLine("welcome server " .. nx_string(self.ServerId) .. "\r")
	return 1
end

--维护连接关闭
function echo_close(self, addr, port)
	self:WriteLine("echo client " .. nx_string(addr)
		.. ":" .. nx_string(port) .. " closed")
	return 1
end

--处理维护输入命令
function echo_input(self, command)
	if command == "quit" then
		--退出程序
		echo_response(self, "server will shutdown...")
		close()
	elseif command == "fquit" then
		--强制退出
		echo_response(self, "server will shutdown...")
		force_close()
	elseif command == "end" then
		--关闭连接
		self:CloseEcho()
	elseif command == "players" then
		--返回在线人数
		echo_response(self, "player: " .. nx_string(self.PlayerCount)
			.. ", online: " .. nx_string(self.OnlineCount)
			.. ", offline: " .. nx_string(self.OfflineCount)
			.. ", active: " .. nx_string(self.ActiveCount)
			.. ", queuing: " .. nx_string(self.QueueCount))
	elseif command == "open" then
		--开启服务
		if self.State == "Opened" then
			echo_response(self, "game server already opened")
			return 0
		end

		if self.State ~= "CanOpen" then
			echo_response(self, "game server not ready, can't open")
			return 0
		end

		if self:OpenServer() then
			echo_response(self, "send open command to game server")
		end
	elseif command == "close" then
		--关闭服务
		if self.State ~= "Opened" then
			echo_response(self, "game server not open")
			return 0
		end

		if self:CloseServer() then
			echo_response(self, "send close command to game server")
		end
	elseif command == "status" then
		--当前服务状态
		echo_response(self, "game server state is " .. self.State)
	else
		echo_response(self, "undefined command: " .. command)
	end

	return 1
end

--发送回应信息
function echo_response(self, info)
	self:SendEchoLine(">" .. info .. "\r")
	self:WriteLine(info)
	return true
end

--查找实体
function get_entity(entity_name)
	local id = nx_value(entity_name)

	if id ~= nil then
		return id
	end

	id = nx_lookup(entity_name)

	if not nx_is_null(id) then
		return id
	end

	id = nx_object(entity_name)

	if nx_is_valid(id) then
		return id
	end

	return nil
end

--转换到合适的类型
function get_new_value(old_v, value)
	local type = nx_type(old_v)

	if type == "boolean" then
		return nx_boolean(value)
	elseif type == "number" then
		return nx_number(value)
	elseif type == "string" then
		return nx_string(value)
	elseif type == "widestr" then
		return nx_widestr(value)
	elseif type == "int" then
		return nx_int(value)
	elseif type == "int64" then
		return nx_int64(value)
	elseif type == "float" then
		return nx_float(value)
	elseif type == "double" then
		return nx_double(value)
	elseif type == "object" then
		return nx_object(value)
	end

	return nil
end

--查询实体内部属性
function echo_input_getp(self, entity, prop)
	if entity == nil or prop == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local value = nx_property(obj, nx_string(prop))

	echo_response(self, "entity " .. nx_name(obj) .. " property "
		.. nx_string(prop) .. " value is " .. nx_string(value))

	return 1
end

--设置实体内部属性
function echo_input_setp(self, entity, prop, value)
	if entity == nil or prop == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local old_v = nx_property(obj, nx_string(prop))

	local new_v = get_new_value(old_v, value)

	if not nx_set_property(obj, nx_string(prop), new_v) then
		echo_response(self, "set property failed")
		return 0
	end

	new_v = nx_property(obj, nx_string(prop))

	echo_response(self, "set entity " .. nx_name(obj) .. " property "
		.. nx_string(prop) .. " value " .. nx_string(new_v))

	return 1
end

--查询实体自定义属性
function echo_input_getc(self, entity, prop)
	if entity == nil or prop == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local value = nx_custom(obj, nx_string(prop))

	echo_response(self, "entity " .. nx_name(obj) .. " custom "
		.. nx_string(prop) .. " value is " .. nx_string(value))

	return 1
end

--设置实体自定义属性
function echo_input_setc(self, entity, prop, value)
	if entity == nil or prop == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local old_v = nx_custom(obj, nx_string(prop))

	local new_v = get_new_value(old_v, value)

	if not nx_set_custom(obj, nx_string(prop), new_v) then
		echo_response(self, "set custom failed")
		return 0
	end

	new_v = nx_custom(obj, nx_string(prop))

	echo_response(self, "set entity " .. nx_name(obj) .. " custom "
		.. nx_string(prop) .. " value " .. nx_string(new_v))

	return 1
end

--执行实体方法
function echo_input_method(self, entity, method, ...)
	if entity == nil or method == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local res = nx_method(obj, nx_string(method), unpack(arg))

	if type(res) == "table" then
		echo_response(self, "run entity " .. nx_name(obj) .. " method "
			.. nx_string(method) .. " return table")

		for i = 1, table.getn(res) do
			echo_response(self, "table value " .. nx_string(i) .. ": " .. nx_string(res[i]))
		end
	else
		echo_response(self, "run entity " .. nx_name(obj) .. " method "
			.. nx_string(method) .. " return " .. nx_string(res))
	end

	return 1
end

--列出实体的所有内部属性
function echo_input_listp(self, entity)
	if entity == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local prop_table = nx_property_list(obj)

	local num = table.getn(prop_table)

	echo_response(self, "entity " .. nx_name(obj) .. " property number is " .. nx_string(num))

	for i = 1, num do
		echo_response(self, nx_string(prop_table[i]))
	end

	return 1
end

--列出实体的所有自定义属性
function echo_input_listc(self, entity)
	if entity == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local prop_table = nx_custom_list(obj)

	local num = table.getn(prop_table)

	echo_response(self, "entity " .. nx_name(obj) .. " custom number is " .. nx_string(num))

	for i = 1, num do
		echo_response(self, nx_string(prop_table[i]))
	end

	return 1
end

--列出实体的所有方法
function echo_input_listm(self, entity)
	if entity == nil then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local prop_table = nx_method_list(obj)

	local num = table.getn(prop_table)

	echo_response(self, "entity " .. nx_name(obj) .. " method number is " .. nx_string(num))

	for i = 1, num do
		echo_response(self, nx_string(prop_table[i]))
	end

	return 1
end

--列出所有全局变量
function echo_input_listg(self)
	local global_table = nx_value_list()

	local num = table.getn(global_table)

	echo_response(self, "global number is " .. nx_string(num))

	for i = 1, num do
		local name = global_table[i]
		local value = nx_value(name)

		echo_response(self, nx_string(name) .. "[" .. nx_type(value) .. "]: " .. nx_string(value))
	end

	return 1
end

--执行脚本扩展函数
function echo_input_function(self, func, ...)
	if func == nil or func == "" then
		return 0
	end

	local res = nx_function(func, unpack(arg))

	if type(res) == "table" then
		echo_response(self, "run function " .. nx_string(func) .. " return table")

		for i = 1, table.getn(res) do
			echo_response(self, "result" .. nx_string(i) .. ": " .. nx_string(res[i]))
		end
	else
		echo_response(self, "run function " .. nx_string(func) .. " return " .. nx_string(res))
	end

	return 1
end

--调试函数
function echo_input_debugger(self, entity, method, ...)
	if entity == nil or entity == "" or method == nil or method == "" then
		return 0
	end

	local obj = get_entity(nx_string(entity))

	if obj == nil then
		return 0
	end

	local res = nx_method(obj, nx_string(method), unpack(arg))

	if type(res) == "table" then
		self:SendEchoLine(">" .. "run entity " .. nx_string(entity) .. " method " .. nx_string(method) .. " return table" .. "\r")

		for i = 1, table.getn(res) do
			self:SendEchoLine(">" .. "result" .. nx_string(i) .. ": " .. nx_string(res[i]) .. "\r")
		end
	else
		self:SendEchoLine(">" .. "run entity " .. nx_string(entity) .. " method " .. nx_string(method) .. " return " .. nx_string(res) .. "\r")
	end

	return 1
end
