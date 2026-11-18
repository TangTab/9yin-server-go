--不同门派分配的属性点
school_baseprop_lst = {
	["school_shaolin"] = {
			["Dex"] = 2,
			["Sta"] = 1,
		},
	["school_wudang"] = {
			["Ing"] = 2,
			["Spi"] = 1,
		},
	["school_emei"] = {
			["Ing"] = 1,
			["Spi"] = 2,
		},
	["school_gaibang"] = {
			["Str"] = 2,
			["Dex"] = 1,
		},
	["school_tangmen"] = {
			["Str"] = 1,
			["Dex"] = 2,
		},
	["school_junzi"] = {
			["Ing"] = 1,
			["Spi"] = 1,
			["Sta"] = 1,
		},
	["school_jinyi"] = {
			["Str"] = 1,
			["Sta"] = 2,
		},
	["school_jile"] = {
			["Ing"] = 2,
			["Sta"] = 1,
		},
}

-- 为人物分配基础属性
function alloc_baseprop_point(self, newlevel, oldlevel)
	for level = oldlevel + 1, newlevel do
		if level <= 10 then
			nx_inc_property(self, "Str", 1)
			nx_inc_property(self, "Dex", 1)
			nx_inc_property(self, "Ing", 1)
			nx_inc_property(self, "Spi", 1)
			nx_inc_property(self, "Sta", 1)
		else
			--根据门派的不同分配不同的属性点
			local school = nx_query_property(self, "School")
			if string.len(school) == 0 or school_baseprop_lst[school] == nil then
				--无门派,加3点力量
				nx_inc_property(self, "Str", 3)
			else
				for prop, value in pairs(school_baseprop_lst[school]) do
					nx_inc_property(self, prop, value)
				end
			end
		end
	end
end