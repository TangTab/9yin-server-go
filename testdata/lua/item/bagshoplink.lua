
function on_use(item, self)
    if not nx_exists(item) then
		return 0
	end
	
	nx_set_property(self,"BagShopLink",nx_int(1))
	reduce_item(item)
end

--¼õÉÙITEMÊıÁ¿
function reduce_item(item)
    nx_inc_property(item, "Amount", -1)
end
