require ("define")
--技能使用时的限制函数定义，返回0表示此技能不能使用，返回1表示可以使用

--target的hp未满时，可以使用，已满，则不能使用
function hp_not_full(self, skill, target, x, y, z)
	--hp未满
	return 1;
end

--HP未低于25%
function hp_less_25percent(self, skill, target, x, y, z)
	if not nx_exists(target) then
		return 0;
	end
	if not nx_find_property(target, "HP") then
		return 0;
	end
	local hp = nx_query_property(target, "HP");
	local maxhp = nx_query_property(target, "MaxHP") + nx_query_property(target, "MaxHPAdd");
	if (hp / maxhp) > 0.25 then
		--hp大于25%
		return 0;
	else
		--小于等于25%
		return 1;
	end
end

--target的mp未满时，可以使用，已满，则不能使用
function mp_not_full(self, skill, target, x, y, z)
	if not nx_exists(target) then
		return 0;
	end
	if not nx_find_property(target, "MP") then
		return 0;
	end
	local mp = nx_query_property(target, "MP");
	local maxmp = nx_query_property(target, "MaxMP") + nx_query_property(target, "MaxMPAdd");
	if mp >= maxmp then
		--mp已满
		if nx_type(self) == TYPE_PLAYER then
			--MP已满，无需补充
			nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9201");
		end
		return 0;
	else
		--mp未满
		return 1;
	end
end

--target未死不能使用
function targe_is_dead(self, skill, target, x, y, z)
	if not nx_exists(target) then
		return 0;
	end

	if nx_query_property(target, "Dead") == 0 then
		--目标必须在死亡中
		if nx_type(self) == TYPE_PLAYER then
			--MP已满，无需补充
			nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9202");
		end
		return 0;
	else
		return 1;
	end
	
end

--角色当前手部武器是宝剑时，才能使用这个技能，武器属性EquipCategory=101
function sword_skill_use(self, skill, target, x, y, z)
	--角色的装备栏
	local equipbox = nx_null();
	local weapon = nx_null();
	if nx_type(self) == TYPE_PLAYER then --玩家
		equipbox = nx_get_child(self, nx_widestr("EquipBox"));
		if nx_exists(equipbox) then
			weapon = nx_get_item(equipbox, 8);
		end;
	end
	
	if not nx_exists(weapon) then
		--未装备武器
		nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9050");
		return 0;
	end
	
	local equip_category = nx_query_property(weapon, "EquipCategory");
	if equip_category ~= 101 then
		--装备的不是一把剑
		nx_sysinfo(self, TIPSTYPE_TIPSINFO, "9051");
		return 0;
	end
	
	return 1;
end
