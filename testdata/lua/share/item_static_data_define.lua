--
-- File: item_static_data_define.h
--	 NB: (变量定义和ItemPropQuery.h保持一致)
-- Desp: 道具静态属性查询
-- Auth: YeHui
-- Date: 2010-6-25
--

	--
	-- 道具逻辑包
	--
	--	BindType		绑定类型		2
	--	BindStatus		绑定状态		2
	--	SpecialItem		唯一道具		2
	--	CantDepot		能否放入仓库	2
	--	CantExchange	能否交易		2
	--	CantDelete		是否能删除		2
	--	CantChuck		是否能丢弃		2
	--	CantMove		是否能移动		2
	--	CantSell		是否能出售		2
	--	DropRatio		掉落几率		2
	--	ValidTime		有效期			2
	--	ValidType		计时类型		2
	--	InlayType		镶嵌类型		2
	--	UseIndirect		是否必须间接使用
	--
	Item_LogicPack_Begin	= 0
	Item_BindType			= 0
	Item_SpecialItem		= 1
	Item_CantDepot			= 2
	Item_CantExchange		= 3
	Item_CantDelete			= 4
	Item_CantChuck			= 5
	Item_CantMove			= 6
	Item_CantSell			= 7
	Item_DropRatio			= 8
	Item_ValidTime			= 9
	Item_ValidType			= 10
	Item_InlayType			= 11
	Item_UseIndirect		= 12
	Item_LogicPack_End		= 19

	--
	-- 道具功能包
	--	
	--	LuaScript		
	--	PrepareAction
	--	PrepareTime
	--	CanCopy
	--	CanMusic
	--	Level
	--	CropID
	--	Rare
	--	CoolDownCategory		冷却类型编号
	--	CoolDownTime			冷却时间
	--	CoolDownTeam			冷却影响分组编号
	--	AddCoolDownTime			附加冷却时间
	--	
	--
	Item_FuncPack_Begin		= 20
	Item_LuaScript			= 20
	Item_PrepareAction		= 21
	Item_PrepareTime		= 22
	Item_FuncPack_End		= 29

	--
	-- 美术包
	--	Photo
	--
	Item_ArtPack_Begin		= 30
	Item_Photo				= 30
	Item_ArtPack_End		= 39

	--
	-- [副本道具]专有静态属性
	--
	--	TaskID				任务编号
	--	LuaScript			脚本
	--	PrepareAction		准备动作
	--	PrepareTime			准备时间
	--	CanCopy				是否能抄录
	--	CanMusic			是否能奏乐
	--	Level				等级
	--	CoolDownCategory	冷却类型
	--	CoolDownTime		冷却时间
	--
	--
	Item_CopyPack_Begin		= 40
	Item_CopyPack_TaskID	= 40
	Item_CopyPack_End		= 49

	--
	-- [配方道具]专有静态属性
	--
	--	FormulaID			合成公式ID	Para1
	--	LuaScript			脚本
	--	PrepareAction		准备动作
	--	PrepareTime			准备时间
	--	CoolDownCategory	冷却类型
	--	CoolDownTime		冷却时间
	--
	--
	Item_IngredientPack_Begin		= 50
	Item_IngredientPack_FormulaID	= 50
	Item_IngredientPack_End			= 59

	--
	-- [任务道具]专有静态属性
	--
	--	TaskID				任务编号	Para1
	--	LuaScript			脚本
	--	PrepareAction		准备动作
	--	PrepareTime			准备时间
	--	CanCopy				是否能抄录
	--	CanMusic			是否能奏乐
	--	Level				等级
	--	CoolDownCategory	冷却类型
	--	CoolDownTime		冷却时间
	--
	--
	Item_TaskPack_Begin		= 60
	Item_TaskPack_TaskID	= 60
	Item_TaskPack_End		= 69


	--
	-- [使用技能类道具]专有静态属性
	--
	--	SkillID				技能编号	Para1
	--	PrepareAction		准备动作
	--	PrepareTime			准备时间
	--	CanCopy				是否能抄录
	--	CanMusic			是否能奏乐
	--	Level				等级
	--	CoolDownCategory	冷却类型
	--	CoolDownTime		冷却时间
	--
	--
	Item_UseSkillPack_Begin			= 70
	Item_UseSkillPack_SkillID		= 70
	Item_UseSkillPack_SkillLevel	= 71
	Item_UseSkillPack_End			= 79


	--
	-- [学习技能类道具]专有静态属性
	--
	--	TragetID			学习技能/内功的ID		Para1
	--	TargetCondition		使用道具的条件			Para2
	--	TargetLevel			学习技能/内功的等级		Para3
	--	PrepareAction		准备动作
	--	PrepareTime			准备时间
	--	CanCopy				是否能抄录
	--	CanMusic			是否能奏乐
	--	CoolDownCategory	冷却类型
	--	CoolDownTime		冷却时间
	--
	--
	Item_LearnSkillPack_Begin			= 80
	Item_LearnSkillPack_TargetID		= 80
	Item_LearnSkillPack_TargetLevel		= 81
	Item_LearnSkillPack_ZhaoShiNeeded	= 82
	Item_LearnSkillPack_ZhaoShiLevel	= 83
	Item_LearnSkillPack_NeiGongNeeded	= 84
	Item_LearnSkillPack_NeiGongLevel	= 85
	Item_LearnSkillPack_SchoolNeed		= 86
	Item_LearnSkillPack_PropNeed		= 87
	Item_LearnSkillPack_PropValue		= 88
	Item_LearnSkillPack_End				= 89


	--
	-- [种子道具]专有静态属性
	--
	--	SeedTime			播种时间(ms)
	--	SeedAction			播种动作
	--	Level				等级
	--	CropID				农作物id
	--	ToolType			工具类型
	--	NeedTool			需要工具
	--
	Item_SeedPack_Begin			= 90
	Item_SeedPack_SeedTime		= 90
	Item_SeedPack_SeedAction	= 91
	Item_SeedPack_CropID		= 92
	Item_SeedPack_CropLevel		= 93	
	Item_SeedPack_ToolType		= 94
	Item_SeedPack_NeedTool		= 95
	Item_SeedPack_End			= 99

	--
	-- [宝箱道具]专有静态属性(待定)
	--
	--	Para1				物品集合
	--
	--
	Item_MaterialPack_Begin		= 100
	Item_MaterialPack_Result	= 100
	ITEM_MaterialPack_Count     = 101
	Item_MaterialPack_End		= 109

	--
	-- [属性修改道具]专有静态属性
	--	PropName			属性名称
	--	PropValue			属性值
	--
	Item_PropModifyPack_Begin		= 110
	Item_PropModifyPack_PropName	= 110
	Item_PropModifyPack_PropValue	= 111
	Item_PropModifyPack_End			= 119