--jn_lmsj_03	六脉神剑	中冲剑	阴属性伤害	产生BUFF	BUFF时间内使用技能	会损失生命
require("define")
require("command_define")

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
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--计算技能伤害(每一次命中都会调用一次), 返回5种伤害(外功, 刚, 柔, 阴, 阳)
function get_damage(self, skill, target)
	--取自己技能的级别
	local level = nx_query_property(skill, "Level")
	--获得自己的最大内力上限
  	local maxMP = nx_query_property(self, "MaxMP") + nx_query_property(self, "MaxMPAdd")
	--获得自己的内功威力
	local magicPower = nx_query_property(self, "MagicPower") + nx_query_property(self, "MagicPowerAdd")
	--临时变量
	local negDamage = 0
	inc_value = 0
	
--等级1：阴伤害=50+内功威力*1，30％几率施加BUFF，BUFF时间内使用技能，会损失生命（技能耗魔*倍率）
--等级2：阴伤害=100+内功威力*1.5，30％几率施加BUFF，BUFF时间内使用技能，会损失生命（技能耗魔*倍率）
--等级3：阴伤害=150+内功威力*2，30％几率施加BUFF，BUFF时间内使用技能，会损失生命（技能耗魔*倍率）
--等级4：阴伤害=200+内功威力*2.5，30％几率施加BUFF，BUFF时间内使用技能，会损失生命（技能耗魔*倍率）
--等级5：阴伤害=250+内功威力*3，30％几率施加BUFF，BUFF时间内使用技能，会损失生命（技能耗魔*倍率）
	
	if level == 1 then
		negDamage = math.random(25,75) + magicPower * 1
		buf_level = 1

	elseif level == 2 then
		negDamage = math.random(150,250) + magicPower * 1.5
		buf_level = 2

	elseif level == 3 then
		negDamage = math.random(200,400) + magicPower * 2
		buf_level = 3

	elseif level == 4 then
		negDamage = math.random(300,500) + magicPower * 2.5
		buf_level = 4

	elseif level == 5 then
		negDamage = math.random(400,600) + magicPower * 3
		buf_level = 5
	end

	if math.random(0,100) <= 100 then
		add_buffer(self, target, buf_level)
	end
	
	--输出阴属性伤害
	return 0, 0, 0, negDamage, 0
end
----------------------------------------------------------------------------------------------------

--定义Buffer函数，获取参数
function add_buffer(self, target, buf_level)

	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_jn_lmsj_03")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性	
	nx_set_property(buffer, "Level", buf_level)
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 2)
		
	--Buffer的生命期（时间按照技能等级获取）
	nx_set_property(buffer, "LifeTime",30000)
	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 5)
	--Buffer每次触发时是否成功作用的成功率，百分比数
	nx_set_property(buffer, "EffectRatio", 100)
	--BUFFER的事件触发(添加到Buffer的EventEffectRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "EventEffectRec", COMMAND_SKILL_BEGIN)	
	
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true
end