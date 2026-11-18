--监控服务器主控脚本

--入口函数
function main(console)
	nx_set_value("console", console)
	
	--屏幕输出字符颜色
	console.Color = "LightPurple"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\watch"
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
	ini.FileName = "GameWatch.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--最大连接数
	local max_client = ini:ReadInteger("Main", "MaxClient", 16)
	--监控密码
	local monitor_password = ini:ReadString("main", "MonitorPassword", "")
	--更新密码
	local update_password = ini:ReadString("main", "UpdatePassword", "")
	--查询密码
	local query_password = ini:ReadString("main", "QueryPassword", "")
	--程序启动后是否锁定系统
	local lock_system = nx_boolean(ini:ReadString("main", "LockSystem", ""))
	
	--外网地址
	local outer_addr = ini:ReadString("Outer", "Addr", "127.0.0.1")
	local outer_port = ini:ReadInteger("Outer", "Port", 6000)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15006
	
	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
	--nx_destroy(ini)
	
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
	mass_sock.WorkThreads = 1
	mass_sock.InBufLen = 1024 * 1024
	mass_sock.OutBufLen = 1024 * 1024
	mass_sock.ListenPendings = 5
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "watch"
	app_manager.AppAddress = echo_addr .. ":" .. nx_string(echo_port)
	--监控程序才需要定时检测其他应用是否存在
	app_manager.NeedCheckApp = true
	
	--创建数据库系统
	local db_system = nx_create("SqlDBSystem")
	
	--创建监控服务
	local game_watch = nx_create("GameWatch")
	
	nx_set_value("game_watch", game_watch)
	
	--进程路径
	local server_table = ini:GetItemList("Servers")
	
	for i = 1, table.getn(server_table) do
		local key = server_table[i]
		local val = ini:ReadString("Servers", key, "")
		
		if val ~= "" then
			game_watch:AddServer(key, val, "")
		end
	end
	
	--可执行文件路径
	local exe_path_table = ini:GetItemList("ExePath")
	
	for i = 1, table.getn(exe_path_table) do
		local key = exe_path_table[i]
		local val = ini:ReadString("ExePath", key, "")
		
		if val ~= "" then
			game_watch:AddExePath(key, val)
		end
	end
	
	--资源文件路径
	local res_path_table = ini:GetItemList("ResPath")
	
	for i = 1, table.getn(res_path_table) do
		local key = res_path_table[i]
		local val = ini:ReadString("ResPath", key, "")
		
		if val ~= "" then
			game_watch:AddResPath(key, val)
		end
	end
	
	--允许连接的合法客户端地址
	local addr_num = ini:ReadInteger("Address", "AddressNum", 0)
	
	for i = 0, (addr_num - 1) do
		local address = ini:ReadString("Address", "Address" .. nx_string(i), "")
		
		if address ~= "" then
			game_watch:AddClientAddress(address)
		end
	end
	
	nx_destroy(ini)
	
	--加载数据库连接配置文件
	game_watch:LoadAccessConfig("access.ini")
	
	game_watch.MonitorPassword = monitor_password
	game_watch.UpdatePassword = update_password
	game_watch.QueryPassword = query_password
	game_watch.LockSystem = lock_system
	game_watch.Console = console
	game_watch.EchoSock = echo_sock
	game_watch.MassSock = mass_sock
	game_watch.AppManager = app_manager
	game_watch.DBSystem = db_system
	
	--启动进程监控
	app_manager:Start()
	
	--暂不接受连接
	mass_sock.Closed = true
	--启动客户端连接
	mass_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动监控服务
	game_watch:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Outer socket " .. outer_addr .. ":" .. nx_string(outer_port))
	
	--启动完成
	console:WriteLine("Game watch server started")
	
	--开始接受连接
	mass_sock.Closed = false
	
	return 1
end

--关闭
function close()
	local console = nx_value("console")
	
	local game_watch = nx_value("game_watch")
	
	local app_manager = game_watch.AppManager
	local mass_sock = game_watch.MassSock
	local echo_sock = game_watch.EchoSock
	
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
	
	--关闭监控服务
	console:WriteLine("watch server shutdown...")
	
	game_watch:Stop()
	
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

