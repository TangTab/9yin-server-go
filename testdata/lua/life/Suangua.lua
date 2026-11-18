----------------------------------------------------------------------
-- 文件名:		Suangua.lua
-- 内  容:		算卦相关判定脚本
-- 说  明:
-- 创建日期:	2009年11月10日
-- 创建人:		xuch
-- 版权所有:	苏州蜗牛电子有限公司
----------------------------------------------------------------------

require("define")
require("share\\itemtype_define")
--************************
--是否能够算卦
--************************
function can_Suangua(player, target)

	if not nx_exists(target) then
		--算卦目标不存在
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12500")
		return 0
	end
	
	if not (nx_query_property(target, "CanSuanguaFlag") == 1) then
		--目标不可被算卦
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12501")
		return 0
	end

	--player是否学习算卦技能,是否达到算卦等级
		-- 查找有没有这个技能
	local skill_type = "sh_gs";
	local therow = nx_find_record_value(player, "job_rec", 0, nx_string(skill_type))
	if therow < 0 then
		nx_sysinfo(player, 2, "11714", nx_string(skill_type))
		return 0
	end 
		-- 查询级别够不够
	local has_level = nx_query_record_value(player, "job_rec", therow, 1)
	local skill_level = nx_int(nx_query_property(target, "SuanguaLevel"))
	local job_name = nx_query_record_value(player, "job_rec", therow, 0)
		
	if has_level < skill_level then
		nx_sysinfo(player, 2, "11711", job_name, skill_level)
		return 0
	end	

	--距离判断
	local dist = nx_number(nx_distance3d(player, target))
	if dist > 3.0 then
		--距离太远，无法算卦
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12502")
		return 0
	end
	
	--是否在已经算卦过
	if nx_exist_Suanguarecord(player, target) then
		--算卦过了，无法再次算卦
		nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12503")
  	return 0;
  end
  
  --判断是否存在算卦工具
	local ToolBox = nx_get_child(player, nx_widestr("ToolBox"))	-- 获得人物装备栏
	-- 判断玩家装备栏里的物品是否是该工具
	local need_item_num = nx_container_get_itemcount(ToolBox, "sh_guijia")
	if need_item_num <= 0 then
		-- 必须装备工具提示
		nx_sysinfo(player, 2, "12504")
		return 0
	end  
	
	--在骑马时不能算卦
	if nx_find_property(player, "Mount") then
		local mount = nx_query_property(player, "Mount");
		if mount ~= nil and string.len(mount) > 0 then
			--出提示 正在骑马，无法进行算卦
			nx_sysinfo(player, 2, "12505");
			return 0;
		end
	end
  
	return 1	
end

--************************
--开始算卦
--************************
function begin_Suangua(player, target)
	--player做算卦动作
	local npc = nx_query_property(player, "LastObject")
	local action_name = nx_query_property(npc, "SuanguaAction")	
	--如果原来玩家出于战斗或者警戒状态，显示武器
--	if nx_int(nx_query_property(player, "LogicState"))==nx_int(2) or nx_int(nx_query_property(player, "LogicState"))==nx_int(1) then
		--以下"shenghuo_action"对应的消息为SERVER_CUSTOMMSG_SHENGHUO_ACTION
		--nx_custom_byken(player, "shenghuo_action", player, 0)	
--	end

	--8:LIFETOOL_SUANGUA 显示破碗
	nx_set_property(player, "LifeJobTool", 8)

	--播放动作
	nx_custom_stateaction(player, action_name)
	--目标做被算卦动作
	--nx_custom_byken(target, SERVER_CUSTOMMSG_ACTION, target, "beBag");
	
	--开始算卦
	nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12506")
	
	return 1
end

--************************
--结束（放弃）算卦
--************************
function end_Suangua(player, target)
	
	-- 设置动作
	local action_name = ""
	nx_custom_stateaction(player, action_name)
	
	--算卦停止
	nx_sysinfo(player, TIPSTYPE_TIPSINFO, "12507")
	
	--如果原来玩家出于战斗或者警戒状态，显示武器
--	if nx_int(nx_query_property(player, "LogicState"))==nx_int(2) or nx_int(nx_query_property(player, "LogicState"))==nx_int(1) then
		--以下"shenghuo_action"对应的消息为SERVER_CUSTOMMSG_SHENGHUO_ACTION
--		nx_custom_byken(player, "shenghuo_action", player, 1)	
--	end
	--0:LIFETOOL_NOTHING 显示武器
	nx_set_property(player, "LifeJobTool", 0)
	return 1;
end
--************************
--准备算卦（无用）
--************************
function on_prepare(player)
	return 1;
end

--************************
--算卦命中
--************************
function on_hit(player, target)
	--得到算卦NPC
	local npc = target
	if not nx_exists(npc) then
		return 0
	end	
	
	-- -- 体力限制
	local need_ene = nx_query_property(npc, "SuanguaNeedEne")
	local have_ene = nx_query_property(player, "Ene")
		
	if have_ene < need_ene then
		-- 提示体力不足
		nx_sysinfo(player, 2, "12508")
		-- 结束流程
		end_Suangua(player)
		return 0
	end
	
	-- 减少体力
	have_ene = have_ene - need_ene
	nx_set_property(player, "Ene", have_ene)
	nx_sysinfo(player, 2, "11712", need_ene)	
	
	-- 算卦经验值增加
	nx_process_life_exp_package( player, npc )

	-- 算卦掉落添加
	nx_process_life_drop_package( player, npc )


	--更新NPC数据，增加玩家算卦记录
	nx_updata_Suanguanpc(player, target);	


	--如果原来玩家出于战斗或者警戒状态，显示武器
--	if nx_int(nx_query_property(player, "LogicState"))==nx_int(2) or nx_int(nx_query_property(player, "LogicState"))==nx_int(1) then
		--以下"shenghuo_action"对应的消息为SERVER_CUSTOMMSG_SHENGHUO_ACTION
--		nx_custom_byken(player, "shenghuo_action", player, 1)	
--	end
	--0:LIFETOOL_NOTHING 显示武器
	nx_set_property(player, "LifeJobTool", 0)	
	
	-- 设置动作
	local action_name = ""
	nx_custom_stateaction(player, action_name)	
end
