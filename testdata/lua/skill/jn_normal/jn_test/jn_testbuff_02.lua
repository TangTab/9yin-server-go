--隔空点穴	降低移动速度

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
	--等级1：降低1，持续5秒
	--等级2：降低1.1，持续5秒
	--等级3：降低1.2，持续5秒
	--等级4：降低1.35，持续5秒
	--等级5：降低1.5，持续5秒

	--取自己技能的级别
	local level = nx_query_property(skill, "Level")
	local inc_value = 0
	if level == 1 then
		inc_value = -1.2

	elseif level == 2 then
		inc_value = -1.4

	elseif level == 3 then
		inc_value = -1.6

	elseif level == 4 then
		inc_value = -1.8

	elseif level == 5 then
		inc_value = -2
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
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_02")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 1)
	
	--设置Buffer伤害效果是否显示，
	--1:显示,0:不显示
	nx_set_property(buffer, "IsDamage", 1)
	
	--Buffer的生命期
	nx_set_property(buffer, "LifeTime", 8000)
	
	--Buffer的生命期类型
	--0:只计算在线时间,1:计算全部时间,包括下线时间
	nx_set_property(buffer, "LifeTimeType", 0)

	--BUFFER持续改变的属性（降低速度）
	--添加到Buffer的DirectRecordRec表格中，参见buffer.xml文件中此表格的定义
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "RunSpeed", nx_double(inc_value), 0, 0)
	
	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end
