--测试	点穴	限制实招

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
	--取自己技能的级别
	local level = nx_query_property(skill, "Level")
	
	--等级1：触发限制BUFF，15秒内出实招，损失力量降低50点，持续30秒
	--等级2：触发限制BUFF，15秒内出实招，损失力量降低100点，持续30秒
	--等级3：触发限制BUFF，15秒内出实招，损失力量降低150点，持续30秒
	--等级4：触发限制BUFF，15秒内出实招，损失力量降低200点，持续30秒
	--等级5：触发限制BUFF，15秒内出实招，损失力量降低250点，持续30秒

	--添加BUFF	
	add_buffer(self, target, level)
end

--技能使用结束
function skill_finish(self, skill)
end

--技能被打断
function skill_break(self, skill)
end

--定义Buffer函数，获取参数
function add_buffer(self, target, level)

	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_03")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 3)

	--Buffer的生命期（时间按照技能等级获取）
	nx_set_property(buffer, "LifeTime",600000)
	--Buffer计时的间隔时间或计次的次数
	nx_set_property(buffer, "EffectTimer", 5000)
	--Buffer每次触发时是否成功作用的成功率，百分比数
	nx_set_property(buffer, "EffectRatio", 100)

	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true
end