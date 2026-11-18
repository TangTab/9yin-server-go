--装备的颜色级别
function compute_hole(item)
    local colorlevel = nx_query_property(item, "ColorLevel")
    
    if not nx_find_property(item,"CurHoleCount") then 
        return
    end
    
    local probabilty = 0 --随机出来的当前孔
    local curhole = 0
    
    if colorlevel>=2 then --绿色
        probabilty = math.random(0,100)
		--nx_msgbox("probabilty:"..nx_string(probabilty))
		--[[
        if probabilty <= 1 then  --1%概率
            curhole = 2
        elseif probabilty <= 100 then
            curhole = 1
        end
		]]--
		if probabilty <= 5 then
			curhole = 1
		end
    end 
	nx_set_property(item, "CurHoleCount", curhole);
end
