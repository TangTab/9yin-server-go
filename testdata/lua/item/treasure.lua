require("share\\view_define")
require("share\\item_static_data_define")
require("define")
require("share\\server_custom_define")
--打开大礼包操作
function on_use(item, self)
    if not nx_exists(item) then
		return 0
	end
	
	local opened = nx_query_property(item, "Opened") --获取是否打开的标记
	local uid    = nx_query_property(item,"UniqueID")
	
	if not nx_find_data(self,"pickitem") then --记录打开的宝箱item
	    nx_add_data(self,"pickitem",VTYPE_OBJECT)
	end
	
	nx_set_data(self,"pickitem",item)
	
	if opened >=1 then --已经打开过，不再处理
	    nx_custom(self,SERVER_CUSTOMMSG_PICK_ITEM,nx_string(uid))--通知UI显示
	    return 0
	end
		-- 判断item的礼包编号
	local result = nx_item_static_query(item, Item_MaterialPack_Result)
	local count  = nx_item_static_query(item, ITEM_MaterialPack_Count)--要随机出来的个数
	local array =  nx_split_string(result, ";")         --数组单个元素;configid,count,proability
	local tablecount = table.getn(array)

	if tablecount < count then              --安全
	    return 0
	end
		
	local bit={}--随机标志数组,及初始化
	local proabilty={} --每个item的概率
	local proabilitysum = 0.0 --概率之和
	for i=1,tablecount do
	    bit[i] = false
	    local m = nx_split_string(array[i], ",") 
	    proabilty[i] = m[3]         --概率
	    proabilitysum = proabilitysum + proabilty[i]
	end

	--产生随机序列，保存在最后
	for i=1,count do
	    local n = math.random(0,nx_number(proabilitysum*100))
	    local m = 0.0
	    for j=1,tablecount do
	        if not bit[j] then--没有被选中
	            m = m + proabilty[j]
	        end
	        if m*100 > n then  --
	        --	nx_msgbox(nx_string(n)..","..nx_string(m*100)..","..nx_string(j))
	            bit[j]=true
	            proabilitysum = proabilitysum - proabilty[j]
	            break
	        end
	    end
	end
	
	local stresult = "" --保存选中的结果
	for i=1,tablecount do
	    if bit[i] then
            local ar = nx_split_string(array[i], ",") --1:configid,2:个数
            local configid= ar[1]
            local config_count = ar[2]
            stresult = configid..","..config_count..";"..stresult
	    end
	end
	--保存结果
	nx_set_property(item,"ItemContain",stresult)
	--设置打开标记
	nx_set_property(item,"Opened",1)
	
	nx_custom(self,SERVER_CUSTOMMSG_PICK_ITEM,nx_string(uid))--通知UI显示
end

