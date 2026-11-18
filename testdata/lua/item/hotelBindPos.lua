require("share\\itemtype_define")
require("define")
require("share\\pk_define")
require("share\\logicstate_define")
require("share\\item_static_data_define")

--使用物品的通用脚本
--返回0使用成功
--返回1使用失败
function on_use(item, target)
	-- 有效性检查：该物品是否存在
	if not nx_exists(item) then
		return 0
	end
	nx_gotopos(item,target)
	return 0
end

--是否可以使用
function can_use(item, target)	
	-- 有效性检查：该物品是否存在
	if not nx_exists(item) then
		return 0
	end
	
	-- 执行C++函数判断炉石是否能使用 能：1，不能：0
	local result = nx_canuse(item, target)
	
	return result
end

