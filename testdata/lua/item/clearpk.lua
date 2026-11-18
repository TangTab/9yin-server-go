require("share\\item_static_data_define")

--清除通缉值点数
function on_use(item, target)
    nx_item_decpkvalue(target)
	return 1
end