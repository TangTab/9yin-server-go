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
function main(store)
	nx_set_value("store", store)

	--快速日志系统
	store.UseQuickLog = true
	--应用进程记录
	store.UseAppRecord = true

	--屏幕输出字符颜色
	store:SetConsoleColor("LightYellow")

	--读取程序配置
	local ini = nx_create("IniDocument")

	--配置文件名
	--ini.FileName = "game_store.ini"
	ini.FileName = nx_main_config()

	--加载配置
	if not ini:LoadFromFile() then
		nx_msgbox("can't read config file!")
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
	--记录数据库存取性能
	local log_db_perf = nx_boolean(ini:ReadString("Main", "LogDBPerformance", "false"))

	--内网地址
	local inner_addr = "127.0.0.1"
	local inner_port = 0
	--local inner_port = 16100 + store_id

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

	--唤醒连接地址
	local wake_addr = "127.0.0.1"
	local wake_port = 0
	--local wake_port = 16800 + store_id

	wake_addr = ini:ReadString("Wake", "Addr", wake_addr)
	wake_port = ini:ReadInteger("Wake", "Port", wake_port)

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

	--创建数据库系统
	local db_system

	if use_mysql then
		db_system = nx_create("MysqlDBSystem")
	else
		db_system = nx_create("SqlDBSystem")
	end

	--绑定脚本
	nx_bind_script(store, "main", "echo_init")

	store.AppName = "store_" .. nx_string(server_id) .. "_" .. nx_string(store_id)
	store.DistrictId = district_id
	store.ServerId = server_id
	store.StoreId = store_id
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
	store.InnerPort = inner_port
	store.BroadAddr = broad_addr
	store.BroadPort = broad_port
	store.EchoAddr = echo_addr
	store.EchoPort = echo_port
	store.WakeAddr = wake_addr
	store.WakePort = wake_port
	--记录数据库存取性能日志
	store.LogDBPerformance = log_db_perf
	--保证启动日志服务
	store.LogLauncher = "game_log game_log.ini daemon"

	init_custom_log(store)

	--启动存储服务
	store:Start()

	--实际使用的内网侦听端口
	inner_port = store.InnerPort
	--实际使用的唤醒侦听端口
	wake_port = store.WakePort

	--显示应用程序名
	store:WriteLine("app name " .. store.AppName)
	--显示通讯地址和端口
	store:WriteLine("echo socket " .. echo_addr .. ":" .. nx_string(echo_port))
	store:WriteLine("inner socket " .. inner_addr .. ":" .. nx_string(inner_port))
	store:WriteLine("broadcast socket " .. broad_addr .. ":" .. nx_string(broad_port))
	store:WriteLine("wake socket " .. wake_addr .. ":" .. nx_string(wake_port))
	--启动完成
	store:WriteLine("game server " .. nx_string(server_id) .. " store "
		.. nx_string(store_id) .. " started")

	--设置进程开启状态
	store:SetAppState(WAPS_OPENED);

	return 1
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
	local store = nx_value("store")

	--设置进程正在关闭状态
	--store:SetAppState(WAPS_CLOSING);

	--等待所有数据库操作完成
	--store:WriteLine("waiting all access finish...")

	--while not store:AccessEmpty() do
	--	nx_pause(0.0)
	--end

	--关闭存储服务
	store:WriteLine("store server shutdown...")
	store:Stop()

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
	self:SendEchoLine("welcome server " .. nx_string(self.ServerId)
		.. " store " .. nx_string(self.StoreId) .. "\r")
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
	elseif command == "end" then
		--关闭连接
		self:CloseEcho()
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
