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
	local member = nx_create("SoloMember")
	local public = nx_create("SoloPublic")
	local entry = nx_create("SoloEntry")
	local store = nx_create("SoloStore")
	local logger = nx_create("SoloLog")
    local auction = nx_create("SoloAuction")

	nx_set_value("world", world)
	nx_set_value("member", member)
	nx_set_value("public", public)
	nx_set_value("entry", entry)
	nx_set_value("store", store)
	nx_set_value("logger", logger)
    nx_set_value("auction", auction)

	--创建日志目录
	if not nx_path_exists(nx_work_path() .. "log") then
		nx_path_create(nx_work_path() .. "log")
	end

	logger.UseThread = true
	--日志文件存放目录
	logger.LogFileDir = nx_work_path() .. "log\\"
	--是否每天一个日志文件
	logger.EveryDayAlone = false

	--快速日志系统
	world.UseQuickLog = true

	--应用进程记录
	world.UseAppRecord = true
	member.UseAppRecord = false
	public.UseAppRecord = false
    auction.UseAppRecord = false
	entry.UseAppRecord = false
	store.UseAppRecord = false

	--使用线程
	world.UseThread = false
	member.UseThread = true
	public.UseThread = true
    auction.UseThread = true
	entry.UseThread = true
	store.UseThread = true

	--屏幕输出字符颜色
	world:SetConsoleColor("White")

	--读取程序配置
	local ini = nx_create("IniDocument")

	--配置文件名
	--ini.FileName = "game_unite.ini"
	ini.FileName = nx_main_config()

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
    --公共拍卖行逻辑DLL文件
	local auction_logic_dll = ini:ReadString("Main", "AuctionLogicDll", "")
	--最大线人数
	local max_players = ini:ReadInteger("Main", "MaxPlayers", 8192)
	--场景碰撞信息文件的根目录
	local collide_root_path = ini:ReadString("Main", "CollideRootPath", "")
	--使用MYSQL数据库
	local use_mysql = nx_boolean(ini:ReadString("Main", "UseMysql", "false"))
	--记录数据库存取性能
	local log_db_perf = nx_boolean(ini:ReadString("Main", "LogDBPerformance", "false"))
	--通讯加密键
	local msg_encode_key = ini:ReadString("Main", "MessageEncodeKey", "")
	--通讯校验DLL
	local msg_verify_dll = ini:ReadString("Main", "MessageVerifyDll", "")
	--出现错误是否提示
	local debug_halt = nx_boolean(ini:ReadString("Main", "DebugHalt", "true"))
	--刷新账户信息的间隔分钟数
	local flush_minutes = ini:ReadInteger("Main", "FlushMinutes", 0)
	--创建角色对角色名字长度的限制
	local rolenamesizelimit = ini:ReadInteger("Main", "RoleNameSizeLimit", 16)

	if collide_root_path ~= "" then
		collide_root_path = nx_function("ext_get_full_path", collide_root_path)
	end

	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 0
	--local inner_port = 16000

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

    --监控服务器
	local monitor_inuse = nx_boolean(ini:ReadString("Monitor", "Inuse", ""))
	local monitor_addr = ini:ReadString("Monitor", "Addr", "")
	local monitor_port = ini:ReadInteger("Monitor", "Port", 0)

    --转储服务器
	local dumpserver_inuse = nx_boolean(ini:ReadString("DumpServer", "Inuse", ""))
	local dumpserver_addr = ini:ReadString("DumpServer", "Addr", "")
	local dumpserver_port = ini:ReadInteger("DumpServer", "Port", 0)

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
				world:WriteLine("load loigc module " .. dll_table[i])
			else
				world:WriteLine("load loigc module " .. dll_table[i] .. " failed")
			end
		end

		nx_destroy(fs)
	else
		for i = 0, modules do
			local dll_name = ini:ReadString("Logic", "Module" .. nx_string(i), "")

			if dll_name ~= "" then
				if logic_loader:Load(nx_work_path() .. logic_path, dll_name) then
					world:WriteLine("load loigc module " .. dll_name)
				else
					world:WriteLine("load loigc module " .. dll_name .. " failed")
				end
			end
		end
	end

	nx_destroy(ini)

	--公共数据逻辑模块
	local public_logic_loader = nx_create("LogicLoader")

	if public_logic_dll ~= "" then
		if public_logic_loader:Load(nx_work_path(), public_logic_dll) then
			world:WriteLine("load loigc module " .. public_logic_dll)
		else
			world:WriteLine("load loigc module " .. public_logic_dll .. " failed")
		end
	end

    --拍卖行数据逻辑模块
	local auction_logic_loader = nx_create("LogicLoader")

	if auction_logic_dll ~= "" then
		if auction_logic_loader:Load(nx_work_path(), auction_logic_dll) then
			world:WriteLine("load loigc module " .. auction_logic_dll )
		else
			world:WriteLine("load loigc module " .. auction_logic_dll .. " failed")
		end
	end

	--场景碰撞信息
	local collide_manager = nx_create("CollideManager")

	--创建数据库系统
	local db_system

	if use_mysql then
		db_system = nx_create("MysqlDBSystem")
	else
		db_system = nx_create("SqlDBSystem")
	end

	--绑定脚本
	nx_bind_script(world, "main", "echo_init")

	--游戏服务
	world.AppName = "unite_" .. nx_string(server_id)
	world.DistrictId = district_id
	world.ServerId = server_id
	world.ServerName = nx_widestr(server_name)
	world.MaxPlayers = max_players
	--每个帐号可以创建的角色数量
	world.AccountRoles = 5
	--创建角色对角色名字长度的限制
	world.RoleNameSizeLimit = rolenamesizelimit
	world.FlushMinutes = flush_minutes
	world.InnerAddr = inner_addr
	world.InnerPort = 0
	--world.InnerPort = inner_port + 1
	world.BroadAddr = broad_addr
	world.BroadPort = broad_port
	world.EchoAddr = echo_addr
	world.EchoPort = echo_port

	--加载名字过滤文件
	world:LoadNameFilter(nx_resource_path() .. "filter.txt")

	--加载验证码字符图像
	load_char_image(world)

	--添加普通场景
	load_scene_config(world)

	--公共数据服务
	public.AppName = "public_" .. nx_string(server_id)
	public.DistrictId = district_id
	public.ServerId = server_id
	public.LogicLoader = public_logic_loader
	public.InnerAddr = inner_addr
	public.InnerPort = 0
	--public.InnerPort = inner_port + 2
	public.BroadAddr = broad_addr
	public.BroadPort = broad_port

	--添加公共数据空间
	public:AddPubSpace("domain", "nx_domains")
	public:AddPubSpace("guild", "nx_guilds")
	--public:AddPubSpace("pubdata", "nx_pubdata")
	--public:AddPubSpace("guild", "nx_guilds")

	--存储服务
	store.AppName = "store_" .. nx_string(server_id) .. "_0"
	store.DistrictId = district_id
	store.ServerId = server_id
	store.StoreId = 0
	store.AccountBase = account_base
	store.AccountPools = account_pool
	store.RoleBase = role_base
	store.RolePools = role_pool
	store.LogBase = log_base
	store.LogPools = log_pool
	store.PublicBase = public_base
	store.PublicPools = public_pool
	store.ManageBase = manage_base
	store.ManagePools = manage_pool
	store.DBSystem = db_system
	store.InnerAddr = inner_addr
	store.InnerPort = 0
	--store.InnerPort = inner_port + 100
	store.BroadAddr = broad_addr
	store.BroadPort = broad_port
	store.WakeAddr = "127.0.0.1"
	store.WakePort = 0
	--store.WakePort = 16800
	--记录数据库存取性能日志
	store.LogDBPerformance = log_db_perf

	init_custom_log(store)

    --拍卖行数据服务
	auction.AppName = "auction_" .. nx_string(server_id)
	auction.DistrictId = district_id
	auction.ServerId = server_id
	auction.LogicLoader = auction_logic_loader
	auction.InnerAddr = inner_addr
	auction.InnerPort = 0
	auction.BroadAddr = broad_addr
	auction.BroadPort = broad_port

	--接入服务
	entry.AppName = "entry_" .. nx_string(server_id) .. "_0"
	entry.IDC = 0
	entry.DistrictId = district_id
	entry.ServerId = server_id
	entry.EntryId = 0
	entry.MaxConnects = 1024
	entry.License = license
	entry.EncodeKey = entry:ConvertEncodeKey(msg_encode_key)
	entry.DecodeKey = entry:ConvertEncodeKey(msg_encode_key)
	entry.VerifyDll = msg_verify_dll
	entry.InnerAddr = inner_addr
	--entry.InnerPort = inner_port + 200
	entry.InnerPort = 0
	entry.BroadAddr = broad_addr
	entry.BroadPort = broad_port
	entry.OuterAddr = outer_addr
	entry.OuterPort = outer_port
	--接收缓冲区大小
	entry.InBufferLen = 8 * 1024
	--发送缓冲区大小
	entry.OutBufferLen = 32 * 1024
	--发送缓冲区上限
	entry.OutBufferMax = 128 * 1024
	--记录与客户端的收发消息到message.log文件
	--entry.LogConnect = true
	--entry.LogSend = true
	--entry.LogReceive = true

	--场景服务
	member.AppName = "member_" .. nx_string(server_id) .. "_0"
	member.DistrictId = district_id
	member.ServerId = server_id
	member.MemberId = 0
	member.LogicLoader = logic_loader
	member.CollideManager = collide_manager
	member.ScriptPath = nx_resource_path() .. "lua\\"
	member.DebugHalt = debug_halt
	member.CollideRootPath = collide_root_path
	member.InnerAddr = inner_addr
	member.InnerPort = 0
	--member.InnerPort = inner_port + 300
	member.BroadAddr = broad_addr
	member.BroadPort = broad_port
	--是否开启视野消息优化
	--member.SightOptimize = true
	--是否不发送为缺省空值的对象属性
	member.HideEmptyProperty = true

	--加载名字过滤文件
	member:LoadNameFilter(nx_resource_path() .. "filter.txt")
	--加载全局点信息
	member:LoadLocations(nx_resource_path() .. "location.xml")

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

	--启动日志服务
	logger:Start()
	--启动游戏服务
	world:Start()
	--启动存储服务
	store:Start()
	--启动接入服务
	entry:Start()
	--启动公共数据服务
	public:Start()
    --启动拍卖行数据服务
	auction:Start()
	--启动场景服务
	member:Start()

	--显示应用程序名
	world:WriteLine("app name " .. world.AppName)
	--显示通讯地址和端口
	world:WriteLine("echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	--world:WriteLine("inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	world:WriteLine("broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	world:WriteLine("outer socket " .. outer_addr .. ":" .. nx_string(outer_port))

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

--设置自定义的游戏日志数据库表格结构
function init_custom_log(store)
	--物品日志表
	store:AddLogTable("item_log")
	store:AddLogField("item_log", "time", "datetime", 8)
	store:AddLogField("item_log", "itemfrom", "nvarchar", 64)
	store:AddLogField("item_log", "itemto", "nvarchar", 64)
	store:AddLogField("item_log", "itemid", "varchar", 32)
	store:AddLogField("item_log", "instance_id", "varchar", 32)
	store:AddLogField("item_log", "item_num", "decimal", 8)
	store:AddLogField("item_log", "logtype", "int", 4)
	store:AddLogField("item_log", "comment1", "nvarchar", 255)
	store:AddLogField("item_log", "comment2", "nvarchar", 255)
	store:AddLogField("item_log", "comment3", "nvarchar", 255)


	--玩家交易表
	store:AddLogTable("deal_log")
	store:AddLogField("deal_log", "tradeid", "varchar", 64)
	store:AddLogField("deal_log", "time", "datetime", 8)
	store:AddLogField("deal_log", "buyer", "nvarchar", 64)
	store:AddLogField("deal_log", "seller", "nvarchar", 64)
	store:AddLogField("deal_log", "itemid", "varchar", 32)
	store:AddLogField("deal_log", "instance_id", "varchar", 32)
	store:AddLogField("deal_log", "count", "int", 4)
	store:AddLogField("deal_log", "getorlost", "int", 4)
	store:AddLogField("deal_log", "tradechannel", "int", 4)

	--game_player_action_log
	store:AddLogTable("game_player_action_log")
	store:AddLogField("game_player_action_log", "s_account", "varchar", 64)
	store:AddLogField("game_player_action_log", "s_role_name", "nvarchar", 64)
	store:AddLogField("game_player_action_log", "d_create", "datetime", 8)
	store:AddLogField("game_player_action_log", "s_act_id", "varchar", 32)
	store:AddLogField("game_player_action_log", "n_type", "int", 4)
	store:AddLogField("game_player_action_log", "n_act_value", "int", 4)


	--玩家登陆日志表
	store:AddLogTable("role_log")
	store:AddLogField("role_log", "account", "varchar", 100)
	store:AddLogField("role_log", "role", "nvarchar", 64)
	store:AddLogField("role_log", "logintime", "datetime", 8)
	store:AddLogField("role_log", "logouttime", "datetime", 8)


	--玩家升级日志表
	--store:AddLogTable("role_upgrade_log")
	--store:AddLogField("role_upgrade_log", "time", "datetime", 8)
	--store:AddLogField("role_upgrade_log", "role", "nvarchar", 64)
	--store:AddLogField("role_upgrade_log", "logtype", "int", 4)
	--store:AddLogField("role_upgrade_log", "old_value", "int", 4)
	--store:AddLogField("role_upgrade_log", "new_value", "int", 4)
	--store:AddLogField("role_upgrade_log", "prop_name", "varchar", 32)
	--store:AddLogField("role_upgrade_log", "ip", "varchar", 16)
	--store:AddLogField("role_upgrade_log", "mac", "varchar", 16)

	--任务日志表
	store:AddLogTable("task_log")
	store:AddLogField("task_log", "time", "datetime", 8)
	store:AddLogField("task_log", "role", "nvarchar", 64)
	store:AddLogField("task_log", "task", "int", 4)
	store:AddLogField("task_log", "logtype", "int", 4)

	--副本日志表
	store:AddLogTable("instance_log")
	store:AddLogField("instance_log", "instanceid", "varchar", 32)
	store:AddLogField("instance_log", "instancetypeid", "varchar", 32)
	store:AddLogField("instance_log", "starttime", "datetime", 8)
	store:AddLogField("instance_log", "endtime", "datetime", 8)
	store:AddLogField("instance_log", "duration", "int", 4)
	store:AddLogField("instance_log", "troop_count1", "int", 4)
	store:AddLogField("instance_log", "troop_count2", "int", 4)

    --副本日志表2
	store:AddLogTable("instance_role_log")
	store:AddLogField("instance_role_log", "instanceid", "varchar", 32)
	store:AddLogField("instance_role_log", "instancetypeid", "varchar", 32)
	store:AddLogField("instance_role_log", "entertime", "datetime", 8)
	store:AddLogField("instance_role_log", "leavetime", "datetime", 8)
	store:AddLogField("instance_role_log", "duration", "int", 4)
	store:AddLogField("instance_role_log", "role", "nvarchar", 64)


	store:AddLogTable("instance_npc_log")
	store:AddLogField("instance_npc_log", "time", "datetime", 8)
	store:AddLogField("instance_npc_log", "npcname",  "nvarchar", 64)
	store:AddLogField("instance_npc_log", "sceneid", "int", 4)
    store:AddLogField("instance_npc_log", "location", "varchar", 32)
	store:AddLogField("instance_npc_log", "role", "nvarchar", 64)
    store:AddLogField("instance_npc_log", "ip", "varchar", 16)
	store:AddLogField("instance_npc_log", "mac", "varchar", 16)

	--场景日志表
	store:AddLogTable("scene_log")
	store:AddLogField("scene_log", "sceneid", "int", 4)
	store:AddLogField("scene_log", "date", "datetime", 8)
	store:AddLogField("scene_log", "enter", "int", 4)
	store:AddLogField("scene_log", "totaltime", "decimal", 8)

	--惩罚日志表
	store:AddLogTable("punish_log")
	store:AddLogField("punish_log", "time", "datetime", 8)
	store:AddLogField("punish_log", "operator", "varchar", 100)
	store:AddLogField("punish_log", "role", "nvarchar", 64)
	store:AddLogField("punish_log", "minutes", "int", 4)
	store:AddLogField("punish_log", "type", "int", 4)


	--program_log
	store:AddLogTable("program_log")
	store:AddLogField("program_log", "programid", "int", 4)
	store:AddLogField("program_log", "programname", "varchar", 64)
	store:AddLogField("program_log", "opentime", "datetime", 8)
	store:AddLogField("program_log", "endtime", "datetime", 8)


	--游戏世界资金总量调整日志
	store:AddLogTable("total_capital_log")
	store:AddLogField("total_capital_log", "time", "datetime", 8)
	store:AddLogField("total_capital_log", "type", "int", 4)
	store:AddLogField("total_capital_log", "res", "int", 4)
	store:AddLogField("total_capital_log", "capital_type", "int", 4)
	store:AddLogField("total_capital_log", "inc_value", "decimal", 20)
	store:AddLogField("total_capital_log", "old_value", "decimal", 20)
	store:AddLogField("total_capital_log", "new_value", "decimal", 20)
	store:AddLogField("total_capital_log", "gm", "varchar", 128)
	store:AddLogField("total_capital_log", "comment", "nvarchar", 255)

	--玩家经验日志表
	store:AddLogTable("player_exp_log")
	store:AddLogField("player_exp_log", "time", "datetime", 8)
	store:AddLogField("player_exp_log", "role", "nvarchar", 64)
	store:AddLogField("player_exp_log", "logtype", "int", 4)
	store:AddLogField("player_exp_log", "amount", "int", 4)
	store:AddLogField("player_exp_log", "comment", "nvarchar", 255)
    store:AddLogField("player_exp_log", "ip", "varchar", 16)
	store:AddLogField("player_exp_log", "mac", "varchar", 16)

    --行为统计日志
	store:AddLogTable("deed_log")
	store:AddLogField("deed_log", "role", "nvarchar", 64)
	store:AddLogField("deed_log", "d_create", "datetime", 8)
	store:AddLogField("deed_log", "s_act_id", "varchar", 32)
	store:AddLogField("deed_log", "n_type", "int", 4)
	store:AddLogField("deed_log", "n_act_value", "int", 4)
	store:AddLogField("deed_log", "ip", "varchar", 16)
	store:AddLogField("deed_log", "mac", "varchar", 16)

    --角色传送日志
    store:AddLogTable("role_transmit_log")
    store:AddLogField("role_transmit_log", "time", "datetime", 8)
    store:AddLogField("role_transmit_log", "log_type", "int", 4)
    store:AddLogField("role_transmit_log", "ip", "varchar", 16)
    store:AddLogField("role_transmit_log", "mac", "varchar", 16)
    store:AddLogField("role_transmit_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_transmit_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_transmit_log", "cur_x", "decimal", 8)
    store:AddLogField("role_transmit_log", "cur_y", "decimal", 8)
    store:AddLogField("role_transmit_log", "cur_z", "decimal", 8)
    store:AddLogField("role_transmit_log", "target_scene_id", "int", 4)
    store:AddLogField("role_transmit_log", "target_x", "decimal", 8)
    store:AddLogField("role_transmit_log", "target_y", "decimal", 8)
    store:AddLogField("role_transmit_log", "target_z", "decimal", 8)

    --角色经验日志
    store:AddLogTable("role_experience_log")
    store:AddLogField("role_experience_log", "time", "datetime", 8)
    store:AddLogField("role_experience_log", "log_type", "int", 4)
    store:AddLogField("role_experience_log", "ip", "varchar", 16)
    store:AddLogField("role_experience_log", "mac", "varchar", 16)
    store:AddLogField("role_experience_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_experience_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_experience_log", "cur_x", "decimal", 8)
    store:AddLogField("role_experience_log", "cur_y", "decimal", 8)
    store:AddLogField("role_experience_log", "cur_z", "decimal", 8)
    store:AddLogField("role_experience_log", "add_exp", "int", 4)
    store:AddLogField("role_experience_log", "cur_level", "int", 4)
    store:AddLogField("role_experience_log", "cur_exp", "int", 4)
    store:AddLogField("role_experience_log", "comment", "nvarchar", 128)

    --角色物品增强日志
    store:AddLogTable("role_item_change_log")
    store:AddLogField("role_item_change_log", "time", "datetime", 8)
    store:AddLogField("role_item_change_log", "log_type", "int", 4)
    store:AddLogField("role_item_change_log", "ip", "varchar", 16)
    store:AddLogField("role_item_change_log", "mac", "varchar", 16)
    store:AddLogField("role_item_change_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_item_change_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_item_change_log", "cur_x", "decimal", 8)
    store:AddLogField("role_item_change_log", "cur_y", "decimal", 8)
    store:AddLogField("role_item_change_log", "cur_z", "decimal", 8)
    store:AddLogField("role_item_change_log", "item_config_id", "varchar", 32)
    store:AddLogField("role_item_change_log", "item_unique_id", "varchar", 32)
    store:AddLogField("role_item_change_log", "success", "int", 4)
    store:AddLogField("role_item_change_log", "consume_material", "varchar", 256)
    store:AddLogField("role_item_change_log", "comment", "nvarchar", 128)

    --角色登录日志
    store:AddLogTable("role_login_log")
    store:AddLogField("role_login_log", "account", "varchar", 64)
    store:AddLogField("role_login_log", "time", "datetime", 8)
    store:AddLogField("role_login_log", "log_type", "int", 4)
    store:AddLogField("role_login_log", "ip", "varchar", 16)
    store:AddLogField("role_login_log", "mac", "varchar", 16)
    store:AddLogField("role_login_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_login_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_login_log", "cur_x", "decimal", 8)
    store:AddLogField("role_login_log", "cur_y", "decimal", 8)
    store:AddLogField("role_login_log", "cur_z", "decimal", 8)
    store:AddLogField("role_login_log", "entry_time", "datetime", 8)
    store:AddLogField("role_login_log", "duration", "int", 4)

    --角色切场景日志
    store:AddLogTable("role_scene_log")
    store:AddLogField("role_scene_log", "time", "datetime", 8)
    store:AddLogField("role_scene_log", "log_type", "int", 4)
    store:AddLogField("role_scene_log", "ip", "varchar", 16)
    store:AddLogField("role_scene_log", "mac", "varchar", 16)
    store:AddLogField("role_scene_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_scene_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_scene_log", "cur_x", "decimal", 8)
    store:AddLogField("role_scene_log", "cur_y", "decimal", 8)
    store:AddLogField("role_scene_log", "cur_z", "decimal", 8)
    store:AddLogField("role_scene_log", "entry_time", "datetime", 8)
    store:AddLogField("role_scene_log", "duration", "int", 4)    

    --排行榜 日志
    store:AddLogTable("sys_rank_log")
    store:AddLogField("sys_rank_log", "time", "datetime", 8)
    store:AddLogField("sys_rank_log", "rank_name", "varchar", 64)
    store:AddLogField("sys_rank_log", "cur_No", "int", 4)
    store:AddLogField("sys_rank_log", "last_No", "int", 4)
    store:AddLogField("sys_rank_log", "role_uid", "varchar", 64)
    store:AddLogField("sys_rank_log", "role_name", "nvarchar", 64)
    store:AddLogField("sys_rank_log", "main_sort", "decimal", 8)
    store:AddLogField("sys_rank_log", "sub_sort", "decimal", 8)
    store:AddLogField("sys_rank_log", "camp_or_job", "int", 4)
    store:AddLogField("sys_rank_log", "guild", "nvarchar", 64)

    --角色任务日志
    store:AddLogTable("role_task_log")
    store:AddLogField("role_task_log", "time", "datetime", 8)
    store:AddLogField("role_task_log", "account", "varchar", 64)
    store:AddLogField("role_task_log", "log_type", "int", 4)
    store:AddLogField("role_task_log", "ip", "varchar", 16)
    store:AddLogField("role_task_log", "mac", "varchar", 16)
    store:AddLogField("role_task_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_task_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_task_log", "cur_x", "decimal", 8)
    store:AddLogField("role_task_log", "cur_y", "decimal", 8)
    store:AddLogField("role_task_log", "cur_z", "decimal", 8)
    store:AddLogField("role_task_log", "task_id", "int", 4)

	--公会 日志
    store:AddLogTable("role_guild_log")
    store:AddLogField("role_guild_log", "time", "datetime", 8)
    store:AddLogField("role_guild_log", "account", "varchar", 64)
    store:AddLogField("role_guild_log", "log_type", "int", 4)
    store:AddLogField("role_guild_log", "ip", "varchar", 16)
    store:AddLogField("role_guild_log", "mac", "varchar", 16)
    store:AddLogField("role_guild_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_guild_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_guild_log", "cur_x", "decimal", 8)
    store:AddLogField("role_guild_log", "cur_y", "decimal", 8)
    store:AddLogField("role_guild_log", "cur_z", "decimal", 8)
    store:AddLogField("role_guild_log", "target_name", "nvarchar", 64)
	store:AddLogField("role_guild_log", "guild_name", "nvarchar", 64)
	store:AddLogField("role_guild_log", "comment", "nvarchar", 128)

	--角色创建删除 日志
    store:AddLogTable("role_cre_del_log")
    store:AddLogField("role_cre_del_log", "time", "datetime", 8)
    store:AddLogField("role_cre_del_log", "log_type", "int", 4)
    store:AddLogField("role_cre_del_log", "account", "varchar", 64)
    store:AddLogField("role_cre_del_log", "role_name", "nvarchar", 64)

    --角色升级日志
    store:AddLogTable("role_upgrade_log")
    store:AddLogField("role_upgrade_log", "time", "datetime", 8)
    store:AddLogField("role_upgrade_log", "role", "nvarchar", 64)
    store:AddLogField("role_upgrade_log", "logtype", "int", 4)
    store:AddLogField("role_upgrade_log", "level_before", "int", 4)
    store:AddLogField("role_upgrade_log", "level_after", "int", 4)
    store:AddLogField("role_upgrade_log", "skill_id", "varchar", 32)
    store:AddLogField("role_upgrade_log", "account", "varchar", 64)

    --角色物品日志
    store:AddLogTable("role_item_log")
    store:AddLogField("role_item_log", "time", "datetime", 8)
    store:AddLogField("role_item_log", "log_type", "int", 4)
    store:AddLogField("role_item_log", "ip", "varchar", 16)
    store:AddLogField("role_item_log", "mac", "varchar", 16)
    store:AddLogField("role_item_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_item_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_item_log", "cur_x", "decimal", 8)
    store:AddLogField("role_item_log", "cur_y", "decimal", 8)
    store:AddLogField("role_item_log", "cur_z", "decimal", 8)
    store:AddLogField("role_item_log", "item_config_id", "varchar", 32)
    store:AddLogField("role_item_log", "item_unique_id", "varchar", 32)
    store:AddLogField("role_item_log", "amount", "decimal", 8)
    store:AddLogField("role_item_log", "comment", "nvarchar", 128)

    --角色拍卖行日志
    store:AddLogTable("role_auction_log")
    store:AddLogField("role_auction_log", "time", "datetime", 8)
    store:AddLogField("role_auction_log", "log_type", "int", 4)
    store:AddLogField("role_auction_log", "ip", "varchar", 16)
    store:AddLogField("role_auction_log", "mac", "varchar", 16)
    store:AddLogField("role_auction_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_auction_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_auction_log", "cur_x", "decimal", 8)
    store:AddLogField("role_auction_log", "cur_y", "decimal", 8)
    store:AddLogField("role_auction_log", "cur_z", "decimal", 8)
    store:AddLogField("role_auction_log", "item_config_id", "varchar", 32)
    store:AddLogField("role_auction_log", "item_unique_id", "varchar", 32)
    store:AddLogField("role_auction_log", "amount", "int", 4)
    store:AddLogField("role_auction_log", "money", "int", 4)
    store:AddLogField("role_auction_log", "comment", "nvarchar", 128)

	--玩法 日志
    store:AddLogTable("role_enjoy_log")
    store:AddLogField("role_enjoy_log", "time", "datetime", 8)
    store:AddLogField("role_enjoy_log", "account", "varchar", 64)
    store:AddLogField("role_enjoy_log", "log_type", "int", 4)
    store:AddLogField("role_enjoy_log", "ip", "varchar", 16)
    store:AddLogField("role_enjoy_log", "mac", "varchar", 16)
    store:AddLogField("role_enjoy_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_enjoy_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_enjoy_log", "cur_x", "decimal", 8)
    store:AddLogField("role_enjoy_log", "cur_y", "decimal", 8)
    store:AddLogField("role_enjoy_log", "cur_z", "decimal", 8)
    store:AddLogField("role_enjoy_log", "step_id", "int", 4)
	store:AddLogField("role_enjoy_log", "act_id", "varchar", 32)
	store:AddLogField("role_enjoy_log", "job", "int", 4)
	store:AddLogField("role_enjoy_log", "level", "int", 4)
	store:AddLogField("role_enjoy_log", "guild_name", "nvarchar", 64)
	store:AddLogField("role_enjoy_log", "comment", "nvarchar", 128)

	--非法消息 日志
    store:AddLogTable("role_illegal_req_log")
    store:AddLogField("role_illegal_req_log", "time", "datetime", 8)
    store:AddLogField("role_illegal_req_log", "log_type", "int", 4)
    store:AddLogField("role_illegal_req_log", "ip", "varchar", 16)
    store:AddLogField("role_illegal_req_log", "mac", "varchar", 16)
    store:AddLogField("role_illegal_req_log", "role_name", "nvarchar", 64)
    store:AddLogField("role_illegal_req_log", "cur_scene_id", "int", 4)
    store:AddLogField("role_illegal_req_log", "cur_x", "decimal", 8)
    store:AddLogField("role_illegal_req_log", "cur_y", "decimal", 8)
    store:AddLogField("role_illegal_req_log", "cur_z", "decimal", 8)
	store:AddLogField("role_illegal_req_log", "camp", "int", 4)
	store:AddLogField("role_illegal_req_log", "job", "int", 4)
	store:AddLogField("role_illegal_req_log", "level", "int", 4)
	store:AddLogField("role_illegal_req_log", "msg_id", "int", 4)
	store:AddLogField("role_illegal_req_log", "sub_msg_id", "int", 4)
	store:AddLogField("role_illegal_req_log", "comment", "nvarchar", 128)

	--游戏世界资金总量统计 日志
    store:AddLogTable("sys_total_capital_snap")
    store:AddLogField("sys_total_capital_snap", "time", "datetime", 8)
    store:AddLogField("sys_total_capital_snap", "log_type", "int", 4)
	store:AddLogField("sys_total_capital_snap", "tcrc_type", "int", 4)
	store:AddLogField("sys_total_capital_snap", "tcrc_switch", "int", 4)
	store:AddLogField("sys_total_capital_snap", "tcrc_sys_total", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "tcrc_can_buy", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "tcrc_can_prize", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "tcrc_user_cur", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "tcrc_offline_prize", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "tcrc_user_onway", "decimal", 8)
	store:AddLogField("sys_total_capital_snap", "comment", "nvarchar", 128)

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
	local member = nx_value("member")
	local public = nx_value("public")
    local auction = nx_value("auction")
	local entry = nx_value("entry")
	local store = nx_value("store")
	local logger = nx_value("logger")

	--关闭存储服务
	world:WriteLine("store server shutdown...")
	store:Stop()

    --关闭拍卖行服务
	world:WriteLine("auction server shutdown...")
	auction:Stop()

	--关闭公共数据服务
	world:WriteLine("public server shutdown...")
	public:Stop()

	--关闭接入服务
	world:WriteLine("entry server shutdown...")
	entry:Stop()

	--关闭场景服务
	world:WriteLine("member server shutdown...")
	member:Stop()

	--关闭游戏服务
	world:WriteLine("world server shutdown...")
	world:Stop()

	--关闭日志
	logger:Stop()

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

--发送回应信息
function echo_response(self, info)
	self:SendEchoLine(">" .. info .. "\r")
	self:WriteLine(info)
	return true
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
	elseif command == "perf" then
		--导出逻辑模块性能统计数据
		self:DumpPerformance("")
		echo_response(self, "dump logic module performance...")
	else
		echo_response(self, "undefined command: " .. command)
	end

	return 1
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
