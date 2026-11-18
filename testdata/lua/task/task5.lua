-----------------------------------------
-- @ 脚本名称: task5.lua
-- @ 脚本功能: ID为5的任务执行脚本逻辑
-- @ 创建者:   dReAmSun
-- @ 创建日期: 2009.3.11
-----------------------------------------
require("share\\view_define")


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	-- 得到npc的商店ID编号
	local shop_id = nx_query_property(npc, "ShopID")
	
	-- 自动弹出该商店界面
	open_shopbox(npc, player, shop_id)
	
	-- 得到商店容器
	local shopbox = nx_get_child(npc, nx_widestr("ShopBox"))
	if not nx_exists(shopbox) then
		return 0
	end
	
	-- 寻找商店中的要购买的物品taskitem_2高亮显示
	local pos = nx_find_item_pos(shopbox, nx_string("taskitem_2"))
	local realpos = nx_int(pos) - nx_int(1)
		
	-- 如果得到正确位置则高亮显示此装备
	if nx_int(pos) > nx_int(0) then
		nx_custom(player, nx_string("show_animation"), nx_string("ShopBoxShow"), nx_int(realpos))
	end

	return 1
end

--打开普通商店
function open_shopbox(self, sender, shop_id)

	--通知客户端打开指定的商店
	if nx_find_viewport(sender, VIEWPORT_SHOP) then
		nx_remove_viewport(sender, VIEWPORT_SHOP);
	end

	local shopbox = nx_get_child(self, nx_widestr("ShopBox"));

	nx_add_viewport(sender, VIEWPORT_SHOP, shopbox);--打开仓库容器的视图
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)

	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)

	return 1
end

