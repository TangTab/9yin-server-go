// Package gmapi 是 GM 面板的传输契约层：HTTP/JSON 请求体、校验后的
// 命令纯数据、会话快照，以及校验大 switch 的单一权威实现。
//
// 它同时被两端消费——游戏服（cmd/protocol-probe 的内嵌面板与 internal
// API）与独立面板进程（cmd/gm-panel 的预校验）——因此刻意不依赖
// channel、DB 或任何运行时句柄；命令的执行仍只属于游戏会话循环。
package gmapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// GM 线协议的数值边界。原属 cmd/protocol-probe（player_actor.go /
// neigong_rage_cap.go / gm_stats.go），因 HTTP 校验需要同样的界值而
// 上移到契约层；游戏服侧保留同名别名，行为不变。
const (
	// MaxPlayerSP 与 Java 参考服的怒气基础上限一致。
	MaxPlayerSP int32 = 80
	// MaxNeiGongRageCapAdd 是任何内功能提供的最大怒气上限加成
	//（五内/心法档位封顶）。
	MaxNeiGongRageCapAdd int32 = 50
	// BaseAttributeCap 与 Java PlayerProgress.BASE_ATTRIBUTE_CAP 一致。
	BaseAttributeCap int32 = 1_000_000
)

// CommandBody 是 POST 命令端点的 JSON 请求体。字段名与标签从
// cmd/protocol-probe/gm_panel.go 的匿名结构原样搬来，线格式不变。
type CommandBody struct {
	Action      string  `json:"action"`
	Silver      int32   `json:"silver"`
	SP          int32   `json:"sp"`
	Value       float32 `json:"value"`
	Destination string  `json:"destination"`
	Preset      string  `json:"preset"`
	Position    string  `json:"position"`
	Faction     string  `json:"faction"`
	Code        string  `json:"code"`
	Type        int32   `json:"type"`
	ItemConfig  string  `json:"itemConfig"`
	Amount      int32   `json:"amount"`
	WearNow     bool    `json:"wearNow"`
	CardId      int32   `json:"cardId"`
	Gold        int32   `json:"gold"`
	GuanYin     int32   `json:"guanYin"`
	PkMode      int32   `json:"pkMode"`
	Justice     int32   `json:"justice"`
	Evil        int32   `json:"evil"`
	FreeBase    int32   `json:"freeBase"`
	Ene         int32   `json:"ene"`
	MaxEne      int32   `json:"maxEne"`
	ResumeEne   int32   `json:"resumeEne"`
	Sat         int32   `json:"sat"`
	MaxSat      int32   `json:"maxSat"`
	PkValue     int32   `json:"pkValue"`
	DrunkLevel  int32   `json:"drunkLevel"`
	AttrStr     int32   `json:"attrStr"`
	AttrSta     int32   `json:"attrSta"`
	AttrDex     int32   `json:"attrDex"`
	AttrIng     int32   `json:"attrIng"`
	AttrSpi     int32   `json:"attrSpi"`
	Base        int32   `json:"base"`
	Multiplier  float32 `json:"multiplier"`
	Points      int32   `json:"points"`
	MsgId       int32   `json:"msgId"`
	SubType     int32   `json:"subType"`
	Weather     string  `json:"weather"`
	FadeSec     int32   `json:"fadeSec"`
}

// Command 是校验通过后的命令纯数据。原 cmd/protocol-probe 的 gmCommand
// 去掉 reply channel 后的形态；游戏服侧以嵌入方式包装回 gmCommand，
// 执行器对字段的访问经字段提升自然生效。
//
// Destination 用 role.Location 而不是游戏服的 sceneDestination：
// 后者是场景运行时的私有类型，契约层只承诺"已通过解析的目的地"。
type Command struct {
	Action        string
	Silver        int32
	SP            int32
	Value         float32
	Destination   role.Location
	Position      role.Position
	Faction       string
	Code          string // lua_eval 的 Lua 源码
	ItemConfig    string // grant_item/add_item 的物品配置 ID
	Amount        int32  // grant_item 数量
	WearNow       bool   // grant_item 穿戴形态（否则入背包）
	CardId        int32  // equip_card/equip_hair 皮肤号
	ShowEquipType int32  // set_showequip_type 0/1/2
	Gold          int32  // set_gold（会话级 CapitalType0）
	GuanYin       int32  // set_guanyin（会话级 CapitalType2）
	PkMode        int32  // set_pk_mode 0..8
	Justice       int32  // set_alignment 三值（0=跳过，前端 +v||0 折叠语义）
	Evil          int32
	FreeBase      int32
	Ene           int32 // set_vitals 生存属性层（0=跳过；Add 字段官方表无，忽略）
	MaxEne        int32
	ResumeEne     int32
	Sat           int32
	MaxSat        int32
	PkValue       int32
	DrunkLevel    int32
	AttrStr       int32 // set_five_attributes 基础五维（0=跳过该维）
	AttrSta       int32
	AttrDex       int32
	AttrIng       int32
	AttrSpi       int32
	FacultyBase   int32   // set_fill_speed 基数
	FillMultiplier float32 // set_fill_speed 倍率
	Points        int32   // add_livepoint 历练点数
	MsgId         int32   // custom_probe 数值寻址消息号
	SubType       int32   // custom_probe 探针参数
	Weather       string  // set_weather 闭名单键
	FadeSec       int32   // set_weather 过渡秒
}

// PlayerState 是 GET 会话列表端点的行结构。原 gmPlayerState 原样搬来
//（JSON 标签即面板前端契约）。
type PlayerState struct {
	SessionID    uint64        `json:"sessionId"`
	Remote       string        `json:"remote"`
	Name         string        `json:"name"`
	Scene        role.Scene    `json:"scene"`
	Position     role.Position `json:"position"`
	Silver       int32         `json:"silver"`
	GuanYin      int32         `json:"guanYin"`
	Gold         int32         `json:"gold"`
	SP           int32         `json:"sp"`
	RageRecovery int32         `json:"rageRecovery"`
	Faculty      int32         `json:"faculty"`
	Faction      string        `json:"faction"`
	LastAction   string        `json:"lastAction"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

// WriteJSON 以 GM 面板统一的 JSON 形态写响应。
func WriteJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

// WriteError 写 {"error": message}，两端的错误出口保持同形。
func WriteError(writer http.ResponseWriter, status int, message string) {
	WriteJSON(writer, status, map[string]string{"error": message})
}
