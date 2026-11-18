--接入服务器主控脚本

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
	console.Color = "LightRed"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\entry"
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
	ini.FileName = "GameEntry.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--IDC编号
	local idc = ini:ReadInteger("Main", "IDC", 1)
	--游戏分区编号
	local district_id = ini:ReadInteger("Main", "DistrictID", 1)
	--服务器编号
	local server_id = ini:ReadInteger("Main", "ServerID", 1)
	--组内接入编号
	local entry_id = ini:ReadInteger("Main", "EntryID", 0)
	--通讯线程数
	local sock_threads = ini:ReadInteger("Main", "SockThreads", 1)
	--最大连接数
	local max_player = ini:ReadInteger("Main", "MaxPlayer", 1024)
	--通讯加密键
	local msg_encode_key = ini:ReadString("Main", "MessageEncodeKey", "")
	--通讯校验DLL
	local msg_verify_dll = ini:ReadString("Main", "MessageVerifyDll", "")
	
	--外网地址
	local outer_addr = ini:ReadString("Outer", "Addr", "127.0.0.1")
	local outer_port = ini:ReadInteger("Outer", "Port", 2001)
	
	--授权文件名
	local license = outer_addr .. ".lic"
	
	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 16200 + entry_id
	
	inner_addr = ini:ReadString("Inner", "Addr", inner_addr)
	inner_port = ini:ReadInteger("Inner", "Port", inner_port)
	
	--广播地址
	local broad_addr = "127.255.255.255"
	local broad_port = 15000
	
	broad_addr = ini:ReadString("Broadcast", "Addr", broad_addr)
	broad_port = ini:ReadInteger("Broadcast", "Port", broad_port)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15200 + entry_id
	
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
	mass_sock.ConnectMax = 8 * 1024
	mass_sock.WorkThreads = sock_threads
	mass_sock.InBufLen = 16 * 1024
	mass_sock.OutBufLen = 64 * 1024
	mass_sock.ListenPendings = 20
	
	--创建广播通讯
	local broad_sock = nx_create("BroadSock")
	
	broad_sock.BindAddress = inner_addr
	broad_sock.BroadcastAddress = broad_addr
	broad_sock.BroadcastPort = broad_port
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "entry_" .. nx_string(server_id) .. "_" .. nx_string(entry_id)
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
	
	--创建接入服务
	local game_entry = nx_create("GameEntry")
	
	nx_set_value("game_entry", game_entry)
	
	game_entry.IDC = idc
	game_entry.DistrictId = district_id
	game_entry.ServerId = server_id
	game_entry.EntryId = entry_id
	game_entry.MaxConnects = max_player
	game_entry.License = license
	game_entry.EncodeKey = game_entry:ConvertEncodeKey(msg_encode_key)
	game_entry.DecodeKey = game_entry:ConvertEncodeKey(msg_encode_key)
	game_entry.VerifyDll = msg_verify_dll
	game_entry.Console = console
	game_entry.EchoSock = echo_sock
	game_entry.MassSock = mass_sock
	game_entry.BroadSock = broad_sock
	game_entry.AppManager = app_manager
	game_entry.Adapter = adapter
	--记录与客户端的收发消息到message.log文件
	--game_entry.LogConnect = true
	--game_entry.LogSend = true
	--game_entry.LogReceive = true
	
	--启动进程监控
	app_manager:Start()
	
	--设置进程启动状态
	app_manager:SetAppState(WAPS_STARTING)
	
	--启动内部通讯
	adapter:Start()
	
	--启动广播通讯
	broad_sock:Start()
	
	--暂不接受连接
	mass_sock.Closed = true
	--启动客户端连接
	mass_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动接入服务
	game_entry:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	console:WriteLine("Broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	console:WriteLine("Outer socket " .. outer_addr .. ":" .. nx_string(outer_port))
	
	--启动完成
	console:WriteLine("Game server " .. nx_string(server_id) .. " entry "
		.. nx_string(entry_id) .. " started")
	
	--设置进程开启状态
	app_manager:SetAppState(WAPS_OPENED);
	
	return 1
end

--关闭
function close()
	local console = nx_value("console")
	
	local game_entry = nx_value("game_entry")
	
	local app_manager = game_entry.AppManager
	local broad_sock = game_entry.BroadSock
	local mass_sock = game_entry.MassSock
	local echo_sock = game_entry.EchoSock
	local adapter = game_entry.Adapter
	
	--关闭外网连接
	console:WriteLine("client socket shutdown...")
	
	mass_sock.Closed = true
	mass_sock:CloseAll()
	
	while mass_sock.ConnectCount > 0 do
		nx_pause(0.0)
	end
	
	mass_sock:Stop()
	
	--关闭广播
	console:WriteLine("broad shutdown...")
	
	broad_sock:Stop()
	
	--关闭内网通讯
	console:WriteLine("adapter shutdown...")
	
	adapter:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭接入服务
	console:WriteLine("entry server shutdown...")
	
	game_entry:Stop()
	
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

