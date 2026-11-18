require("define")
function on_use(item, self)

    if not nx_exists(item) then
		return 0
	end
	
	if nx_find_property(self,"AddDepotBagNum") == false then
	    --reduce_item(item)
	    return
	end
	
	local ext_depot_value = nx_query_property(self, "AddDepotBagNum") --扩展槽位值
	if nx_number(ext_depot_value) >= nx_number(0) and nx_number(ext_depot_value) < nx_number(2) then
	    ext_depot_value = ext_depot_value + 1
	    if ext_depot_value < 0 then
	        ext_depot_value = 0
	    end
	    nx_set_property(self,"AddDepotBagNum",nx_int(ext_depot_value))
	    reduce_item(item)
	    nx_sysinfo(self, TIPSTYPE_SYSTEMNO, "BG_123011")
	elseif nx_number(ext_depot_value) >= nx_number(2) then
        nx_sysinfo(self, TIPSTYPE_SYSTEMNO, "BG_123010")
	end
end

--减少ITEM数量
function reduce_item(item)
    nx_inc_property(item, "Amount", -1)
end
