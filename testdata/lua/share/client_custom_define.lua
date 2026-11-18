--聊天命令，格式：int msgid int chat_type wstring content(用空格分隔的一个宽字符串) [wstring targetname](私聊专用：目标名称)
CLIENT_CUSTOMMSG_CHAT = 1

--发送GM命令，格式：int msgid wstring info
CLIENT_CUSTOMMSG_GM = 2

--切换移动状态，格式：int msgid int move_type(0:跑步 1:走路)
CLIENT_CUSTOMMSG_MOVE_TYPE = 3

--返回MessageBox的应答结果，消息格式：
--unsigned msgid int msgboxid int answer(客户端点击的按纽索引)
CLIENT_CUSTOMMSG_ANSWERMESSAGEBOX = 5
--返回InputBox的应答结果，消息格式：
--unsigned msgid int inputboxid wstring input(客户端输入的内容)
CLIENT_CUSTOMMSG_ANSWERINPUTBOX = 6

--基础属性加点，消息格式：int msgid string prop1 int addpoint1(添加的点数) .. string propn int addpointn(添加的点数)
CLIENT_CUSTOMMSG_BASEPROP_ADDPOINT = 10
--基础属性洗点，消息格式：int msgid string prop1 int decpoint1(减少的点数) .. string propn int decpointn(减少的点数)
CLIENT_CUSTOMMSG_BASEPROP_DECPOINT = 11

--压力测试专用消息int msgid string test_type..各种测试对应参数
CLIENT_CUSTOMMSG_STRESS_TEST = 12

--容器操作相关命令
--客户端上传使用物品的消息，消息格式：unsigned msgid int srcviewid int srcpos int dstviewid int dstpos
CLIENT_CUSTOMMSG_USEITEM_ON_ITEM = 29
--客户端上传移动物品的消息，消息格式：unsigned msgid int srcviewid int srcpos int destviewid int destpos
CLIENT_CUSTOMMSG_MOVEITEM = 30
--客户端上传使用物品的消息，消息格式：unsigned msgid int srcviewid int srcpos
CLIENT_CUSTOMMSG_USEITEM = 31
--客户端上传丢弃物品的消息，消息格式：unsigned msgid int srcviewid int srcpos int amount(数量，为0时丢弃全部物品)
CLIENT_CUSTOMMSG_CHUCKITEM = 32
--客户端上传拆分物品的消息，消息格式：unsigned msgid int srcviewid int srcpos int destviewid int destpos int amount
CLIENT_CUSTOMMSG_SPLITITEM = 33
--客户端上传删除物品的消息，消息格式：unsigned msgid int srcviewid int srcpos int amount(数量，为0时丢弃全部物品)
CLIENT_CUSTOMMSG_DELETEITEM = 34
--客户端上传锁定物品的消息，消息格式：unsigned msgid int srcviewid int srcpos
CLIENT_CUSTOMMSG_LOCKITEM = 35
--客户端上传整理容器中的物品，消息格式：unsigned msgid int srcviewid int beginpos int endpos
CLIENT_CUSTOMMSG_ARANGEITEM = 36

--客户端上传主动关闭某个容器，消息格式：unsigned msgid int srcviewid
CLIENT_CUSTOMMSG_CLOSTVIEW = 39

--客户端上传修理容器中的装备，消息格式：unsigned msgid int srcviewid int srcpos
CLIENT_CUSTOMMSG_REPAIRITEM = 40
--客户端上传修理容器中的全部装备，消息格式：unsigned msgid int srcviewid
CLIENT_CUSTOMMSG_REPAIRALL = 41

--物品操作相关42-59
--鉴定物品 消息格式: unsigned cmdid
CLIENT_CUSTOMMSG_IDENTIFY = 42

--显示哪些装备 消息格式: unsigned cmdid int set
CLIENT_CUSTOMMSG_SHOWEQUIP_SET = 60
--显示那类装备 消息格式: unsigned cmdid int type(0:显示江湖装1:显示门派装2:显示时装)
CLIENT_CUSTOMMSG_SHOWEQUIP_TYPE = 61

--客户端上传从普通商店中购买物品，消息格式：unsigned msgid int shopid int itemid
CLIENT_CUSTOMMSG_BUY_FROM_SHOP = 70
--客户端出售物品到普通商店，消息格式：unsigned msgid int shopid int src_viewid int src_viewindex
CLIENT_CUSTOMMSG_SELL_TO_SHOP = 71
--客户端上传从随机商店中购买物品，消息格式：unsigned msgid int pos
CLIENT_CUSTOMMSG_BUY_FROM_RANDOMSHOP = 72
--客户端上传从跑商商店购买物品消息格式：unsigned msgidstring shopidint posint amountobject npc
CLIENT_CUSTOMMSG_BUY_FROM_BUSINESSSHOP = 74
--客户端上传从跑商商店购买物品，消息格式：unsigned msgidstring shopidint posint amount
CLIENT_CUSTOMMSG_SELL_TO_BUSINESSSHOP = 75
--客户端上传从跑商商店购卖物品请求，消息格式：unsigned msgidstring shopidint posint amount
CLIENT_CUSTOMMSG_SELL_TO_BUSINESSSHOP_REQUEST = 76
--客户端上传跑商商店的货物价格刷新请求消息格式：unsigned msgidstring shopid
CLIENT_CUSTOMMSG_REQUEST_REFRESH_ITEM_INFO = 77
--客户端上传跑商商店的货物价格刷新请求消息格式：unsigned msgid
CLIENT_CUSTOMMSG_REQUEST_BUSINESS_FINISH = 78


--打孔、镶嵌、分解
--客户端打开镶嵌窗口，消息格式：unsigned msgid, object npc, string formname
CLIENT_CUSTOMMSG_ENCHASE_OPNE = 90
--客户端关闭镶嵌窗口，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_ENCHASE_CLOSE = 91
--客户端选择一个装备，消息格式：unsigned msgid,int srcviewid, int srcpos
CLIENT_CUSTOMMSG_EQUIP_SELECT = 92
--客户端放弃选中装备，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_EQUIP_GIVEUP = 93
--客户端添加一个镶嵌，消息格式：unsigned msgid,int srcpos, int dstpos
CLIENT_CUSTOMMSG_ADD_ENCHASE = 94
--客户端取消一个镶嵌，消息格式：unsigned msgid,int pos
CLIENT_CUSTOMMSG_DEL_ENCHASE = 95
--客户端请求执行打孔，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_EQUIP_ADDHOLE = 96
--客户端请求执行镶嵌，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_EQUIP_ENCHASE = 97
--客户端请求执行拆分，消息格式：unsigned msgid, int pos...
CLIENT_CUSTOMMSG_EQUIP_SEPARATE = 98


--用户交互请求，消息格式：unsigned msgid int request_type wstring name(被请求者角色名称)
CLIENT_CUSTOMMSG_REQUEST = 100
--用户交互请求回应，消息格式：unsigned msgid int request_type wstring name(被请求者角色名称) int result(0:拒绝1:同意)
CLIENT_CUSTOMMSG_REQUEST_ANSWER = 101
--用户设置对其他用户的请求开关，消息格式：unsigned msgid int newvalue(每位对应request_type的一种类型)
CLIENT_CUSTOMMSG_REQUEST_SET = 102

--查看其他角色的全身装备信息，消息格式：unsigned msgid wstring name(被请求者角色名称)
CLIENT_CUSTOMMSG_LOOK_EQUIPS = 105
--查看其他角色的背包中的某件装备的信息，消息格式：unsigned msgid wstring name(被请求者角色名称) string item_uniqueid
CLIENT_CUSTOMMSG_LOOK_ITEM = 106

-- 好友模块
-- 客户端请求加好友，消息格式：unsigned msgid wstring name(好友名称)
CLIENT_CUSTOMMSG_ADD_FRIEND = 110
-- 客户端请求删好友，消息格式：unsigned msgid wstring name(好友名称)
CLIENT_CUSTOMMSG_DEL_FRIEND = 111
-- 客户端请求加黑名单，消息格式：unsigned msgid wstring name(黑名单名称)
CLIENT_CUSTOMMSG_ADD_BLACKLIST = 112
-- 客户端请求删黑名单，消息格式：unsigned msgid wstring name(黑名单名称)
CLIENT_CUSTOMMSG_DEL_BLACKLIST = 113
-- 客户端请求加仇人，消息格式：unsigned msgid, wstring name(仇人名称)
CLIENT_CUSTOMMSG_ADD_ENEMY = 114
-- 客户端请求删仇人，消息格式：unsigned msgid, wstring name(仇人名称)
CLIENT_CUSTOMMSG_DEL_ENEMY = 115
-- 客户端请求把好友移动到黑名单，消息格式：unsigned msgid, wstring name(好友名称)
CLIENT_CUSTOMMSG_MOVE_TO_BLACKLIST = 116

--拾取礼包item
CLIENT_CUSTOMMSG_PICK_ITEM =123
-- 副本模块
-- 客户端请求重置所有普通副本
CLIENT_CUSTOMMSG_RESET_ALL_CLONE = 130
--客户端请求团队副本重置的剩余时间
CLIENT_CUSTOMMSG_GET_LEAVE_TIME_CLONE = 140

--经脉系统(150~170)
--升级经脉，消息格式：unsigned msgid, string jingmaiid
CLIENT_CUSTOMMSG_UPGRADE_JINGMAI = 150

--通过页数请求刷新挑战排行总表 unsigned msgid,  int page
CLIENT_CUSTOMMSG_GET_CHALLENGE_RANK_BY_PAGE = 190
--通过名字请求刷新挑战排行总表 unsigned msgid,  wstring name
CLIENT_CUSTOMMSG_GET_CHALLENGE_RANK_BY_NAME = 191
--获得自己当前的挑战目标名单unsigned msgid,
CLIENT_CUSTOMMSG_GET_CHALLENGE_TARGET_LIST = 192
--打听某个玩家的位置unsigned msgid,  wstring name
CLIENT_CUSTOMMSG_GET_PLAYER_POSITION = 193
	
--快捷栏操作命令
--设置某个键位，消息格式：unsigned msgid int no int para1 int para2
CLIENT_CUSTOMMSG_SETSHORTCUT = 200
--删除某个键位，消息格式：unsigned msgid int no
CLIENT_CUSTOMMSG_REMOVESHORTCUT = 201

--仓库操作
--打开远程仓库，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_OPEN_DIRECTDEPOT = 202
--购买或续费远程仓库，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_BUY_DIRECTDEPOT = 203

--资金操作相关命令
--存款，消息格式：unsigned msgid int captial_type int value ...  int capital_type int value
CLIENT_CUSTOMMSG_SAVECAPITAL = 205
--取款，消息格式：unsigned msgid int captial_type int value ...  int capital_type int value
CLIENT_CUSTOMMSG_UNSAVECAPITAL = 206
--转账，消息格式：unsigned msgid widestr destname int captial_value1 ... int capital_valuen
CLIENT_CUSTOMMSG_TRANSFER = 207

--战斗技能相关命令

--使用技能，消息格式：unsigned msgid string skillid
CLIENT_CUSTOMMSG_USE_SKILL = 211
--继续暂停的技能，消息格式：unsigned msgid, x, y, z, orient
CLIENT_CUSTOMMSG_RESUME_SKILL = 212
--使用准备技能，消息格式：unsigned msgid, string skillid
CLIENT_CUSTOMMSG_USE_NEXT_SKILL = 213

--使用内功，消息格式：unsigned msgid string neigongid
CLIENT_CUSTOMMSG_USE_NEIGONG = 215

--升级内功，消息格式：unsigned msgid string neigongid int inclevel
CLIENT_CUSTOMMSG_UPGRAD_NEIGONG = 222


--通用技能的使用
--打坐(取消打坐)：unsigned msgid
CLIENT_CUSTOMMSG_SITCROSS	= 240
--跳舞(取消跳舞)：unsigned msgid
CLIENT_CUSTOMMSG_DANCING	= 241

--骑乘(取消骑乘)：unsigned msgid
CLIENT_CUSTOMMSG_RIDE	= 243
--骑乘冲刺：unsigned msgid
CLIENT_CUSTOMMSG_RIDE_SPURT	= 244

--拾取农作物：unsigned msgid int index
CLIENT_CUSTOMMSG_PICK_CROP = 250
--拾取全部农作物：unsigned msgid
CLIENT_CUSTOMMSG_PICK_ALL_CROP = 251
--拾取农作物窗口关闭：unsigned msgid
CLIENT_CUSTOMMSG_PICK_CROP_CLOSE = 252

--复活命令，消息格式：unsigned msgid int relive_type
CLIENT_CUSTOMMSG_RELIVE = 290
--设置PK模式，消息格式：unsigned msgid int pkmode
CLIENT_CUSTOMMSG_SETPKMODE = 291
--改变警戒状态，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_CHANGEWARNING = 292
--改变PK保护属性，消息格式：unsigned msgid int pkprotect int checked
CLIENT_CUSTOMMSG_SETPKPROTECT = 293
--设置当前称号，消息格式：unsigned msgid string title
CLIENT_CUSTOMMSG_SETTITLE = 295

-- 任务模块客户端操作
-- 放弃任务 消息格式: unsigned msgid int task_id
--CLIENT_CUSTOMMSG_GIVEUP_TASK    =   300
-- 分支类型需要对话框操作返回选择结果给服务器端： 消息格式 unsigned msgid int task_id int select_index
--CLIENT_CUSTOMMSG_SELECT_RESULT 	=	301
-- 查询头顶标记 消息格式 unsigned msgid obj npc
--CLIENT_CUSTOM_MSG_QUERY_TASK_FLAG = 302
-- 查询是否有粒子特效 消息格式 unsigned msgid obj npc
--CLIENT_CUSTOM_MSG_QUERY_TASK_EFFECT = 303
-- 自动接任务 消息格式 unsigned msgid task_id
--CLIENT_CUSTOM_MSG_ACCEPT_TASK = 304
-- 共享任务，消息格式: unsigned msgid int task_id
--CLIENT_CUSTOMMSG_SHARE_TASK = 305
-- 返回共享选择结果，消息格式: unsigned msgid int task_id wide_str sharer_name int result
--CLIENT_CUSTOMMSG_RETURN_SHARE_RESULT = 306
-- 客户端请求完成任务, 消息格式: unsigned msg_id, int task_id, int present_id, PERSISTID npc
--CLIENT_CUSTOM_MSG_COMPLETE_TASK = 307

-- 队伍模块
-- 设置队伍中人物属性，消息格式 unsigned msgid unsigned set_type usigned set_param wstring target
CLIENT_CUSTOMMSG_TEAM_MEMBER_SET = 308
-- 设置队伍模式，消息格式 unsigned msgid unsigned team_type
CLIENT_CUSTOMMSG_TEAM_TYPE_SET = 309
-- 直接单人创建队伍，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_TEAM_CREATE = 310
-- T出队伍，消息格式：unsigned msgid wstring target
CLIENT_CUSTOMMSG_TEAM_KICKOUT = 311
-- 退出队伍，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_TEAM_LEAVE = 312
-- 移交队长，消息格式：unsigned msgid wstring target
CLIENT_CUSTOMMSG_TEAM_CHANGE_CAPTAIN = 313
-- 设置队伍说明信息消息格式:unsigned cmdid widestr info
CLIENT_CUSTOMMSG_TEAM_SET_INFO = 314
-- 设置队伍拾取分配品质消息格式:unsigned cmdid int quality
CLIENT_CUSTOMMSG_TEAM_SET_PICK_QUALITY = 314
--设置队伍拾取分配方式消息格式:unsigned cmdid int mode
CLIENT_CUSTOMMSG_TEAM_SET_PICK_MODE = 315
--要求加入队伍消息格式:unsigned cmdid wstring target
CLIENT_CUSTOMMSG_TEAM_REQUEST_JOIN = 316
--刷新招募寻组数据消息格式:unsigned cmdid
CLIENT_CUSTOMMSG_TEAM_REFRESH_INFO = 317
-- 添加招募寻组信息消息格式:unsigned cmdidint LVwstring Typewstring Missionwstring Info
CLIENT_CUSTOMMSG_TEAM_ADD_INFO = 318
--删除招募寻组信息消息格式:unsigned cmdid
CLIENT_CUSTOMMSG_TEAM_REMOVE_INFO = 319

--玩家交易系统
--确定		命令格式: unsigned cmdid
CLIENT_CUSTOMMSG_EXCHANGE_OK		=   320
--锁定		命令格式: unsigned cmdid
CLIENT_CUSTOMMSG_EXCHANGE_LOCK		=	321
--取消		命令格式: unsigned cmdid
CLIENT_CUSTOMMSG_EXCHANGE_CANCEL	=	322
--Money		命令格式: unsigned cmdid  int moneytype  int money
CLIENT_CUSTOMMSG_EXCHANGE_MONEY		= 323
--玩家添加交换栏物品  命令格式：unsigned cmdid, int src_viewid, int src_pos, int dest_viewid, int dest_pos
CLIENT_CUSTOMMSG_EXCHANGE_ADD_ITEM	=324
--玩家删除交换栏物品  命令格式: unsigned cmdid, int pos
CLIENT_CUSTOMMSG_EXCHANGE_REMOVE_ITEM	=325

--团队消息补充
--设置标记  命令格式: unsigned cmdid,string sign_str
CLIENT_CUSTOMMSG_TEAM_SET_SIGN	=326

-- 任务系统消息
CLIENT_CUSTOMMSG_TASK_MSG	= 327

--玩家添加加工栏物品  命令格式：unsigned cmdid, int src_viewid, int src_pos, int dest_viewid, int dest_pos
CLIENT_CUSTOMMSG_PROCESS_ADD_ITEM	=328
--玩家删除加工栏物品  命令格式: unsigned cmdid, int pos
CLIENT_CUSTOMMSG_PROCESS_REMOVE_ITEM	=329

-- 摆摊系统
-- 玩家打开摊位，命令格式：unsigned cmdidwidestr shopname
CLIENT_CUSTOMMSG_OPEN_STALL = 330
-- 玩家开始摆摊，命令格式：unsigned cmdid
CLIENT_CUSTOMMSG_BEGIN_STALL = 331
-- 玩家结束摆摊或者关闭视图，命令格式：unsigned cmdid
CLIENT_CUSTOMMSG_CLOSE_STALL = 332
-- 玩家设置出售物品，命令格式：unsigned cmdid int pos(在包裹栏中的位置) int pos(在摊位栏中的位置) int64 gold int64 silver
CLIENT_CUSTOMMSG_ADD_STALL_ITEM = 333
-- 玩家移除出售物品，命令格式：unsigned cmdid int pos(在摊位栏中的位置)
CLIENT_CUSTOMMSG_REMOVE_STALL_ITEM = 334

-- 玩家设置出售物品价格，命令格式：unsigned cmdid int pos int gold int silver
CLIENT_CUSTOMMSG_SET_STALL_ITEMPRICE = 336
-- 玩家买了一件商品，命令格式：unsigned cmdidint posint count int gold int silver
CLIENT_CUSTOMMSG_BUY_OTHERONE_STALL = 337
-- 玩家需要改变出售的物品信息：unsigned cmdid
CLIENT_CUSTOMMSG_END_STALL = 338
-- 玩家请求打开某个摊位，命令格式: unsigned cmdid widestr playername
CLIENT_CUSTOMMSG_OPEN_LOOKUP_STALL = 341
-- 玩家请求关闭观察摊位，命令格式: unsigned cmdid
CLIENT_CUSTOMMSG_CLOSE_LOOKUP_STALL = 342
-- 玩家请求添加收购物品，命令格式：unsigned cmdid int pos string configid int count int64 gold int64 silver
CLIENT_CUSTOMMSG_SET_PURCHASE_ITEM = 343
-- 玩家请求删除收购物品，命令格式：unsigned cmdid int pos
CLIENT_CUSTOMMSG_DEL_PURCHASE_ITEM = 344
-- 玩家从摊位的收购栏卖出了物品，命令格式：unsignd cmdid int pos string configid int count int64 gold int64 silver
CLIENT_CUSTOMMSG_SEL_PURCHASE_ITEM = 345
--玩家从摊位的收购栏卖出了物品(通过左键或右键点击收购物品)，命令格式：unsignd cmdid,  int pos, int sellnum
CLIENT_CUSTOMMSG_SEL_PURCHASE_ITEM_SELECT = 346
--玩家请求摆摊 命令格式：unsignd cmdid
CLIENT_CUSTOMMSG_REQUEST_STALL = 347
-- 340-349由摆摊消息占用，请不要重复 ：）

--邮寄系统相关操作
--玩家打开邮寄界面命令格式：unsigned cmdidint type
CLIENT_CUSTOMMSG_POST_OPENBOX = 350
--玩家取消寄信
CLIENT_CUSTOMMSG_POST_CANCEL = 351
--玩家寄信命令格式：unsigned cmdid widestr targetnamewidestr letternamewidestr content int money
CLIENT_CUSTOMMSG_POST_SEND = 352
--玩家读信
CLIENT_CUSTOMMSG_POST_READ = 353
--玩家提取银两
CLIENT_CUSTOMMSG_POST_GET_MONEY = 354
--玩家提取物品
CLIENT_CUSTOMMSG_POST_GET_ITEM = 355
--玩家删除邮件
CLIENT_CUSTOMMSG_POST_DEl = 356
--玩家关闭邮寄界面
CLIENT_CUSTOMMSG_POST_CLOSEBOX = 357
--玩家关闭读信界面
CLIENT_CUSTOMMSG_POST_CLOSE_READBOX = 358
--玩家提取附件
CLIENT_CUSTOMMSG_POST_GET_APPEDIX = 359
--玩家删除全部邮件
CLIENT_CUSTOMMSG_POST_DEl_ALL = 360


-- 拜师消息，暂时( 370 - 385)
-- 单方解除师徒关系 ，消息格式：unsigned msgid int Flag widestr targetname
CLIENT_CUSTOMMSG_SINGLEFREETEACHPRNRELATION	= 370
-- 主动加入师徒候选名单消息格式： unsigned msgid int Flag (0为师傅候选，1为徒弟候选)
CLIENT_CUSTOMMSG_JOINTEACHPRNELIST = 371
-- 主动退出师徒候选名单消息格式： unsigned msgid int Flag (0为师傅候选，1为徒弟候选)
CLIENT_CUSTOMMSG_EXITTEACHPRNELIST = 372
-- 查看师徒候选名单消息格式： unsigned msgid int Flag (0为师傅候选，1为徒弟候选)，int PageType(翻页类型)
CLIENT_CUSTOMMSG_QUERYTEACHPRNELIST = 373
-- 得到师徒推荐名单消息格式： unsigned msgid int sex
CLIENT_CUSTOMMSG_GETLISTBYCONDITION = 374
--师傅得到徒弟收益:    unsigned msgidwstring name;
CLIENT_CUSTOMMSG_GETPRENTICE_INCOME = 375
--查看自己的所有徒弟名单:    unsigned msgid
CLIENT_CUSTOMMSG_QUERYALLPRENTICELIST = 376
--评价师傅结果:    unsigned msgid int ResType
CLIENT_CUSTOMMSG_JUDGETEACH_RESULT		= 377

--掉落捡取消息
--从掉落视窗捡取某件东西消息格式： unsigned cmdid int pos(pos为0时表示拾取所有可拾取的物品)
CLIENT_CUSTOMMSG_PICKUP_DROPITEM = 380
--客户端申请关闭当前查看的掉落视窗，消息格式:unsigned cmdid
CLIENT_CUSTOMMSG_CLOSE_DROPBOX = 382
--玩家骰子结果消息格式：unsigned cmdid object item int mode
CLIENT_CUSTOMMSG_ROLL_ITEM_RESULT = 387
--队长分配物品，消息格式: unsigned cmdid int pos widestr targetplayer
CLIENT_CUSTOMMSG_TEAMLEADER_ALLOC_ITEM = 388

--结婚消息
--组队申请结婚
CLIENT_CUSTOMMSG_REQUEST_MARRIAGE = 390

--恋人消息
--解除恋人 消息格式: widstr name
CLIENT_CUSTOMMSG_REQUEST_LOVER_FREE = 400

--托管系统相关操作--410-420
--开始挂机，消息格式：unsigned cmdid，unsigned trusteetype
CLIENT_CUSTOMMSG_START_TRUSTEE = 410
--停止挂机消息格式：unsigned cmdid
CLIENT_CUSTOMMSG_STOP_TRUSTEE = 411
--修改挂机设置
CLIENT_CUSTOMMSG_SET_TRUSTEE = 412
--清除挂机log
CLIENT_CUSTOMMSG_CLEAR_TRUSTEE_LOG = 413

--随机商店消息
CLIENT_CUSTOMMSG_RANDOM_SHOP_CLOSE = 420 --随机商店关闭

-- 表情动作消息 430-439
-- 上传动作请求 消息格式: unsigned cmdid string action_id
CLIENT_CUSTOMMSG_REQUEST_ACTION = 430

--撤销选中对象
CLIENT_CUSTOMMSG_SELECT_CANCEL = 440

--组队消息（添加）
CLIENT_CUSTOMMSG_TEAM_INFO_GET = 441--取得当前招募消息的具体内容
--玩家打开组队招募界面的时候，通知服务器记录
CLIENT_CUSTOMMSG_TEAM_PLAYER_OPEN = 442
--玩家关闭组队招募界面的时候，通知服务器删除
CLIENT_CUSTOMMSG_TEAM_PLAYER_CLOSE = 443
--解散队伍
CLIENT_CUSTOMMSG_TEAM_DESTROY = 444
--邀请玩家加入
CLIENT_CUSTOMMSG_TEAM_INVITE = 445

--轻功移动
CLIENT_CUSTOMMSG_QGMOVE = 450

-- 传授修为 确定开始 unsigned cmdid, int ImpartValue,
CLIENT_CUSTOMMSG_IMPART_BEGIN = 460
-- 传授修为 取消
CLIENT_CUSTOMMSG_IMPART_CANCEL = 461

-- 客户端上发书法游戏操作消息
CLIENT_CUSTOMMSG_MINIGAME_HANDWRITING_RESULT = 482
-- 客户端游戏结果上传: unsigned cmdid，string gamename int result int time
--CLIENT_CUSTOMMSG_GAME_RESULT = 500

-- 客户端请求特殊武器查询: unsigned cmdid，int type
CLIENT_CUSTOMMSG_SPECIAL_ITEM_QUERY = 510

-- 奇遇模块(预留ID范围520-529)
-- 奇遇界面打开请求服务器验证: unsigned cmdid
CLIENT_CUSTOMMSG_REFRESH_ADVENTURE	= 520

-- 奇遇道具使用按钮: unsigned cmdid
CLIENT_CUSTOMMSG_USE_ADVENTURE_ITEM = 521

-- 奇遇接受按钮: unsigned cmdid int index
CLIENT_CUSTOMMSG_CLICK_ADVENTURE_ACCEPT = 522

-- 奇遇拒绝按钮: unsigned cmdid int index
CLIENT_CUSTOMMSG_CLICK_ADVENTURE_DEFUSE = 523

-- 共享奇遇给好友: unsigned cmdid int taskid widestr friendname
CLIENT_CUSTOMMSG_SHARE_ADVENTURE = 524

-- 奇遇好友共享界面打开请求服务器验证: unsigned cmdid
CLIENT_CUSTOMMSG_REFRESH_ADVENTURE_SHAREMSG	= 525

-- 接受或拒绝共享奇遇的消息: unsigned cmdid int taskid widestr friendname int result
CLIENT_CUSTOMMSG_DO_SHARE_RESULT = 526

-- 放弃已经激活的共享奇遇: unsigned cmdid int row
CLIENT_CUSTOMMSG_GIVEUP_SHARE_ADV = 527


-- 自定义编辑任务模块(预留ID范围530-549)
-- 打开或关闭奖励容器: unsigned cmdid int flag
CLIENT_CUSTOMMSG_AUTO_PRIZE_BOX = 530

-- 客户端发布收集类任务: unsigned cmdid wstring title string collect_info int golden int silver int time_limit
CLIENT_CUSTOMMSG_PUBLISH_COLLECT_TASK = 531

-- 客户端打开发布任务总界面: unsigned cmdid
CLIENT_CUSTOMMSG_OPEN_PUBLISH_FORM = 532

-- 客户端打开查询发布任务界面: unsigned cmdid
CLIENT_CUSTOMMSG_OPEN_QUERY_EDIT_FORM = 533

-- 根据任务类型进行查询: unsigned cmdid int type
CLIENT_CUSTOMMSG_QUERY_TYPE_TASK = 534

-- 上一页下一页消息: unsigned cmd_id string prev_next int type int taskid
CLIENT_CUSTOMMSG_QUERY_PREV_NEXT_PAGE = 535

-- 通过任务ID查询所有任务信息
CLIENT_CUSTOMMSG_QUERY_SELECT_TASK_INFO = 536

-- 接受玩家编辑的任务: unsigned cmdid int taskid
CLIENT_CUSTOMMSG_ACCEPT_EDIT_TASK = 537

-- 客户端打开已接编辑任务界面: unsigned cmdid
CLIENT_CUSTOMMSG_OPEN_EDIT_FORM_ACCEPTED = 538

-- 放弃已经接受的编辑任务: unsigned cmdid int taskid widestr title
CLIENT_CUSTOMMSG_GIVEUP_EDIT_TASK = 539

-- 提交已经接受的编辑任务: unsigned cmdid int taskid
CLIENT_CUSTOMMSG_SUBMIT_EDIT_TASK = 540

-- 打开或关闭收集物品提交容器: unsigned cmdid int flag
CLIENT_CUSTOMMSG_SHOW_SUBMIT_BOX = 541

-- 客户端确认提交收集类任务: unsigned cmdid int taskid
CLIENT_CUSTOMMSG_SUBMIT_COLLECT_EDITTASK = 542

--客户端创建世界马车：unsigned cmd_id int string to
CLIENT_CUSTOMMSG_CREATE_WORLD_TRANS_TOOL = 550

-- 生活技能相关消息
-- 客户端上发合成消息：unsigned cmd_id, string formula_name
CLIENT_CUSTOMMSG_COMPOSITE = 600

-- 客户端上发乞讨请求：unsigned cmd_id
CLIENT_CUSTOMMSG_BEG = 601

-- 客户端上发算卦请求：unsigned cmd_id
CLIENT_CUSTOMMSG_SUANGUA = 602

-- 合成相关
-- 合成视图打开
CLIENT_CUSTOMMSG_OPEN_COMPOSITEBOX = 603
-- 合成视窗关闭
CLIENT_CUSTOMMSG_CLOSE_COMPOSITEBOX = 604
-- 合成视图的物品添加
CLIENT_CUSTOMMSG_COMPOSITEBOX_ADDITEM = 605
-- 合成视图的物品删除
CLIENT_CUSTOMMSG_COMPOSITEBOX_REMOVEITEM = 606
--合成物品自动放置
CLIENT_CUSTOMMSG_COMPOSITEBOX_AUTO_PUT = 607


-- 身份相关信息
-- 客户端上发请求激活身份：unsigned cmd_id, int origin_id
CLIENT_CUSTOMMSG_ACTIVE_ORIGIN = 700

-- 客户端上发头顶显示身份信息
CLIENT_CUSTOMMSG_SET_SHOW_ORIGIN = 701

-- 客户端发送获取奖励请求: unsigned cmd_id, int origin_id
CLIENT_CUSTOMMSG_GET_ORIGIN_PRIZE = 702

-- 客户端发送获取身份请求: unsigned cmd_id, int origin_id
CLIENT_CUSTOMMSG_GET_ORIGIN = 705

-- 客户端发送请求提示身份获取信息: unsigned cmd_id, int origin_id
CLIENT_CUSTOMMSG_REQUIRE_NOTIFY_ORIGIN = 706

-- 客户端确认消息 cmd_id, string submsg, int ret
CLIENT_CUSTOMMSG_CONFIRM = 710

--离线相关信息
--客户端上发离线设置申请信息：unsigned cmd_id, string off_id
CLIENT_CUSTOMMSG_OFFLINE_ADD_ACT = 800

--客户端上发离线设置申请信息：unsigned cmd_id, int index
CLIENT_CUSTOMMSG_OFFLINE_DEL_ACT = 801

--客户端上发开始挂机消息 unsigned cmd_id
CLIENT_CUSTOMMSG_START_OFFLIN = 802

--客户端请求领离线奖励
CLIENT_CUSTOMMSG_OFFLINE_GET_PRIZE = 803

--客户端请求放弃离线奖励
CLIENT_CUSTOMMSG_OFFLINE_GIVEUP_PRIZE = 804

--客户端设置离线执行托管事件 unsigned cmd_id
CLIENT_CUSTOMMSG_OFFLINE_SET_ACT = 805

--客户端剧情结束
CLIENT_CUSTOMMSG_END_MOVIE = 806

--客户端上发结束挂机消息 unsigned cmd_id
CLIENT_CUSTOMMSG_END_OFFLIN = 807

--客户端请求剧情
CLIENT_CUSTOMMSG_START_MOVIE = 808

--客户端上发开始挂机消息 unsigned cmd_id
CLIENT_CUSTOMMSG_START_OFFLINE_JOB = 809

--客户端上发结束挂机消息 unsigned cmd_id
CLIENT_CUSTOMMSG_END_OFFLINE_JOB = 810

--客户端上发离线交互BUFF消息 unsigned cmd_id
CLIENT_CUSTOMMSG_OFFLINE_JOB_BUFF = 811

--客户端上发请求同步位置消息 unsigned cmd_id
CLIENT_CUSTOMMSG_UPDATE_PLAYER_POS = 812

--客户端上发离线打工日志阅读消息 unsigned cmd_id date
CLIENT_CUSTOMMSG_UPDATE_LOG_READ = 813

--客户端上发开始离线打怪消息
CLIENT_CUSTOMMSG_START_OFFLINE_FIGHT = 814

--客户端上发停止离线打怪消息
CLIENT_CUSTOMMSG_END_OFFLINE_FIGHT = 815

--客户端上发离线交互消息
CLIENT_CUSTOMMSG_OFFLINE_INTERACT = 816

--客户端上发获得离线打赏奖励消息
CLIENT_CUSTOMMSG_OFFLINE_GET_FEE_PRIZE = 817

--客户端上发获得离线每日奖励消息
CLIENT_CUSTOMMSG_OFFLINE_GET_DAILY_PRIZE = 818

--客户端上发同意播放剧情
CLIENT_CUSTOMMSG_AGREE_MOVIE = 820

--客户端请求准备算卦摆摊
CLIENT_CUSTOMMSG_PREPARE_FORTUNETELLING = 853

--客户端请求开始算卦摆摊
CLIENT_CUSTOMMSG_BEGIN_FORTUNETELLING = 854

--客户端请求停止算卦
CLIENT_CUSTOMMSG_END_FORTUNETELLING = 855

--客户端请求要算卦
CLIENT_CUSTOMMSG_REQUEST_FORTUNETELLING = 856

--客户端点击开始钓鱼
CLIENT_CUSTOMMSG_BEGIN_FISHING = 850

--客户端点击开始钓鱼操作
CLIENT_CUSTOMMSG_OP_FISHING = 851

--客户端点击开始抄录操作
CLIENT_CUSTOMMSG_BEGIN_COPY = 852

--客户端查询环式任务状态是否可以承接信息: unsigned int cmd_id, string taskids
--CLIENT_CUSTOMMSG_REQUEST_TASK_STATE_CAN_ACCEPT = 900

-- 客户端完成新手帮助步骤: unsigned int cmd_id int stepID
CLIENT_CUSTOMMSG_COMPLETE_FRESHMAN_HELP_STEP = 950

-- 结束新手教程
CLIENT_CUSTOMMSG_END_FRESHMAN_HELP = 951

-- 客户端请求开始打铁小游戏
CLIENT_CUSTOMMSG_START_FORGE = 952

-- 客户端请求结束打铁小游戏
CLIENT_CUSTOMMSG_END_FORGE = 953

-- 客户端请求【命中测试】
CLIENT_CUSTOMMSG_TEST_FORGE_STEP_HIT = 954

-- 客户端请求【取消Step】
CLIENT_CUSTOMMSG_CANCEL_STEP = 955


--客户端跳上浮台请求绑定,unsigned int cmd_id, obj npc, float x, float y, float z, float orient
CLIENT_CUSTOMMSG_LINK_TO_FLOAT = 980
--客户端请求链接情况下的移动,unsigned int cmd_id, float off_x, float off_y, float off_z, float off_orient
CLIENT_CUSTOMMSG_LINK_MOVE = 981

--请求服务器同步到NPC的客户端位置(NPC必须已死亡),unsigned int cmd_id, PERSISTID npc, float x, float y, float z
CLIENT_CUSTOMMSG_SYN_NPC_POS = 990

-- 客户端请求剧本 unsigned int cmd_id, obj obj_ident, string drama_name, unsigned int chapter_id, unsigned int round_id
CLIENT_CUSTOMMSG_DRAMA = 1000

-- 客户端请求[救助]
CLIENT_CUSTOMMSG_RESCUE_REQUEST = 1001

-- 客户端请求[对某个玩家实施救助] string player_name
CLIENT_CUSTOMMSG_RESCUE_DEAL = 1002

-- 客户端请求[取消救助]
CLIENT_CUSTOMMSG_RESCUE_CANCEL = 1003

-- 客户端请求[某玩家的装备信息] string player_name
CLIENT_CUSTOMMSG_GET_EQUIP_INFO = 1004

-- 客户端请求[某玩家的基本信息] string player_name
CLIENT_CUSTOMMSG_GET_BASE_INFO = 1005

-- 客户端请求修为相关消息
CLIENT_CUSTOMMSG_FACULTY_INFO = 1006

-- 客户端通知服务端动画播放完毕 string ani_name
CLIENT_CUSTOMMSG_FIGHT_ENT_ANI_END = 1007

-- 拾取掉落金钱 : int msd_id, int index
CLIENT_CUSTOMMSG_GET_DROP_MONEY_ITEM = 1008

--客户端播放剧情黑幕出现
CLIENT_CUSTOMMSG_SCENARIO_BLACK = 1009

-- 客户端请求镶嵌，消息 int msg_id, int inlay_type, int tool_box_pos, int inlay_pos
CLIENT_CUSTOMMSG_REQUEST_INLAY = 1010

-- 客户端回应镶嵌，消息 int msg_id, int inlay_ok_or_cancel
CLIENT_CUSTOMMSG_RESPONE_INLAY = 1011

--客户端请求解除玩家与火炮npc的绑定
CLIENT_CUSTOMMSG_STOP_BIND = 1012

--客户端请求使用NPC技能
CLIENT_CUSTOMMSG_USE_NPC_SKILL = 1013

-- 客户端请求创建公会 int msg_id, wstring guild_name, wstring guild_purpose
CLIENT_CUSTOMMSG_GUILD = 1014

-- 门派战客户端请求消息
CLIENT_CUSTOMMSG_SCHOOL_WAR = 1015

--客户端上发，取消轻功等增益BUFFER，消息: int msd_id, string buffer_id
CLIENT_CUSTOMMSG_REMOVE_BUFFER = 1021

-- 客户端请求设置任务跟踪信息, 消息: int msg_id, int task_id, int row
--CLIENT_CUSTOMMSG_UPDATE_TASK_TRACE = 1022

--开始内功比拼，消息格式：unsigned msgid, Object target
CLIENT_CUSTOMMSG_REQ_START_NEIGONG_COMPARE = 230

--内功比拼玩家下注，消息格式：unsigned msgid
CLIENT_CUSTOMMSG_REQ_CHIP_IN = 231

--请求内功比拼, 消息格式: unsigned msgid, int requesttype, obj target
CLIENT_COUSOMMSG_REQUEST_NEIG_PK = 232

--加入内功比拼 msgid, obj target
CLIENT_CUSTOMMSG_JOIN_NEIG_PK = 233

--客户端向服务端同步时间
CLIENT_CUSTOMMSG_AS_TIME = 1020