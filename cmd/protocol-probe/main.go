package main

import (
	"bytes"
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
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/neigong"
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
	demoDamage := flag.Int("demo-damage", 0, "test-only damage applied once after the player reaches the active scene (0 disables it)")
	starterSilver := flag.Int("starter-silver", 99999, "test-only starting CapitalType1 (碎银) for the active player")
	gmListen := flag.String("gm-listen", "127.0.0.1:19062", "local GM HTTP listen address (empty disables)")
	logFile := flag.String("log-file", runtimeProjectPath("artifacts", "local", "logs", "protocol-probe-live.log"), "append runtime log to this local file")
	npcFuncPath := flag.String("npc-func", defaultLegacyNPCFuncPath, "Npc_Func.xml used to build per-NPC service menus")
	scriptsPath := flag.String("scripts", runtimeProjectPath("scripts", "host"), "server-side event Lua scripts directory (game.on contract; empty disables)")
	captureDir := flag.String("capture-dir", runtimeProjectPath("captures", "local"), "dump raw wire bytes of every session to <dir>/<ip-pid>_c2s.bin and _s2c.bin (empty disables)")
	neigongScriptsPath := flag.String("neigong-scripts", runtimeProjectPath("scripts", "module", "neigong"), "neigong effect module scripts directory (buf_ng_*.lua; empty disables)")
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
	// 触发资源仓初始化（本地 > patch > base，带读缓存），启动日志
	// 会给出各层条目与缓存上限；不可用时回退直接文件访问。
	resourceAccessorOf()
	qingGongDefine, qingGongReadErr := readResourceBytes(*qingGongDefinePath)
	if qingGongReadErr != nil {
		log.Fatalf("read current-client qinggong definitions %s: %v", *qingGongDefinePath, qingGongReadErr)
	}
	qingGongCatalog, qingGongErr := qinggong.LoadFromReader(bytes.NewReader(qingGongDefine), *qingGongDefinePath)
	if qingGongErr != nil {
		log.Fatalf("load current-client qinggong definitions: %v", qingGongErr)
	}
	log.Printf("qinggong catalog ready definitions=%d path=%s", qingGongCatalog.Count(), *qingGongDefinePath)
	npcFuncRaw, npcFuncReadErr := readResourceBytes(*npcFuncPath)
	var npcFuncs *npcfunc.Registry
	var npcFuncErr error
	if npcFuncReadErr != nil {
		npcFuncErr = npcFuncReadErr
	} else {
		npcFuncs, npcFuncErr = npcfunc.LoadFromBytes(npcFuncRaw, *npcFuncPath)
	}
	if npcFuncErr != nil {
		log.Printf("Npc_Func menu registry unavailable (%s): %v; NPCs without this table will expose no fabricated service menu", *npcFuncPath, npcFuncErr)
	} else {
		log.Printf("Npc_Func menu registry ready configured_npcs=%d path=%s", npcFuncs.Count(), *npcFuncPath)
	}
	// 任务目录是进程级静态配置（快照或资源派生，见 task_catalog.go）；
	// 在任何连接进入前装配。
	taskCatalog = openTaskCatalog()
	scriptHost := luascript.New(*scriptsPath, log.Printf)
	if *scriptsPath != "" {
		loaded, scriptErr := scriptHost.LoadDir()
		if scriptErr != nil {
			log.Printf("Lua scripts partially loaded count=%d path=%s: %v", loaded, *scriptsPath, scriptErr)
		} else {
			log.Printf("Lua scripts ready count=%d path=%s", loaded, *scriptsPath)
		}
	}
	// 内功特效模块脚本（scripts/module/neigong）：独立 VM、单本禁用隔离；
	// 战斗二期接线前仅装载建目录，钩子调用桥见 internal/neigong。
	neigongEffects := neigong.New(*neigongScriptsPath, log.Printf)
	if *neigongScriptsPath != "" {
		loaded, neigongErr := neigongEffects.Load()
		if neigongErr != nil {
			log.Printf("内功特效脚本 partially loaded count=%d path=%s: %v", loaded, *neigongScriptsPath, neigongErr)
		} else {
			log.Printf("内功特效脚本 ready count=%d path=%s", loaded, *neigongScriptsPath)
		}
	}
	if *npcAuditAll {
		entries, auditErr := auditNPCSceneRoot(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir))
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
		sceneRegistry, err = newSceneNPCRegistry(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir))
		if err == nil {
			log.Printf("modern scene NPC registry ready template_tables=%d creator_root=%s", len(sceneRegistry.tables), sceneRegistry.root)
		}
	} else {
		staticNPCs, err = loadNPCSpawns(*npcTablePath, *npcCreatorPath, *npcConfigID)
		if err == nil {
			// 单目录回退模式同样绑定 mappath 巡逻（mappath 在 npc_creator
			// 的兄弟目录，与注册表模式同一布局）。
			creatorDir := filepath.Dir(*npcCreatorPath)
			bindPatrolRoutes(staticNPCs, loadPatrolRoutes(filepath.Join(filepath.Dir(creatorDir), "mappath"), filepath.Base(creatorDir)), time.Now().UnixMilli())
		}
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
		_, propertyErr := npc.resolved.OrderedProperties()
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
	facultyStore := newFacultyDBStore(store.db)
	inventoryStore := newInventoryStore(store.db)
	skillStore := newSkillDBStore(store.db)
	customConfig := newCustomConfigStores(store.db)
	appearanceCfg := newAppearanceStores(store.db)
	gm := newGMHub()
	gm.itemNames = func(configIDs []string) map[string]string {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return inventoryStore.itemNames(ctx, configIDs)
	}
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
		go handle(conn, store, facultyStore, inventoryStore, skillStore, customConfig, appearanceCfg, sceneRegistry, staticNPCs, npcFuncs, qingGongCatalog, float32(*npcRadius), int32(*demoDamage), int32(*starterSilver), switchDestination, gm, scriptHost, neigongEffects, *captureDir)
	}
}

func handle(conn net.Conn, store *roleStore, facultyStore *facultyDBStore, inventory *inventoryStore, skills *skillDBStore, customConfig customConfigStores, appearanceCfg appearanceStores, sceneRegistry *sceneNPCRegistry, staticNPCs []npcSpawn, npcFuncs *npcfunc.Registry, qingGongCatalog *qinggong.Catalog, npcRadius float32, demoDamage, starterSilver int32, switchDestination *sceneDestination, gm *gmHub, scriptHost *luascript.Host, neigongEffects *neigong.Effects, captureDir string) {
	if captureDir != "" {
		var tap *captureTap
		conn, tap = newCapturedConn(conn, captureDir)
		if tap != nil {
			// defer link.Close()（更晚注册、先执行）会经由 capturedConn.Close
			// 落盘并关闭这对文件；此处只负责宣告文件位置。
			log.Printf("capture session=%s prefix=%s", conn.RemoteAddr(), tap.logName(captureDir))
		}
	}
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
	world := newSharedSceneLifecycle(conn.RemoteAddr().String(), link, npcFuncs)
	world.scripts = scriptHost
	world.radius = npcRadius
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
	// dbLearnedSkills 是 0x0C 装载的 role_skills 已学集（wuxue/anqi）。
	// 空切片 = 无 DB 行或读库失败，grantStarterProgress 回退 starter 集。
	dbLearnedSkills := []learnedSkillView{}
	// 自定义配置域的会话级状态（Java LoginSession 对应物）：DB 快捷栏
	// 基线装载后 starter 授予整体让位；键位/个性配置推送一次推全，
	// 切图重推只针对挂在玩家对象上的 0x10 属性。
	shortcutsFromDB := false
	shortcutRowsSent := false
	customizingSess := customizingSession{}
	customizingPushed := false
	keybindPushed := false
	currentRoleID := func() role.RoleID {
		if selected != nil {
			return selected.ID
		}
		return 0
	}
	// 物品域会话上下文（msg30/31/34/36 共用；store 缺库时纯内存降级）。
	newItemSession := func() *itemSession {
		return &itemSession{
			link: link, player: player, store: inventory, skins: appearanceCfg.skins,
			ctx: ctx, roleID: currentRoleID(), remote: conn.RemoteAddr().String(),
			now: time.Now,
		}
	}
	// 外观域会话上下文（msg61/19 共用）。
	newAppearanceSession := func() *appearanceSession {
		return &appearanceSession{
			link: link, player: player, inventory: inventory,
			appearance: appearanceCfg.appearance, fashionStore: appearanceCfg.fashion,
			ctx: ctx, roleID: currentRoleID(), remote: conn.RemoteAddr().String(),
		}
	}
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
		skillFrames, err := learnedSkillViewFrames(mergeLearnedSkills(dbLearnedSkills))
		if err != nil {
			return fmt.Errorf("encode View=40: %w", err)
		}
		for _, frame := range skillFrames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=40: %w", err)
			}
		}
		// 套路快捷栏授予是 starter 回退：DB 有玩家编辑基线时让位，
		// 否则每次重入都会复活玩家已删除的条目。
		if shortcutsFromDB {
			log.Printf("%s: skip starter scheme shortcuts, %d rows restored from role_shortcuts", conn.RemoteAddr(), len(player.shortcutSnapshot()))
		} else {
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
	// 只有 stage_main 在原始玩家归档完成后发出的一字节
	// ClientReady 才是实机 0x0C 测试的安全边界。
	// 更早的 0x0A 包属于客户端常规自定义/活动流量，
	// 可能在首个 ServerAddObject 尚未处理完时就到达。
	explicitSceneReady := false
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
		case "set_showequip_type":
			player.setShowEquipTypePref(int(command.showEquipType))
			if saveErr := appearanceCfg.appearance.Save(selected.ID, int(command.showEquipType)); saveErr != nil {
				log.Printf("%s: GM persist show equip type role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			if refreshErr := pushAppearanceRefresh(link, player); refreshErr != nil {
				gmSession.setAction("显示类型切换失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction(fmt.Sprintf("外显类型已切为 %s", showEquipTypeName(int(command.showEquipType))))
			log.Printf("%s: GM set ShowEquipType=%d", conn.RemoteAddr(), command.showEquipType)
		case "set_gold":
			player.setGold(command.gold)
			update, updateErr := player.goldUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("金币设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("金币设为 %d（会话级）", command.gold))
			log.Printf("%s: GM set CapitalType0=%d", conn.RemoteAddr(), command.gold)
		case "grant_identity_suits":
			before := len(player.fashionSuitSnapshot())
			grantIdentitySuitsAtLogin(&appearanceSession{
				link: link, player: player, inventory: inventory,
				appearance: appearanceCfg.appearance, fashionStore: appearanceCfg.fashion,
				ctx: ctx, roleID: selected.ID, remote: conn.RemoteAddr().String(),
			})
			granted := len(player.fashionSuitSnapshot()) - before
			if granted > 0 && player.fashionRowsPushed() {
				if rowsErr := pushFashionSuitRows(link, player.fashionSuitSnapshot()); rowsErr != nil {
					gmSession.setAction("身份装行推送失败：" + rowsErr.Error())
					return rowsErr
				}
			}
			gmSession.setAction(fmt.Sprintf("身份装激活完成，本次新增 %d 套", granted))
		case "grant_all_fashion":
			fashionAll := fashionSuits.fashionSuitsAll()
			if len(fashionAll) == 0 {
				gmSession.setAction("时装目录为空（缺 ini/form/fashion_suit.ini）")
				return fmt.Errorf("fashion suit catalog is empty")
			}
			if activateErr := appearanceCfg.fashion.Activate(selected.ID, fashionAll); activateErr != nil {
				gmSession.setAction("时装激活落库失败：" + activateErr.Error())
				return activateErr
			}
			player.restoreFashionSuits(mergeFashionSuits(player.fashionSuitSnapshot(), fashionAll))
			if player.fashionRowsPushed() {
				if rowsErr := pushFashionSuitRows(link, player.fashionSuitSnapshot()); rowsErr != nil {
					gmSession.setAction("时装行推送失败：" + rowsErr.Error())
					return rowsErr
				}
			}
			gmSession.setAction(fmt.Sprintf("已激活全部时装 %d 套", len(fashionAll)))
			log.Printf("%s: GM grant_all_fashion role=%d suits=%d", conn.RemoteAddr(), selected.ID, len(fashionAll))
		case "grant_item", "add_item":
			session := &itemSession{
				link: link, player: player, store: inventory, skins: appearanceCfg.skins,
				ctx: ctx, roleID: selected.ID, remote: conn.RemoteAddr().String(), now: time.Now,
			}
			entry, known := inventory.lookupGrantable(ctx, command.itemConfig)
			if !known {
				gmSession.setAction("发放失败：game_items 无此配置 " + command.itemConfig)
				return fmt.Errorf("unknown item config %q", command.itemConfig)
			}
			equipType := entry.equipType
			if equipType == "" {
				if stat := equipmentModels.itemStatic(command.itemConfig); stat != nil {
					equipType = stat.equipType
				}
			}
			if command.wearNow && equipSlotFor(equipType, player.sessionEquips()) > 0 {
				slot := equipSlotFor(equipType, player.sessionEquips())
				worn, wearErr := inventory.upsertEquipSlot(ctx, selected.ID, slot, command.itemConfig, entry.itemType, equipType)
				if wearErr != nil {
					gmSession.setAction("穿戴发放失败：" + wearErr.Error())
					return wearErr
				}
				player.replaceEquipSlot(worn)
				if framesErr := pushWearFrames(link, selected.ID, player.visualSexOf(), effectiveShowEquipType(player), worn, player.sessionEquips()); framesErr != nil {
					gmSession.setAction("穿戴帧推送失败：" + framesErr.Error())
					return framesErr
				}
				gmSession.setAction(fmt.Sprintf("已发放并穿戴 %s（槽位 %d）", command.itemConfig, slot))
				log.Printf("%s: GM grant_item wear config=%s slot=%d", conn.RemoteAddr(), command.itemConfig, slot)
				return nil
			}
			landed := session.grantBagItem(entry, command.amount)
			if landed <= 0 {
				gmSession.setAction("发放失败：背包已满（1..90）")
				return fmt.Errorf("bag full")
			}
			gmSession.setAction(fmt.Sprintf("已发放 %s x%d（背包 %d 页槽 %d）", command.itemConfig, command.amount, entry.viewID, landed))
			log.Printf("%s: GM grant_item bag config=%s amount=%d slot=%d", conn.RemoteAddr(), command.itemConfig, command.amount, landed)
		case "equip_card", "equip_hair":
			entry, known := skinCards.lookup(int(command.cardId))
			if !known {
				gmSession.setAction(fmt.Sprintf("皮肤 %d 不在 card.ini 目录", command.cardId))
				return fmt.Errorf("unknown skin %d", command.cardId)
			}
			if command.action == "equip_hair" && !entry.hairOnly() {
				gmSession.setAction(fmt.Sprintf("皮肤 %d 不是纯发型卡（%s）", command.cardId, skinPartSummary(entry, player.visualSexOf())))
				return fmt.Errorf("skin %d is not hair-only", command.cardId)
			}
			state := player.skinSnapshot()
			sex := player.visualSexOf()
			state.addCollected(command.cardId)
			if command.action == "equip_hair" {
				mounted := make([]int32, 0, len(state.mounted))
				for _, id := range state.mounted {
					other, ok := skinCards.lookup(int(id))
					if ok && other.hairOnly() && id != command.cardId {
						continue // 其他发型卡让位
					}
					if id == command.cardId {
						continue // 重挂刷新到队尾
					}
					mounted = append(mounted, id)
				}
				state.mounted = append(mounted, command.cardId)
			} else {
				state.removeMounted(command.cardId)
				state.mounted = append(state.mounted, command.cardId)
			}
			if saveErr := appearanceCfg.skins.Save(selected.ID, state); saveErr != nil {
				log.Printf("%s: GM persist skins role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			player.applySkins(state)
			if refreshErr := pushSkinRefreshFrames(link, player); refreshErr != nil {
				gmSession.setAction("衣橱刷新失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction(fmt.Sprintf("已挂载皮肤 %d（%s）", command.cardId, skinPartSummary(entry, sex)))
			log.Printf("%s: GM %s skin=%d", conn.RemoteAddr(), command.action, command.cardId)
		case "unequip_card":
			state := player.skinSnapshot()
			if len(state.mounted) == 0 {
				gmSession.setAction("当前无挂载皮肤")
				return nil
			}
			state.mounted = nil
			if saveErr := appearanceCfg.skins.Save(selected.ID, state); saveErr != nil {
				log.Printf("%s: GM persist skins role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			player.applySkins(state)
			if refreshErr := pushSkinRefreshFrames(link, player); refreshErr != nil {
				gmSession.setAction("衣橱刷新失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction("已卸载全部皮肤（图鉴保留）")
			log.Printf("%s: GM unequip_card role=%d", conn.RemoteAddr(), selected.ID)
		case "grant_all_cards":
			allIDs := skinCards.allIDs()
			if len(allIDs) == 0 {
				gmSession.setAction("皮肤目录为空（缺 share/rule/card.ini）")
				return fmt.Errorf("skin catalog is empty")
			}
			state := player.skinSnapshot()
			for _, id := range allIDs {
				state.addCollected(int32(id))
			}
			if saveErr := appearanceCfg.skins.Save(selected.ID, state); saveErr != nil {
				log.Printf("%s: GM persist skins role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			player.applySkins(state)
			if refreshErr := pushSkinRefreshFrames(link, player); refreshErr != nil {
				gmSession.setAction("衣橱刷新失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction(fmt.Sprintf("已全量收录 %d 张皮肤卡", len(allIDs)))
			log.Printf("%s: GM grant_all_cards role=%d skins=%d", conn.RemoteAddr(), selected.ID, len(allIDs))
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
		case "lua_eval":
			if !scriptHost.Enabled() {
				gmSession.setAction("Lua 脚本未启用（-scripts 为空）")
				return fmt.Errorf("lua scripts are disabled")
			}
			result, evalErr := scriptHost.Eval(command.code)
			if evalErr != nil {
				gmSession.setAction("Lua 执行失败：" + evalErr.Error())
				return evalErr
			}
			if result == "" {
				result = "(无返回值)"
			}
			if len(result) > 512 {
				result = result[:512] + "…"
			}
			gmSession.setAction("Lua 结果：" + result)
			log.Printf("%s: GM lua_eval ok", conn.RemoteAddr())
		case "lua_reload":
			if !scriptHost.Enabled() {
				gmSession.setAction("Lua 脚本未启用（-scripts 为空）")
				return fmt.Errorf("lua scripts are disabled")
			}
			loaded, reloadErr := scriptHost.Reload()
			if reloadErr != nil {
				gmSession.setAction(fmt.Sprintf("Lua 重载完成但有失败脚本 count=%d：%v", loaded, reloadErr))
				return reloadErr
			}
			// 内功特效模块一并热重载；失败只回报原因，不影响事件脚本结果。
			if neigongEffects.Enabled() {
				neigongLoaded, neigongErr := neigongEffects.Reload()
				if neigongErr != nil {
					gmSession.setAction(fmt.Sprintf("Lua 脚本已重载 count=%d；内功特效部分失败 count=%d：%v", loaded, neigongLoaded, neigongErr))
					return neigongErr
				}
				gmSession.setAction(fmt.Sprintf("Lua 脚本已重载 count=%d；内功特效 count=%d", loaded, neigongLoaded))
				log.Printf("%s: GM lua_reload host=%d neigong=%d", conn.RemoteAddr(), loaded, neigongLoaded)
				break
			}
			gmSession.setAction(fmt.Sprintf("Lua 脚本已重载 count=%d", loaded))
			log.Printf("%s: GM lua_reload count=%d", conn.RemoteAddr(), loaded)
		case "task_rebuild":
			// 只重派生并覆写快照；运行中的场景继续使用装配时的目录，
			// 新目录在下次启动经 openTaskCatalog 生效。
			_, stats, rebuildErr := rebuildTaskSnapshot()
			if rebuildErr != nil {
				gmSession.setAction("任务快照重建失败：" + rebuildErr.Error())
				return rebuildErr
			}
			gmSession.setAction(fmt.Sprintf("任务快照已重建：任务 %d、绑定 %d、NPC %d（重启后生效）",
				stats.Tasks, stats.Bindings, stats.BoundNPCs))
			log.Printf("%s: GM task_rebuild tasks=%d bindings=%d npcs=%d",
				conn.RemoteAddr(), stats.Tasks, stats.Bindings, stats.BoundNPCs)
		default:
			return fmt.Errorf("unsupported GM action %q", command.action)
		}
		return nil
	}
	// 只保留一个 goroutine 阻塞在 ReadFrame 上，
	// 再将其完成的帧与下方的 GM 命令多路复用。
	// 若在主循环中直接读取，一个安静的客户端会无限期阻塞
	// GM 命令，直到它下次移动或发送心跳。这里刻意不设读取超时：
	// 在带转义传输帧的中途超时可能丢弃其不完整的包体。
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
			combatRageAdvanced := player.rageRecoveryTick()
			// NPC 战斗模拟已上移到场景运行时的中央 tick；
			// 连接循环只保留玩家侧结算与复活。
			combatAdvanced := false
			if player.reviveWhenDue(now) {
				combatAdvanced = true
				log.Printf("%s: auto-revived player after NPC defeat", conn.RemoteAddr())
			}
			var tickErr error
			if advanced {
				var frame []byte
				frame, tickErr = player.progressBookFrame(slot)
				if tickErr == nil {
					tickErr = link.WriteFrame(frame)
				}
			}
			if tickErr == nil && (advanced || sitcrossAdvanced || combatBruiseAdvanced || combatRageAdvanced || combatAdvanced || innerPowerPassiveExpired) {
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
				state := player.stateSnapshot()
				log.Printf("%s: settled sitcross HP=%d/%d MP=%d/%d", conn.RemoteAddr(), state.HP, state.MaxHP, state.MP, state.MaxMP)
			}
			if combatBruiseAdvanced {
				state := player.stateSnapshot()
				log.Printf("%s: settled combat bruised blood HP=%d HitHP=%d", conn.RemoteAddr(), state.HP, state.HitHP)
			}
			if combatRageAdvanced {
				log.Printf("%s: settled combat rage SP=%d/%d", conn.RemoteAddr(), player.skillSP(), maxPlayerSP)
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
				// 选角列表外观（Java roleListVisual）：f6=显示类型存档、
				// f9/f10/f11=按显示模式合并的衣裤鞋模型——选角界面与游戏内
				// 外观一致；读库失败回 golden 形态。
				look := roleListLookOf(ctx, inventory, appearanceCfg, selected.ID, selected.Appearance.Values)
				payload = oneRoleLoginSuccessWithLook(time.Now(), selected.Name, selected.Appearance.Values,
					selected.Location.Scene.Config, selected.Location.Scene.Resource, look)
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
			// ClientMsgWorldInfo。旧的世界服务器以 opcode
			// 0x05、uint16 信息类型以及 NUL 结尾的 UCS-2 字符串应答。
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
			if err := link.WriteFrame(oneRoleLoginSuccess(time.Now(), name, selected.Appearance.Values,
				selected.Location.Scene.Config, selected.Location.Scene.Resource)); err != nil {
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
			// 外观域会话态（docs/装备外观与风物志挂载.md V4）：显示类型
			// 偏好（role_appearance）、时装/身份装激活集（role_fashion_suits，
			// 登录按 faction 幂等补发身份装）、皮肤衣橱（role_skins）。
			// 全部软依赖：缺表/坏行回默认（江湖装/空衣橱），不阻塞登录。
			if pref, saved := appearanceCfg.appearance.Load(activeRole.ID); saved {
				player.setShowEquipTypePref(pref)
				log.Printf("%s: restored show equip type role=%d type=%d", conn.RemoteAddr(), activeRole.ID, pref)
			}
			if suits, saved := appearanceCfg.fashion.List(activeRole.ID); saved {
				player.restoreFashionSuits(suits)
			}
			grantIdentitySuitsAtLogin(&appearanceSession{
				link: link, player: player, inventory: inventory,
				appearance: appearanceCfg.appearance, fashionStore: appearanceCfg.fashion,
				ctx: ctx, roleID: activeRole.ID, remote: conn.RemoteAddr().String(),
			})
			if saved, ok := appearanceCfg.skins.Load(activeRole.ID); ok {
				player.restoreSkins(saved)
				log.Printf("%s: restored skins role=%d collected=%d mounted=%d", conn.RemoteAddr(), activeRole.ID, len(saved.collected), len(saved.mounted))
			}
			if saved, exists := facultyStore.Load(activeRole.ID); exists {
				player.restoreFaculty(saved, time.Now())
				log.Printf("%s: restored persistent faculty role=%d state=%d style=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyState, player.progress.facultyStyle, player.progress.facultyName)
			} else {
				persistFaculty()
				log.Printf("%s: initialized persistent normal faculty role=%d name=%s", conn.RemoteAddr(), activeRole.ID, player.progress.facultyName)
			}
			// 已学技能装载（Java LoginFlow.loadDbSkills 同口径）：读库失败
			// 或空表只记日志回退 starter 集，不阻塞登录。
			if dbSkills, skillErr := skills.learnedSkills(ctx, activeRole.ID); skillErr != nil {
				log.Printf("%s: load learned skills (skip to starter set) role=%d: %v", conn.RemoteAddr(), activeRole.ID, skillErr)
			} else {
				dbLearnedSkills = dbSkills
				log.Printf("%s: learned skills role=%d db_rows=%d starter=%d merged=%d", conn.RemoteAddr(), activeRole.ID, len(dbSkills), len(starterSkillViews), len(mergeLearnedSkills(dbSkills)))
			}
			// 自定义配置域装载：键位/操作模式必须在 sendPlayerSpawn 之前
			// 进内存——0x0D 档（引擎建对象时即读）内嵌 ShortcutKey(198)/
			// OperateMode(223)。快捷栏基线与个性配置串在 ClientReady 后
			// 经 0x11 重放 / S2C 705 下发。
			if saved, exists := customConfig.keybind.Load(activeRole.ID); exists {
				player.setKeybind(saved)
				log.Printf("%s: restored keybind role=%d bind=%dB scheme1=%dB scheme2=%dB operate_mode=%t", conn.RemoteAddr(), activeRole.ID, len(saved.bind), len(saved.scheme1), len(saved.scheme2), saved.operateModeSet)
			}
			if saved, exists := customConfig.shortcuts.List(activeRole.ID); exists && len(saved) > 0 {
				player.replaceShortcuts(saved)
				shortcutsFromDB = true
				log.Printf("%s: restored %d shortcut rows from role_shortcuts role=%d", conn.RemoteAddr(), len(saved), activeRole.ID)
			}
			if saved, exists := customConfig.customizing.Load(activeRole.ID); exists {
				customizingSess.saved, customizingSess.hasSaved = saved, true
			}
			// 场景服务器在发送任何对象归档之前，必须先定义
			// 可见属性和可见记录两种 schema。参考服务器中的
			// CClassSet::InitTableMessage 将它们构建为 opcode 0x09 和 0x0A。
			// 若缺少 0x09，客户端就没有权威的索引/类型映射，
			// 对象创建会在解码属性时于不同坐标处失败。
			propertyTable := officialPropertyTableFrame()
			if err := link.WriteFrame(propertyTable); err != nil {
				log.Printf("%s: write property table: %v", conn.RemoteAddr(), err)
				return
			}
			// QingGongRec 是玩家记录，而非可见对象属性。当前
			// 客户端在该 schema 完成协商之前拒绝将轻功视为已学习，
			// 即使其能量属性（QingGongPoint/MaxQingGongPoint）
			// 已经存在。
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
			// 三期官方方言：17 视图容器（8 主页 BaseCap=18 + 8 扩展页 +
			// View1 纸娃娃），随后按 DB 扩容锚点行下发扩展槽位。
			for _, frame := range bagViewCreateFrames() {
				if err := link.WriteFrame(frame); err != nil {
					log.Printf("%s: create bag view frame: %v", conn.RemoteAddr(), err)
					return
				}
			}
			bagpacks, packErr := inventory.bagpacks(ctx, activeRole.ID)
			if packErr != nil {
				log.Printf("%s: load bagpack anchors (skip): %v", conn.RemoteAddr(), packErr)
			}
			for _, frame := range itemBagFrames(bagpacks, time.Now()) {
				if err := link.WriteFrame(frame); err != nil {
					log.Printf("%s: add bagpack anchor: %v", conn.RemoteAddr(), err)
					return
				}
			}
			log.Printf("%s: created %d bag view containers with %d expansion anchors", conn.RemoteAddr(), len(bagViewCreateFrames()), len(itemBagFrames(bagpacks, time.Now())))

			if err := world.begin(activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, sceneNPCs); err != nil {
				log.Printf("%s: begin scene generation: %v", conn.RemoteAddr(), err)
				return
			}
			if err := sendEntryScene(link, activeRole.Location.Scene); err != nil {
				log.Printf("%s: write entry scene: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent PlayerEntry opcode=0x0B object=1 scene=%s resource=%s", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)

			// 三期物品下发（DB 模式）：纸娃娃 + 背包成对行 + 武器外观；
			// DB 读失败回退空背包，不阻塞登录（Java LoginFlow 同口径，
			// 但不以其「穿戴为空即整段跳过」为门槛——有背包行即下发）。
			// 读取先于 spawn：已穿衣/裤/鞋模型经 mergeVisual 并进 0x0D 档
			//（Java AppearanceSync.mergeVisual——重登不回落「初始屌丝装」）。
			equips, equipErr := inventory.equips(ctx, activeRole.ID)
			if equipErr != nil {
				log.Printf("%s: load equips (skip inventory): %v", conn.RemoteAddr(), equipErr)
			}
			bagItems, bagErr := inventory.bagItems(ctx, activeRole.ID)
			if bagErr != nil {
				log.Printf("%s: load bag items (skip inventory): %v", conn.RemoteAddr(), bagErr)
			}
			visual = mergeVisual(visual, equips)

			if err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {
				log.Printf("%s: write player spawn chain: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent player 0x0D + 0x1F + authoritative vital 0x10", conn.RemoteAddr())
			if equipErr == nil && bagErr == nil && (len(equips) > 0 || len(bagItems) > 0) {
				frames, frameErr := inventoryFrames(activeRole.ID, visual.sex, equips, bagItems)
				if frameErr != nil {
					log.Printf("%s: encode inventory (skip): %v", conn.RemoteAddr(), frameErr)
				} else {
					for _, frame := range frames {
						if err := link.WriteFrame(frame); err != nil {
							log.Printf("%s: push inventory frame: %v", conn.RemoteAddr(), err)
							return
						}
					}
					log.Printf("%s: pushed inventory equips=%d bagItems=%d role=%d", conn.RemoteAddr(), len(equips), len(bagItems), activeRole.ID)
				}
			}
			// 会话物品模型：登录装载（DB 读失败=空模型；后续 msg30/31/34/36 的权威内存态）。
			player.restoreSessionInventory(visual.sex, equips, bagItems, bagpacks)
			// 登录外观补帧（Java pushEquipAppearance 同门槛）：帽/身份装/
			// 时装任一槽占用或显示类型非默认江湖装才推 0x10，否则零帧
			//（0x0D 档已带装备合并视觉，帧序对齐 golden）。
			if err := pushEquipAppearance(link, player, equips); err != nil {
				log.Printf("%s: push equip appearance: %v", conn.RemoteAddr(), err)
			}
			// 现在就注册完整场景目录中初始的邻近切片，
			// 但在客户端就绪屏障之前不将其发送上线。
			// 加载场景会清空客户端的原生对象表。因此在
			// ClientReady 之前发送的对象会闪现一次随即被移除。
			world.attachMember(player, activeRole.Location.Position, visual)
			world.refreshVisibility(sceneNPCs, activeRole.Location.Position)
			nearbyCount := countNPCsWithin(sceneNPCs, activeRole.Location.Position, npcRadius)
			if err := machine.RoleChosen(); err != nil {
				log.Printf("%s: complete choose-role transition: %v", conn.RemoteAddr(), err)
				return
			}
			runtime = &sceneRuntime{
				conn: link, world: world, machine: machine, registry: sceneRegistry,
				staticNPCs: staticNPCs, radius: npcRadius,
				player: player, activeRole: activeRole, npcCatalog: sceneNPCs,
			}
			gmSession = gm.attach(machine.ID(), conn.RemoteAddr().String(), activeRole, roleFaction(activeRole.Appearance.Values), starterSilver, player.skillSP())
			log.Printf("%s: registered %d nearby NPC objects from catalog=%d; waiting for ClientReady", conn.RemoteAddr(), nearbyCount, len(sceneNPCs))

		} else if plain[0] == 0x09 {
			// 实机 0x0C 场景替换会在旧 stage 尚在收尾时
			// 产生一个较早的 0x0A。随后的 0x09 才是目标场景
			// 原生对象表能承受写入的第一个时点。
			// 在此处落实被延迟的玩家/NPC 生成，
			// 而不是把那个更早的活动包当作就绪信号。
			if runtime != nil && runtime.awaitingStableReentryReady {
				if err := world.clientReady(); err != nil {
					log.Printf("%s: arm stable re-entry barrier: %v", conn.RemoteAddr(), err)
					return
				}
				// 目标 stage_main 在打开场景之前需要它的 1-1 玩家对象，
				// 因此 0x0D 已在 EntryScene 之后发送。只把
				// 0x1F ServerLocation（原生视觉移动）延迟到这个目标
				// ClientReady 边界。
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
				// ServerExitScene 会清除原生玩家对象，包括其挂载的记录和
				// View 46 内容。各 schema 仍为连接级作用域，
				// 但每次完成 0x0C 重入后，必须为新创建的 1-1 玩家对象
				// 恢复这些行。
				if player != nil {
					if grantErr := grantStarterQingGong(link); grantErr != nil {
						log.Printf("%s: restore qinggong after stable re-entry: %v", conn.RemoteAddr(), grantErr)
						return
					}
					if shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {
						log.Printf("%s: restore shortcuts after stable re-entry: %v", conn.RemoteAddr(), shortcutErr)
						return
					}
					// 键位域属性（198/199/200/223）挂玩家对象属性视图，
					// 0x0C 已随对象清空——0x0D 档内嵌之外还须单属性帧
					// 重推一次（Java completeSceneReentry 同款重臂）。
					// 705 是连接级配置对象，不随切图重置，无需重发。
					if keybindErr := pushKeybindFrames(link, player); keybindErr != nil {
						log.Printf("%s: restore keybind after stable re-entry: %v", conn.RemoteAddr(), keybindErr)
						return
					}
					// 外观域同样挂玩家对象：风物志/时装记录行 + ShowEquipType
					// 属性视图 + 最终模型 0x10 全部随 0x0C 清空，重入后整推
					//（记录行镜像计数先归零——客户端已无旧行可删）。
					player.resetSkinRowMirrors()
					if skinsErr := pushSavedSkins(link, player); skinsErr != nil {
						log.Printf("%s: restore skins after stable re-entry: %v", conn.RemoteAddr(), skinsErr)
						return
					}
					if fashionErr := pushFashionSuitRows(link, player.fashionSuitSnapshot()); fashionErr != nil {
						log.Printf("%s: restore fashion rows after stable re-entry: %v", conn.RemoteAddr(), fashionErr)
						return
					}
					if appearanceErr := pushAppearanceRefresh(link, player); appearanceErr != nil {
						log.Printf("%s: restore appearance after stable re-entry: %v", conn.RemoteAddr(), appearanceErr)
						return
					}
					qingGongGranted = true
					log.Printf("%s: restored %d QingGongRec rows and %d View=46 items after 0x0C re-entry", conn.RemoteAddr(), len(starterQingGongIDs), len(starterQingGongIDs)+len(starterQingGongSkillIDs))
					// ServerExitScene 会连同玩家对象一起移除 View 43。
					// neigongGranted 只描述旧场景，因此绝不能让它
					// 抑制新原生玩家对象的这次恢复。
					// 若缺少 View 43 条目，faculty.lua 能收到 1006/1
					// 却无法将 FacultyName 解析为 wuxue 对象，
					// 从而导致普通开始按钮保持禁用（不会发出 1006/11）。
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
				runtime.awaitingStableReentryReady = false
				log.Printf("%s: target scene confirmed by 0x09; activated deferred target NPCs", conn.RemoteAddr())
				continue
			}
			explicitSceneReady = true
			log.Printf("%s: observed explicit stage_main ClientReady (0x09)", conn.RemoteAddr())
			// 在当前客户端上，初始 0x0A 会在原生 stage 构造完成之前到达，
			// 而后者稍后还会清除已学习技能状态。
			// 只有显式的 stage_main 0x09 对 QingGongRec/View=46 是安全的。
			if player != nil && !qingGongGranted {
				if grantErr := grantStarterQingGong(link); grantErr != nil {
					log.Printf("%s: grant stable starter qinggong: %v", conn.RemoteAddr(), grantErr)
					return
				}
				qingGongGranted = true
				log.Printf("%s: granted %d QingGongRec rows and %d View=46 items after explicit ClientReady", conn.RemoteAddr(), len(starterQingGongIDs), len(starterQingGongIDs)+len(starterQingGongSkillIDs))
			}
			// 自定义配置域 ClientReady 推送（Java grantOncePerConnectionRows
			// 同窗口）：个性配置 705 必须赶在客户端登录期上传 107（默认键位
			// 转储）之前；键位域四属性挂玩家对象属性视图，0x0C 切图清对象
			// 即丢、重入时由 stable re-entry 分支重推。无档零帧。
			if player != nil && !customizingPushed {
				customizingPushed = true
				if pushErr := pushSavedCustomizing(link, customizingSess.saved); pushErr != nil {
					log.Printf("%s: push saved customizing: %v", conn.RemoteAddr(), pushErr)
					return
				} else if customizingSess.hasSaved {
					log.Printf("%s: pushed saved customizing config %d bytes", conn.RemoteAddr(), len(customizingSess.saved))
				}
			}
			if player != nil && !keybindPushed {
				keybindPushed = true
				if pushErr := pushKeybindFrames(link, player); pushErr != nil {
					log.Printf("%s: push saved keybind: %v", conn.RemoteAddr(), pushErr)
					return
				}
			}
			// 外观域 ClientReady 推送（Java ClientReady 链同窗口：键位后、
			// 时装页签前）：风物志收录位图 + 挂载行 + 显示开关（空态零帧，
			// 客户端保持未收录）；FashionSuitRec 行（已激活套装 configId，
			// 数值列零填充）。两者挂玩家对象记录视图，0x0C 重入随对象清空
			// 由 stable re-entry 分支重推。
			if player != nil && !player.skinsRowsPushed() {
				player.setSkinsRowsPushed()
				if pushErr := pushSavedSkins(link, player); pushErr != nil {
					log.Printf("%s: push saved skins: %v", conn.RemoteAddr(), pushErr)
					return
				}
			}
			if player != nil && !player.fashionRowsPushed() {
				player.setFashionRowsPushed()
				if pushErr := pushFashionSuitRows(link, player.fashionSuitSnapshot()); pushErr != nil {
					log.Printf("%s: push fashion suit rows: %v", conn.RemoteAddr(), pushErr)
					return
				}
			}
			if player != nil && !shortcutRowsSent {
				shortcutRowsSent = true
				// DB 无玩家编辑基线时补 starter 打坐条目；有基线则以玩家
				// 编辑结果为准（删除即删除，不复活）。
				if !shortcutsFromDB && ensureSitcrossShortcut(player) {
					log.Printf("%s: granted native sitcross shortcut index=%d kind=%s id=%s", conn.RemoteAddr(), sitcrossShortcutIndex, sitcrossShortcutKind, sitcrossShortcutID)
				}
				if shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {
					log.Printf("%s: replay shortcut rows after ClientReady: %v", conn.RemoteAddr(), shortcutErr)
					return
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
			// 当前正式服流量可能在场景已经激活后重复发送 ClientReady
			// （例如在较晚的 UI/stage 回调之后）。
			// 这不是创建新场景世代的请求；拒绝它会拆掉一个
			// 本来健康的连接。
			if machine.Phase() == session.PhaseInWorld {
				log.Printf("%s: ignored duplicate ClientReady in active scene", conn.RemoteAddr())
				continue
			}
			// ClientReady 是原生场景及其对象管理器稳定下来的边界，
			// 在此处校正玩家的视口。
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
				// 这个包紧随 0x0C/0x0B 之后发出，此时目标地形尚未
				// 加载完成。它属于旧 stage，绝不能用来打开
				// 新场景的对象屏障。
				log.Printf("%s: ignored early 0x0A while target scene is loading", conn.RemoteAddr())
				continue
			}
			// 当前正式服客户端有时会省略旧抓包中称为
			// ClientReady 的单字节 0x09，而在进入链之后
			// 直接发送第一个 0x0A。将该观测到的包视为显式
			// 兼容屏障，而不是在会话仍处于
			// PhaseEnteringWorld 时尝试重入。
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
			// 请求的实机切图优先于旧场景的实体化。
			// 在 ExitScene 之前立刻发送数百个旧 NPC 对象
			// 既浪费带宽，又会与原生拆除流程竞态。
			// 绝不从兼容 0x0A 启动实机切图。
			// 上面具体的 stage_main ClientReady 只在当前场景的
			// 玩家/归档处理落定之后才会发出。
			switchAfterActivity := switchDestination != nil && runtime != nil && explicitSceneReady && (machine.Phase() == session.PhaseAwaitingActivity || machine.Phase() == session.PhaseInWorld)
			if !switchAfterActivity {
				if err := world.clientActivity(); err != nil {
					log.Printf("%s: activate ready scene objects: %v", conn.RemoteAddr(), err)
					return
				}
			}
			if err := machine.Activity(); err != nil {
				log.Printf("%s: complete client-activity transition: %v", conn.RemoteAddr(), err)
				return
			}
			if demoDamage > 0 && player != nil && player.applyDamage(demoDamage) {
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
				switchDestination = nil // 每个连接只执行一次显式测试切图
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
				changed := world.updateMemberPosition(location.Position, sceneNPCs)
				nearbyCount := countNPCsWithin(sceneNPCs, location.Position, npcRadius)
				if changed {
					if reconcileErr := world.reconcileViewport(); reconcileErr != nil {
						log.Printf("%s: reconcile moving NPC viewport: %v", conn.RemoteAddr(), reconcileErr)
						return
					}
					log.Printf("%s: NPC viewport moved center=(%.2f,%.2f) active=%d", conn.RemoteAddr(), x, z, nearbyCount)
				}
				// 正式服 Door 对象在角色越过其配置的
				// MoveTarget/SpringRange 时激活，而不仅在手动点击之后。在
				// 本次移动更新被服务器接受后检测该边界，
				// 再复用完全相同的传送门执行/持久化路径。
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
			// 当前客户端把 GameVisual:CustomSend 放在其 0x0A
			// 活动信封内。尤其是真实的轻功动作在此以
			// TVarList(216, qinggong_id, begin_or_end) 到达，而不是独立的
			// 顶层 0x1E 帧。只解析完整的类型化列表，使其他
			// 活动/移动包保持原有的生命周期行为。
			if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 && (custom.Values[0].Int32 == clientCustomSetShortcut || custom.Values[0].Int32 == clientCustomRemoveShortcut || custom.Values[0].Int32 == clientCustomSitCross || custom.Values[0].Int32 == clientCustomSaveCustomizing || custom.Values[0].Int32 == clientCustomKeybindUpload || custom.Values[0].Int32 == clientCustomSetOperateMode || custom.Values[0].Int32 == 211 || custom.Values[0].Int32 == 215 || custom.Values[0].Int32 == 216 || custom.Values[0].Int32 == 217 || custom.Values[0].Int32 == clientCustomActiveParry || custom.Values[0].Int32 == 1006 || custom.Values[0].Int32 == clientCustomSelectTarget || custom.Values[0].Int32 == clientCustomBlockState || custom.Values[0].Int32 == clientCustomMovementUpload || custom.Values[0].Int32 == clientCustomCastDashLanding || custom.Values[0].Int32 == clientCustomChangeWarning || custom.Values[0].Int32 == clientCustomMoveItem || custom.Values[0].Int32 == clientCustomUseItem || custom.Values[0].Int32 == clientCustomDeleteItem || custom.Values[0].Int32 == clientCustomArrangeBag || custom.Values[0].Int32 == clientCustomShowEquipType || custom.Values[0].Int32 == clientCustomFashionSuit || custom.Values[0].Int32 == clientCustomWardrobeCard) {
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
					if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.shortcuts, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity shortcut: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomSaveCustomizing:
					if _, handleErr := handleCustomizingCustom(link, player, custom, conn.RemoteAddr().String(), &customizingSess, customConfig.customizing, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity customizing: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomKeybindUpload:
					if _, handleErr := handleKeybindCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity keybind: %v", conn.RemoteAddr(), handleErr)
						return
					}
				case clientCustomSetOperateMode:
					if _, handleErr := handleOperateModeCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity operate mode: %v", conn.RemoteAddr(), handleErr)
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
				case clientCustomSelectTarget:
					if handleErr := handleSelectTargetCustom(world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle select target: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomBlockState:
					if handleErr := handleBlockStateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle block state: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomChangeWarning:
					if _, handleErr := handleChangeWarningCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle change warning: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomCastDashLanding:
					if _, handleErr := handleCastDashLanding(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle dash landing: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomMovementUpload:
					// 高频移动上报：无冲锋在途时静默返回，不记日志。
					if _, handleErr := handleMovementUpload(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle movement upload: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomMoveItem:
					if _, handleErr := handleMoveItemCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity move item: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomUseItem:
					if _, handleErr := handleBodyItemUse(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity use item: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomDeleteItem:
					if _, handleErr := handleDeleteItemCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity delete item: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomArrangeBag:
					if _, handleErr := handleArrangeItemCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity arrange bag: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomShowEquipType:
					if _, handleErr := handleShowEquipTypeCustom(newAppearanceSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity show equip type: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomFashionSuit:
					if _, handleErr := handleFashionSuitCustom(newAppearanceSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity fashion suit: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomWardrobeCard:
					if _, handleErr := handleWardrobeCardCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity wardrobe card: %v", conn.RemoteAddr(), handleErr)
					}
				case 215, 1006:
					if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
						log.Printf("%s: handle activity inner-power msg=%d: %v", conn.RemoteAddr(), custom.Values[0].Int32, handleErr)
						return
					}
					persistFaculty()
				}
			}
		} else if plain[0] == 0x18 {
			// 客户端 ~1s 一发的 12B ping。fork（49840 抓包 326 对
			// 逐帧验证）回 0x38 纯回显：整帧拷贝、仅换 opcode。
			// 无回包时客户端 RTT 配对恒超时，时序补偿劣化
			//（真机：出招特效延迟出现 / HUD 延时封顶，见
			// docs/combat-traffic-compare.md D11）。
			if err := link.WriteFrame(heartbeatReply(plain)); err != nil {
				log.Printf("%s: heartbeat reply: %v", conn.RemoteAddr(), err)
				return
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
			// 门是直接的对象交互，而非 NPC 菜单。目标
			// 路线只来自已安装的 DoorPackageID/door_data.ini
			// 或 GotoDoor PlayerTransferPosition 数据；未还原的剧情/克隆
			// 门刻意回落到普通选择处理。
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
			// msg15 移动上报是引擎直发高频帧（Java 同款静默），不进逐条日志。
			if custom.Values[0].Int32 != clientCustomMovementUpload {
				log.Printf("%s: client CustomSend opcode=0x%02X msg_id=%d args=[%s]", conn.RemoteAddr(), custom.Opcode, custom.Values[0].Int32, strings.Join(arguments, ", "))
			}
			switch custom.Values[0].Int32 {
			case clientCustomUseSkill:
				if _, handleErr := handleSkillCustom(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle skill: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomSetShortcut, clientCustomRemoveShortcut:
				if _, handleErr := handleShortcutCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.shortcuts, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle shortcut: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomSaveCustomizing:
				if _, handleErr := handleCustomizingCustom(link, player, custom, conn.RemoteAddr().String(), &customizingSess, customConfig.customizing, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle customizing: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomKeybindUpload:
				if _, handleErr := handleKeybindCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle keybind: %v", conn.RemoteAddr(), handleErr)
					return
				}
			case clientCustomSetOperateMode:
				if _, handleErr := handleOperateModeCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle operate mode: %v", conn.RemoteAddr(), handleErr)
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
			case clientCustomSelectTarget:
				if handleErr := handleSelectTargetCustom(world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle select target: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomBlockState:
				if handleErr := handleBlockStateCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle block state: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomChangeWarning:
				if _, handleErr := handleChangeWarningCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle change warning: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomCastDashLanding:
				if _, handleErr := handleCastDashLanding(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle dash landing: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomMovementUpload:
				// 高频移动上报：无冲锋在途时静默返回，不记日志。
				if _, handleErr := handleMovementUpload(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle movement upload: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomMoveItem:
				if _, handleErr := handleMoveItemCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle move item: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomUseItem:
				if _, handleErr := handleBodyItemUse(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle use item: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomDeleteItem:
				if _, handleErr := handleDeleteItemCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle delete item: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomArrangeBag:
				if _, handleErr := handleArrangeItemCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle arrange bag: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomShowEquipType:
				if _, handleErr := handleShowEquipTypeCustom(newAppearanceSession(), custom); handleErr != nil {
					log.Printf("%s: handle show equip type: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomFashionSuit:
				if _, handleErr := handleFashionSuitCustom(newAppearanceSession(), custom); handleErr != nil {
					log.Printf("%s: handle fashion suit: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomWardrobeCard:
				if _, handleErr := handleWardrobeCardCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle wardrobe card: %v", conn.RemoteAddr(), handleErr)
				}
			case 215, 1006: // 内功穿戴 / faculty 状态机
				if _, handleErr := handlePlayerProgressCustom(link, player, custom); handleErr != nil {
					log.Printf("%s: handle inner-power msg=%d: %v", conn.RemoteAddr(), custom.Values[0].Int32, handleErr)
					return
				}
				persistFaculty()
			}
		}
	}
}

// handleQingGongCustom 处理一个经核实的当前客户端 216 请求。抓包
// 0x0A 形态中的第三个值是开始/结束标记：连续型
// 轻功在启动时发 1、停止时发 0。两次都扣费会让
// 一次按住/释放消耗双倍，因此只有开始标记才扣减
// 权威的点数资源。
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
	cost, accepted := player.useQingGong(definition.Consume, now)
	if !accepted {
		state := player.stateSnapshot()
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
	state := player.stateSnapshot()
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
	// mappath 巡逻绑定结果（bindPatrolRoutes 在目录加载后写入）：
	// schedule 非 nil 即在巡（spawn 坐标已被吸附改写到最近路点）。
	route         *patrolRoute
	routeStartIdx int
	schedule      *patrolSchedule
}

func countNPCsWithin(catalog []npcSpawn, position role.Position, radius float32) int {
	radiusSquared := radius * radius
	count := 0
	for i := range catalog {
		if !npcSpawnEligible(catalog[i]) {
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

func npcAddObject(objectID uint32, npc npcSpawn, probeIndex int) ([]byte, error) {
	msg := make([]byte, 0x3f, 256)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	// 该字段是当前场景/所有者标识，而非视口槽位。
	// 玩家入场为该连接确立了场景标识 1。
	binary.LittleEndian.PutUint32(msg[5:], 1)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(npc.x))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(npc.y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(npc.z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(npc.orient))
	// 偏移 0x09..0x3c 是当前 FxNet2::ServerAddObject 复制的
	// 13 值变换/移动头部。对象类别属于下方协商的
	// Type 属性；不要把猜测的类别塞进尾部。
	_ = probeIndex
	properties, err := npc.resolved.OrderedProperties()
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
	msg[1] = 0    // 场景对象，而非视图对象
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
	msg[1] = 0    // 场景对象，而非视图对象
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

// serverLocation 编码 FxNet2 现代 ServerLocation 消息。与
// ServerObjectProperty 更新不同，opcode 0x1F 更新对象原生的 Posi
// 和 Dest 变换并触发 on_location，这正是游戏客户端
// 放置一个已创建的异步 Actor2 可视对象所用的路径。
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
	msg[1] = 0    // 场景对象，而非视图对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	return appendStringProperty(msg, propertyIndex, value)
}

// sceneObjectObjectProperty 更新一个协商类型为
// 原生八字节对象引用的属性。FxNet2 将场景对象标识为
// 低字对象 ID 后跟高字所有者 ID。
func sceneObjectObjectProperty(objectID, ownerID, propertyIndex, targetID, targetOwnerID uint32) []byte {
	msg := make([]byte, 12, 22)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // 场景对象，而非视图对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	return appendObjectProperty(msg, uint16(propertyIndex), targetID, targetOwnerID)
}

// officialPropertyTableFrame 逐行透传内嵌的官方 0x09 属性表
// (internal/clientdata/official_property_table.txt,2370 项)。
// 帧体 = [0x09][u16 count](name\0 u8 type)*;客户端按位置绑定索引,
// 因此表内容必须与官方方言逐字节一致,不做任何删改。
// 来源:F 盘新版服务端 exe 抓包提取(tools/export_official_table.py),
// 与 Java 参考服 official_property_table.txt 三方比对 0 差异。
func officialPropertyTableFrame() []byte {
	entries := clientdata.OfficialPropertyEntries()
	msg := make([]byte, 3, 3+64*1024)
	msg[0] = 0x09
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(entries)))
	for _, entry := range entries {
		msg = append(msg, entry.Name...)
		msg = append(msg, 0, byte(entry.TypeCode))
	}
	return msg
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
