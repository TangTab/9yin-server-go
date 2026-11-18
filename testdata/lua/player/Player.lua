require("share\\server_custom_define")

-- 玩家就绪
function on_ready(self, first)
	if first > 0 then
		--第一次上线
		nx_custom_byken(self, SERVER_CUSTOMMSG_EFFECT, self, "BORN")
	end
	
	--切换场景后，打断切磋
	if nx_find_data(self, "fight_guid_start") then
		-- 移除玩家身上切磋标识
		nx_remove_data(self, "fight_guid_start")
		-- 当前技能停止闪烁
		nx_custom(self, nx_string("skill_icon_flash"), nx_int(1), nx_int(0))
	end
end

