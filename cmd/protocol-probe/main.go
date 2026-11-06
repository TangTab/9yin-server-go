package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/qinggong"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/session"
	"github.com/local/9yin-go-server/internal/transport"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func main() {
	addr := flag.String("listen", "127.0.0.1:19061", "TCP listen address")
	allChengduNPCs := flag.Bool("npc-all-chengdu", true, "compatibility flag: load NPCs dynamically from the active modern scene")
	npcRadius := flag.Float64("npc-radius", 120, "NPC viewport radius in scene units")
	npcTablePath := flag.String("npc-table", defaultNPCTablePath, "modern client NPC template table")
	npcCreatorPath := flag.String("npc-creator", defaultNPCCreatorPath, "modern client scene NPC creator")
	npcConfigID := flag.String("npc-config", "Relivecity05B", "NPC template ID to materialize")
	npcPatrolCount := flag.Int("npc-patrol-count", 3, "nearby NPCs simulated by the low-rate server patrol loop (0 disables it)")
	demoDamage := flag.Int("demo-damage", 0, "test-only damage applied once after the player reaches the active scene (0 disables it)")
	starterSilver := flag.Int("starter-silver", 99999, "test-only starting CapitalType1 (碎银) for the active player")
	gmListen := flag.String("gm-listen", "127.0.0.1:19062", "local GM HTTP listen address (empty disables)")
	logFile := flag.String("log-file", runtimeProjectPath("artifacts", "local", "logs", "protocol-probe-live.log"), "append runtime log to this local file")
	npcFuncPath := flag.String("npc-func", defaultLegacyNPCFuncPath, "Npc_Func.xml used to build per-NPC service menus")
	switchScene := flag.String("switch-scene", "", "test-only live destination: config,resource,x,y,z,orient (empty disables it)")
	npcAuditAll := flag.Bool("npc-audit-all", false, "inventory every modern NPC creator scene and exit")
	npcServiceAuditScene := flag.String("npc-service-audit-scene", "", "write effective NPC service audit for config,resource,x,y,z,orient and exit")
	qingGongDefinePath := flag.String("qinggong-define", filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgdefine.ini"), "current-client QGDefine.ini used to validate and charge 216")
	flag.Parse()
	if *logFile != "" {
		if err := os.MkdirAll(filepath.Dir(*logFile), 0755); err != nil {
			log.Fatalf("create log directory: %v", err)
		}
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("open runtime log %s: %v", *logFile, err)
		}
		defer file.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, file))
	}
	var switchDestination *sceneDestination
	if *switchScene != "" {
		parsed, parseErr := parseSceneDestination(*switchScene)
		if parseErr != nil {
			log.Fatalf("parse -switch-scene: %v", parseErr)
		}
		switchDestination = &parsed
	}
	npcSchema := clientdata.VisibleNPCModernV1()
	qingGongCatalog, qingGongErr := qinggong.Load(*qingGongDefinePath)
	if qingGongErr != nil {
		log.Fatalf("load current-client qinggong definitions: %v", qingGongErr)
	}
	log.Printf("qinggong catalog ready definitions=%d path=%s", qingGongCatalog.Count(), *qingGongDefinePath)
	npcFuncs, npcFuncErr := npcfunc.Load(*npcFuncPath)
	if npcFuncErr != nil {
		log.Printf("Npc_Func menu registry unavailable (%s): %v; NPCs without this table will expose no fabricated service menu", *npcFuncPath, npcFuncErr)
	} else {
		log.Printf("Npc_Func menu registry ready configured_npcs=%d path=%s", npcFuncs.Count(), *npcFuncPath)
	}
	if *npcAuditAll {
		entries, auditErr := auditNPCSceneRoot(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir), npcSchema)
		if auditErr != nil {
			log.Fatal(auditErr)
		}
		payload, marshalErr := json.MarshalIndent(entries, "", "  ")
		if marshalErr != nil {
			log.Fatal(marshalErr)
		}
		output := runtimeProjectPath("artifacts", "audit", "npc-scene-audit.json")
		if writeErr := os.MkdirAll(filepath.Dir(output), 0755); writeErr != nil {
			log.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(output, append(payload, '\n'), 0644); writeErr != nil {
			log.Fatal(writeErr)
		}
		var creators, resolved, unresolved, failed int
		for _, entry := range entries {
			creators += entry.CreatorInstances
			resolved += entry.Resolved
			unresolved += entry.Unresolved
			if entry.Error != "" {
				failed++
			}
		}
		log.Printf("NPC full audit scenes=%d creator_instances=%d resolved=%d unresolved=%d failed_scenes=%d output=%s", len(entries), creators, resolved, unresolved, failed, output)
		return
	}
	var staticNPCs []npcSpawn
	var sceneRegistry *sceneNPCRegistry
	var err error
	if *allChengduNPCs {
		sceneRegistry, err = newSceneNPCRegistry(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir), npcSchema)
		if err == nil {
			log.Printf("modern scene NPC registry ready template_tables=%d creator_root=%s", len(sceneRegistry.tables), sceneRegistry.root)
		}
	} else {
		staticNPCs, err = loadNPCSpawns(*npcTablePath, *npcCreatorPath, *npcConfigID, npcSchema)
	}
	if err != nil {
		log.Fatalf("load modern NPC catalog: %v", err)
	}
	if *npcServiceAuditScene != "" {
		if sceneRegistry == nil {
			log.Fatal("npc service audit requires dynamic modern scene registry")
		}
		destination, parseErr := parseSceneDestination(*npcServiceAuditScene)
		if parseErr != nil {
			log.Fatalf("parse -npc-service-audit-scene: %v", parseErr)
		}
		scene := normalizeClientScene(destination.location.Scene)
		catalog, stats, auditErr := sceneRegistry.catalogFor(scene)
		if auditErr != nil {
			log.Fatal(auditErr)
		}
		report := buildNPCServiceAudit(scene, stats, catalog, npcFuncs)
		payload, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			log.Fatal(marshalErr)
		}
		output := runtimeProjectPath("artifacts", "audit", "npc-service-audit-"+auditFileComponent(scene.Resource)+".json")
		if writeErr := os.MkdirAll(filepath.Dir(output), 0755); writeErr != nil {
			log.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(output, append(payload, '\n'), 0644); writeErr != nil {
			log.Fatal(writeErr)
		}
		log.Printf("NPC service audit scene=%s resource=%s NPCs=%d resolved=%d output=%s", scene.Config, scene.Resource, len(report.NPCs), stats.Resolved, output)
		return
	}
	for _, npc := range staticNPCs {
		_, propertyErr := npc.resolved.OrderedProperties(npcSchema)
		if propertyErr != nil {
			log.Fatalf("validate modern NPC %s: %v", npc.resolved.ConfigID, propertyErr)
		}
	}
	if sceneRegistry == nil {
		log.Printf("static modern NPC catalog ready objects=%d viewport_radius=%.1f", len(staticNPCs), *npcRadius)
	}
	store, err := openRoleStore()
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	facultyStore, err := openFacultyProgressStore()
	if err != nil {
		log.Fatal(err)
	}
	gm := newGMHub()
	if *gmListen != "" {
		go serveGM(*gmListen, gm)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("protocol probe listening on %s", *addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn, store, facultyStore, npcSchema, sceneRegistry, staticNPCs, npcFuncs, qingGongCatalog, float32(*npcRadius), *npcPatrolCount, int32(*demoDamage), int32(*starterSilver), switchDestination, gm)
	}
}

func handle(conn net.Conn, store *roleStore, facultyStore *facultyProgressStore, npcSchema clientdata.Schema, sceneRegistry *sceneNPCRegistry, staticNPCs []npcSpawn, npcFuncs *npcfunc.Registry, qingGongCatalog *qinggong.Catalog, npcRadius float32, npcPatrolCount int, demoDamage, starterSilver int32, switchDestination *sceneDestination, gm *gmHub) {
	config := transport.DefaultConnectionConfig()
	config.MaxFrameSize = 64 << 10
	link, err := transport.NewConnection(conn, config)
	if err != nil {
		log.Printf("%s: initialize transport: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}
	defer link.Close()
	machine := session.New()
	var sceneNPCs []npcSpawn
	log.Printf("session=%d remote=%s opened phase=%s", machine.ID(), conn.RemoteAddr(), machine.Phase())
	defer func() {
		machine.Close()
		log.Printf("session=%d remote=%s closed phase=%s", machine.ID(), conn.RemoteAddr(), machine.Phase())
	}()
	world := newSceneLifecycle(conn.RemoteAddr().String(), link, npcFuncs)
	defer world.close()
	ctx := context.Background()
	var accountKey role.AccountKey
	var account role.Account
	var selected *role.RoleSnapshot
	var player *playerActor
	var runtime *sceneRuntime
	var gmSession *gmSession
	persistFaculty := func() {
		if facultyStore == nil || selected == nil || player == nil {
			return
		}
		if err := facultyStore.Save(selected.ID, player.facultySnapshot()); err != nil {
			log.Printf("%s: persist faculty progress: %v", conn.RemoteAddr(), err)
		}
	}
	defer persistFaculty()
	qingGongGranted := false
	neigongGranted := false
	grantStarterProgress := func() error {
		if player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		frames, err := player.progressViewFrames()
		if err != nil {
			return fmt.Errorf("encode View=43: %w", err)
		}
		for _, frame := range frames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=43: %w", err)
			}
		}
		skillFrames, err := starterSkillViewFrames()
		if err != nil {
			return fmt.Errorf("encode View=40: %w", err)
		}
		for _, frame := range skillFrames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=40: %w", err)
			}
		}
		if granted, err := grantHeartBuddhaPalmShortcuts(link, player); err != nil {
			return fmt.Errorf("grant heart buddha palm shortcuts: %w", err)
		} else if granted > 0 {
			log.Printf("%s: granted %d 心佛掌 shortcuts numeric_page_2=[%d..%d]", conn.RemoteAddr(), granted, heartBuddhaPalmShortcutBaseIndex, heartBuddhaPalmShortcutBaseIndex+int32(len(heartBuddhaPalmShortcutIDs))-1)
		}
		if granted, err := grantTaiJiQuanGuPuShortcuts(link, player); err != nil {
			return fmt.Errorf("grant tai ji quan gu pu shortcuts: %w", err)
		} else if granted > 0 {
			log.Printf("%s: granted %d 太极拳(古谱) shortcuts numeric_page_3=[%d..%d]", conn.RemoteAddr(), granted, taiJiQuanGuPuShortcutBaseIndex, taiJiQuanGuPuShortcutBaseIndex+int32(len(taiJiQuanGuPuShortcutIDs))-1)
		}
		state, err := player.vitalUpdate()
		if err != nil {
			return fmt.Errorf("encode inner-power/player attributes: %w", err)
		}
		if err := link.WriteFrame(state); err != nil {
			return fmt.Errorf("write inner-power/player attributes: %w", err)
		}
		neigongGranted = true
		return nil
	}
	grantSceneRedSuperArmor := func() error {
		if player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		expires, err := grantRedSuperArmor(link, player)
		if err != nil {
			return err
		}
		scheduleRedSuperArmorExpiry(link, player, expires)
		return nil
	}
	defer func() {
		if gmSession != nil {
			gm.detach(gmSession.id)
		}
	}()
	// Only the one-byte ClientReady sent by stage_main after the original
	// player archive has completed is a safe boundary for a live 0x0C test.
	// Earlier 0x0A packets are normal client custom/activity traffic and can
	// arrive while the initial ServerAddObject is still being processed.
	explicitSceneReady := false
	activeNPCs := make(map[int]struct{})
	lastPositionSave := time.Time{}
	var pendingLocation *role.Location
	defer func() {
		if pendingLocation != nil && selected != nil {
			if err := store.saveLocation(ctx, selected, *pendingLocation); err != nil {
				log.Printf("%s: persist final position: %v", conn.RemoteAddr(), err)
			}
		}
	}()
	processGMCommand := func(command gmCommand) (commandErr error) {
		defer func() { command.complete(commandErr) }()
		if gmSession == nil || player == nil {
			return fmt.Errorf("character has not entered a scene")
		}
		switch command.action {
		case "set_silver":
			player.setSilver(command.silver)
			update, updateErr := player.silverUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("碎银设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSilver(command.silver)
			log.Printf("%s: GM set CapitalType1=%d", conn.RemoteAddr(), command.silver)
		case "set_sp", "add_sp", "fill_sp":
			var sp int32
			var update []byte
			var updateErr error
			switch command.action {
			case "set_sp":
				sp, update, updateErr = player.setSP(command.sp)
			case "add_sp":
				sp, update, updateErr = player.addSP(command.sp)
			default:
				sp, update, updateErr = player.setSP(maxPlayerSP)
			}
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("怒气设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSP(sp, fmt.Sprintf("怒气设为 %d/%d", sp, maxPlayerSP))
			log.Printf("%s: GM %s SP=%d", conn.RemoteAddr(), command.action, sp)
		case "set_move_speed":
			update, updateErr := player.setMoveSpeed(command.value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("移动速度设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("移动速度设为 %.2f", command.value))
			log.Printf("%s: GM set MoveSpeed/RunSpeed=%.2f", conn.RemoteAddr(), command.value)
		case "set_gravity":
			update, updateErr := player.setGravity(command.value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("下落重力设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("下落重力设为 %.2f", command.value))
			log.Printf("%s: GM set Gravity=%.2f", conn.RemoteAddr(), command.value)
		case "set_faction":
			preset, presetErr := gmFactionPresetByKey(command.faction)
			if presetErr != nil {
				return presetErr
			}
			if saveErr := store.saveFaction(ctx, selected, preset.Faction); saveErr != nil {
				gmSession.setAction("门派保存失败：" + saveErr.Error())
				return saveErr
			}
			update, updateErr := player.setFaction(preset.Faction)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("门派同步失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setFaction(preset.Faction, preset.Label)
			log.Printf("%s: GM set School=%s (%s)", conn.RemoteAddr(), preset.Faction, preset.Label)
		case "switch_scene":
			if runtime == nil || selected == nil {
				gmSession.setAction("切图失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if switchErr := runtime.SwitchScene(command.destination); switchErr != nil {
				gmSession.setAction("切图失败：" + switchErr.Error())
				return switchErr
			}
			sceneNPCs = runtime.npcCatalog
			pendingLocation = &runtime.activeRole.Location
			if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
				log.Printf("%s: persist GM scene switch: %v", conn.RemoteAddr(), saveErr)
			} else {
				pendingLocation = nil
			}
			gmSession.setLocation(runtime.activeRole.Location, "已发送 GM 在线切图，等待目标场景 Ready")
			log.Printf("%s: GM sent live 0x0C re-entry to scene=%s resource=%s", conn.RemoteAddr(), runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
		case "teleport":
			if runtime == nil || selected == nil {
				gmSession.setAction("场景内瞬移失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if teleportErr := runtime.TeleportWithinScene(command.position); teleportErr != nil {
				gmSession.setAction("场景内瞬移失败：" + teleportErr.Error())
				return teleportErr
			}
			pendingLocation = &runtime.activeRole.Location
			if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
				log.Printf("%s: persist GM same-scene teleport: %v", conn.RemoteAddr(), saveErr)
			} else {
				pendingLocation = nil
			}
			gmSession.setLocation(runtime.activeRole.Location, "已发送场景内瞬移（0x1F）")
			log.Printf("%s: GM sent same-scene ServerLocation x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), command.position.X, command.position.Y, command.position.Z, command.position.Orient)
		case "npc_bubble":
			ident, bubbleErr := world.sendGMNPCBubble("ui_shop")
			if bubbleErr != nil {
				gmSession.setAction("NPC 气泡测试失败：" + bubbleErr.Error())
				return bubbleErr
			}
			gmSession.setAction("已发送 NPC 气泡：" + ident)
			log.Printf("%s: GM sent custom %d NPC bubble target=%s", conn.RemoteAddr(), customNPCTalk, ident)
		case "drama_prompt":
			if dramaErr := world.sendDramaPrompt(gmDramaProbe()); dramaErr != nil {
				gmSession.setAction("章节提示卡测试失败：" + dramaErr.Error())
				return dramaErr
			}
			gmSession.setAction("已发送章节提示卡（协议测试）")
			log.Printf("%s: GM sent custom %d drama prompt", conn.RemoteAddr(), customBeginDrama)
		case "red_super_armor":
			if selected == nil {
				gmSession.setAction("红霸体发送失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			if armorErr := grantSceneRedSuperArmor(); armorErr != nil {
				gmSession.setAction("红霸体发送失败：" + armorErr.Error())
				return armorErr
			}
			gmSession.setAction("已发送红霸体（20 秒）")
			log.Printf("%s: GM sent BufferInfo1 red super armor for %s", conn.RemoteAddr(), selected.Name)
		case "action_state_fishing":
			frame, frameErr := playerActionStateFrame("interact268")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("State 钓鱼动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送 interact268")
			log.Printf("%s: GM action probe route=State action=interact268", conn.RemoteAddr())
		case "action_state_hsqs05":
			frame, frameErr := playerActionStateFrame("hsqs_05_h")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("State 花神动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送 hsqs_05_h")
			log.Printf("%s: GM action probe route=State action=hsqs_05_h", conn.RemoteAddr())
		case "action_state_sgmd":
			frame, frameErr := playerActionStateFrame("hsqs_07")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("神鬼莫敌动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 发送神鬼莫敌 hsqs_07")
			log.Printf("%s: GM action probe route=State skill=CS_yhwq_hsqs07 action=hsqs_07 frames=147", conn.RemoteAddr())
		case "action_native_fishing":
			frame, frameErr := nativeGenericActionFrame("interact268")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("action 钓鱼动作发送失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过字符串 action 发送 interact268")
			log.Printf("%s: GM action probe route=string-action action=interact268", conn.RemoteAddr())
		case "action_state_stop":
			frame, frameErr := playerActionStateFrame("stand")
			if frameErr == nil {
				frameErr = link.WriteFrame(frame)
			}
			if frameErr != nil {
				gmSession.setAction("恢复待机失败：" + frameErr.Error())
				return frameErr
			}
			gmSession.setAction("已通过 State 恢复 stand")
			log.Printf("%s: GM action probe route=State action=stand", conn.RemoteAddr())
		default:
			return fmt.Errorf("unsupported GM action %q", command.action)
		}
		return nil
	}
	// Keep exactly one goroutine in ReadFrame, then multiplex its completed
	// frames with GM commands below.  Reading directly in the main loop made a
	// quiet client block GM commands indefinitely until it next moved or sent a
	// heartbeat.  A read deadline is deliberately not used here: timing out in
	// the middle of an escaped transport frame could discard its partial body.
	type inboundFrame struct {
		plain []byte
		err   error
	}
	inbound := make(chan inboundFrame, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			plain, readErr := link.ReadFrame()
			select {
			case inbound <- inboundFrame{plain: plain, err: readErr}:
			case <-done:
				return
			}
			if readErr != nil {
				return
			}
		}
	}()
	facultyTicker := time.NewTicker(time.Second)
	defer facultyTicker.Stop()
	for {
		var commandQueue <-chan gmCommand
		if gmSession != nil {
			commandQueue = gmSession.commands
		}
		var plain []byte
		var readErr error
		select {
		case command := <-commandQueue:
			_ = processGMCommand(command)
			continue
		case now := <-facultyTicker.C:
			if player == nil || selected == nil {
				continue
			}
			slot, advanced := player.facultyTick(now)
			innerPowerPassiveExpired := player.expireInnerPowerPassive(now)
			sitcrossAdvanced := player.sitcrossTick()
			combatBruiseAdvanced := player.combatBruiseTick()
			combatAdvanced := false
			if runtime != nil {
				combat, combatErr := world.combatTick(player, runtime.activeRole.Location.Position, now)
				if combatErr != nil {
					log.Printf("%s: run NPC combat tick: %v", conn.RemoteAddr(), combatErr)
					return
				}
				combatAdvanced = combat.playerStateChanged
				if combat.playerRevived {
					log.Printf("%s: auto-revived player after NPC defeat", conn.RemoteAddr())
				}
			}
			var tickErr error
			if advanced {
				var frame []byte
				frame, tickErr = player.progressBookFrame(slot)
				if tickErr == nil {
					tickErr = link.WriteFrame(frame)
				}
			}
			if tickErr == nil && (advanced || sitcrossAdvanced || combatBruiseAdvanced || combatAdvanced || innerPowerPassiveExpired) {
				var state []byte
				state, tickErr = player.vitalUpdate()
				if tickErr == nil {
					tickErr = link.WriteFrame(state)
				}
			}
			if tickErr != nil {
				log.Printf("%s: publish role timer state: %v", conn.RemoteAddr(), tickErr)
				return
			}
			if advanced {
				persistFaculty()
				log.Printf("%s: settled inner cultivation role=%d slot=%d", conn.RemoteAddr(), selected.ID, slot)
			}
			if sitcrossAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: settled sitcross HP=%d/%d MP=%d/%d", conn.RemoteAddr(), state.HP, state.MaxHP, state.MP, state.MaxMP)
			}
			if combatBruiseAdvanced {
				state := player.actor.Snapshot()
				log.Printf("%s: settled combat bruised blood HP=%d HitHP=%d", conn.RemoteAddr(), state.HP, state.HitHP)
			}
			continue
		case received := <-inbound:
			plain, readErr = received.plain, received.err
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("%s: %v", conn.RemoteAddr(), readErr)
			}
			return
		}
		if len(plain) == 0 {
			log.Printf("%s: empty frame", conn.RemoteAddr())
			continue
		}
		if err := machine.ValidateMessage(plain); err != nil {
			log.Printf("%s: reject client message: %v", conn.RemoteAddr(), err)
			return
		}
		if plain[0] == session.ClientLogin {
			identity, err := auth.ParseLoginAccountIdentity(plain)
			if err != nil {
				log.Printf("session=%d remote=%s reject login structure: %v", machine.ID(), conn.RemoteAddr(), err)
				return
			}
			accountKey = role.AccountKey(identity.String())
			log.Printf("session=%d remote=%s opcode=0x02 decoded_len=%d account=%s", machine.ID(), conn.RemoteAddr(), len(plain), identity.Prefix())
		} else {
			fmt.Printf("remote=%s opcode=0x%02X decoded_len=%d\n%s\n", conn.RemoteAddr(), plain[0], len(plain), hex.Dump(plain))
		}
		if plain[0] == 0x02 {
			var err error
			account, selected, err = store.login(ctx, accountKey)
			if err != nil {
				log.Printf("%s: load account role: %v", conn.RemoteAddr(), err)
				return
			}
			exists := selected != nil
			payload := noRoleLoginSuccess(time.Now())
			if exists {
				payload = oneRoleLoginSuccess(time.Now(), selected.Name, selected.Appearance.Values)
			}
			if err := link.WriteFrame(payload); err != nil {
				log.Printf("%s: write no-role login success: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.LoginCompleted(exists); err != nil {
				log.Printf("%s: complete login transition: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerPlayerRoles(opcode=0x04, role_exists=%t)", conn.RemoteAddr(), exists)
		} else if plain[0] == 0x03 {
			// ClientMsgWorldInfo. The old world server replies with opcode
			// 0x05, uint16 info type, and a NUL-terminated UCS-2 string.
			if err := link.WriteFrame(worldInfo(0, "0,book1,0,book2,0,book4,0,book5,0,book8,0,0")); err != nil {
				log.Printf("%s: write empty world info: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerWorldInfo with five open books", conn.RemoteAddr())
		} else if plain[0] == 0x05 {
			name := createRoleName(plain)
			if name == "" {
				name = "本地角色"
			}
			appearance := createRoleStrings(plain)
			var err error
			selected, err = store.create(ctx, account, name, appearance)
			if err != nil {
				log.Printf("%s: persist created role: %v", conn.RemoteAddr(), err)
				return
			}
			if err := link.WriteFrame(oneRoleLoginSuccess(time.Now(), name, selected.Appearance.Values)); err != nil {
				log.Printf("%s: write created role list: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.RoleCreated(); err != nil {
				log.Printf("%s: complete create-role transition: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent ServerPlayerRoles with created role %q", conn.RemoteAddr(), name)
		} else if plain[0] == 0x04 {
			if selected == nil {
				log.Printf("%s: choose role without an account-owned role", conn.RemoteAddr())
				return
			}
			activeRole := selected
			activeRole.Location.Scene = normalizeClientScene(activeRole.Location.Scene)
			if sceneRegistry != nil {
				var stats npcCatalogStats
				sceneNPCs, stats, err = sceneRegistry.catalogFor(activeRole.Location.Scene)
				if err != nil {
					log.Printf("%s: resolve NPC scene catalog: %v", conn.RemoteAddr(), err)
					return
				}
				log.Printf("%s: active NPC scene=%s resource=%s creators=%d resolved=%d unresolved=%d ambiguous=%d", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, stats.CreatorInstances, stats.Resolved, stats.Unresolved, stats.Ambiguous)
			} else {
				sceneNPCs = staticNPCs
			}
			visual := resolveRoleVisual(activeRole.Appearance.Values)
			player = newPlayerActor(activeRole.Name, starterSilver)
			player.faction = roleFaction(activeRole.Appearance.Values)
			if saved, exists := facultyStore.Load(activeRole.ID); exists {
				player.restoreFaculty(saved, time.Now())
				log.Printf("%s: restored persistent faculty role=%d state=%d style=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyState, player.progress.facultyStyle, player.progress.facultyName)
			} else {
				persistFaculty()
				log.Printf("%s: initialized persistent normal faculty role=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyName)
			}
			// The scene server must define the visible-property and visible-record
			// schemas before it sends any object archives. CClassSet::InitTableMessage
			// in the reference server builds these as opcodes 0x09 and 0x0A.
			// Without 0x09 the client has no authoritative index/type mapping and
			// object creation fails at varying coordinates while decoding attrs.
			propertyTable := visiblePropertyTable(sceneVisiblePropertyFields(npcSchema.Fields))
			if err := link.WriteFrame(propertyTable); err != nil {
				log.Printf("%s: write property table: %v", conn.RemoteAddr(), err)
				return
			}
			// QingGongRec is a player record, not a visible object property.  The
			// current client refuses to treat a lightness skill as learned until
			// this schema has been negotiated, even when its energy properties
			// (QingGongPoint/MaxQingGongPoint) are already present.
			recordTable, recordErr := serverRecordTable(qingGongRecordSchemas)
			if recordErr != nil {
				log.Printf("%s: encode record table: %v", conn.RemoteAddr(), recordErr)
				return
			}
			if err := link.WriteFrame(recordTable); err != nil {
				log.Printf("%s: write record table: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent property table (%d entries) and record table (%d entries, QingGongRec)", conn.RemoteAddr(), binary.LittleEndian.Uint16(propertyTable[1:]), len(qingGongRecordSchemas))
			for _, view := range starterBagViews() {
				if err := link.WriteFrame(serverCreateView(view)); err != nil {
					log.Printf("%s: create starter bag view id=%d: %v", conn.RemoteAddr(), view.ID, err)
					return
				}
			}
			log.Printf("%s: created %d empty starter bag views", conn.RemoteAddr(), len(starterBagViews()))

			if err := world.begin(activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource); err != nil {
				log.Printf("%s: begin scene generation: %v", conn.RemoteAddr(), err)
				return
			}
			if err := sendEntryScene(link, activeRole.Location.Scene); err != nil {
				log.Printf("%s: write entry scene: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent PlayerEntry opcode=0x0B object=1 scene=%s resource=%s", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)

			if err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {
				log.Printf("%s: write player spawn chain: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent player 0x0D + 0x1F + authoritative vital 0x10", conn.RemoteAddr())
			// Register the initial nearby slice of the complete scene catalog now,
			// but do not put it on the wire until the client readiness barrier.
			// Loading a scene clears the client's native object table. Objects sent
			// before ClientReady therefore flash once and are immediately removed.
			clear(activeNPCs)
			nearbyCount, _, err := synchronizeNPCViewport(world, sceneNPCs, activeNPCs, activeRole.Location.Position, npcRadius)
			if err != nil {
				log.Printf("%s: register scene NPC viewport: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.RoleChosen(); err != nil {
				log.Printf("%s: complete choose-role transition: %v", conn.RemoteAddr(), err)
				return
			}
			runtime = &sceneRuntime{
				conn: link, world: world, machine: machine, registry: sceneRegistry,
				staticNPCs: staticNPCs, radius: npcRadius, activeNPCs: activeNPCs,
				player: player, activeRole: activeRole, npcCatalog: sceneNPCs,
			}
			gmSession = gm.attach(machine.ID(), conn.RemoteAddr().String(), activeRole, roleFaction(activeRole.Appearance.Values), starterSilver, player.skillSP())
			log.Printf("%s: registered %d nearby NPC objects from catalog=%d; waiting for ClientReady", conn.RemoteAddr(), nearbyCount, len(sceneNPCs))

		} else if plain[0] == 0x09 {
			// A live 0x0C scene replacement produces an early 0x0A while the
			// old stage is still unwinding.  The later 0x09 is the first point
			// at which the target scene's native object table survives writes.
			// Materialize the deferred player/NPC generation here, rather than
			// treating that earlier activity packet as readiness.
			if runtime != nil && runtime.awaitingStableReentryReady {
				if err := world.clientReady(); err != nil {
					log.Printf("%s: arm stable re-entry barrier: %v", conn.RemoteAddr(), err)
					return
				}
				// Target stage_main requires its 1-1 player object before it opens
				// the scene, so 0x0D was already sent after EntryScene. Delay only
				// 0x1F ServerLocation (the native visual move) until this target
				// ClientReady boundary.
				if err := sendPlayerLocationAndVitals(link, runtime.player, runtime.activeRole.Location.Position); err != nil {
					log.Printf("%s: write deferred player location/vitals: %v", conn.RemoteAddr(), err)
					return
				}
				log.Printf("%s: target ClientReady; sent deferred player 0x1F + vital state", conn.RemoteAddr())
				if err := machine.Ready(); err != nil {
					log.Printf("%s: complete stable re-entry ready: %v", conn.RemoteAddr(), err)
					return
				}
				if err := world.clientActivity(); err != nil {
					log.Printf("%s: activate stable re-entry objects: %v", conn.RemoteAddr(), err)
					return
				}
				if err := machine.Activity(); err != nil {
					log.Printf("%s: complete stable re-entry activity: %v", conn.RemoteAddr(), err)
					return
				}
				// ServerExitScene clears the native player object, including its
				// attached records and View 46 contents. The schemas remain
				// connection-scoped, but the rows must be restored for the newly
				// created 1-1 player object after every completed 0x0C re-entry.
				if player != nil {
					if grantErr := grantStarterQingGong(link); grantErr != nil {
						log.Printf("%s: restore qinggong after stable re-entry: %v", conn.RemoteAddr(), grantErr)
						return
					}
					if shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {
						log.Printf("%s: restore shortcuts after stable re-entry: %v", conn.RemoteAddr(), shortcutErr)
						return
					}
					qingGongGranted = true
					log.Printf("%s: restored %d QingGongRec rows and %d View=46 items after 0x0C re-entry", conn.RemoteAddr(), len(starterQingGongIDs), len(starterQingGongIDs)+len(starterQingGongSkillIDs))
					// ServerExitScene removes View 43 together with the player object.
					// neigongGranted only describes the old scene, so it must never
					// suppress this restore for a new native player object. Without
					// the View 43 entries faculty.lua can receive 1006/1 but cannot
					// resolve FacultyName to a wuxue object, and consequently leaves
					// the normal-begin button disabled (no 1006/11 is emitted).
					if progressErr := grantStarterProgress(); progressErr != nil {
						log.Printf("%s: restore inner-power/attributes after 0x0C re-entry: %v", conn.RemoteAddr(), progressErr)
						return
					}
					log.Printf("%s: restored View=43 inner-power and View=40 花无缺七招 after 0x0C re-entry", conn.RemoteAddr())
					if switchErr := enableSkillActionSwitch(link); switchErr != nil {
						log.Printf("%s: restore native skill-action switch after 0x0C re-entry: %v", conn.RemoteAddr(), switchErr)
						return
					}
					log.Printf("%s: enabled native skill-action switch=%d via modern S2C 0x27 message=%d after re-entry", conn.RemoteAddr(), skillActionInputSwitch, serverSwitchControlMessage)
				}
				if npcPatrolCount > 0 {
					world.startPatrol(npcPatrolCount)
				}
				runtime.awaitingStableReentryReady = false
				log.Printf("%s: target scene confirmed by 0x09; activated deferred target NPCs", conn.RemoteAddr())
				continue
			}
			explicitSceneReady = true
			log.Printf("%s: observed explicit stage_main ClientReady (0x09)", conn.RemoteAddr())
			// On the current client the initial 0x0A arrives before the native
			// stage finishes constructing and clears learned-skill state later.
			// Only the explicit stage_main 0x09 is safe for QingGongRec/View=46.
			if player != nil && !qingGongGranted {
				if grantErr := grantStarterQingGong(link); grantErr != nil {
					log.Printf("%s: grant stable starter qinggong: %v", conn.RemoteAddr(), grantErr)
					return
				}
				qingGongGranted = true
				log.Printf("%s: granted %d QingGongRec rows and %d View=46 items after explicit ClientReady", conn.RemoteAddr(), len(starterQingGongIDs), len(starterQingGongIDs)+len(starterQingGongSkillIDs))
			}
			if player != nil {
				if granted, shortcutErr := grantSitcrossShortcut(link, player); shortcutErr != nil {
					log.Printf("%s: grant native sitcross shortcut: %v", conn.RemoteAddr(), shortcutErr)
					return
				} else if granted {
					log.Printf("%s: granted native sitcross shortcut index=%d kind=%s id=%s", conn.RemoteAddr(), sitcrossShortcutIndex, sitcrossShortcutKind, sitcrossShortcutID)
				}
			}
			if player != nil && !neigongGranted {
				if progressErr := grantStarterProgress(); progressErr != nil {
					log.Printf("%s: grant stable inner-power/player attributes: %v", conn.RemoteAddr(), progressErr)
					return
				}
				log.Printf("%s: granted View=43 inner-power and View=40 花无缺七招 after explicit ClientReady", conn.RemoteAddr())
			}
			if player != nil {
				if switchErr := enableSkillActionSwitch(link); switchErr != nil {
					log.Printf("%s: enable native skill-action switch: %v", conn.RemoteAddr(), switchErr)
					return
				}
				log.Printf("%s: enabled native skill-action switch=%d via modern S2C 0x27 message=%d", conn.RemoteAddr(), skillActionInputSwitch, serverSwitchControlMessage)
			}
			// Current retail traffic can repeat ClientReady after a scene has
			// already become active (for example after a late UI/stage callback).
			// It is not a request to create a new scene generation; rejecting it
			// tears down an otherwise healthy connection.
			if machine.Phase() == session.PhaseInWorld {
				log.Printf("%s: ignored duplicate ClientReady in active scene", conn.RemoteAddr())
				continue
			}
			// ClientReady is the boundary at which the native scene and its
			// object manager are stable. Reconcile the player's viewport here.
			if err := world.clientReady(); err != nil {
				log.Printf("%s: reconcile ready viewport: %v", conn.RemoteAddr(), err)
				return
			}
			if err := machine.Ready(); err != nil {
				log.Printf("%s: complete client-ready transition: %v", conn.RemoteAddr(), err)
				return
			}
			if player != nil {
				vitals, syncErr := player.vitalUpdate()
				if syncErr != nil {
					log.Printf("%s: encode player ready vital sync: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if writeErr := link.WriteFrame(vitals); writeErr != nil {
					log.Printf("%s: write player ready vital sync: %v", conn.RemoteAddr(), writeErr)
					return
				}
			}
		} else if plain[0] == 0x0A {
			if runtime != nil && runtime.awaitingStableReentryReady {
				// This packet is emitted just after 0x0C/0x0B, before the target
				// terrain finishes. It belongs to the old stage and must not open
				// the new scene's object barrier.
				log.Printf("%s: ignored early 0x0A while target scene is loading", conn.RemoteAddr())
				continue
			}
			// The current retail client sometimes omits the one-byte 0x09 that
			// older captures called ClientReady and sends its first 0x0A directly
			// after the entry chain. Treat that observed packet as the explicit
			// compatibility barrier, rather than attempting a re-entry while the
			// session is still PhaseEnteringWorld.
			if machine.Phase() == session.PhaseEnteringWorld {
				if err := world.clientReady(); err != nil {
					log.Printf("%s: arm compatibility ready barrier: %v", conn.RemoteAddr(), err)
					return
				}
				if err := machine.Ready(); err != nil {
					log.Printf("%s: complete compatibility ready transition: %v", conn.RemoteAddr(), err)
					return
				}
				if player != nil {
					vitals, syncErr := player.vitalUpdate()
					if syncErr != nil {
						log.Printf("%s: encode compatibility ready vital sync: %v", conn.RemoteAddr(), syncErr)
						return
					}
					if writeErr := link.WriteFrame(vitals); writeErr != nil {
						log.Printf("%s: write compatibility ready vital sync: %v", conn.RemoteAddr(), writeErr)
						return
					}
					if switchErr := enableSkillActionSwitch(link); switchErr != nil {
						log.Printf("%s: enable compatibility native skill-action switch: %v", conn.RemoteAddr(), switchErr)
						return
					}
					log.Printf("%s: enabled native skill-action switch=%d via compatibility ClientReady", conn.RemoteAddr(), skillActionInputSwitch)
				}
				log.Printf("%s: promoted observed 0x0A to compatibility ClientReady", conn.RemoteAddr())
			}
			// A requested live transition takes priority over materializing the
			// old scene.  Sending hundreds of old NPC objects immediately before
			// ExitScene both wastes bandwidth and races the native teardown.
			// Never start a live scene switch from the compatibility 0x0A.
			// The concrete stage_main ClientReady above is emitted only after
			// the current scene's player/archive processing has settled.
			switchAfterActivity := switchDestination != nil && runtime != nil && explicitSceneReady && (machine.Phase() == session.PhaseAwaitingActivity || machine.Phase() == session.PhaseInWorld)
			if !switchAfterActivity {
				if err := world.clientActivity(); err != nil {
					log.Printf("%s: activate ready scene objects: %v", conn.RemoteAddr(), err)
					return
				}
				if npcPatrolCount > 0 {
					world.startPatrol(npcPatrolCount)
				}
			}
			if err := machine.Activity(); err != nil {
				log.Printf("%s: complete client-activity transition: %v", conn.RemoteAddr(), err)
				return
			}
			if demoDamage > 0 && player != nil && player.actor.ApplyDamage(demoDamage) {
				vitals, syncErr := player.vitalUpdate()
				if syncErr != nil {
					log.Printf("%s: encode demo damage update: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if writeErr := link.WriteFrame(vitals); writeErr != nil {
					log.Printf("%s: write demo damage update: %v", conn.RemoteAddr(), writeErr)
					return
				}
				log.Printf("%s: applied authoritative demo damage=%d", conn.RemoteAddr(), demoDamage)
				demoDamage = 0
			}
			if switchAfterActivity {
				destination := *switchDestination
				switchDestination = nil // one explicit test transition per connection
				neigongGranted = false
				if err := runtime.SwitchScene(destination); err != nil {
					log.Printf("%s: live scene switch: %v", conn.RemoteAddr(), err)
					return
				}
				sceneNPCs = runtime.npcCatalog
				pendingLocation = &runtime.activeRole.Location
				if err := store.saveLocation(ctx, selected, runtime.activeRole.Location); err != nil {
					log.Printf("%s: persist switched scene: %v", conn.RemoteAddr(), err)
				} else {
					pendingLocation = nil
				}
				log.Printf("%s: sent live 0x0C re-entry to scene=%s resource=%s; waiting for ClientReady", conn.RemoteAddr(), runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
			}
			if x, y, z, orient, ok := parseMotionPosition(plain); ok {
				if selected == nil {
					log.Printf("%s: position update without selected role", conn.RemoteAddr())
					return
				}
				location := selected.Location
				location.Position = role.Position{X: x, Y: y, Z: z, Orient: orient}
				nearbyCount, changed, syncErr := synchronizeNPCViewport(world, sceneNPCs, activeNPCs, location.Position, npcRadius)
				if syncErr != nil {
					log.Printf("%s: synchronize moving NPC viewport: %v", conn.RemoteAddr(), syncErr)
					return
				}
				if changed {
					if reconcileErr := world.reconcileViewport(); reconcileErr != nil {
						log.Printf("%s: reconcile moving NPC viewport: %v", conn.RemoteAddr(), reconcileErr)
						return
					}
					log.Printf("%s: NPC viewport moved center=(%.2f,%.2f) active=%d", conn.RemoteAddr(), x, z, nearbyCount)
				}
				// Retail Door objects activate as the role crosses their installed
				// MoveTarget/SpringRange, not only after a manual click. Detect that
				// boundary after this motion update has been accepted by the server,
				// then reuse the exact same portal execution/persistence path.
				if portal, enteredPortal := world.portalAt(location.Position, time.Now()); enteredPortal {
					if runtime == nil {
						log.Printf("%s: ignore proximity portal %s before scene runtime", conn.RemoteAddr(), portal.source)
						continue
					}
					var portalErr error
					if portal.sameScene {
						portalErr = runtime.TeleportWithinScene(portal.position)
					} else {
						portalErr = runtime.SwitchScene(portal.destination)
					}
					if portalErr != nil {
						log.Printf("%s: proximity portal %s failed: %v", conn.RemoteAddr(), portal.source, portalErr)
						continue
					}
					if !portal.sameScene {
						sceneNPCs = runtime.npcCatalog
					}
					pendingLocation = &runtime.activeRole.Location
					if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
						log.Printf("%s: persist proximity portal %s: %v", conn.RemoteAddr(), portal.source, saveErr)
					} else {
						pendingLocation = nil
					}
					if gmSession != nil {
						action := "传送门范围内移动"
						if !portal.sameScene {
							action = "传送门范围内切图"
						}
						gmSession.setLocation(runtime.activeRole.Location, action)
					}
					log.Printf("%s: entered portal %s trigger=(%.3f,%.3f) radius=%.1f same_scene=%t scene=%s resource=%s",
						conn.RemoteAddr(), portal.source, portal.trigger.X, portal.trigger.Z, portal.triggerRadius, portal.sameScene,
						runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource)
					continue
				}
				pendingLocation = &location
				if time.Since(lastPositionSave) >= 2*time.Second {
					if err := store.saveLocation(ctx, selected, location); err != nil {
						log.Printf("%s: persist movement: %v", conn.RemoteAddr(), err)
					} else {
						lastPositionSave = time.Now()
						pendingLocation = nil
					}
				}
			}
			// The current client places GameVisual:CustomSend inside its 0x0A
			// activity envelope.  In particular, real qinggong actions arrive
			// here as TVarList(216, qinggong_id, begin_or_end), rather than as a
			// top-level 0x1E frame.  Parse only a complete typed list so other
			// activity/motion packets keep their existing lifecycle behaviour.
			if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 && (custom.Values[0].Int32 == clientCustomSetShortcut || custom.Values[0].Int32 == clientCustomRemoveShortcut || custom.Values[0].Int32 == clientCustomSitCross || custom.Values[0].Int32 == 211 || custom.Values[0].Int32 == 215 || custom.Values[0].Int32 == 216 || custom.Values[0].Int32 == 217 || custom.Values[0].Int32 == clientCustomActiveParry || custom.Values[0].Int32 == 1006) {
				arguments := make([]string, 0, len(custom.Values)-1)
				for _, value := range custom.Values[1:] {
					arguments = append(arguments, value.String())
				}
				log.Printf("%s: client activity CustomSend opcode=0x0A msg_id=%d args=[%s]", conn.RemoteAddr(), custom.Values[0].Int32, strings.Join(arguments, ", "))
				switch custom.Values[0].Int32 {
				case clientCustomUseSkill:
					if _, handleErr := handleSkillCustom(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity skill: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomSetShortcut, clientCustomRemoveShortcut:
					if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity shortcut: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomSitCross:
					if _, handleErr := handleSitcrossCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity sitcross: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case 216:
					if _, handleErr := handleQingGongCustom(link, player, qingGongCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity qinggong: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case 217:
					if _, handleErr := handleActiveQingGongCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle active qinggong: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomActiveParry:
					if _, handleErr := handleActiveParryCustom(player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle active parry: %v", conn.RemoteAddr(), handleErr)
					}
				case 215, 1006:
					if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
						log.Printf("%s: handle activity inner-power msg=%d: %v", conn.RemoteAddr(), custom.Values[0].Int32, handleErr)
						return
					}
					persistFaculty()
				}
			}
		} else if plain[0] == 0x07 {
			request, ok := parseClientObjectRequest(plain)
			if !ok {
				log.Printf("%s: reject malformed C2S 0x07 object request len=%d", conn.RemoteAddr(), len(plain))
				continue
			}
			if request.FuncID != 0 {
				if err := world.selectNPCMenu(request); err != nil {
					log.Printf("%s: NPC menu selection id=%d func=%d: %v", conn.RemoteAddr(), request.ObjectID, request.FuncID, err)
				}
				continue
			}
			// Doors are direct object interactions, not NPC menus. The target
			// route comes solely from the installed DoorPackageID/door_data.ini
			// or GotoDoor PlayerTransferPosition data; unrecovered plot/clone
			// doors deliberately fall through to ordinary selection.
			portal, isPortal, portalErr := world.portalRoute(request)
			if portalErr != nil {
				log.Printf("%s: resolve portal id=%d owner=%d: %v", conn.RemoteAddr(), request.ObjectID, request.OwnerID, portalErr)
				continue
			}
			if isPortal {
				if runtime == nil {
					log.Printf("%s: ignore portal id=%d before scene runtime", conn.RemoteAddr(), request.ObjectID)
					continue
				}
				var routeErr error
				if portal.sameScene {
					routeErr = runtime.TeleportWithinScene(portal.position)
				} else {
					routeErr = runtime.SwitchScene(portal.destination)
				}
				if routeErr != nil {
					log.Printf("%s: portal %s id=%d failed: %v", conn.RemoteAddr(), portal.source, request.ObjectID, routeErr)
					continue
				}
				if !portal.sameScene {
					sceneNPCs = runtime.npcCatalog
				}
				pendingLocation = &runtime.activeRole.Location
				if saveErr := store.saveLocation(ctx, selected, runtime.activeRole.Location); saveErr != nil {
					log.Printf("%s: persist portal %s: %v", conn.RemoteAddr(), portal.source, saveErr)
				} else {
					pendingLocation = nil
				}
				if gmSession != nil {
					action := "传送门场景内移动"
					if !portal.sameScene {
						action = "传送门切换场景"
					}
					gmSession.setLocation(runtime.activeRole.Location, action)
				}
				log.Printf("%s: portal %s id=%d completed same_scene=%t scene=%s resource=%s position=(%.3f,%.3f,%.3f)",
					conn.RemoteAddr(), portal.source, request.ObjectID, portal.sameScene,
					runtime.activeRole.Location.Scene.Config, runtime.activeRole.Location.Scene.Resource,
					runtime.activeRole.Location.Position.X, runtime.activeRole.Location.Position.Y, runtime.activeRole.Location.Position.Z)
				continue
			}
			if _, err := world.objectRequest(request); err != nil {
				log.Printf("%s: object request id=%d owner=%d: %v", conn.RemoteAddr(), request.ObjectID, request.OwnerID, err)
			}
		} else if plain[0] == 0x1E || plain[0] == 0x27 {
			custom, err := parseClientCustomMessage(plain)
			if err != nil {
				log.Printf("%s: reject malformed C2S custom opcode=0x%02X: %v", conn.RemoteAddr(), plain[0], err)
				continue
			}
			if custom.Values[0].Type != 2 {
				log.Printf("%s: C2S custom first value must be int message ID, got %s", conn.RemoteAddr(), custom.Values[0])
				continue
			}
			arguments := make([]string, 0, len(custom.Values)-1)
			for _, value := range custom.Values[1:] {
				arguments = append(arguments, value.String())
			}
			log.Printf("%s: client CustomSend opcode=0x%02X msg_id=%d args=[%s]", conn.RemoteAddr(), custom.Opcode, custom.Values[0].Int32, strings.Join(arguments, ", "))
			switch custom.Values[0].Int32 {
			case clientCustomUseSkill:
				if _, handleErr := handleSkillCustom(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle skill: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomSetShortcut, clientCustomRemoveShortcut:
				if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle shortcut: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomSitCross:
				if _, handleErr := handleSitcrossCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle sitcross: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 216: // CLIENT_CUSTOMMSG_USE_QINGGONG
				if _, handleErr := handleQingGongCustom(link, player, qingGongCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle qinggong: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case 217: // CLIENT_CUSTOMMSG_ACTIVE_QINGGONG
				if _, handleErr := handleActiveQingGongCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle active qinggong: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomActiveParry:
				if _, handleErr := handleActiveParryCustom(player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle active parry: %v", conn.RemoteAddr(), handleErr)
				}
			case 215, 1006: // inner-power equip / faculty state machine
				if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
					log.Printf("%s: handle inner-power msg=%d: %v", conn.RemoteAddr(), custom.Values[0].Int32, handleErr)
					return
				}
				persistFaculty()
			}
		}
	}
}

// handleQingGongCustom applies a verified current-client 216 request. The
// third value in the captured 0x0A form is a begin/end marker: continuous
// qinggong types send 1 when starting and 0 when stopping.  Charging both
// would make a single hold/release cost twice, so only a begin marker spends
// the authoritative point resource.
func handleQingGongCustom(link sceneMessageConnection, player *playerActor, catalog *qinggong.Catalog, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 216 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject qinggong before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 3 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" || custom.Values[2].Type != 2 || (custom.Values[2].Int32 != 0 && custom.Values[2].Int32 != 1) {
		log.Printf("%s: reject malformed USE_QINGGONG values=%v", remote, custom.Values)
		return true, nil
	}
	definition, exists := catalog.Lookup(custom.Values[1].Text)
	if !exists {
		log.Printf("%s: reject unknown qinggong id=%q", remote, custom.Values[1].Text)
		return true, nil
	}
	if custom.Values[2].Int32 == 0 {
		if err := endQingGongBuff(link, player, definition); err != nil {
			return true, err
		}
		log.Printf("%s: qinggong end id=%s", remote, definition.ID)
		return true, nil
	}
	now := time.Now()
	if spec, hasBuff := qinggongBuffFor(definition); hasBuff && spec.clearOnEnd && player.hasActiveBuff(spec.slot, spec.staticData, now) {
		log.Printf("%s: ignore duplicate qinggong start id=%s", remote, definition.ID)
		return true, nil
	}
	cost, accepted := player.actor.UseQingGong(definition.Consume, now)
	if !accepted {
		state := player.actor.Snapshot()
		log.Printf("%s: reject qinggong id=%s cost=%d point=%d", remote, definition.ID, cost, state.QingGongPoint)
		return true, nil
	}
	alreadyActive, expires, err := activateQingGongBuff(player, definition, now)
	if err != nil {
		return true, err
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return true, fmt.Errorf("encode qinggong update: %w", err)
	}
	if err = link.WriteFrame(update); err != nil {
		return true, fmt.Errorf("write qinggong update: %w", err)
	}
	if !alreadyActive && !expires.IsZero() {
		spec, _ := qinggongBuffFor(definition)
		schedulePlayerBuffExpiry(link, player, spec.slot, spec.staticData, expires)
	}
	state := player.actor.Snapshot()
	log.Printf("%s: accepted qinggong id=%s type=%d cost=%d point=%d/%d", remote, definition.ID, definition.Type, cost, state.QingGongPoint, state.MaxQingGongPoint+state.MaxQingGongPointAdd)
	return true, nil
}

func handleActiveQingGongCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != 217 {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject active qinggong before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
		log.Printf("%s: reject malformed ACTIVE_QINGGONG values=%v", remote, custom.Values)
		return true, nil
	}
	changed, err := activateQingGongRecord(link, player, custom.Values[1].Text)
	if err != nil {
		return true, err
	}
	if changed {
		log.Printf("%s: activated qinggong id=%s via ActiveQGSkillRec", remote, custom.Values[1].Text)
	} else {
		log.Printf("%s: active qinggong already present id=%s", remote, custom.Values[1].Text)
	}
	return true, nil
}

type npcSpawn struct {
	resolved        clientdata.ResolvedNPC
	x, y, z, orient float32
}

func synchronizeNPCViewport(world *sceneLifecycle, catalog []npcSpawn, active map[int]struct{}, position role.Position, radius float32) (int, bool, error) {
	radiusSquared := radius * radius
	desired := make(map[int]struct{})
	for i := range catalog {
		// EventNpc instances are mission/event triggers, not ambient scene
		// residents.  The production server only gives a player an EventNpc
		// when that player's task/event state owns it.  This local server does
		// not yet persist that ownership, so the safe default is to omit the
		// object completely: no 0x0D means the client never constructs a
		// visual object for it.  When task state is implemented, this helper is
		// the single admission point for selectively allowing the matching
		// EventNpc back into this viewport.
		if !npcVisibleInLocalViewport(catalog[i]) {
			continue
		}
		dx := catalog[i].x - position.X
		dz := catalog[i].z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			desired[i] = struct{}{}
		}
	}
	changed := false
	for i := range active {
		if _, keep := desired[i]; keep {
			continue
		}
		if err := world.unregister(uint32(i + 2)); err != nil {
			return len(active), changed, err
		}
		delete(active, i)
		changed = true
	}
	for i := range desired {
		if _, exists := active[i]; exists {
			continue
		}
		npc := catalog[i].clone()
		npc.synchronizeTransform()
		if err := world.registerNPC(uint32(i+2), npc); err != nil {
			return len(active), changed, err
		}
		active[i] = struct{}{}
		changed = true
	}
	return len(active), changed, nil
}

func countNPCsWithin(catalog []npcSpawn, position role.Position, radius float32) int {
	radiusSquared := radius * radius
	count := 0
	for i := range catalog {
		if !npcVisibleInLocalViewport(catalog[i]) {
			continue
		}
		dx := catalog[i].x - position.X
		dz := catalog[i].z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			count++
		}
	}
	return count
}

// npcVisibleInLocalViewport is deliberately a server-side admission gate.
// The modern creator templates encode actual trigger objects with
// ScriptClass=EventNpc (for example, EventNpc_gmpzyb_028 is described as a
// 古墓派事件触发 point).  These are only meaningful after the server has bound
// them to a player's active task/event.  No such binding exists in the local
// role model yet, so an EventNpc has no eligible viewer.
func npcVisibleInLocalViewport(npc npcSpawn) bool {
	return !strings.EqualFold(strings.TrimSpace(npc.resolved.ScriptClass), "EventNpc")
}

func npcAddObject(objectID uint32, npc npcSpawn, probeIndex int) ([]byte, error) {
	msg := make([]byte, 0x3f, 256)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	// This field is the current scene/owner identity, not the viewport slot.
	// The player entry establishes scene identity 1 for this connection.
	binary.LittleEndian.PutUint32(msg[5:], 1)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(npc.x))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(npc.y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(npc.z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(npc.orient))
	// Offsets 0x09..0x3c are the 13-value transform/motion header copied by
	// current FxNet2::ServerAddObject. Object kind belongs in the negotiated
	// Type property below; do not smuggle a speculative kind into the tail.
	_ = probeIndex
	properties, err := npc.resolved.OrderedProperties(clientdata.VisibleNPCModernV1())
	if err != nil {
		return nil, err
	}
	for _, property := range properties {
		msg = appendNPCProperty(msg, property)
	}
	binary.LittleEndian.PutUint16(msg[0x3d:], uint16(len(properties)))
	return msg, nil
}

func sceneObjectByteProperty(objectID, ownerID uint32, propertyIndex uint16, value byte) []byte {
	msg := make([]byte, 12, 15)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // scene object, not view object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	msg = binary.LittleEndian.AppendUint16(msg, propertyIndex)
	msg = append(msg, value)
	return msg
}

func sceneObjectTransformProperties(objectID, ownerID uint32, transform worldcore.Transform) []byte {
	msg := make([]byte, 12, 36)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // scene object, not view object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 4)
	for _, property := range []struct {
		index uint16
		value float32
	}{
		{9, transform.X},
		{10, transform.Y},
		{11, transform.Z},
		{12, transform.Orient},
	} {
		msg = binary.LittleEndian.AppendUint16(msg, property.index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(property.value))
	}
	return msg
}

// serverLocation encodes FxNet2's modern ServerLocation message. Unlike a
// ServerObjectProperty update, opcode 0x1F updates the object's native Posi
// and Dest transforms and raises on_location, which is the path used by the
// game client to place an already-created asynchronous Actor2 visual.
func serverLocation(objectID, ownerID uint32, transform worldcore.Transform) []byte {
	msg := make([]byte, 25)
	msg[0] = 0x1F
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	binary.LittleEndian.PutUint32(msg[9:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(msg[13:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(msg[17:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(msg[21:], math.Float32bits(transform.Orient))
	return msg
}

func sceneObjectStringProperty(objectID, ownerID uint32, propertyIndex uint16, value string) []byte {
	msg := make([]byte, 12, 18+len(value))
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // scene object, not view object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	return appendStringProperty(msg, propertyIndex, value)
}

// sceneObjectObjectProperty updates a property whose negotiated type is the
// native eight-byte object reference.  FxNet2 identifies a scene object as
// the low-word object ID followed by the high-word owner ID.
func sceneObjectObjectProperty(objectID, ownerID, propertyIndex, targetID, targetOwnerID uint32) []byte {
	msg := make([]byte, 12, 22)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // scene object, not view object
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	return appendObjectProperty(msg, uint16(propertyIndex), targetID, targetOwnerID)
}

func visiblePropertyTable(properties []clientdata.FieldSpec) []byte {
	msg := make([]byte, 3, 3+len(properties)*12)
	msg[0] = 0x09
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(properties)))
	for _, property := range properties {
		msg = append(msg, property.Name...)
		msg = append(msg, 0, byte(property.Type))
	}
	return msg
}

func sceneVisiblePropertyFields(npcFields []clientdata.FieldSpec) []clientdata.FieldSpec {
	fields := append([]clientdata.FieldSpec(nil), npcFields...)
	fields = append(fields, clientdata.FieldSpec{Index: 99, Name: "MaxVisuals", Type: clientdata.WireInt32})
	fields = append(fields, clientdata.FieldSpec{Index: 100, Name: "LastObject", Type: clientdata.WireObject})
	// View archives use the same negotiated property table as scene objects.
	// These fields are read directly by the current form_shop.lua createview
	// branch, so they must be negotiated before a shop View=61 is sent.
	fields = append(fields,
		clientdata.FieldSpec{Index: 101, Name: "ShopID", Type: clientdata.WireString},
		clientdata.FieldSpec{Index: 102, Name: "PageNum", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 103, Name: "PageCount", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 104, Name: "ShopType", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 105, Name: "ConfigID", Type: clientdata.WireString},
		clientdata.FieldSpec{Index: 106, Name: "Amount", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 107, Name: "SellPrice0", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 108, Name: "SellPrice1", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 109, Name: "SellPrice2", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 110, Name: "MaxAmount", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 111, Name: "CapitalType1", Type: clientdata.WireInt32},
		// These live-player fields are named by the latest Lua property binds;
		// they are appended beyond the modern scene schema so 0x10 can carry
		// the same authority after the initial 0x0D archive.
		clientdata.FieldSpec{Index: 112, Name: "HitHP", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 113, Name: "HitHPRatio", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 114, Name: "QingGongPoint", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 115, Name: "MaxQingGongPoint", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: 116, Name: "MaxQingGongPointAdd", Type: clientdata.WireInt32},
		// Current-client QingGong motion inputs.  These are ordinary player
		// properties, not action packets: FxGameLogic reads them when resolving
		// rush, water, air and climb actions.  In particular, a missing
		// LandRushDist is displayed and used as zero by the client.
		clientdata.FieldSpec{Index: 117, Name: "RunSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 118, Name: "JumpSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 119, Name: "DriftSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 120, Name: "DriftSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 121, Name: "DriftJumpSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 122, Name: "DriftJumpSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 123, Name: "SndJumpSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 124, Name: "SndJumpSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 125, Name: "ThdJumpSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 126, Name: "ThdJumpSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 127, Name: "AirRushSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 128, Name: "AirRushSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 129, Name: "AirRushDist", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 130, Name: "AirRushDistAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 131, Name: "LandRushSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 132, Name: "LandRushSpeedAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 133, Name: "LandRushDist", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 134, Name: "LandRushDistAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 135, Name: "ClimbSpeed", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 136, Name: "ClimbSpeedAdd", Type: clientdata.WireFloat32},
		// The native movement code also queries these range aliases while
		// resolving the three rush families.
		clientdata.FieldSpec{Index: 137, Name: "GRushRange", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 138, Name: "GRushRangeAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 139, Name: "ARushRange", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 140, Name: "ARushRangeAdd", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 141, Name: "WRushRange", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 142, Name: "WRushRangeAdd", Type: clientdata.WireFloat32},
		// Buff state is native player-object state. FxGameLogic's form_main_buff
		// reads and binds this string; it is not a 0x1E Lua custom message.
		clientdata.FieldSpec{Index: 143, Name: "BufferListStr", Type: clientdata.WireString},
	)
	for slot := uint16(1); slot <= bufferInfoSlotCount; slot++ {
		fields = append(fields, clientdata.FieldSpec{
			Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Type: clientdata.WireString,
		})
	}
	// Base jump physics is read by the native player action object, rather
	// than derived from the qinggong action packet.  Keep these appended: the
	// preceding field order is already live on the modern client.
	fields = append(fields,
		clientdata.FieldSpec{Index: 168, Name: "Gravity", Type: clientdata.WireFloat32},
		clientdata.FieldSpec{Index: 169, Name: "GravityAdd", Type: clientdata.WireFloat32},
		// FallDownManager reads this player property directly.  When it is
		// absent the modern client defaults its safe-drop threshold to 5.0,
		// which makes ordinary local-test terrain drops enter the fall-down
		// state instead of returning to the grounded action.
		clientdata.FieldSpec{Index: 170, Name: "DropHeightPub", Type: clientdata.WireFloat32},
	)
	// Inner-power and role-panel fields are deliberately appended after every
	// already-live movement/Buff field.  View 43 uses the same negotiated
	// archive, so its per-book values must be named here before ViewAdd.
	fields = append(fields, playerProgressFields()...)
	// Fight::UseShortcutMain reaches sub_127FE280 before it can emit C2S 211.
	// NeigongPKStatus is read there without a HasProp guard, so leaving it
	// unnegotiated is not equivalent to an explicit normal value. Dead is also
	// part of the same pre-send gate; keep both as ordinary actor properties.
	fields = append(fields,
		clientdata.FieldSpec{Index: propNeigongPKStatus, Name: "NeigongPKStatus", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propDead, Name: "Dead", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propCantUseSkill, Name: "CantUseSkill", Type: clientdata.WireInt32},
		// USE_SKILL (C2S 211) is only the request. FxGameLogic waits for this
		// authoritative player-property callback before refreshing shortcut
		// cooldown state and staging the native Fight action.
		clientdata.FieldSpec{Index: propCurSkillID, Name: "CurSkillID", Type: clientdata.WireString},
		// Learned View=40 objects expose the action lease through this shared
		// property table. The native Fight manager reads milliseconds as float.
		clientdata.FieldSpec{Index: propPauseTime, Name: "PauseTime", Type: clientdata.WireFloat32},
		// Native main-role HUD binds the four recovery-rate fields. Their values
		// are informational; HP/MP still change only after a server tick.
		clientdata.FieldSpec{Index: propHPHeartSpeed, Name: "HPHeartSpeed", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propHPHeartSpeedAdd, Name: "HPHeartSpeedAdd", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propMPHeartSpeed, Name: "MPHeartSpeed", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propMPHeartSpeedAdd, Name: "MPHeartSpeedAdd", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propHPUpSpeed, Name: "HPUpSpeed", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propHPUpSpeedAdd, Name: "HPUpSpeedAdd", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propMPUpSpeed, Name: "MPUpSpeed", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propMPUpSpeedAdd, Name: "MPUpSpeedAdd", Type: clientdata.WireInt32},
		// A normal skill acceptance publishes this complete public tuple. The
		// self-target action gate additionally reads ModifySkillLockTime as an
		// integer tolerance in milliseconds.
		clientdata.FieldSpec{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Type: clientdata.WireString},
		clientdata.FieldSpec{Index: propCurSkillLevel, Name: "CurSkillLevel", Type: clientdata.WireInt32},
		clientdata.FieldSpec{Index: propCurSkillTarget, Name: "CurSkillTarget", Type: clientdata.WireObject},
		clientdata.FieldSpec{Index: propModifySkillLockTime, Name: "ModifySkillLockTime", Type: clientdata.WireInt32},
		// Skillbase declares CanUse as a private byte, and current FxGameLogic
		// reads it from learned View=40 objects before presenting/using a skill.
		clientdata.FieldSpec{Index: propSkillCanUse, Name: "CanUse", Type: clientdata.WireByte},
		// Later affiliations are selected natively from Force/NewSchool before
		// School. These must be part of the negotiated table before PlayerAdd.
		clientdata.FieldSpec{Index: propForce, Name: "Force", Type: clientdata.WireString},
		clientdata.FieldSpec{Index: propNewSchool, Name: "NewSchool", Type: clientdata.WireString},
	)
	return fields
}

func appendStringProperty(msg []byte, index uint16, value string) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}

func appendObjectProperty(msg []byte, index uint16, objectID, ownerID uint32) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	return binary.LittleEndian.AppendUint64(msg, uint64(objectID)|uint64(ownerID)<<32)
}

func appendWideStringProperty(msg []byte, index uint16, value string) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, index)
	units := utf16.Encode([]rune(value))
	msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
	for _, unit := range units {
		msg = binary.LittleEndian.AppendUint16(msg, unit)
	}
	return binary.LittleEndian.AppendUint16(msg, 0)
}
