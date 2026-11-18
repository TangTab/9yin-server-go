require ("define")
require ("skill\\skill_error_define")

local PKMODEL_PEACE = 0; --和平模式
local PKMODEL_NATION = 1; --国家模式
local PKMODEL_WHOLE = 2; --全体模式
local PKMODEL_TEAM = 3; --组队模式
local PKMODEL_GUILD = 4; --公会模式
local PKMODEL_FAMILY = 5; --家族模式
local PKMODEL_EVIL = 6; --善恶模式
local PKMODEL_UNION = 7; --盟国模式

--技能错误报告脚本
--冯红文
function skill_report_error(self, skillid, target, error)

	if error == USESKILL_RESULT_ERR_TARGETDEAD or 
		error == USESKILL_RESULT_ERR_FUNCLIMIT or 
		error == USESKILL_RESULT_ERR_PROPERTY or 
		error == USESKILL_RESULT_ERR_EQUIPLIMIT or 
		error == USESKILL_RESULT_ERR_OTHERSKILL_ISHITING then
		return;--不需要提示
	end
	
		
	-- 普通攻击不进行错误提示，【提示的频率太频繁】
	if string.find(nx_string(skillid),"normal") ~= nil then
		return
	end	
	
	if error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_ENIMY then
		--报告错误信息
		if nx_exists(target) and nx_type(target) == TYPE_PLAYER then
			--检查场景是否允许PK
			local scene_cantpk = nx_query_property(nx_get_scene(), "CantPK");
			if scene_cantpk > 0 then
				--当前场景不允许PK
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8115");
				return;
			end
			
			--检查玩家的PK模式
			local self_pkmode = nx_query_property(self, "PKMode");
			if self_pkmode == PKMODEL_NATION then --国家模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8100");
			elseif self_pkmode == PKMODEL_PEACE then --和平模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8101");
			elseif self_pkmode == PKMODEL_WHOLE then --全体模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8102");
			elseif self_pkmode == PKMODEL_TEAM then --组队模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8103");
			elseif self_pkmode == PKMODEL_GUILD then --公会模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8104");
			elseif self_pkmode == PKMODEL_FAMILY then --家族模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8105");
			elseif self_pkmode == PKMODEL_EVIL then --善恶模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8106");
			elseif self_pkmode == PKMODEL_UNION then --盟国模式
				nx_sysinfo(self, TIPSTYPE_TIPSINFO, "8107");
			end
			return;
		else
			return;
		end	
	end
	
	local stringid = "" .. 9000 + error;
	nx_sysinfo(self, TIPSTYPE_TIPSINFO, stringid);
	
--	if error == USESKILL_RESULT_ERR_OTHER then			--其它错误
--	elseif error == USESKILL_RESULT_ERR_CONSUME then	--消耗不足
--	elseif error == USESKILL_RESULT_ERR_PROPERTY then	--属性值不能满足要求
--	elseif error == USESKILL_RESULT_ERR_RECORD then		--表格值不能满足要求
--	elseif error == USESKILL_RESULT_ERR_SCENE then		--场景不符合要求
--	elseif error == USESKILL_RESULT_ERR_EVENT then		--事件不符合要求
--	elseif error == USESKILL_RESULT_ERR_COOLDOWN then	--技能在冷却中
--	elseif error == USESKILL_RESULT_ERR_NOSKILL then		--对象身上没有这个技能
--	elseif error == USESKILL_RESULT_ERR_INITIATIVE then	--不是主动技能，不能主动施放
--	elseif error == USESKILL_RESULT_ERR_TARGETDEAD then	--对象已死亡
--	elseif error == USESKILL_RESULT_ERR_FILTERLIMIT then	--不满足该技能的使用要求！
--	elseif error == USESKILL_RESULT_ERR_EQUIPLIMIT then	--不能满足该技能的装备需求！
--	elseif error == USESKILL_RESULT_ERR_HPLESS then		--机体耐久度不足！
--	elseif error == USESKILL_RESULT_ERR_MPLESS then		--机体能量不足！
--	elseif error == USESKILL_RESULT_ERR_SPLESS then		--机体能量不足！
--	elseif error == USESKILL_RESULT_ERR_NOTARGET then		--没有选择技能使用目标！
--	elseif error == USESKILL_RESULT_ERR_CANTUSESKILL then	--不能使用技能
--	elseif error == USESKILL_RESULT_ERR_FUNCLIMIT then	--不满足该技能的脚本要求
--	elseif error == TARGETFILTER_RESULT_ERR_TOONEAR then	--距离太近
--	elseif error == TARGETFILTER_RESULT_ERR_TOOFAR then	--距离太远
--	elseif error == TARGETFILTER_RESULT_ERR_HORI then		--不在水平角度范围
--	elseif error == TARGETFILTER_RESULT_ERR_VERT then		--不在垂直角度范围
--	elseif error == TARGETFILTER_RESULT_ERR_COLLIDE then	--直线距离之间有碰撞
--	elseif error == TARGETFILTER_RESULT_ERR_RECTANGLE then	--不在矩形范围内
--	elseif error == TARGETFILTER_RESULT_ERR_OBJTYPE_SELF then		--技能使用目标必须是自己！
--	elseif error == TARGETFILTER_RESULT_ERR_OBJTYPE_PET then		--技能使用目标必须是宠物！
--	elseif error == TARGETFILTER_RESULT_ERR_OBJTYPE_OWNER then		--技能使用目标必须是宠物的主人！
--	elseif error == TARGETFILTER_RESULT_ERR_OBJTYPE_SELECT then	--技能使用目标必须是选择目标！
--	elseif error == TARGETFILTER_RESULT_ERR_OBJTYPE_SELECTORSELF then--技能使用目标必须是自己或选择目标！
--	elseif error == TARGETFILTER_RESULT_ERR_CLASSTYPE_PLAYER then	--技能使用目标必须是玩家！
--	elseif error == TARGETFILTER_RESULT_ERR_CLASSTYPE_NPC then		--技能使用目标必须是NPC！
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_ENIMY then--技能使用目标必须是敌对阵营的！
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_FRIEND then--技能使用目标必须同一阵营的！
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_TEAM then	--技能使用目标必须队友！
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_GUILD then--技能使用目标必须是同一公会成员！
--	elseif error == TARGETFILTER_RESULT_ERR_DIRTYPE_FRONT then		--必须在技能使用目标的正面才能使用该技能！
--	elseif error == TARGETFILTER_RESULT_ERR_DIRTYPE_BACK then		--必须在技能使用目标的后面才能使用该技能！
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_PET then		--技能使用目标必须是召唤的宠物
--	elseif error == TARGETFILTER_RESULT_ERR_RELATIONTYPE_MASTER then		--技能使用目标必须是宠物的主人

--	end
end