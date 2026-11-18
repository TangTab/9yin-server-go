--二级公会逻辑脚本
--能否创建公会
function can_create_guild(self, guild)
end

--能否解散公会
function can_disband_guild(self, guild)
end

--玩家上线
function on_player_recover(self)
end

--创建公会
function create_guild(self, guild)
end

--解散公会
function disband_guild(self, guild)
end

--加入公会
function join_guild(self, invitor, guild)
end

--离开公会
function leave_guild(self)
end

--开除成员
function dismiss_member(self, name)
end

--更换成员职位
function change_member_position(self, name, newpos)
end

--更换管理员
function change_commander(self, name)
end

--玩家验证公会信息回调
function on_check_playerinfo(self, result)
end

--创建公会回调
function on_create_guild(self, result, guild)
end

--解散公会回调
function on_disband_guild(self, result, guild)
end

--加入公会回调
function on_join_guild(self, result, invitor, guild)
end

--离开公会回调
function on_leave_guild(self, result, guild)
end

--开除成员回调
function on_dismiss_member(self, result, name)
end

--更换成员职位回调
function on_change_member_position(self, result, name, newpos)
end

--更换管理员回调
function on_change_commander(self, result, name)
end