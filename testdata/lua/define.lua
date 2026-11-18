TYPE_SCENE	= 1;	-- 场景
TYPE_PLAYER	= 2;	-- 玩家
TYPE_NPC	= 4;	-- NPC
TYPE_ITEM	= 8;	-- 物品
TYPE_HELPER	= 16;	-- 辅助对象

--属性数据类型
VTYPE_UNKNOWN = 0;	-- 未知
VTYPE_BOOL = 1;		-- 布尔
VTYPE_INT = 2;		-- 32位整数
VTYPE_INT64 = 3;	-- 64位整数
VTYPE_FLOAT = 4;	-- 单精度浮点数
VTYPE_DOUBLE = 5;	-- 双精度浮点数
VTYPE_STRING = 6;	-- 字符串
VTYPE_WIDESTR = 7;	-- 宽字符串
VTYPE_OBJECT = 8;	-- 对象号
VTYPE_POINTER = 9;	-- 指针
VTYPE_USERDATA = 10;-- 用户数据
VTYPE_TABLE = 11;	-- 表
VTYPE_MAX = 12;

--提示信息类型
TIPSTYPE_SYSTEMINFO = 0;	--系统信息(引擎模块发送的提示信息用，逻辑模块不要使用)
TIPSTYPE_SYSTEMNO = 1;		--系统提示(客户端显示加[提示]前缀)
TIPSTYPE_TIPSINFO = 2;		--主画面向上滚动的提示信息
TIPSTYPE_GMINFO = 99;		--GM调试信息

SS_TALKER = 0;		--对话
SS_PICKITEM = 1;	--可拾取物品
SS_FIGHTER = 10;	--战斗对象

--绑定类型
BIND_TYPE_NONE = 0; --不会被绑定
BIND_TYPE_PICK = 1; --拾取绑定（一旦物品进入玩家的背包，立即绑定）
BIND_TYPE_EQUIP = 2; --装备绑定（一旦物品进入玩家的装备栏，立即绑定）

--绑定状态
BIND_STATUS_NONE = 0; --未绑定
BIND_STATUS_BOUND = 1; --已绑定

--伤害类型
DAMAGE_TYPE_PHY = 0;    --外功
DAMAGE_TYPE_MAGIC_A = 1;--刚
DAMAGE_TYPE_MAGIC_B = 2;--柔
DAMAGE_TYPE_MAGIC_C = 3;--阴
DAMAGE_TYPE_MAGIC_D = 4;-- 阳

-- 对象的移动模式
MMODE_STOP = 0		-- 停止
MMODE_MOTION = 1	-- 地表移动
MMODE_JUMP = 2		-- 跳跃
MMODE_JUMPTO = 3	-- 改变跳跃的目标方向
MMODE_FLY = 4		-- 空中移动
MMODE_SWIM = 5		-- 水中移动
MMODE_DRIFT = 6	    -- 水面移动
MMODE_CLIMB = 7	    -- 爬墙
MMODE_SLIDE = 8	    -- 在不可行走范围内滑行
MMODE_SINK = 9		-- 下沉

-- 技能类型
SKILLTYPE_TOSELF = 1		--给自己用
SKILLTYPE_TOENEMY = 2		--给敌人用
SKILLTYPE_TOFRIEND = 3		--给朋友用
SKILLTYPE_LOCK = 4			--锁定技能
