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
function main(console)
	nx_set_value("console", console)
	
	--屏幕输出字符颜色
	console.Color = "LightGreen"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\world"
	--缓冲数量
	log_pool.TraceMax = 256
	--是否每天一个日志文件
	log_pool.EveryDayAlone = false
	
	nx_set_value("log_pool", log_pool)
	
	if nx_is_debug() then
		--在调试版里将日志信息输出到控制台
		log_pool.Console = console
	end
	
	--启动日志记录
	log_pool:Start()
	
	--读取程序配置
	local ini = nx_create("IniDocument")
	
	--配置文件名
	ini.FileName = "GameWorld.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
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
	
	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 16001
	
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
	
	--统计服务器
	local stat_inuse = nx_boolean(ini:ReadString("Stat", "Inuse", ""))
	local stat_addr = ini:ReadString("Stat", "Addr", "")
	local stat_port = ini:ReadInteger("Stat", "Port", 0)
	
	--管理服务器
	local manage_inuse = nx_boolean(ini:ReadString("ManageServer", "Inuse", ""))
	local manage_addr = ini:ReadString("ManageServer", "Addr", "")
	local manage_port = ini:ReadInteger("ManageServer", "Port", 0)
	
	nx_destroy(ini)
	
	--创建维护连接服务
	local echo_sock = nx_create("EchoSock")
	
	echo_sock.Address = echo_addr
	echo_sock.Port = echo_port
	
	nx_bind_script(echo_sock, "echo", "echo_init")
	
	--创建广播通讯
	local broad_sock = nx_create("BroadSock")
	
	broad_sock.BindAddress = inner_addr
	broad_sock.BroadcastAddress = broad_addr
	broad_sock.BroadcastPort = broad_port
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "world_" .. nx_string(server_id)
	app_manager.AppAddress = echo_addr .. ":" .. nx_string(echo_port)
	
	--创建内部通讯
	local adapter = nx_create("Adapter")
	
	--内部通讯地址
	adapter.Address = inner_addr
	adapter.Port = inner_port
	--最大内网连接数量
	adapter.ConnectMax = 64
	--工作线程数量
	adapter.WorkThreads = 1
	--缓冲区尺寸
	adapter.InBufLen = 1024 * 1024
	adapter.OutBufLen = 1024 * 1024
	
	--创建游戏服务
	local game_world = nx_create("GameWorld")
	
	nx_set_value("game_world", game_world)
	
	game_world.DistrictId = district_id
	game_world.ServerId = server_id
	game_world.ServerName = nx_widestr(server_name)
	game_world.MaxPlayers = max_players
	--每个帐号可以创建的角色数量
	game_world.AccountRoles = 1
	game_world.Console = console
	game_world.EchoSock = echo_sock
	game_world.BroadSock = broad_sock
	game_world.AppManager = app_manager
	game_world.Adapter = adapter
	
	--加载名字过滤文件
	game_world:LoadNameFilter(nx_resource_path() .. "filter.txt")
	
	--添加普通场景
	load_scene_config()
	
	--计费连接
	if charge_inuse then
		local charge_sock = nx_create("UserSock")
		
		charge_sock.Address = charge_addr
		charge_sock.Port = charge_port
		charge_sock.Async = true
		
		game_world.ChargeSock = charge_sock
		
		charge_sock:Start()
	end
	
	--客服连接
	if gmcc_inuse then
		local gmcc_sock = nx_create("UserSock")
		
		gmcc_sock.Address = gmcc_addr
		gmcc_sock.Port = gmcc_port
		gmcc_sock.Async = true
		
		game_world.GmccSock = gmcc_sock
		
		gmcc_sock:Start()
	end
	
	--统计连接
	if stat_inuse then
		local stat_sock = nx_create("UserSock")
		
		stat_sock.Address = stat_addr
		stat_sock.Port = stat_port
		stat_sock.Async = true
		
		game_world.StatSock = stat_sock
		
		stat_sock:Start()
	end
	
	--管理连接
	if manage_inuse then
		local manage_sock = nx_create("MassSock")
		
		manage_sock.Address = manage_addr
		manage_sock.Port = manage_port
		manage_sock.ConnectMax = 2
		manage_sock.WorkThreads = 1
		manage_sock.InBufLen = 4 * 1024
		manage_sock.OutBufLen = 4 * 1024
		manage_sock.ListenPendings = 2
		
		game_world.ManageSock = manage_sock
		
		manage_sock:Start()
	end
	
	--启动进程监控
	app_manager:Start()
	
	--设置进程启动状态
	app_manager:SetAppState(WAPS_STARTING)
	
	--启动内部通讯
	adapter:Start()
	
	--启动广播通讯
	broad_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动游戏服务
	game_world:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	console:WriteLine("Broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	
	if manage_inuse then
		console:WriteLine("Manage socket " .. manage_addr .. ":" .. nx_string(manage_port))
	end
	
	--启动完成
	console:WriteLine("Game server " .. nx_string(server_id) .. " starting...")
	
	--设置进程开启状态
	app_manager:SetAppState(WAPS_OPENED);
	
	return 1
end

--加载场景配置文件
function load_scene_config()
	local game_world = nx_value("game_world")
	
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
		local clonable = nx_boolean(ini:ReadString(sect, "Clonable", ""))
		
		if clonable then
			--副本原型场景
			game_world:AddPrototypeScene(scene_id, member_id, script, config)
		else
			--普通场景
			game_world:AddScene(scene_id, member_id, script, config)
		end
	end
	
	nx_destroy(ini)
	
	return true
end

--加载场景配置文件
--function load_scene_config()
--	local game_world = nx_value("game_world")
--	
--	local ini = nx_create("IniDocument")
--	
--	ini.FileName = nx_resource_path() .. "scenes.ini"
--	
--	if not ini:LoadFromFile() then
--		nx_destroy(ini)
--		return false;
--	end
--	
--	local sect_table = ini:GetSectionList()
--	
--	for i = 1, table.getn(sect_table) do
--		local sect = sect_table[i]
--		
--		local scene_id = nx_number(sect)
--		local member_id = ini:ReadInteger(sect, "Member", 0)
--		local script = ini:ReadString(sect, "Script", "scene")
--		local config = ini:ReadString(sect, "Config", "")
--		local clonable = nx_boolean(ini:ReadString(sect, "Clonable", ""))
--		
--		game_world:AddScene(scene_id, member_id, script, config)
--	end
--	
--	nx_destroy(ini)
--	
--	return true
--end

--关闭
function close()
	local console = nx_value("console")
	
	local game_world = nx_value("game_world")
	
	local app_manager = game_world.AppManager
	local broad_sock = game_world.BroadSock
	local echo_sock = game_world.EchoSock
	local adapter = game_world.Adapter
	local charge_sock = game_world.ChargeSock
	local gmcc_sock = game_world.GmccSock
	local stat_sock = game_world.StatSock
	
	--关闭广播
	console:WriteLine("broad shutdown...")
	
	broad_sock:Stop()
	
	--关闭内网通讯
	console:WriteLine("adapter shutdown...")
	
	adapter:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭游戏服务
	console:WriteLine("world server shutdown...")
	
	game_world:Stop()
	
	if nx_is_valid(charge_sock) then
		charge_sock:Stop()
	end
	
	if nx_is_valid(gmcc_sock) then
		gmcc_sock:Stop()
	end
	
	if nx_is_valid(stat_sock) then
		stat_sock:Stop()
	end
	
	--关闭进程监控
	console:WriteLine("app_manager shutdown...")
	
	app_manager:Stop()
	
	--关闭日志缓冲
	local log_pool = nx_value("log_pool")
	
	log_pool:Stop()
	
	--结束程序
	nx_quit()
	
	return true
end

