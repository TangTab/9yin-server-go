---****************************************************************************
-- FILENAME	: movie\\after_movie_action.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: Mrkk
-- DATE		: 2010年4月8日
-------------------------------------------------------------------------------
-- DESC		: 关于 武当任务之"为民除恶"剧情1 的处理逻辑
---****************************************************************************
local npc_id_1 = "BossClone007004";
local npc_id_2 = "BossClone007005";
local npc_cp_1 = 1;
local npc_cp_2 = 1;
local x1 = 580.352356;
local y1 = 8.691605;
local z1 = 766.366699;
local o1 = 1.214432;
local x2 = 580.180725;
local y2 = 8.73611;
local z2 = 770.398071;
local o2 = 3.101836;
local lifeTime = 600000;
-- 结束剧情后的特殊处理
function after_end_movie(player, npc, movie_id)
	--获得玩家身上保存的触发npc
	local npc = nx_null_obj();
	if nx_find_data(player, "task_event_npc") then
		npc = nx_query_data(player, "task_event_npc");
	end

	if not nx_exists(npc) then
		return;
	end

	--如果已经创建了怪物,则不重复创建
	local obj_1 = nx_query_data(npc, "CreateObj_1");
	local obj_2 = nx_query_data(npc, "CreateObj_2");
	if nx_exists(obj_1) or nx_exists(obj_2) then
		return;
	end

	--创建两个怪物
	--local x, y, z, o = nx_get_position(npc);
	--nx_sysinfo_broadcast( 1, "x,"..tostring(x).." y,"..tostring(y).." z,"..tostring(z) );
	obj_1 = nx_create_obj(nx_string(npc_id_1), npc_cp_1, x1, y1, z1, o1);
	obj_2 = nx_create_obj(nx_string(npc_id_2), npc_cp_2, x2, y2, z2, o2);

	--设置怪物的父对象为触发npc,并设置触发npc的自对象为创建的怪物
	if nx_exists(obj_1) then
		nx_set_data(npc, "CreateObj_1", obj_1);
		nx_add_data(obj_1, "ParentNpc", 8);
		nx_set_property(obj_1, "LifeTime", lifeTime);
		nx_set_data(obj_1, "ParentNpc", npc);
	end

	if nx_exists(obj_2) then
		nx_set_data(npc, "CreateObj_2", obj_2);
		nx_add_data(obj_2, "ParentNpc", 8);
		nx_set_property(obj_2, "LifeTime", lifeTime);
		nx_set_data(obj_2, "ParentNpc", npc);
	end
	--nx_msgbox("end")
	return 1
end
