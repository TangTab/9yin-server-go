--气功	小狮子吼	伤害周围敌人
--等级1：阳伤害=100+内功威力，降低他们的力量敏捷
--等级2：阳伤害=150+内功威力，降低他们的力量敏捷
--等级3：阳伤害=200+内功威力，降低他们的力量敏捷
--等级4：阳伤害=250+内功威力，降低他们的力量敏捷
--等级5：阳伤害=300+内功威力，降低他们的力量敏捷

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
	
	--获取自己的意志（SPI）
	local spi = nx_query_property(self, "Spi") + nx_query_property(self, "SpiAdd")
	
	--获得自己的内功威力
	local magicPower = nx_query_property(self, "MagicPower") + nx_query_property(self, "MagicPowerAdd")

	--临时变量：伤害初始化
	local masDamage = 0
	
	--临时变量：BUFF效果数值初始化	
	inc_value = 0

--等级1：阳伤害=100+内功威力
--等级2：阳伤害=150+内功威力
--等级3：阳伤害=200+内功威力
--等级4：阳伤害=250+内功威力
--等级5：阳伤害=300+内功威力
	if level == 1 then
		masDamage = (math.random(50,150) + magicPower)
		inc_value = spi * 0.5

	elseif level == 2 then
		masDamage = (math.random(100,200) + magicPower)
		inc_value = spi * 0.8

	elseif level == 3 then
		masDamage = (math.random(150,250) + magicPower)
		inc_value = spi * 1

	elseif level == 4 then
		masDamage = (math.random(200,300) + magicPower)
		inc_value = spi * 1.3

	elseif level == 5 then
		masDamage = (math.random(250,350) + magicPower)
		inc_value = spi * 1.5
	end

	--添加BUFF	
	add_buffer(self, target, inc_value)
	
	--输出外功伤害
	return 0, 0, 0, 0, masDamage
end

----------------------------------------------------------------------------------------------------
function add_buffer(self, target, inc_value)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testaoe_01")

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
	nx_set_property(buffer, "LifeTime",10000)
	
	--Buffer的生命期类型
	--0:只计算在线时间,1:计算全部时间,包括下线时间
	nx_set_property(buffer, "LifeTimeType", 0)

	--BUFFER持续改变的属性（降低命中）
	--添加到Buffer的RolePropRec表格中，参见buffer.xml文件中此表格的定义
	nx_add_record_row_values(buffer, "RolePropRec", "StrAdd", nx_double(inc_value))
	nx_add_record_row_values(buffer, "RolePropRec", "DexAdd", nx_double(inc_value))
	
	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end






