--机器人主控脚本

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
	log_pool.FilePrefix = nx_work_path() .. "log\\robot"
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
	ini.FileName = "GameRobot.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--通讯线程数
	local sock_threads = ini:ReadInteger("Main", "SockThreads", 1)
	--最大连接数
	local max_player = ini:ReadInteger("Main", "MaxPlayer", 1024)
	--通讯加密键
	local msg_encode_key = ini:ReadString("Main", "MessageEncodeKey", "")
	
	--外网地址
	local outer_addr = ini:ReadString("Outer", "Addr", "127.0.0.1")
	local outer_port = ini:ReadInteger("Outer", "Port", 2001)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15900
	
	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
	--帐号和密码前缀
	local account_prefix = "robot"
	local password_prefix = "robot"
	
	account_prefix = ini:ReadString("Main", "AccountPrefix", account_prefix)
	password_prefix = ini:ReadString("Main", "PasswordPrefix", password_prefix)
	
	nx_destroy(ini)
	
	--创建维护连接服务
	local echo_sock = nx_create("EchoSock")
	
	echo_sock.Address = echo_addr
	echo_sock.Port = echo_port
	
	nx_bind_script(echo_sock, "echo", "echo_init")
	
	--创建客户端连接通讯服务
	local multi_sock = nx_create("MultiSock")
	
	multi_sock.Address = outer_addr
	multi_sock.Port = outer_port
	multi_sock.ConnectMax = 8 * 1024
	multi_sock.WorkThreads = sock_threads
	multi_sock.InBufLen = 16 * 1024
	multi_sock.OutBufLen = 64 * 1024
	
	--创建机器人
	local game_robot = nx_create("GameRobot")
	
	nx_set_value("game_robot", game_robot)
	
	game_robot.MaxConnects = max_player
	game_robot.AccountPrefix = account_prefix
	game_robot.PasswordPrefix = password_prefix
	game_robot.Console = console
	game_robot.EchoSock = echo_sock
	game_robot.MultiSock = multi_sock
	
	--启动客户端连接
	multi_sock:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动接入服务
	game_robot:Start()
	
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Outer socket " .. outer_addr .. ":" .. nx_string(outer_port))
	
	--启动完成
	console:WriteLine("Game robot started")
	
	nx_execute(nx_current(), "begin_connect", multi_sock, max_player)
	
	return 1
end

--启动客户连接
function begin_connect(multi_sock, connect_num)
	local i = 0
	
	while i < connect_num do
		nx_pause(0.05)
		
		multi_sock:ClientBegin(i)
		
		i = i + 1
	end
end

--关闭
function close()
	local console = nx_value("console")
	local game_robot = nx_value("game_robot")
	local multi_sock = game_robot.MultiSock
	local echo_sock = game_robot.EchoSock
	
	--关闭外网连接
	console:WriteLine("client socket shutdown...")
	
	multi_sock:CloseAll()
	
	while multi_sock.ConnectCount > 0 do
		nx_pause(0.0)
	end
	
	multi_sock:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭机器人
	console:WriteLine("entry server shutdown...")
	
	game_robot:Stop()
	
	--关闭日志缓冲
	local log_pool = nx_value("log_pool")
	
	log_pool:Stop()
	
	--结束程序
	nx_quit()
	
	return true
end

