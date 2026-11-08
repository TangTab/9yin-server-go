package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/local/9yin-go-server/internal/gmapi"
)

// 九阴志全局门控：全服唯一的单向阶段刻度（1..9，1=1内 … 9=6内，
// 对应客户端 power_guide.ini 的 step 主步骤 1..6）。阶段只增不减；
// 击杀/任务事件点累计到阈值自动推进（可连跳）。
//
// 蓝本 = Java 参考服已被删除的 NinJiuZhi.java（G:\pojie\ai\9yin-server
// git 3d1df4a「新增九阴志全服阶段门控」，docs/ninjiuzhi-gating.md）。
// Go 差异（其余语义逐条照录）：
//   - 初始缺省阶段 6 而非 Java 的 1：保住登录 419/414 gamestep 帧与
//     官服 c.pcapng #111 gamestep=6 逐字节一致的性质（2026-11-08
//     用户拍板）；
//   - 五类门控覆写（features/tasks/npcs/scenes/inner_books）只装载
//     并提供 isOpen 查询，尚未接线到场景/任务/NPC 过滤（Java 的五处
//     接入点回归面大，留待后续批次）；
//   - 阶段变化不跨 goroutine 写连接：发起会话内联下发，其他会话由
//     1s ticker 的 stageRevision 漂移检查补推（单写者纪律）；
//   - 状态 JSON 落盘在 <server root>/data/ninjiuzhi.json（Java 为
//     CWD 相对 data/；Go 随 NINEYIN_SERVER_ROOT 锚定，沙箱天然隔离），
//     字段与 Java 及 data-import 同构。
const (
	// jyzGateDefaultStage 初始缺省阶段（官服 #111 gamestep=6）。
	jyzGateDefaultStage = 6
)

// jyzStageNames 阶段展示名（GM 面板口径：1=1内 … 9=6内）。下标 0 占位。
var jyzStageNames = []string{"", "1内", "2内", "3内", "4内", "5内", "5内后期", "6内前期", "6内中期", "6内"}

// jyzDefaultThresholds 默认升阶累计事件点：升到 2..9 阶分别需要累计
// 10/30/60/100/150/210/280/360 点（本地验证节奏，requirements
// thresholds 可覆写）。下标 0/1 不参与（阶段 1 无门槛）。
var jyzDefaultThresholds = [gmapi.JyzStageMax + 1]int64{0, 0, 10, 30, 60, 100, 150, 210, 280, 360}

// jyzGateCategory 门控覆写类别（对应 game_ninjiuzhi_overrides.category）。
type jyzGateCategory string

// jyzGateCategories 五类覆写的固定枚举序（status/日志口径稳定）。
var jyzGateCategories = []jyzGateCategory{
	"features", "tasks", "npcs", "scenes", "inner_books",
}

// jyzGateState 九阴志全服门控。所有公开方法自持 mu；游戏侧唯一实例
// jyzGate（main() 经 loadJyzGate 重装为落盘形态）。
type jyzGateState struct {
	mu      sync.Mutex
	stage   int
	initial int
	active  int
	points  int64
	events  map[string]int64
	// revision 每次变更 +1（落盘口径，Java 同名字段）；
	// stageRevision 仅在阶段值变化时 +1（会话 ticker 漂移检查用）。
	revision       int64
	stageRev       int64
	thresholds     [gmapi.JyzStageMax + 1]int64
	requirements   map[jyzGateCategory]map[string]int
	thresholdsUsed bool
	// eventState data_record 58 行的事件状态值（进程级 env，运行期
	// 不变；与阶段正交，仅随 config() 拼进下发帧）。
	eventState uint16
	// stateFile 空 = 不落盘（测试/未接线）；loadFailed 后拒绝覆写。
	stateFile  string
	loadFailed bool
}

// newJyzGate 构造无落盘实例（测试与 main() 装载前的包级缺省）。
func newJyzGate(initial int, eventState uint16) *jyzGateState {
	gate := &jyzGateState{
		initial:      clampJyzStage(initial),
		events:       make(map[string]int64),
		thresholds:   jyzDefaultThresholds,
		requirements: make(map[jyzGateCategory]map[string]int),
		eventState:   eventState,
	}
	gate.stage = gate.initial
	return gate
}

// clampJyzStage 把任意输入钳进 1..9（env 步骤值 999 之类只在装载时
// 出现一次，装载体面收敛）。
func clampJyzStage(stage int) int {
	if stage < int(gmapi.JyzStageMin) {
		return int(gmapi.JyzStageMin)
	}
	if stage > int(gmapi.JyzStageMax) {
		return int(gmapi.JyzStageMax)
	}
	return stage
}

// loadJyzGate 从 dataDir 装载：门控覆写 ninjiuzhi_requirements.json
// （容缺）+ 运行时状态 ninjiuzhi.json（容缺，存在则覆盖初始阶段）。
// initial.step>0 视为 env NINEYIN_JYZ_STEP 的初始阶段覆写（优先于
// requirements 的 initial_stage；>9 钳 9），eventState 为 data_record
// 行值。任一文件损坏只保持缺省并留日志，不阻断起服。
func loadJyzGate(dataDir string, initial jyzConfig) *jyzGateState {
	stage := jyzGateDefaultStage
	req := readJyzGateRequirements(filepath.Join(dataDir, "ninjiuzhi_requirements.json"))
	if req != nil && req.initialStage > 0 {
		stage = clampJyzStage(req.initialStage)
	}
	if initial.step > 0 {
		stage = clampJyzStage(int(initial.step))
	}
	gate := newJyzGate(stage, initial.eventState)
	if req != nil {
		gate.requirements = req.categories
		gate.thresholds = req.thresholds
		gate.thresholdsUsed = req.thresholdsUsed
	}
	gate.stateFile = filepath.Join(dataDir, "ninjiuzhi.json")
	gate.loadState()
	log.Printf("jyz gate loaded: stage=%d (%s) points=%d overrides=%d thresholds=custom:%v file=%s",
		gate.stage, jyzStageName(gate.stage), gate.points, gate.overrideCountLocked(), gate.thresholdsUsed, gate.stateFile)
	return gate
}

// jyzGate 全服九阴志门控缺省实例（阶段 6、无落盘）。main() 启动时经
// loadJyzGate 重装为落盘形态；测试进程保持缺省即可（combat 路径的
// onKill 只动点数，不落盘不广播）。
var jyzGate = newJyzGate(jyzGateDefaultStage, defaultJyzConfig().eventState)

// ---------- 查询 ----------

func (g *jyzGateState) currentStage() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stage
}

// stageRevision 阶段值变化计数（ticker 漂移检查的比较基线）。
func (g *jyzGateState) stageRevision() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stageRev
}

// config 帧下发用配置：step=当前全服阶段，eventState=进程级 env 值。
func (g *jyzGateState) config() jyzConfig {
	g.mu.Lock()
	defer g.mu.Unlock()
	return jyzConfig{step: int32(g.stage), eventState: g.eventState}
}

// eventStateValue data_record 事件行值（进程级，运行期不变）。
func (g *jyzGateState) eventStateValue() uint16 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.eventState
}

// jyzStageName 阶段展示名（1内/2内/…）。
func jyzStageName(stage int) string {
	if stage >= int(gmapi.JyzStageMin) && stage <= int(gmapi.JyzStageMax) {
		return jyzStageNames[stage]
	}
	return "?"
}

// isOpen 门控查询：未配置覆写的键一律开放（Java 同语义；当前尚无
// 接线方，供后续批次挂场景/任务/NPC 过滤）。
func (g *jyzGateState) isOpen(category jyzGateCategory, key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isOpenLocked(category, key)
}

func (g *jyzGateState) isOpenLocked(category jyzGateCategory, key string) bool {
	required, exists := g.requirements[category][key]
	return !exists || g.stage >= required
}

func (g *jyzGateState) overrideCountLocked() int {
	count := 0
	for _, overrides := range g.requirements {
		count += len(overrides)
	}
	return count
}

// ---------- 变更（GM / 事件点） ----------

// setStage 单调设阶。返回 (拒绝原因, 旧阶段, 新阶段)；原因非空 = 未
// 生效（界值外或目标 ≤ 当前）。
func (g *jyzGateState) setStage(target int) (string, int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if target < int(gmapi.JyzStageMin) || target > int(gmapi.JyzStageMax) {
		return fmt.Sprintf("stage 应为 %d~%d", gmapi.JyzStageMin, gmapi.JyzStageMax), g.stage, g.stage
	}
	if target <= g.stage {
		return fmt.Sprintf("阶段只增不减：当前 %d（%s），目标 %d 被忽略", g.stage, jyzStageNames[g.stage], target), g.stage, g.stage
	}
	old := g.stage
	g.stage = target
	g.active = 1
	g.stageRev++
	g.afterChangeLocked("set")
	return "", old, g.stage
}

// addEvent 事件点 +1（kill/task）：累计到阈值自动升阶（可连跳多阶）。
func (g *jyzGateState) addEvent(kind string) (int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.events[kind]++
	g.points++
	g.active = 1
	old := g.stage
	for g.stage < int(gmapi.JyzStageMax) && g.points >= g.thresholds[g.stage+1] {
		g.stage++
	}
	if g.stage != old {
		g.stageRev++
	}
	g.afterChangeLocked("event:" + kind)
	return old, g.stage
}

// onKill 击杀事件（战斗管线 NPC 死亡；Java SceneFlow 同款事件源）。
func (g *jyzGateState) onKill() {
	g.addEvent("kill")
}

// reset 回 initial 阶段、点数/事件清零（GM 面板专用）。
func (g *jyzGateState) reset() (int, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	old := g.stage
	g.stage = g.initial
	g.active = 0
	g.points = 0
	g.events = make(map[string]int64)
	if g.stage != old {
		g.stageRev++
	}
	g.afterChangeLocked("reset")
	return old, g.stage
}

func (g *jyzGateState) afterChangeLocked(cause string) {
	g.revision++
	g.saveLocked(cause)
}

// ---------- 状态文本（GM worldInfo 回显口径，Java status 逐句对齐） ----------

func (g *jyzGateState) status() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "九阴志：阶段 %d/%d（%s）", g.stage, gmapi.JyzStageMax, jyzStageNames[g.stage])
	if g.stage < int(gmapi.JyzStageMax) {
		remaining := g.thresholds[g.stage+1] - g.points
		if remaining < 0 {
			remaining = 0
		}
		fmt.Fprintf(&b, "，距下一阶还差 %d 事件点", remaining)
	}
	fmt.Fprintf(&b, "；累计 %d 点", g.points)
	if len(g.events) > 0 {
		kinds := make([]string, 0, len(g.events))
		for kind := range g.events {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		b.WriteString("（")
		for i, kind := range kinds {
			if i > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%s:%d", kind, g.events[kind])
		}
		b.WriteString("）")
	}
	fmt.Fprintf(&b, "；门控覆写 %d 条", g.overrideCountLocked())
	return b.String()
}

// jyzStageChangeNotice 阶段生效的世界提示（Java broadcastNinJiuZhiStage
// 同文案）。
func jyzStageChangeNotice(stage int) string {
	return fmt.Sprintf("九阴志推进：阶段 %d（%s），江湖暗流涌动", stage, jyzStageName(stage))
}

// ---------- 落盘与装载 ----------

// jyzGatePersist 与 Java data/ninjiuzhi.json 及 data-import 的
// server_ninjiuzhi 表同构（revision 随每次变更递增）。
type jyzGatePersist struct {
	Stage    int64            `json:"stage"`
	Active   int64            `json:"active"`
	Points   int64            `json:"points"`
	Events   map[string]int64 `json:"events"`
	Revision int64            `json:"revision"`
}

func (g *jyzGateState) loadState() {
	if g.stateFile == "" {
		return
	}
	raw, err := os.ReadFile(g.stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("jyz gate state read failed: %s: %v", g.stateFile, err)
		}
		return
	}
	var persist jyzGatePersist
	if err := json.Unmarshal(stripJSONBOM(raw), &persist); err != nil {
		// 损坏档拒绝覆写（loadFailed 闸），下次 save 只报错不动文件。
		g.loadFailed = true
		log.Printf("jyz gate state load failed, refusing to overwrite: %s: %v", g.stateFile, err)
		return
	}
	g.stage = clampJyzStage(int(persist.Stage))
	if persist.Events != nil {
		g.events = persist.Events
	}
	g.active = int(persist.Active)
	g.points = persist.Points
	g.revision = persist.Revision
}

func (g *jyzGateState) saveLocked(cause string) {
	if g.stateFile == "" {
		return
	}
	if g.loadFailed {
		log.Printf("jyz gate save skipped — load previously failed, refusing to overwrite %s (%s)", g.stateFile, cause)
		return
	}
	persist := jyzGatePersist{
		Stage:    int64(g.stage),
		Active:   int64(g.active),
		Points:   g.points,
		Events:   g.events,
		Revision: g.revision,
	}
	encoded, err := json.MarshalIndent(persist, "", "\t")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(g.stateFile), 0755)
		if err == nil {
			err = os.WriteFile(g.stateFile, append(encoded, '\n'), 0644)
		}
	}
	if err != nil {
		log.Printf("jyz gate save failed (%s): %s: %v", cause, g.stateFile, err)
	}
}

// stripJSONBOM 去掉 UTF-8 BOM（Java fastjson 版明确容忍手写档 BOM）。
func stripJSONBOM(raw []byte) []byte {
	if len(raw) > 0 && raw[0] == 0xEF && len(raw) > 2 && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	return raw
}

// ---------- requirements 覆写 ----------

// jyzGateRequirements ninjiuzhi_requirements.json 的装载结果。
type jyzGateRequirements struct {
	categories     map[jyzGateCategory]map[string]int
	thresholds     [gmapi.JyzStageMax + 1]int64
	thresholdsUsed bool
	initialStage   int
}

type jyzGateRequirementsFile struct {
	Features     map[string]any `json:"features"`
	Tasks        map[string]any `json:"tasks"`
	NPCs         map[string]any `json:"npcs"`
	Scenes       map[string]any `json:"scenes"`
	InnerBooks   map[string]any `json:"inner_books"`
	InitialStage any            `json:"initial_stage"`
	// Thresholds 兼容两种形态：对象 {"2":30} 或数组 [0,0,30,…]。
	Thresholds json.RawMessage `json:"thresholds"`
}

// readJyzGateRequirements 读门控覆写文件（容缺返回 nil；非法键值忽略
// 并留日志，与 Java loadRequirements 的宽容口径一致）。
func readJyzGateRequirements(path string) *jyzGateRequirements {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("jyz gate requirements read failed: %s: %v", path, err)
		}
		return nil
	}
	var file jyzGateRequirementsFile
	if err := json.Unmarshal(stripJSONBOM(raw), &file); err != nil {
		log.Printf("jyz gate requirements parse failed: %s: %v", path, err)
		return nil
	}
	req := &jyzGateRequirements{
		categories: make(map[jyzGateCategory]map[string]int),
		thresholds: jyzDefaultThresholds,
	}
	perCategory := map[jyzGateCategory]map[string]any{
		"features":    file.Features,
		"tasks":       file.Tasks,
		"npcs":        file.NPCs,
		"scenes":      file.Scenes,
		"inner_books": file.InnerBooks,
	}
	for _, category := range jyzGateCategories {
		source := perCategory[category]
		if len(source) == 0 {
			continue
		}
		overrides := make(map[string]int, len(source))
		for key, value := range source {
			parsed, ok := jyzCoerceInt(value)
			if !ok {
				log.Printf("jyz gate requirement ignored: %s %s = %v", category, key, value)
				continue
			}
			overrides[key] = parsed
		}
		req.categories[category] = overrides
	}
	if parsed, ok := jyzCoerceInt(file.InitialStage); ok {
		req.initialStage = parsed
	}
	req.thresholdsUsed = parseJyzGateThresholds(file.Thresholds, &req.thresholds)
	return req
}

// parseJyzGateThresholds 解析 thresholds 覆写（对象按 2..9 键、数组按
// 下标 0..9），返回是否发生覆写。
func parseJyzGateThresholds(raw json.RawMessage, thresholds *[gmapi.JyzStageMax + 1]int64) bool {
	if len(raw) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" {
		return false
	}
	var asObject map[string]any
	if err := json.Unmarshal(raw, &asObject); err == nil {
		used := false
		for key, value := range asObject {
			index, err := strconv.Atoi(key)
			if err != nil || index < int(gmapi.JyzStageMin)+1 || index > int(gmapi.JyzStageMax) {
				continue
			}
			parsed, ok := jyzCoerceInt(value)
			if !ok {
				continue
			}
			thresholds[index] = int64(parsed)
			used = true
		}
		return used
	}
	var asArray []any
	if err := json.Unmarshal(raw, &asArray); err == nil {
		used := false
		for index, value := range asArray {
			if index > len(thresholds)-1 {
				break
			}
			parsed, ok := jyzCoerceInt(value)
			if !ok {
				continue
			}
			thresholds[index] = int64(parsed)
			used = true
		}
		return used
	}
	log.Printf("jyz gate thresholds ignored: unsupported shape")
	return false
}

// jyzCoerceInt JSON 数值/字符串双形态整数（Java fastjson getInteger 同宽容）。
func jyzCoerceInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed == math.Trunc(typed) && typed >= math.MinInt32 && typed <= math.MaxInt32 {
			return int(typed), true
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
