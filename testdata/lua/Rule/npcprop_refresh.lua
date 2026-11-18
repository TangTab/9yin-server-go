---------------------------------------------------
-- @ 文件名称: npc_prop_refresh.lua
-- @ 文件功能: 计算NPC属性刷新
-- @ 作者    : dReAmSun (孙明杰)
-- @ 日期    : 2010-3-10
---------------------------------------------------

-- 属性枚举
local Npc_HPBase = 2
local Npc_MPBase = 3
local Npc_PhyDefBase = 9
local Npc_PIMagicDefBase = 10
local Npc_AIMagicDefBase = 11
local Npc_PIPhyDefBase = 12
local Npc_AIPhyDefBase = 13
local Npc_NegDefBase = 14
local Npc_JujDefBase = 15
local Npc_MasDefBase = 16
local Npc_StiDefBase = 17

-- 根据一级属性计算二级属性
function compute_prop_value(self)
	
	-- 保护
	if not nx_exists(self) then
		return
	end
	
	-- 等级
	local level = nx_query_property(self, "Level")

	-- 得到一级属性值
	local str = nx_query_property(self, "Str")
	local dex = nx_query_property(self, "Dex")
	local ing = nx_query_property(self, "Ing")
	local spi = nx_query_property(self, "Spi")
	local sta = nx_query_property(self, "Sta")
	
	-- 得到附加属性数值
	local strAdd = nx_query_property(self, "StrAdd")
	local dexAdd = nx_query_property(self, "DexAdd")
	local ingAdd = nx_query_property(self, "IngAdd")
	local spiAdd = nx_query_property(self, "SpiAdd")
	local staAdd = nx_query_property(self, "StaAdd")
	
	-- 得到防御系数
	local phydefbase = nx_query_npc_float(self, Npc_PhyDefBase)
	local pimagicdefbase = nx_query_npc_float(self, Npc_PIMagicDefBase)
	local aimagicdefbase = nx_query_npc_float(self, Npc_AIMagicDefBase)
	local piphydefbase = nx_query_npc_float(self, Npc_PIPhyDefBase)
	local aiphydefbase = nx_query_npc_float(self, Npc_AIPhyDefBase)
	local negdefbase = nx_query_npc_float(self, Npc_NegDefBase)
	local jujdefbase = nx_query_npc_float(self, Npc_JujDefBase)
	local masdefbase = nx_query_npc_float(self, Npc_MasDefBase)
	local stidefbase = nx_query_npc_float(self, Npc_StiDefBase)
		
	-- 根据公式计算二级属性
	-- MaxHP
	local maxhp = 2 * (str + strAdd) + 10 * (sta + staAdd)
	maxhp = maxhp * (1 + level * 0.02) + 10 * sta * nx_number(nx_query_npc_float(self, Npc_HPBase))
	nx_set_property(self, "MaxHP", nx_int(maxhp))
	
	-- MaxMP
	local maxmp = 3 * (ing + ingAdd) + 2 * (spi + spiAdd) + nx_number(nx_query_npc_float(self, Npc_MPBase))
	nx_set_property(self, "MaxMP", nx_int(maxmp))
	
	-- Def
	nx_set_property(self, "PhyDef", level * nx_number(phydefbase))
	nx_set_property(self, "PIMagicDef", nx_float(level * nx_number(pimagicdefbase)))
	nx_set_property(self, "AIMagicDef", level * nx_number(aimagicdefbase))
	nx_set_property(self, "PIPhyDef", nx_float(level * nx_number(piphydefbase)))
	nx_set_property(self, "AIPhyDef", level * nx_number(aiphydefbase))
	nx_set_property(self, "NegDef", level * nx_number(negdefbase))
	nx_set_property(self, "JujDef", level * nx_number(jujdefbase))
	nx_set_property(self, "MasDef", level * nx_number(masdefbase))
	nx_set_property(self, "StiDef", level * nx_number(stidefbase))

	-- MeleePower
	local meleepower = 1 * (str+strAdd)
	nx_set_property(self, "MeleePower", meleepower)
	
	-- ShotPower
	local shotpower = 1 * (dex + dexAdd)
	nx_set_property(self, "ShotPower", shotpower)
	
	-- PhyVa
	local phyva = 0.3 * (dex + dexAdd)
	nx_set_property(self, "PhyVa", phyva)
	
	-- MagicPower
	local magicpower = 1 * (ing + ingAdd)
	nx_set_property(self, "MagicPower", magicpower)
	
	-- MagicVa
	local magicva = 0.3 * (spi + spiAdd)
	nx_set_property(self, "MagicVa", magicva)
	
	-- MinMagicDef
	local minmagicdef = 0.2 * (spi + spiAdd)
	nx_set_property(self, "MinMagicDef", minmagicdef)
	
	-- PhyHit
	local phyhit = 0.3 * (dex + dexAdd)
	nx_set_property(self, "PhyHit", phyhit)
	
	-- MagicHit
	local magichit = 0.3 * (spi + spiAdd)
	nx_set_property(self, "MagicHit", magichit)
	
	-- Dodge
	local dodge = 0.3 + (dex + dexAdd)
	nx_set_property(self, "Dodge", dodge)
	
	-- PhyParry
	local phyparry = 0.3 + (dex + dexAdd)
	nx_set_property(self, "PhyParry", phyparry)
	
	-- MagicParry
	local magicparry = 0.3 * (ing + ingAdd) + 0.3 * (spi + spiAdd)
	nx_set_property(self, "MagicParry", magicparry)

	-- VaDef
	local vadef = 0.2 * (sta + staAdd)
	nx_set_property(self, "VaDef", vadef)

	-- HealPower
	local healpower = 1 * (spi + spiAdd)
	nx_set_property(self, "HealPower", healpower)
	
	-- MinMeleeDamage
	local minMeleedamage = 0.75 * (meleepower + nx_query_property(self, "MeleePowerAdd")) * (1 + level * 0.02)
	nx_set_property(self, "MinMeleeDamage", minMeleedamage)
	
	-- MaxMeleeDamage
	local maxMeleedamage = 1.25 * (meleepower + nx_query_property(self, "MeleePowerAdd")) * (1 + level * 0.02)
	nx_set_property(self, "MaxMeleeDamage", maxMeleedamage)
	
	-- MinShotDamage
	local minshotdamage = 0.75 * (shotpower + nx_query_property(self, "ShotPowerAdd")) * (1 + level * 0.02)
	nx_set_property(self, "MinShotDamage", minshotdamage)
	
	-- MaxShotDamage
	local maxshotdamage = 1.25 * (shotpower + nx_query_property(self, "ShotPowerAdd")) * (1 + level * 0.02)
	nx_set_property(self, "MaxShotDamage", maxshotdamage)
	
end