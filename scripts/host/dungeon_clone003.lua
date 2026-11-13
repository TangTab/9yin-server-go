-- 暮色之村（ini\scene\clone003）副本玩法脚本——Lua 接入试水。
-- 波次值域（creator.NpcProgress 实测）：2,3,4,5,7；7=末波。
-- 演示三种玩法形态：
--   1) 进本剧情提示（dungeon_enter）；
--   2) 默认波次引擎 + 每波肃清播报（dungeon_wave_cleared 返回 false）；
--   3) 波 3 肃清后接管节奏：ctx:after 延时 2.5s 再 ctx:spawn_wave
--      物化下一波（返回 true 抑制默认立即推进）。
-- 注意：game.on 每事件名单处理器——新增第二个副本脚本时先引入
-- 路由层（按 ctx:dungeon().config 分发），勿再直接注册同名事件。

local CONFIG = [[ini\scene\clone003]]

local function is_clone003(ctx)
	return ctx:dungeon().config == CONFIG
end

game.on("dungeon_enter", function(ctx)
	if not is_clone003(ctx) then
		return false
	end
	local player = ctx:player()
	ctx:notify(string.format("暮色四合，鸦声凄冷。%s 踏入暮色之村——纵火者正在村口徘徊。", player.name), "ui_shop")
	return true
end)

game.on("dungeon_npc_killed", function(ctx)
	if not is_clone003(ctx) then
		return false
	end
	local dungeon = ctx:dungeon()
	game.log("clone003 kill npc=%s wave=%d killed=%d", ctx:npc().config_id, dungeon.wave, dungeon.killed)
	return true
end)

game.on("dungeon_wave_cleared", function(ctx)
	if not is_clone003(ctx) then
		return false
	end
	local dungeon = ctx:dungeon()
	if dungeon.next_wave == 0 then
		ctx:notify("村中杀声止息，暮色之村已肃清。稍候片刻即送侠士回乡。", "ui_shop")
		return false -- 末波：默认收尾（10s 后回城销毁）
	end
	if dungeon.wave == 3 then
		-- 接管节奏演示：官差已清，稍候片刻再放下一波。
		ctx:notify("官差尸横就地。远处传来更沉的脚步声……", "ui_shop")
		game.log("clone003 wave %d cleared; spawn wave %d in 2.5s", dungeon.wave, dungeon.next_wave)
		ctx:after(2.5, "clone003_advance")
		return true -- 接管：默认引擎跳过，由 clone003_advance 物化下一波
	end
	ctx:notify(string.format("第 %d 波已肃清，敌人仍在逼近。", dungeon.wave), "ui_shop")
	return false -- 其余波：默认立即推进
end)

game.on("clone003_advance", function(ctx)
	if not is_clone003(ctx) then
		return false
	end
	local dungeon = ctx:dungeon()
	if dungeon.next_wave == 0 then
		return false
	end
	ctx:spawn_wave(dungeon.next_wave)
	return true
end)

game.on("dungeon_destroy", function(ctx)
	if not is_clone003(ctx) then
		return false
	end
	local dungeon = ctx:dungeon()
	game.log("clone003 instance=%s destroyed kills=%d", dungeon.instance, dungeon.killed)
	return true
end)
