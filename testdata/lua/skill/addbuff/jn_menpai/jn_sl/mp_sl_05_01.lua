-- 增益buff大幅增加臂力
-- lv1:30分钟内增加臂力10点
-- lv2:35分钟内增加臂力15点
-- lv3:40分钟内增加臂力20点
-- lv4:50分钟内增加臂力25点
-- lv5:60分钟内增加臂力30点

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local strAdd = 0

	if level == 1 then
		strAdd = 10
	elseif level == 2 then
		strAdd = 15
	elseif level == 3 then
		strAdd = 20
	elseif level == 4 then
		strAdd = 25
	elseif level == 5 then
		strAdd = 30
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_05_01")
	if not nx_exists(buffer) then
		return false
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "StrAdd", nx_double(strAdd), 0,0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	return true
end
