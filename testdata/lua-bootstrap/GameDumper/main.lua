--数据转储服务器主控脚本

--入口函数
function main(console)
	nx_set_value("console", console)
	
	--屏幕输出字符颜色
	console.Color = "Yellow"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\dumper"
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
	ini.FileName = "GameDumper.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--数据转储逻辑DLL文件
	local logic_dll = ini:ReadString("Main", "DumpLogicDll", "")
	--使用MYSQL数据库
	local use_mysql = nx_boolean(ini:ReadString("Main", "UseMysql", "true"))
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15009

	local outer_addr = ini:ReadString("Outer", "Addr", "")
	local outer_port = ini:ReadInteger("Outer", "Port", 0)

	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
	--创建维护连接服务
	local echo_sock = nx_create("EchoSock")
	
	echo_sock.Address = echo_addr
	echo_sock.Port = echo_port
	
	nx_bind_script(echo_sock, "echo", "echo_init")
	
	--创建进程监控
	local app_manager = nx_create("AppManager")
	
	app_manager.AppName = "dumper"
	app_manager.AppAddress = echo_addr .. ":" .. nx_string(echo_port)
	
	--创建数据库系统
	local db_system
	
	if use_mysql then
		db_system = nx_create("MysqlDBSystem")
	else
		db_system = nx_create("SqlDBSystem")
	end
	
	--创建数据转储服务
	local game_dumper = nx_create("GameDumper")
	
	nx_set_value("game_dumper", game_dumper)
	
	--设置数据库连接配置
	local role_db, public_db, manage_db, output_db = get_db_config(ini)
    
	nx_destroy(ini)
	
	--逻辑模块
	local logic_loader = nx_create("LogicLoader")
	
	if logic_dll ~= "" then
		if logic_loader:Load(nx_work_path(), logic_dll) then
			console:WriteLine("Load loigc module " .. logic_dll)
		else
			console:WriteLine("Load loigc module " .. logic_dll .. " failed")
		end
	end
	
	game_dumper.Console = console
	game_dumper.EchoSock = echo_sock
	game_dumper.AppManager = app_manager
	game_dumper.LogicLoader = logic_loader
	game_dumper.DBSystem = db_system
	
	game_dumper:AddAccess("role", role_db)
	game_dumper:AddAccess("public", public_db)
	game_dumper:AddAccess("manage", manage_db)
	game_dumper:AddAccess("output", output_db)
	
--设置服务器连接端口
	game_dumper:SetOutAddr(outer_addr, outer_port)
	--启动进程监控
	app_manager:Start()
	
	--启动维护连接
	echo_sock:Start()
	
	--启动数据转储服务
	game_dumper:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	
	--启动完成
	console:WriteLine("Game dumper server started")
	
	return 1
end

--获得数据库连接的配置信息
function get_db_config(ini)
	--尝试从联合服务器的配置读取
	local unite_ini = nx_create("IniDocument")
	
	unite_ini.FileName = "GameDumper.ini"
	
	if unite_ini:LoadFromFile() then
		local role_db = unite_ini:ReadString("Role", "ConnectString", "")
		local public_db = unite_ini:ReadString("Public", "ConnectString", "")
		local manage_db = unite_ini:ReadString("Manage", "ConnectString", "")
		local output_db = unite_ini:ReadString("Manage", "ConnectString", "")
		
		if public_db == "" then
			public_db = role_db
		end
		
		if manage_db == "" then
			manage_db = role_db
		end
		
		if output_db == "" then
			output_db = role_db
		end
		
		nx_destroy(unite_ini)
		
		return role_db, public_db, manage_db ,output_db
	end
	
	nx_destroy(unite_ini)
	
	--尝试从存储服务器的配置读取
	local store_ini = nx_create("IniDocument")
	
	store_ini.FileName = "GameStore.ini"
	
	if store_ini:LoadFromFile() then
		local role_db = store_ini:ReadString("Role", "ConnectString", "")
		local public_db = store_ini:ReadString("Public", "ConnectString", "")
		local manage_db = store_ini:ReadString("Manage", "ConnectString", "")
		local output_db = store_ini:ReadString("Output", "ConnectString", "")
		
		if public_db == "" then
			public_db = role_db
		end
		
		if manage_db == "" then
			manage_db = role_db
		end
		
		if output_db == "" then
			output_db = role_db
		end
		
		nx_destroy(store_ini)
		
		return role_db, public_db, manage_db, output_db
	end
	
	nx_destroy(store_ini)
	
	--从数据转储服务器的配置读取
	local role_db = ini:ReadString("Role", "ConnectString", "")
	local public_db = ini:ReadString("Public", "ConnectString", "")
	local manage_db = ini:ReadString("Manage", "ConnectString", "")
	local output_db = ini:ReadString("Output", "ConnectString", "")
		
	if public_db == "" then
		public_db = role_db
	end
	
	if manage_db == "" then
		manage_db = role_db
	end
		
	if output_db == "" then
		output_db = role_db
	end
	
	return role_db, public_db, manage_db, output_db
end

--关闭
function close()
	local console = nx_value("console")
	local game_dumper = nx_value("game_dumper")
	local app_manager = game_dumper.AppManager
	local echo_sock = game_dumper.EchoSock
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
	--关闭数据转储服务
	console:WriteLine("dumper server shutdown...")
	
	game_dumper:Stop()
	
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

