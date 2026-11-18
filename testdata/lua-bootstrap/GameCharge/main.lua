--监控服务器主控脚本

--入口函数
function main(console)
	nx_set_value("console", console)
	
	--屏幕输出字符颜色
	console.Color = "Purple"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\charge"
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
	ini.FileName = "GameCharge.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--最大连接数
	local max_client = ini:ReadInteger("Main", "MaxClient", 16)
	--游戏编号
	local game_id = ini:ReadInteger("Main", "GameID", 1)
	--游戏服务器起始编号
	local server_id_offset = ini:ReadInteger("Main", "ServerIDOffset", 0)
	--每点对应的游戏时间（分钟）
	local point_minutes = ini:ReadFloat("Main", "PointMinutes", 10)
	--定时导出用户信息间隔时间（分钟）
	local auto_dump_minutes = ini:ReadInteger("Main", "AutoDumpMinutes", 0) 
	--提醒余额不足间隔时间（秒）
	local awake_time = ini:ReadInteger("Main", "AwakeTime", 60)
	--连续多少次扣费不成功时转入程序自主判断，如果费用不够将把玩家踢下线
	local flush_error_max = ini:ReadInteger("Main", "FlushErrorMax", 0)
	--用户不活动的最大时间（秒），超过这个时间系统自动将此用户信息清除（暂时未用）
	local user_shake_time = ini:ReadInteger("Main", "UserShakeTime", 0)
	--写日志到控制台
	local console_log = nx_boolean(ini:ReadString("Main", "ConsoleLog", ""))
	--写日志到文件
	local file_log = nx_boolean(ini:ReadString("Main", "FileLog", ""))
	--是否记录扣费成功日志（不记的话可以减少日志文件大小）
	local flush_log = nx_boolean(ini:ReadString("Main", "FlushLog", ""))
	--分区数据连接数量
	local access_pools = ini:ReadInteger("District", "ConnPool", 0)
	--分区数据连接串
	local access_conn_string = ini:ReadString("District", "ConnString", "")
	--中心数据连接数量
	local center_pools = ini:ReadInteger("Center", "ConnPool", 0)
	--中心数据连接串
	local center_conn_string = ini:ReadString("Center", "ConnString", "")
	
	--外网地址
	local outer_addr = ini:ReadString("Outer", "Addr", "127.0.0.1")
	local outer_port = ini:ReadInteger("Outer", "Port", 7000)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15008
	
	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
	nx_destroy(ini)
	
	--创建维护连接服务
	local echo_sock = nx_create("EchoSock")
	
	echo_sock.Address = echo_addr
	echo_sock.Port = echo_port
	
	nx_bind_script(echo_sock, "echo", "echo_init")
	
	--创建客户端接入通讯服务
	local mass_sock = nx_create("MassSock")
	
	mass_sock.Address = outer_addr
	mass_sock.Port = outer_port
	mass_sock.ConnectMax = max_client
	mass_sock.WorkThreads = 2
	mass_sock.InBufLen = 1024 * 1024
	mass_sock.OutBufLen = 1024 * 1024
	mass_sock.ListenPendings = 5
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "charge"
	app_manager.AppAddress = echo_addr .. ":" .. nx_string(echo_port)
	
	--创建数据库系统
	local db_system = nx_create("SqlDBSystem")
	
	--创建计费服务
	local game_charge = nx_create("GameCharge")
	
	nx_set_value("game_charge", game_charge)
	
	game_charge.GameID = game_id
	game_charge.ServerIDOffset = server_id_offset
	game_charge.PointMinutes = point_minutes
	game_charge.AutoDumpMinutes = auto_dump_minutes
	game_charge.AwakeTime = awake_time
	game_charge.FlushErrorMax = flush_error_max
	game_charge.UserShakeTime = user_shake_time
	game_charge.FlushLog = flush_log
	game_charge.AccessPools = access_pools
	game_charge.AccessConnectString = access_conn_string
	game_charge.CenterPools = center_pools
	game_charge.CenterConnectString = center_conn_string
	game_charge.Console = console
	game_charge.EchoSock = echo_sock
	game_charge.MassSock = mass_sock
	game_charge.AppManager = app_manager
	game_charge.DBSystem = db_system
	
	--启动进程监控
	app_manager:Start()
	
	--暂不接受连接
	mass_sock.Closed = true
	--启动客户端连接
	mass_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动监控服务
	game_charge:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Outer socket " .. outer_addr .. ":" .. nx_string(outer_port))
	
	--启动完成
	console:WriteLine("Game charge server started")
	
	--开始接受连接
	mass_sock.Closed = false
	
	return 1
end

--关闭
function close()
	local console = nx_value("console")
	
	local game_charge = nx_value("game_charge")
	
	local app_manager = game_charge.AppManager
	local mass_sock = game_charge.MassSock
	local echo_sock = game_charge.EchoSock
	
	--关闭外网连接
	console:WriteLine("client socket shutdown...")
	
	mass_sock.Closed = true
	mass_sock:CloseAll()
	
	while mass_sock.ConnectCount > 0 do
		nx_pause(0.0)
	end
	
	mass_sock:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭计费服务
	console:WriteLine("charge server shutdown...")
	
	game_charge:Stop()
	
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

