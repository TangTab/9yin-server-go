---****************************************************************************
-- FILENAME	: movie\\after_movie_action.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: Mrkk
-- DATE		: 2010年4月8日
-------------------------------------------------------------------------------
-- DESC		: 关于 武当任务之"为民除恶"剧情2 的处理逻辑
---****************************************************************************

local npc_id_3 = "FunnpcClone007004";
local npc_cp_3 = 1;
local lifeTime = 600000
local complete_task = 2302;
local x1=579.421936
local y1=8.691605
local z1=757.395569
local o1=0.646116
--local accept_task_1 = 2303;
--local accept_task_2 = 2304;
-- 结束剧情后的特殊处理
function after_end_movie(player, movie_id)
	--设置任务完成
	local rows = nx_get_record_rows(player, "Task_Record");
	for i = 0, rows-1 do
		local id = nx_query_record_value(player, "Task_Record", i, 0);
		if id == complete_task then
			local total = nx_query_record_value(player, "Task_Record", i, 3);
			nx_set_record_value(player, "Task_Record", i, 2, total);
		end
	end
--创建萧千凝
	local npc = nx_null_obj();
	if nx_find_data(player, "task_event_npc") then
		npc = nx_query_data(player, "task_event_npc");
	end

	if not nx_exists(npc) then
		return;
	end

	--如果已经创建了,则不重复创建
	local obj_3 = nx_query_data(npc, "CreateObj_3");
	if nx_exists(obj_3) then
		return;
	end

	--创建
	--local x, y, z, o = nx_get_position(npc);
	--nx_sysinfo_broadcast( 1, "x,"..tostring(x).." y,"..tostring(y).." z,"..tostring(z) );
	obj_3 = nx_create_obj(nx_string(npc_id_3), npc_cp_3, x1, y1, z1, o1);

	--设置npc的父对象为触发npc,并设置触发npc的自对象为创建的怪物
	if nx_exists(obj_3) then
		nx_set_data(npc, "CreateObj_3", obj_3);
		nx_add_data(obj_3, "ParentNpc", 8);
		nx_set_data(obj_3, "ParentNpc", npc);
		nx_set_property(obj_3, "LifeTime", lifeTime);
	end

	--如果满足承接下一个任务的条件,则自动承接任务
--	if nx_test_accept_task(player, nx_null_obj(), accept_task_1) then
--		nx_accept_task(player, accept_task_1);
--	end

--		if nx_test_accept_task(player, nx_null_obj(), accept_task_2) then
--		nx_accept_task(player, accept_task_2);
--	end

	return 1
end
