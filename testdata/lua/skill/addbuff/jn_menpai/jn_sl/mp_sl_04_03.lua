-- 增益buff
-- lv1:30秒内每3秒回复20点
-- lv2:30秒内每3秒回复30点
-- lv3:30秒内每3秒回复40点
-- lv4:30秒内每3秒回复50点
-- lv5:30秒内每3秒回复60点

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	local healpoint = 0

	if level == 1 then
		healpoint = 20
	elseif level == 2 then
		healpoint = 30
	elseif level == 3 then
		healpoint = 40
	elseif level == 4 then
		healpoint = 50
	elseif level == 5 then
		healpoint = 60
	end

	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_04_03")
	if not nx_exists(buffer) then
		return false
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_add_record_row_values(buffer, "ContinuePropertyRec", 0, 0, "", "HP", nx_double(healpoint), 0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end
	return true
end
