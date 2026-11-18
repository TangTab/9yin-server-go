
--通缉值减少操作
function on_use(item, self)

    if not nx_exists(item) then
		return 0
	end
	
	if nx_find_property(self,"ArrestValue")==false then
	    reduce_item(item)
	    return
	end
	
	local arrest_value = nx_query_property(self, "ArrestValue") --获取通缉值
	if nx_number(arrest_value)>nx_number(0) then
	    arrest_value = arrest_value - 100 
	    if arrest_value < 0 then
	        arrest_value = 0
	    end
	    nx_set_property(self,"ArrestValue",nx_int(arrest_value))
	end
	
	reduce_item(item)
end

--减少ITEM数量
function reduce_item(item)
    nx_inc_property(item, "Amount", -1)
end

