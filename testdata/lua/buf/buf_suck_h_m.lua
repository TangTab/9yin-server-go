--吸血&吸内buff,物理攻击时,恢复一定量hp&mp
--恢复比例
--hp: 0.08 * buffer等级
--mp: 0.05 * buffer等级

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
	local hp_healpoint = 0
	local mp_healpoint = 0
	
	if 0 == damagetype then
		hp_healpoint = damage * 0.08 * level
		mp_healpoint = damage * 0.05 * level
	end
	
	nx_custom_byken(self, SERVER_CUSTOMMSG_EFFECT, self, "Buffer_hit_effect", "buffhit_suck_h_m")
	nx_inc_property(self, "HP", hp_healpoint)
	nx_inc_property(self, "MP", mp_healpoint)

end

--状态被打断
function buffer_break(self, buffer)
end