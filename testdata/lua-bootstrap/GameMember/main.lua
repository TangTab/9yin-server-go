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
	console.Color = "Gray"
	
	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end
	
	--日志缓冲池
	local log_pool = nx_create("LogPool")
	
	--日志文件前缀
	log_pool.FilePrefix = nx_work_path() .. "log\\member"
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
	ini.FileName = "GameMember.ini"
	
	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("无法读取程序配置文件")
		nx_destroy(ini)
		return 0
	end
	
	--游戏分区编号
	local district_id = ini:ReadInteger("Main", "DistrictID", 1)
	--游戏服务器编号
	local server_id = ini:ReadInteger("Main", "ServerID", 1)
	--场景服务器编号
	local member_id = ini:ReadInteger("Main", "MemberID", 0)
	--场景碰撞信息文件的根目录
	local collide_root_path = ini:ReadString("Main", "CollideRootPath", "")
	
	if collide_root_path ~= "" then
		collide_root_path = nx_function("ext_get_full_path", collide_root_path)
	end
	
	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 16300 + member_id
	
	inner_addr = ini:ReadString("Inner", "Addr", inner_addr)
	inner_port = ini:ReadInteger("Inner", "Port", inner_port)
	
	--广播地址
	local broad_addr = "127.255.255.255"
	local broad_port = 15000
	
	broad_addr = ini:ReadString("Broadcast", "Addr", broad_addr)
	broad_port = ini:ReadInteger("Broadcast", "Port", broad_port)
	
	--维护连接地址
	local echo_addr = "127.0.0.1"
	local echo_port = 15300 + member_id
	
	echo_addr = ini:ReadString("Echo", "Addr", echo_addr)
	echo_port = ini:ReadInteger("Echo", "Port", echo_port)
	
	--逻辑模块
	local logic_loader = nx_create("LogicLoader")
	
	local logic_path = ini:ReadString("Logic", "Path", "")
	local modules = ini:ReadInteger("Logic", "Modules", 0)
	
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
	
	--for i = 0, modules do
	--	local dll_name = ini:ReadString("Logic", "Module" .. nx_string(i), "")
	--	
	--	if dll_name ~= "" then
	--		logic_loader:Load(nx_work_path() .. logic_path, dll_name)
	--	end
	--end
	
	nx_destroy(ini)
	
	--场景碰撞信息
	local collide_manager = nx_create("CollideManager")
	
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
	
	app_manager.AppName = "member_" .. nx_string(server_id) .. "_" .. nx_string(member_id)
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
	
	--创建场景服务
	local game_member = nx_create("GameMember")
	
	nx_set_value("game_member", game_member)
	
	game_member.DistrictId = district_id
	game_member.ServerId = server_id
	game_member.MemberId = member_id
	game_member.Console = console
	game_member.EchoSock = echo_sock
	game_member.BroadSock = broad_sock
	game_member.AppManager = app_manager
	game_member.Adapter = adapter
	game_member.LogicLoader = logic_loader
	game_member.CollideManager = collide_manager
	game_member.ScriptPath = nx_resource_path() .. "lua\\"
	game_member.TaskPath = nx_resource_path() .. "task\\"
	game_member.CollideRootPath = collide_root_path
	
	--加载名字过滤文件
	game_member:LoadNameFilter(nx_resource_path() .. "filter.txt")
	
	--加载全局点信息
	game_member:LoadLocations(nx_resource_path() .. "location.ini")
	
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
	
	--启动场景服务
	game_member:Start()
	
	--显示应用程序名
	console:WriteLine("App name " .. app_manager.AppName)
	--显示通讯地址和端口
	console:WriteLine("Echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	console:WriteLine("Inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	console:WriteLine("Broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	
	--启动完成
	console:WriteLine("Game server " .. nx_string(server_id) 
		.. " member " .. nx_string(member_id) .. " started")
	
	--设置进程开启状态
	app_manager:SetAppState(WAPS_OPENED);
	
	return 1
end

--关闭
function close()
	local console = nx_value("console")
	
	local game_member = nx_value("game_member")
	
	local app_manager = game_member.AppManager
	local broad_sock = game_member.BroadSock
	local echo_sock = game_member.EchoSock
	local adapter = game_member.Adapter
	
	--关闭广播
	console:WriteLine("broad shutdown...")
	
	broad_sock:Stop()
	
	--关闭内网通讯
	console:WriteLine("adapter shutdown...")
	
	adapter:Stop()
	
	--关闭维护连接
	console:WriteLine("echo socket shutdown...")
	
	echo_sock:Stop()
	
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

