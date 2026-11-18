-- ? 或者 h 表示帮助,列出所有参数说明
-- g 或者 表示放弃任务 任务编号
-- a 或者 表示接受任务 任务编号
-- p 或者 表示显示任务表 1 任务承接表 2 任务进度表 3 任务完成表
-- s 表示设置表
-- c 或者 表示清空任务表
function task_help(gm, type)
	-- 打印帮助
	nx_sysinfo(gm, type, "0", nx_widestr("========================gm命令[task]==============================="));
	nx_sysinfo(gm, type, "0", nx_widestr("gm命令[task] 参数表说明: "));
	nx_sysinfo(gm, type, "0", nx_widestr("(1), 比如: task ? 表示显示帮助"));
	nx_sysinfo(gm, type, "0", nx_widestr("     比如: task g 24 表示放弃24号任务"));
	nx_sysinfo(gm, type, "0", nx_widestr("     比如: task g 表示放弃所有任务"));
	nx_sysinfo(gm, type, "0", nx_widestr("     比如: task a 24 表示强制承接24号任务"));
	nx_sysinfo(gm, type, "0", nx_widestr("     比如: task p 1 表示显示承接任务表内容"));
	nx_sysinfo(gm, type, "0", nx_widestr("     比如: task s 2 0 4 1 表示设置任务进度表第0行4列为1"));
	nx_sysinfo(gm, type, "0", nx_widestr("(2), ? 或 h 用来显示帮助信息"));
	nx_sysinfo(gm, type, "0", nx_widestr("(3), a [id] 即accept 表示接受任务 task a 24 表示强制接受24号任务"));
	nx_sysinfo(gm, type, "0", nx_widestr("(4), p [id] 即print 表示打印任务表 id为整数说明如下:"));
	nx_sysinfo(gm, type, "0", nx_widestr("     1:承接任务表           Task_Accepted"));
	nx_sysinfo(gm, type, "0", nx_widestr("     2:任务进度表           Task_Record"));
	nx_sysinfo(gm, type, "0", nx_widestr("     3:任务完成表           Task_Completed"));
	nx_sysinfo(gm, type, "0", nx_widestr("     4:可重复任务表         TaskRepeatRec"));
	nx_sysinfo(gm, type, "0", nx_widestr("     5:隐藏npc表            Task_Hide_Rec"));
	nx_sysinfo(gm, type, "0", nx_widestr("     6:表示完成的副本任务表 CmpltCloneTaskRec"));
	nx_sysinfo(gm, type, "0", nx_widestr("(5), s [id row col value] 即set 表示设置任务表 "));
	nx_sysinfo(gm, type, "0", nx_widestr("     id同上 row即行数,col即列,value即要修改的值"));
	nx_sysinfo(gm, type, "0", nx_widestr("(6), c 即clear 清空所有任务表"));
	nx_sysinfo(gm, type, "0", nx_widestr("========================/gm命令[task]==============================="));
end