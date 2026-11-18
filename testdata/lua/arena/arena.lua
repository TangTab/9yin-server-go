----------------------------------------------------------------------
-- 文件名:		arena.lua
-- 内  容:		竞技场相关判定脚本
-- 说  明:
-- 创建日期:	2012年10月09日
-- 创建人:		张肖峰
-- 版权所有:	苏州蜗牛电子有限公司
----------------------------------------------------------------------

-- 竞技系统积分计算
-- 参数:胜利方场景内队伍号 A队积分 B队积分
function score_calculate(win_team, match_score_a, match_score_b)
	local ARENAPLANE_TEAM_A = 1		--A队
	local ARENAPLANE_TEAM_B = 2		--B队
	local basescore = 48
	local basepara1 = 10
	local basepara2 = 400
	local rate_a = 1 / ( 1 + basepara1 ^ ((match_score_b - match_score_a) / basepara2))
	local rate_b = 1 / ( 1 + basepara1 ^ ((match_score_a - match_score_b) / basepara2))
	local result = 0
	if nx_number(win_team) == nx_number(ARENAPLANE_TEAM_A) then
		result = nx_int(basescore * (1 - rate_a) + 0.5)
		return result, -result
	elseif nx_number(win_team) == nx_number(ARENAPLANE_TEAM_B) then
		result = nx_int(basescore * (1 - rate_b) + 0.5)
		return -result, result
	end
	return 0, 0
end

-- 竞技场开放时间计算
-- 参数：当前年月日周小时分钟
function arena_canopen()
	--利用datetable表获得对应日期数据(year,month,day,yday,wday,hour,min,sec,isdst)
	local datetable = os.date("*t")
	
	if datetable.wday - 1 == 3 then
		return nx_int(1)
	end
	
	if datetable.wday - 1 == 0 then
		if datetable.hour < 21 then
			return nx_int(1)
		end
	end

	return nx_int(0)
end