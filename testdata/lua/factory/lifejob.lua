----------------------------------------------------------------------
-- 文件名:		lifejob.lua
-- 内  容:		生活职业相关判定脚本
-- 说  明:
-- 创建日期:	2011年10月27日
-- 创建人:		lihl
-- 版权所有:	苏州蜗牛电子有限公司
----------------------------------------------------------------------

--require("define")
--require("share\\pk_define")
--require("share\\itemtype_define")


-- 根据最后更新时间计算精力/疲劳值, 返回计算值和计算的时间戳
function energy(lasttime)
	curtime = os.time();
	lasttime = nx_int(lasttime);
	hour = os.difftime(curtime, lasttime) / 3600;
	value = math.floor(hour) * 12.5;

	return nx_int(value), os.difftime(curtime, os.difftime(curtime, lasttime)%3600);
end

-- 计算工作效率工式
function effect(player, factorylevel)
	--[[a= 0.032 b=0.678
	员工工作效率=
	玩家职业等级>工厂等级-5=(a*x^0.5+b)
	玩家职业等级<工厂等级-5=(a*x^0.5+b)-（等级差*0.015）
	X 代表当前的精力值	--]]
	factorylevel = nx_int(factorylevel);
	if factorylevel <= 0 then
		return 0;
	end

	a = 0.032;
	b=0.678;
	e = 0.5;
	d = 5;
	dx = 0.015;
	level = nx_int(nx_query_property(player, "Level"));
	x = nx_int(nx_query_property(player, "Energy"));
	diff = factorylevel - level;
	if diff >= d then
		return nx_double((a*x^e+b) - (diff * dx));
	else
		return nx_double(a*x^e+b);
	end	
end