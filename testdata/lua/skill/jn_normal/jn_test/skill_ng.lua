--17040502	笛DeBuf	OtherSkill
require("define")

local neigong_buffer_id = "buf_neigong_effect"

--刷新技能之间相互影响的属性
function refresh_property(self, skill)
end

--技能开始准备
function skill_prepare(self, usetype, target, x, z, skill)
end

--技能准备之后
function skill_after_prepare(self, usetype, target, x, z, skill)
	--添加一个全身特效
	nx_set_property(self, "NeiGongEffect", "neigong_di")
	nx_inc_property(self, "CantMove", 1)
	
	nx_add_data(skill, "BeginTime", VTYPE_INT)
	nx_set_data(skill, "BeginTime", nx_get_ticks())
	
	--气场强度清0
	nx_set_property(self, "NeiGongPower", 0)
end

--技能命中
function skill_before_hit(self, skill)
	--消耗MP
	local self_cur_neigong_id = nx_query_property(self, "CurNeiGong")
	local self_cur_neigong = nx_find_neigong(self, self_cur_neigong_id)
	if not nx_exists(self_cur_neigong) then
		return 0
	end
	
	local consume_mp = nx_query_property(self_cur_neigong, "ConsumeMP")
	local cur_mp = nx_query_property(self, "MP")
	if cur_mp < consume_mp then
		--不够消耗
		return 0
	end
	
	nx_set_property(self, "MP", cur_mp - consume_mp)
	
	--增加气场强度
	nx_inc_property(self, "NeiGongPower", 1)
	
	return 1
end

--技能被命中之后
function skill_after_hit(self, target, skill)
	if can_effect(self, target, skill) then
		local row = nx_find_record_value(self, "neigong_effect_obj_rec", 0, target)
		if row < 0 then
			if effect(self, target, skill) then
				nx_add_record_row_values(self, "neigong_effect_obj_rec", target)
			end
		end
	else
		local row = nx_find_record_value(self, "neigong_effect_obj_rec", 0, target)
		if row >= 0 then
			uneffect(self, target)
			nx_remove_record_row(self, "neigong_effect_obj_rec", row)
		end
	end
end

--技能使用结束
function skill_finish(self, skill)
	clear_effect(self, skill)
end

--技能被打断
function skill_break(self, skill)
	clear_effect(self, skill)
end

--清除数据
function clear_effect(self, skill)
	if nx_query_property(self, "NeiGongEffect") ~= "" then
		nx_set_property(self, "NeiGongEffect", "")
		nx_inc_property(self, "CantMove", -1)
	end
	
	--清除被自已影响的对象
	local rows = nx_get_record_rows(self, "neigong_effect_obj_rec")
	for i = 0, rows - 1 do
		local target = nx_query_record_value(self, "neigong_effect_obj_rec", i, 0)
		uneffect(self, target)
	end
	
	nx_clear_record(self, "neigong_effect_obj_rec")
end

--能否影响他人
function can_effect(self, target, skill)
	--激活的内功
	local self_cur_neigong = nx_null()
	local target_cur_neigong = nx_null()
	
	local self_cur_neigong_id = nx_query_property(self, "CurNeiGong")
	self_cur_neigong = nx_find_neigong(self, self_cur_neigong_id)
	
	if nx_find_property(target, "CurNeiGong") then
		local target_cur_neigong_id = nx_query_property(target, "CurNeiGong")
		target_cur_neigong = nx_find_neigong(target, target_cur_neigong_id)
	end

	--内功类型的相生相克
	local self_type = nx_query_property(self_cur_neigong, "EffectType")
	local target_type = 0
	if nx_exists(target_cur_neigong) then
		target_type = nx_query_property(target_cur_neigong, "EffectType")
	end

	if self_type == target_type then
		--内功类型相等，根据内功的级别
		local self_neigong_level = nx_query_property(self_cur_neigong, "Level")
		local target_neigong_level = 0
		
		if nx_exists(target_cur_neigong) then
			target_neigong_level = nx_query_property(target_cur_neigong, "Level")
		end

		if self_neigong_level > target_neigong_level then
			return true
		elseif self_neigong_level < target_neigong_level then
			return false
		end

		--级别相同，根据境界
		local self_neigong_jingjie = nx_query_property(self_cur_neigong, "JingJie")
		local target_neigong_jingjie = 0
		if nx_exists(target_cur_neigong) then
			target_neigong_jingjie = nx_query_property(target_cur_neigong, "JingJie")
		end
		
		if self_neigong_jingjie > target_neigong_jingjie then
			return true
		elseif self_neigong_jingjie < target_neigong_jingjie then
			return false
		end

		return true
	end

	if target_type == 0 or self_type == 4 then
		--对方的内功类型被所有类型相克
		return true
	end

	if self_type == 0 or target_type == 4 then
		--自己的内功类型被所有类型相克
		return false
	end

	if (self_type == 2 and target_type == 3) or
		(self_type == 3 and target_type == 1) or
		(self_type == 1 and target_type == 2) then
		--天克地, 地克人,人克地
		return true
	else
		return false
	end
end

--影响他人
function effect(self, target, skill)
	--添加BUFFER
	local buffer = nx_create_from_config(nx_get_scene(), neigong_buffer_id)
	if not nx_exists(buffer) then
		return false
	end
	
	--设置Buffer的属性
	--Buffer的作用类型(0:瞬时Buffer,1:在X秒内有效,2:在X秒内Y次事件有效,3:在X秒内有效,每Y秒触发一次)
	nx_set_property(buffer, "EffectType", 1)
	--Buffer的生命期
	local begin_time = nx_query_data(skill, "BeginTime")
	local now = nx_get_ticks()
	local skill_period = nx_query_property(skill, "LeadTime")

	local life_time = skill_period - (now - begin_time)
	nx_set_property(buffer, "LifeTime", life_time)
	--Buffer的生命期类型(0:只计算在线时间,1:计算全部时间,包括下线时间)
	nx_set_property(buffer, "LifeTimeType", 1)
	--BUFFER改变的属性(添加到Buffer的DirectPropertyRec表格中，参见buffer.xml文件中此表格的定义)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "CantMove", nx_double(1.0), 0, 0)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "CantUseSkill", nx_double(1.0), 0, 0)
	
	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	
	nx_stop(target)
	
	return true
end

--取消对他人的影响
function uneffect(self, target)
	nx_remove_buffer(target, self, neigong_buffer_id)
end