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
	console.Color = "White"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\unite"
	--缓冲数量
	log_pool.TraceMax = 256
	--是否每天一个日志文件
	log_pool.EveryDayAlone = false
	
	if nx_is_debug() then
		--在调试版里将日志信息输出到控制台
		log_pool.Console = console
	end
	
	nx_set_value("log_pool", log_pool)
	
	--启动日志记录
	log_pool:Start()
	
	--读取程序配置
	local ini = nx_create("IniDocument")
	
	--配置文件名
	ini.FileName = "GameUnite.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("can't load configure file")
		nx_destroy(ini)
		return 0
	end
	
	--游戏分区编号
	local district_id = ini:ReadInteger("Main", "DistrictID", 1)
	--服务器编号
	local server_id = ini:ReadInteger("Main", "ServerID", 1)
	--服务器名称
	local server_name = ini:ReadString("Main", "ServerName", "")
	--公共数据逻辑DLL文件
	local public_logic_dll = ini:ReadString("Main", "PublicLogicDll", "")
	--最大线人数
	local max_players = ini:ReadInteger("Main", "MaxPlayers", 8192)
	--场景碰撞信息文件的根目录
	local collide_root_path = ini:ReadString("Main", "CollideRootPath", "")
	--使用MYSQL数据库
	local use_mysql = nx_boolean(ini:ReadString("Main", "UseMysql", "false"))
	--通讯加密键
	local msg_encode_key = ini:ReadString("Main", "MessageEncodeKey", "")
	--通讯校验DLL
	local msg_verify_dll = ini:ReadString("Main", "MessageVerifyDll", "")
	
	if collide_root_path ~= "" then
		collide_root_path = nx_function("ext_get_full_path", collide_root_path)
	end
	
	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 16000
	
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
	
	--数据库连接
	local account_base = ini:ReadString("Account", "ConnectString", "")
	local account_pool = ini:ReadInteger("Account", "ConnectPool", 0)
	
	local role_base = ini:ReadString("Role", "ConnectString", "")
	local role_pool = ini:ReadInteger("Role", "ConnectPool", 0)
	
	local log_base = ini:ReadString("Log", "ConnectString", "")
	local log_pool = ini:ReadInteger("Log", "ConnectPool", 0)
	
	local public_base = ini:ReadString("Public", "ConnectString", "")
	local public_pool = ini:ReadInteger("Public", "ConnectPool", 0)
	
	local manage_base = ini:ReadString("Manage", "ConnectString", "")
	local manage_pool = ini:ReadInteger("Manage", "ConnectPool", 0)
	
	--外网地址
	local outer_addr = ini:ReadString("Outer", "Addr", "127.0.0.1")
	local outer_port = ini:ReadInteger("Outer", "Port", 2001)
	
	--授权文件名
	local license = outer_addr .. ".lic"
	
	--逻辑模块
	local logic_loader = nx_create("LogicLoader")
	
	local logic_path = ini:ReadString("Logic", "Path", "")
	local modules = ini:ReadInteger("Logic", "Modules", 0)
	
	if logic_path ~= "" and modules == 0 then
		local fs = nx_create("FileSearch")
		
		fs:SearchFile(nx_work_path() .. logic_path .. "\\", "*.dll")
		
		local dll_table = fs:GetFileList()
		
		for i = 1, table.getn(dll_table) do
			if logic_loader:Load(nx_work_path() .. logic_path .. "\\", dll_table[i]) then
				console:WriteLine("Load loigc module " .. dll_table[i])
			else
				console:WriteLine("Load loigc module " .. dll_table[i] .. " failed")
			end
		end
		
		nx_destroy(fs)
	else
		for i = 0, modules do
			local dll_name = ini:ReadString("Logic", "Module" .. nx_string(i), "")
			
			if dll_name ~= "" then
				if logic_loader:Load(nx_work_path() .. logic_path, dll_name) then
					console:WriteLine("Load loigc module " .. dll_name)
				else
					console:WriteLine("Load loigc module " .. dll_name .. " failed")
				end
			end
		end
	end
	
	nx_destroy(ini)
	
	--公共数据逻辑模块
	local public_logic_loader = nx_create("LogicLoader")
	
	if public_logic_dll ~= "" then
		if public_logic_loader:Load(nx_work_path(), public_logic_dll) then
			console:WriteLine("Load loigc module " .. public_logic_dll)
		else
			console:WriteLine("Load loigc module " .. public_logic_dll .. " failed")
		end
	end
	
	--场景碰撞信息
	local collide_manager = nx_create("CollideManager")
	
	--创建维护连接服务
	local echo_sock = nx_create("EchoSock")
	
	echo_sock.Address = echo_addr
	echo_sock.Port = echo_port
	
	nx_bind_script(echo_sock, "echo", "echo_init")
	
	--创建广播通讯
	local world_broad_sock = nx_create("BroadSock")
	
	world_broad_sock.BindAddress = inner_addr
	world_broad_sock.BroadcastAddress = broad_addr
	world_broad_sock.BroadcastPort = broad_port
	
	local store_broad_sock = nx_create("BroadSock")
	
	store_broad_sock.BindAddress = inner_addr
	store_broad_sock.BroadcastAddress = broad_addr
	store_broad_sock.BroadcastPort = broad_port
	
	local public_broad_sock = nx_create("BroadSock")
	
	public_broad_sock.BindAddress = inner_addr
	public_broad_sock.BroadcastAddress = broad_addr
	public_broad_sock.BroadcastPort = broad_port
	
	local entry_broad_sock = nx_create("BroadSock")
	
	entry_broad_sock.BindAddress = inner_addr
	entry_broad_sock.BroadcastAddress = broad_addr
	entry_broad_sock.BroadcastPort = broad_port
	
	local member_broad_sock = nx_create("BroadSock")
	
	member_broad_sock.BindAddress = inner_addr
	member_broad_sock.BroadcastAddress = broad_addr
	member_broad_sock.BroadcastPort = broad_port
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "unite_" .. nx_string(server_id)
	app_manager.AppAddress = echo_addr .. ":" .. nx_string(echo_port)
	
	--创建客户端接入通讯服务
	local mass_sock = nx_create("MassSock")
	
	mass_sock.Address = outer_addr
	mass_sock.Port = outer_port
	mass_sock.ConnectMax = 8 * 1024
	mass_sock.WorkThreads = 1
	mass_sock.InBufLen = 16 * 1024
	mass_sock.OutBufLen = 64 * 1024
	mass_sock.ListenPendings = 20
	
	--创建内部通讯
	local world_adapter = nx_create("Adapter")
	
	--内部通讯地址
	world_adapter.Address = inner_addr
	world_adapter.Port = inner_port + 1
	--最大内网连接数量
	world_adapter.ConnectMax = 64
	--工作线程数量
	world_adapter.WorkThreads = 1
	--缓冲区尺寸
	world_adapter.InBufLen = 1024 * 1024
	world_adapter.OutBufLen = 1024 * 1024
	
	local public_adapter = nx_create("Adapter")
	
	--内部通讯地址
	public_adapter.Address = inner_addr
	public_adapter.Port = inner_port + 2
	--最大内网连接数量
	public_adapter.ConnectMax = 64
	--工作线程数量
	public_adapter.WorkThreads = 1
	--缓冲区尺寸
	public_adapter.InBufLen = 1024 * 1024
	public_adapter.OutBufLen = 1024 * 1024
	
	local store_adapter = nx_create("Adapter")
	
	--内部通讯地址
	store_adapter.Address = inner_addr
	store_adapter.Port = inner_port + 100
	--最大内网连接数量
	store_adapter.ConnectMax = 64
	--工作线程数量
	store_adapter.WorkThreads = 1
	--缓冲区尺寸
	store_adapter.InBufLen = 1024 * 1024
	store_adapter.OutBufLen = 1024 * 1024
	
	local entry_adapter = nx_create("Adapter")
	
	--内部通讯地址
	entry_adapter.Address = inner_addr
	entry_adapter.Port = inner_port + 200
	--最大内网连接数量
	entry_adapter.ConnectMax = 64
	--工作线程数量
	entry_adapter.WorkThreads = 1
	--缓冲区尺寸
	entry_adapter.InBufLen = 1024 * 1024
	entry_adapter.OutBufLen = 1024 * 1024
	
	local member_adapter = nx_create("Adapter")
	
	--内部通讯地址
	member_adapter.Address = inner_addr
	member_adapter.Port = inner_port + 300
	--最大内网连接数量
	member_adapter.ConnectMax = 64
	--工作线程数量
	member_adapter.WorkThreads = 1
	--缓冲区尺寸
	member_adapter.InBufLen = 1024 * 1024
	member_adapter.OutBufLen = 1024 * 1024
	
	--创建数据库系统
	local db_system
	
	if use_mysql then
		db_system = nx_create("MysqlDBSystem")
	else
		db_system = nx_create("SqlDBSystem")
	end
	
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
	game_world.BroadSock = world_broad_sock
	game_world.AppManager = app_manager
	game_world.Adapter = world_adapter
	
	--加载名字过滤文件
	game_world:LoadNameFilter(nx_resource_path() .. "filter.txt")
	
	--添加普通场景
	load_scene_config()
	
	--创建存储服务
	local game_store = nx_create("GameStore")
	
	nx_set_value("game_store", game_store)
	
	game_store.DistrictId = district_id
	game_store.ServerId = server_id
	game_store.StoreId = 0
	game_store.AccountBase = account_base
	game_store.AccountPools = account_pool
	game_store.RoleBase = role_base
	game_store.RolePools = role_pool
	game_store.LogBase = log_base
	game_store.LogPools = log_pool
	game_store.PublicBase = public_base
	game_store.PublicPools = public_pool
	game_store.ManageBase = manage_base
	game_store.ManagePools = manage_pool
	game_store.Console = console
	game_store.BroadSock = store_broad_sock
	game_store.Adapter = store_adapter
	game_store.DBSystem = db_system
	
	init_custom_log(game_store)
	
	--创建公共数据服务
	local game_public = nx_create("GamePublic")
	
	nx_set_value("game_public", game_public)
	
	game_public.DistrictId = district_id
	game_public.ServerId = server_id
	game_public.Console = console
	game_public.BroadSock = public_broad_sock
	game_public.Adapter = public_adapter
	game_public.LogicLoader = public_logic_loader
	
	--添加公共数据空间
	game_public:AddPubSpace("pubdata", "nx_pubdata")
	game_public:AddPubSpace("guild", "nx_guilds")
	
	--创建接入服务
	local game_entry = nx_create("GameEntry")
	
	nx_set_value("game_entry", game_entry)
	
	game_entry.IDC = 0
	game_entry.DistrictId = district_id
	game_entry.ServerId = server_id
	game_entry.EntryId = 0
	game_entry.MaxConnects = 1024
	game_entry.License = license
	game_entry.EncodeKey = game_entry:ConvertEncodeKey(msg_encode_key)
	game_entry.DecodeKey = game_entry:ConvertEncodeKey(msg_encode_key)
	game_entry.VerifyDll = msg_verify_dll
	game_entry.Console = console
	game_entry.MassSock = mass_sock
	game_entry.BroadSock = entry_broad_sock
	game_entry.Adapter = entry_adapter
	--记录与客户端的收发消息到message.log文件
	--game_entry.LogConnect = true
	--game_entry.LogSend = true
	--game_entry.LogReceive = true
	
	--创建场景服务
	local game_member = nx_create("GameMember")
	
	nx_set_value("game_member", game_member)
	
	game_member.DistrictId = district_id
	game_member.ServerId = server_id
	game_member.MemberId = 0
	--每个玩家使用单独的消息缓冲区
	game_member.PlayerSingleBuffer = true
	--开启视野消息优化
	game_member.SightOptimize = true
	game_member.Console = console
	game_member.BroadSock = member_broad_sock
	game_member.Adapter = member_adapter
	game_member.LogicLoader = logic_loader
	game_member.CollideManager = collide_manager
	game_member.ScriptPath = nx_resource_path() .. "lua\\"
	game_member.TaskPath = nx_resource_path() .. "task\\"
	game_member.CollideRootPath = collide_root_path
	
	--加载名字过滤文件
	game_member:LoadNameFilter(nx_resource_path() .. "filter.txt")
	
	--加载全局点信息
	game_member:LoadLocations(nx_resource_path() .. "location.ini")
	
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
	world_adapter:Start()
	store_adapter:Start()
	public_adapter:Start()
	entry_adapter:Start()
	member_adapter:Start()
	
	--启动广播通讯
	world_broad_sock:Start()
	store_broad_sock:Start()
	public_broad_sock:Start()
	entry_broad_sock:Start()
	member_broad_sock:Start()
	
	--暂不接受连接
	mass_sock.Closed = true
	--启动客户端连接
	mass_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动游戏服务
	game_world:Start()
	
	--启动存储服务
	game_store:Start()
	
	--启动公共数据服务
	game_public:Start()
	
	--启动接入服务
	game_entry:Start()
	
	--启动场景服务
	game_member:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	console:WriteLine("Broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	console:WriteLine("Outer socket " .. outer_addr .. ":" .. nx_string(outer_port))
	
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

--设置自定义的游戏日志数据库表格结构
function init_custom_log(game_store)
	--物品日志表
	--物品日志表
	game_store:AddLogTable("item_log")
	game_store:AddLogField("item_log", "time", "datetime", 8)
	game_store:AddLogField("item_log", "logtype", "int", 4)
	game_store:AddLogField("item_log", "itemid", "varchar", 32)
	game_store:AddLogField("item_log", "itemuid", "varchar", 32)
	game_store:AddLogField("item_log", "amount", "int", 4)
	game_store:AddLogField("item_log", "itemto", "nvarchar", 64)
	game_store:AddLogField("item_log", "itemfrom", "nvarchar", 64)
	game_store:AddLogField("item_log", "comment1", "nvarchar", 255)
	game_store:AddLogField("item_log", "comment2", "nvarchar", 255)
	game_store:AddLogField("item_log", "comment3", "nvarchar", 255)
	game_store:AddLogField("item_log", "ip", "varchar", 16)
	game_store:AddLogField("item_log", "mac", "varchar", 16)
	
	--玩家交易表
	game_store:AddLogTable("deal_log")
	game_store:AddLogField("deal_log", "tradeid", "varchar", 50)
	game_store:AddLogField("deal_log", "time", "datetime", 8)
	game_store:AddLogField("deal_log", "buyer", "nvarchar", 64)
	game_store:AddLogField("deal_log", "seller", "nvarchar", 64)
	game_store:AddLogField("deal_log", "itemid", "varchar", 32)
	game_store:AddLogField("deal_log", "itemuid", "varchar", 32)
	game_store:AddLogField("deal_log", "amount", "int", 4)
	game_store:AddLogField("deal_log", "tradechannel", "int", 4)
	game_store:AddLogField("deal_log", "ip_get", "varchar", 16)
	game_store:AddLogField("deal_log", "ip_lost", "varchar", 16)
	game_store:AddLogField("deal_log", "mac_get", "varchar", 16)
	game_store:AddLogField("deal_log", "mac_lost", "varchar", 16)
	
	--玩家登陆日志表
	game_store:AddLogTable("role_log")
	game_store:AddLogField("role_log", "account", "varchar", 100)
	game_store:AddLogField("role_log", "role", "nvarchar", 64)
	game_store:AddLogField("role_log", "logintime", "datetime", 8)
	game_store:AddLogField("role_log", "logouttime", "datetime", 8)
	game_store:AddLogField("role_log", "ip", "varchar", 16)
	game_store:AddLogField("role_log", "mac", "varchar", 16)
	
	--玩家升级日志表
	game_store:AddLogTable("role_upgrade_log")
	game_store:AddLogField("role_upgrade_log", "time", "datetime", 8)
	game_store:AddLogField("role_upgrade_log", "role", "nvarchar", 64)
	game_store:AddLogField("role_upgrade_log", "logtype", "int", 4)
	game_store:AddLogField("role_upgrade_log", "level_before", "int", 4)
	game_store:AddLogField("role_upgrade_log", "level_after", "int", 4)
	game_store:AddLogField("role_upgrade_log", "skill_id", "varchar", 32)
	game_store:AddLogField("role_upgrade_log", "ip", "varchar", 16)
	game_store:AddLogField("role_upgrade_log", "mac", "varchar", 16)
	
	--任务日志表
	game_store:AddLogTable("task_log")
	game_store:AddLogField("task_log", "time", "datetime", 8)
	game_store:AddLogField("task_log", "role", "nvarchar", 64)
	game_store:AddLogField("task_log", "task", "int", 4)
	game_store:AddLogField("task_log", "logtype", "int", 4)
    game_store:AddLogField("task_log", "ip", "varchar", 16)
	game_store:AddLogField("task_log", "mac", "varchar", 16)
	
	--副本日志表
	game_store:AddLogTable("instance_log")
	game_store:AddLogField("instance_log", "cloneid", "int", 4)
	game_store:AddLogField("instance_log", "prototype", "int", 4)
	game_store:AddLogField("instance_log", "starttime", "datetime", 8)
	game_store:AddLogField("instance_log", "endtime", "datetime", 8)
	game_store:AddLogField("instance_log", "duration", "int", 4)
	game_store:AddLogField("instance_log", "troop_count1", "int", 4)
	game_store:AddLogField("instance_log", "troop_count2", "int", 4)
	
	game_store:AddLogTable("instance_role_log")
	game_store:AddLogField("instance_role_log", "cloneid", "int", 4)
	game_store:AddLogField("instance_role_log", "prototype", "int", 4)
	game_store:AddLogField("instance_role_log", "entertime", "datetime", 8)
	game_store:AddLogField("instance_role_log", "leavetime", "datetime", 8)
	game_store:AddLogField("instance_role_log", "duration", "int", 4)
	game_store:AddLogField("instance_role_log", "role", "nvarchar", 64)
	game_store:AddLogField("instance_role_log", "ip", "varchar", 16)
	game_store:AddLogField("instance_role_log", "mac", "varchar", 16)

	game_store:AddLogTable("instance_npc_log")
	game_store:AddLogField("instance_npc_log", "time", "datetime", 8)
	game_store:AddLogField("instance_npc_log", "npcname",  "nvarchar", 64)
	game_store:AddLogField("instance_npc_log", "sceneid", "int", 4)
    game_store:AddLogField("instance_npc_log", "location", "varchar", 32)
	game_store:AddLogField("instance_npc_log", "role", "nvarchar", 64)
    game_store:AddLogField("instance_npc_log", "ip", "varchar", 16)
	game_store:AddLogField("instance_npc_log", "mac", "varchar", 16)

	--场景日志表
	game_store:AddLogTable("scene_log")
	game_store:AddLogField("scene_log", "entertime", "datetime", 8)
	game_store:AddLogField("scene_log", "leavetime", "datetime", 8)
	game_store:AddLogField("scene_log", "duration", "int", 4)
	game_store:AddLogField("scene_log", "role", "nvarchar", 64)
	game_store:AddLogField("scene_log", "sceneid", "int", 4)
	game_store:AddLogField("scene_log", "ip", "varchar", 16)
	game_store:AddLogField("scene_log", "mac", "varchar", 16)

	--惩罚日志表
	game_store:AddLogTable("punish_log")
	game_store:AddLogField("punish_log", "time", "datetime", 8)
	game_store:AddLogField("punish_log", "operator", "varchar", 100)
	game_store:AddLogField("punish_log", "role", "nvarchar", 64)
	game_store:AddLogField("punish_log", "minutes", "int", 4)
	game_store:AddLogField("punish_log", "type", "int", 4)
	
	
	--活动日志表
	game_store:AddLogTable("program_log")
	game_store:AddLogField("program_log", "programid", "int", 4)
	game_store:AddLogField("program_log", "programname", "varchar", 64)
	game_store:AddLogField("program_log", "opentime", "datetime", 8)
	game_store:AddLogField("program_log", "endtime", "datetime", 8)
	
	return true
end

--关闭
function close()
	local console = nx_value("console")
	
	--游戏服务
	local game_world = nx_value("game_world")
	
	local app_manager = game_world.AppManager
	local world_broad_sock = game_world.BroadSock
	local echo_sock = game_world.EchoSock
	local world_adapter = game_world.Adapter
	local charge_sock = game_world.ChargeSock
	local gmcc_sock = game_world.GmccSock
	local stat_sock = game_world.StatSock
	
	--存储服务
	local game_store = nx_value("game_store")
	
	local store_broad_sock = game_store.BroadSock
	local store_adapter = game_store.Adapter
	
	--公共数据服务
	local game_public = nx_value("game_public")
	
	local public_broad_sock = game_public.BroadSock
	local public_adapter = game_public.Adapter
	
	--接入服务
	local game_entry = nx_value("game_entry")
	
	local entry_mass_sock = game_entry.MassSock
	local entry_broad_sock = game_entry.BroadSock
	local entry_adapter = game_entry.Adapter
	
	--场景服务
	local game_member = nx_value("game_member")
	
	local member_broad_sock = game_member.BroadSock
	local member_adapter = game_member.Adapter
	
	if game_world.State ~= "Closed" then
		--发出关闭服务器命令
		game_world:CloseServer()
		
		console:WriteLine("waiting server close...")
		
		--等待服务器关闭
		while game_world.State ~= "Closed" do
			nx_pause(0.0)
		end
		
		console:WriteLine("server close finished")
	end
	
	--关闭外网连接
	console:WriteLine("client socket shutdown...")
	
	entry_mass_sock.Closed = true
	entry_mass_sock:CloseAll()
	entry_mass_sock:Stop()
	
	--while entry_mass_sock.ConnectCount > 0 do
	--	nx_pause(0.0)
	--end
	
	--entry_mass_sock:Stop()
	
	--等待所有玩家退出游戏
	--console:WriteLine("waiting players exit...")
	
	--while game_world.PlayerCount > 0 do
	--	nx_pause(0.0)
	--end
	
	--等待所有数据库操作完成
	--console:WriteLine("waiting store finished...")
	
	--while not game_store:AccessEmpty() do
	--	nx_pause(0.0)
	--end
	
	--关闭广播
	console:WriteLine("broad shutdown...")
	
	world_broad_sock:Stop()
	store_broad_sock:Stop()
	public_broad_sock:Stop()
	entry_broad_sock:Stop()
	member_broad_sock:Stop()
	
	--关闭内网通讯
	console:WriteLine("adapter shutdown...")
	
	world_adapter:Stop()
	store_adapter:Stop()
	public_adapter:Stop()
	entry_adapter:Stop()
	member_adapter:Stop()
	
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
	
	--关闭存储服务
	console:WriteLine("store server shutdown...")
	
	game_store:Stop()
	
	--关闭公共数据服务
	console:WriteLine("public server shutdown...")
	
	game_public:Stop()
	
	--关闭接入服务
	console:WriteLine("entry server shutdown...")
	
	game_entry:Stop()
	
	--关闭场景服务
	console:WriteLine("member server shutdown...")
	
	game_member:Stop()
	
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

