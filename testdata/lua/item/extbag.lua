require("define")
function on_use(item, self)

    if not nx_exists(item) then
		return 0
	end
	
	if nx_find_property(self,"AddToolBagNum") == false then
	    --reduce_item(item)
	    return
	end
	
	local ext_bag_value = nx_query_property(self, "AddToolBagNum") --扩展槽位值
	if nx_number(ext_depot_value) >= nx_number(0) and nx_number(ext_bag_value) < nx_number(2) then
	    ext_bag_value =  ext_bag_value + 1
	    if ext_bag_value < 0 then
	        ext_bag_value = 0
	    end
	    nx_set_property(self,"AddToolBagNum",nx_int(ext_bag_value))
	    reduce_item(item)
	    nx_sysinfo(self, TIPSTYPE_SYSTEMNO, "BG_122011")
	elseif nx_number(ext_bag_value) >= nx_number(2) then
        nx_sysinfo(self, TIPSTYPE_SYSTEMNO, "BG_122010")
	end
end

--减少ITEM数量
function reduce_item(item)
    nx_inc_property(item, "Amount", -1)
end
