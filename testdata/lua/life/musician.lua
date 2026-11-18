----------------------------------------------------------------------
-- 文件名:		musician.lua
-- 内  容:		琴师的琴谱使用脚本
-- 说  明:
-- 创建日期:	2009年11月9日
-- 创建人:		王征浩
-- 版权所有:	苏州蜗牛电子有限公司
----------------------------------------------------------------------
require("define")

--使用琴谱
function on_use(music, user)

	--检查是否可以使用
	if true ~= can_use(user, music) then
		return 0
	end
	
	nx_begin_music(user, music)
	return 0
end

--是否可以使用
function can_use(user, music)	

	local result = nx_can_music(user, music)
	
	return result
end

--添加buffer
function add_buffer(self)

	--玩家身上琴曲的临时数据
	if not nx_find_data(self, "Musician") then
		return false
	end

	local music = nx_query_data(self, "Musician")
	if not nx_exists(music) then
		return false
	end


	--获取buffer
	local bufferid = nx_query_property(music, "Para4")
	local buffer = nx_create_from_config(nx_get_scene(), nx_string(bufferid))
	if not nx_exists(buffer) then
		return false
	end

	--设置buffer
	nx_set_property(buffer, "EffectType", 1)
	nx_set_property(buffer, "LifeTime", 100000)
	if not nx_find_data(buffer, "LifeSkill") then
		nx_add_data(buffer, "LifeSkill", VTYPE_STRING)
	end
	nx_set_data(buffer, "LifeSkill", "Musician")
	
	nx_add_buffer(self, self, buffer)
	
	return true
end



