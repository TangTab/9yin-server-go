---****************************************************************************
-- FILENAME	: movie\\after_movie_action.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: yrc
-- DATE		: 2010年4月8日
-------------------------------------------------------------------------------
-- DESC		: 关于 武当任务之"滕雷闯山"剧情 的处理逻辑
---****************************************************************************

local x = 526.76;
local y = 109.68;
local z = 465.52;
local o = 4.7;
local complete_task = 2307;
-- 结束剧情后的特殊处理
function after_end_movie(player, movie_id)
	--切换到本场景的指定位置
	nx_moveto(player, x, y, z, o);
	--设置任务完成
	local rows = nx_get_record_rows(player, "Task_Record");
	for i = 0, rows-1 do
		local id = nx_query_record_value(player, "Task_Record", i, 0);
		if id == complete_task then
			local total = nx_query_record_value(player, "Task_Record", i, 3);
			nx_set_record_value(player, "Task_Record", i, 2, total);
		end
	end
	return 1
end