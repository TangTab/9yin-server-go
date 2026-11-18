

function nx_get_CritProb(self, target)
	--基本暴击率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "CritProb", 0) )
	--装备与buff直接提供的暴击率
	local add_prob = nx_number( nx_query_prob_by_record(self, "CritProb", 1) )
	
	local target_level = nx_query_property(target, "Level")

	local value = 0.0
	if target_level < nx_int(30) then 
		value = math.max(base_prob * (4-0.09*target_level) + add_prob + 0.03,0.03)
	elseif target_level < nx_int(40) then
		value = math.max(base_prob * (1-0.011*target_level) + add_prob + 0.02,0.02)
	elseif target_level < nx_int(50) then
		value = math.max(base_prob * (0.72-0.0071*target_level) + add_prob + 0.01,0.01)
	elseif target_level < nx_int(60) then
		value = math.max(base_prob * (0.67-0.0069*target_level) + add_prob + 0.01,0.01)
	elseif target_level < nx_int(70) then
		value = math.max(base_prob * (0.64-0.0068*target_level) + add_prob + 0.01,0.01)
	end
	
	return value
end

function nx_get_MagCritProb(self, target)
	--基本法术暴击率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "MagCritProb", 0) )
	--装备与buff直接提供的法术暴击率
	local add_prob = nx_number( nx_query_prob_by_record(self, "MagCritProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = math.max(base_prob * (4-0.09*target_level) + add_prob + 0.03,0.03)
	elseif target_level < nx_int(40) then
		value = math.max(base_prob * (1-0.011*target_level) + add_prob + 0.02,0.02)
	elseif target_level < nx_int(50) then
		value = math.max(base_prob * (0.72-0.0071*target_level) + add_prob + 0.01,0.01)
	elseif target_level < nx_int(60) then
		value = math.max(base_prob * (0.67-0.0069*target_level) + add_prob + 0.01,0.01)
	elseif target_level < nx_int(70) then
		value = math.max(base_prob * (0.64-0.0068*target_level) + add_prob + 0.01,0.01)
	end
	
	return value
end
function nx_get_CritMulProb(self, target)
	--基本暴击倍率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "CritMulProb", 0) )
	--装备与buff直接提供的暴击倍率
	local add_prob = nx_number( nx_query_prob_by_record(self, "CritMulProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (4-0.09*target_level) + add_prob + 1.3
	elseif target_level < nx_int(40) then
		value = base_prob * (1-0.011*target_level) + add_prob + 1.25
	elseif target_level < nx_int(50) then
		value = base_prob * (0.72-0.0071*target_level) + add_prob + 1.2
	elseif target_level < nx_int(60) then
		value = base_prob * (0.67-0.0069*target_level) + add_prob + 1.15
	elseif target_level < nx_int(70) then
		value = base_prob * (0.64-0.0068*target_level) + add_prob + 1.15
	end
	
	return value
end
function nx_get_DefenseProb(self, target)
	--基本暴击防御率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "DefenseProb", 0) )
	--装备与buff直接提供的暴击防御率
	local add_prob = nx_number( nx_query_prob_by_record(self, "DefenseProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (3.6-0.081*target_level) + add_prob
	elseif target_level < nx_int(40) then
		value = base_prob * (0.9-0.0099*target_level) + add_prob
	elseif target_level < nx_int(50) then
		value = base_prob * (0.648-0.00639*target_level) + add_prob
	elseif target_level < nx_int(60) then
		value = base_prob * (0.603-0.00621*target_level) + add_prob
	elseif target_level < nx_int(70) then
		value = base_prob * (0.576-0.00612*target_level) + add_prob
	end
	
	return value
end
function nx_get_PenetrateProb(self, target)
	--基本护甲穿透率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "PenetrateProb", 0) )
	--装备与buff直接提供的护甲穿透率
	local add_prob = nx_number( nx_query_prob_by_record(self, "PenetrateProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (6.5-0.0743*target_level) + add_prob
	elseif target_level < nx_int(40) then
		value = base_prob * (1.82-0.0039*target_level) + add_prob
	elseif target_level < nx_int(50) then
		value = base_prob * (1.225-0.0025*target_level) + add_prob
	elseif target_level < nx_int(60) then
		value = base_prob * (1.0218-0.004*target_level) + add_prob
	elseif target_level < nx_int(70) then
		value = base_prob * (0.8775-0.00135*target_level) + add_prob
	end
	
	return value
end
function nx_get_GuardRateProb(self, target)
	--基本护甲减伤率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "GuardRateProb", 0) )
	--装备与buff直接提供的护甲减伤率
	local add_prob = nx_number( nx_query_prob_by_record(self, "GuardRateProb", 1) )
	
	local target_level = nx_query_property(target, "Level")

	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (6.5-0.00743*target_level) + add_prob
	elseif target_level < nx_int(40) then
		value = base_prob * (1.82-0.0039*target_level) + add_prob
	elseif target_level < nx_int(50) then
		value = base_prob * (1.225-0.0025*target_level) + add_prob
	elseif target_level < nx_int(60) then
		value = base_prob * (1.0218-0.004*target_level) + add_prob
	elseif target_level < nx_int(70) then
		value = base_prob * (0.8775-0.00135*target_level) + add_prob
	end
    
	return value
end
function nx_get_AllResistProb(self, target)
	--全抗减伤率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "AllResistProb", 0) )
	--装备与buff直接提供的全抗减伤率
	local add_prob = nx_number( nx_query_prob_by_record(self, "AllResistProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = math.min(math.max(base_prob * (4-0.00575*target_level) + add_prob,0.00001),0.8)
	elseif target_level < nx_int(40) then
		value = math.min(math.max(base_prob * (1.82614-0.000131*target_level) + add_prob,0.00001),0.8)
	elseif target_level < nx_int(50) then
		value = math.min(math.max(base_prob * (1.1528-0.0000655*target_level) + add_prob,0.00001),0.8)
	elseif target_level < nx_int(60) then
		value = math.min(math.max(base_prob * (0.87248-0.00133*target_level) + add_prob,0.00001),0.8)
	elseif target_level < nx_int(70) then
		value = math.min(math.max(base_prob * (0.675-0.00135*target_level) + add_prob,0.00001),0.8)
	end
	
	return value
end
function nx_get_AccurateProb(self, target)
	--精准率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "AccurateProb", 0) )
	--装备与buff直接提供的精准率
	local add_prob = nx_number( nx_query_prob_by_record(self, "AccurateProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (4-0.09*target_level) + add_prob + 0.95
	elseif target_level < nx_int(40) then
		value = base_prob * (1-0.011*target_level) + add_prob + 0.95
	elseif target_level < nx_int(50) then
		value = base_prob * (0.72-0.0071*target_level) + add_prob + 0.95
	elseif target_level < nx_int(60) then
		value = base_prob * (0.67-0.0069*target_level) + add_prob + 0.95
	elseif target_level < nx_int(70) then
		value = base_prob * (0.64-0.0068*target_level) + add_prob + 0.95
	end
	
	return value
	
end
function nx_get_MagPrecProb(self, target)
	--法术精准率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "MagPrecProb", 0) )
	--装备与buff直接提供的法术精准率
	local add_prob = nx_number( nx_query_prob_by_record(self, "MagPrecProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (4-0.09*target_level) + add_prob + 0.95
	elseif target_level < nx_int(40) then
		value = base_prob * (1-0.011*target_level) + add_prob + 0.95
	elseif target_level < nx_int(50) then
		value = base_prob * (0.72-0.0071*target_level) + add_prob + 0.95
	elseif target_level < nx_int(60) then
		value = base_prob * (0.67-0.0069*target_level) + add_prob + 0.95
	elseif target_level < nx_int(70) then
		value = base_prob * (0.64-0.0068*target_level) + add_prob + 0.95
	end
	return value
end
function nx_get_DodgeProb(self, target)
	--闪避率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "DodgeProb", 0) )
	--装备与buff直接提供的闪避率
	local add_prob = nx_number( nx_query_prob_by_record(self, "DodgeProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (3.6-0.081*target_level) + add_prob
	elseif target_level < nx_int(40) then
		value = base_prob * (0.9-0.0099*target_level) + add_prob
	elseif target_level < nx_int(50) then
		value = base_prob * (0.648-0.00639*target_level) + add_prob
	elseif target_level < nx_int(60) then
		value = base_prob * (0.603-0.00621*target_level) + add_prob
	elseif target_level < nx_int(70) then
		value = base_prob * (0.576-0.00612*target_level) + add_prob
	end
	
	return value
end
function nx_get_MagdodgeProb(self, target)
	--法术闪避率包含次级属性影响的
	local base_prob = nx_number( nx_query_prob_by_record(self, "MagdodgeProb", 0) )
	--装备与buff直接提供的法术闪避率
	local add_prob = nx_number( nx_query_prob_by_record(self, "MagdodgeProb", 1) )
	
	local target_level = nx_query_property(target, "Level")
	local value = 0.0
	if target_level < nx_int(30) then 
		value = base_prob * (3.6-0.081*target_level) + add_prob
	elseif target_level < nx_int(40) then
		value = base_prob * (0.9-0.0099*target_level) + add_prob
	elseif target_level < nx_int(50) then
		value = base_prob * (0.648-0.00639*target_level) + add_prob
	elseif target_level < nx_int(60) then
		value = base_prob * (0.603-0.00621*target_level) + add_prob
	elseif target_level < nx_int(70) then
		value = base_prob * (0.576-0.00612*target_level) + add_prob
	end
	return value
end
