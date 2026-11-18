-----------------------------------------
-- @ 脚本功能: 黑金生产价值计算和公式调整
-----------------------------------------

--计算每个ITEM价值
function get_sum(item)
    local sum = 0.0
    sum = nx_query_property(item, "Worth")*nx_query_property(item, "Amount")
    return sum
end

--生成的数量
function get_count(sumvalue,camp)
    local count = sumvalue/500
    local config = nx_string("EnergyF_ty_00003C")
    
    if nx_int(camp)==nx_int(2) then
        config = nx_string("Energyh_qh_00001A")
    end
    
    return nx_int(count),nx_string(config)
end


