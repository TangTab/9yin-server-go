--装备的颜色级别
function refresh_color(equip)
	local colorlevel = 0;
	for i = 0, 3 do
		local prop = "AddPropNo" .. i;
		if nx_query_property(equip, prop) > 0 then
			colorlevel = colorlevel + 1;
		end
	end

	nx_set_property(equip, "ColorLevel", colorlevel);
end
