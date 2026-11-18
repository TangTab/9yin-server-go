--眩晕buff,角色一定时间无法移动,进行攻击
--buff移除或结束后,角色恢复stand动作

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
function buffer_timer(self, buffer, sender)
end

--状态被打断
function buffer_break(self, buffer)
end

--AI模块接口,buff消失或移除时调用
function on_buffer_remove(buffer, self)
	local name = nx_query_property(buffer, "ConfigID")
	if name == "buf_dizzy" then
		nx_set_property(self, "State", stand)
	end
end