----------------------------------------------------------------------
-- ÎÄ¼şÃû:		compose.lua
-- ÄÚ  Èİ:		ºÏ³ÉÏà¹ØÅĞ¶¨½Å±¾
-- Ëµ  Ã÷:
-- ´´½¨ÈÕÆÚ:	2009Äê8ÔÂ20ÈÕ
-- ´´½¨ÈË:		ÑîÀûÆ½
-- °æÈ¨ËùÓĞ:	ËÕÖİÎÏÅ£µç×ÓÓĞÏŞ¹«Ë¾
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

--¼ì²éµ±Ç°×´Ì¬ÏÂÊÇ·ñ¿ÉÒÔºÏ³É
function can_compose(player, formula_name)

    -- µ±Ç°ÊÇ·ñÔÚ¸ÉÆäËüÊÂÇé£¬ÎŞ·¨½øĞĞºÏ³É
	if is_doing_other_thing(player) == 1 then
		return 0
	end

	-- ²éÕÒÅä·½±í£¬ÕÒ²»µ½·µ»Ø0
	if not nx_find_record(player, "FormulaRec") then
		--³öÌáÊ¾£¬³öÌáÊ¾£ºÈËÎïÉíÉÏÎ´ÕÒµ½Åä·½±í£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1800");
		return 0;
	end

	if -1 ==  nx_find_record_value(player, "FormulaRec", 1, formula_name) then
		-- ÕÒ²»µ½,³öÌáÊ¾:»¹Î´Ñ§Ï°¸ÃÅä·½,ÎŞ·¨½øĞĞºÏ³É¡£
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1801");
		return 0
	end

    -- 4.	¼ì²éÊÇµ±Ç°ÊÇ·ñ¸½ºÏºÏ³ÉËùĞè»·¾³£¨Èç´òÌú£¬ÊÇ·ñÔÚ´òÌú	Ì¨·¶Î§ÄÚ£©
	-- »ñÈ¡Åä·½Ö°Òµ
	local profession = nx_get_formula_profession(formula_name);

	if profession == nil or profession == "" then
		-- ÕÒ²»µ½,³öÌáÊ¾:»¹Î´Ñ§Ï°¸ÃÅä·½,ÎŞ·¨½øĞĞºÏ³É¡£
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1801");
		return 0;
	end

--[[
	--4ÖÖºÏ³É²»Ğè·¶Î§ÏŞÖÆ,ÆäËüµÄĞèÒªÅĞ¶ÏºÏ³É±êÖ¾£¬¿´ÊÇ·ñÔÚºÏ³ÉÇøÓòÄÚ
	if profession == "sh_kg" or profession == "sh_tj" or  profession == "sh_jq" or profession == "sh_cs"then
		-- Í¨¹ıÖ°Òµ»ñÈ¡ºÏ³ÉÀàĞÍ±êÖ¾Î»
		local compose_type = table_compose[profession];
		local result = nx_get_compose_flag(player, compose_type)
		if result == nil or result == 0 then
			--³öÌáÊ¾£º²»ÔÚ¿ÉºÏ³É·¶Î§ÄÚ£¬ÎŞ·¨½øĞĞºÏ³É
			--nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1812");
			if profession == "sh_kg" then--¿ó¹¤
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1830")
			elseif profession == "sh_tj" then--Ìú½³
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1831")
			elseif profession == "sh_jq" then--½ğ½³
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1834")
			elseif profession == "sh_cs" then--³øÊ¦
				nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1835")
			end
			return 0;
		end
	end
--]]

    -- 5.	¼ì²éÌåÁ¦ÖµÊÇ·ñÂú×ã´Ë´ÎºÏ³ÉËùĞèÖµ
	local need_ene = nx_get_formula_ene(formula_name)
	local have_ene = nx_query_property(player, "Ene")
	
	if need_ene == nil or have_ene == nil or need_ene > have_ene then
		--³öÌáÊ¾£ºÄúµ±Ç°ÌåÁ¦Öµ²»¹»ºÏ³ÉËùĞè£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1802");
		return 0;
	end

	-- ¼ì²éÆøÁ¦ÖµÊÇ·ñÂú×ã´Ë´ÎºÏ³ÉËùĞèÖµ
	local need_act = nx_get_formula_act(formula_name)
	local have_act = nx_query_property(player, "Act")
	
	if need_act == nil or have_act == nil or need_act > have_act then
		--³öÌáÊ¾£ºÄúµ±Ç°ÆøÁ¦Öµ²»¹»ºÏ³ÉËùĞè£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1837");
		return 0;
	end
	
    -- 6.	¼ì²é±³°üÖĞµÄ²ÄÁÏ£¬ÊÇ·ñÂú×ãºÏ³ÉËùĞè
	if 0 == nx_is_have_all_material(player, formula_name) then
		--³öÌáÊ¾£ºÄúµ±Ç°µÄ²ÄÁÏ²»¹»£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1813");
		return 0;
	end

  --7.	¼ì²é¼ÙÉèºÏ³É³É¹¦ºó¶«Î÷ÄÜ·ñ·ÅÈë±³°ü
	if 0 == nx_is_have_space_for_compose(player, formula_name) then
		--³öÌáÊ¾£º±³°ü¿Õ¼ä²»×ã,ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1814");
		return 0
	end

	return 1;
end

--ÊÇ·ñÕıÔÚ´¦ÀíÊÂÇé²»ÄÜºÏ³É
function is_doing_other_thing(player)

	if nx_find_data(player, "IsCompose") then
	    local value = nx_query_data(player, "IsCompose");

	    if value == 1 then
	        --³öÌáÊ¾:ÕıÔÚºÏ³É,ÇëÉÔºò
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1806");
	        return 1
        end
    end

	local dead = nx_query_property(player, "Dead")

	if dead == 1 then
		--³öÌáÊ¾£¬ËÀÍö×´Ì¬£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1805");
		return 1
	end

	local exchange = nx_query_property(player, "IsExchange")

	if exchange == 1 then
		--³öÌáÊ¾ ÕıÔÚ½»Ò×£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1803");
		return 1
	end

	local stall = nx_query_property(player, "StallState")

	if stall > 0 then
		--³öÌáÊ¾ ÕıÔÚ°ÚÌ¯£¬ÎŞ·¨½øĞĞºÏ³É
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1804");
		return 1
	end

	--ÔÚ×øÂí³µÊ±²»ÄÜ°ÚÌ¯
	if nx_find_property(player, "OnTransToolState") then
		local transstate = nx_query_property(player, "OnTransToolState");

		if transstate ~= nil and transstate > 0 then
			--³öÌáÊ¾ ×øÂí³µÊ±£¬ÎŞ·¨½øĞĞºÏ³É
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1810");
			return 1;
		end
	end

	--ÔÚÆïÂíÊ±²»ÄÜºÏ³É
	if nx_find_property(player, "Mount") then
		local mount = nx_query_property(player, "Mount");

		if mount ~= nil and string.len(mount) > 0 then
			--³öÌáÊ¾ ÕıÔÚÆïÂí£¬ÎŞ·¨½øĞĞºÏ³É
			nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1811");
			return 1;
		end
	end

	return 0
end

--¼ì²éÊÇÊÂ¿ÉÒÔÑ§Ï°Åä·½
function can_learn_formula(player, formula_name)

	--²éÕÒÖ°ÒµÊÇ·ñ·ûºÏÅä·½ÒªÇó
	if not nx_find_record(player, "job_rec") then
		return 0;
	end

	local need_profession = nx_get_formula_profession(formula_name)
	local need_level = nx_get_formula_level(formula_name)

    --±í¸ñ¶¨Òåjob_rec
    -- 0ÁĞ Ö°Òµid 1ÁĞ Ö°ÒµµÈ¼¶

    -- ²éÕÒ±íÖĞÖ°Òµ¶ÔÓ¦ĞĞ
	local row = nx_find_record_value(player, "job_rec", 0, need_profession)

	if row == nil or row == -1 then
		--Î´ÕÒµ½,³öÌáÊ¾:Äú»¹Î´Ñ§Ï°{@0:Ö°ÒµÃû}Ö°Òµ£¬ÎŞ·¨Ñ§Ï°¸ÃÖ°ÒµÅä·½¡£
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1821", need_profession);
		return 0;
	end

	local cur_level = nx_query_record_value(player, "job_rec", row, 1)

	if need_level > cur_level then
		--Ö°ÒµµÈ¼¶²»¹»£¬³öÌáÊ¾:Äúµ±Ç°µÄÖ°ÒµµÈ¼¶Î´´ï{@0:ToInt¼¶Êı}¼¶£¬ÎŞ·¨Ñ§Ï°¸ÃÅä·½¡££
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1822", need_profession,nx_int(need_level));
		return 0;
	end

	--²éÕÒÅä·½±í£¬ÕÒ²»µ½·µ»Ø0
	if not nx_find_record(player, "FormulaRec") then
		--ÕÒ²»µ½Åä·½±í£¬³öÌáÊ¾£ºÈËÎïÉíÉÏÎ´ÕÒµ½Åä·½±í
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1823");
		return 0;
	end

    --Åä·½±íµÚ0ÁĞÎªÅä·½Ö°Òµ£¬µÚ1ÁĞÎªÅä·½Ãû
    -- ²éÕÒÊÇ·ñÒÑÑ§¹ı´ËÅä·½
	local row = nx_find_record_value(player, "FormulaRec", 1, formula_name)

	if row ~= nil and row > -1 then
		--ÕÒµ½£¬³öÌáÊ¾£ºÄúÒÑÑ§Ï°¹ı¸ÃÅä·½£¬ÎŞĞèÔÙ´ÎÑ§Ï°¡£
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "1824", need_profession);
		return 0;
	end

	return 1;
end

