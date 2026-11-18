----------------------------------------------------------------------
-- 文件名:		compose.lua
-- 内  容:		合成相关判定脚本
-- 说  明:
-- 创建日期:	2009年8月20日
-- 创建人:		杨利平
-- 版权所有:	苏州蜗牛电子有限公司
----------------------------------------------------------------------
--
require("define");

local table_compose = {
	["sh_kg"] = 0,
	["sh_yf"] = 1,
	["sh_qf"] = 2,
	["sh_nf"] = 3,
	["sh_lh"] = 4,
	["sh_tj"] = 5,
	["sh_cf"] = 6,
	["sh_jq"] = 7,
	["sh_ys"] = 8,
	["sh_ds"] = 9,
	["sh_cs"] = 10,
	["sh_qs"] = 11,
	["sh_qis"] = 12,
	["sh_sh"] = 13,
	["sh_hs"] = 14,
	["sh_gs"] = 15,
	["sh_qg"] = 16,
}

--检查当前状态下是否可以合成
function can_compose(player, formula_name)

    -- 当前是否在干其它事情，无法进行合成
	if is_doing_other_thing(player) == 1 then
		return 0
	end

	-- 查找配方表，找不到返回0
	if not nx_find_record(player, "FormulaRec") then
		--出提示，出提示：人物身上未找到配方表，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1800");
		return 0;
	end

	if -1 ==  nx_find_record_value(player, "FormulaRec", 1, formula_name) then
		-- 找不到,出提示:还未学习该配方,无法进行合成。
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1801");
		return 0
	end

    -- 4.	检查是当前是否附合合成所需环境（如打铁，是否在打铁	台范围内）
	-- 获取配方职业
	local profession = nx_get_formula_profession(formula_name);

	if profession == nil or profession == "" then
		-- 找不到,出提示:还未学习该配方,无法进行合成。
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1801");
		return 0;
	end

--[[
	--4种合成不需范围限制,其它的需要判断合成标志，看是否在合成区域内
	if profession == "sh_kg" or profession == "sh_tj" or  profession == "sh_jq" or profession == "sh_cs"then
		-- 通过职业获取合成类型标志位
		local compose_type = table_compose[profession];
		local result = nx_get_compose_flag(player, compose_type)
		if result == nil or result == 0 then
			--出提示：不在可合成范围内，无法进行合成
			--nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1812");
			if profession == "sh_kg" then--矿工
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1830")
			elseif profession == "sh_tj" then--铁匠
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1831")
			elseif profession == "sh_jq" then--金匠
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1834")
			elseif profession == "sh_cs" then--厨师
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1835")
			end
			return 0;
		end
	end
--]]

    -- 5.	检查体力值是否满足此次合成所需值
	local need_ene = nx_get_formula_ene(formula_name)
	local have_ene = nx_query_property(player, "Ene")
	
	if need_ene == nil or have_ene == nil or need_ene > have_ene then
		--出提示：您当前体力值不够合成所需，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1802");
		return 0;
	end

	-- 检查气力值是否满足此次合成所需值
	local need_act = nx_get_formula_act(formula_name)
	local have_act = nx_query_property(player, "Act")
	
	if need_act == nil or have_act == nil or need_act > have_act then
		--出提示：您当前气力值不够合成所需，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1837");
		return 0;
	end
	
    -- 6.	检查背包中的材料，是否满足合成所需
	if 0 == nx_is_have_all_material(player, formula_name) then
		--出提示：您当前的材料不够，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1813");
		return 0;
	end

  --7.	检查假设合成成功后东西能否放入背包
	if 0 == nx_is_have_space_for_compose(player, formula_name) then
		--出提示：背包空间不足,无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1814");
		return 0
	end

	return 1;
end

--是否正在处理事情不能合成
function is_doing_other_thing(player)

	if nx_find_data(player, "IsCompose") then
	    local value = nx_query_data(player, "IsCompose");

	    if value == 1 then
	        --出提示:正在合成,请稍候
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1806");
	        return 1
        end
    end

	local dead = nx_query_property(player, "Dead")

	if dead == 1 then
		--出提示，死亡状态，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1805");
		return 1
	end

	local exchange = nx_query_property(player, "IsExchange")

	if exchange == 1 then
		--出提示 正在交易，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1803");
		return 1
	end

	local stall = nx_query_property(player, "StallState")

	if stall > 0 then
		--出提示 正在摆摊，无法进行合成
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1804");
		return 1
	end

	--在坐马车时不能摆摊
	if nx_find_property(player, "OnTransToolState") then
		local transstate = nx_query_property(player, "OnTransToolState");

		if transstate ~= nil and transstate > 0 then
			--出提示 坐马车时，无法进行合成
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1810");
			return 1;
		end
	end

	--在骑马时不能合成
	if nx_find_property(player, "Mount") then
		local mount = nx_query_property(player, "Mount");

		if mount ~= nil and string.len(mount) > 0 then
			--出提示 正在骑马，无法进行合成
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1811");
			return 1;
		end
	end

	return 0
end

--检查是事可以学习配方
function can_learn_formula(player, formula_name)

	--查找职业是否符合配方要求
	if not nx_find_record(player, "job_rec") then
		return 0;
	end

	local need_profession = nx_get_formula_profession(formula_name)
	local need_level = nx_get_formula_level(formula_name)

    --表格定义job_rec
    -- 0列 职业id 1列 职业等级

    -- 查找表中职业对应行
	local row = nx_find_record_value(player, "job_rec", 0, need_profession)

	if row == nil or row == -1 then
		--未找到,出提示:您还未学习{@0:职业名}职业，无法学习该职业配方。
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1821", need_profession);
		return 0;
	end

	local cur_level = nx_query_record_value(player, "job_rec", row, 1)

	if need_level > cur_level then
		--职业等级不够，出提示:您当前的职业等级未达{@0:ToInt级数}级，无法学习该配方。�
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1822", need_profession,nx_int(need_level));
		return 0;
	end

	--查找配方表，找不到返回0
	if not nx_find_record(player, "FormulaRec") then
		--找不到配方表，出提示：人物身上未找到配方表
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1823");
		return 0;
	end

    --配方表第0列为配方职业，第1列为配方名
    -- 查找是否已学过此配方
	local row = nx_find_record_value(player, "FormulaRec", 1, formula_name)

	if row ~= nil and row > -1 then
		--找到，出提示：您已学习过该配方，无需再次学习。
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1824", need_profession);
		return 0;
	end

	return 1;
end

