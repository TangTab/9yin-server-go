---****************************************************************************
-- FILENAME	: movie\\after_movie_action.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: yurc
-- DATE		: 2010年4月14日
-------------------------------------------------------------------------------
-- DESC		: 播放完 村长出现的动画后，创建村长的逻辑
---****************************************************************************
local npc_id_1 = "FunnpcClone007013";
local npc_cp_1 = 1;
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
	if nx_exists(obj_1) then
		return;
	end

	--创建两个怪物
	local x, y, z, o = nx_get_position(npc);
	--nx_sysinfo_broadcast( 1, "x,"..tostring(x).." y,"..tostring(y).." z,"..tostring(z) );
	obj_1 = nx_create_obj(nx_string(npc_id_1), npc_cp_1, x, y, z, o);
	
	--设置怪物的父对象为触发npc,并设置触发npc的自对象为创建的怪物
	if nx_exists(obj_1) then
		nx_set_data(npc, "CreateObj_1", obj_1);
		nx_add_data(obj_1, "ParentNpc", 8);
		nx_set_data(obj_1, "ParentNpc", npc);
	end
	--nx_msgbox("end")
	return 1
end