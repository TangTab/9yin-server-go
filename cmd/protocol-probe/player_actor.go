package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"

	"github.com/local/9yin-go-server/internal/gmapi"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// fork 对象 id 方案（docs/id-system-analysis.md §5/§6，权威=fork exe
// 9yin-game-native-menu.exe 抓包 127-0-0-1-57378 真机进游戏会话）：
// 玩家 = (0x11000001, 0x7A5C0B) 常量对。0x0B PlayerEntry 首携此对，
// 客户端据此绑定自体化身并做移动/接近子系统的高位类型位校验。
// 此前真机三连败的重新解读：51357=(0x11000001,1) 败在 owner=1 撞
// NPC 专属位；49925=(0x07000001,618) 败在官服 0x07 段非本客户端方言；
// （1,1) 基线能进但 id 校验门全关。fork 的精确对此前从未上过真机——
// "obj 必须恒 1"的旧结论作废，本对即 fork 逐字节实证的可进游戏对。
// 会话变量而非 const：留 per-role 演进口；默认对（无 DB 场景/测试）
// 即 fork 常量。
var (
	playerObjectID uint32 = 0x11000001
	playerOwnerID  uint32 = 0x7A5C0B
)

// setPlayerWireIdentity 角色装载时定玩家 (obj, owner)：当前照抄 fork
// 常量对（fork 抓包实证其客户端不验 owner 取值规则，只验配对合法
// 性）；roleID 参数保留为将来 per-role 分配的接入口，暂不参与取值。
func setPlayerWireIdentity(roleID uint32) {
	playerObjectID = 0x11000001
	playerOwnerID = 0x7A5C0B
	_ = roleID
}

const (

	// BufferListStr 是原生玩法的 BuffContainer 快照。BufferInfo 槽位是
	// 原生视觉存储的投影。两者必须描述同一批活动 Buff：仅有 UI 的
	// BufferInfo 不足以通过 qinggong_3 的原生 HasBuff("buf_qg_drift") 门控。
	bufferInfoSlotCount uint16 = 24

	redSuperArmorBuffSlot uint16 = 1
	waterDriftBuffSlot    uint16 = 2
	waterJumpBuffSlot     uint16 = 3
	innerPowerBuffSlot    uint16 = 4
	yiHuaSkillBuffSlot    uint16 = 5
	yiHuaSkill05BuffSlot  uint16 = 6
	taiJiStanceBuffSlot   uint16 = 7
	taiJiSkill05BuffSlot  uint16 = 8
	taiJiSkill07BuffSlot  uint16 = 9
	// 槽 10 为内功"料敌先机"Proc Buff 保留（dodge_add 闪避窗激活期间
	// 挂载，见 neigong_dispatch.go；动态技能 Buff 协商分配范围 11..24
	// 不变，槽 10 客户端侧属原生 Proc Buff 家族，不作他用）。
	neigongProcBuffSlot uint16 = 10
	// 幻玉神功 Proc Buff 的原生家族勘定值（buff_new.ini [buf_ng_yh_001_5]
	// StaticData=13107，varprop→proppack 链含 DodgeProbAdd=20）。
	neigongProcBuffStaticData uint32 = 13107
	neigongProcBuffConfigID   string = "buf_ng_yh_001_5"
	firstAutoBuffSlot         uint16 = 11

	// 轻功使用型/能力型 Buff 槽（qgdefine BufferID/ExistBufID 链，
	// StaticData 出自 F: 部署 resources/modern/share/skill/buff_new.ini
	// 实证；DHadd 两槽是激活 qinggong_15/17 记录期间的常驻能力 Buff，
	// 其余为 216 begin 的使用型 Buff）。
	qgAirRushBuffSlot  uint16 = 12 // buf_qg_airrush  StaticData 6894（LifeTime 100ms）
	qgFastBuffSlot     uint16 = 13 // buf_qg_fast     StaticData 217（10min）
	qgClimbBuffSlot    uint16 = 14 // buf_qg_climb    StaticData 219（10min）
	qgGlidingBuffSlot  uint16 = 15 // buf_qg_gliding  StaticData 23070
	qgDiveBuffSlot     uint16 = 16 // buf_qg_dive     StaticData 23292
	qgDHadd2rdBuffSlot uint16 = 17 // buf_DHadd_2rd   StaticData 1578（常驻）
	qgDHadd3rdBuffSlot uint16 = 18 // buf_DHadd_3rd   StaticData 1579（常驻）
	// qinggong_9 轻身缓降使用型 Buff（Java GLIDE_BUFF_* 同源：buff_static
	// [218]、buff_new LifeTime 600000；槽取 qg 族块首位空闲，对齐 Java
	// freeBuffSlot 从 12 起扫的首个可得位）。挂/摘除槽帧外还伴随
	// msg169 九值 timedBuff 对（见 qinggong_records.go）。
	qgLightBuffSlot     uint16 = 19 // buf_qg_light  StaticData 218（10min）
	glideBuffStaticData uint32 = 218

	// 右键格挡 BuffInParry（Java PlayerActor 同源）：槽 11、
	// StaticData 4558、24h 租约。按下挂槽、松开清槽。
	parryBuffSlot       uint16 = 11
	parryBuffStaticData uint32 = 4558

	redSuperArmorStaticData uint32 = 8682
	waterDriftStaticData    uint32 = 216
	waterJumpStaticData     uint32 = 5786

	// 装备中的内功使用同一个原生 BuffStore，但不得与临时战斗/轻功槽位
	// 共用。其租约在出生时以及每次 215 响应时都从权威装备状态续期。
	innerPowerBuffLease = 24 * time.Hour

	// SP 是当前现代客户端的怒气资源。基础满池值对齐 fork 抓包实证
	// (49840 帧 #16 登录全量批 MaxSP=80);有效上限 = 基础 80 + 当前
	// 运行内功的档位加成(playerProgress.rageCap)。wire 上按官服口径
	// 只随 MaxSP(580) 下发折算后的最终值(rageCapWireValue),MaxSPAdd
	// (583) 从不出帧。所有 GM 及战斗获取路径都经由这些辅助函数,客户端
	// 属性永远不会成为该资源的权威。
	// 权威定义在 internal/gmapi（GM 线协议界值与之共用），此处为别名。
	maxPlayerSP int32 = gmapi.MaxPlayerSP

	// 本地测试的打坐恢复按现有的服务器秒每秒结算一次。用户规则：存活
	// 的角色无论装备或内功推导出的总量如何，都在十秒内回满
	// MaxHP/MaxMP。
	sitcrossFullRecoverySeconds int32 = 10
	sitcrossHeartMillis         int32 = 1000

	// 战斗恢复从不恢复真实 HP。它只在每个权威服务器秒内削减一次原生
	// HitHP-HP 瘀血层。
	combatBruiseRecoveryPerSecond int32 = 10

	// 怒气事件得怒(官方规则,用户钉死):格挡状态被揍 +2、直接被揍 +1、
	// 实招命中(目标未格挡)+2、攻击被招架 0。NPC 无格挡态,命中即 +2。
	ragGainParriedHit   int32 = 2
	ragGainUnparriedHit int32 = 1
	ragGainCleanHit     int32 = 2
)

// 属性索引按名解析（0x09 表方言门控换表自动跟随；fork 位见各行注记）。
var (
	propNeigongPKStatus          = clientdata.MustPlayerProperty("NeigongPKStatus").Index       // NeigongPKStatus Word(官方线型 2)（fork=85）
	propDead                     = clientdata.MustPlayerProperty("Dead").Index                  // Dead Byte(官方线型 1)（fork=464）
	propCurSkillID               = clientdata.MustPlayerProperty("CurSkillID").Index            // CurSkillID String（fork=460）
	propCurSkillCanNoLimit       = clientdata.MustPlayerProperty("CurSkillCanNoLimit").Index    // CurSkillCanNoLimit Byte（fork=624）
	propCantBeSkillLocked        = clientdata.MustPlayerProperty("CantBeSkillLocked").Index     // CantBeSkillLocked Byte（fork=208）
	propCantBeSkillMustLocked    = clientdata.MustPlayerProperty("CantBeSkillMustLocked").Index // CantBeSkillMustLocked Byte（fork=209）
	propState                    = clientdata.MustSceneProperty("State").Index                  // State String（动作总时长后 stand 复位帧用）（fork=19）
	propPauseTime                = clientdata.MustPlayerProperty("PauseTime").Index             // PauseTime（fork=2240）
	propHPHeartSpeed             = clientdata.MustPlayerProperty("HPHeartSpeed").Index          // HPHeartSpeed（fork=617）
	propHPHeartSpeedAdd          = clientdata.MustPlayerProperty("HPHeartSpeedAdd").Index       // HPHeartSpeedAdd（fork=621）
	propMPHeartSpeed             = clientdata.MustPlayerProperty("MPHeartSpeed").Index          // MPHeartSpeed（fork=618）
	propMPHeartSpeedAdd          = clientdata.MustPlayerProperty("MPHeartSpeedAdd").Index       // MPHeartSpeedAdd（fork=622）
	propHP                       = clientdata.MustPlayerProperty("HP").Index                    // HP（fork=465）
	propMP                       = clientdata.MustPlayerProperty("MP").Index                    // MP（fork=466）
	propHPRatio                  = clientdata.MustPlayerProperty("HPRatio").Index               // HPRatio（fork=469）
	propMPRatio                  = clientdata.MustPlayerProperty("MPRatio").Index               // MPRatio（fork=470）
	propHitHP                    = clientdata.MustPlayerProperty("HitHP").Index                 // HitHP（fork=481）
	propHitHPRatio               = clientdata.MustPlayerProperty("HitHPRatio").Index            // HitHPRatio（fork=482）
	propHPUpSpeed                = clientdata.MustPlayerProperty("HPUpSpeed").Index             // HPUpSpeed（fork=615）
	propHPUpSpeedAdd             = clientdata.MustPlayerProperty("HPUpSpeedAdd").Index          // HPUpSpeedAdd（fork=619）
	propMPUpSpeed                = clientdata.MustPlayerProperty("MPUpSpeed").Index             // MPUpSpeed（fork=616）
	propMPUpSpeedAdd             = clientdata.MustPlayerProperty("MPUpSpeedAdd").Index          // MPUpSpeedAdd（fork=620）
	propCurSkillTarget           = clientdata.MustPlayerProperty("CurSkillTarget").Index        // CurSkillTarget Object（fork=627）
	propModifySkillLockTime      = clientdata.MustPlayerProperty("ModifySkillLockTime").Index   // ModifySkillLockTime（fork=221）
	propSkillCanUse              = clientdata.MustPlayerProperty("CanUse").Index                // CanUse Byte(官服怒气灰化心跳簇同索引)（fork=2236）
	propSkillCurTargetCount      = clientdata.MustPlayerProperty("CurTargetCount").Index        // CurTargetCount（fork=577）
	propForce                    = clientdata.MustPlayerProperty("Force").Index                 // Force String(玩家域)（fork=194）
	propNewSchool                = clientdata.MustPlayerProperty("NewSchool").Index             // NewSchool String（fork=190）
	bufferInfoFirstPropertyIndex = clientdata.MustPlayerProperty("BufferInfo1").Index           // BufferInfo1（fork=629）
	propInParry                  = clientdata.MustPlayerProperty("InParry").Index               // InParry（fork=471）
	propInManualParry            = clientdata.MustPlayerProperty("InManualParry").Index         // InManualParry（fork=474）
	propParryValue               = clientdata.MustPlayerProperty("ParryValue").Index            // ParryValue（fork=475）
	propCantParry                = clientdata.MustPlayerProperty("CantParry").Index             // CantParry（fork=472）
	propParryRatio               = clientdata.MustPlayerProperty("ParryRatio").Index            // ParryRatio（fork=478）
	propParryUpSpeed             = clientdata.MustPlayerProperty("ParryUpSpeed").Index          // ParryUpSpeed（fork=479）
)

var nativeBuffSlots = func() []uint16 {
	slots := make([]uint16, bufferInfoSlotCount)
	for index := range slots {
		slots[index] = uint16(index + 1)
	}
	return slots
}()

func bufferInfoPropertyIndex(slot uint16) uint16 {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return bufferInfoFirstPropertyIndex + slot - 1
}

func bufferInfoPropertyName(slot uint16) string {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return fmt.Sprintf("BufferInfo%d", slot)
}

// playerActor 将权威运行时角色桥接到当前客户端协商出的
// 可见属性 schema。对当前本地测试服务器而言，货币的权威属于会话；
// 一旦引入物品/货币存储，购买事务将成为
// 它的持久化所有者。
//
// 锁纪律：p.mu 是玩家状态（下方全部字段，含三份 sim 组件值）的
// 唯一锁。跨对象锁序恒为 sceneLifecycle.mu（外）→ p.mu（内）；
// 持 p.mu 期间不得调用 sceneLifecycle 的任何加锁方法。异步 timer
// 路径（技能清除、buff 到期）一律经由本类型的持锁方法访问状态。
type playerActor struct {
	mu       sync.Mutex
	name     string
	vitals   sim.Vitals
	qinggong sim.QingGong
	speed    sim.Motion
	silver   int32
	faction  string
	motion   playerMotion
	// lastCombatAt 是最近一次战斗活动时刻（命中敌人/被命中），
	// 驱动 setCombatPresence 的 10s 脱战迟滞窗口。p.mu 保护。
	lastCombatAt time.Time
	// rageRecoveryPerSecond 是 1s ticker 的怒气自动回复速率（GM
	// set_rage_recovery 会话级设置；默认 0=不自动回复，官方满档 5）。
	rageRecoveryPerSecond int32
	buffs                 map[uint16]activePlayerBuff
	activeQingGong        map[string]struct{}
	shortcuts             []playerShortcut
	keybind               playerKeybind
	learnedSkills         map[string]int32
	skillCooldowns        map[string]time.Time
	skillTeamCooldowns    map[int32]time.Time
	// 公共 GCD（跨套路切换锁，bug.txt 官方规则；Java publicCdUntil/
	// publicCDTaolu 同构）：到期时刻 + 发起套路串——同套路连招与出招者
	// 自身只走个人 CD，切套路才付 7s；每次受理都刷新窗口（连招把切换
	// 锁后推是官方语义）。p.mu 保护。
	publicCDUntil time.Time
	publicCDTaolu string

	currentSkillID       string
	currentSkillEffectID string
	currentSkillLevel    int32
	currentSkillTarget   uint64
	// currentSkillEpoch 是出招代次（Java castEpoch 同构）：收尾定时器按
	// (id, epoch) 配对，新招顶掉旧招时旧定时器空转，add/del 恒 1:1。
	currentSkillEpoch uint64
	// skillViewRows 是 View40 行号映射（configID → 行号，出招 CanUse
	// 行帧寻址用；登录 merge 与秘籍重建时随视图帧一起刷新）。
	skillViewRows  map[string]uint16
	activeParry    activeParryRequest
	parrying       bool
	skillLockUntil time.Time
	reviveAt       time.Time
	progress       playerProgress
	// timedBuffs 限时增益（timed_buffs.go）：超长时限 buff 的运行态，
	// role_timed_buffs 快照持久化；与战斗槽位租约（p.buffs）分离。
	timedBuffs []timedBuffRuntime
	// equipAdds 普通装备随机属性的战斗面板折算（属性索引→合计加成，
	// official_login_align.go equipAttributeSums）：登录装载与穿脱时
	// 刷新；combatPanelProperties 在 progressProperties 基础上合并本层，
	// 登录全量簇与换装增量簇共用同一出口。
	equipAdds map[uint16]int32
	// GM 面板域（货币/属性/状态批次，gm_stats.go）：官银为会话级货币
	//（Java PlayerActor 同口径，不落库）；gm 是实力等级/善恶/PK 模式/
	// 生存属性的会话权威，spawn 后按需补推。
	guanYin int32
	gm      gmPanelDomain
	// 冲锋暂存链（fork qinggong_gj.go / Java SkillCastHandler 同源）：
	// 211 越程按下时暂存出招，msg15/212 探测到位后续招。
	// 与同名方法 logicState() 区分：权威运行状态在 vitals.Logic。
	rushLogicState byte // 0=普通 2=预警（msg292 翻转；463 号属性回帧）
	pendingCast    *pendingCastState
	rushInFlight   bool
	rushDashIssued bool
	// resumeSettleSkip：冲锋落地重入受理链时的一次性跳过标记
	//（pending.settled 时置位——CD/MP 已在 c61 时点结算）。
	resumeSettleSkip bool
	// cooldown_rec 双槽覆写簿记（官服方言，official_frames.go）：每玩家
	// 固定个人/公共两槽，col0/col3 变化才重写、col1/col2 每次覆写，
	// 过期行由下次受理/执行惰性 0x12 清扫（f/g/b 实证）。
	cdPersonalRowBook cooldownRowBook
	cdPublicRowBook   cooldownRowBook
	// rushStatePingAt：官服状态四连（586/645/646/750）上次下发时刻——
	// 官服全程 ~3s 周期重发，冲锋接近段按此节流补发（锁 p.mu）。
	rushStatePingAt time.Time
	// moveEchoWalking 官服移动回显状态机（技能.pcapng §一）：state7↔4
	// 走停迁移沿发状态四连，同态重发（追怪改向/在途重报）不重发
	//（锁 p.mu）。
	moveEchoWalking bool
	// 物品域会话模型（msg30/31/34/36 的权威内存态；登录时从
	// inventoryStore 恢复，操作后写穿 DB）。行类型直接复用读侧
	// bagRow/equipRow（帧构造器同型）。锁纪律仍走 p.mu。
	bag           []bagRow
	equip         []equipRow
	bagpackRows   []bagpackRow
	visualSex     byte // 武器外观解析用（登录装载时回填）
	lastItemUseAt time.Time
	// 坐骑运行域（mount_runtime.go）：拥有集会话镜像（登录时装入，
	// mount_ride/266 链变更为权威）+ 骑乘/挂起态 + NewRideRec 已推行数
	//（整刷清场记账，0x0C 重入随对象清空复位）。p.mu 保护。
	mounts        []mountEntry
	mountRowsSent int
	riding        bool
	ridingConfig  string // 骑乘中坐骑 configId（空=步行）
	pendingMount  string // USEITEM 挂起 configId（等 266 sub=1 确认后消耗）
	qingGongExtra map[string]struct{} // 秘籍习得的非 starter 轻功（View46 追加行去重）
	// tersure 宝物洗炼会话（688 主链：放入锁 0 起、解锁 1 止；pending
	// 候选未存即丢——重登后客户端重开界面重洗）。p.mu 保护。
	tersure tersureWashSession
	// 轻功经济域（fork Java QingGongHandler/PlayerActor，2026-11-07）：
	// 持续型轻功（UseType=1：踏波行/上天梯/壁虎游墙/轻身缓降）挂秒耗
	// 租约，官方 3s 分档回复在租约期暂停；即时型（UseType=0）引擎逐帧
	// 重发 begin，500ms 同 id 重报闸拦掉重复扣费。
	qingGongLease      *qingGongLease
	qingGongRecoveryMs int64
	instantQingGongAt  map[string]time.Time
	// 跳段反作弊域（2026-11-07 跳段限制批次）：按空格跳跃经 buff 链
	// 限段是客户端机制（学到的轻功决定段数上限，落地重置循环）；服务端
	// 对计段轻功（isJumpStageQingGong：Type 4/5/20）再按腾空计段兜底——
	// 同一腾空内同 id 只放行一次，落地（0x0A 位移 y 回落到着地基准
	// 1m 内）或超时后恢复，技能重置只给一段。
	airborne     bool
	airJumpsUsed map[string]bool
	airJumpGrant int // 技能重置授予的额外段（腾空中放技能 +1，落地清零）
	groundY      float32
	hasGroundY   bool
	lastJumpAt   time.Time
	// 跳段预输入（input buffer）：被跳段闸拒绝的 216 begin 快照。
	// 仅存最新一条——连打时旧输入被新输入顶替（格斗预输入语义），
	// 落地观测拍由位移路径回放（takeBufferedAirJump）。
	pendingAirJump *bufferedAirJump
	// 外观域会话状态（docs/装备外观与风物志挂载.md V4）：
	// 显示类型偏好（custom 61 三 radio，role_appearance 持久化）、
	// 时装/身份装激活集（role_fashion_suits）、皮肤衣橱（role_skins）、
	// 会话级金币（CapitalType0，Java 口径不落库）。
	showEquipType    int
	showEquipTypeSet bool
	fashionSuits     []string // 激活集（升序去重；收录判定大小写不敏感）
	fashionPushed    bool     // FashionSuitRec 行是否已推（购买补行用）
	skins            skinState
	skinsPushed      bool
	skinRows         uint32 // 已推 CardRec 位图行数（整刷前先倒序删）
	skinMountedRows  uint32 // 已推 UseCardRec 挂载行数
	gold             int32  // CapitalType0（会话级；购买扣 1 金）
	silverCard       int32  // CapitalType2（会话级；fork 同口径不落库）
	// 称号域（msg295）：佩戴位挂在玩家对象 TitleID@360；拥有集合与佩戴
	// 状态持久化 role_titles（title_db.go），title_rec 记录行进世界重放。
	titleID     int32
	ownedTitles []int32
	// 经脉域（msg150/161，jingmai_progress.go）：逐经脉进度、修炼位、
	// 激活序（active_jingmai_rec 行镜像）与真气池；快照持久化
	// role_meridians（migrations/0004）。
	jingmai jingMaiState
	// 任务域（task_records.go）：已接（含可交付态）与已完成集合；快照
	// 持久化 role_tasks（migrations/0009），GM authored_task_* 是当前
	// 唯一写入口。
	tasks playerTaskDomain
	// 内功特效脚本分发状态（scripts/module/neigong，见 neigong_dispatch.go）：
	// neigongState 是上一次有效触发回写的扁平状态（脚本互斥窗/层数），
	// shields 为 FIFO 吸收池，dodge 为闪避加成窗（料敌先机），periodicAt
	// 驱动 on_periodic 的 1s 节拍，passive 为 M.passive 静态属性的代数缓存。
	neigongState      map[string]any
	neigongShields    []neigongShield
	neigongDodge      neigongDodgeWindow
	neigongPeriodicAt time.Time
	neigongPassive    neigongPassiveCache
}

// playerMotion 是刻意由服务器持有的 GM 状态。其值通过常规的现代对象
// 属性接收端发送，绝不通过 S->C 0x1E 发送。
type playerMotion struct {
	moveSpeed float32
	runSpeed  float32
	gravity   float32
}

// qingGongLease 是持续型轻功的秒耗租约（Java PlayerActor
// drifting/driftQingGongId/driftDrainPerSecond 三字段合并）：
// 216 begin 已扣 qgdefine Consume，此后每秒按 CCQGP 扣点，
// end/耗尽摘除。
type qingGongLease struct {
	qingGongID     string
	drainPerSecond int32
}

// activePlayerBuff 是单个原生 BufferInfo 槽位在服务器侧的对应物。渲染
// 归客户端所有，但玩法决策必须读取这份状态，而不是信任客户端侧的
// 图标。
type activePlayerBuff struct {
	staticData uint32
	expiresUTC time.Time
	configID   string
	level      int32
}

// playerBuffRules 镜像 buf_fixed_red -> proppack 9679。它刻意不声明
// 伤害免疫：当前配置只授予受击特效免疫和技能锁定免疫。战斗/控制
// 入口将在这些 C2S 路径引入时
// 查询这份状态。
type playerBuffRules struct {
	CantHitEffect     bool
	CantBeSkillLocked bool
}

func newPlayerActor(name string, silver int32) *playerActor {
	learnedSkills := make(map[string]int32, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		learnedSkills[skill.configID] = skill.level
	}
	now := time.Now()
	p := &playerActor{
		name:     name,
		vitals:   sim.NewVitals(1000, 500),
		qinggong: sim.NewQingGong(now),
		speed:    sim.NewMotion(),
		silver:   silver,
		faction:  defaultRoleFaction,
		// 默认移动/奔跑速度 5：与 Java 参考服对齐（2026-11-02 由 25 下调，
		// 25 过快）；GM 面板「移动参数」可运行时改。
		motion:            playerMotion{moveSpeed: 5, runSpeed: 5, gravity: 9.8},
		buffs:             make(map[uint16]activePlayerBuff),
		activeQingGong:    make(map[string]struct{}),
		learnedSkills:     learnedSkills,
		skillCooldowns:    make(map[string]time.Time),
		instantQingGongAt: make(map[string]time.Time),
		airJumpsUsed:      make(map[string]bool),
		progress:          newPlayerProgress(),
		// GM 面板域以种子起步（Java PanelState 同口径）：实力 151 与
		// 选人界面 f23 同档（title027 返璞归真），其余零值合法即下发。
		gm: gmPanelDomain{
			powerLevel: roleListPowerLevel,
		},
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
	return p
}

// snapshotLocked 把三份组件值组合为协议层消费的 entity.ActorState
// DTO，保证 0x10/0x0D 属性投影与迁移前字节一致。
// 调用方必须已持有 p.mu。
func (p *playerActor) snapshotLocked() entity.ActorState {
	state := p.vitals.ActorState(playerObjectID, playerOwnerID)
	state.Name = p.name
	state.QingGongPoint = p.qinggong.Point
	state.MaxQingGongPoint = p.qinggong.Max
	// 镇派兵器"轻功值上限"注入位与 p.qinggong.MaxAdd 同帧（盘点 §4）。
	state.MaxQingGongPointAdd = p.qinggong.MaxAdd + p.progress.zhenpaiEquip.qinggong
	state.QingGongConsumeMul = p.qinggong.ConsumeMul
	state.SpeedRatio = p.speed.SpeedRatio
	return state
}

// stateSnapshot 供不持 p.mu 的调用方（连接循环、combatTick 的
// 日志路径）获取一份一致快照。
func (p *playerActor) stateSnapshot() entity.ActorState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshotLocked()
}

func (p *playerActor) logicState() uint8 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.Logic
}

// logicStateSnapshot 是 logicState 的快照命名变体，与
// currentSkillSnapshot/parryingSnapshot 对齐。
func (p *playerActor) logicStateSnapshot() uint8 { return p.logicState() }

func (p *playerActor) setLogicState(state uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vitals.SetLogicState(state)
}

func (p *playerActor) applyDamage(amount int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.ApplyDamage(amount)
}

func (p *playerActor) spendMP(cost int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.SpendMP(cost)
}

func (p *playerActor) useQingGong(baseCost int32, now time.Time) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.qinggong.Use(baseCost, now)
}

// engageDrift 挂持续轻功秒耗租约（Java PlayerActor.engageDrift 同款）。
// 同 id 续约返回 false——客户端按住冲刺时引擎逐帧重发 begin，
// 逐条扣费会秒清轻功条（2026-11-03 Java 真机反馈同根因）。异 id 新
// begin 直接顶替旧租约（踏波行→上天梯切换不叠加双份秒耗）。
func (p *playerActor) engageDrift(qingGongID string, drainPerSecond int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.qingGongLease != nil && p.qingGongLease.qingGongID == qingGongID {
		return false
	}
	if drainPerSecond < 0 {
		drainPerSecond = 0
	}
	p.qingGongLease = &qingGongLease{qingGongID: qingGongID, drainPerSecond: drainPerSecond}
	return true
}

// releaseDriftIfOwned 仅当租约确属该轻功 id 时摘除（Java handle 在
// flag=0 end 只对同 id 租约有摘除意义；冲刺类引擎每次使用也发 end）。
func (p *playerActor) releaseDriftIfOwned(qingGongID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.qingGongLease == nil || p.qingGongLease.qingGongID != qingGongID {
		return false
	}
	p.qingGongLease = nil
	return true
}

// engagedDriftID 当前持续轻功租约的轻功 id（无租约空串）。
func (p *playerActor) engagedDriftID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.qingGongLease == nil {
		return ""
	}
	return p.qingGongLease.qingGongID
}

// instantQingGongRechargeable 即时型轻功（UseType=0）同 id 重报闸
// （Java INSTANT_QINGGONG_REFIRE_GUARD=500ms）：窗口内的重发视为
// 引擎重报不扣费（人手极限 ~4 次/秒，引擎逐帧重发 10-30/s）。
func (p *playerActor) instantQingGongRechargeable(qingGongID string, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if last, exists := p.instantQingGongAt[qingGongID]; exists && now.Sub(last) < 500*time.Millisecond {
		return false
	}
	p.instantQingGongAt[qingGongID] = now
	return true
}

// 跳段闸参数：落地判定取「腾空后 y 回落到着地基准 1m 内」（0x0A 位移
// 包每帧喂 observePositionY）；腾空超时兜底 15s——重力必然使客户端
// 在数秒内落地，超时仍未观测到回落（位移包缺失/壁虎游墙滞空）时强制
// 重置，防状态卡死误伤下一次地面起跳。
const (
	airJumpLandingEpsilon  = 1.0
	airJumpLandingFallback = 15 * time.Second
)

// beginAirJump 是 216 begin 的服务端跳段闸：同一腾空内同 id 的跳跃型
// 轻功只放行一次；腾空开始时清计数。技能重置（airJumpGrant>0）期间
// 放行并消耗一段——「重置后也只能再跳一段」。
func (p *playerActor) beginAirJump(qingGongID string, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.airborne {
		p.airborne = true
		p.airJumpsUsed = make(map[string]bool)
		p.airJumpGrant = 0
	} else if p.airJumpsUsed[qingGongID] {
		if p.airJumpGrant <= 0 {
			return false
		}
		p.airJumpGrant--
	}
	p.airJumpsUsed[qingGongID] = true
	p.lastJumpAt = now
	return true
}

// grantAirJumpReset 技能施放成功时对腾空玩家的跳跃重置：只授予一段
// （下一次 beginAirJump 放行后恢复拒绝）。着地状态下无意义（落地已
// 全量重置）。
func (p *playerActor) grantAirJumpReset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.airborne {
		p.airJumpGrant++
	}
}

// observePositionY 由 0x0A 位移路径逐帧喂入：着地时持续校准 groundY
// 基准；腾空中 y 回落到基准 1m 内即视为落地——清空跳段计数进入
// 新循环，并返回 true 供调用方回放跳段预输入。跳崖（y 瞬间低于基准）
// 也按落地处理：宁可早重置也不误伤。
func (p *playerActor) observePositionY(y float32, now time.Time) (landed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.airborne {
		p.groundY = y
		p.hasGroundY = true
		return false
	}
	if p.hasGroundY && y <= p.groundY+airJumpLandingEpsilon {
		p.landResetLocked(y)
		return true
	}
	return false
}

// 跳段预输入窗口：被拒 begin 与落地观测之间只隔位移包在途延迟
// （0x0A 逐帧 ≤100ms 一拍），300ms 足够覆盖网络抖动。纯空中连打
// 不会有落地包跟随——窗口过期即静默丢弃，不扣费不执行（与跳段闸
// 拒绝语义一致，蹭跳无利可图）。
const airJumpBufferWindow = 300 * time.Millisecond

// bufferedAirJump 是被跳段闸拒绝后暂存的 216 begin 快照（仅跳段型：
// 过能力门、只败在段位耗尽的输入）。
type bufferedAirJump struct {
	qingGongID     string
	capabilityFree bool
	queuedAt       time.Time
}

// bufferAirJumpInput 记录被跳段闸拒绝的跳段 begin（预输入）。仅存
// 最新一条：连打时旧输入被新输入顶替，落地拍至多回放一次。
func (p *playerActor) bufferAirJumpInput(entry bufferedAirJump) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pendingAirJump = &entry
}

// clearBufferedAirJump 同 id 的 begin 已被正常受理时作废其预输入：
// 该次按键已被服务，落地拍不再回放旧输入（防预输入「复活」早前
// 已被受理挤掉的按次）。
func (p *playerActor) clearBufferedAirJump(qingGongID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pendingAirJump != nil && p.pendingAirJump.qingGongID == qingGongID {
		p.pendingAirJump = nil
	}
}

// takeBufferedAirJump 落地观测拍取出预输入：过期（纯空中连打无落地
// 跟随）或无预输入返回 false。取出即清——一次落地至多回放一条。
func (p *playerActor) takeBufferedAirJump(now time.Time) (bufferedAirJump, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry := p.pendingAirJump
	p.pendingAirJump = nil
	if entry == nil || now.Sub(entry.queuedAt) > airJumpBufferWindow {
		return bufferedAirJump{}, false
	}
	return *entry, true
}

// landResetLocked 清空腾空跳段态并校准着地基准。调用方必须已持 p.mu。
func (p *playerActor) landResetLocked(groundY float32) {
	p.airborne = false
	p.airJumpsUsed = make(map[string]bool)
	p.airJumpGrant = 0
	p.groundY = groundY
	p.hasGroundY = true
}

// qingGongTick 是 1s ticker 的轻功结算（Java qingGongRecoveryTick +
// driftTick 合并，facultyTicker 同拍）：持有秒耗租约时按 CCQGP 扣点、
// 回复暂停；无租约时累积 3s 推一步官方分档回复（101..上限 +10 /
// 51..100 +5 / 0..50 +3）。返回 QingGongPoint 是否变化（调用方推
// 单属性帧）与耗尽强摘的轻功 id（无强摘为空串——调用方须同步摘
// 对应 clearOnEnd Buff，否则客户端表现钉到 buff 寿命尽头）。
func (p *playerActor) qingGongTick(now time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.airborne && now.Sub(p.lastJumpAt) > airJumpLandingFallback {
		// 落地观测兜底：位移包缺失/长时间滞空时强制重置跳段，
		// 防止腾空标志卡死误伤下一次地面起跳。
		p.landResetLocked(p.groundY)
	}
	if p.qingGongLease != nil {
		p.qingGongRecoveryMs = 0 // 消耗期不累积（档位按余量瞬时判定）
		before := p.qinggong.Point
		drain := p.qingGongLease.drainPerSecond
		id := p.qingGongLease.qingGongID
		if !p.qinggong.Drain(drain) {
			p.qingGongLease = nil
			log.Printf("qinggong hold lease ended: point exhausted id=%s", id)
			return p.qinggong.Point != before, id
		}
		return p.qinggong.Point != before, ""
	}
	p.qingGongRecoveryMs += 1000
	if p.qingGongRecoveryMs < 3000 {
		return false, ""
	}
	p.qingGongRecoveryMs -= 3000
	return p.qinggong.RecoverTiered(), ""
}

// qingGongPointFrame 轻功值单属性帧（Java qingGongPointFrame 同款：
// 秒耗/分档回复的逐秒推送；QingGongPoint@692 全表唯一）。
func (p *playerActor) qingGongPointFrame() []byte {
	p.mu.Lock()
	point := p.qinggong.Point
	p.mu.Unlock()
	return sceneObjectIntProperty(playerObjectID, playerOwnerID, clientdata.MustPlayerProperty("QingGongPoint").Index, point) // fork=692
}

func (p *playerActor) learnedSkillLevel(configID string) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	level, exists := p.learnedSkills[configID]
	return level, exists && level > 0
}

func (p *playerActor) learnSkill(configID string, level int32) {
	if configID == "" || level < 1 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.learnedSkills[configID] = level
}

// learnedSkillsSnapshot 返回已学表的副本（GM learn_all_taolu 满级计算用）。
func (p *playerActor) learnedSkillsSnapshot() map[string]int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := make(map[string]int32, len(p.learnedSkills))
	for id, level := range p.learnedSkills {
		snapshot[id] = level
	}
	return snapshot
}

// replaceLearnedSkills 整表替换已学技能等级（GM learn_all_taolu 满级覆写 /
// forget_all_taolu 基线重置）。
func (p *playerActor) replaceLearnedSkills(levels map[string]int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.learnedSkills = levels
}

// skillViewRowOrder 按行号序返回 View40 行的 ConfigID（下标=行号-1；
// GM 等级帧按行序逐条下发用）。
func (p *playerActor) skillViewRowOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	order := make([]string, len(p.skillViewRows))
	for id, row := range p.skillViewRows {
		if row >= 1 && int(row) <= len(order) {
			order[row-1] = id
		}
	}
	return order
}

// progressBookCount 已收录内功书数（GM learn_all_neigong 回执用）。
func (p *playerActor) progressBookCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.progress.books)
}

// mergeDBLearnedSkills 把登录装载的 role_skills 已学集并入释放闸
// 内存表，语义与 mergeLearnedSkills 的并集同口径（starter 优先、
// DB 新行补齐）。View40 下发与 learnedSkillLevel 判定必须出自同一
// 集合，否则 DB 学的技能（如江湖套路）客户端可拖栏施放，服务端
// 却按「未学」拒绝。
func (p *playerActor) mergeDBLearnedSkills(rows []learnedSkillView) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, row := range rows {
		if row.configID == "" || row.level < 1 {
			continue
		}
		if _, exists := p.learnedSkills[row.configID]; exists {
			continue
		}
		p.learnedSkills[row.configID] = row.level
	}
}

func (p *playerActor) progressProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.properties()
}

// factionValue 会话门派读侧（msg973/997 joinFaction 链测试与日志用）。
func (p *playerActor) factionValue() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.faction
}

// setFaction 只在 GM 白名单和持久化外观更新都成功后才修改原生的
// School 属性。现代客户端对初始门派、势力和新门派统一使用这一个
// 属性。
func (p *playerActor) setFaction(faction string) ([]byte, error) {
	p.mu.Lock()
	p.faction = faction
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, factionProperties(faction))
}

// factionProperties 镜像 form_main_player.refresh_role_changed 的逻辑：
// 初始门派填充 School，江湖势力填充 Force，后期门派填充 NewSchool。
// 留空的伴随字段用于刻意清除先前的 GM 选择。
func factionProperties(faction string) []clientdata.IndexedProperty {
	school, force, newSchool := faction, "", ""
	switch {
	case strings.HasPrefix(faction, "force_"):
		school, force = "", faction
	case strings.HasPrefix(faction, "newschool_"):
		school, newSchool = "", faction
	}
	return []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("School").Index, Name: "School", Value: clientdata.StringValue(school)},
		{Index: propForce, Name: "Force", Value: clientdata.StringValue(force)},
		{Index: propNewSchool, Name: "NewSchool", Value: clientdata.StringValue(newSchool)},
	}
}

// setActiveParry 记录一条客户端发起的 218 请求，供未来权威的受击裁定
// 器使用。它自身没有任何客户端视觉副作用：视觉上的招架/受击动作要等
// NPC 或玩家的攻击到达该裁定器之后
// 才会被选定。
func (p *playerActor) setActiveParry(value activeParryRequest) {
	p.mu.Lock()
	p.activeParry = value
	p.mu.Unlock()
}

func (p *playerActor) activeParrySnapshot() activeParryRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeParry
}

type npcAttackOutcome struct {
	stateChanged bool
	parried      bool
	dodged       bool
	dead         bool
	// rageDelta 是本次受击裁定的怒气增量(官方规则:格挡状态被揍 +2、
	// 直接受击 +1;主动招架窗口等同格挡态)。由调用方在锁外编码成
	// 18 字节 SP 单帧。
	rageDelta int32
	// neigongReflect 是 on_victim 特效（reflect 指令）产生的反打值，
	// 由战斗 tick 在锁外反打 NPC（击杀走 NPC 常规死亡链）。
	neigongReflect int32
}

// resolveNPCAttack 是服务器权威的受击裁定。当前客户端已提供 C2S 的
// 主动招架入口（218），但其确切的战斗枚举尚未还原；对本地服务器而言，
// 每条新的 218 请求都授予一个 600ms 的招架窗口。闪避按攻击者和秒
// 确定性生成，重连无法重掷已接受的判定结果。
//
// 内功特效以 scripts/module/neigong 脚本为唯一权威：常驻被动
// （M.passive：幻玉 evasion_add 并入闪避、坐忘 damage_reduction_add 平减）
// 与事件钩子在此结算；格挡 → on_parry（零伤，只落 state/heal/mp/shield）；
// 命中 → on_victim（reduce_pct 折减、盾池 FIFO 吸收、heal/mp、reflect 反打）。
func (p *playerActor) resolveNPCAttack(attackerID uint32, damage int32, now time.Time) npcAttackOutcome {
	p.mu.Lock()
	deadAlready := p.vitals.HP <= 0
	p.mu.Unlock()
	if damage <= 0 || deadAlready {
		return npcAttackOutcome{}
	}
	parry := p.activeParrySnapshot()
	if !parry.At.IsZero() && now.Sub(parry.At) >= 0 && now.Sub(parry.At) <= 600*time.Millisecond {
		// 主动招架窗口内的攻击按"格挡状态被揍"得怒（零伤，特效只落
		// state/heal/mp/shield）。
		triggered := p.dispatchNeiGongParry(attackerID, damage, now)
		return npcAttackOutcome{parried: true, stateChanged: triggered, rageDelta: ragGainParriedHit}
	}
	// 闪避掷 = 基础 + 脚本常驻被动（M.passive evasion_add）+ 闪避窗
	//（dodge_add，料敌先机 Proc 期间）。闪避成功派发 on_dodge（移花接玉）。
	dodgeChance := baseLocalDodgeProbability + p.neigongPassiveEvasion() + p.neigongDodgeAdd(now)
	if combatPercentRoll(attackerID, now, 18) < dodgeChance {
		triggered := p.dispatchNeiGongDodge(attackerID, damage, now)
		return npcAttackOutcome{dodged: true, stateChanged: triggered}
	}
	// 常驻被动平减（damage_reduction_add），floor 1 点（完全招架才零伤）。
	if flat := p.neigongPassiveDamageReduction(); flat > 0 && damage > 1 {
		damage -= flat
		if damage < 1 {
			damage = 1
		}
	}
	applied, reflectAmount, neigongChanged := p.dispatchNeiGongVictim(attackerID, damage, now)
	// 手动格挡(msg54)状态下的实伤同样按格挡态得怒 +2,未格挡 +1。
	ragGain := ragGainUnparriedHit
	if p.parryingSnapshot() {
		ragGain = ragGainParriedHit
	}
	p.mu.Lock()
	changed := applied > 0 && p.vitals.ApplyDamage(applied)
	if !changed {
		dead := p.vitals.HP <= 0
		p.mu.Unlock()
		// 盾全吸收或零伤：不落血，但特效可能已改血/蓝/状态。
		if neigongChanged {
			return npcAttackOutcome{stateChanged: true, neigongReflect: reflectAmount}
		}
		return npcAttackOutcome{dead: dead}
	}
	// 实际吃到伤害即战斗活动：驱动 10s 脱战迟滞窗口。
	p.lastCombatAt = now
	dead := p.vitals.HP <= 0
	if dead {
		p.reviveAt = now.Add(playerAutoReviveDelay)
	}
	p.mu.Unlock()
	rules := p.buffRules(now)
	if !rules.CantBeSkillLocked {
		p.mu.Lock()
		p.skillLockUntil = now.Add(500 * time.Millisecond)
		p.mu.Unlock()
	}
	return npcAttackOutcome{stateChanged: true, dead: dead, rageDelta: ragGain, neigongReflect: reflectAmount}
}

// combatPercentRoll 对同一攻击者和同一服务器 tick 是确定性的。它让
// 本地测试保持可复现，同时不让重连重掷一次已经结算的 NPC 攻击。
func combatPercentRoll(attackerID uint32, now time.Time, salt uint64) int32 {
	value := uint64(attackerID)*1103515245 + uint64(now.Unix())*2654435761 + salt*97
	return int32(value % 100)
}

func (p *playerActor) skillLocked(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skillLockUntil.After(now)
}

// reviveWhenDue 实现本地测试的复活策略。它刻意使用常规角色属性；官方
// 确切的复活 UI/传送路由是一个独立的协议特性，此处不做虚构。
func (p *playerActor) reviveWhenDue(now time.Time) bool {
	p.mu.Lock()
	if p.reviveAt.IsZero() || now.Before(p.reviveAt) {
		p.mu.Unlock()
		return false
	}
	p.reviveAt = time.Time{}
	p.skillLockUntil = time.Time{}
	// 完整复活迁移：数值回满 + 轻功条回满 + 速度比重置，
	// 等价于原 entity.Actor.Revive 的全语义。持续轻功租约一并
	// 摘除（复活点数已回满，旧租约会立刻再抽干）。
	p.vitals.Revive()
	p.qinggong.Reset(now)
	p.qingGongLease = nil
	p.qingGongRecoveryMs = 0
	p.speed = sim.NewMotion()
	p.mu.Unlock()
	return true
}

func (p *playerActor) progressViewFrames() ([][]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.viewFrames()
}

func (p *playerActor) equipNeiGong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.equip(id); err != nil {
		return err
	}
	// 换书即换特效：旧书的脚本状态窗/盾池/闪避窗/被动缓存不跨书存活。
	p.neigongState = nil
	p.neigongShields = nil
	p.neigongDodge = neigongDodgeWindow{}
	delete(p.buffs, neigongProcBuffSlot)
	p.neigongPassive = neigongPassiveCache{}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return nil
}

// syncEquippedNeiGongResourcesLocked 将官方全量资源 Add（五维属性分量
// + 已装备秘籍 pack 分量，playerProgress.derivedResourceAdds）应用到
// 真正的服务器 Actor 上。可见协议仍会发送基础 MaxHP/MaxMP 加原始的
// *Add 字段，但所有服务器端的消耗、伤害和恢复都使用 Actor.MaxHP/
// MaxMP（即推导后的总上限）。当前现代客户端直接按 MaxMP + MaxMPAdd
// 刷新 MP；线上值没有乘数。
//
// 调用方必须已持有 p.mu。五维基础值（GM set_five_attributes）或装备
// 内功变化后都必须重调。
func (p *playerActor) syncEquippedNeiGongResourcesLocked() {
	hpAdd, mpAddRaw := p.progress.derivedResourceAdds()
	p.vitals.SetDerivedMaxima(p.vitals.BaseMaxHP+hpAdd, p.vitals.BaseMaxMP+mpAddRaw)
}

// syncEquippedNeiGongBuffLocked 将已装备秘籍由已安装 varprop 推导出的
// 确切效果镜像到一个专用原生 BufferInfo 槽位。这刻意不用服务器发起的
// 0x1E 自定义消息：FxGameLogic 通过普通对象属性消费 BufferInfo 和
// BufferListStr。
func (p *playerActor) syncEquippedNeiGongBuffLocked(now time.Time) {
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.buffStaticData == 0 || book.buffLevel <= 0 {
		delete(p.buffs, innerPowerBuffSlot)
		return
	}
	p.buffs[innerPowerBuffSlot] = activePlayerBuff{
		staticData: book.buffStaticData,
		expiresUTC: now.UTC().Add(innerPowerBuffLease),
		configID:   book.buffID,
		level:      book.buffLevel,
	}
}

func (p *playerActor) facultyReady(id string) (uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.facultyReady(id)
}

func (p *playerActor) facultyBegin(style int32) (uint16, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.facultyBegin(style, time.Now()); err != nil {
		return 0, false, err
	}
	// 普通内修是一种持久的后台转换，不得强制角色进入 LS_FACULTY。只有
	// 主动的演武路径才拥有那个可见的逻辑/动作状态。
	if style == facultyStyleAct {
		p.vitals.SetLogicState(logicStateFaculty)
	} else {
		p.vitals.SetLogicState(0)
	}
	return 0, false, nil
}

func (p *playerActor) facultyExit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.facultyExit()
	p.vitals.SetLogicState(0)
}

// bookUpdateFrame View43 行级增量帧（已存在行；武学面板升级后刷新的
// 官方方言，见 playerProgress.bookUpdateFrame）。
func (p *playerActor) bookUpdateFrame(slot uint16) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.bookUpdateFrame(slot)
}

func (p *playerActor) facultyTick(now time.Time) (uint16, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot, changed := p.progress.advanceFaculty(now)
	if changed {
		// 一次结算可能跨越一级或多级。确切的 varprop 记录已由
		// advanceFaculty 重新求值；在调用方发送常规的角色属性/View 刷新
		// 之前，先发布它的原生 BuffStore 状态，并让新的推导上限
		// 成为权威值。
		p.syncEquippedNeiGongBuffLocked(now)
		p.syncEquippedNeiGongResourcesLocked()
	}
	return slot, changed
}

// livepointTick 历练层 100 秒步进（facultyTick 同拍）：累积 +1 历练/100s
// 进槽、按槽速转化产修为（普通档 70% + 30% 潜能）。返回是否有变化。
func (p *playerActor) livepointTick(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.livepointTick(now, livepointVIP)
}

// buyHoldFaculty custom 235/2：N 黄金 → N×price 潜能修为转正式修为。
// count≤0/黄金不足/潜能不足静默拒绝（客户端同校验，双保险防刷）。
func (p *playerActor) buyHoldFaculty(count int32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if count <= 0 || count > p.gold {
		return false
	}
	amount := int64(count) * int64(livepointPrice)
	if amount > int64(p.progress.holdFaculty) {
		return false
	}
	p.gold -= count
	p.progress.holdFaculty -= int32(amount)
	if sum := int64(p.progress.faculty) + amount; sum > math.MaxInt32 {
		p.progress.faculty = math.MaxInt32
	} else {
		p.progress.faculty = int32(sum)
	}
	return true
}

// addLivepoint GM add_livepoint：发 N 点历练（×1000 毫按序填槽）。
// 返回实际入槽毫数。
func (p *playerActor) addLivepoint(points int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.addLivepoint(points)
}

// setHoldFaculty GM set_hold_faculty：潜能修为直设。
func (p *playerActor) setHoldFaculty(value int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.setHoldFaculty(value)
}

// setLiveSpeedPower GM set_live_speed_power：历练转化速度加成 %。
func (p *playerActor) setLiveSpeedPower(value int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.setLiveSpeedPower(value)
}

func (p *playerActor) facultySnapshot() facultyProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	value := p.progress.snapshot()
	// GM set_vitals 的生存属性层随 role_faculty 快照走（Java vitals 节点
	// 同口径）：仅本会话设置过后才落节点，旧档保持 nil。
	if p.gm.vitalsSet {
		value.Vitals = &facultyVitalsSnapshot{
			Ene: p.gm.vitals.ene, MaxEne: p.gm.vitals.maxEne, ResumeEne: p.gm.vitals.resumeEne,
			Sat: p.gm.vitals.sat, MaxSat: p.gm.vitals.maxSat,
			PKValue: p.gm.vitals.pkValue, DrunkLevel: p.gm.vitals.drunkLevel,
		}
	}
	return value
}

func (p *playerActor) restoreFaculty(value facultyProgressSnapshot, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.restore(value, now)
	if value.Vitals != nil {
		p.gm.vitals = gmVitalsState{
			ene: value.Vitals.Ene, maxEne: value.Vitals.MaxEne, resumeEne: value.Vitals.ResumeEne,
			sat: value.Vitals.Sat, maxSat: value.Vitals.MaxSat,
			pkValue: value.Vitals.PKValue, drunkLevel: value.Vitals.DrunkLevel,
		}
		p.gm.vitalsSet = true
	}
	if p.progress.facultyState == facultyStateConvert && p.progress.facultyStyle == facultyStyleAct {
		p.vitals.SetLogicState(logicStateFaculty)
	} else {
		p.vitals.SetLogicState(0)
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
}

// activateQingGong 为本场景会话记住一条服务器批准的 ActiveQGSkillRec
// 条目。客户端在启用已装备的 qinggong_gaojie（高级轻功）技能之前会
// 检查这张表。
func (p *playerActor) activateQingGong(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.activeQingGong[id]; exists {
		return false
	}
	p.activeQingGong[id] = struct{}{}
	return true
}

func (p *playerActor) stateProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := p.snapshotLocked()
	now := time.Now()
	hpRecovery := sitcrossRecoveryPerSecond(state.MaxHP)
	mpRecovery := sitcrossRecoveryPerSecond(state.MaxMP)
	// 官方 *UpSpeedAdd 位是五维的分数系数分量（12173/55866 抓包：
	// round(0.07·Sta)、round(0.2·Spi)），*UpSpeed 基础位则是总上限的
	// ceil(总/10)——sitcrossRecoveryPerSecond 已是同式。
	_, staEff, _, _, spiEff := p.progress.effectiveFive()
	hpUpSpeedAdd := staHundredthsAdd(staEff)
	mpUpSpeedAdd := spiTenthsAdd(spiEff)
	parryTotal := p.progress.parryValueTotal()
	currentSkillID, currentSkillTarget := p.currentSkillID, p.currentSkillTarget
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	hitHPRatio := int32(int64(state.HitHP) * 100 / int64(state.MaxHP))
	mpRatio := int32(int64(state.MP) * 100 / int64(state.MaxMP))
	properties := []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("LogicState").Index, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: propHPRatio, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: propMPRatio, Name: "MPRatio", Value: clientdata.Int32Value(mpRatio)},
		// 客户端用这些基础值加上 MaxHPAdd/MaxMPAdd 推导显示的总量。
		// Actor.Max* 仍是服务器侧的总上限。
		{Index: clientdata.MustPlayerProperty("MaxHP").Index, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: clientdata.MustPlayerProperty("MaxMP").Index, Name: "MaxMP", Value: clientdata.Int32Value(state.BaseMaxMP)},
		{Index: propHP, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: propMP, Name: "MP", Value: clientdata.Int32Value(state.MP)},
		{Index: clientdata.MustPlayerProperty("SpeedRatio").Index, Name: "SpeedRatio", Value: clientdata.Int32Value(state.SpeedRatio)},
		{Index: propHitHP, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: propHitHPRatio, Name: "HitHPRatio", Value: clientdata.Int32Value(hitHPRatio)},
		{Index: clientdata.MustPlayerProperty("QingGongPoint").Index, Name: "QingGongPoint", Value: clientdata.Int32Value(state.QingGongPoint)},
		{Index: clientdata.MustPlayerProperty("MaxQingGongPoint").Index, Name: "MaxQingGongPoint", Value: clientdata.Int32Value(state.MaxQingGongPoint)},
		{Index: clientdata.MustPlayerProperty("MaxQingGongPointAdd").Index, Name: "MaxQingGongPointAdd", Value: clientdata.Int32Value(state.MaxQingGongPointAdd)},
		{Index: propNeigongPKStatus, Name: "NeigongPKStatus", Value: clientdata.WordValue(0)},
		{Index: propDead, Name: "Dead", Value: clientdata.ByteValue(boolByte(state.HP <= 0))},
		{Index: propInParry, Name: "InParry", Value: clientdata.Int32Value(boolInt32(p.parrying))},
		{Index: propInManualParry, Name: "InManualParry", Value: clientdata.Int32Value(boolInt32(p.parrying))},
		// 格挡族登录补位：官方首帧 0x10 同拍必发（472=0/478=100/479=4，
		// 三台官方系抓包逐点一致）。ParryRatio/ParryUpSpeed 官方只此
		// 一发、之后不再重推，公式未定位（样本恒 100/4）。
		{Index: propCantParry, Name: "CantParry", Value: clientdata.Int32Value(0)},
		{Index: propParryRatio, Name: "ParryRatio", Value: clientdata.Int32Value(100)},
		{Index: propParryUpSpeed, Name: "ParryUpSpeed", Value: clientdata.Int32Value(4)},
		// ParryValue@475 官方为 MaxParry(100)+MaxParryAdd 的推导和
		//（抓包 116/157 逐点吻合），不再用静态参考值。
		{Index: propParryValue, Name: "ParryValue", Value: clientdata.Int32Value(parryTotal)},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue(currentSkillID)},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.Value{Type: clientdata.WireObject, U64: currentSkillTarget}},
		// ModifySkillLockTime 官方口径 200（Java PlayerActor #2014 exe
		// 反汇编钉死；此前 5000 是「本地容差兜底」占位，会拉长客户端
		// 技能锁定窗口）。
		{Index: propModifySkillLockTime, Name: "ModifySkillLockTime", Value: clientdata.Int32Value(200)},
		{Index: propHPHeartSpeed, Name: "HPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propHPHeartSpeedAdd, Name: "HPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propMPHeartSpeed, Name: "MPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propMPHeartSpeedAdd, Name: "MPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propHPUpSpeed, Name: "HPUpSpeed", Value: clientdata.Int32Value(hpRecovery)},
		{Index: propHPUpSpeedAdd, Name: "HPUpSpeedAdd", Value: clientdata.Int32Value(hpUpSpeedAdd)},
		{Index: propMPUpSpeed, Name: "MPUpSpeed", Value: clientdata.Int32Value(mpRecovery)},
		{Index: propMPUpSpeedAdd, Name: "MPUpSpeedAdd", Value: clientdata.Int32Value(mpUpSpeedAdd)},
		{Index: clientdata.MustPlayerProperty("Name").Index, Name: "Name", Value: clientdata.WideStringValue(state.Name)},
		// 官方玩家域 CapitalType1@417 线型 4(int64):货币按小端
		// 八字节编码(真机抓包实证)。
		{Index: clientdata.MustPlayerProperty("CapitalType1").Index, Name: "CapitalType1", Value: clientdata.Int64Value(int64(p.silver))},
	}
	// BufferListStr(fork-only@143)：官服表无此名（buff 条官方走
	// BufferInfo CSV 槽+m169，§14.5），官服方言停发整帧。
	if bufferList, ok := clientdata.PlayerProperty("BufferListStr"); ok {
		properties = append(properties, clientdata.IndexedProperty{
			Index: bufferList.Index, Name: bufferList.Name,
			Value: clientdata.StringValue(p.bufferListStringLocked(now)),
		})
	}
	properties = append(properties, factionProperties(p.faction)...)
	for _, slot := range nativeBuffSlots {
		properties = append(properties, clientdata.IndexedProperty{
			Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Value: clientdata.StringValue(p.bufferInfoLocked(slot, now)),
		})
	}
	properties = append(properties, baseMotionProperties(p.motion)...)
	properties = append(properties, qinggongMotionProperties()...)
	properties = append(properties, p.progress.properties()...)
	// 经脉面板 8 属性（Java PlayerActor.stateProperties 同位，progress 块
	// 之后）：进对象属性批即带，面板首开就有 CurJingMai/总等级，不必等
	// ClientReady 后 pushJingMaiFrames 的独立 0x10。
	return append(properties, p.jingMaiPropsLocked()...)
}

func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func boolByte(value bool) byte {
	if value {
		return 1
	}
	return 0
}

// baseMotionProperties 提供 FxGameLogic 在创建普通跳跃动作时读取的
// 玩家物理参数。客户端对 Gravity 没有服务器侧的兜底来源：只有当这个
// 协商属性存在时它才会应用该值。9.8 是当前客户端使用的
// 标准跳跃重力。
func baseMotionProperties(motion playerMotion) []clientdata.IndexedProperty {
	return []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Gravity").Index, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
		{Index: clientdata.MustPlayerProperty("GravityAdd").Index, Name: "GravityAdd", Value: clientdata.Float32Value(0)},
		// 现代 FxGameLogic 的 FallDownManager 会从本地玩家对象查询
		// DropHeightPub。它的兜底值只有 5.0 个世界单位；这里刻意使用
		// 偏高的本地测试安全阈值，让普通的地形降落退出回着地动作，而非
		// 进入倒地动作。 TODO: 后面改GM参数控制，默认 5.0
		{Index: clientdata.MustPlayerProperty("DropHeightPub").Index, Name: "DropHeightPub", Value: clientdata.Float32Value(100000)},
	}
}

func (p *playerActor) motionSnapshot() playerMotion {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.motion
}

// setMoveSpeed 和 setGravity 返回单个常规 0x10 更新档案。MoveSpeed 与
// RunSpeed 一并修改，使 GM 控制对行走和奔跑只有一个明确的含义。
func (p *playerActor) setMoveSpeed(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.moveSpeed = value
	p.motion.runSpeed = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("MoveSpeed").Index, Name: "MoveSpeed", Value: clientdata.Float32Value(motion.moveSpeed)},
		{Index: clientdata.MustPlayerProperty("RunSpeed").Index, Name: "RunSpeed", Value: clientdata.Float32Value(motion.runSpeed)},
	})
}

func (p *playerActor) setGravity(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.gravity = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Gravity").Index, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
	})
}

// activateRedSuperArmor 开启一段权威的红色霸体租约。其返回值是供
// FxGameLogic 的 BuffStore 设置器消费的原生 BufferInfo1 载荷：
// StaticData,OwnerID,OwnerSerial,Level,EndTimeMs,Count（均为十六进制）。
func (p *playerActor) activateRedSuperArmor(now time.Time) (string, time.Time) {
	return p.activateBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, 20*time.Second, now)
}

func (p *playerActor) activateBuff(slot uint16, staticData uint32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{staticData: staticData, expiresUTC: expires, level: 1}
	return p.bufferInfoLocked(slot, now), expires
}

func (p *playerActor) activateNamedBuff(slot uint16, staticData uint32, configID string, lifetime time.Duration, now time.Time) (string, time.Time) {
	return p.activateNamedBuffAtLevel(slot, staticData, configID, 1, lifetime, now)
}

func (p *playerActor) activateNamedBuffAtLevel(slot uint16, staticData uint32, configID string, level int32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if level < 1 {
		level = 1
	}
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{
		staticData: staticData,
		expiresUTC: expires,
		configID:   configID,
		level:      level,
	}
	return p.bufferInfoLocked(slot, now), expires
}

// activateSkillBuff 将由资源编译出的技能 Buff 投影到原生 BufferInfo1..24
// 存储中。系统租约保留 1..10 槽位；常规 Buff 复用其现有 ConfigID 槽位，
// 否则从 11..24 分配。
func (p *playerActor) activateSkillBuff(definition skillBuffDefinition, now time.Time) (uint16, string, time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := definition.preferredSlot
	if slot == 0 {
		for candidate, active := range p.buffs {
			if active.configID == definition.configID {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 {
		for candidate := firstAutoBuffSlot; candidate <= bufferInfoSlotCount; candidate++ {
			active, occupied := p.buffs[candidate]
			if !occupied || !active.expiresUTC.After(now.UTC()) {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 || slot > bufferInfoSlotCount {
		return 0, "", time.Time{}, false
	}
	lifetime := definition.lifetime
	if lifetime <= 0 {
		lifetime = time.Millisecond
	}
	expires := now.UTC().Add(lifetime)
	level := definition.level
	if level < 1 {
		level = 1
	}
	p.buffs[slot] = activePlayerBuff{
		staticData: definition.staticData,
		expiresUTC: expires,
		configID:   definition.configID,
		level:      level,
	}
	return slot, p.bufferInfoLocked(slot, now), expires, true
}

func (p *playerActor) nextSkillBuffVariant(variants []skillBuffDefinition, now time.Time) (skillBuffDefinition, bool) {
	if len(variants) == 0 {
		return skillBuffDefinition{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := variants[0].preferredSlot
	active, exists := p.buffs[slot]
	if !exists || !active.expiresUTC.After(now.UTC()) {
		return variants[0], true
	}
	for index, variant := range variants {
		if active.staticData == variant.staticData {
			return variants[(index+1)%len(variants)], true
		}
	}
	return variants[0], true
}

func (p *playerActor) bufferInfo(slot uint16, now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferInfoLocked(slot, now)
}

func (p *playerActor) bufferInfoLocked(slot uint16, now time.Time) string {
	buff, ok := p.buffs[slot]
	if !ok || !buff.expiresUTC.After(now.UTC()) {
		return ""
	}
	level := buff.level
	if level < 1 {
		level = 1
	}
	return fmt.Sprintf("%X,%X,%X,%X,%X,%X", buff.staticData, playerObjectID, playerOwnerID, level, buff.expiresUTC.UnixMilli(), 1)
}

// bufferListString 遵循已保留的 BufferModule::RefreshBuffersInfo 布局：
// ConfigID,UniqueID,Level,OwnerIdent-OwnerSerial,EndTime,EffectType,
// OverlayRows|。它与 BufferInfo 刻意由同一批服务器权威租约生成，因此
// 玩法与视觉的 Buff 状态不会分叉。
func (p *playerActor) bufferListString(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferListStringLocked(now)
}

func (p *playerActor) bufferListStringLocked(now time.Time) string {
	var list strings.Builder
	for _, slot := range nativeBuffSlots {
		buff, ok := p.buffs[slot]
		if !ok || !buff.expiresUTC.After(now.UTC()) {
			continue
		}
		configID := buff.configID
		if configID == "" {
			configID, ok = activeBuffConfigID(buff.staticData)
		}
		if !ok {
			continue
		}
		level := buff.level
		if level < 1 {
			level = 1
		}
		fmt.Fprintf(&list, "%s,local-%d,%d,%d-%d,%d,1,0|", configID, slot, level, playerObjectID, playerOwnerID, buff.expiresUTC.UnixMilli())
	}
	p.appendTimedBuffsLocked(&list, now)
	return list.String()
}

func activeBuffConfigID(staticData uint32) (string, bool) {
	switch staticData {
	case redSuperArmorStaticData:
		return "buf_fixed_red", true
	case waterDriftStaticData:
		return "buf_qg_drift", true
	case waterJumpStaticData:
		return "buf_qg_dst", true
	default:
		return "", false
	}
}

func (p *playerActor) hasActiveBuff(slot uint16, staticData uint32, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	return ok && buff.staticData == staticData && buff.expiresUTC.After(now.UTC())
}

// buffExpiry 读取槽位租约的到期时刻（msg169 缓降摘帧 v7 取租约原到期
// 而非当前时刻——Java releaseGlide 的 glideEndsAt 同口径）；无匹配
// 活跃租约返回 ok=false，调用方回落 now。
func (p *playerActor) buffExpiry(slot uint16, staticData uint32, now time.Time) (time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != staticData || !buff.expiresUTC.After(now.UTC()) {
		return time.Time{}, false
	}
	return buff.expiresUTC, true
}

func (p *playerActor) clearBuff(slot uint16, expectedStaticData uint32) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

// --- 手动格挡（msg54；Java PlayerActor engage/releaseParryBuff 同源） ---

func (p *playerActor) parryingSnapshot() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.parrying
}

// snapshotName 供 msg169 双宽串等帧使用。
func (p *playerActor) snapshotName() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.name
}

// engageParryBuff 按下格挡：挂 BuffInParry 到槽 11。已在格挡时保留
// 原到期时间（幂等由调用方的状态变化检查保证，这里再防一层）。
func (p *playerActor) engageParryBuff(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.parrying {
		p.buffs[parryBuffSlot] = activePlayerBuff{
			staticData: parryBuffStaticData,
			expiresUTC: now.UTC().Add(24 * time.Hour),
			level:      1,
		}
		p.parrying = true
	}
	return p.bufferInfoLocked(parryBuffSlot, now)
}

// releaseParryBuff 松开格挡：清槽 11，返回空 BufferInfo 串。
func (p *playerActor) releaseParryBuff() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.buffs, parryBuffSlot)
	p.parrying = false
	return ""
}

// parryStateFrame 回格挡属性帧（0x10）：InParry/InManualParry 置位 +
// BufferInfo11 槽串 + 当前格挡值（Java parryStateFrame 同形）。
func (p *playerActor) parryStateFrame(engage bool, buffInfo string) ([]byte, error) {
	flag := int32(0)
	if engage {
		flag = 1
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propInParry, Name: "InParry", Value: clientdata.Int32Value(flag)},
		{Index: propInManualParry, Name: "InManualParry", Value: clientdata.Int32Value(flag)},
		{Index: bufferInfoPropertyIndex(parryBuffSlot), Name: bufferInfoPropertyName(parryBuffSlot), Value: clientdata.StringValue(buffInfo)},
		{Index: propParryValue, Name: "ParryValue", Value: clientdata.Int32Value(p.progress.parryValueTotal())},
	})
}

// expireBuff 只清除创建此定时器的那份确切租约。后续的刷新绝不会被一个
// 已过期的定时器移除。
func (p *playerActor) expireBuff(slot uint16, expectedStaticData uint32, expected time.Time, now time.Time) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData || !buff.expiresUTC.Equal(expected) || buff.expiresUTC.After(now.UTC()) {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

func (p *playerActor) buffRules(now time.Time) playerBuffRules {
	red := p.hasActiveBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, now)
	return playerBuffRules{CantHitEffect: red, CantBeSkillLocked: red}
}

// qinggongMotionProperties 的取值来源是当前已安装的
// share/Skill/QingGong/QGDefine.ini。客户端不会从 216 动作请求推导
// 这些值：它在解析动作之前先从自己的玩家对象读取。请保持基础属性与
// Add 属性分离，因为 Lua 和 FxGameLogic 都会成对查询两者。
func qinggongMotionProperties() []clientdata.IndexedProperty {
	const (
		waterSpeed      float32 = 7.5 // qinggong_2 Para1
		waterJumpSpeed  float32 = 6.0 // qinggong_3 Para1
		secondJumpSpeed float32 = 6.0 // qinggong_4 Para1
		thirdJumpSpeed  float32 = 6.0 // qinggong_5 Para1
		airRushSpeed    float32 = 15  // qinggong_6 Para1
		airRushDist     float32 = 12  // qinggong_6 Para2
		// 实机测试确认类型 7 的 Para 顺序是先速度、后距离：Para1=25 产生
		// 预期的快速雁行功冲刺，而 Para2=6.5 是它的行进距离。这与类型 6
		// 的空中冲刺参数顺序不同。
		landRushSpeed float32 = 25  // qinggong_7 Para1
		landRushDist  float32 = 6.5 // qinggong_7 Para2
		climbSpeed    float32 = 7.5 // qinggong_11 Para1
		// 这些并不是上面三个动作速度值的别名。FxGameLogic 在冲刺动作的
		// 门控中使用 share/Skill/rush_range.ini，包括 qinggong_3 / 点水跳。
		// 在这里保留历史上以速度冒充距离的占位值，会让那个原生门控按
		// 错误的水上距离求值。
		groundRushRange    float32 = 5.0 // rush_range.ini GRushRange
		airRushRange       float32 = 8.0 // rush_range.ini ARushRange
		waterRushRange     float32 = 8.0 // rush_range.ini WRushRange
		groundRushRangeAdd float32 = 1.5 // rush_range.ini GRushRangeAdd
		airRushRangeAdd    float32 = 6.5 // rush_range.ini ARushRangeAdd
		waterRushRangeAdd  float32 = 6.5 // rush_range.ini WRushRangeAdd
	)
	f := func(index uint16, name string, value float32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: index, Name: name, Value: clientdata.Float32Value(value)}
	}
	return []clientdata.IndexedProperty{
		f(clientdata.MustPlayerProperty("RunSpeedAdd").Index, "RunSpeedAdd", 0), f(clientdata.MustPlayerProperty("JumpSpeedAdd").Index, "JumpSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("DriftSpeed").Index, "DriftSpeed", waterSpeed), f(clientdata.MustPlayerProperty("DriftSpeedAdd").Index, "DriftSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("DriftJumpSpeed").Index, "DriftJumpSpeed", waterJumpSpeed), f(clientdata.MustPlayerProperty("DriftJumpSpeedAdd").Index, "DriftJumpSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("SndJumpSpeed").Index, "SndJumpSpeed", secondJumpSpeed), f(clientdata.MustPlayerProperty("SndJumpSpeedAdd").Index, "SndJumpSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("ThdJumpSpeed").Index, "ThdJumpSpeed", thirdJumpSpeed), f(clientdata.MustPlayerProperty("ThdJumpSpeedAdd").Index, "ThdJumpSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("AirRushSpeed").Index, "AirRushSpeed", airRushSpeed), f(clientdata.MustPlayerProperty("AirRushSpeedAdd").Index, "AirRushSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("AirRushDist").Index, "AirRushDist", airRushDist), f(clientdata.MustPlayerProperty("AirRushDistAdd").Index, "AirRushDistAdd", 0),
		f(clientdata.MustPlayerProperty("LandRushSpeed").Index, "LandRushSpeed", landRushSpeed), f(clientdata.MustPlayerProperty("LandRushSpeedAdd").Index, "LandRushSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("LandRushDist").Index, "LandRushDist", landRushDist), f(clientdata.MustPlayerProperty("LandRushDistAdd").Index, "LandRushDistAdd", 0),
		f(clientdata.MustPlayerProperty("ClimbSpeed").Index, "ClimbSpeed", climbSpeed), f(clientdata.MustPlayerProperty("ClimbSpeedAdd").Index, "ClimbSpeedAdd", 0),
		f(clientdata.MustPlayerProperty("GRushRange").Index, "GRushRange", groundRushRange), f(clientdata.MustPlayerProperty("GRushRangeAdd").Index, "GRushRangeAdd", groundRushRangeAdd),
		f(clientdata.MustPlayerProperty("ARushRange").Index, "ARushRange", airRushRange), f(clientdata.MustPlayerProperty("ARushRangeAdd").Index, "ARushRangeAdd", airRushRangeAdd),
		f(clientdata.MustPlayerProperty("WRushRange").Index, "WRushRange", waterRushRange), f(clientdata.MustPlayerProperty("WRushRangeAdd").Index, "WRushRangeAdd", waterRushRangeAdd),
	}
}

func (p *playerActor) vitalUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.stateProperties())
}

// combatVitalsFrame 战斗 tick 专用的小面板（纯数值 vitals，n≈10）：
// 官服战斗中从不重推全量面板——全量含 State="stand"/LogicState/
// CurSkillID/CurSkillTarget/PauseSkill，会把冲锋受理簇刚设的接近态
// 洗掉（53051 会话实证：簇后 1s 内全量面板到达 → 客户端退出自动接近、
// 220 冻结刷）。战斗血量官方通道是 0x2D/c1=7/SP 单帧（broadcastNPCAttacks
// 已发），这里只补数值 vitals 同步。
func (p *playerActor) combatVitalsFrame() ([]byte, error) {
	p.mu.Lock()
	state := p.snapshotLocked()
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	props := []clientdata.IndexedProperty{
		{Index: propHPRatio, Name: "HPRatio", Value: clientdata.Int32Value(int32(int64(state.HP) * 100 / int64(state.MaxHP)))},
		{Index: propMPRatio, Name: "MPRatio", Value: clientdata.Int32Value(int32(int64(state.MP) * 100 / int64(state.MaxMP)))},
		{Index: propHP, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: propMP, Name: "MP", Value: clientdata.Int32Value(state.MP)},
		{Index: propHitHP, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: propHitHPRatio, Name: "HitHPRatio", Value: clientdata.Int32Value(int32(int64(state.HitHP) * 100 / int64(state.MaxHP)))},
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
		{Index: propQingGongPointIndex, Name: "QingGongPoint", Value: clientdata.Int32Value(state.QingGongPoint)},
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, props)
}

// rageCap 是服务器侧怒气饱和用的有效上限(基础+加成)。GM 满怒与
// 战斗日志消费同一入口。
func (p *playerActor) rageCap() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.rageCap()
}

// rageCapWireValue 计算最终随 MaxSP(580) 下发的怒气上限值:基础 80 +
// 运行内功档位加成折算合一(基础值保持 80 不变)。官服口径(docs/
// official 三份时间线 2026-11-09 复核):官方全程只下发 MaxSP(登录
// 4.7s MaxSP=50 ×2),MaxSPAdd(583) 零下发——加成在服务器侧折进
// MaxSP 一次给出。客户端怒气球分母 = MaxSP + MaxSPAdd,583 不出帧
// 时客户端按默认 0 参与,分母与本服有效上限一致。
func (p *playerActor) rageCapWireValue() int32 {
	return p.rageCap()
}

// rageCapUpdate 生成只含 MaxSP(580,折算最终值)的单次 0x10 专帧,
// 与纯 SP 事件帧同形。用于 215 换书/升级抬档后向客户端刷新怒气球
// 分母——MaxSP 不得进入常规大批/0x0D 归档,专帧是登录全量批之外
// 唯一的合法载体。
func (p *playerActor) rageCapUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propMaxSP, Name: "MaxSP", Value: clientdata.Int32Value(p.rageCapWireValue())},
	})
}

// loginVitalUpdate 是登录/切图链的首个权威 0x10,也是 MaxSP 全量批
// 形态的唯一下发点:MaxSP 为 fork 抓包实证(49840 帧全场 5181 帧
// 仅 #16 一处,121 属性全量批携带,fork 值 80;Java 参考服同为登录一次)。
// 本服在此批只携带 MaxSP 一条,值 = rageCapWireValue(基础 80 + 运行
// 内功档位加成;官方抓包全程无 MaxSPAdd(583),加成折进 580,基础值
// 80 不变)。之后的恢复批/事件批都不再携带;把上限塞进 0x0D 归档
// 或周期批会让客户端在进世界时崩溃。
// SP(467) 本就在 stateProperties 大批内,无独立登录怒气帧;这里只是
// 在同一批上补一次上限声明。
func (p *playerActor) loginVitalUpdate() ([]byte, error) {
	properties := p.stateProperties()
	properties = append(properties,
		clientdata.IndexedProperty{Index: propMaxSP, Name: "MaxSP", Value: clientdata.Int32Value(p.rageCapWireValue())},
	)
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, properties)
}

// sitcrossTick 只推进服务器持有的 HP/MP 状态，并返回逐层结算明细；
// 调用方按官方形态出帧（sitcrossRecoveryFrames：n=1 单属性帧 +
// msg24 c1=16 治疗飘字），客户端绝不会自行凭 HPHeartSpeed/
// MPHeartSpeed 虚构资源值。
func (p *playerActor) sitcrossTick() sim.SitcrossSettlement {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.RestoreWhileSit(sitcrossRecoveryPerSecond(p.vitals.MaxHP), sitcrossRecoveryPerSecond(p.vitals.MaxMP))
}

// sitcrossRecoveryFrames 生成打坐回复一跳的官方形态帧簇（FsGame
// 官服抓包 00:11:50.408–00:12:00.882 九跳逐帧实证）：逐层 n=1
// 单属性帧（HitHPRatio→HitHP→HPRatio→HP，MP 同形），HP/MP 层
// 随后各补自己的 msg24 飘字——c1=16 治疗绿字、c1=18 回蓝飘字
// （49.4.14.150:7074 实时实证，HP/MP 分帧各飘各的）。
// 坐姿期间的周期帧绝不能改走带 LogicState 的全量 vitalUpdate：
// 官服坐姿周期帧从不携带 LogicState，重推状态帧会让客户端重入
// 坐姿动画。
func (p *playerActor) sitcrossRecoveryFrames(settlement sim.SitcrossSettlement) ([][]byte, error) {
	p.mu.Lock()
	state := p.snapshotLocked()
	name := p.name
	p.mu.Unlock()
	frames := make([][]byte, 0, 6)
	if settlement.HitHPChanged {
		frames = append(frames,
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propHitHPRatio, int32(int64(state.HitHP)*100/int64(state.MaxHP))),
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propHitHP, state.HitHP),
		)
	}
	if settlement.HPApplied > 0 {
		frames = append(frames,
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propHPRatio, int32(int64(state.HP)*100/int64(state.MaxHP))),
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propHP, state.HP),
		)
		frame, err := healFloatFrame(playerObjectID, playerOwnerID, 16, settlement.HPApplied, settlement.HPWasted, sitcrossShortcutID, "SkillNormal", name)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	if settlement.MPApplied > 0 {
		frames = append(frames,
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propMPRatio, int32(int64(state.MP)*100/int64(state.MaxMP))),
			sceneObjectIntProperty(playerObjectID, playerOwnerID, propMP, state.MP),
		)
		frame, err := healFloatFrame(playerObjectID, playerOwnerID, 18, settlement.MPApplied, settlement.MPWasted, sitcrossShortcutID, "SkillNormal", name)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// sitcrossRecoveryPerSecond 向上取整，使无法整除的推导上限仍能在至多
// 十次一秒结算内回满。
func sitcrossRecoveryPerSecond(maximum int32) int32 {
	if maximum <= 0 {
		return 0
	}
	return (maximum + sitcrossFullRecoverySeconds - 1) / sitcrossFullRecoverySeconds
}

// combatBruiseTick 在角色仍处于战斗中时结算次要的 HitHP 层。常规的
// 属性档案更新已包含 HitHP 和 HitHPRatio，因此这里沿用已被验证的常规
// 0x10 属性路由。
func (p *playerActor) combatBruiseTick() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.vitals.RecoverBruisedHPWhileFighting(combatBruiseRecoveryPerSecond)
}

// setRageRecovery 设定 1s ticker 的怒气自动回复速率（GM 会话级设置；
// 0=关闭）。纯服务器侧状态，无客户端帧；后续 tick 自然按新速率结算。
func (p *playerActor) setRageRecovery(perSecond int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if perSecond < 0 {
		perSecond = 0
	}
	p.rageRecoveryPerSecond = perSecond
	return perSecond
}

// rageRecoveryTick 结算每秒怒气自动回复,速率为 GM 会话级设置
// rageRecoveryPerSecond(默认 0=不回复;官方满档 5,55866 实证)。官方模型
// (rage-recovery-frames.md §5):不限战斗态,脱战照样回复;到有效上限
// 即停(基础 80 + 运行内功加成),满值期不再推送任何维持帧。tick 相位
// 由调用方(1s facultyTicker)决定。
func (p *playerActor) rageRecoveryTick() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rageRecoveryPerSecond <= 0 {
		return false
	}
	if p.vitals.Logic == entity.LogicStateDied {
		return false
	}
	before := p.progress.attributes.sp
	p.progress.attributes.sp = p.progress.clampSP(before + p.rageRecoveryPerSecond)
	return p.progress.attributes.sp != before
}

// combatPresenceIdleExit 是脱战迟滞窗口：官方抓包规则「命中入战，
// 10s 无战斗活动才退出」（combat-skill-211.md §9.1）。没有这个窗口，
// 逐 tick 重建的威胁表在击杀清仇恨/栓绳回家/攻击冷却间隙会闪断，
// 战斗中频繁发出 LogicState 0/1 抖动帧。
const combatPresenceIdleExit = 10 * time.Second

// noteCombatActivity 记录一次战斗活动（命中敌人或被敌人命中），
// 是 10s 迟滞窗口的驱动源。调用方无需持 p.mu。
func (p *playerActor) noteCombatActivity(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastCombatAt = now
}

// setCombatPresence 负责本地敌对 AI 的玩家侧生命周期。威胁（NPC
// 持有自己为目标）或 10s 内的战斗活动都保持 Fighting；两者都消失
// 才恢复 Normal。这使恢复、伤害和客户端 LogicState 保持在同一个
// 权威转换之上，且击杀/栓绳间隙不再闪发脱战帧。
func (p *playerActor) setCombatPresence(threatened bool, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.vitals.HP <= 0 {
		return false
	}
	active := threatened || now.Before(p.lastCombatAt.Add(combatPresenceIdleExit))
	if active {
		if p.vitals.Logic == entity.LogicStateFighting {
			return false
		}
		p.vitals.SetLogicState(entity.LogicStateFighting)
		return true
	}
	if p.vitals.Logic != entity.LogicStateFighting {
		return false
	}
	p.vitals.SetLogicState(entity.LogicStateNormal)
	return true
}

func (p *playerActor) currentSkillSnapshot() string {
	id, _, _, _ := p.currentSkillStateSnapshot()
	return id
}

func (p *playerActor) currentSkillStateSnapshot() (string, string, int32, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSkillID, p.currentSkillEffectID, p.currentSkillLevel, p.currentSkillTarget
}

// clearCurrentSkill 只清除调度该定时器的那一次确切技能调用（按
// (id, epoch) 配对——Java clearCurrentSkill(id, castEpoch) 同构：新招
// 顶掉旧招时旧定时器失配空转，不会误删新招的 active_skill_rec 行）。
// 后续的调用绝不能被更早的定时器取消。LogicState 不在此处落 0：战斗
// 存在性唯一权威是 setCombatPresence（威胁/10s 迟滞），收招帧只回显
// 当前值——62433 战斗抓包 LogicState 跨招恒 1 实证。
func (p *playerActor) clearCurrentSkill(expected string, epoch uint64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.currentSkillID != expected || p.currentSkillEpoch != epoch {
		return false
	}
	p.currentSkillID = ""
	p.currentSkillEffectID = ""
	p.currentSkillLevel = 0
	p.currentSkillTarget = 0
	return true
}

// currentSkillEpochSnapshot 返回当前出招代次（收尾定时器闭包捕获用）。
func (p *playerActor) currentSkillEpochSnapshot() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSkillEpoch
}

// noteSkillViewRows 记录 View40 行号映射（登录 merge / 秘籍重建视图时
// 调用；CanUse 行帧按 configID 寻址行号）。
func (p *playerActor) noteSkillViewRows(rows []learnedSkillView) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skillViewRows = make(map[string]uint16, len(rows))
	for index, row := range rows {
		p.skillViewRows[row.configID] = uint16(index + 1)
	}
}

// skillViewRow 返回技能的 View40 行号（0=不在已学容器，不发行帧）。
func (p *playerActor) skillViewRow(configID string) uint16 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skillViewRows[configID]
}

// meleeDamageRange 是伤害公式攻方掷值区间（Java PlayerProgress.
// meleeDamageRange 同口径：基础 + 装备外功 + 内功 Min/MaxMeleeDamage；
// Go 装备/内功外功分量未移植，当前=基础 20-35，接入后在此叠加）。
func (p *playerActor) meleeDamageRange() (int32, int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.attributes.minMeleeDamage, p.progress.attributes.maxMeleeDamage
}

// currentSkillClearUpdate 是收招技能状态清除帧（参考服 exe #2014 六项
// 形态，Java skillStateClearUpdate 同源）：LogicState 回显当前权威值，
// CurSkillID/Target 清空，CurSkillCanNoLimit/CantBe(,)SkillMustLocked
// 归零。
func (p *playerActor) currentSkillClearUpdate() ([]byte, error) {
	p.mu.Lock()
	logic := p.vitals.Logic
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("LogicState").Index, Name: "LogicState", Value: clientdata.ByteValue(logic)},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue("")},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.ObjectValue(0, 0)},
		{Index: propCurSkillCanNoLimit, Name: "CurSkillCanNoLimit", Value: clientdata.ByteValue(0)},
		{Index: propCantBeSkillLocked, Name: "CantBeSkillLocked", Value: clientdata.ByteValue(0)},
		{Index: propCantBeSkillMustLocked, Name: "CantBeSkillMustLocked", Value: clientdata.ByteValue(0)},
		// 官服收尾簇（f 46.175）成对清 CantMove/CantParry——落地簇 msg19 后
		// 置位的两帧若不随收招清除，会悬挂到下一招把人物冻住（63073 前科）。
		{Index: propRushCantMove, Name: "CantMove", Value: clientdata.ByteValue(0)},
		{Index: propCantParry, Name: "CantParry", Value: clientdata.Int32Value(0)},
	})
}

// facultyNameResetFrame 只清除客户端侧的 FacultyName 属性。打开
// form_wuxue_faculty 发生在 1006/1 选择响应之后，因此它的 FacultyName
// 绑定器可能错过最初的值变更。1006/24 刷新会在常规的属性档案更新之前
// 立即发送这个清空帧；随后恢复真实名称即可可靠地执行
// set_control_state，既不改变服务器持有的选择，也不假装演武已经开始。
func (p *playerActor) facultyNameResetFrame() ([]byte, bool, error) {
	p.mu.Lock()
	hasFacultyName := p.progress.facultyName != ""
	p.mu.Unlock()
	if !hasFacultyName {
		return nil, false, nil
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propFacultyName, Name: "FacultyName", Value: clientdata.StringValue("")},
	})
	return frame, true, err
}

// setSP 修改服务器权威的怒气值，并立即返回主角 UI 消费的常规 0x10
// 属性更新。饱和到 [0, 有效上限]。
func (p *playerActor) setSP(value int32) (int32, []byte, error) {
	p.mu.Lock()
	p.progress.attributes.sp = p.progress.clampSP(value)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

// addSP 是战斗/技能怒气奖励以及 GM 调整未来唯一的入口。在这里做饱和
// 处理可保证 211 的消耗检查读到的正是客户端刚显示过的值。
// addSP 是战斗/技能怒气奖励以及 GM 调整未来唯一的入口。饱和
// [0, 有效上限](基础 80 + 运行内功加成);值未变化(已在边界)时不
// 重推帧——官方满值停推,纯 SP 事件走 18 字节单帧
// `10 00 01 00 00 11 0B 5C 7A 00 01 00 D3 01 <i32>`。
func (p *playerActor) addSP(delta int32) (int32, []byte, error) {
	p.mu.Lock()
	before := p.progress.attributes.sp
	next := int64(before) + int64(delta)
	if next < 0 {
		next = 0
	}
	if cap := int64(p.progress.rageCap()); next > cap {
		next = cap
	}
	p.progress.attributes.sp = int32(next)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	if sp == before {
		return sp, nil, nil
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

func (p *playerActor) addSilver(amount int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if amount > 0 && p.silver > math.MaxInt32-amount {
		p.silver = math.MaxInt32
	} else {
		p.silver += amount
	}
	return p.silver
}

func (p *playerActor) silverUpdate() ([]byte, error) {
	p.mu.Lock()
	silver := p.silver
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("CapitalType1").Index, Name: "CapitalType1", Value: clientdata.Int64Value(int64(silver))},
	})
}

func (p *playerActor) addObject(location role.Position, visual roleVisual) []byte {
	// 本机视角的玩家对象恒为 1-1：stage_main 开图与 0x11 记录绑定
	// 都依赖这一身份。
	return p.addObjectAs(playerObjectID, playerOwnerID, location, visual)
}

// addObjectAs 以给定的线上对象身份生成 0x0D 玩家档案。远程玩家
// 的物化走本入口：帧头换成其在场景内的 wire objectID，帧体仍是
// 该玩家（p 自身）的实时状态快照。
func (p *playerActor) addObjectAs(objectID, ownerID uint32, location role.Position, visual roleVisual) []byte {
	p.mu.Lock()
	state := p.snapshotLocked()
	state.ObjectID, state.OwnerID = objectID, ownerID
	motion := p.motion
	now := time.Now()
	msg := make([]byte, 0x3f, 0x120)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], state.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], state.OwnerID)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(location.X))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(location.Y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(location.Z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(location.Orient))

	count := uint16(0)
	addByte := func(index uint16, value byte) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = append(msg, value)
		count++
	}
	addInt := func(index uint16, value int32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(value))
		count++
	}
	addFloat := func(index uint16, value float32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
		count++
	}
	addWord := func(index uint16, value uint16) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint16(msg, value)
		count++
	}
	addInt64 := func(index uint16, value int64) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint64(msg, uint64(value))
		count++
	}
	addString := func(index uint16, value string) { msg = appendStringProperty(msg, index, value); count++ }
	addObject := func(index uint16, value uint64) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint64(msg, value)
		count++
	}

	addByte(clientdata.MustPlayerProperty("Type").Index, 2)               // Type = TYPE_PLAYER（fork=181，Byte）
	addByte(clientdata.MustPlayerProperty("Sex").Index, visual.sex)       // fork=182
	addString(clientdata.MustPlayerProperty("Photo").Index, visual.photo) // fork=184
	addByte(clientdata.MustPlayerProperty("Camp").Index, 0)               // Camp（fork=174，官方线型 Byte）
	// Race(fork-only@4)：官服表缺席，按名门控——官服方言跳过。
	if race, ok := clientdata.SceneProperty("Race"); ok {
		addInt(race.Index, 0)
	}
	addString(clientdata.MustPlayerProperty("Job").Index, "") // Job（fork=1588，官方线型 String）
	addByte(clientdata.MustPlayerProperty("Level").Index, 1)  // Level（fork=1465，官方线型 Byte）
	// 键位域（Java PlayerActor.addObject 同位）：引擎建对象时即读，
	// post-ready 0x10 补推太晚（2026-11-04「模式没推」根因）。档内
	// 198 无尾空格（0x10 单帧形态带 " "，勿混用）；无档零属性。
	if p.keybind.bind != "" {
		addString(keybindPropertyIndex, p.keybind.bind)
	}
	if p.keybind.operateModeSet {
		addInt(operateModePropertyIndex, p.keybind.operateMode)
	}
	// NeiGongBind(201) 同窗口内嵌：内功栏主窗体打开即 QueryProp 快照
	//（get_svr_ng_bind 按拆段数校验，读不到就整串回落全零），补推晚于
	// 主窗体打开会留一次「绑定丢失」表象。档内同样无尾空格；无档零属性。
	if p.keybind.neigongBind != "" {
		addString(neiGongBindPropertyIndex, p.keybind.neigongBind)
	}
	// PosiX/Y/Z/Orient(fork-only 9-12)：官服 2433 表无此名（坐标走 0x0D
	// 固定头），官服方言下按名缺席即整组跳过。
	if posiX, ok := clientdata.SceneProperty("PosiX"); ok {
		addFloat(posiX.Index, location.X)
	}
	if posiY, ok := clientdata.SceneProperty("PosiY"); ok {
		addFloat(posiY.Index, location.Y)
	}
	if posiZ, ok := clientdata.SceneProperty("PosiZ"); ok {
		addFloat(posiZ.Index, location.Z)
	}
	if orient, ok := clientdata.SceneProperty("Orient"); ok {
		addFloat(orient.Index, location.Orient)
	}
	addString(clientdata.MustPlayerProperty("Hair").Index, visual.hair)           // fork=188
	addString(clientdata.MustPlayerProperty("Face").Index, visual.face)           // fork=185
	addString(clientdata.MustPlayerProperty("Cloth").Index, visual.cloth)         // fork=393
	addString(clientdata.MustPlayerProperty("Pants").Index, visual.pants)         // fork=394
	addString(clientdata.MustPlayerProperty("Shoes").Index, visual.shoes)         // fork=395
	addString(clientdata.MustPlayerProperty("ActionSet").Index, visual.actionSet) // fork=406
	addString(clientdata.MustPlayerProperty("State").Index, "stand")              // fork=183
	// FloatingState(fork-only@20)：官服表缺席，按名门控——官服方言跳过。
	if floating, ok := clientdata.SceneProperty("FloatingState"); ok {
		addInt(floating.Index, 0)
	}
	addByte(clientdata.MustPlayerProperty("LogicState").Index, state.LogicState) // fork=463
	addFloat(clientdata.MustPlayerProperty("JumpSpeed").Index, 8.0)              // JumpSpeed（fork=681）
	addFloat(clientdata.MustPlayerProperty("MoveSpeed").Index, motion.moveSpeed) // fork=384
	addFloat(clientdata.MustPlayerProperty("WalkSpeed").Index, 7.5)              // WalkSpeed（fork=385）
	addFloat(clientdata.MustPlayerProperty("RunSpeed").Index, motion.runSpeed)   // fork=386
	addInt(clientdata.MustPlayerProperty("SpeedRatio").Index, state.SpeedRatio)  // fork=390
	// CantJump(fork-only@27)：官服表缺席，按名门控——官服方言跳过。
	if cantJump, ok := clientdata.SceneProperty("CantJump"); ok {
		addByte(cantJump.Index, 0)
	}
	addInt(propHPRatio, int32(int64(state.HP)*100/int64(state.MaxHP)))
	addInt(propMPRatio, int32(int64(state.MP)*100/int64(state.MaxMP)))
	addInt(clientdata.MustPlayerProperty("MaxHP").Index, state.BaseMaxHP) // fork=578
	addInt(clientdata.MustPlayerProperty("MaxMP").Index, state.BaseMaxMP) // fork=579
	addInt(propHP, state.HP)
	addInt(propMP, state.MP)
	addInt(propHitHP, state.HitHP)
	addInt(propHitHPRatio, int32(int64(state.HitHP)*100/int64(state.MaxHP)))
	addInt(clientdata.MustPlayerProperty("QingGongPoint").Index, state.QingGongPoint)             // fork=692
	addInt(clientdata.MustPlayerProperty("MaxQingGongPoint").Index, state.MaxQingGongPoint)       // fork=693
	addInt(clientdata.MustPlayerProperty("MaxQingGongPointAdd").Index, state.MaxQingGongPointAdd) // fork=694
	// ParryRatio 建对象即带：官方 NPC 0x0D 同款（三方抓包 NPC 帧均含
	// 478=100）。引擎建对象时即读格挡族，post-ready 0x10 补推可能太晚
	// （2026-11-04 键位域同教训）；玩家侧值恒 100。
	addInt(propParryRatio, 100)
	addWord(propNeigongPKStatus, 0)            // 官方线型 Word
	addByte(propDead, boolByte(state.HP <= 0)) // 官方线型 Byte
	// CantUseSkill:官方 2370 项表无此名,不再下发。
	addString(propCurSkillID, p.currentSkillID)
	// CurSkillEffectID/CurSkillLevel:官方表无此名,不再下发。
	addObject(propCurSkillTarget, p.currentSkillTarget)
	addInt(propModifySkillLockTime, 200) // 官方口径 200（Java #2014 exe；旧 5000 为本地占位）
	// BufferListStr(fork-only@143)：官服方言停发（§14.5）。
	if bufferList, ok := clientdata.PlayerProperty("BufferListStr"); ok {
		addString(bufferList.Index, p.bufferListStringLocked(now))
	}
	for _, property := range baseMotionProperties(motion) {
		addFloat(property.Index, property.Value.F32)
	}
	for _, property := range qinggongMotionProperties() {
		addFloat(property.Index, property.Value.F32)
	}
	addInt64(417, int64(p.silver)) // CapitalType1(玩家域,官方线型 Int64):碎银
	for _, slot := range nativeBuffSlots {
		addString(bufferInfoPropertyIndex(slot), p.bufferInfoLocked(slot, now))
	}
	for _, property := range p.progress.properties() {
		msg = appendNPCProperty(msg, property)
		count++
	}
	// 经脉 8 属性（Java addObject 同位）：0x0D 建对象即带，客户端面板
	// 首开即有值；post-ready 0x10 补推太晚（键位域同教训）。
	for _, property := range p.jingMaiPropsLocked() {
		msg = appendNPCProperty(msg, property)
		count++
	}
	addString(clientdata.MustPlayerProperty("Hat").Index, visual.hair)                           // Hat（fork=391）                                                                  // Hat
	msg = appendWideStringProperty(msg, clientdata.MustPlayerProperty("Name").Index, state.Name) // fork=178（双表同位）
	count++
	msg = appendObjectProperty(msg, uint16(clientdata.MustPlayerProperty("LastObject").Index), 0, 0) // LastObject（fork=414）
	count++
	// 官服位物理档案（2026-11-10）：官服进场给玩家自己发 0x28 详情帧
	//（3187B，obj=玩家）——其中 CollideRadius 是客户端寻路引擎的玩家
	// 碰撞体输入。0x28 通道未通前，物理属性经已验证的 0x0D 通道补发
	//（与 NPC 的 officialNPCProperties 同集）。
	addFloat(clientdata.MustPlayerProperty("CollideRadius").Index, 0.9) // fork=179
	addString(clientdata.MustPlayerProperty("State").Index, "stand")    // fork=183
	addFloat(clientdata.MustPlayerProperty("ActionSpeed").Index, 1.0)   // fork=214
	addInt(clientdata.MustPlayerProperty("HPRatio").Index, 100)         // fork=469
	addInt(clientdata.MustPlayerProperty("ParryRatio").Index, 100)      // fork=478
	addInt(clientdata.MustPlayerProperty("HitHPRatio").Index, 100)      // fork=482
	binary.LittleEndian.PutUint16(msg[0x3d:], count)
	p.mu.Unlock()
	return msg
}

func sendPlayerSpawn(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	// 在新建的玩家对象中保留仍处于活动状态的原生 BuffStore 状态。
	// BufferInfo 由客户端的 C++ 属性绑定器处理。
	log.Printf("native Buff spawn player=%d-%d red=%q drift=%q jump=%q neigong=%q", playerObjectID, playerOwnerID, player.bufferInfo(redSuperArmorBuffSlot, time.Now()), player.bufferInfo(waterDriftBuffSlot, time.Now()), player.bufferInfo(waterJumpBuffSlot, time.Now()), player.bufferInfo(innerPowerBuffSlot, time.Now()))
	if err := sendPlayerAddObject(conn, player, location, visual); err != nil {
		return err
	}
	return sendPlayerLocationAndVitals(conn, player, location)
}

// sendPlayerAddObject 创建原生玩家对象，但不发出单独的 ServerLocation
// 移动更新。现场场景替换需要这个拆分：stage_main 要求先收到 1-1 才
// 会打开目标场景，而 0x1F 视觉重定位必须等到目标 ClientReady 之后。
func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	return conn.WriteFrame(player.addObject(location, visual))
}

// sendPlayerLocationAndVitals 发 0x1F 重定位 + 常规 vitals 大批 0x10。
// 官方方言下没有独立的登录怒气帧:SP(467) 随 stateProperties 大批下发,
// MaxSP(580,折算最终值) 只由 loginVitalUpdate 在这批全量档携带一次
// (官服抓包全程零下发 583,加成折进 580;新服 exe 反汇编零引用
// 580/583,客户端疑似按本地口径呈现怒气球)。
func sendPlayerLocationAndVitals(conn sceneMessageConnection, player *playerActor, location role.Position) error {
	if err := conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(location))); err != nil {
		return err
	}
	vitals, err := player.loginVitalUpdate()
	if err != nil {
		return err
	}
	return conn.WriteFrame(vitals)
}

func worldTransform(position role.Position) worldcore.Transform {
	return worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
}
