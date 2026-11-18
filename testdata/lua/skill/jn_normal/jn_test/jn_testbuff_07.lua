--增加自己的格挡

require("define")

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
end

--技能命中之前
function skill_before_hit(self, skill)
	return 1
end

--技能命中对象
function skill_after_hit(self, skill, target)
	--等级1：增加3，持续10秒
	--等级1：增加5，持续10秒
	--等级1：增加7，持续10秒
	--等级1：增加9，持续10秒
	--等级1：增加10，持续10秒

	--取自己技能的级别
	local level = nx_query_property(skill, "Level")
	local inc_value = 0
	if level == 1 then
		inc_value = 3

	elseif level == 2 then
		inc_value = 5

	elseif level == 3 then
		inc_value = 7

	elseif level == 4 then
		inc_value = 9

	elseif level == 5 then
		inc_value = 10
	end

	--添加BUFF
	add_buffer(self, target, inc_value)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

function add_buffer(self, target, inc_value)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_07")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 1)
	
	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",300000)
	
	--Buffer的生命期类型
	--0:只计算在线时间,1:计算全部时间,包括下线时间
	nx_set_property(buffer, "LifeTimeType", 0)

	--BUFFER持续改变的属性，添加到Buffer的RolePropRec表格中，参见buffer.xml文件中此表格的定义
	nx_add_record_row_values(buffer, "RolePropRec", "DamageReduceAdd", nx_double(inc_value))

	
	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end
