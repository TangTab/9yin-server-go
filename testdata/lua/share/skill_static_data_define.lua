-- 技能通用静态属性

	Skill_LauchType = 0;        -- 发动类型 0主动型, 1被动型"	
	Skill_SkillType = 1;            -- 0,普通; 1,伤害;2,回复;3,增益;4,减益	
	Skill_EffectType = 2;           -- 招式类别 0：无类别，1：实招，2：虚招，3：架招 1克制2，2克制3，3克制1，所有都克制0（空）"	
	Skill_CoolDownCategory = 3;     -- 冷却分类
	Skill_CoolEffectType = 4;       -- 冷却传递方式	
	Skill_CoolDownTeam = 5;         -- 冷却影响分组编号(数字)"	"
	Skill_ShowName = 6;             -- 是否发招显示(显示参数)0.不显示1.显示"	
	Skill_Photo = 7;                -- 图片(显示参数)	
	Skill_TaoLu = 8;                -- 所属套路	
	Skill_IsDamage = 9;             -- 善意恶意倾向
	Skill_DamageType = 10;           -- 攻击模式
    Skill_TargetType = 11;           -- 目标类型
    Skill_TargetSelect = 12;         -- 目标选择参数包
    Skill_LockType = 13;             -- 锁定包
	Skill_LuaScript = 14;            -- 脚本函数名
	Skill_MinVarPropNo = 15;         -- 起始编号
	Skill_MaxVarPropNo = 16;         -- 结束编号
	Skill_NormalAttack = 17;
	Skill_Loop = 18
	
	Skill_LockStart = 19;
--	招式编号	锁定几率	"命中动作
--	触发受创"	"己方锁定时间
--	出招命中硬直"	"目标锁定时间
--	受创硬直"	"锁定己方位移
--	攻击位移"	"锁定对方位移
--	受创位移"	"攻击落空动作
--	触发闪避"	"己方锁定时间
--	出招落空硬直"	"目标锁定时间
--	闪避表演硬直"	"锁定己方位移
--	攻击位移"	"锁定对方位移
--	闪避位移"	"攻击被防动作
--	触发防御"	"己方锁定时间
--	出招落空硬直"	"目标锁定时间
--	闪避表演硬直"	"锁定己方位移
--	攻击位移"	"锁定对方位移
--	闪避位移"
	
	Skill_LockRatio = 20;			--感觉需要修正，当前不修正
	Skill_HitAction = 21;		    --c++ no use 
	Skill_LockSrcTime = 22;			--c++ no use 
	Skill_LockTargetTime = 23;		--c++ no use 
	Skill_LockSrcDist = 24;			--c++ no use 
	Skill_LockTargetDist = 25;		--c++ no use 
	Skill_DodgeAction = 26;			--c++ no use 
	Skill_SrcDodgeTime = 27;			--c++ no use 
	Skill_TargetDodgeTime = 28;		--c++ no use 
	Skill_SrcDodgeDist = 29;			--c++ no use 
	Skill_TargetDodgeDist = 30;		--c++ no use 
	Skill_ParryAction = 31;			--c++ no use 
	Skill_SrcParryTime = 32;			--c++ no use 
	Skill_TargetParryTime = 33;		--c++ no use 
	Skill_SrcParryDist = 34;			--c++ no use 
	Skill_TargetParryDist = 35;		--c++ no use 
	Skill_LockEnd = 36;

	-- 技能射程范围
	Skill_TargetShapeBegin = 37;		--依赖于SkillRange属性
	Skill_TargetShape = 38;
	Skill_TargetShapePara1 = 39;		-- 可变属性，基础值,仅能在初始化时设置
	Skill_TargetShapePara2 = 40;		-- 可变属性，基础值,仅能在初始化时设置
    Skill_TargetShapeEnd = 41

    -- 作用范围
    Skill_HitShapeBegin = 42;
	Skill_HitShape = 43;
	Skill_HitShapePara1 = 44;		-- 可变属性，基础值,仅能在初始化时设置
	Skill_HitShapePara2 = 45;		-- 可变属性，基础值,仅能在初始化时设置
	Skill_HitCount = 46;				-- 可变属性，基础值,仅能在初始化时设置
	Skill_HitShapeEnd = 47;

	-- 技能目标数据包
	Skill_SelectBegin = 48;			--依赖于SkillSelect属性
	Skill_TargetClassType = 49;
	Skill_TargetRelationType = 50
	Skill_IncludeDead = 51
	Skill_SelectEnd = 52

	--[[-- 技能流程包，SkillNormal专用
	Skill_FlowBegin;			-- 依赖于Flow属性
	Skill_PrepareTime;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_HitTime;				-- 可变属性，基础值,仅能在初始化时设置
	Skill_LeadTime;				-- 可变属性，基础值,仅能在初始化时设置
	Skill_LeadSepTime;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_PrepareAction,
	Skill_FlowHitAction,
	Skill_LeadAction,
	Skill_FinishAction,
	Skill_FlowEnd,

	-- 技能消耗，通用
	Skill_ConsumeBegin;			-- 依赖于Consume属性
	Skill_AConsumeHP;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_AConsumeMP;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_PConsumeHP;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_PConsumeMP;			-- 可变属性，基础值,仅能在初始化时设置
	Skill_ConsumeItemID;		-- 可变属性，基础值,仅能在初始化时设置
	Skill_ConsumeItemCount;		-- 可变属性，基础值,仅能在初始化时设置
	Skill_ConsumeEnd,]]