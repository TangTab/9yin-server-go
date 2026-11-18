require("define")

--PK惩罚系数(PK数值按10等级)
--PK值范围
--0~999
--1000~1999
--2000~2999
--3000~3999
--4000~4999
--5000~5999
--6000~6999
--7000~7999
--8000~8999
--9000~9999
--10000
-- 金钱掉落系数	 装备耐久损失系数	修为减少系数	耐久归零几率
pkvaule_publish = 
{
	["0"]  = { 1.0,	 1.0,	 1.0,		0 },
	["1"]  = { 1.5,	 1.5,	 1.5,		0 },
	["2"]  = { 2.0,	 2.0,	 2.0,		0 },
	["3"]  = { 2.5,	 2.5,	 2.5,		0.15 },
	["4"]  = { 3.0,	 3.0,	 3.0,		0.15 },
	["5"]  = { 3.5,	 3.5,	 3.5,		0.30 },
	["6"]  = { 4.0,	 4.0,	 4.0,		0.30 },
	["7"]  = { 4.5,	 4.5,	 4.5,		0.50 },
	["8"]  = { 5.0,	 5.0,	 5.0,		0.50 },
	["9"]  = { 5.5,	 5.5,	 5.5,		0.70 },
	["10"] = { 5.5,	 5.5,	 5.5,		0.70 },
}

--杀死对方的惩罚
function kill_punish(self)
end

--杀死对方的收益
function kill_income(self,target)
    	-- 保护
	if not nx_exists(self) then
		return 0
	end
	
	nx_kill_income(self,target)
end

--被杀死后的惩罚
function bekill_punish(self, attacker)
	-- 保护
	if not nx_exists(self) then
		return 0
	end
	
	--给NPC杀死或者自然死亡
--	local moneyper     = 0.1
	local exper   = 0.0
	local hardinessper = 0.0
	local arrestvalue = 0

	--如果是被玩家杀死
	if nx_exists(attacker) then-- and 
--	   nx_type(attacker) == TYPE_PLAYER then
		
		local ArrestValue = nx_query_property(self,"ArrestValue")
		local nextexp = nx_query_property(self,"PlayerUpgradeExp")
		
		if ArrestValue<300 then
			exper=0.05*nextexp
			hardinessper=0.1
		else
			exper=0.05*nextexp*ArrestValue/100
			hardinessper=0.2
		end
	
	end
	
	--获取自己的PK系数
--	local index = "1"
--	if nx_find_property(self, "PKValue") then
--		index = nx_string(nx_int(nx_query_property(self, "PKValue")/1000))
--	end
	
	
	--系数生效
--	moneyper      = moneyper * pkvaule_publish[index][1]
	--facultyper    = facultyper * pkvaule_publish[index][2]
	--hardinessper  = hardinessper * pkvaule_publish[index][3]
--	local zeroper = pkvaule_publish[index][4]
	
	--传服务器扣除
	local success = nx_deadpunish(self,exper, hardinessper,arrestvalue)
end