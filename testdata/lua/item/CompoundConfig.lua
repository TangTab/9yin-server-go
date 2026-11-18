-----------------------------------------
-- @ 脚本名称: CompoundConfig.lua
-- @ 脚本功能: 道具合成属性的加成公式，合成的几率公式
-- @ 创建者:  王振闯
-- @ 创建日期: 11.08.04
-----------------------------------------


--合成的几率公式
function get_compound_odds()
    -- 成功几率
    local succeed = 0.15
    -- 失败几率（不摧毁物品）
    local unsucceed = 0.6
    --失败（摧毁物品）
    local fail = 0.25
    
    return succeed, unsucceed, fail
end

--合成需要的金钱
function get_request_money()
    return 10101
end

-- 物品合成属性的结果  (is_succees==1则合成成功的公式， 负则是不成功的公式)
function get_compound_result(fri_value, sec_value, is_succees)

    if 1 == is_succees then
        if math.fmod((fri_value + sec_value), 2)> nx_number(0) then
            return (fri_value + sec_value + 1) / 2
        else
            return (fri_value + sec_value) / 2 + 1
        end
    else    
        if math.fmod((fri_value + sec_value), 2) > nx_number(0) then
            return (fri_value + sec_value - 1)/ 2
        else
            local res_value = (fri_value + sec_value) / 2 - 1
            if res_value < 0 then
                res_value = 0
            end
            return res_value
        end
    end

end



---------------------------------------------
-- 道具打孔配置信息
-------------------------------------------------

--根据道具的级别获取道具可以打多少空需要多少个石头
function get_hole_maxcount(nlevel)
    
    if 2 == nlevel then
        return 1
    elseif 3 == nlevel then
    	return 2
    elseif 4 == nlevel then
    	return 3
    elseif 5 == nlevel then
    	return 4
    elseif 6 == nlevel then 
		return 5
    end
    
end

-- 获取石头类型
function get_stone_type()
    return nx_string("Other_001")
end

-- 获取道具打孔的成功几率
function get_slot_odds()
    -- 成功几率
    local succeed = 1
    --失败（摧毁物品）
    local fail = 0
    
    return succeed, fail
end

-----------------------------------------
-- @ 添加功能: 道具打孔需要黑金或者点券的数量
-- @ 添加者:  孟腾飞
-- @ 添加日期: 2012.03.05
-----------------------------------------
--打孔需要的点券和黑金
function get_request_material()
    --点券数
    local ticket = 0
    --黑金数
    local blackgold = 1
    return ticket, blackgold
end

--获取消耗的物品和数量
function get_blackgold_type()
    return nx_string("EnergyF_ty_00003C"),nx_int(1)
end