--等于某个等级   
function lv_equ(player,actName,val)
	local lv = nx_query_property(player,"Level");
	if lv == val then
		return true;
	end
	return false;
end

--大于某个等级
function lv_lthan(player,actName,val)
	local lv = nx_query_property(player,"Level");
	if lv > val then
		return true;
	end
	return false;
end
--大于等于某个等级
function lv_lequ(player,actName,val)
	local lv = nx_query_property(player,"Level");  
	if lv >= val then
		return true;
	end
	return false;
end
--小于某个等级
function lv_sthan(player,actName,val)
    local lv = nx_query_property(player,"Level");
	if lv < val then
		return true;
	end
	return false;
end
--小于等于某个等级
function lv_seque(player,actName,val)
	local lv = nx_query_property(player,"Level");
	if lv <= val then
		return true;
	end
	return false;
end
	
--属于某个国家
function nation_equ(player,actName,val)
	local nation = nx_query_property(player,"Nation");
	if nation == val then
		return true;
	end
	return false;
end

--不属于某个国家
function nation_nequ(player,actName,val)
	local nation = nx_query_property(player,"Nation");
	if nation ~= val then
		return true;
	end
	return false;
end 

--上线送奖品时间

--领取奖品 
function login_time(player,actName,val)
    if nx_query_property(player,"Level") < 30 then
	return false;
    end
    
    local timeTab; 
    local curTime = nx_global_time();
	timeTab = os.date("*t",curTime);
	
    local row = nx_find_record_value(player,"FreeGiftCompleted",0,actName);
	if row >= 0 then
	    local lastTime = nx_query_record_value(player,"FreeGiftCompleted",row,1);
		lastTab = os.date("*t",lastTime);
		
		if timeTab.year == lastTab.year
		and timeTab.month == lastTab.month
		and timeTab.day == lastTab.day  then
			 return false;            
		else
			nx_remove_record_row(player,"FreeGiftCompleted",row);		   
		end
	end
	
	local timeStartTab = {year=2008, month=8, day=18, hour=0, min=0, sec=0};
	local timeStartVal = os.time(timeStartTab);
	local timeEndTab = {year=2009, month=8, day=24, hour=0, min=0, sec=0};
	local timeEndVal = os.time(timeEndTab);

	if curTime >= timeStartVal and curTime <= timeEndVal and timeTab.hour >= 18 and timeTab.hour < 22 then
		return true;	
	end
	return false;
end

--领取奖品 
function Get_time(player,actName,val)
    if nx_query_property(player,"Level") < 30 then
	return false;
    end

    local timeTab; 
    local curTime = nx_global_time();
	timeTab = os.date("*t",curTime);
	
    local row = nx_find_record_value(player,"FreeGiftCompleted",0,actName);
	if row >= 0 then
	    local lastTime = nx_query_record_value(player,"FreeGiftCompleted",row,1);
		lastTab = os.date("*t",lastTime);
		
		if timeTab.year == lastTab.year
		and timeTab.month == lastTab.month
		and timeTab.day == lastTab.day  then
			 return false;            
		else
			nx_remove_record_row(player,"FreeGiftCompleted",row);		   
		end
	end
	
	local timeStartTab = {year=2008, month=8, day=18, hour=0, min=0, sec=0};
	local timeStartVal = os.time(timeStartTab);
	local timeEndTab = {year=2009, month=10, day=5, hour=0, min=0, sec=0};
	local timeEndVal = os.time(timeEndTab);

	if curTime >= timeStartVal and curTime <= timeEndVal and timeTab.hour >= 10 and timeTab.hour < 13 then
		return true;	
	end
	return false;
end

function bugle1(player,actName,val)
    local timeTab; 
    local curTime = nx_global_time();
	timeTab = os.date("*t",curTime);
	
    local row = nx_find_record_value(player,"FreeGiftCompleted",0,actName);
	if row >= 0 then
	    local lastTime = nx_query_record_value(player,"FreeGiftCompleted",row,1);
		lastTab = os.date("*t",lastTime);
		
		if timeTab.year == lastTab.year
		and timeTab.month == lastTab.month
		and timeTab.day == lastTab.day  then
			 return false;            
		else
			nx_remove_record_row(player,"FreeGiftCompleted",row);		   
		end
	end
	
	local timeStartTab = {year=2008, month=9, day=29, hour=0, min=0, sec=0};
	local timeStartVal = os.time(timeStartTab);
	local timeEndTab = {year=2008, month=10, day=6, hour=0, min=0, sec=0};
	local timeEndVal = os.time(timeEndTab);

	if curTime >= timeStartVal and curTime <= timeEndVal and timeTab.hour >= 15 and timeTab.hour < 17 then
		return true;	
	end
	return false;	
end

function bugle2(player,actName,val)
    local timeTab; 
    local curTime = nx_global_time();
	timeTab = os.date("*t",curTime);
	
    local row = nx_find_record_value(player,"FreeGiftCompleted",0,actName);
	if row >= 0 then
	    local lastTime = nx_query_record_value(player,"FreeGiftCompleted",row,1);
		lastTab = os.date("*t",lastTime);
		
		if timeTab.year == lastTab.year
		and timeTab.month == lastTab.month
		and timeTab.day == lastTab.day  then
			 return false;            
		else
			nx_remove_record_row(player,"FreeGiftCompleted",row);		   
		end
	end
	
	local timeStartTab = {year=2008, month=9, day=29, hour=0, min=0, sec=0};
	local timeStartVal = os.time(timeStartTab);
	local timeEndTab = {year=2008, month=10, day=6, hour=0, min=0, sec=0};
	local timeEndVal = os.time(timeEndTab);

	if curTime >= timeStartVal and curTime <= timeEndVal and timeTab.hour >= 20 and timeTab.hour < 22 then
		return true;	
	end
	return false;	
end

--七天奖励领取判断 
function SevenDateJudge(player,actName)
	local bJudge = SevenDateJudgeSub(player,actName);
	if not bJudge then
		return false;
	end
	
	--一次只能领取一个
	--if nx_find_data(player,"SevenDateJudge") then
	--	return false;
	--end
	--nx_add_data(player,"SevenDateJudge",VARTYPE_INT);
	return true;
end

function SevenDateJudgeSub(player,actName)

    local curTime = nx_global_time();
    local firstGetTime = nx_query_property(player,"SevenDateTime");
    
	if firstGetTime == 0 then
    	--第一次领取，都可以拿
    	if nx_query_property(player,"Level") >= 10 then
    		return true;
		end
		return false;
    end

    --7天后不能领取,开始事间是归0的 
	if math.abs(curTime - firstGetTime) > (7*24*60*60) then	
		return false;
	end
	
	--一天只能领取一次 
    local timeTab; 
	timeTab = os.date("*t",curTime);
	
   	local row = nx_find_record_value(player,"FreeGiftCompleted",0,actName);
	if row >= 0 then
	    local lastTime = nx_query_record_value(player,"FreeGiftCompleted",row,1);
		lastTab = os.date("*t",lastTime);
		
		if timeTab.year == lastTab.year
		and timeTab.month == lastTab.month
		and timeTab.day == lastTab.day  then
			 return false;            
		else
		    --每天大5级
   			timeTab.hour = 0;
    		timeTab.min = 0;
    		timeTab.sec = 0;
			local todayVal = os.time(timeTab);
			local days = (todayVal - firstGetTime)/(24*60*60);
			
            if days < 1 or nx_query_property(player,"Level") < (15 + 5*days) then
                return false;
            end

			nx_remove_record_row(player,"FreeGiftCompleted",row);		   
			
		end
	end	
	
	return true;	
end	
	

function SevenDateSucceed(player,actName)
    local curTime = nx_global_time();
    local firstGetTime = nx_query_property(player,"SevenDateTime");
    
	if firstGetTime > 0 then
		return;   	
    end
    
    --第一次领取，记录开始事间 
    --7天后不能领取,开始事间是归0的 
    local timeTab; 
	timeTab = os.date("*t",curTime);
    timeTab.hour = 0;
    timeTab.min = 0;
    timeTab.sec = 0;
	local timeStartVal = os.time(timeTab);
    nx_set_property(player,"SevenDateTime",timeStartVal);
end

function StartQueryGift(player)
	--if nx_find_data(player,"SevenDateJudge") then
	--	nx_remove_data(player,"SevenDateJudge")
	--end
end

--奖励物品的条件配置
--16000	等级判断
--16001 种族条件
--16002 时间判断
--16003 魅力奖判断
--16004 登陆奖励
--16005 男士魅力奖励
--16006 女士魅力奖励
PresenterGitf = {    	
    ["9000003"] = {
	    [1] =  { cmp = login_time , val = "1" , desc = "16002"}
    	},
     ["9000004"] = {
	    [1] =  { cmp = Get_time , val = "1" , desc = "16002"}
    	},
    ["9000006"]	= {
		[1] = {cmp = bugle1 , val = "1" , desc = "16002"}
	},
    ["9000007"]	= {
		[1] = {cmp = bugle2 , val = "1" , desc = "16002"}
	},
	--["9000100"] = {
	--	[1] = {cmp = lv_lequ , val = "60" , desc = "1"}
	--},		
};


--奖励物品表  
PrizeGiftItems = {
	
	--赠送物品 18-22
	["9000003"] = {title = "16004",
	        items = {
	                [1] = { id = 14110501 , num = 10 },
	                [2] = { id = 14110801 , num = 10 }
			}
	} ,
	--赠送物品10-13
	["9000004"] = {title = "16004",
	        items = {
	                [1] = { id = 14110901 , num = 10 },
	                [2] = { id = 14110902 , num = 5 }
			}
	} ,

	--小喇叭
	["9000006"] = {title = "16004",
		items = {
			[1] = {id = 14110002, num = 1}
		}	
	},
	["9000007"] = {title = "16004",
		items = {
			[1] = {id = 14110002, num = 1}
		}	
	},
	--披风
	--["9000100"] = {title = "16053",
	--	items = {
	--		[1] = {id = 14110503,num = 1}
	--	}			
	--},	
	--新手卡
	["TZ_10000001"] = {title = "16031",
	        items = {
	                [1] = { id = 14240711 , num = 1 },
	                [2] = { id = 13170011 , num = 1 },
	                [3] = { id = 14010402 , num = 1 },
	                [4] = { id = 14010412 , num = 1 },
	                [5] = { id = 14110204 , num = 10 },
	                [6] = { id = 14220101 , num = 1 },
	                [7] = { id = 14220201 , num = 1 },
	                [8] = { id = 14220301 , num = 1 },
	                [9] = { id = 14220401 , num = 1 },
	                [10] = { id = 14220501 , num = 1 },
	                [11] = { id = 14221101 , num = 1 },
	                [12] = { id = 14221201 , num = 1 }
	                
		}
	} ,
	["TZ_14110003"] = {title = "16051",
		    judge_func = SevenDateJudge,
			comp_func  = SevenDateSucceed,
			gift_mark = 1,
			each_count = 1,
	        items = {
	                [1] = { id = 14110003 , num = 1 }	                
		}
	},
		["TZ_10000003"] = {title = "16052",
	        items = {
	                [1] = { id = 14240411 , num = 1 },
	                [2] = { id = 13170011 , num = 1 },
	                [3] = { id = 14010402 , num = 1 },
	                [4] = { id = 14010412 , num = 1 },
	                [5] = { id = 14110204 , num = 10 },
	                [6] = { id = 14220101 , num = 1 },
	                [7] = { id = 14220201 , num = 1 },
	                [8] = { id = 14220301 , num = 1 },
	                [9] = { id = 14220401 , num = 1 },
	                [10] = { id = 14220501 , num = 1 },
	                [11] = { id = 14221101 , num = 1 },
	                [12] = { id = 14221201 , num = 1 },
	                [13] = { id = 14110501 , num = 10 },
	                [14] = { id = 14110801 , num = 10 }
	                
		}
	},
	--活动奖励1
	["TZ_10000004"] = {title = "16054",
	        items = {
	                [1] = { id = 14220103 , num = 1 },
	                [2] = { id = 14220303 , num = 1 },
	                [3] = { id = 14220503 , num = 1 },
	                [4] = { id = 14221203 , num = 1 },
	                [5] = { id = 14225203 , num = 1 },
	                [6] = { id = 14225403 , num = 1 },
	                [7] = { id = 14250102 , num = 2 },
	                [8] = { id = 14220203 , num = 1 },
	                [9] = { id = 14220403 , num = 1 },
	                [10] = { id = 14221103 , num = 1 },
	                [11] = { id = 14225103 , num = 1 },
	                [12] = { id = 14225303 , num = 1 },
	                [13] = { id = 14225503 , num = 1 }
	                
		}
	},
	--活动奖励2
	["TZ_10000005"] = {title = "16055",
	        items = {
	                [1] = { id = 14220103 , num = 1 },
	                [2] = { id = 14220203 , num = 1 },
	                [3] = { id = 14220303 , num = 1 },
	                [4] = { id = 14220403 , num = 1 },
	                [5] = { id = 14220503 , num = 1 },
	                [6] = { id = 14221103 , num = 1 },
	                [7] = { id = 14221203 , num = 1 },
	                [8] = { id = 14250102 , num = 2 }
	                
		}
	},
	--活动奖励3
	["TZ_10000006"] = {title = "16056",
	        items = {
	                [1] = { id = 14220103 , num = 1 },
	                [2] = { id = 14220203 , num = 1 },
	                [3] = { id = 14220303 , num = 1 },
	                [4] = { id = 14220403 , num = 1 },
	                [5] = { id = 14220503 , num = 1 },
	                [6] = { id = 14250102 , num = 1 }
		}
	},
	--活动奖励4
	["TZ_10000007"] = {title = "16057",
	        items = {
	                [1] = { id = 14220102 , num = 1 },
	                [2] = { id = 14220202 , num = 1 },
	                [3] = { id = 14220302 , num = 1 },
	                [4] = { id = 14220402 , num = 1 },
	                [5] = { id = 14220502 , num = 1 },
	                [6] = { id = 14221102 , num = 1 },
	                [7] = { id = 14221202 , num = 1 },
	                [8] = { id = 14225102 , num = 1 },
	                [9] = { id = 14225202 , num = 1 },
	                [10] = { id = 14225302 , num = 1 },
	                [11] = { id = 14225402 , num = 1 },
	                [12] = { id = 14225502 , num = 1 },
	                [13] = { id = 14250101 , num = 1 }
	                
		}
	},
	--活动奖励5
	["TZ_10000008"] = {title = "16058",
	        items = {
         [1] = { id = 14220102 , num = 1 }
		}
	},
		--开卡送好礼一
	["TZ_10000009"] = {title = "16125",
	        items = {
         [1] = { id = 14110004 , num = 10 }
		}
	},
		--开卡送好礼二
	["TZ_10000010"] = {title = "16126",
	        items = {
         [1] = { id = 14110204 , num = 2 }
		}
	},
		--开卡送好礼三
	["TZ_10000011"] = {title = "16127",
	        items = {
         [1] = { id = 14010108 , num = 30 }
		}
	},
		--开卡送好礼四
	["TZ_10000012"] = {title = "16128",
	        items = {
         [1] = { id = 13170014 , num = 1 }
		}
	},
		--开卡送好礼五
	["TZ_10000013"] = {title = "16129",
	        items = {
         [1] = { id = 14226604 , num = 1 }
		}
	},
		--开卡送好礼六
	["TZ_10000014"] = {title = "16130",
	        items = {
         [1] = { id = 14226605 , num = 1 }
		}
	},
	--中级经验卷
	["14110202"] = {title = "Exp_Add_02",
	        items = {
         [1] = { id = 14110202 , num = 1 }
		}
	},
	--天子帝王卡
	["TZ_10000015"] = {title = "TZ_Gift_01",
	        items = {
	                [1] = { id = 14241030 , num = 1 },
	                [2] = { id = 14110509 , num = 1 },
	                [3] = { id = 14110204 , num = 10 },
	                [4] = { id = 14010402 , num = 1 },
	                [5] = { id = 14010412 , num = 1 },
	                [6] = { id = 14226102 , num = 1 },
	                [7] = { id = 14226202 , num = 1 },
	                [8] = { id = 14226302 , num = 1 },
	                [9] = { id = 14226402 , num = 1 },
	                [10] = { id = 14226502 , num = 1 },
	                [11] = { id = 14226602 , num = 1 },
	                [12] = { id = 14226702 , num = 1 },
	                [13] = { id = 13170011 , num = 1 },
	                [14] = { id = 14110006 , num = 1 }
	                
		}
	},
	--龙凤婚庆礼包
	["TZ_10000016"] = {title = "TZ_Gift_02",
	        items = {
         [1] = { id = 14110801 , num = 10 },
         [2] = { id = 14110205 , num = 2 }
		}
	}
};