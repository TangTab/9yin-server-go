require("share\\itemtype_define")
require("define")
require("share\\logicstate_define")
require("share\\item_static_data_define")
require("share\\client_custom_define")
require("share\\view_define")
require("share\\server_custom_define")

function on_use(self,eventype)
    	-- 保护
	if not nx_exists(self) then
		return 0
	end

	if eventype == CLIENT_CUSTOMMSG_PICK_ITEM then--宝箱事件
	   return pick_all(self)
	end
	return 0
end

--拾取礼包的全部item
function pick_all(self)    
 
    local item = nx_query_data(self,"pickitem")
   
    if nx_exists(item) then
     
        local itemcontain = nx_query_property(item,"ItemContain")
        local arr = nx_split_string(itemcontain, ";")
        local containeraval = nx_get_avalcontainer(self,VIEWPORT_TOOL)
        local toolcontainer = nx_get_container_fromconfigid(self,nx_widestr("ToolBox"))
       
        if table.getn(arr)-1 > containeraval then
            nx_sysinfo(self,TIPSTYPE_TIPSINFO,"BG_35015")
            return 0;
        end

        local a,configid,count
        --创建对象
        for i=1,table.getn(arr) do
            a = nx_split_string(arr[i], ",")
            configid = a[1]
            count = a[2]

            local item1 = nx_create_from_config(nx_get_scene(),nx_string(configid))--
            if nx_exists(item1) then
                if nx_number(count) > nx_query_property(item1,"MaxAmount") then
                    count = nx_query_property(item1,"MaxAmount") 
                end
                nx_set_property(item1,"Amount",nx_int(count))
                nx_add_ranpro(item1)
                nx_place(item1,toolcontainer) --放入背包之中
            end
        end
        	--保存结果
	    nx_set_property(item,"ItemContain","")
	   
	    local uid = nx_query_property(item,"UniqueID")
	    nx_custom(self,SERVER_CUSTOMMSG_PICK_ITEM,nx_string(uid))--通知UI显示
	    use_item_disappear(item)
    end
   --[[ ]]--
    
    return 0
end

--道具的使用消失问题
function use_item_disappear(item)
    nx_inc_property(item,"CurrentUseNumber",1)--使用次数+1
    local disappear = nx_query_property(item,"Disappear")
    if nx_number(disappear)>0 then--要消失的
        nx_inc_property(item, "Amount", -1)
    end
end
