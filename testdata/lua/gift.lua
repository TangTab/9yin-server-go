require("define")
require("logdefine")
require("gift_config")


--物品ID到名称对应表  
ItemNameMap = {};

bInit = 0;
--初始化物品ID与Name的对应表    
function InitNameMap(npc)
	if bInit == 1 then
		return 1;
	end
	bInit = 1;
	for i , it in pairs(PrizeGiftItems) do
		for j , sub in pairs(it.items) do
			if ItemNameMap[sub.id] == nil then
				ItemNameMap[sub.id] = {};
				--先用接口取NameID
				local item_name = nx_get_config_property(sub.id,"NameID");
				local item_amount = nx_get_config_property(sub.id,"Amount");
				local item_maxamt = nx_get_config_property(sub.id,"MaxAmount");
				local item_bind = 0;
				--如果接口取不到 创建物品再取NameID属性
				if item_name == nil or item_name == "" or item_amount == nil or item_amount == "" or item_maxamt == nil or item_maxamt == "" then
					local titem = nx_create_from_id(npc,sub.id);
					if titem ~= nil and nx_exists(titem) then

						--NameID
						ItemNameMap[sub.id].name = nx_query_property(titem,"NameID");

						--Amount
						if not nx_find_property(titem,"Amount") then
                        	ItemNameMap[sub.id].amount = 1;
						else
                            ItemNameMap[sub.id].amount = nx_query_property(titem,"Amount");
						end

						--MaxAmount
						if not nx_find_property(titem,"MaxAmount") then
							ItemNameMap[sub.id].maxamt = 1;
						else
							ItemNameMap[sub.id].maxamt = nx_query_property(titem,"MaxAmount");
						end
						
						--Bind
						ItemNameMap[sub.id].bind = 0;
						
						nx_destroy_self(titem);
					end
				else
				    ItemNameMap[sub.id].name = item_name;
					ItemNameMap[sub.id].amount = nx_int(item_amount);
					ItemNameMap[sub.id].maxamt = nx_int(item_maxamt);
					ItemNameMap[sub.id].bind = 0;
				end
			end 
		end
	end

	for j , it in pairs(PrizeGiftItems) do
		for j , sub in pairs(it.items) do
			if ItemNameMap[sub.id] ~= nil then
				sub.name = ItemNameMap[sub.id].name;
				sub.amount = ItemNameMap[sub.id].amount;
				sub.maxamt = ItemNameMap[sub.id].maxamt;
				
				if sub.bind == nil then 
					sub.bind = ItemNameMap[sub.id].bind;
				end
			end
		end
	end
end

--查询该等级的奖励是否已经发放过
function LevelGift_IsAdded(player,gift)
	   --在已经完成的等级奖励表中查找
	   local row = nx_find_record_value(player,"FreeGiftCompleted",0,gift);
	   if row >= 0 then
			return true;
	   end
	   return false;
end

--发放一个等级奖励
function AddOneTaskGift(player,giftName,num)
	--0数量
	if num < 1 then
	    return;
	end
	local row = nx_add_record_row_values(player,"PrizeGiftList",tonumber(giftName),num,giftName,2);
	if row < 0 then
		return;
	end
	return
end

--完成一个等级奖励领取
function CompleteOneTaskGift(player,giftName,num)
	local row = nx_find_record_value(player,"FreeGiftCompleted",0,giftName);
	if row >= 0 then
		local str = "complete level gift " .. giftName .. " repeat "  .. " maybe error";
		nx_save_giftlog(nx_query_property(player,"Name"),nx_widestr(str));
		return ;
	end
	local arow = nx_add_record_row_values(player,"FreeGiftCompleted",giftName,nx_global_time());
	if arow < 0 then
		local ss = "save complete level gift " .. giftName .. " faild   !!!!error!!!!";
		nx_save_giftlog(nx_query_property(player,"Name"),nx_widestr(ss));
	end
end 

--获得任务型奖励
function GetLevelGift(npc,player)

	for name , it in pairs(PresenterGitf) do
		--判断 是否满足领取条件	
		local br = JudgeGiftSend(player,name);
		if br == 1 then
		--判断是否已经领取
            if not LevelGift_IsAdded(player,name) then
                	AddOneTaskGift(player,name,1);
			end
		end
	end
end

function ViewWaitingInfo(self, sender,mark)
	nx_begin_menu(sender,self);
	--提示玩家进行等待
	if mark == 0 then
		nx_add_title(sender, "gift_str_1");
	else
		nx_add_title(sender, "gift_str_2");
	end
	nx_end_menu(sender, self);
end

function JudgeGiftSend(sender,name)
	local giftJudge = PresenterGitf[name];
	if giftJudge ~= nil then
		--判断 是否满足领取条件
		for j , fn in ipairs(giftJudge) do
			if fn.cmp ~= nil then
				if not fn.cmp(sender,name,nx_int(fn.val)) then
				 	return 0;
				end
			end
		end 		
	end
	return 1;
end

function InsertItemAction(sender,name)
	if PrizeGiftItems[name] ~= nil then
		return PrizeGiftItems[name]; 
	end

	local itemId = tonumber(string.sub(name,4))	
	if itemId == nil or itemId == 0 then
		return nil;
	end
	
	--nx_msgbox("11111");
	local item_name = nx_get_config_property(itemId,"NameID");
	if item_name ~= nil and string.len(item_name) > 0 then  

		local item_amount = nx_get_config_property(itemId,"Amount");
		local item_maxamt = nx_get_config_property(itemId,"MaxAmount");		
        
		PrizeGiftItems[name] = {};
		PrizeGiftItems[name].items = {};
		PrizeGiftItems[name].title = item_name;
		
	    --插入物品 
		local inItem = {};
		inItem.num  = 1; 
		inItem.id   = itemId;					
		inItem.name = item_name;
		inItem.amount = item_amount;
		inItem.maxamt = item_maxamt;       
		inItem.bind = 0;

		--插入活动
		table.insert(PrizeGiftItems[name].items,inItem);				
		--nx_msgbox("insert");
		--nx_msgbox(name);
		--nx_msgbox(PrizeGiftItems[name].title);
    end
	
	return PrizeGiftItems[name];		
end

--活动列表
function ViewActionList(self,sender)
    --nx_add_record_row_values(sender,"PrizeGiftList",14110901,1,"TZ_14110901",0);
	--nx_add_record_row_values(sender,"PrizeGiftList",10000002,1,"TZ_10000002",0);

	--显示所有可以接受的活动
	local bMenu = true;
	local rows = nx_get_record_rows(sender,"PrizeGiftList");
	for i = 0,rows - 1 do

		local id  = nx_query_record_value(sender,"PrizeGiftList",i,0);
		local num = nx_query_record_value(sender,"PrizeGiftList",i,1);
		local name= nx_query_record_value(sender,"PrizeGiftList",i,2);
		local type= nx_query_record_value(sender,"PrizeGiftList",i,3);

		local it = PrizeGiftItems[name];
		
		--[[nx_msgbox("11");
		nx_msgbox(name);
		--没有配置的物品奖励
		if it == nil then
			--但必须是物品，"TZ_100.....",不是物品
			local pos = string.find(name,"_100")			 
			--看看是不是物品型的奖励			
			if pos == nil or pos < 1 then 	
				--插入一个物品奖励 
				nx_msgbox("1ww");
				it = InsertItemAction(sender,name);	
			end					 
		end]]
		
		if it ~= nil then
		    local bShow = true;
			if it.judge_func ~= nil then  
				bShow = it.judge_func(sender,name);	
			end
			
			if bShow then
				--nx_msgbox(it.title);
				if bMenu then
					--显示领取菜单
				    bMenu = false;
					nx_begin_menu(sender, self);
					--领取物品
					nx_add_title(sender, "1020X", nx_query_property(self, "Name"));
				end
				
				if type == 0 and it.each_count ~= nil then
				    --每次有限制次数
				    if it.each_count < num then
				        nx_set_record_value(sender,"PrizeGiftList",i,1,it.each_count);
				    end
				end
				
				nx_add_menu(sender, 55200 + i,it.title);
			end
		end
	end
		
	if not bMenu then
		--离开
		nx_add_menu(sender, 9600, "gift_str_5");
		nx_end_menu(sender, self);
	else
		ViewWaitingInfo(self, sender,1);
	end
	return 0;
end    

--奖励信息列表
function ViewPrizeInfo(self, sender, idx)    
	--提示获得的奖励
	nx_begin_menu(sender, self);

	local prizeNum  = nx_query_record_value(sender,"PrizeGiftList",idx,1);
    local prizeName = nx_query_record_value(sender,"PrizeGiftList",idx,2);
	local prizeType = nx_query_record_value(sender,"PrizeGiftList",idx,3);
	
	if prizeNum < 1 or string.len(prizeName) <= 0 then
	    return 0;
	end
	--nx_msgbox(prizeName);
	local item = PrizeGiftItems[prizeName];
	if item == nil then	
		return 0;
	end 	

	--以为是异步领取所以帮定一个临时的数据
	if not nx_find_data(sender,"GetPrizeNum") then
		nx_add_data(sender,"GetPrizeNum",2);
	end
	
	local sTitle = "";
	local sArg = {};
	local sParam = {};	
	items = item.items;
	if items ~= nil then
        --物品信息
		for i,it in ipairs(items) do
		    local aindex = (i - 1)*2;
		    local sItem = "{@" .. aindex + 0 .. "} X {@" .. aindex + 1 .. "}<br/>";			
			table.insert(sArg,it.name);
			table.insert(sArg,it.num);
			sTitle = sTitle .. sItem;
		end
		

		--检查背包能不能装下所有的物品
		for n = 1 , prizeNum  do
	    	for i,it in pairs(items) do
				local str = "" .. tostring(it.id) .. " " .. tostring(it.amount) .. " " .. tostring(it.maxamt) .. " " .. tostring(it.bind);
				for j = 1 , it.num do
					table.insert(sParam,str);
				end
	    	end
	    end
	    --不能全部装下的话 检查能不能分批领取
		local toolbox = nx_get_child(sender,nx_widestr("ToolBox"));
		if nx_container_testplace_items(toolbox,unpack(sParam)) == 0 then
			local type = nx_query_record_value(sender,"PrizeGiftList",idx,3);
			--提示空间不足
			if type == 0 then       --数据库中的奖励可以分批取
				--计算背包可以容纳多少个
				local realnum = CalcToolBoxCapacity(toolbox,prizeName,prizeNum);
				if realnum <= 0 then
					nx_sysinfo(sender,TIPSTYPE_SYSTEMNO,"gift_str_7");
					return 0;       
				end
				nx_set_data(sender,"GetPrizeNum",realnum);	
				
			elseif type == 1 then   --公共数据中的奖励 不能分批取
				nx_sysinfo(sender,TIPSTYPE_SYSTEMNO,"gift_str_7");
				return 0;
			elseif type == 2 then   --升级奖励 不能分批取
				nx_sysinfo(sender,TIPSTYPE_SYSTEMNO,"gift_str_7");
				return 0;
			end
	    end
	end

	--设置数量
	nx_set_data(sender,"GetPrizeNum",prizeNum);

	--离开
	nx_add_title(sender, sTitle , unpack(sArg));
	nx_add_menu(sender, 55400 + idx, "gift_str_6");
	nx_add_menu(sender, 5600, "gift_str_5");
	nx_end_menu(sender, self);    
	return 0;
end



--计算当前背包可以容纳多少件prize
function CalcToolBoxCapacity(toolbox,prize_name,num)

	  local prize = PrizeGiftItems[prize_name];
	  local items = prize.items;
	  local anum = 0;
      local szParam = "";
	  local titems = {};
	  for i = 1 , num do
		--每件prize里的物品
		for j , it in pairs(items) do
			szParam  = szParam .. tostring(it.id) .. " " .. tostring(it.amount) .. " " .. tostring(it.maxamt) .. " " .. tostring(it.bind) .. "|";
			if string.len(szParam) <= 0 then
				return anum;
			end
			table.insert(titems,szParam);
		end

		--测试能不能放下
		if nx_container_testplace_items(toolbox,unpack(titems)) <= 0 then
			return anum;
		end
		anum = anum + 1;
	  end
	  
	  return anum;
end

--玩家选择Ok菜单
function OnMenuOk(self, sender, idx)
 	local giftId   = nx_query_record_value(sender,"PrizeGiftList",idx,0);
	local prizeNum = nx_query_record_value(sender,"PrizeGiftList",idx,1);
	local type     = nx_query_record_value(sender,"PrizeGiftList",idx,3);
	local prizename = nx_query_record_value(sender,"PrizeGiftList",idx,2);

	--因为是异步所以保存index
	if not nx_find_data(sender,"GetPrizeIndex") then
	    nx_add_data(sender,"GetPrizeIndex",2);
	end

    --取出要领取的数量 (仅数据库中领取的时候有效)
	if not nx_find_data(sender,"GetPrizeNum") then
		return 0;
	end
	
	local pNum = nx_query_data(sender,"GetPrizeNum")
	if pNum < 1 then
	    return 0;
	end

	if type == 0 then --数据库中奖励 
	    --检查能放进多少个
	    local realnum = CalcToolBoxCapacity(nx_get_child(sender,nx_widestr("ToolBox")),prizename,pNum);
	    if realnum <= 0 then
			return 0;
		end
		nx_get_gift(sender,giftId,prizename,realnum);
	elseif type == 1 then --公共数据记录的奖励 
	    if OnGetGift(self,sender,giftId,prizeNum,prizename) == 1 then
	        if not nx_romove_failgift_row(sender,giftId,prizename,prizeNum) then
		    	local pName = nx_query_property(sender,"Name");
				local sLog  = "romove fail gift:gift=" .. tostring(giftId) .. ",giftname=" .. tostring(prizename);
				nx_save_playerlog(pName,PLAYER_LOG_GIFT,nx_widestr(sLog));
			end 
	    end
	elseif type == 2 then 	--任务型奖励
	    OnGetGift(self,sender,giftId,prizeNum,prizename);
	end

	return 1;
end

function OnGetGift(npc,sender,actionId,num,name)
	if num < 1 then
	    return 0;
	end
	
	if not nx_find_data(sender,"GetPrizeIndex") then
	    return 0;
	end
	local prizeIndex = nx_query_data(sender,"GetPrizeIndex");

	local prizetype = nx_query_record_value(sender,"PrizeGiftList",prizeIndex,3);
	local action = PrizeGiftItems[name];
	if action == nil then
		return 0;
	end	

	local sParam = {};
	items = action.items;
	
	--测试所有物品能不能放进背包
	if items ~= nil then
	    --物品信息串
		for n = 1 , num  do
	    	for i,it in pairs(items) do
				local str = "" .. tostring(it.id) .. " " .. tostring(it.amount) .. " " .. tostring(it.maxamt) .. " " .. tostring(it.bind);
				for j = 1 , it.num do
					table.insert(sParam,str);
				end
	    	end
	    end
	end
	
    --不能全部装下的话失败
	local toolbox = nx_get_child(sender,nx_widestr("ToolBox"));
	if nx_container_testplace_items(toolbox,unpack(sParam)) == 0 then 		
		return 0;
	end
	
	local vt = {};
	--创建物品放进背包
	if items ~= nil then
		for n = 1 , num do
		
			for i , it in pairs(items) do
			
				for j = 1 ,it.num do 
					local item = nx_create_from_id(nx_get_scene(),it.id);
					if item == nil then
						--如果有一个失败全部删除
						for v , vitem in pairs(vt) do
      						nx_destroy_self(vitem);
						end
						return 0;
					end

					nx_set_property(item,"BindStatus",1);		        	
                	if nx_container_placeitem(toolbox,item) == 0 then
                	    for v , vitem in pairs(vt) do
      						nx_destroy_self(vitem);
						end
						return 0;
					end
					
					table.insert(vt,item);
					nx_sysinfo(sender, TIPSTYPE_SYSTEMNO , "gift_getmsg", it.name);
				end
			end
		end
	end
	
	--提示获得的物品
	local sLog = "get gift:" .. name .. ",items="; 
	for i , it in pairs(vt) do		
		sLog = sLog .. nx_get_config(it);
		sLog = sLog .. ",";
	end
		
	nx_save_playerlog(nx_query_property(sender,"Name"),PLAYER_LOG_GIFT,nx_widestr(sLog)); 
	
	--if action.money > 0 then
	--end
	--if action.exp > 0 then
	--end
	if action.comp_func ~= nil then
       action.comp_func(sender,name)
	end
	
	if prizetype == 2 or action.gift_mark ~= nil then
		CompleteOneTaskGift(sender,name,num);
	end
	
	ViewSuccessInfo(npc,sender,action.title);
	return 1;
end

function ViewSuccessInfo(self,sender,title)
    --nx_sysinfo(sender,1,"tostring(funcid)")
	nx_begin_menu(sender, self);
	--欢迎光临
	nx_add_title(sender, "gift_getgift", nx_widestr(title));
	nx_add_menu(sender, 2, "gift_str_8");
	nx_add_menu(sender, 5600, "gift_str_5");
	                                            
	nx_end_menu(sender, self);	
end  

--NPC读取配置文件之后
function npc_on_load_npc(self, sender)
	return InitNameMap(self);
end

--有玩家选择了NPC
function npc_on_select_giftnpc(self, sender, func)
	nx_begin_menu(sender);
	
	nx_add_title(sender, "10011");--此功能未开放
	
	nx_add_menu(sender, 2, "10002");--返回
	
	nx_end_menu(sender, self);	
	
	return
	--if func == 55000 then
	--	npc_func_55000(self, sender);
	--elseif func == 55002 then
	--	npc_func_55002(self,sender,func);
	--elseif func >= 55100 and func < 55200 then
	--	npc_func_55100(self,sender);
	--elseif func >= 55200 and func < 55300 then
	--	npc_func_55200(self,sender,func);
	--elseif func >= 55400 and func < 55900 then
	--	npc_func_55400(self,sender,func); 		
	--end
end

function npc_func_55000(self, sender)
	nx_begin_menu(sender, self);
	--领取物品 
	nx_add_title(sender, "1020X", nx_query_property(self, "Name"));
	nx_add_menu(sender, 55100, "gift_str_4"); --察看物品 
	--nx_add_menu(sender, 55002, "get_exp_mul"); -- 1.5倍经验
	nx_add_menu(sender, 56000, "gift_str_5");
	nx_end_menu(sender, self);
end

function npc_func_55002(self,sender,func)
	nx_lua_runfunc("Program.lua","on_get_addtionexp",self,sender,func);
end


function npc_func_55100(self,sender)
	--初始化名字表 
	InitNameMap(self);

	--开始查询	
	StartQueryGift(sender);
	
	--奖励列表清空 
	nx_clear_record(sender,"PrizeGiftList");
	                                                                                
	if nx_query_gift(sender) then
		--获得任务型奖励 
		GetLevelGift(self,sender);
		return ViewWaitingInfo(self,sender,0);
	end
end


function npc_func_55200(self,sender,func)
		--点击奖励 
		local row = func - 55200;
		ViewPrizeInfo(self,sender,row);
		return 0;
end

function npc_func_55400(self,sender,func)
		--点确定
	    local row = func - 55400;
	    OnMenuOk(self,sender,row);
	    return 0;
end