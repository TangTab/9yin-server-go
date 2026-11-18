-----------------------------------
--  @ 名称: FreshmanHelp.lua
--  @ 功能: 玩家进入场景时的处理
--  @ 作者: wanglei
--  @ 时间: 2009年12月25日
-----------------------------------

local endStepID = "Freshman_End"
-- 拜师任务
local g_baishi_task_id = {
		["born04"] = 40 ,--千灯镇
		["born03"] = 49 ,--烟雨庄
		["city02"] = 50 ,--苏州

		};

-- 玩家进入场景时的处理, 参数1:player, 参数2: 所在场景名(Resource)
function on_ready(player, scene_name)

	local curStepID = nx_query_property(player, "HelpStepID")
	-- 说明玩家是新手
	if tostring(curStepID) == "" then

		if tostring(scene_name) == "born03" then-- 烟雨庄
			curStepID = "yanyuzhuang_00"
			nx_require_write_drama(player, "yyz01_7")
			nx_accept_task(player, 22001)
		elseif tostring(scene_name) == "city02" then-- 苏州
			curStepID = "suzhou_00"
			nx_require_write_drama(player, "sz01_21")
			nx_accept_task(player, 1)
		elseif tostring(scene_name) == "born04" then -- 千灯镇
			curStepID = "qiandengzhen_00"
			--nx_require_write_drama(player, "qdz01_8")
			nx_accept_task(player, 24001)
		else
			curStepID = endStepID
		end
		-- 玩家自动接到拜师任务
		local taskid = g_baishi_task_id[scene_name]
		if taskid ~= nil then
			nx_accept_task(player, taskid);
		end
		nx_set_property(player, "HelpStepID", curStepID)
	elseif tostring(curStepID) == "yanyuzhuang_00" then
	elseif tostring(curStepID) == "suzhou_00" then
	elseif tostring(curStepID) == "qiandengzhen_00" then
	end

end

-- 进入下一步的回调
function on_next_step(player, nextStepID)
	if nextStepID == "" then
		return
	end

	if nextStepID == "yanyuzhuang_02" then

	elseif nextStepID == "suzhou_02" then

	end

end
