--测试技能，立刻恢复生命，徐徐恢复生命

require("define")
require("share\\server_custom_define")

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

	local level = nx_query_property(skill, "Level")
	
	--获得自己的治疗等级
	local healpower = nx_query_property(self, "HealPower") + nx_query_property(self, "HealPowerAdd")

	local healpoint = 0
	local inc_value = 0

--等级1，恢复生命，60+治疗等级*1，每3秒恢复生命=20+治疗等级，持续30秒
--等级2，恢复生命，120+治疗等级*1.5，每3秒恢复生命=40+治疗等级，持续30秒
--等级3，恢复生命，180+治疗等级*2，每3秒恢复生命=60+治疗等级，持续30秒
--等级4，恢复生命，240+治疗等级*2.5，每3秒恢复生命=80+治疗等级，持续30秒
--等级5，恢复生命，320+治疗等级*3，每3秒恢复生命=100+治疗等级，持续30秒
	if level == 1 then
		healpoint = 60 + healpower * 1
		inc_value = 20 + healpower
		
	elseif level == 2 then
		healpoint = 120 + healpower * 1.5
		inc_value = 20 + healpower
	
	elseif level == 3 then
		healpoint = 180 + healpower * 2
		inc_value = 20 + healpower

	elseif level == 4 then
		healpoint = 240 + healpower * 2.5
		inc_value = 20 + healpower

	elseif level == 5 then
		healpoint = 320 + healpower * 3
		inc_value = 20 + healpower
	end

	nx_inc_property(target, "HP", healpoint)
	--通知视野范围内的客户端显示加血特效
	nx_custom_byken(target, SERVER_CUSTOMMSG_EFFECT, target, "ADDHP", healpoint)
	
	--添加BUFFER
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_06")
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	--Buffer的作用类型(0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 3)
	--设置Buffer伤害效果是否显示
	nx_set_property(buffer, "IsDamage", 1)
	--Buffer的生命期
	nx_set_property(buffer, "LifeTime", 30000)
	--Buffer的生命期类型(0:只计算在线时间,1:计算全部时间,包括下线时间)
	nx_set_property(buffer, "LifeTimeType", 1)
	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 3000)
	--Buffer每次触发时是否成功作用的成功率，百分比数
	nx_set_property(buffer, "EffectRatio", 100)
	--BUFFER改变的属性(添加到Buffer的ContinuePropertyRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(inc_value), 0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end
