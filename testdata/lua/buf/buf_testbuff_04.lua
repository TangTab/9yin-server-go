	--等级1：触发限制BUFF，15秒内出虚招，命中降低60点，持续30秒
	--等级2：触发限制BUFF，15秒内出虚招，命中降低70点，持续30秒
	--等级3：触发限制BUFF，15秒内出虚招，命中降低80点，持续30秒
	--等级4：触发限制BUFF，15秒内出虚招，命中降低90点，持续30秒
	--等级5：触发限制BUFF，15秒内出虚招，命中降低100点，持续30秒
	
require("define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer)
    --判断获取角色当前使用技能的行为
	local cur_skill = nx_query_property(self, "CurSkill")
	
	--如果没有使用技能，则返回并停止
	if not nx_exists(cur_skill) then
		return 0
	end
	
	--如果没有使用虚招，则返回并停止
	if nx_query_property(cur_skill, "EffectType") ~= 2 then
		return 0
	end

	local buffer_level = nx_query_property(buffer, "Level")
	local sender = nx_query_property(buffer, "Sender")
		
	--使用了虚招，激活了另一个BUFF	
	add_buffer(sender, self, buffer_level)
		return 1	
end

--状态被打断
function buffer_break(self, buffer)
end

function add_buffer(self, target, level)
	--创建一个Buffer对象,BufferID必须存在于Buffer表中
	local buffer = nx_create_from_config(nx_get_scene(), "buf_testbuff_04a")

	--如果创建失败则返回
	if not nx_exists(buffer) then
		return false
	end
	
	--Buffer的作用类型
	--0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 1)

	--Buffer的生命期（时间按照技能等级获取）
	nx_set_property(buffer, "LifeTime",30000)
	--BUFFER改变的属性(添加到Buffer的RolePropRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "RolePropRec", "PhyHitProbAdd", nx_double(-50 - level * 10))
		
	--如果添加失败,则删除创建的Buffer对象
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	return true
end