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
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/gmapi"
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/combatcalc"
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
	gmInternalListen := flag.String("gm-internal-listen", "127.0.0.1:19063", "internal GM API listen address for the standalone gm-panel process (loopback only; requires -gm-internal-token, empty disables)")
	gmInternalToken := flag.String("gm-internal-token", os.Getenv("NINEYIN_GM_TOKEN"), "bearer token guarding the internal GM API (defaults to NINEYIN_GM_TOKEN)")
	logFile := flag.String("log-file", runtimeProjectPath("artifacts", "local", "logs", "protocol-probe-live.log"), "append runtime log to this local file")
	npcFuncPath := flag.String("npc-func", defaultLegacyNPCFuncPath, "Npc_Func.xml used to build per-NPC service menus")
	scriptsPath := flag.String("scripts", runtimeProjectPath("scripts", "host"), "server-side event Lua scripts directory (game.on contract; empty disables)")
	captureDir := flag.String("capture-dir", runtimeProjectPath("captures", "local"), "dump raw wire bytes of every session to <dir>/<ip-pid>_c2s.bin and _s2c.bin (empty disables)")
	neigongScriptsPath := flag.String("neigong-scripts", runtimeProjectPath("scripts", "module", "neigong"), "neigong effect module scripts directory (buf_ng_*.lua; empty disables)")
	combatScriptsPath := flag.String("combat-scripts", runtimeProjectPath("scripts", "module", "combat"), "combat per-hit formula engine scripts directory (formula.lua; empty disables, damage falls back to legacy dice)")
	switchScene := flag.String("switch-scene", "", "test-only live destination: config,resource,x,y,z,orient (empty disables it)")
	npcAuditAll := flag.Bool("npc-audit-all", false, "inventory every modern NPC creator scene and exit")
	npcServiceAuditScene := flag.String("npc-service-audit-scene", "", "write effective NPC service audit for config,resource,x,y,z,orient and exit")
	qingGongDefinePath := flag.String("qinggong-define", filepath.Join(defaultModernShareRoot, "skill", "qinggong", "qgdefine.ini"), "current-client QGDefine.ini used to validate and charge 216")
	respawnInterval := flag.Int("respawn-interval", 30, "batch monster respawn interval in seconds: every interval all DEAD combat NPCs revive at their spawn points, alive ones untouched (0 disables batching and falls back to per-NPC RefreshTime)")
	corpseTTL := flag.Int("corpse-ttl", 10, "corpse removal delay in seconds: dead monster corpses are removed (0x0E) after this long and rematerialize on respawn (0 keeps corpses until revive)")
	lootBoxTTLFlag := flag.Int("lootbox-ttl", 300, "drop loot box lifetime in seconds: unclaimed loot boxes disappear (0x0E) after this long, changes apply to live boxes on the next tick (0 keeps boxes until fully looted; official attacknpc DestroyTime2=60)")
	autosaveSecs := flag.Int("autosave-interval", 300, "periodic online snapshot flush in seconds: every interval each online player's faculty/jingmai snapshots are upserted to MySQL (guards against hard kills; 0 disables)")
	livepointVIPFlag := flag.Bool("livepoint-vip", false, "livepoint conversion tier: true = everyone converts losslessly (VIP), false = normal 70% to faculty + 30% to hold-faculty (Java -Dnineyin.livepoint.vip)")
	livepointPriceFlag := flag.Int("livepoint-price", 500, "hold-faculty redemption price: N gold converts N*price potential faculty (Java -Dnineyin.livepoint.price)")
	facultySeedFlag := flag.Int("faculty-seed", int(initialFaculty), "new-character faculty seed; existing roles keep their snapshot value (Java -Dnineyin.faculty.seed)")
	flag.Parse()
	livepointVIP = *livepointVIPFlag
	livepointPrice = int32(*livepointPriceFlag)
	facultySeed = int32(*facultySeedFlag)
	setNpcRespawnBatchInterval(time.Duration(*respawnInterval) * time.Second)
	setNpcCorpseTTL(time.Duration(*corpseTTL) * time.Second)
	setLootBoxTTL(time.Duration(*lootBoxTTLFlag) * time.Second)
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
	// 每击公式引擎（scripts/module/combat/formula.lua + formula.ini 注入；
	// 2026-11-08 数值重写）：装载失败/禁用只影响新数值路径，rollSkillHit
	// 自动回退 legacy 面板×倍率掷骰，战斗链不受阻。
	if *combatScriptsPath != "" {
		specs, specErr := loadCombatFormulaSpecs()
		if specErr != nil {
			log.Printf("formula.ini 不可用（每击公式路径禁用）：%v", specErr)
		} else {
			combatFormulaSpecs = specs
			combatEngine := combatcalc.New(*combatScriptsPath, log.Printf)
			installed, engineErr := combatEngine.Load(specs)
			if engineErr != nil {
				log.Printf("公式引擎不可用（伤害走 legacy 掷骰）：%v", engineErr)
			} else {
				installedCombatCalc = combatEngine
				_, skipped := combatEngine.Loaded()
				note := combatEngine.SkipSummary()
				if len(note) > 160 {
					note = note[:160] + "…"
				}
				log.Printf("公式引擎 ready formulas=%d skipped=%d path=%s skip=%s", installed, skipped, *combatScriptsPath, note)
			}
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
	// 经脉目录（面板「经脉周天设置」下拉）：共享库 game_meridians 直查
	//（Java GmCatalogDb.meridians 同款 SQL）；缺表/缺库回落包级目录。
	gm.meridianRows = func() []gmMeridianRow {
		if store.db == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := store.db.QueryContext(ctx,
			`SELECT meridian_id, name, max_level FROM game_meridians ORDER BY meridian_id`)
		if err != nil {
			log.Printf("meridian catalog lookup: %v", err)
			return nil
		}
		defer rows.Close()
		catalog := make([]gmMeridianRow, 0, 35)
		for rows.Next() {
			var row gmMeridianRow
			if err := rows.Scan(&row.ID, &row.Name, &row.DefaultMax); err != nil {
				log.Printf("meridian catalog scan: %v", err)
				return nil
			}
			catalog = append(catalog, row)
		}
		return catalog
	}
	if *gmListen != "" {
		// internal API 未开（无 token）时传空串：登记册没有“本面板所连”
		// 卡片，但探活其它已登记服务器仍然可用。
		manageBound := *gmInternalListen
		if *gmInternalToken == "" {
			manageBound = ""
		}
		go serveGM(*gmListen, gm, manageBound)
	}
	// 独立面板进程（gm-panel.exe）经 internal API 访问同一个 gmHub。
	// 刻意不默认启用：需要显式提供 token（flag 或 NINEYIN_GM_TOKEN），
	// 否则按设计文档 §5.2 fail-soft——记日志后不开监听，游戏服照常跑。
	if *gmInternalListen != "" {
		if *gmInternalToken == "" {
			log.Printf("GM internal API disabled: -gm-internal-token (or NINEYIN_GM_TOKEN) is empty")
		} else {
			go serveGMInternal(*gmInternalListen, *gmInternalToken, gm, gmHealthInfo{
				ListenAddr: *addr,
				ServerRoot: runtimeProjectPath("."),
			})
		}
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	// 镇派兵器/宝物/娃娃生成目录预热（equipment/proppack/varprop/
	// randomequiprule 全表一次性装载；失败只降级对应功能并记日志，
	// 首次登录不再付装载延迟）。
	if catalog := randomEquipCatalogOf(); catalog.err != nil {
		log.Printf("random equip catalog degraded: %v", catalog.err)
	} else {
		log.Printf("random equip catalog ready items=%d packs=%d", len(catalog.itemFields), len(catalog.packRows))
	}
	// 自动存盘周期装进共享原子值：GM 面板 set_autosave_interval 运行中
	// 可改，重启回落本 flag 缺省。
	setAutosaveInterval(time.Duration(*autosaveSecs) * time.Second)
	log.Printf("protocol probe listening on %s", *addr)
	// 关服存盘：收到 Ctrl+C 时 Go 默认直接终止进程，各连接 defer 的
	// persistFaculty/persistJingMai 快照不会执行。这里监听信号，停止收新
	// 连接并主动断开全部在线连接——断线会驱动 handle 沿"玩家正常下线"
	// 同一路径退出并落库（内功/经脉快照、场景收尾），drain 完成后再由
	// defer store.Close() 关库退出。
	// 注意：Windows 下直接点控制台窗口的 X 或 taskkill /F 不给进程发
	// 信号，只有 Ctrl+C（CTRL_BREAK 等同 SIGTERM）能走到这条存盘路径。
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	closing := make(chan struct{})
	var sessions sync.WaitGroup
	liveMu := new(sync.Mutex)
	liveConns := make(map[net.Conn]struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-closing:
					return
				default:
				}
				log.Print(err)
				continue
			}
			liveMu.Lock()
			liveConns[conn] = struct{}{}
			liveMu.Unlock()
			sessions.Add(1)
			go func() {
				defer sessions.Done()
				defer func() {
					liveMu.Lock()
					delete(liveConns, conn)
					liveMu.Unlock()
				}()
				handle(conn, store, facultyStore, inventoryStore, skillStore, customConfig, appearanceCfg, sceneRegistry, staticNPCs, npcFuncs, qingGongCatalog, float32(*npcRadius), int32(*demoDamage), int32(*starterSilver), switchDestination, gm, scriptHost, neigongEffects, *captureDir)
			}()
		}
	}()
	sig := <-shutdown
	log.Printf("received %v: shutting down and flushing online player snapshots", sig)
	close(closing)
	_ = ln.Close()
	if n := closeLiveSessions(liveMu, liveConns); n > 0 {
		log.Printf("closed %d live session(s); waiting up to %s for snapshot flush", n, shutdownDrainTimeout)
	}
	// ln.Close 与并发 Accept 返回之间存在极小窗口：个别连接可能刚被
	// 接受还没来得及入表。稍等后补扫一遍，避免 drain 空等超时。
	time.Sleep(50 * time.Millisecond)
	if n := closeLiveSessions(liveMu, liveConns); n > 0 {
		log.Printf("closed %d late session(s)", n)
	}
	if !waitGroupTimeout(&sessions, shutdownDrainTimeout) {
		log.Printf("session drain timed out after %s; some online snapshots may be lost", shutdownDrainTimeout)
	}
	log.Printf("shutdown complete")
}

// shutdownDrainTimeout 是关服时等待全部在线连接走完"下线落库"路径的上限。
const shutdownDrainTimeout = 15 * time.Second

// autosaveIntervalNanos 是进程级自动存盘周期（GM 面板 set_autosave_interval
// 可调；初值来自 -autosave-interval）。各在线会话的 1s tick 读它判定到期，
// 改动即刻对全部在线玩家生效；0=关闭（断线与 Ctrl+C 关服路径不受影响）。
// 与 npc_refresh.go 的批量刷新间隔同款：包级原子 + getter/setter，任意
// 会话循环可写，无锁。
var autosaveIntervalNanos atomic.Int64

func autosaveInterval() time.Duration { return time.Duration(autosaveIntervalNanos.Load()) }

func setAutosaveInterval(interval time.Duration) { autosaveIntervalNanos.Store(int64(interval)) }

// closeLiveSessions 关闭注册表中当前全部在线连接并清出注册表，返回关闭数。
func closeLiveSessions(mu *sync.Mutex, conns map[net.Conn]struct{}) int {
	mu.Lock()
	defer mu.Unlock()
	closed := 0
	for conn := range conns {
		_ = conn.Close()
		delete(conns, conn)
		closed++
	}
	return closed
}

// waitGroupTimeout 等待 WaitGroup 清零，超时返回 false。
func waitGroupTimeout(group *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
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
	// msg294 对钟帧 1s 一发（Java LoginFlow 1s scheduler、52905 抓包
	// 747 条计数器恒 +1000 实证）：cooldown_rec 的 start/end 是绝对
	// unix 毫秒，客户端靠它对齐本机时钟偏移；随连接生命周期启停。
	timeSyncStop := make(chan struct{})
	timeSyncStart := time.Now()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-timeSyncStop:
				return
			case now := <-ticker.C:
				uptime := int32(now.Sub(timeSyncStart).Milliseconds())
				frame, err := timeSyncFrame(uptime, now)
				if err != nil {
					log.Printf("%s: encode time sync: %v", conn.RemoteAddr(), err)
					continue
				}
				if err := link.WriteFrame(frame); err != nil {
					return
				}
			}
		}
	}()
	defer close(timeSyncStop)
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
	// P2 域 store（msg70/71 货币、msg295 称号、msg150/161 经脉）：缺表/
	// 缺库一律降级会话态（建表权威在 migrations/，运行时不做 DDL）。
	currencyStore := newCurrencyDBStore(store.db)
	randomPropsStore := newRandomPropsDBStore(store.db)
	titleStore := newTitleDBStore(store.db)
	jingMaiStore := newJingMaiDBStore(store.db)
	jingMaiCatalog, jingMaiCatalogErr := loadJingMaiCatalog(defaultJingMaiINIPath, defaultJingMaiStaticINIPath)
	if jingMaiCatalogErr == nil {
		// 包级缓存刷新（战斗真气入账等无连接上下文路径取 globalJingMaiCatalog）。
		globalJingMaiCatalog = jingMaiCatalog
	}
	if jingMaiCatalogErr != nil {
		log.Printf("%s: jingmai catalog unavailable (msg150/161 degrade): %v", conn.RemoteAddr(), jingMaiCatalogErr)
	}
	persistJingMai := func() {
		if jingMaiStore == nil || selected == nil || player == nil {
			return
		}
		if err := jingMaiStore.Save(selected.ID, player.jingMaiSnapshotOf()); err != nil {
			log.Printf("%s: persist jingmai progress: %v", conn.RemoteAddr(), err)
		}
	}
	defer persistJingMai()
	// 邮件/坐骑域（Java lettersGranted/mountsPushed 同位）：role_letters/
	// role_mounts 快照读档 → ClientReady 整表重放；无存档零帧。
	letterStore := newLetterDBStore(store.db)
	mountStore := newMountDBStore(store.db)
	mountCatalog := loadMountCatalog(defaultToolItemINIPath, defaultItemArtStaticPath, defaultNewRideINIPath)
	sessionLetters := []letterEntry(nil)
	sessionMounts := []mountEntry(nil)
	lettersPushed := false
	mountsPushed := false
	mountSentRows := 0
	pushLetters := func() error {
		lettersPushed = true
		if len(sessionLetters) == 0 {
			return nil
		}
		if err := pushLetterFrames(link, sessionLetters); err != nil {
			return fmt.Errorf("replay RecvLetterRec rows: %w", err)
		}
		log.Printf("%s: replayed %d RecvLetterRec rows", conn.RemoteAddr(), len(sessionLetters))
		return nil
	}
	pushMounts := func() error {
		mountsPushed = true
		rows, err := pushMountFrames(link, sessionMounts, mountCatalog, mountSentRows)
		if err != nil {
			return fmt.Errorf("replay NewRideRec rows: %w", err)
		}
		mountSentRows = rows
		if len(sessionMounts) > 0 {
			log.Printf("%s: pushed saved mounts owned=%d current=%d", conn.RemoteAddr(), len(sessionMounts), mountCurrentPack(sessionMounts))
		}
		return nil
	}
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
	// 九阴志（419 步骤 + data_record 59 事件行）连接级只发一次：章节
	// 门槛的功能开放数据源（native-menu main.go:626/629 同位）。
	jyzPushed := false
	// jyzGameStepSent 414 'gamestep' 收尾帧连接级一次（native #1744）。
	jyzGameStepSent := false
	// SwitchManager 批量开关（0x1E 414 "sync"）连接级只发一次：引擎开关
	// 树不随场景代际清空，0x0C 重入不重放（Java switchSyncPushed 同口径）。
	switchSyncPushed := false
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
			currency: currencyStore, randomProps: randomPropsStore,
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
		learnedRows := mergeLearnedSkills(dbLearnedSkills)
		// View40 行号映射进内存（出招 CanUse 行帧寻址；Java skillRows 同构）。
		player.noteSkillViewRows(learnedRows)
		skillFrames, err := learnedSkillViewFrames(learnedRows)
		if err != nil {
			return fmt.Errorf("encode View=40: %w", err)
		}
		for _, frame := range skillFrames {
			if err := link.WriteFrame(frame); err != nil {
				return fmt.Errorf("write View=40: %w", err)
			}
		}
		// 经脉域（msg150/161）：View45 容器+全行 + 面板属性帧 +
		// active_jingmai_rec 记录行。0x0C 重入随玩家对象清空，本函数
		// 在重入分支同样会被调用，记录行随之重放。
		if err := pushJingMaiFrames(link, player, jingMaiCatalog); err != nil {
			return fmt.Errorf("grant jingmai view: %w", err)
		}
		// 称号域（msg295）：title_rec 拥有行 + 佩戴位 0x10（无档零帧）。
		if err := pushTitleFrames(link, player); err != nil {
			return fmt.Errorf("push title frames: %w", err)
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
		switch command.Action {
		case "set_silver":
			player.setSilver(command.Silver)
			update, updateErr := player.silverUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("碎银设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSilver(command.Silver)
			log.Printf("%s: GM set CapitalType1=%d", conn.RemoteAddr(), command.Silver)
		case "set_sp", "add_sp", "fill_sp":
			var sp int32
			var update []byte
			var updateErr error
			switch command.Action {
			case "set_sp":
				sp, update, updateErr = player.setSP(command.SP)
			case "add_sp":
				sp, update, updateErr = player.addSP(command.SP)
			default:
				sp, update, updateErr = player.setSP(player.rageCap())
			}
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("怒气设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setSP(sp, fmt.Sprintf("怒气设为 %d/%d", sp, player.rageCap()))
			log.Printf("%s: GM %s SP=%d", conn.RemoteAddr(), command.Action, sp)
		case "set_rage_recovery":
			rate := player.setRageRecovery(int32(command.Value))
			gmSession.setRageRecovery(rate, fmt.Sprintf("怒气自动回复设为 %d/秒", rate))
			log.Printf("%s: GM set rage recovery=%d/s", conn.RemoteAddr(), rate)
		case "set_move_speed":
			update, updateErr := player.setMoveSpeed(command.Value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("移动速度设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("移动速度设为 %.2f", command.Value))
			log.Printf("%s: GM set MoveSpeed/RunSpeed=%.2f", conn.RemoteAddr(), command.Value)
		case "set_gravity":
			update, updateErr := player.setGravity(command.Value)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("下落重力设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("下落重力设为 %.2f", command.Value))
			log.Printf("%s: GM set Gravity=%.2f", conn.RemoteAddr(), command.Value)
		case "set_respawn_interval":
			seconds := int32(command.Value)
			setNpcRespawnBatchInterval(time.Duration(seconds) * time.Second)
			if seconds > 0 {
				gmSession.setAction(fmt.Sprintf("批量刷新间隔设为 %d 秒（死了的统一重生，活怪不动）", seconds))
			} else {
				gmSession.setAction("批量刷新已关闭：回到每怪独立 RefreshTime 计时")
			}
			log.Printf("%s: GM set respawn interval=%ds", conn.RemoteAddr(), seconds)
		case "set_corpse_ttl":
			seconds := int32(command.Value)
			setNpcCorpseTTL(time.Duration(seconds) * time.Second)
			if seconds > 0 {
				gmSession.setAction(fmt.Sprintf("尸体清除延迟设为 %d 秒（超时 0x0E，重生时重新物化）", seconds))
			} else {
				gmSession.setAction("尸体清除已关闭：尸体保留到重生")
			}
			log.Printf("%s: GM set corpse ttl=%ds", conn.RemoteAddr(), seconds)
		case "set_lootbox_ttl":
			seconds := int32(command.Value)
			setLootBoxTTL(time.Duration(seconds) * time.Second)
			if seconds > 0 {
				gmSession.setAction(fmt.Sprintf("宝箱存留时间设为 %d 秒（未拾超时消失，对在场宝箱即刻生效）", seconds))
			} else {
				gmSession.setAction("宝箱存留已关闭：宝箱保留到被拾空")
			}
			log.Printf("%s: GM set lootbox ttl=%ds", conn.RemoteAddr(), seconds)
		case "spawn_chest":
			if runtime == nil || runtime.world == nil || runtime.world.run == nil {
				gmSession.setAction("世界宝箱生成失败：角色尚未进入场景")
				return fmt.Errorf("character has not entered a scene")
			}
			position := runtime.activeRole.Location.Position
			transform := worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
			contents := currentDropPacks().roll("", 1, itemUseRandom())
			chestRun := runtime.world.run
			chestRun.mu.Lock()
			if chestRun.closed {
				chestRun.mu.Unlock()
				gmSession.setAction("世界宝箱生成失败：场景已关闭")
				return fmt.Errorf("scene run closed")
			}
			chestID := chestRun.spawnChestEntityAtLocked(transform, contents, "", 0, time.Now())
			chestRun.mu.Unlock()
			chestRun.flushPendingObjectFrames()
			if chestID == 0 {
				gmSession.setAction("世界宝箱生成失败：实体注册未成")
				return fmt.Errorf("chest entity register failed")
			}
			gmSession.setAction(fmt.Sprintf("已在当前位置生成世界宝箱（实体 %d，内含 %d 件，未拾超时消失）", chestID, len(contents)))
			log.Printf("%s: GM spawned world chest entity=%d items=%d", conn.RemoteAddr(), chestID, len(contents))
		case "set_autosave_interval":
			seconds := int32(command.Value)
			setAutosaveInterval(time.Duration(seconds) * time.Second)
			if seconds > 0 {
				gmSession.setAction(fmt.Sprintf("自动存盘间隔设为 %d 秒（在线玩家内功/经脉快照周期落库）", seconds))
			} else {
				gmSession.setAction("自动存盘已关闭：仅断线与 Ctrl+C 关服时存")
			}
			log.Printf("%s: GM set autosave interval=%ds", conn.RemoteAddr(), seconds)
		case "set_faction":
			preset, presetErr := gmapi.FactionPresetByKey(command.Faction)
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
			if switchErr := runtime.SwitchScene(sceneDestination{location: command.Destination}); switchErr != nil {
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
			if teleportErr := runtime.TeleportWithinScene(command.Position); teleportErr != nil {
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
			log.Printf("%s: GM sent same-scene ServerLocation x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), command.Position.X, command.Position.Y, command.Position.Z, command.Position.Orient)
		case "set_showequip_type":
			player.setShowEquipTypePref(int(command.ShowEquipType))
			if saveErr := appearanceCfg.appearance.Save(selected.ID, int(command.ShowEquipType)); saveErr != nil {
				log.Printf("%s: GM persist show equip type role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			if refreshErr := pushAppearanceRefresh(link, player); refreshErr != nil {
				gmSession.setAction("显示类型切换失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction(fmt.Sprintf("外显类型已切为 %s", showEquipTypeName(int(command.ShowEquipType))))
			log.Printf("%s: GM set ShowEquipType=%d", conn.RemoteAddr(), command.ShowEquipType)
		case "set_gold":
			player.setGold(command.Gold)
			update, updateErr := player.goldUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("金币设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setGold(command.Gold)
			log.Printf("%s: GM set CapitalType0=%d", conn.RemoteAddr(), command.Gold)
		case "set_guanyin":
			player.setGuanYin(command.GuanYin)
			update, updateErr := player.guanYinUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("官银设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setGuanYin(command.GuanYin)
			log.Printf("%s: GM set CapitalType2=%d", conn.RemoteAddr(), command.GuanYin)
		case "set_power_level":
			update, updateErr := player.setPowerLevel(int32(command.Value))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("实力等级设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("实力等级设为 %d（%s）", int32(command.Value), roleListTitle(int(command.Value))))
			log.Printf("%s: GM set PowerLevel=%d", conn.RemoteAddr(), int32(command.Value))
		case "set_alignment":
			update, updateErr := player.setAlignment(command.Justice, command.Evil, command.FreeBase)
			if updateErr == nil && update != nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("性格善恶设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("性格设为 善 %d / 恶 %d / 基数 %d", command.Justice, command.Evil, command.FreeBase))
			log.Printf("%s: GM set Justice=%d Evil=%d FreeBase=%d", conn.RemoteAddr(), command.Justice, command.Evil, command.FreeBase)
		case "set_pk_mode":
			update, updateErr := player.setPKMode(command.PkMode)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("PK 模式设置失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("PK 模式设为 %d", command.PkMode))
			log.Printf("%s: GM set PKMode=%d", conn.RemoteAddr(), command.PkMode)
		case "set_vitals":
			update, updateErr := player.setGMVitals(gmVitalsState{
				ene: command.Ene, maxEne: command.MaxEne, resumeEne: command.ResumeEne,
				sat: command.Sat, maxSat: command.MaxSat,
				pkValue: command.PkValue, drunkLevel: command.DrunkLevel,
			})
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("生存属性设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("体力 %d/%d 饱食 %d/%d 杀戮 %d 酒量 %d",
				command.Ene, command.MaxEne, command.Sat, command.MaxSat, command.PkValue, command.DrunkLevel))
			log.Printf("%s: GM set vitals ene=%d/%d sat=%d/%d pk=%d drunk=%d", conn.RemoteAddr(),
				command.Ene, command.MaxEne, command.Sat, command.MaxSat, command.PkValue, command.DrunkLevel)
		case "set_faculty":
			update, updateErr := player.setFacultyValue(int32(command.Value))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("修为设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setFaculty(int32(command.Value))
			gmSession.setAction(fmt.Sprintf("修为设为 %d", int32(command.Value)))
			log.Printf("%s: GM set Faculty=%d", conn.RemoteAddr(), int32(command.Value))
		case "set_five_attributes":
			update, updateErr := player.setBaseAttributes(command.AttrStr, command.AttrSta, command.AttrDex, command.AttrIng, command.AttrSpi)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("五维属性设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("五维设为 臂 %d 身 %d 内 %d 罡 %d 体 %d",
				command.AttrStr, command.AttrSta, command.AttrIng, command.AttrSpi, command.AttrDex))
			log.Printf("%s: GM set base attributes str=%d sta=%d dex=%d ing=%d spi=%d", conn.RemoteAddr(),
				command.AttrStr, command.AttrSta, command.AttrDex, command.AttrIng, command.AttrSpi)
		case "set_fill_speed":
			actual, update, updateErr := player.setFillSpeed(command.FacultyBase, command.FillMultiplier)
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("修炼速度设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("修炼速度设为 %d（%d × %.2f）", actual, command.FacultyBase, command.FillMultiplier))
			log.Printf("%s: GM set FillSpeed=%d base=%d multiplier=%.2f", conn.RemoteAddr(), actual, command.FacultyBase, command.FillMultiplier)
		case "set_act_day_limit":
			display, update, updateErr := player.setActDayLimit(int32(command.Value))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("演武上限设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			if int32(command.Value) <= 0 {
				gmSession.setAction("演武每日上限已解除（不限制）")
			} else {
				gmSession.setAction(fmt.Sprintf("演武每日上限设为 %d 两", display))
			}
			log.Printf("%s: GM set ActDayValueMax=%d", conn.RemoteAddr(), int32(command.Value))
		case "add_livepoint":
			added := player.addLivepoint(command.Points)
			update, updateErr := player.vitalUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("历练发放失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("发放历练 %d 点（入槽 %d 毫历练）", command.Points, added))
			log.Printf("%s: GM add_livepoint points=%d addedMillis=%d", conn.RemoteAddr(), command.Points, added)
		case "set_hold_faculty":
			player.setHoldFaculty(int32(command.Value))
			update, updateErr := player.vitalUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("潜能修为设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("潜能修为设为 %d", int32(command.Value)))
			log.Printf("%s: GM set HoldFaculty=%d", conn.RemoteAddr(), int32(command.Value))
		case "set_live_speed_power":
			player.setLiveSpeedPower(int32(command.Value))
			update, updateErr := player.vitalUpdate()
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("转化加成设置失败：" + updateErr.Error())
				return updateErr
			}
			persistFaculty()
			gmSession.setAction(fmt.Sprintf("历练转化加成设为 %d%%", int32(command.Value)))
			log.Printf("%s: GM set LiveSpeedPower=%d", conn.RemoteAddr(), int32(command.Value))
		case "set_zhenqi_cap":
			// 经脉每日真气硬闸（Java applyZhenqiCap 同构）：角色级设置随
			// role_meridians 快照持久化；ZhenQiDayValue 口径即时变化
			//（开闸显示阈值/关闸回落官方缺省 500，被动/战斗同闸）。
			player.setJingMaiDailyCap(command.Enabled == 1, int32(command.Value))
			persistJingMai()
			frame, updateErr := player.jingMaiPropsFrame()
			if updateErr == nil {
				updateErr = link.WriteFrame(frame)
			}
			if updateErr != nil {
				gmSession.setAction("真气硬闸设置失败：" + updateErr.Error())
				return updateErr
			}
			if command.Enabled == 1 {
				gmSession.setAction(fmt.Sprintf("每日真气硬闸已开启（阈值 %d，今日已获 %d）",
					int32(command.Value), player.jingMaiSnapshotOf().ZhenQi.Used))
			} else {
				gmSession.setAction(fmt.Sprintf("每日真气硬闸已关闭（阈值 %d 已保存，重开沿用；进度条上限回落官方缺省 500）", int32(command.Value)))
			}
			log.Printf("%s: GM set_zhenqi_cap enabled=%d value=%d", conn.RemoteAddr(), command.Enabled, int32(command.Value))
		case "set_jingmai_tier":
			// 经脉周天直设（Java applySetJingmaiTier 同构）：目录收录 +
			// 已学双闸 → 钳 1..目录上限、fill 归零走新周天起点 → View45
			// 升级行帧 + CurJingMai/JingMaiTotalLevel 属性帧 → 落盘。
			def, known := jingMaiCatalog.def(command.JingMaiID)
			if !known {
				gmSession.setAction("周天设置失败：经脉目录无 " + command.JingMaiID)
				return fmt.Errorf("unknown jingmai id %q", command.JingMaiID)
			}
			old, row, slot, set := player.setJingMaiLevel(jingMaiCatalog, command.JingMaiID, command.JingMaiTier)
			if !set {
				gmSession.setAction("周天设置失败：" + command.JingMaiID + " 未学（先在客户端开通）")
				return fmt.Errorf("unlearned jingmai %q", command.JingMaiID)
			}
			tierFrame, tierErr := jingMaiViewUpdateFrame(uint16(slot), row, true)
			if tierErr == nil {
				tierErr = link.WriteFrame(tierFrame)
			}
			if tierErr != nil {
				gmSession.setAction("周天设置失败：" + tierErr.Error())
				return tierErr
			}
			propsFrame, propsErr := player.jingMaiPropsFrame()
			if propsErr == nil {
				propsErr = link.WriteFrame(propsFrame)
			}
			if propsErr != nil {
				gmSession.setAction("经脉属性推送失败：" + propsErr.Error())
				return propsErr
			}
			persistJingMai()
			gmSession.setAction(fmt.Sprintf("%s 周天 %d → %d（上限 %d）", command.JingMaiID, old, row.Level, def.maxLevel))
			log.Printf("%s: GM set_jingmai_tier %s %d -> %d (max %d)", conn.RemoteAddr(), command.JingMaiID, old, row.Level, def.maxLevel)
		case "set_weather":
			preset, presetErr := gmapi.WeatherPresetByKey(command.Weather)
			if presetErr != nil {
				return presetErr
			}
			// Java modernCustomCommandMessage("on_weather_sky", 4, key,
			// fade, "") 同形：0x27 命令名寻址，客户端 ChangeWeatherEffect。
			update, updateErr := serverModernCustomStringMessage("on_weather_sky",
				customInt(4), customString(preset.Key), customInt(command.FadeSec), customString(""))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("天气切换失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("天气已切为 %s（过渡 %d 秒）", preset.Label, command.FadeSec))
			log.Printf("%s: GM set weather=%s fade=%ds", conn.RemoteAddr(), preset.Key, command.FadeSec)
		case "popup_gm_panel":
			// 客户端 custom_handler close_gmcc_form msg_id=2 → 弹出
			// GMCC 聊天面板（Java applyPopupGmPanel 同帧）。
			update, updateErr := serverModernCustomStringMessage("close_gmcc_form", customInt(2))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("GM 面板弹出失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction("已请求客户端弹出 GM 面板")
			log.Printf("%s: GM popup client gm panel", conn.RemoteAddr())
		case "custom_probe":
			// 数值寻址探针（Go 原生补齐，Java 无 GM 侧对应）：
			// 0x27 [msgId, subType]，用于面板预置消息号的客户端行为观察。
			update, updateErr := serverModernCustomIntMessage(command.MsgId, customInt(command.SubType))
			if updateErr == nil {
				updateErr = link.WriteFrame(update)
			}
			if updateErr != nil {
				gmSession.setAction("探针发送失败：" + updateErr.Error())
				return updateErr
			}
			gmSession.setAction(fmt.Sprintf("已发送探针 msgId=%d subType=%d", command.MsgId, command.SubType))
			log.Printf("%s: GM custom probe msgId=%d subType=%d", conn.RemoteAddr(), command.MsgId, command.SubType)
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
				currency: currencyStore, randomProps: randomPropsStore,
				ctx: ctx, roleID: selected.ID, remote: conn.RemoteAddr().String(), now: time.Now,
			}
			entry, known := inventory.lookupGrantable(ctx, command.ItemConfig)
			if !known {
				gmSession.setAction("发放失败：game_items 无此配置 " + command.ItemConfig)
				return fmt.Errorf("unknown item config %q", command.ItemConfig)
			}
			equipType := entry.equipType
			if equipType == "" {
				if stat := equipmentModels.itemStatic(command.ItemConfig); stat != nil {
					equipType = stat.equipType
				}
			}
			randomKind := randomEquipCatalogOf().randomEquipKind(command.ItemConfig)
			if command.WearNow && equipSlotFor(equipType, player.sessionEquips()) > 0 {
				slot := equipSlotFor(equipType, player.sessionEquips())
				worn, wearErr := inventory.upsertEquipSlot(ctx, selected.ID, slot, command.ItemConfig, entry.itemType, equipType)
				if wearErr != nil {
					gmSession.setAction("穿戴发放失败：" + wearErr.Error())
					return wearErr
				}
				player.replaceEquipSlot(worn)
				if framesErr := pushWearFrames(link, selected.ID, player.visualSexOf(), effectiveShowEquipType(player), worn, player.sessionEquips()); framesErr != nil {
					gmSession.setAction("穿戴帧推送失败：" + framesErr.Error())
					return framesErr
				}
				// 宝物/娃娃：掷随机属性 → 存档 → 纸娃娃槽位 record 整推；
				// 镇派兵器：动态 4 属性按实力点数重算并补面板帧。
				if randomKind != randomKindNone {
					session.pushRandomPropsForEquip(command.ItemConfig, worn.uid, slot)
				}
				if slot == 22 && player.refreshZhenpaiEquipBonuses() {
					if err := player.pushZhenpaiPanelFrame(link); err != nil {
						log.Printf("%s: zhenpai panel frame: %v", conn.RemoteAddr(), err)
					}
				}
				gmSession.setAction(fmt.Sprintf("已发放并穿戴 %s（槽位 %d）", command.ItemConfig, slot))
				log.Printf("%s: GM grant_item wear config=%s slot=%d", conn.RemoteAddr(), command.ItemConfig, slot)
				return nil
			}
			inserted, landed := session.grantBagItemRow(entry, command.Amount)
			if !landed || inserted.slot <= 0 {
				gmSession.setAction("发放失败：背包已满（1..90）")
				return fmt.Errorf("bag full")
			}
			// 宝物/娃娃入包：掷值 + 存档 + 背包槽位 record 整推。
			if randomKind != randomKindNone {
				session.pushRandomPropsForBag(command.ItemConfig, inserted.uid, inserted.viewID, inserted.slot)
			}
			gmSession.setAction(fmt.Sprintf("已发放 %s x%d（背包 %d 页槽 %d）", command.ItemConfig, command.Amount, entry.viewID, inserted.slot))
			log.Printf("%s: GM grant_item bag config=%s amount=%d slot=%d", conn.RemoteAddr(), command.ItemConfig, command.Amount, inserted.slot)
		case "equip_card", "equip_hair":
			entry, known := skinCards.lookup(int(command.CardId))
			if !known {
				gmSession.setAction(fmt.Sprintf("皮肤 %d 不在 card.ini 目录", command.CardId))
				return fmt.Errorf("unknown skin %d", command.CardId)
			}
			if command.Action == "equip_hair" && !entry.hairOnly() {
				gmSession.setAction(fmt.Sprintf("皮肤 %d 不是纯发型卡（%s）", command.CardId, skinPartSummary(entry, player.visualSexOf())))
				return fmt.Errorf("skin %d is not hair-only", command.CardId)
			}
			state := player.skinSnapshot()
			sex := player.visualSexOf()
			state.addCollected(command.CardId)
			if command.Action == "equip_hair" {
				mounted := make([]int32, 0, len(state.mounted))
				for _, id := range state.mounted {
					other, ok := skinCards.lookup(int(id))
					if ok && other.hairOnly() && id != command.CardId {
						continue // 其他发型卡让位
					}
					if id == command.CardId {
						continue // 重挂刷新到队尾
					}
					mounted = append(mounted, id)
				}
				state.mounted = append(mounted, command.CardId)
			} else {
				state.removeMounted(command.CardId)
				state.mounted = append(state.mounted, command.CardId)
			}
			if saveErr := appearanceCfg.skins.Save(selected.ID, state); saveErr != nil {
				log.Printf("%s: GM persist skins role=%d: %v", conn.RemoteAddr(), selected.ID, saveErr)
			}
			player.applySkins(state)
			if refreshErr := pushSkinRefreshFrames(link, player); refreshErr != nil {
				gmSession.setAction("衣橱刷新失败：" + refreshErr.Error())
				return refreshErr
			}
			gmSession.setAction(fmt.Sprintf("已挂载皮肤 %d（%s）", command.CardId, skinPartSummary(entry, sex)))
			log.Printf("%s: GM %s skin=%d", conn.RemoteAddr(), command.Action, command.CardId)
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
			result, evalErr := scriptHost.Eval(command.Code)
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
			// 每击公式引擎一并热重载（formula.ini 规格复用启动缓存，不重读
			// 文件）；失败只回报，不回滚事件脚本结果。
			if installedCombatCalc != nil && len(combatFormulaSpecs) > 0 {
				if formulas, combatErr := installedCombatCalc.Reload(combatFormulaSpecs); combatErr != nil {
					gmSession.setAction(fmt.Sprintf("Lua 脚本已重载 count=%d；公式引擎重载失败：%v", loaded, combatErr))
					log.Printf("%s: GM lua_reload combat engine failed: %v", conn.RemoteAddr(), combatErr)
				} else {
					log.Printf("%s: GM lua_reload combat formulas=%d", conn.RemoteAddr(), formulas)
				}
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
			return fmt.Errorf("unsupported GM action %q", command.Action)
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
	// 周期自动存盘的节拍锚点：下方 1s tick 里按共享周期（GM 可调）
	// 判定是否到期，见 case now := <-facultyTicker.C。
	lastAutosave := time.Now()
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
			// 周期自动存盘（与断线下线同一组快照，均幂等 upsert）：
			// 进程被强杀（没走 Ctrl+C 优雅关服）时在线玩家最多丢一个
			// 周期的增量。周期读共享原子值，GM 面板改完即刻对全部
			// 在线会话生效；persist 对未登录态是 no-op。
			if interval := autosaveInterval(); interval > 0 && now.Sub(lastAutosave) >= interval {
				persistFaculty()
				persistJingMai()
				lastAutosave = now
			}
			if player == nil || selected == nil {
				continue
			}
			slot, advanced := player.facultyTick(now)
			// 历练层 100 秒步进（facultyTick 同拍）：整步才结算，静默秒
			// 零帧零落库。
			livepointChanged := player.livepointTick(now)
			innerPowerPassiveExpired := player.expireInnerPowerPassive(now)
			sitcrossSettlement := player.sitcrossTick()
			combatBruiseAdvanced := player.combatBruiseTick()
			combatRageAdvanced := player.rageRecoveryTick()
			// 轻功结算（Java facultyTicker 同拍）：租约秒耗 / 3s 分档
			// 回复；点变化才推 QingGongPoint 单属性帧。点耗尽强摘时
			// 同步摘对应 clearOnEnd Buff（神行/踏波行/上天梯），否则
			// 租约已停而客户端表现钉到 buff 寿命尽头。
			qingGongPointChanged, qingGongForceEnded := player.qingGongTick(now)
			// NPC 战斗模拟已上移到场景运行时的中央 tick；
			// 连接循环只保留玩家侧结算与复活。
			combatAdvanced := false
			if player.reviveWhenDue(now) {
				combatAdvanced = true
				log.Printf("%s: auto-revived player after NPC defeat", conn.RemoteAddr())
			}
			var tickErr error
			if advanced {
				// 行级增量帧（0x10 isView=1）：裸 0x18 重发行不触发客户端
				// updateitem，武学面板内功页会停在旧等级（2026-11-08 抓包
				// 实证）。上限对(MaxSP/MaxSPAdd)不进常规批，升级可能抬
				// 内功档位，这里用 215 同款专帧刷新怒气球分母。
				var frame []byte
				frame, tickErr = player.bookUpdateFrame(slot)
				if tickErr == nil {
					tickErr = link.WriteFrame(frame)
				}
				if tickErr == nil {
					if frame, err := player.rageCapUpdate(); err != nil {
						tickErr = err
					} else {
						tickErr = link.WriteFrame(frame)
					}
				}
			}
		if tickErr == nil && qingGongPointChanged {
			tickErr = link.WriteFrame(player.qingGongPointFrame())
		}
		if tickErr == nil && sitcrossSettlement.Changed() {
			// 打坐回复走官方形态帧簇：逐层 n=1 单属性帧 + msg24
			// c1=16 治疗飘字（UI 的 +回复量 绿字来源）。绝不能并入
			// 下面的全量 vitalUpdate——官服坐姿周期帧从不带
			// LogicState，整帧重推会让客户端重入坐姿动画。
			var frames [][]byte
			frames, tickErr = player.sitcrossRecoveryFrames(sitcrossSettlement)
			for _, frame := range frames {
				if tickErr != nil {
					break
				}
				tickErr = link.WriteFrame(frame)
			}
		}
		if tickErr == nil && qingGongForceEnded != "" {
			// 点耗尽强摘：与 216 end 同口径清轻功 Buff 后推全量
			// 状态（vitalUpdate 会带上空掉的 BufferInfo 槽）。
			if definition, exists := qingGongCatalog.Lookup(qingGongForceEnded); exists {
				// 缓降先补 msg169 摘帧（v7 需在清槽前读租约原到期）。
				writeGlideBuffRemovalFor(link, player, definition, now)
				if err := endQingGongBuff(link, player, definition); err != nil {
						log.Printf("%s: clear exhausted qinggong buff id=%s: %v", conn.RemoteAddr(), qingGongForceEnded, err)
					}
				}
			}
			if tickErr == nil && (advanced || combatBruiseAdvanced || combatRageAdvanced || combatAdvanced || innerPowerPassiveExpired || livepointChanged) {
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
			if livepointChanged {
				persistFaculty()
				log.Printf("%s: settled livepoint role=%d", conn.RemoteAddr(), selected.ID)
			}
			if sitcrossSettlement.Changed() {
				state := player.stateSnapshot()
				log.Printf("%s: settled sitcross HP=%d/%d MP=%d/%d heal=%d overflow=%d", conn.RemoteAddr(), state.HP, state.MaxHP, state.MP, state.MaxMP, sitcrossSettlement.HPApplied, sitcrossSettlement.HPWasted)
			}
			if combatBruiseAdvanced {
				state := player.stateSnapshot()
				log.Printf("%s: settled combat bruised blood HP=%d HitHP=%d", conn.RemoteAddr(), state.HP, state.HitHP)
			}
			if combatRageAdvanced {
				log.Printf("%s: settled combat rage SP=%d/%d", conn.RemoteAddr(), player.skillSP(), player.rageCap())
			}
			// 经脉修炼结算（msg150/1 起）：整分钟 +43 真气/分钟，750/周天
			// 升级；变化才发 View45 行帧并落库（fork jingMaiCultivateTick）。
			if frames := player.jingMaiCultivateTick(jingMaiCatalog, now); len(frames) > 0 {
				for _, frame := range frames {
					if err := link.WriteFrame(frame); err != nil {
						log.Printf("%s: publish jingmai tick frame: %v", conn.RemoteAddr(), err)
						return
					}
				}
				persistJingMai()
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
			// P2 域登录装载（msg70/71 货币、msg295 称号、msg150/161 经脉）：
			// 无存档=starter/空态，缺表降级只记日志，不阻塞登录。
			if saved, exists := currencyStore.Load(activeRole.ID); exists {
				player.setSilver(saved.Silver)
				log.Printf("%s: restored currency role=%d silver=%d", conn.RemoteAddr(), activeRole.ID, saved.Silver)
			}
			if titles := titleStore.Load(activeRole.ID); len(titles) > 0 {
				player.restoreTitles(titles)
				log.Printf("%s: restored titles role=%d owned=%d", conn.RemoteAddr(), activeRole.ID, len(titles))
			}
			if saved, exists := jingMaiStore.Load(activeRole.ID); exists {
				player.restoreJingMai(saved, time.Now())
				log.Printf("%s: restored jingmai role=%d cur=%s activated=%d", conn.RemoteAddr(), activeRole.ID, saved.Cur, len(saved.Activated))
			}
			// 邮件/坐骑镜像装载（Java LoginFlow restorePanel 同位）：
			// 无存档零帧，缺表降级只记日志，不阻塞登录。
			if letters := letterStore.Load(activeRole.ID); len(letters) > 0 {
				sessionLetters = letters
				log.Printf("%s: restored letters role=%d count=%d", conn.RemoteAddr(), activeRole.ID, len(letters))
			}
			if mounts := mountStore.Load(activeRole.ID); len(mounts) > 0 {
				sessionMounts = mounts
				log.Printf("%s: restored mounts role=%d owned=%d current=%d", conn.RemoteAddr(), activeRole.ID, len(mounts), mountCurrentPack(mounts))
			}
			// 已学技能装载（Java LoginFlow.loadDbSkills 同口径）：读库失败
			// 或空表只记日志回退 starter 集，不阻塞登录。
			if dbSkills, skillErr := skills.learnedSkills(ctx, activeRole.ID); skillErr != nil {
				log.Printf("%s: load learned skills (skip to starter set) role=%d: %v", conn.RemoteAddr(), activeRole.ID, skillErr)
			} else {
				dbLearnedSkills = dbSkills
				// 与 View40 的 merge 集同步进释放闸内存表，否则
				// DB 学的技能会被 learnedSkillLevel 判「未学」拒绝。
				player.mergeDBLearnedSkills(dbSkills)
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
			// 已经存在。记录表逐字节透传官方 447 schema 方言
			// （客户端原生消费者按官方下标绑定，自建子表会错绑）。
			recordTable, recordErr := serverRecordTable()
			if recordErr != nil {
				log.Printf("%s: encode record table: %v", conn.RemoteAddr(), recordErr)
				return
			}
			if err := link.WriteFrame(recordTable); err != nil {
				log.Printf("%s: write record table: %v", conn.RemoteAddr(), err)
				return
			}
			log.Printf("%s: sent property table (%d entries) and official record table (%d entries)", conn.RemoteAddr(), binary.LittleEndian.Uint16(propertyTable[1:]), binary.LittleEndian.Uint16(recordTable[1:]))
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
			// 面板身份/域块补推（Java PanelState.identityProperties/
			// domainProperties 随 0x0D 档下发的对应物，走独立瘦帧不动
			// 档形）：实力=自动推导（varprop.NeiGongLevel，GM set 过则
			// 覆盖）+ 称号/性格/侠恶/生存属性零值常发——缺发则人物面板
			// 整块空白（实力一行不显示的根因）；vitals 由 role_faculty
			// 快照在 restoreFaculty 时播种。穿戴武器的镇派动态 4 属性
			// 先重算（identity/vital 帧与注入位同源）。
			player.refreshZhenpaiEquipBonuses()
			if err := gmWritePanelFrames(link, player); err != nil {
				log.Printf("%s: push panel identity/domain frames: %v", conn.RemoteAddr(), err)
			}
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
				// 娃娃/宝物随机属性 record 整推（官服 0x11 flag=1 方言）：
				// 纸娃娃槽位行（view1）与背包行（对应页）各按 uid 取快照。
				pushLoginRandomProps(link, randomPropsStore, ctx, activeRole.ID, equips, bagItems)
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
			gmSession = gm.attach(machine.ID(), conn.RemoteAddr().String(), activeRole, roleFaction(activeRole.Appearance.Values), starterSilver, player.guanYinValue(), player.goldSnapshot(), player.skillSP(), player.facultyValue())
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
				// 切图重进同口径补推 GM 面板域（会话级状态在 world.begin
				// 的对象代际重置中保留，客户端对象表则被清空）。
				if err := gmWritePanelFrames(link, runtime.player); err != nil {
					log.Printf("%s: push GM panel replay frames: %v", conn.RemoteAddr(), err)
				}
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
					// 邮件/坐骑记录行挂玩家对象，0x0C 已随对象清空——重入后
					// 整表重放（镜像计数先归零：客户端已无旧行可删，外观域
					// 同口径；Java lettersGranted/mountsPushed 重入复位）。
					lettersPushed = false
					if err := pushLetters(); err != nil {
						log.Printf("%s: %v", conn.RemoteAddr(), err)
						return
					}
					mountsPushed = false
					mountSentRows = 0
					if err := pushMounts(); err != nil {
						log.Printf("%s: %v", conn.RemoteAddr(), err)
						return
					}
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
			if player != nil && !lettersPushed {
				// 邮件行（Java lettersGranted，位于 43/40 之前）：空邮箱零帧。
				if err := pushLetters(); err != nil {
					log.Printf("%s: %v", conn.RemoteAddr(), err)
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
			if player != nil && !mountsPushed {
				// 坐骑（Java mountsPushed，43/40 之后、switch 同步之前）：
				// NewRideRec 拥有行 + 骑乘态帧（重登步行口径）。
				if err := pushMounts(); err != nil {
					log.Printf("%s: %v", conn.RemoteAddr(), err)
					return
				}
			}
			if player != nil && !jyzPushed {
				// 九阴志数据部（native 62311 帧序：坐骑之后、开关同步
				// 之前）：quest49 簇 + 419 步骤 + data_record 58 行打包。
				// gamestep 步骤帧不在此处——sync 之后由收尾帧下发
				// （sync 重置 414 域，先发会被抹掉）。
				jyzPushed = true
				ownedMountPacks := make(map[uint32]bool, len(sessionMounts))
				for _, mount := range sessionMounts {
					ownedMountPacks[mount.PackID] = true
				}
				if jyzErr := pushJyzRecords(link, jyzRecordsConfig, ownedMountPacks); jyzErr != nil {
					log.Printf("%s: push jyz records: %v", conn.RemoteAddr(), jyzErr)
					return
				}
			}
			if player != nil && !switchSyncPushed {
				// 参考服 pushSwitchSync：ClientReady 收尾一次性批量开
				// 客户端 SwitchManager（0x1E 414 "sync" ×203）。2919 在列
				// ——人物面板「切换新版」按钮的显示门槛；缺此帧按钮不出现。
				// 必须先于 enableSkillActionSwitch：sync 清空整棵开关树后
				// 重设，1223 随后走 reset 单开。
				switchSyncPushed = true
				if syncErr := pushSwitchSync(link); syncErr != nil {
					log.Printf("%s: push switch sync: %v", conn.RemoteAddr(), syncErr)
					return
				}
				log.Printf("%s: pushed switch sync switches=%d via S2C 0x1E message=%d", conn.RemoteAddr(), len(switchSyncIDs), serverSwitchControlMessage)
			}
			if player != nil {
				if switchErr := enableSkillActionSwitch(link); switchErr != nil {
					log.Printf("%s: enable native skill-action switch: %v", conn.RemoteAddr(), switchErr)
					return
				}
				log.Printf("%s: enabled native skill-action switch=%d via modern S2C 0x27 message=%d", conn.RemoteAddr(), skillActionInputSwitch, serverSwitchControlMessage)
			}
			if player != nil && jyzPushed && !jyzGameStepSent {
				// 414 'gamestep' 收尾帧（native #1744：switch sync 之后
				// 的最后一帧；sync 重置 414 域状态，先发的 gamestep 会被
				// 抹掉——功能入口菜单不出现的根因）。
				jyzGameStepSent = true
				if stepErr := pushJyzGameStepFinal(link, jyzRecordsConfig); stepErr != nil {
					log.Printf("%s: push jyz gamestep final: %v", conn.RemoteAddr(), stepErr)
					return
				}
				log.Printf("%s: pushed jyz gamestep step=%d final via S2C 0x1E message=%d", conn.RemoteAddr(), jyzRecordsConfig.step, serverJyzGameStepMessage)
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
					if !switchSyncPushed {
						// 兼容 0x0A 晋升的 ClientReady 同样要开开关树
						//（该客户端路径不经过显式 0x09 分支时兜底）。
						switchSyncPushed = true
						if syncErr := pushSwitchSync(link); syncErr != nil {
							log.Printf("%s: push compatibility switch sync: %v", conn.RemoteAddr(), syncErr)
							return
						}
						log.Printf("%s: pushed switch sync switches=%d via compatibility ClientReady", conn.RemoteAddr(), len(switchSyncIDs))
					}
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
				// 跳段落地探测（2026-11-07 跳段限制批次）：0x0A 位移包
				// 逐帧喂 y——着地时校准基准，腾空回落到基准 1m 内即清
				// 跳段计数进入新循环（observePositionY 内含互斥锁）。
				// 落地拍同时回放跳段预输入：被跳段闸拒绝的提前输入在
				// 300ms 窗口内于此执行（腾空判定滞后于真实落地的补偿）。
				if runtime != nil && runtime.player != nil {
					if runtime.player.observePositionY(y, time.Now()) {
						flushBufferedAirJump(link, runtime.player, qingGongCatalog, conn.RemoteAddr().String())
					}
				}
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
				if custom, ok, _ := parseClientActivityCustomMessage(plain); ok && len(custom.Values) > 0 && custom.Values[0].Type == 2 && (custom.Values[0].Int32 == clientCustomSetShortcut || custom.Values[0].Int32 == clientCustomRemoveShortcut || custom.Values[0].Int32 == clientCustomSitCross || custom.Values[0].Int32 == clientCustomSaveCustomizing || custom.Values[0].Int32 == clientCustomKeybindUpload || custom.Values[0].Int32 == clientCustomSetOperateMode || custom.Values[0].Int32 == clientCustomNeiGongBind || custom.Values[0].Int32 == 211 || custom.Values[0].Int32 == 215 || custom.Values[0].Int32 == 216 || custom.Values[0].Int32 == 217 || custom.Values[0].Int32 == clientCustomActiveParry || custom.Values[0].Int32 == 1006 || custom.Values[0].Int32 == clientCustomSelectTarget || custom.Values[0].Int32 == clientCustomAttackTarget || custom.Values[0].Int32 == clientCustomBlockState || custom.Values[0].Int32 == clientCustomMovementUpload || custom.Values[0].Int32 == clientCustomRushMoveUpload || custom.Values[0].Int32 == clientCustomCastDashLanding || custom.Values[0].Int32 == clientCustomChangeWarning || custom.Values[0].Int32 == clientCustomMoveItem || custom.Values[0].Int32 == clientCustomUseItem || custom.Values[0].Int32 == clientCustomDeleteItem || custom.Values[0].Int32 == clientCustomArrangeBag || custom.Values[0].Int32 == clientCustomShowEquipType || custom.Values[0].Int32 == clientCustomFashionSuit || custom.Values[0].Int32 == clientCustomWardrobeCard || custom.Values[0].Int32 == clientCustomShopBuy || custom.Values[0].Int32 == clientCustomShopSell || custom.Values[0].Int32 == clientCustomSetTitle || custom.Values[0].Int32 == clientCustomJingMaiCultivate || custom.Values[0].Int32 == clientCustomJingMaiActive || custom.Values[0].Int32 == clientCustomFreshManChoice || custom.Values[0].Int32 == clientCustomGuideAddSchool || custom.Values[0].Int32 == clientCustomSchoolMessage || custom.Values[0].Int32 == clientCustomPickupDropItem || custom.Values[0].Int32 == clientCustomCloseDropBox || custom.Values[0].Int32 == clientCustomJyzQuery) {
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
				case clientCustomNeiGongBind:
					if _, handleErr := handleNeiGongBindCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity neigong bind: %v", conn.RemoteAddr(), handleErr)
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
				case clientCustomAttackTarget:
					if handleErr := handleAttackTargetCustom(world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle attack target: %v", conn.RemoteAddr(), handleErr)
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
				case clientCustomRushMoveUpload:
					// 冲锋在途位移上报（msg220，4×f32）：非在途静默。
					if _, handleErr := handleRushMovementUpload(link, player, world, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle rush movement upload: %v", conn.RemoteAddr(), handleErr)
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
				case clientCustomShopBuy:
					if _, handleErr := handleShopBuyCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity shop buy: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomShopSell:
					if _, handleErr := handleShopSellCustom(newItemSession(), custom); handleErr != nil {
						log.Printf("%s: handle activity shop sell: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomSetTitle:
					if _, handleErr := handleSetTitleCustom(link, player, custom, conn.RemoteAddr().String(), titleStore, currentRoleID()); handleErr != nil {
						log.Printf("%s: handle activity set title: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomJingMaiCultivate:
					if _, handleErr := handleJingMaiCultivateCustom(link, player, jingMaiCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity jingmai cultivate: %v", conn.RemoteAddr(), handleErr)
					} else {
						persistJingMai()
					}
				case clientCustomJingMaiActive:
					if _, handleErr := handleJingMaiActiveCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity jingmai active: %v", conn.RemoteAddr(), handleErr)
					} else {
						persistJingMai()
					}
				case clientCustomFreshManChoice:
					if _, handleErr := handleFreshManChoice(custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity fresh-man choice: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomGuideAddSchool:
					if _, handleErr := handleGuideAddSchoolCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity guide add school: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomSchoolMessage:
					if _, handleErr := handleSchoolCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity school message: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomPickupDropItem:
					if _, handleErr := handlePickupDropItemCustom(world, newItemSession(), custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity pickup drop item: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomCloseDropBox:
					if _, handleErr := handleCloseDropBoxCustom(world, custom); handleErr != nil {
						log.Printf("%s: handle activity close drop box: %v", conn.RemoteAddr(), handleErr)
					}
				case clientCustomJyzQuery:
					if handleErr := handleJyzQueryCustom(link, custom, conn.RemoteAddr().String()); handleErr != nil {
						log.Printf("%s: handle activity jyz query: %v", conn.RemoteAddr(), handleErr)
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
			case clientCustomNeiGongBind:
				if _, handleErr := handleNeiGongBindCustom(link, player, custom, conn.RemoteAddr().String(), customConfig.keybind, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle neigong bind: %v", conn.RemoteAddr(), handleErr)
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
			case clientCustomShopBuy:
				if _, handleErr := handleShopBuyCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle shop buy: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomShopSell:
				if _, handleErr := handleShopSellCustom(newItemSession(), custom); handleErr != nil {
					log.Printf("%s: handle shop sell: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomSetTitle:
				if _, handleErr := handleSetTitleCustom(link, player, custom, conn.RemoteAddr().String(), titleStore, currentRoleID()); handleErr != nil {
					log.Printf("%s: handle set title: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomJingMaiCultivate:
				if _, handleErr := handleJingMaiCultivateCustom(link, player, jingMaiCatalog, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle jingmai cultivate: %v", conn.RemoteAddr(), handleErr)
				} else {
					persistJingMai()
				}
			case clientCustomJingMaiActive:
				if _, handleErr := handleJingMaiActiveCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle jingmai active: %v", conn.RemoteAddr(), handleErr)
				} else {
					persistJingMai()
				}
			case clientCustomFreshManChoice:
				if _, handleErr := handleFreshManChoice(custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle fresh-man choice: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomGuideAddSchool:
				if _, handleErr := handleGuideAddSchoolCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle guide add school: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomSchoolMessage:
				if _, handleErr := handleSchoolCustom(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle school message: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomPickupDropItem:
				if _, handleErr := handlePickupDropItemCustom(world, newItemSession(), custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle pickup drop item: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomCloseDropBox:
				if _, handleErr := handleCloseDropBoxCustom(world, custom); handleErr != nil {
					log.Printf("%s: handle close drop box: %v", conn.RemoteAddr(), handleErr)
				}
			case clientCustomJyzQuery:
				if handleErr := handleJyzQueryCustom(link, custom, conn.RemoteAddr().String()); handleErr != nil {
					log.Printf("%s: handle jyz query: %v", conn.RemoteAddr(), handleErr)
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
		// end：持续型租约摘除（点数不再扣——begin 已付全程开局费，
		// 双扣=一次按放双倍耗）。冲刺类引擎每次使用也发 flag=0，
		// 只有真持有同 id 租约时才有摘除意义（Java QingGongHandler 同款）。
		if definition.UseType == 1 && player.releaseDriftIfOwned(definition.ID) {
			log.Printf("%s: qinggong hold end id=%s", remote, definition.ID)
		}
		// 缓降摘帧先于清槽（msg169 v7 取租约原到期）；游离 end 也回帧
		// ——客户端状态重同步（Java releaseGlide 同口径）。
		writeGlideBuffRemovalFor(link, player, definition, time.Now())
		if err := endQingGongBuff(link, player, definition); err != nil {
			return true, err
		}
		log.Printf("%s: qinggong end id=%s", remote, definition.ID)
		return true, nil
	}
	now := time.Now()
	// begin 重报闸（Java 2026-11-03 真机「按住 Shift 轻功条秒清」根因）：
	// ①持续型：同 id 续约不重复扣开局费；②即时型：500ms 重报闸拦
	// 引擎逐帧重发（人手 ~4/s 以下）。
	if definition.UseType == 1 && player.engagedDriftID() == definition.ID {
		log.Printf("%s: qinggong begin repeat id=%s suppressed (renewal)", remote, definition.ID)
		return true, nil
	}
	if definition.UseType != 1 && !player.instantQingGongRechargeable(definition.ID, now) {
		log.Printf("%s: qinggong begin repeat id=%s suppressed (instant_guard)", remote, definition.ID)
		return true, nil
	}
	if spec, hasBuff := qinggongBuffFor(definition); hasBuff && spec.clearOnEnd && player.hasActiveBuff(spec.slot, spec.staticData, now) {
		log.Printf("%s: ignore duplicate qinggong start id=%s", remote, definition.ID)
		return true, nil
	}
	// ExistBufID 能力门槛（2026-11-07 跳段限制批次，用户钉死语义）：
	// 未挂对应能力 Buff（qinggong_4→buf_DHadd_2rd / qinggong_5→
	// buf_DHadd_3rd，激活 qinggong_15/17 挂）→ 拒绝——没学没激活就
	// 不能二/三段跳；已挂 → 免 Consume（native-menu 反编译实证的免费
	// 分支）。未知的 ExistBufID（如 qinggong_9 的 buf_DHadd_light，
	// 能力来源未实现）不设门槛，保旧表/缓降链现状。
	capabilityFree := false
	if spec, known := qingGongExistBuffSpec(definition.ExistBufID); known {
		if !player.hasActiveBuff(spec.slot, spec.staticData, now) {
			log.Printf("%s: reject qinggong id=%s missing capability buff %s", remote, definition.ID, definition.ExistBufID)
			return true, nil
		}
		capabilityFree = true
	}
	// 跳段闸：同一腾空内同 id 计段轻功（Type 4/5/20）只放行一次；
	// 落地（observePositionY）/超时（qingGongTick 兜底）/技能重置
	// （grantAirJumpReset 只给一段）后恢复。拒绝不回滚段位——客户端
	// 物理照跳（Java 口径），段位跟随消耗防蹭跳。
	if isJumpStageQingGong(definition) {
		if !player.beginAirJump(definition.ID, now) {
			// 跳段预输入（input buffer）：腾空判定可能滞后于客户端真实
			// 落地（落地位移包在途），该 begin 或是下一周期的合法跳。
			// 暂存最新一条，落地观测拍回放（300ms 窗口过期即丢弃——
			// 纯空中连打不会有落地包跟随，不执行不扣费）。
			player.bufferAirJumpInput(bufferedAirJump{
				qingGongID:     definition.ID,
				capabilityFree: capabilityFree,
				queuedAt:       now,
			})
			log.Printf("%s: reject qinggong id=%s air jump stage exhausted (buffered)", remote, definition.ID)
			return true, nil
		}
		// 受理即作废同 id 旧预输入：该次按键已被服务，落地拍不再回放。
		player.clearBufferedAirJump(definition.ID)
	}
	return true, acceptQingGongBegin(link, player, definition, capabilityFree, now, remote)
}

// acceptQingGongBegin 执行通过全部闸门后的 216 begin 尾段（扣费→
// 持续租约→Buff 链→推帧）。正常路径与跳段预输入回放
// （flushBufferedAirJump）共用，保证两条路径的扣费/帧序完全同构；
// 计段（beginAirJump）由调用方在扣费前完成——拒绝不回滚段位同口径。
func acceptQingGongBegin(link sceneMessageConnection, player *playerActor, definition qinggong.Definition, capabilityFree bool, now time.Time, remote string) error {
	var cost int32
	if !capabilityFree {
		var accepted bool
		cost, accepted = player.useQingGong(definition.Consume, now)
		if !accepted {
			state := player.stateSnapshot()
			log.Printf("%s: reject qinggong id=%s cost=%d point=%d", remote, definition.ID, cost, state.QingGongPoint)
			return nil
		}
	}
	// 持续型（UseType=1，含 qinggong_9 缓降）：挂秒耗租约（CCQGP 每秒
	// 扣，回复暂停；表缺 CCQGP 时 drain=0 仍挂租约保续约/暂停语义——
	// Java 同口径）。异 id 新 begin 由 engageDrift 直接顶替旧租约。
	if definition.UseType == 1 {
		drain := qingGongDrainPerSecond(definition)
		player.engageDrift(definition.ID, drain)
		log.Printf("%s: qinggong hold begin id=%s consume=%d drain/s=%d", remote, definition.ID, cost, drain)
	}
	alreadyActive, expires, err := activateQingGongBuff(player, definition, now)
	if err != nil {
		return err
	}
	// 缓降（buf_qg_light）：msg169 九值挂帧先于槽帧（Java engageGlide
	// 帧序 ②169 ③BufferInfoN；点数随 vitalUpdate 同批下发）。
	if definition.BufferID == qingGongGlideBuffID && !alreadyActive && !expires.IsZero() {
		if err := writeGlideTimedBuffAdd(link, player, expires); err != nil {
			return err
		}
	}
	update, err := player.vitalUpdate()
	if err != nil {
		return fmt.Errorf("encode qinggong update: %w", err)
	}
	if err = link.WriteFrame(update); err != nil {
		return fmt.Errorf("write qinggong update: %w", err)
	}
	if !alreadyActive && !expires.IsZero() {
		spec, _ := qinggongBuffFor(definition)
		schedulePlayerBuffExpiry(link, player, spec.slot, spec.staticData, expires)
	}
	state := player.stateSnapshot()
	log.Printf("%s: accepted qinggong id=%s type=%d cost=%d point=%d/%d", remote, definition.ID, definition.Type, cost, state.QingGongPoint, state.MaxQingGongPoint+state.MaxQingGongPointAdd)
	return nil
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

// sceneObjectIntProperty 更新一个协商为四字节有符号整数的场景
// 对象属性（如轻功值 QingGongPoint@692 的秒级单帧推送）。
func sceneObjectIntProperty(objectID, ownerID uint32, propertyIndex uint16, value int32) []byte {
	msg := make([]byte, 12, 18)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // 场景对象，而非视图对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	msg = binary.LittleEndian.AppendUint16(msg, propertyIndex)
	return binary.LittleEndian.AppendUint32(msg, uint32(value))
}

// sceneObjectWideStringProperty 更新一个协商为宽字符串的场景对象
// 属性（如挂尸掉落归属 PickMember@1592，逗号分隔玩家名）。
func sceneObjectWideStringProperty(objectID, ownerID uint32, propertyIndex uint16, value string) []byte {
	msg := make([]byte, 12, 12+len(value)*2+4)
	msg[0] = 0x10 // SERVER_OBJECT_PROPERTY
	msg[1] = 0    // 场景对象，而非视图对象
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], 1)
	msg = binary.LittleEndian.AppendUint16(msg, propertyIndex)
	units := utf16.Encode([]rune(value))
	msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
	for _, unit := range units {
		msg = binary.LittleEndian.AppendUint16(msg, unit)
	}
	return binary.LittleEndian.AppendUint16(msg, 0)
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
