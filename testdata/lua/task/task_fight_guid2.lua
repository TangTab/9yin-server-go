-----------------------------------------
-- @ 脚本名称: task\\task_fight_guid.lua
-- @ 脚本功能: 战斗教学
-- @ 创建者:   mrkk
-- @ 创建日期: 2010年4月20日
-----------------------------------------
require("task\\task_define")

local g_track_info = {
[1] = {track = "jiaoxue_real", key_word = "Qiecuo_jiaoxue_real",},
[2] = {track = "jiaoxue_xu", key_word = "Qiecuo_jiaoxue_xu",},
[3] = {track = "jiaoxue_jia", key_word = "Qiecuo_jiaoxue_jia",},
}

-- 副本场景编号
local scene_id = 0

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end

-- 任务完成条件
function Task_Complete_Condition(player, npc, taskid)
	return 1
end

-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 进度的个数
	local len = table.getn(g_track_info)

	-- 进度表添加
	for i = 1, len do
		nx_add_record_row_values(player, "Task_Record", nx_int(taskid), subtype_hunter, nx_int(0), nx_int(i), nx_int(0), nx_int(1), g_track_info[i].key_word, "55", g_track_info[i].track, nx_int(0), g_track_info[i].key_word, nx_int64(0))
	end

	-- 修改接受表的顺序字段
	local row = nx_find_record_value(player, "Task_Accepted", 0, taskid)
	if row >= 0 then
		nx_set_record_value(player, "Task_Accepted", row, 8, 1)
	end

	-- 直接进入副本
	nx_switch_to_subscene(player, nx_int(scene_id))
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
  --nx_msgbox("task complete")
  --删除限制圈，清理创建的对象
	--nx_custom(player, nx_string("task_fight_guid_msg"), nx_int(2));
  --del_created_npc(player, taskid)
  return 1
end

-- 任务放弃
function on_task_giveup(self, task_id)
  --删除限制圈，清理创建的对象
  --nx_custom(self, nx_string("task_fight_guid_msg"), nx_int(2));
  --del_created_npc(self, task_id)
  return 1;
end

-- 任务失败
function on_task_failed(self, task_id)
  --删除限制圈，清理创建的对象
  --nx_custom(self, nx_string("task_fight_guid_msg"), nx_int(2));
  --del_created_npc(self, task_id)
end

function del_created_npc(self, task)
  --local row = nx_find_record_value(self, "Task_Addition_Rec", 0, task)
  --while row >= 0 do
	--local scene = nx_query_record_value(self, "Task_Addition_Rec", row, 1)
	--local obj = nx_query_record_value(self, "Task_Addition_Rec", row, 2);
	--nx_command_toscene(nx_int(scene), nx_int(168), 1, obj);
	--if nx_remove_record_row(self, "Task_Addition_Rec", row) then
	--	row = nx_find_record_value(self, "Task_Addition_Rec", 0, task)
	--else
	--	return;
	--end
  --end
end


