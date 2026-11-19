package fssolostore

import "sync"

// ---- 官方自定义日志表（main.lua init_custom_log 全量 29 表逐字段定案；
// AddLogField 类型枚举 datetime/nvarchar/varchar/decimal/int）----

// LogField 一列定义（AddLogField(table, field, type, len)）。
type LogField struct {
	Name string
	Type string // "datetime" | "nvarchar" | "varchar" | "decimal" | "int"
	Size int
}

// LogTable 一张自定义日志表（AddLogTable；上限 32 表，入口 0x420B）。
type LogTable struct {
	Name   string
	Fields []LogField
}

// CustomLogTables main.lua init_custom_log 注册序（29 表）。
func CustomLogTables() []LogTable {
	f := func(name string, fs ...[3]any) LogTable {
		t := LogTable{Name: name}
		for _, x := range fs {
			t.Fields = append(t.Fields, LogField{Name: x[0].(string), Type: x[1].(string), Size: x[2].(int)})
		}
		return t
	}
	return []LogTable{
		f("item_log",
			[3]any{"time", "datetime", 8}, [3]any{"itemfrom", "nvarchar", 64},
			[3]any{"itemto", "nvarchar", 64}, [3]any{"itemid", "varchar", 32},
			[3]any{"instance_id", "varchar", 32}, [3]any{"item_num", "decimal", 8},
			[3]any{"logtype", "int", 4}, [3]any{"comment1", "nvarchar", 255},
			[3]any{"comment2", "nvarchar", 255}, [3]any{"comment3", "nvarchar", 255}),
		f("deal_log",
			[3]any{"tradeid", "varchar", 64}, [3]any{"time", "datetime", 8},
			[3]any{"buyer", "nvarchar", 64}, [3]any{"seller", "nvarchar", 64},
			[3]any{"itemid", "varchar", 32}, [3]any{"instance_id", "varchar", 32},
			[3]any{"count", "int", 4}, [3]any{"getorlost", "int", 4},
			[3]any{"tradechannel", "int", 4}),
		f("game_player_action_log",
			[3]any{"s_account", "varchar", 64}, [3]any{"s_role_name", "nvarchar", 64},
			[3]any{"d_create", "datetime", 8}, [3]any{"s_act_id", "varchar", 32},
			[3]any{"n_type", "int", 4}, [3]any{"n_act_value", "int", 4}),
		f("role_log",
			[3]any{"account", "varchar", 100}, [3]any{"role", "nvarchar", 64},
			[3]any{"logintime", "datetime", 8}, [3]any{"logouttime", "datetime", 8}),
		f("task_log",
			[3]any{"time", "datetime", 8}, [3]any{"role", "nvarchar", 64},
			[3]any{"task", "int", 4}, [3]any{"logtype", "int", 4}),
		f("instance_log",
			[3]any{"instanceid", "varchar", 32}, [3]any{"instancetypeid", "varchar", 32},
			[3]any{"starttime", "datetime", 8}, [3]any{"endtime", "datetime", 8},
			[3]any{"duration", "int", 4}, [3]any{"troop_count1", "int", 4},
			[3]any{"troop_count2", "int", 4}),
		f("instance_role_log",
			[3]any{"instanceid", "varchar", 32}, [3]any{"instancetypeid", "varchar", 32},
			[3]any{"entertime", "datetime", 8}, [3]any{"leavetime", "datetime", 8},
			[3]any{"duration", "int", 4}, [3]any{"role", "nvarchar", 64}),
		f("instance_npc_log",
			[3]any{"time", "datetime", 8}, [3]any{"npcname", "nvarchar", 64},
			[3]any{"sceneid", "int", 4}, [3]any{"location", "varchar", 32},
			[3]any{"role", "nvarchar", 64}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}),
		f("scene_log",
			[3]any{"sceneid", "int", 4}, [3]any{"date", "datetime", 8},
			[3]any{"enter", "int", 4}, [3]any{"totaltime", "decimal", 8}),
		f("punish_log",
			[3]any{"time", "datetime", 8}, [3]any{"operator", "varchar", 100},
			[3]any{"role", "nvarchar", 64}, [3]any{"minutes", "int", 4},
			[3]any{"type", "int", 4}),
		f("program_log",
			[3]any{"programid", "int", 4}, [3]any{"programname", "varchar", 64},
			[3]any{"opentime", "datetime", 8}, [3]any{"endtime", "datetime", 8}),
		f("total_capital_log",
			[3]any{"time", "datetime", 8}, [3]any{"type", "int", 4},
			[3]any{"res", "int", 4}, [3]any{"capital_type", "int", 4},
			[3]any{"inc_value", "decimal", 20}, [3]any{"old_value", "decimal", 20},
			[3]any{"new_value", "decimal", 20}, [3]any{"gm", "varchar", 128},
			[3]any{"comment", "nvarchar", 255}),
		f("player_exp_log",
			[3]any{"time", "datetime", 8}, [3]any{"role", "nvarchar", 64},
			[3]any{"logtype", "int", 4}, [3]any{"amount", "int", 4},
			[3]any{"comment", "nvarchar", 255}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}),
		f("deed_log",
			[3]any{"role", "nvarchar", 64}, [3]any{"d_create", "datetime", 8},
			[3]any{"s_act_id", "varchar", 32}, [3]any{"n_type", "int", 4},
			[3]any{"n_act_value", "int", 4}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}),
		f("role_transmit_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"target_scene_id", "int", 4},
			[3]any{"target_x", "decimal", 8}, [3]any{"target_y", "decimal", 8},
			[3]any{"target_z", "decimal", 8}),
		f("role_experience_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"add_exp", "int", 4},
			[3]any{"cur_level", "int", 4}, [3]any{"cur_exp", "int", 4},
			[3]any{"comment", "nvarchar", 128}),
		f("role_item_change_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"item_config_id", "varchar", 32},
			[3]any{"item_unique_id", "varchar", 32}, [3]any{"success", "int", 4},
			[3]any{"consume_material", "varchar", 256}, [3]any{"comment", "nvarchar", 128}),
		f("role_login_log",
			[3]any{"account", "varchar", 64}, [3]any{"time", "datetime", 8},
			[3]any{"log_type", "int", 4}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}, [3]any{"role_name", "nvarchar", 64},
			[3]any{"cur_scene_id", "int", 4}, [3]any{"cur_x", "decimal", 8},
			[3]any{"cur_y", "decimal", 8}, [3]any{"cur_z", "decimal", 8},
			[3]any{"entry_time", "datetime", 8}, [3]any{"duration", "int", 4}),
		f("role_scene_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"entry_time", "datetime", 8},
			[3]any{"duration", "int", 4}),
		f("sys_rank_log",
			[3]any{"time", "datetime", 8}, [3]any{"rank_name", "varchar", 64},
			[3]any{"cur_No", "int", 4}, [3]any{"last_No", "int", 4},
			[3]any{"role_uid", "varchar", 64}, [3]any{"role_name", "nvarchar", 64},
			[3]any{"main_sort", "decimal", 8}, [3]any{"sub_sort", "decimal", 8},
			[3]any{"camp_or_job", "int", 4}, [3]any{"guild", "nvarchar", 64}),
		f("role_task_log",
			[3]any{"time", "datetime", 8}, [3]any{"account", "varchar", 64},
			[3]any{"log_type", "int", 4}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}, [3]any{"role_name", "nvarchar", 64},
			[3]any{"cur_scene_id", "int", 4}, [3]any{"cur_x", "decimal", 8},
			[3]any{"cur_y", "decimal", 8}, [3]any{"cur_z", "decimal", 8},
			[3]any{"task_id", "int", 4}),
		f("role_guild_log",
			[3]any{"time", "datetime", 8}, [3]any{"account", "varchar", 64},
			[3]any{"log_type", "int", 4}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}, [3]any{"role_name", "nvarchar", 64},
			[3]any{"cur_scene_id", "int", 4}, [3]any{"cur_x", "decimal", 8},
			[3]any{"cur_y", "decimal", 8}, [3]any{"cur_z", "decimal", 8},
			[3]any{"target_name", "nvarchar", 64}, [3]any{"guild_name", "nvarchar", 64},
			[3]any{"comment", "nvarchar", 128}),
		f("role_cre_del_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"account", "varchar", 64}, [3]any{"role_name", "nvarchar", 64}),
		f("role_upgrade_log",
			[3]any{"time", "datetime", 8}, [3]any{"role", "nvarchar", 64},
			[3]any{"logtype", "int", 4}, [3]any{"level_before", "int", 4},
			[3]any{"level_after", "int", 4}, [3]any{"skill_id", "varchar", 32},
			[3]any{"account", "varchar", 64}),
		f("role_item_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"item_config_id", "varchar", 32},
			[3]any{"item_unique_id", "varchar", 32}, [3]any{"amount", "decimal", 8},
			[3]any{"comment", "nvarchar", 128}),
		f("role_auction_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"item_config_id", "varchar", 32},
			[3]any{"item_unique_id", "varchar", 32}, [3]any{"amount", "int", 4},
			[3]any{"money", "int", 4}, [3]any{"comment", "nvarchar", 128}),
		f("role_enjoy_log",
			[3]any{"time", "datetime", 8}, [3]any{"account", "varchar", 64},
			[3]any{"log_type", "int", 4}, [3]any{"ip", "varchar", 16},
			[3]any{"mac", "varchar", 16}, [3]any{"role_name", "nvarchar", 64},
			[3]any{"cur_scene_id", "int", 4}, [3]any{"cur_x", "decimal", 8},
			[3]any{"cur_y", "decimal", 8}, [3]any{"cur_z", "decimal", 8},
			[3]any{"step_id", "int", 4}, [3]any{"act_id", "varchar", 32},
			[3]any{"job", "int", 4}, [3]any{"level", "int", 4},
			[3]any{"guild_name", "nvarchar", 64}, [3]any{"comment", "nvarchar", 128}),
		f("role_illegal_req_log",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"ip", "varchar", 16}, [3]any{"mac", "varchar", 16},
			[3]any{"role_name", "nvarchar", 64}, [3]any{"cur_scene_id", "int", 4},
			[3]any{"cur_x", "decimal", 8}, [3]any{"cur_y", "decimal", 8},
			[3]any{"cur_z", "decimal", 8}, [3]any{"camp", "int", 4},
			[3]any{"job", "int", 4}, [3]any{"level", "int", 4},
			[3]any{"msg_id", "int", 4}, [3]any{"sub_msg_id", "int", 4},
			[3]any{"comment", "nvarchar", 128}),
		f("sys_total_capital_snap",
			[3]any{"time", "datetime", 8}, [3]any{"log_type", "int", 4},
			[3]any{"tcrc_type", "int", 4}, [3]any{"tcrc_switch", "int", 4},
			[3]any{"tcrc_sys_total", "decimal", 8}, [3]any{"tcrc_can_buy", "decimal", 8},
			[3]any{"tcrc_can_prize", "decimal", 8}, [3]any{"tcrc_user_cur", "decimal", 8},
			[3]any{"tcrc_offline_prize", "decimal", 8}, [3]any{"tcrc_user_onway", "decimal", 8},
			[3]any{"comment", "nvarchar", 128}),
	}
}

// LogTableSet 官方 log_table_set_t：32 表上限（"table too much"），
// 重名拒绝（"table already exists"），字段类型校验（AddLogField 断言族）。
type LogTableSet struct {
	mu     sync.Mutex
	tables []LogTable
}

// NewLogTableSet 注册官方 main.lua 全量 29 表。
func NewLogTableSet() *LogTableSet {
	s := &LogTableSet{}
	for _, t := range CustomLogTables() {
		s.AddTable(t.Name)
		for _, fd := range t.Fields {
			s.AddField(t.Name, fd.Name, fd.Type, fd.Size)
		}
	}
	return s
}

const logTableMax = 32 // 官方 AddLogTable 上限

var validFieldTypes = map[string]bool{
	"datetime": true, "nvarchar": true, "varchar": true, "decimal": true, "int": true,
}

// AddTable 官方 AddLogTable（重名 false；满 32 false）。
func (s *LogTableSet) AddTable(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tables {
		if t.Name == name {
			return false // table already exists
		}
	}
	if len(s.tables) >= logTableMax {
		return false // table too much
	}
	s.tables = append(s.tables, LogTable{Name: name})
	return true
}

// AddField 官方 AddLogField（表不存在/类型非法 false）。
func (s *LogTableSet) AddField(table, field, typ string, size int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validFieldTypes[typ] {
		return false // field type error
	}
	for i := range s.tables {
		if s.tables[i].Name == table {
			s.tables[i].Fields = append(s.tables[i].Fields,
				LogField{Name: field, Type: typ, Size: size})
			return true
		}
	}
	return false // table not found
}

// Table 查表（MessageCustomLog "table not define" 判据）。
func (s *LogTableSet) Table(name string) *LogTable {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.tables {
		if s.tables[i].Name == name {
			t := s.tables[i]
			return &t
		}
	}
	return nil
}

// Count 表数（呈现/测试）。
func (s *LogTableSet) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tables)
}
