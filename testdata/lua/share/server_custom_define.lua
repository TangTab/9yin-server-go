TIPSTYPE_SYSTEMINFO = 0	--系统信息(引擎模块发送的提示信息用，逻辑模块不要使用)
TIPSTYPE_SYSTEMNO = 1	--系统提示(客户端显示加[提示]前缀)
TIPSTYPE_TIPSINFO = 2	--主画面向上滚动的提示信息
TIPSTYPE_GMINFO = 99	--GM调试信息
TIPSTYPE_DEBUGINFO = 100	--开发调试信息

--系统消息，格式：int msgid, int tipstype, string stringid, ...(参数表)
SERVER_CUSTOMMSG_SYSINFO	= "sysinfo"
--服务器同步时间，格式：int msgid, int64 server_time
SERVER_CUSTOMMSG_ASYN_TIME	= "asyntime"
--NPC发出的聊天信息，格式：int msgid, string stringid, ...(参数表)
SERVER_CUSTOMMSG_SPEECH		= "speech"

--菜单消息
--开始菜单，格式：int msgid
SERVER_CUSTOMMSG_BEGIN_MENU	= "begin_menu"
--添加Title，格式：int msgid, string stringid, ...(参数表)
SERVER_CUSTOMMSG_ADD_TITLE	= "add_title"
--添加选择项，格式：int msgid, int func, string stringid, ...(参数表)
SERVER_CUSTOMMSG_ADD_MENU	= "add_menu"
--结束菜单，格式：int msgid, object target
SERVER_CUSTOMMSG_END_MENU	= "end_menu"
--关闭当前菜单，格式：int msgid
SERVER_CUSTOMMSG_CLOSE_MENU	= "close_menu"

--通知客户端弹出一个MessageBox，消息格式：
--unsigned msgid, int msgboxid, int count(按纽个数), ...(按纽的提示信息文本ID), string stringid(提示信息文本ID), ...(提示信息参数)
SERVER_CUSTOMMSG_SHOWMESSAGEBOX	= "show_msgbox"
--通知客户端弹出一个InputBox，消息格式：
--unsigned msgid, int inputboxid, int type(0:只能数值,1:文本), string stringid(提示信息文本ID), ...(提示信息参数)
SERVER_CUSTOMMSG_SHOWINPUTBOX	= "show_inputbox"

--向客户端发送自定义的虚拟表格的内容，消息格式：
--unsigned msgid, string recname, ...(第一行数据), ..., ...(第n行数据)
SERVER_CUSTOMMSG_VIRTUALREC_ADD			= "virtualrec_add"
--通知客户端清除自定义的虚拟表格的内容，消息格式：unsigned msgid, string recname
SERVER_CUSTOMMSG_VIRTUALREC_CLEAR		= "virtualrec_clear"
--通知客户端更新自定义的虚拟表格的某行某列的内容，消息格式：
--unsigned msgid, string recname, int row(行号), int col(列号),(数据)
SERVER_CUSTOMMSG_VIRTUALREC_UPDATE		= "virtualrec_update"
--通知客户端更新自定义的虚拟表格的某行的内容，消息格式：
--unsigned msgid, string recname, int row(行号), ...(行数据)
SERVER_CUSTOMMSG_VIRTUALREC_UPDATE_ROW	= "virtualrec_updaterow"
--通知客户端删除自定义的虚拟表格的某行，消息格式：
--unsigned msgid, string recname, int row(行号)
SERVER_CUSTOMMSG_VIRTUALREC_REMOVE_ROW	= "virtualrec_removerow"

--聊天消息
SERVER_CUSTOMMSG_CHAT			= "chat"

--动作消息，消息格式：unsigned msgid, object target, string action, int loop(0|1)
SERVER_CUSTOMMSG_ACTION			= "action"

--执行技能位移动作，消息格式：unsigned msgid, object target, object lock_target, string action,
--								float x1, float y1, float z1, float orient1,
--								float x2, float y2, float z2, float orient2, float sepdist
SERVER_CUSTOMMSG_SKILL_ACTION_WITH_TRANSLATE = "skill_action_translate"

--执行技能位移动作，消息格式：unsigned msgid, object target, object lock_target, string action, float begx, float begy, float begz, float orient, float dist
SERVER_CUSTOMMSG_SKILL_ACTION = "skill_action"

--执行套路动作，消息格式：unsigned msgid, object target, ...(动作列表)
SERVER_CUSTOMMSG_TAOLU_START	= "taolu_start"

--结束套路动作，消息格式：unsigned msgid
SERVER_CUSTOMMSG_TAOLU_STOP		= "taolu_stop"

--自定义特效，消息格式：unsigned msgid, object target, ...(由客户端程序确定)
SERVER_CUSTOMMSG_EFFECT			= "effect"

--开始念咒，消息格式：unsigned msgid, int tick(时间，单位ms)
SERVER_CUSTOMMSG_BEGIN_CURSE	= "begin_curse"

--开始念咒，消息格式：unsigned msgid
SERVER_CUSTOMMSG_END_CURSE		= "end_curse"

--删除子弹特效，消息格式：unsigned msgid, object owner
SERVER_CUSTOMMSG_DESTROY_BULLET	= "destroy_bullet"

--用户交互请求，消息格式：unsigned msgid, int request_type, wstring name(请求者角色名称), ...(参数，根据request_type的不同而不同)
SERVER_CUSTOMMSG_REQUEST		= "request"

--下发某个角色的装备信息，消息格式：unsigned msgid, wstring name, wstring roleinfo, string equips_info_1, ..., string equips_info_n
SERVER_CUSTOMMSG_LOOK_EQUIPS_ECHO	= "look_equip_echo"

--下发某个角色的物品信息，消息格式：unsigned msgid, wstring name, string item_info
SERVER_CUSTOMMSG_LOOK_ITEM_ECHO		= "look_item_echo"

--通知客户端打开商店界面，消息格式：string msgid, string shop_id
SERVER_CUSTOMMSG_OPEN_SHOP		= "openshop"






-- 通知客户端打开任务管理界面， 消息格式: string msgid, int task_id
SERVER_CUSTOMMSG_OPEN_TASKMANAGER =	"open_task_manager_board"


-- 通知客户端打开未完成任务信息界面，消息格式: string msgid, int task_id, object npc, int recid
SERVER_CUSTOMMSG_NOT_COMPLETE_TASK_INFO = "open_task_not_complete_info"

-- 通知客户端修改任务npc头顶标记, 消息格式: string msgid, object npc, int flagtype
SERVER_CUSTOMMSG_NOTICE_NPCFLAG = "change_tasknpc_flag"

-- 通知客户端打开环式任务的接受信息界面,消息格式: string msgid, int task_id, object npc
SERVER_CUSTOMMSG_OPEN_RANDOM_ACCEPT_TASKINFO = "open_random_task_accept_info"


-- 打开选择类任务对话框，消息格式: string msgid, int taskid, string textid
SERVER_CUSTOMMSG_SHOW_SELECT_DIALOG = "show_task_select_dialog"

-- 通知客户端维护npc的任务特效，消息格式: string msgid, object npc, int type
SREVER_CUSTOMMSG_NOTICE_NPCEFFECT = "change_tasknpc_effect"

-- 通知客户端打开共享任务的接受菜单，消息格式: string msg_id, int task_id, wide_str sharer_name
SERVER_CUSTOMMSG_SHOW_SHARE_TASK_MENU = "show_share_task_menu"

-- 通知客户端在某点播放声音,消息格式: string msg_id, string sound_name, float x, float y, float z
SERVER_CUSTOMMSG_PLAY_SOUND	= "play_point_sound"

-- 发送要显示的编辑任务信息串给客户端: string msg_id, int id, wstring title, int hours, wstring name, ...
SERVER_CUSTOMMSG_SHOW_QUERY_EDIT_TASK_INFO = "show_query_edit_task_info"

-- 发送查询的任务信息给客户端: string msg_id, int type, string require, string prop, string item_info
SERVER_CUSTOMMSG_SEND_SELECT_TASK_INFO = "show_select_edittask_info"

-- 客户端NPC说话消息: string msg_id, object npc, string talk_id
SERVER_CUSTOMMSG_NPC_TALK = "npc_talk_text"

-- 通知客户端打开一个确认对话框: string msg_id, string tetx_id
SERVER_CUSTOMMSG_OPEN_CONFIRM_BOX = "show_task_confirm_box"

-- 通知客户端打开提交收集物品容器界面: string msg_id, int task_id(task_id为0代表关闭)
SERVER_CUSTOMMSG_OPENSUBMIT_COLLECT_BOX = "open_submit_collect_box"

-- 通知客户端只对自己隐藏某NPC: string msg_id, object npc
SERVER_CUSTOMMSG_HIDE_NPC_ONLY_SELF	= "hide_npc_for_self"

-- 通知客户端刷新任务跟踪信息 string msg_id
SERVER_CUSTOMMSG_REFRESH_TASK_TRACKINFO = "refresh_track_info"

-- 通知客户端显示一个被隐身的对象(纯客户端隐身) string msg_id obj
--SERVER_CUSTOMMSG_SHOW_HIDE_NPC = "show_hide_npc"

-- 刷新客户端指定对象的头顶信息 string msg_id obj
SERVER_CUSTOMMSG_REFRESH_HEAD_GAME = "refresh_head_game"

--通知客户端发信结果，消息格式; unsigned msgid，int result
SERVER_CUSTOMMSG_SEND_LETTER_RESULT = "send_letter_result"

--通知客户端提取金钱附件成功，消息格式; unsigned msgid，int ntype
SERVER_CUSTOMMSG_GET_LETTER_MONEY_SUCCESS = "get_letter_money_success"

--通知客户端师徒候选对话框页数信息，消息格式:int Flag,int MaxPages,int CurPage,int count
SERVER_CUSTOMMSG_SEND_TEACHPREN_CANDIDATE_PAGE_INFO = "send_teachpren_candidate_list"

--玩家点击NPC弹出拜师对话框 消息格式: unsigned msgid,int DlgType
SERVER_CUSTOMMSG_POP_TEACHERNPC_DLG  = "pop_teachernpc_dlg"

--通知客户端弹出推荐拜师提示对话框，消息格式: unsigned msgid,
SERVER_CUSTOMMSG_NOTICE_COMMEND_TEACHER_DLG = "notice_commend_teacher_dlg"

--通知客户端弹出师傅评价对话框，消息格式: unsigned msgid,
SERVER_CUSTOMMSG_POP_JUDGETEACHER_DLG  = "pop_judgeteacher_dlg"

--通知客户端弹出推荐拜师确认对话框，消息格式: unsigned msgid,
SERVER_CUSTOMMSG_CONFIRM_COMMEND_TEACHER_DLG = "confirm_commend_teacher_dlg"


-- 通知客户端弹出关于婚姻的配偶的输入框
SERVER_CUSTOMMSG_INPUT_COMMEND_MARRIAGE_DLG = "input_commend_marriage_dlg"

-- 通知客户端加入一个骰子竞争道具,消息格式：unsigned msgid, object npc, object item, string configid, int time
SERVER_CUSTOMMSG_PUSH_ROLL_ITEM = "push_roll_item_info"

-- 通知客户端打开队长分配界面,消息格式：unsigned msgid
SERVER_CUSTOMMSG_POP_ASSIGN_DLG = "pop_team_assign_dlg"

-- 通知客户端弹出随机商店界面
SERVER_CUSTOMMSG_SHOW_RANDOMSHOP  = "show_randomshop"

-- 通知客户端弹出跑商商店界面
SERVER_CUSTOMMSG_SHOW_BUSINESSSHOP = "show_businessshop"

-- 通知客户端弹出跑商商店查询界面
SERVER_CUSTOMMSG_SHOW_BUSINESSQUERYDIALOG = "show_businessquerydialog"

-- 通知客户端弹出跑商出售商品的价格，消息格式：int money
SERVER_CUSTOMMSG_SHOW_MONEY_BUSINESSITEM = "show_money_businessitem"

-- 通知客户端打开包裹界面
SERVER_CUSTOMMSG_OPEN_TOOLBOX = "open_tool_box"

--通知客户端显示动画,消息格式：unsigned msgid,string animationname
SERVER_CUSTOMMSG_SHOW_ANIMATION = "show_animation"

--通知客户端关闭动画,消息格式:unsigned msgid,string animationname
SERVER_CUSTOMMSG_DEL_ANIMATION  = "del_animation"

--通知客户端玩家第一次进入游戏，消息格式：unsigned msgid, object target, ...(由客户端程序确定)
SERVER_CUSTOMMSG_BORN			= "born"

-- 通知客户端开始某个小游戏，消息格式：string msgid, string game_name
SERVER_CUSTOMMSG_MINIGAME_START = "game_minigame_start"

-- 通知客户端结束某个小游戏
SERVER_CUSTOMMSG_MINIGAME_STOP = "game_minigame_stop"

-- 通知客户端玩家开始站桩小游戏,消息格式：unsigned msgid, int time
SERVER_CUSTOMMSG_GAME_START_STUMP = "game_stump_start"

-- 通知客户端玩家终止站桩小游戏,消息格式：unsigned msgid
SERVER_CUSTOMMSG_GAME_STOP_STUMP  = "game_stump_stop"

--通知客户端计时,消息格式:unsigned msgid, int ntype =,int time
SERVER_CUSTOMMSG_START_TIMER  = "start_timer"

--通知客户端马上要进行切场景，可以准备进行预处理unsigned msgid, float x, float y, float z
SERVER_CUSTOMMSG_SWITCH_SCENE_BEGIN = "switch_scene_begin"

--通知客户端在规定时间内完成移动模拟,unsigned msgid, float fTime
SERVER_CUSTOMMSG_NPC_MOVE  = "npc_move"

--打开客户端界面string msg, int compose_type
SERVER_CUSTOMMSG_OPEN_COMPOSE  = "open_compose"

-- 通知client端：获取了一个新身份: string msg_id, int origin_id
SERVER_CUSTOMMSG_GET_A_NEW_ORIGIN = "get_a_new_origin"

-- 通知客户端刷新身份状态: string msg_id, int command_id
SERVER_CUSTOMMSG_REFRESH_ORIGIN	= "refresh_origin_condition"

-- 通知客户端播放生活技能，消息格式：unsigned msgid, object target, string action, int loop(0|1)
SERVER_CUSTOMMSG_SHENGHUO_ACTION = "shenghuo_action"

-- 通知客户端，生活技能释放流程结束，command、flow_name、result
SERVER_CUSTOMMSG_LIFE_SKILL_FLOW_HIT = "lift_skill_flow_hit"

--通知客户端有离线奖励
SERVER_CUSTOMMSG_HAVE_OFFLINE_PRIZE  = "have_offline_prize"

--通知客户端玩家进入场景准备就绪
SERVER_CUSTOMMSG_PLAYER_ONREADY = "player_enter_scene_onready"

--通知客户端玩家继续连接
SERVER_CUSTOMMSG_PLAYER_ONCONTINUE = "player_enter_scene_oncontinue"

-- 通知客户端玩家显示模式对话框，内容为提示信息：string string_id, ...
SERVER_CUSTOMMSG_SHOW_MODEL_DIALOG_INFO = "show_model_dialog_info"

-- 通知客户端开始一段剧情, 参数格式: string cmd_id, Object NPC, int movie_id, int focus_type
SERVER_CUSTOMMSG_START_MOVIE = "start_movie"

--通知客户端有一段剧情等待播放，参数格式：string cmd_id, Object NPC, int movie_id
SERVER_CUSTOMMSG_MOVIE_REQUEST = "movie_request"

--通知客户端新进入一个场景,显示特效:string city_name
SERVER_CUSTOMMSG_ENTER_NEW_SCENE = "enter_new_scene"

-- 客户端查询任务是否能承接的结果
SERVER_CUSTOMMSG_RESPONSE_TASK_CAN_ACCEPT = "task_can_accept"

--通知客户端未完成离线任务
SERVER_CUSTOMMSG_NOT_FINISH_OFFLINEWORK = "not_finish_offlinework"

--通知客户端显示离线日志
SERVER_CUSTOMMSG_SHOW_OFFLINE_JOB_LOG = "show_offline_job_log"

--通知客户端显示离线每日日志
SERVER_CUSTOMMSG_SHOW_OFFLINE_JOB_DAILY = "show_offline_job_daily"

--通知客户端隐藏离线每日日志
SERVER_CUSTOMMSG_HIDE_OFFLINE_JOB_DAILY = "hide_offline_job_daily"

--通知客户端显示离线交互界面
SERVER_CUSTOMMSG_SHOW_OFFLINE_JOB_INTERACT = "show_offline_job_interact"

--通知客户端隐藏离线交互界面
SERVER_CUSTOMMSG_HIDE_OFFLINE_JOB_INTERACT = "hide_offline_job_interact"

--通知客户端显示离线交互奖励界面
SERVER_CUSTOMMSG_SHOW_OFFLINE_JOB_INTERACT_PRIZE = "show_offline_job_interact_prize"

--通知客户端隐藏离线离线交互奖励界面
SERVER_CUSTOMMSG_HIDE_OFFLINE_JOB_INTERACT_PRIZE = "hide_offline_job_interact_prize"

--通知客户端同步主角玩家当前位置
SERVER_CUSTOMMSG_UPDATE_PLAYER_POS = "update_player_pos"

--通知客户端同步一定范围内可视对象位置
SERVER_CUSTOMMSG_UPDATE_VISUAL_OBJ_POS = "update_visual_obj_pos"

--通知客户端学习到新的生活职业
SERVER_CUSTOMMSG_GET_A_NEW_LIFEJOB = "get_a_new_lifejob"

-- 通知客户端打开新手帮助界面 int task_id
SERVER_CUSTOMMSG_OPEN_HELP_FORM	= "open_help_form"

--通知客户端显示二次上线的离线效果
SERVER_CUSTOMMSG_OFFLINEWORK_ONLINE_EFFECT = "offlinework_online_effect"

--通知客户端学习到新的生活职业
SERVER_CUSTOMMSG_GET_A_NEW_SKILL = "get_a_new_skill"

-- 通知客户端启动倒计时，格式: string cmd, int type, int id, int time
SERVER_CUSTOMMSG_BEGIN_TIME_LIMIT_SHOW = "time_limit_show"

-- 通知客户端新手帮助的步骤 string cmd, int stepID
SERVER_CUSTOMMSG_FRESHMAN_HELP_STEP = "freshman_help_step"

-- 通知客户端交互NPC和玩家开始做绑定的动作: string cmd, object player, object npc,
--											string player_action, string npc_action
SERVER_CUSTOMMSG_START_INTERACTIVE_ACTION =	"start_interactive_action"

-- 通知客户端交互NPC和玩家结束做绑定的动作: string cmd, object player, object npc
SERVER_CUSTOMMSG_END_INTERACTIVE_ACTION	= "end_interactive_action"

-- 通知客户端玩家绑定复活点成功，显示动画
SERVER_CUSTOMMSG_BAND_RELIVE_POS =	"band_relive_pos"

-- 通知客户端玩家复活
SERVER_CUSTOMMSG_RELIVE_END	= "relive_end"

-- 通知客户端打铁游戏开始
SERVER_CUSTOMMSG_FORGE_STARTED = "forge_started"

-- 通知客户端打铁游戏结束
SERVER_CUSTOMMSG_FORGE_STOPPED = "forge_stopped"

-- 通知客户端Forge游戏timeout
SERVER_CUSTOMMSG_FORGE_TIMEOUT = "forge_timeout"

-- 通知客户端Step时间测试的差值
SERVER_CUSTOMMSG_FORGE_STEP_DIFF = "forge_step_diff"

-- 通知客户端取消某个Step
SERVER_CUSTOMMSG_CANCEL_STEP = "forge_cancel_step"

-- 通知客户端打开路标界面
SERVER_CUSTOMMSG_SHOW_ROADSIGN = "show_roadsign"

-- 通知客户端接收剧本信息消息
SERVER_CUSTOMMSG_SHOW_DRAMA_INFO = "show_drama_info"

-- 通知客户端[查询某玩家基本信息的结果]
SERVER_CUSTOMMSG_SHOW_BASE_INFO	= "show_base_info"

-- 通知客户端[查询某玩家装备信息的结果]
SERVER_CUSTOMMSG_SHOW_EQUIP_INFO = "show_equip_info"

--通知客户端某npc头上显示字
SERVER_CUSTOMMSG_NPC_SAY  = "npc_say"

--通知客户端玩家头上显示字
SERVER_CUSTOMMSG_PLAYER_SAY = "player_say"

--可以摆摊
SERVER_CUSTOMMSG_CAN_FORTUNETELLINGSTALL = "can_fortunetellingstall"

--摆摊成功
SERVER_CUSTOMMSG_FORTUNETELLINGSTALL_OKAY = "fortunetellingstall_okay"

--撤销摆摊
SERVER_CUSTOMMSG_FORTUNETELLINGSTALL_CANCELED = "fortunetellingstall_canceled"

--询问是否需要算卦
SERVER_CUSTOMMSG_NEED_FORTUNETELLING = "need_fortunetelling"

--可以摆摊
SERVER_CUSTOMMSG_CAN_STALL = "can_stall"

-- 通知客户端进行场景内传送
SERVER_CUSTOMMSG_GOTO = "scene_goto_event"

-- 摆摊成功
SERVER_CUSTOMMSG_STALL_OKAY = "stall_okay"

-- 摆摊已取消
SERVER_CUSTOMMSG_STALL_CANCELED = "stall_canceled"

-- 通知客户端怪物开始使用技能读条: string cmd_id, object npc, int time
SERVER_CUSTOMMSG_BEGIN_PAUSE_SKILL = "begin_pause_skill"

-- 通知客户端怪物结束使用技能读条: string cmd_id, object npc
SERVER_CUSTOMMSG_END_PAUSE_SKILL = "end_pause_skill"

-- 通知农作物特效改变
SERVER_CUSTOMMSG_CROP_EFFECT_CHANGE = "crop_effect_change"

-- 通知npc播放开始战斗动画:string ani_name
SERVER_CUSTOMMSG_NPC_FIGHT_ANI_START = "npc_fight_ani_start"

-- 通知客户端修为相关消息: string subCmd param1 ...
SERVER_CUSTOMMSG_FACULTY_MSG = "faculty_msg"


--通知客户端播放机关被触发的过程
SERVER_CUSTOMMSG_MACNPC_SPRING_ACT_START = "machine_npc_spring_act_start"

--通知客户端播放机关触发动作结束
SERVER_CUSTOMMSG_MACNPC_SPRING_ACT_END = "machine_npc_spring_act_end"

--通知客户端打开合成
SERVER_CUSTOMMSG_OPEN_COMPOSITEBOX = "open_composite_box"

-- 通知客户端攻击特效
SERVER_CUSTOMMSG_TRIGGER_SKILL_EFFECT = "trigger_skill_effect"

-- 通知客户端技能图标闪烁 string msg, int skill_index, int start_or_end
SERVER_CUSTOMMSG_SKILL_ICON_FLASH = "skill_icon_flash"

-- 战斗教学任务 string msg, int subType(1,开始, 2,结束)
SERVER_CUSTOMMSG_FIGHT_GUID = "task_fight_guid_msg"

--通知客户端播放音乐 string msg int flag(0关闭1播放) string musicname(音乐路径)
SERVER_CUSTOMMSG_PLAY_MUSIC  = "play_music"

-- 通知客户端播放或者停止采集特效 string msg, obj npc, int type(0|1), int time(-1), string effectName(""), string effectPos("")
SERVER_CUSTOMMSG_GATHER_EFFECT = "gather_effect"

-- 通知客户端打开镶嵌道具界面 string msg
SERVER_CUSTOMMSG_OPEN_FORM_INLAY = "open_form_inlay"

-- 任务系统主消息 string msg， int subType.....
SERVER_CUSTOMMSG_TASK_MSG = "task_msg"

--显示确认对话框 string msg, string info
SERVER_CUSTOMMSG_CONFIRM_MSG = "confirm_box"

--显示确认对话框 string msg, string info
SERVER_CUSTOMMSG_TIPS_MSG = "tips_box"

-- 通知客户端离开副本倒计时 string msg， int time.....
SERVER_CUSTOMMSG_OUT_CLONE_TIME = "out_clone_time"

-- 通知客户端显示进入副本菜单 string msg, int npcid, int cloneid,....
SERVER_CUSTOMMSG_SHOW_CLONE_MENU = "show_clone_menu"

-- 通知客户端关闭进入副本菜单 string msg....
SERVER_CUSTOMMSG_CLOSE_CLONE_MENU = "close_clone_menu"

-- 询问客户端是否同步副本进度 string msg....
SERVER_CUSTOMMSG_INPHASE_CLONE_MENU = "inphase_clone_menu"

-- 客户端显示通关引导界面 string msg, 关卡号,....
SERVER_CUSTOMMSG_CLONE_LEAD_OPEN = "clone_lead_form_open"

-- 客户端关闭通关引导界面 string msg, 关卡号,....
SERVER_CUSTOMMSG_CLONE_LEAD_CLOSE = "clone_lead_form_close"

-- 客户端改变通关信息 string msg, 机关号，关卡号，关卡触发标记,....
SERVER_CUSTOMMSG_CLONE_LEAD_CHANGE = "clone_lead_form_change"

-- 通知客户端对武学按钮进行闪烁
SERVER_CUSTOMMSG_WUXUE_FLASH = "wuxue_flash"

-- 客户端打开操作npc技能界面
SERVER_CUSTOMMSG_DRIVE_SKILL_OPEN = "drive_skill_open"

-- 客户端显示原地保护时间
SERVER_CUSTOMMSG_CONVOY_TIME_LIMIT = "convoy_time_limit"

-- 公会相关下发消息主消息
SERVER_CUSTOMMSG_GUILD = "guild"

-- 开始空中技能进行浮空
SERVER_CUSTOMMSG_BEGIN_AIR_STOP = "begin_air_stop"

-- 结束空中技能浮空
SERVER_CUSTOMMSG_END_AIR_STOP = "end_air_stop"

-- 修为传授消息
SERVER_CUSTOMMSG_IMPART = "faculty_impart"

-- 打开或关闭掉落视图
SERVER_CUSTOMMSG_SHOW_HIED_DROP_VIEW = "show_hide_drop_view"

-- 进行特殊载具骑乘操作
SERVER_CUSTOMMSG_START_SPECIAL_RIDE = "start_special_ride"

-- 取消特殊载具骑乘操作
SERVER_CUSTOMMSG_END_SPECIAL_RIDE = "end_special_ride"


-- 开启世界驿站界面
SERVER_CUSTOMMSG_START_WORLD_TRANS = "start_world_trans"

-- 关闭世界驿站界面
SERVER_CUSTOMMSG_END_WORLD_TRANS = "end_world_trans"

-- 发布公告 type, info_id ...
SERVERMSG_PUSLISH_NEWS = "Publish_news"

-- 通知客户端打开镶嵌窗口 string msg_id, string formname, object npc
SERVER_CUSTOMMSG_SHOW_ENCHASE_FORM = "show_enchase_form"

-- 发送选中装备的视图和索引 string msg_id, string formname, int view_id, int index
SERVER_CUSTOMMSG_ENCHASE_EQUIP_SELECT = "enchase_equip_select"

-- 发送选中装备镶嵌信息更新 string msg_id, string formname
SERVER_CUSTOMMSG_REFRESH_ENCHASE_ITEMS = "refresh_enchase_items"

--通知客户端马车继续行驶
SERVER_CUSTOMMSG_TRANSTOOL_CONTINUE_MOVE = "transtool_continue_move"

--发送显示时间限制窗体 格式：string form_name, int time (参数表)
SERVER_CUSTOMMSG_SHOW_TIME_LIMIT_FORM = "show_time_limit_form"

--关闭显示时间限制窗体 格式：string form_name(参数表)
SERVER_CUSTOMMSG_CLOSE_TIME_LIMIT_FORM = "close_time_limit_form"

--发送显示计数窗体
SERVER_CUSTOMMSG_SHOW_COUNT_LIMIT_FORM = "show_count_limit_form"

--发送增加计数窗体中的计数值
SERVER_CUSTOMMSG_INC_COUNT_LIMIT_FORM = "inc_count_limit_form"

--发送关闭计数窗体
SERVER_CUSTOMMSG_CLOSE_COUNT_LIMIT_FORM = "close_count_limit_form"

--发送服务器时间
SERVER_CUSTOMMSG_ASYN_SERVER_TIME = "asyn_server_time"

--通知客户端刷新可挑战列表
SERVER_CUSTOMMSG_CHALLENGE_RANK_LIST_UPDATE = "challenge_rank_list_update"

--通知客户端总排行表
SERVER_CUSTOMMSG_CHALLENGE_RANK_UPDATE = "challenge_rank_update"

-- 门派战
SERVER_CUSTOMMSG_SCHOOL_WAR = "school_war"

-- 头顶冒字
SERVER_CUSTOMMSG_HEAD_TEXT = "head_text"

-- 开始水下呼吸计算
SERVER_CUSTOMMSG_BEGIN_BREATH_IN_WATER = "begin_breath_in_water"

-- 停止水下呼吸计算
SERVER_CUSTOMMSG_END_BREATH_IN_WATER = "end_breath_in_water"

--挑战朝向调整
SERVER_CUSTOMMSG_CHALLENGE_ORIENT = "challenge_orient"

--通知客户端 显示离线挂机主界面
SERVER_CUSTOMMSG_OFFLINE_ONHAND_SHOWFORM = "showform_offline_onhand"


-- 打开邮寄件箱
SERVER_CUSTOMMSG_OPEN_MAIL_BOX = "open_mail_box"

-- 打开急件邮箱
SERVER_CUSTOMMSG_OPEN_URGENT_MAIL_BOX = "open_urgent_mail_box"

--通知打开拾取礼包窗口
SERVER_CUSTOMMSG_PICK_ITEM  = "pick_item"

