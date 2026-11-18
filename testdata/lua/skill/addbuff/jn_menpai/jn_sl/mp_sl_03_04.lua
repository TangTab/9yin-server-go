-- 增益buff
-- lv1:提高格挡值10点，持续1分钟
-- lv2:提高格挡值15点，持续2分钟
-- lv3:提高格挡值20点，持续3分钟
-- lv4:提高格挡值25点，持续4分钟
-- lv5:提高格挡值30点，持续5分钟

-- 施放者对象, 被释放者对象, 技能等级
function AddBuff(self, target, level)
	--创建Buffer
	local buffer = nx_create_from_config(nx_get_scene(), "buf_mp_sl_03_04")
	if not nx_exists(buffer) then
		return false
	end

	local parryAdd = 0
	local lifeTime = 0

	if level == 1 then
		parryAdd = 10
		lifeTime = 60000
	elseif level == 2 then
		parryAdd = 15
		lifeTime = 120000
	elseif level == 3 then
		parryAdd = 20
		lifeTime = 180000
	elseif level == 4 then
		parryAdd = 25
		lifeTime = 240000
	elseif level == 5 then
		parryAdd = 30
		lifeTime = 300000
	end

	--设置Buffer的属性
	nx_set_property(buffer, "Level", level)
	nx_set_property(buffer, "LifeTime", lifeTime)
	nx_add_record_row_values(buffer, "DirectPropertyRec", 0, 0, "", "PhyParryAdd", nx_double(parryAdd), 0,0)

	--添加到对方
	if not nx_add_buffer(target, self, buffer) then
		nx_destroy_self(buffer)
		return false
	end

	return true
end
