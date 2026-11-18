-- 每一次触发造成更大的伤害
require("define")
require("share\\server_custom_define")

function OnInit(self, buffer)
	if not nx_add_data(buffer, "CustomTimer", VTYPE_INT) then
		nx_echo("添加自定义数据失败")
		return
	end

	nx_set_data(buffer, "CustomTimer", 1)
end

function OnTimer(self, buffer)
	local dmg = 20
	local mul = 1
	if nx_find_data(buffer, "CustomTimer") then
		mul = nx_query_data(buffer, "CustomTimer")
		nx_set_data(buffer, "CustomTimer", mul+1)
	end

	sender = nx_query_property(buffer, "Sender")
	nx_damage(sender, self, dmg*mul)
	nx_custom_byken(self, SERVER_CUSTOMMSG_EFFECT, self, "HP", dmg*mul)
end

