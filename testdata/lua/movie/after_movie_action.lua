---****************************************************************************
-- FILENAME	: movie\\after_movie_action.lua
-- COPYRIGHT: Snail Game
-- AUTHOR	: Mrkk
-- DATE		: 2010年3月15日
-------------------------------------------------------------------------------
-- DESC		: 关于 剧情播放前后的特殊事件 的处理逻辑
---****************************************************************************

--根据剧情编号定义:玩家醒来后，在另外的指定坐
local g_moveto = {
	[22] = { x = 478.71, y = -43.02, z = 724.49, },	--茶林遇险
};

-- 开始剧情后的特殊处理
function after_custom_movie(player, movie_id)
	--画面一黑，玩家醒来后，在另外的指定坐
	if g_moveto[movie_id] ~= nil then
		local x = g_moveto[movie_id].x;
		local y = g_moveto[movie_id].y;
		local z = g_moveto[movie_id].z;
		--nx_msgbox("go");
		--如果目标位置可行走,则传送过去
		local bCanWalk = nx_can_walk(nx_float(x), nx_float(z));
		local b = (x ~= nil and y ~= nil and z ~= nil);
		if b and bCanWalk then
			local bRet = nx_moveto( player, nx_float(x),  nx_float(y),  nx_float(z), nx_float(6.15) );
		end
		return 1;
	end
	return 0
end