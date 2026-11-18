--吸血buff,物理攻击时,恢复一定量hp
--恢复比例 = 0.1 * buffer等级

require("define")
require("command_define")
require("share\\server_custom_define")

--Buffer被激活
function buffer_active(self, buffer)
end

--Buffer未被激活
function buffer_deactive(self, buffer)
end

--计时或计次时的效果
--如果是计次，返回1时：本次计次有效，返回0时：本次不计次
function buffer_timer(self, buffer, sender, cmdid, target, damage, damagetype)
	if cmdid ~= COMMAND_DAMAGETARGET then
		return 0
	end
	
	local level = nx_query_property(buffer, "Level")
	local healpoint = 0

	if 0 == damagetype then
		healpoint = damage * 0.1 * level
	end
	
	nx_custom_byken(self, SERVER_CUSTOMMSG_EFFECT, self, "Buffer_hit_effect", "buffhit_suck_hp")
	nx_inc_property(self, "HP", nx_int(healpoint))
	
	--local hp = nx_query_property(self, "HP")
	--nx_msgbox(nx_string(hp))

end

--状态被打断
function buffer_break(self, buffer)
end