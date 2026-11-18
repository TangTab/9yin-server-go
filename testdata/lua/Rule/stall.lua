--------------------------------------------------------------------
-- File: Stall.lua 摆摊相关逻辑脚本
-- Author: 袁笑
-- Date: 2009.01.17
--------------------------------------------------------------------

-- 功能：在玩家初始化的时候对自身的摆摊相关属性进行初始化
-- 参数：玩家对象
function reset_player_stall_prop( self )

	-- 1 设置是否允许进行摆摊这个行为
	nx_set_property( self, "StallCan", 1 )

	
	-- 2 设置是否允许进行贩卖栏的相关操作
	nx_set_property( self, "StallCanSell", 1 )

	
	-- 3 设置是否允许进行收购栏的相关操作
	nx_set_property( self, "StallCanBuy", 1 )

	return 0
end

-- 功能：判断是否可以打开一个其他玩家的摊位
-- 参数：1 主角对象 2 摊主对象
function can_look_other_stall( role, stallowner )

	-- 如果摊主并没有摆摊
	local enState = nx_query_property( stallowner, "StallState" )
	if nx_int( enState) ~= nx_int( 2 ) then    
		-- 表示不是正在摆摊状态，不能查看
		return 1;
	end
	
	-- 如果主角在交易状态
	local bTrading = nx_query_property( role, "IsExchange" )
	if nx_int( bTrading ) ~= nx_int( 0 ) then
		-- 不能查看        
		--nx_msgbox( "如果主角在交易状态" )
		return 3;
	end
	
	-- 如果相距太远
	local fDistance = nx_distance( role, stallowner )
	if nx_int(fDistance) > nx_int(20) then
		-- 距离太远，不能查看  
		--nx_msgbox( "如果相距太远" )
		return 2;
	end
	
	-- 如果主角也在摆摊状态
	--local bStalling = nx_query_property( role, "StallState" )
	--if nx_int( bStalling ) ~= nx_int( 0 ) then
		-- 不能查看
	--	return 4;
	--end
	
	return 0;
end