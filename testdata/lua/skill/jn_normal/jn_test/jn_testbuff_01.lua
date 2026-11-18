--测试	投掷暗器	下毒

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
	
	--获得自己当前暗器最大攻击
	local maxDamage = nx_query_property(self, "MaxShotDamage")
	--获得自己当前暗器最小攻击
	local minDamage = nx_query_property(self, "MinShotDamage")
	--当前武器攻击（随机取值）
	local curDamage = math.random(minDamage,maxDamage)

	--临时变量：伤害初始化
	local phyDamage = 0
	
	--临时变量：BUFF效果数值初始化	
	inc_value = 0

--等级1：外功伤害=暗器攻击+20，30秒内每5秒损失15点生命
--等级2：外功伤害=暗器攻击+40，30秒内每5秒损失25点生命
--等级3：外功伤害=暗器攻击+60，30秒内每5秒损失35点生命
--等级4：外功伤害=暗器攻击+80，30秒内每5秒损失50点生命
--等级5：外功伤害=暗器攻击+100，30秒内每5秒损失70点生命
	if level == 1 then
		phyDamage = (math.random(10,30) + curDamage)
		inc_value = -15

	elseif level == 2 then
		phyDamage = (math.random(30,50) + curDamage)
		inc_value = -25

	elseif level == 3 then
		phyDamage = (math.random(50,70) + curDamage)
		inc_value = -35

	elseif level == 4 then
		phyDamage = (math.random(70,90) + curDamage)
		inc_value = -50

	elseif level == 5 then
		phyDamage = (math.random(90,100) + curDamage)
		inc_value = -70
	end

	--添加BUFF	
	add_buffer(self, target, inc_value)
	
	--输出外功伤害
	return phyDamage, 0, 0, 0, 0
end
----------------------------------------------------------------------------------------------------
function add_buffer(self, target, inc_value)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_01")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 3)
	
	--设置Buffer伤害效果是否显示，
	--1:显示,0:不显示
	nx_set_property(buffer, "IsDamage", 1)
	
	--Buffer的生命期
	nx_set_property(buffer, "LifeTime",30000)

	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 5000)

	--Buffer每次触发时是否成功作用的成功率，百分比数
	nx_set_property(buffer, "EffectRatio", 100)

	--Buffer的生命期类型，0:只计算在线时间,1:计算全部时间,包括下线时间
	nx_set_property(buffer, "LifeTimeType", 0)

	--BUFFER连续改变的属性（连续扣血）
	--添加到Buffer的ContinuePropertyRec表格中，参见buffer.xml文件中此表格的定义
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(inc_value), 0)
	
	--添加到目标身上,同时记录结果
	local result = nx_add_buffer(target, self, buffer)
	
	--如果添加失败,则删除创建的Buffer对象
	if not result then
		nx_destroy_self(buffer)
		return false
	end
end

