
function on_use(item, self)
    if not nx_exists(item) then
		return 0
	end
	
	nx_set_property(self,"BagDepotLink",nx_int(1))
	reduce_item(item)
end

--减少ITEM数量
function reduce_item(item)
    nx_inc_property(item, "Amount", -1)
end
