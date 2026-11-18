--存储服务器主控脚本

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
	console.Color = "LightYellow"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\store"
	--缓冲数量
	log_pool.TraceMax = 256
	--是否每天一个日志文件
	log_pool.EveryDayAlone = false
	
	if nx_is_debug() then
		--在调试版里将日志信息输出到控制台
		log_pool.Console = console
	end
	
	--启动日志记录
	log_pool:Start()
	
	nx_set_value("log_pool", log_pool)
	
	--读取程序配置
	local ini = nx_create("IniDocument")
	
	--配置文件名
	ini.FileName = "GameStore.ini"
	
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
	--组内存储编号
	local store_id = ini:ReadInteger("Main", "StoreID", 0)
	--使用MYSQL数据库
	local use_mysql = nx_boolean(ini:ReadString("Main", "UseMysql", "false"))
	
	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 16100 + store_id
	
	inner_addr = ini:ReadString("Inner", "Addr", inner_addr)
	inner_port = ini:ReadInteger("Inner", "Port", inner_port)
	
	--广播地址
	local broad_addr = "127.255.255.255"
	local broad_port = 15000
	
	broad_addr = ini:ReadString("Broadcast", "Addr", broad_addr)
	broad_port = ini:ReadInteger("Broadcast", "Port", broad_port)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15100 + store_id
	
	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
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
	
	app_manager.AppName = "store_" .. nx_string(server_id) .. "_" .. nx_string(store_id)
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
	
	--创建数据库系统
	local db_system
	
	if use_mysql then
		db_system = nx_create("MysqlDBSystem")
	else
		db_system = nx_create("SqlDBSystem")
	end
	
	--创建存储服务
	local game_store = nx_create("GameStore")
	
	nx_set_value("game_store", game_store)
	
	game_store.DistrictId = district_id
	game_store.ServerId = server_id
	game_store.StoreId = store_id
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
	game_store.EchoSock = echo_sock
	game_store.BroadSock = broad_sock
	game_store.AppManager = app_manager
	game_store.Adapter = adapter
	game_store.DBSystem = db_system
	
	init_custom_log(game_store)
	
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
	
	--启动存储服务
	game_store:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	console:WriteLine("Broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	
	--启动完成
	console:WriteLine("Game server " .. nx_string(server_id) .. " store "
		.. nx_string(store_id) .. " started")
	
	--设置进程开启状态
	app_manager:SetAppState(WAPS_OPENED);
	
	return 1
end

--设置自定义的游戏日志数据库表格结构
function init_custom_log(game_store)
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
	game_store:AddLogField("scene_log", "role", "nvarchar", 32)
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
	
	local game_store = nx_value("game_store")
	
	local app_manager = game_store.AppManager
	local broad_sock = game_store.BroadSock
	local echo_sock = game_store.EchoSock
	local adapter = game_store.Adapter
	
	--设置进程正在关闭状态
	app_manager:SetAppState(WAPS_CLOSING);
	
	--等待所有数据库操作完成
	console:WriteLine("waiting all access finish...")
	
	while not game_store:AccessEmpty() do
		nx_pause(0.0)
	end
	
	--关闭广播
	console:WriteLine("broad shutdown...")
	
	broad_sock:Stop()
	
	--关闭内网通讯
	console:WriteLine("adapter shutdown...")
	
	adapter:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭存储服务
	console:WriteLine("store server shutdown...")
	
	game_store:Stop()
	
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

