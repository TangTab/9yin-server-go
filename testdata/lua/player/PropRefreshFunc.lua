require("share\\server_custom_define")


--获取物理攻击强度
function nx_get_intphyattack(self, index)
	local str = nx_query_property_record(self, "Str", index)
	local sta = nx_query_property_record(self, "Sta", index)	
	local value = str * 1.2 + sta * 0.3
	return value
end

--获取最大物理攻击
function nx_get_phymaxattack(self, index)
	--local level = nx_query_property(self, "Level")
	--local dodge_level = nx_query_property(self, "DodgeLevel")
	local intensityphyattack = nx_query_property_record(self,"IntensityPhyAttack", index)
	local value = intensityphyattack * 1.2
	return value
end

--获取最小物理攻击
function nx_get_phyminattack(self, index)
	--local level = nx_query_property(self, "Level")
	--local dodge_level = nx_query_property(self, "DodgeLevel")
	local intensityphyattack = nx_query_property_record(self,"IntensityPhyAttack", index)
	local value = intensityphyattack * 0.8
	return value
end

--获取暴击等级
function nx_get_critlevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.2 + esth * 0.03125
	return value
end

--获取暴击率
function nx_get_critprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local critlevel = nx_query_property_record(self, "CritLevel", index)
	local value = critlevel * 0.0007
	return value
end

--获取护甲穿透率
function nx_get_penetrateprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local penetratelevel = nx_query_property_record(self, "PenetrateLevel", index)
	local value = penetratelevel * 0.00016
	return value
end

--获取精准等级
function nx_get_accuratelevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.25 + esth * 0.03125
	return value
end

--获取精准率
function nx_get_accurateprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local accurateLevel = nx_query_property_record(self, "AccurateLevel", index)
	local value = accurateLevel * 0.0008
	return value
end

--获取暴击倍率
function nx_get_critmulprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local critMulLevel = nx_query_property_record(self, "CritMulLevel", index)
	local value = critMulLevel*0.0005
	return value
end

--获取法术攻击强度
function nx_get_intmagattack(self, index)
	local ing = nx_query_property_record(self, "Ing", index)
	local spi = nx_query_property_record(self, "Spi", index)
	local value = ing * 1.2 + spi * 0.3
	return value
end

--获取最大法术攻击
function nx_get_magicmaxattack(self, index)
	local intensitymagicattack = nx_query_property_record(self,"IntensityMagicAttack", index)
	local value = intensitymagicattack * 1.2
	return value
end

--获取最小法术攻击
function nx_get_magicminattack(self, index)
	local intensitymagicattack = nx_query_property_record(self,"IntensityMagicAttack", index)
	local value = intensitymagicattack * 0.8
	return value
end

--获取法术暴击等级
function nx_get_magcritlevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.03125 + esth * 0.2
	return value
end

--获取法术暴击率
function nx_get_magcritprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local magCritLevel = nx_query_property_record(self, "MagCritLevel", index)
	local value = magCritLevel*0.0007
	return value
end

--获取法术精准等级
function nx_get_magpreclevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.03125 + esth * 0.25
	return value
end

--获取法术精准率
function nx_get_magprecprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local magpreclevel = nx_query_property_record(self, "MagPrecLevel", index)
	local value = magpreclevel*0.0008	
	return value
end

--获取护甲减伤率
function nx_get_guardrateprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local guard = nx_query_property_record(self, "Guard", index)
	local value = guard * 0.00016
	return value
end

--获取闪避等级
function nx_get_dodgelevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.25 + esth * 0.03125
	return value
end

--获取闪避率
function nx_get_dodgeprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local dodgeLevel = nx_query_property_record(self, "DodgeLevel", index)
	local value = dodgeLevel * 0.0006
	return value
end

--获取法术闪避等级
function nx_get_magdodgelevel(self, index)
	local dex = nx_query_property_record(self, "Dex", index)
	local esth = nx_query_property_record(self, "Esth", index)
	local value = dex * 0.03125 + esth * 0.25
	return value
end

--获取法术闪避率
function nx_get_magdodgeprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local magdodgeLevel = nx_query_property_record(self, "MagdodgeLevel", index)
	local value = magdodgeLevel * 0.0006
	return value
end

--获取防御几率
function nx_get_defenseprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local defenseLevel = nx_query_property_record(self, "DefenseLevel", index)
	local value = defenseLevel * 0.00042
	return value
end

--获取全法术抵抗
function nx_get_allresist(self, index)
	local ing = nx_query_property_record(self, "Ing", index)
	local spi = nx_query_property_record(self, "Spi", index)
	local value = ing * 0.01 + spi * 10.5
	return value
end

--获取法术减伤率
function nx_get_allresistprob(self, index)
	local level = nx_query_property(self, "Level")
	local max_level = nx_query_max_level(self)
	local allresist = nx_query_property_record(self, "AllResist", index)
	local value = allresist * 0.00005
	return value
end

--获取HP回复速度(注销部分为按比例恢复)
function nx_get_hp_upspeed(self, index)
   --local max_hp = nx_query_property(self, "MaxHP") + nx_query_property(self, "MaxHPAdd")
   --local value = max_hp * 0.1
   local str = nx_query_property_record(self, "Str", index)
   local sta = nx_query_property_record(self, "Sta", index)	
   local value = str * 0.02 + sta * 0.09 + 4
   return value
end

--获取MP回复速度(注销部分为按比例恢复)
function nx_get_mp_upspeed(self, index)
   --local max_mp = nx_query_property(self, "MaxMP") + nx_query_property(self, "MaxMPAdd")
   --local value = max_mp * 0.1
   local spi = nx_query_property_record(self, "Spi", index)
   local value = spi * 0.25 + 4
   return value
end

--获取EY回复速度(注销部分为按比例恢复)
function nx_get_ey_upspeed(self, index)
   --local max_ey = nx_query_property(self, "MaxEY") + nx_query_property(self, "MaxEYAdd")
   --local value = max_ey * 0.1
   local spi = nx_query_property_record(self, "Spi", index)
   local value = spi * 0.05 + 4
   return value
end

--获取AV回复速度(注销部分为按比例恢复)
function nx_get_av_upspeed(self, index)
   --local max_av = nx_query_property(self, "MaxAV") + nx_query_property(self, "MaxAVAdd")
   --local value = max_av * 0.1
   --local spi = nx_query_record_property(self, "Spi")
   local value = 10
   return value
end

--获取RC递减速度
function nx_get_rc_upspeed(self, index)
	local value = -2
	return value
end

--获取DP回复速度
function nx_get_dp_upspeed(self, index)
	local value = 5
	return value
end

--获取元素加深率
function nx_get_elementDeepenProb(self, index)
   local elementDeepenLevel = nx_query_property_record(self, "ElementDeepenLevel", index)
   local value = elementDeepenLevel * 0.005
   return value
end

--获取元素减伤率
function nx_get_elementResistProb(self, index)
   local elementResistLevel = nx_query_property_record(self, "ElementResistLevel", index)
   local value = elementResistLevel * 0.005
   return value
end

--获取最大HP
function nx_get_max_hp(self, index)
   local str = nx_query_property_record(self, "Str", index)
   local sta = nx_query_property_record(self, "Sta", index)	
   local value = str * 4.5 + sta * 12.8
   return value
end

--获取最大MP
function nx_get_max_mp(self, index)
   local ing = nx_query_property_record(self, "Ing", index)	
   local value = ing * 22.5
   return value
end

--获取最大EY
function nx_get_max_ey(self, index)
   local str = nx_query_property_record(self, "Str", index)	
   local value = str * 15
   return value
end