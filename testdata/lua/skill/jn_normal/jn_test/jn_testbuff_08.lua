require("define")

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, skill, usetype, target, x, y, z)
end

--技能准备之后
function skill_after_prepare(self, skill, usetype, target, x, y, z)
	--添加BUFFER
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_jingchundu")
	if not nx_exists(buffer) then
		return false
	end
	
	--技能的级别
	local level = nx_query_property(skill, "Level")
	
	--设置Buffer的属性
	--Buffer的作用类型(0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 1)
	--设置Buffer伤害效果是否显示
	nx_set_property(buffer, "IsDamage", 1)
	--Buffer的生命期
	nx_set_property(buffer, "LifeTime", 60 * 1000)
	--Buffer的生命期类型(0:只计算在线时间,1:计算全部时间,包括下线时间)
	nx_set_property(buffer, "LifeTimeType", 1)
	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 0)
	--Buffer每次触发时是否成功作用的成功率，百分比数
	nx_set_property(buffer, "EffectRatio", 100)
	
	--BUFFER改变的属性(添加到Buffer的DirectPropertyRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 2, "", "CurPurity", nx_double(-20.0 - level * 2), 1, 0)
	
	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
end

--技能命中之前
function skill_before_hit(self, skill)
	return 1
end

--技能命中对象
function skill_after_hit(self, skill, target)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
