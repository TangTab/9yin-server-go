
--require("task\\task_define")
--使用的表格
REC_TASK_ACCEPT	= "Task_Accepted"
REC_TASK_PLAN	= "Task_Record"
REC_TASK_COMPLETE	= "Task_Completed"
REC_TASK_DAILY	= "TaskDayRec"
random_round_rec_name = "RoundRandomTaskRec"
choice_round_rec_name = "RoundChoiceTaskRec"

--任务子功能类型
subtype_hunter		= 1				-- 猎杀类型
subtype_collect		= 2				-- 收集类
subtype_interactive	= 3				-- 交互类型
subtype_convoy		= 4				-- 护送类
subtype_quest		= 5				-- 问答类
subtype_story		= 6				-- 故事类
subtype_useitem		= 7				-- 使用道具类
subtype_special		= 8				-- 特殊任务类型

-- 任务线分类
task_line_undefined	= 0				-- 未分类
task_line_main		= 1				-- 主线
task_line_wuxue		= 2				-- 武学
task_line_shenghuo	= 3				-- 生活
task_line_menpai	= 4				-- 门派
task_line_side		= 5				-- 支线
task_line_clone		= 6				-- 副本
task_line_cycle		= 7				-- 环式

-- 任务接受表结构 <Task_Accepted>
accept_rec_id		= 0				-- 任务id
accept_rec_line		= 1				-- 任务线
accept_rec_server	= 2				-- 服务器名称
accept_rec_submit_scene	= 3				-- 提交场景名称
accept_rec_submit_npc	= 4				-- 提交npc名称
accept_rec_title	= 5				-- 任务标题
accept_rec_status	= 6				-- 任务状态
accept_rec_track_flag	= 7				-- 任务追踪标志

--任务进度表结构 <Task_Record>
task_rec_id		= 0				-- 任务id
task_rec_subtype	= 1				-- 子任务类型
task_rec_index		= 2				-- 任务子id
task_rec_order		= 3				-- 进度完成顺序
task_rec_curnum		= 4				-- 当前进度
task_rec_maxnum		= 5 				-- 最大进度
task_rec_key		= 6				-- 关键数据,如使用道具config,护送npc,等
task_rec_scene		= 7				-- 目标场景config
task_rec_trackinfo	= 8				-- 追踪信息


-- 环式随机表结构
rand_col_parent_id	= 0				-- 主任务id
rand_col_child_id	= 1				-- 子任务id
rand_col_turn		= 2				-- 当前轮
rand_col_max_turn	= 3				-- 最大轮
rand_col_cycle		= 4				-- 当前环
rand_col_max_cycle	= 5				-- 最大环
rand_col_giveup_nums	= 6				-- 当日子任务放弃次数
rand_col_firs_accept	= 7				-- 当日子任务第一次承接时间


