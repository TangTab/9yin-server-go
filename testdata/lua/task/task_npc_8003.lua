-- 全局数据
local task_id = 8003
local text_id = "select_task_text"


-- npc载入
function on_load(self, sender)
end

-- 触发
function on_spring(npc, player)

	-- 判断玩家是否有该任务
	if not nx_find_record(player, "StateTaskRec") then
		return 0
	end
	
	local row = nx_find_record_value(player, "StateTaskRec", 0, nx_int(task_id))
	if row < 0 then
		return 0
	end
	
	-- 判断是否已经触发并选择过, 为0代表没选择过， 1正， 2邪， 3中立
	local flag = nx_query_record_value(player, "StateTaskRec", nx_int(row), 1)
	if nx_int(flag) > 0 then
		return 0
	end
	
	-- 弹出选择对话框: 服务器发往客户端的消息，消息格式 msgid, taskid, textid
	nx_custom(player, "show_task_select_dialog", task_id, text_id)

end