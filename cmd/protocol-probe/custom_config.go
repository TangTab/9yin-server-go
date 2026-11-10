package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// 自定义配置域（玩家客户端设置持久化，Java docs/customizing-shortcut-persist.md
// 定稿协议的 Go 移植）：引擎个性配置串 107/705、键位方案 195 → 属性
// 198/199/200、操作模式 99 → 属性 223。快捷栏 200/201 见 shortcut_records.go。
const (
	// C2S 107：引擎 customizing_manager:SaveConfigToServer 上传整份个性
	// 配置串。客户端本地不落盘：登录即上传默认键位转储，玩家改设置再传。
	clientCustomSaveCustomizing int32 = 107
	// C2S 195：引擎 ShortcutKey:EndEdit(commit, scheme) 键位方案上传，
	// [195, scheme, str]，scheme 0=当前启用 / 1=方案一 / 2=方案二，
	// 串形 "项,键码;…"。
	clientCustomKeybindUpload int32 = 195
	// C2S 99：CLIENT_CUSTOMMSG_SET_OPERATEMODE（form_system_interface_setting
	// .switch_mode），[99, mode]，0=3D（WASD）/ 1=2.5D（点击）。
	clientCustomSetOperateMode int32 = 99
	// C2S 199：CLIENT_CUSTOMMSG_NEIGONGBIND（custom_update_ng_bind(info)，
	// form_main_shortcut 内功栏绑定框 on_btn_bind_click → update_to_srv），
	// [199, "v1;…;v25;"]。25 个值对应内功栏 25 格（grid_shortcut_3 三页
	// 各 5 格 + grid_shortcut_8 十格，G_NG_GRID_COUNT=25），值 = 该格绑定
	// 的武器快捷栏序号（0=未绑定）。客户端主窗体打开时从玩家属性
	// NeiGongBind 整串快照（get_svr_ng_bind）：服务端不回存档则重登
	// 全零——表现为「内功栏放置武器/内功绑定的快捷栏序号没有存储」。
	clientCustomNeiGongBind int32 = 199
	// S2C 705：on_parser_customizing(configStr) 个性配置下行通道，
	// 0x1E 旧壳（Java LoginFlow.customizingFrame 同形）。
	serverCustomPushCustomizing int32 = 705
)

// 0x09 官方属性表索引（official_property_table.txt）：198 ShortcutKey
// type7 / 199 ShortcutKey1 type7 / 200 ShortcutKey2 type7 / 223 OperateMode
// type3。0x10 单属性帧的键位串一律加 " " 尾空格（参考端 60310 抓包逐字节
// 范本），0x0D 档内嵌的 198 无尾空格——两处形态不可混用。
const (
	keybindPropertyIndex        uint16 = 198
	keybindScheme1PropertyIndex uint16 = 199
	keybindScheme2PropertyIndex uint16 = 200
	operateModePropertyIndex    uint16 = 223
	// NeiGongBind（type7）：内功栏绑定串权威属性。0x10 单属性帧带
	// 尾空格、0x0D 档内嵌无尾空格，与 198 同方言两形态。
	neiGongBindPropertyIndex uint16 = 201
)

// neiGongBindSlotCount 是内功栏绑定串的固定格数（客户端
// form_main_shortcut.lua G_NG_GRID_COUNT=25；get_svr_ng_bind 按拆段数
// 校验，不是 26 段就整串回落全零）。串形 = 25 个 "v;" + 尾分号。
const neiGongBindSlotCount = 25

// validNeiGongBind 校验客户端上传的内功栏绑定串：恰好 25 个值加尾
// 分号（拆 26 段、末段空），各值为 0..99 的十进制数（绑定框 DataSource
// 序号，0=清除绑定）。不符则整串拒收（零帧），防脏串落库后把客户端
// 档位校验打回全零。
func validNeiGongBind(value string) bool {
	if !strings.HasSuffix(value, ";") {
		return false
	}
	fields := strings.Split(value, ";")
	if len(fields) != neiGongBindSlotCount+1 || fields[neiGongBindSlotCount] != "" {
		return false
	}
	for _, field := range fields[:neiGongBindSlotCount] {
		if field == "" || len(field) > 2 {
			return false
		}
		number, err := strconv.Atoi(field)
		if err != nil || number < 0 || number > 99 {
			return false
		}
	}
	return true
}

// customizingConfigLimit 是 role_customizing.customizing_value 的列宽
// （VARBINARY(65535)，0005 迁移定稿）。串为纯 ASCII + 控制字节
// （0x01-0x65），Go string 原样保留。
const customizingConfigLimit = 65535

// keybindValueLimit 对齐 role_keybinds 的 varchar(4096) 列宽与 Java
// handleKeybindUpload 校验。
const keybindValueLimit = 4096

// playerKeybind 镜像 role_keybinds 单行（键位域的权威内存副本）。
// 0x0D addObject 档内嵌 198/223/201 与 ClientReady 单属性帧都读这里；
// operateModeSet 区分「从未设置」（NULL，不下发，客户端保持本地默认）
// 与显式 0。neigongBind 空 = 从未绑定（不下发，客户端回落全零串）。
type playerKeybind struct {
	bind           string
	scheme1        string
	scheme2        string
	operateMode    int32
	operateModeSet bool
	neigongBind    string
}

// applyScheme 更新一份键位串（必须在 p.mu 内调用）。
func (k *playerKeybind) applySchemeLocked(scheme int32, value string) {
	switch scheme {
	case 0:
		k.bind = value
	case 1:
		k.scheme1 = value
	case 2:
		k.scheme2 = value
	}
}

func (p *playerActor) setKeybind(record playerKeybind) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keybind = record
}

func (p *playerActor) applyKeybindScheme(scheme int32, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keybind.applySchemeLocked(scheme, value)
}

func (p *playerActor) setOperateMode(mode int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keybind.operateMode = mode
	p.keybind.operateModeSet = true
}

func (p *playerActor) applyNeiGongBind(value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keybind.neigongBind = value
}

func (p *playerActor) keybindSnapshot() playerKeybind {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.keybind
}

// customizingFrame 编码 S2C 705（0x1E 旧壳 int 路由 + 窄串载荷），
// 客户端 on_parser_customizing → UpdateToConfigObj 灌回配置对象。
func customizingFrame(config string) ([]byte, error) {
	return serverCustomIntMessage(serverCustomPushCustomizing, customString(config))
}

// keybindSchemePropertyFrame 编码键位域单属性 0x10：scheme 0/1/2 →
// ShortcutKey/1/2（索引 198/199/200，type7）。线上串 = value + " "。
func keybindSchemePropertyFrame(scheme int32, value string) ([]byte, error) {
	var index uint16
	var name string
	switch scheme {
	case 0:
		index, name = keybindPropertyIndex, "ShortcutKey"
	case 1:
		index, name = keybindScheme1PropertyIndex, "ShortcutKey1"
	case 2:
		index, name = keybindScheme2PropertyIndex, "ShortcutKey2"
	default:
		return nil, fmt.Errorf("keybind scheme %d out of range", scheme)
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{
		Index: index, Name: name, Value: clientdata.StringValue(value + " "),
	}})
}

// operateModePropertyFrame 编码 0x10 OperateMode 单属性帧（idx 223，
// int32），同步客户端属性缓存（跨服重连 lua 直接 QueryProp）。
func operateModePropertyFrame(mode int32) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{
		Index: operateModePropertyIndex, Name: "OperateMode", Value: clientdata.Int32Value(mode),
	}})
}

// neiGongBindPropertyFrame 编码 0x10 NeiGongBind 单属性帧（idx 201，
// type7）。线上串 = value + " "（键位域同方言）；客户端 get_svr_ng_bind
// 以 QueryProp 读它初始化绑定框，且 update_to_srv 靠与它比对决定是否
// 需要再上传——不回推则两端永不收敛。
func neiGongBindPropertyFrame(value string) ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{{
		Index: neiGongBindPropertyIndex, Name: "NeiGongBind", Value: clientdata.StringValue(value + " "),
	}})
}

// pushSavedCustomizing 推存档个性配置（无档零帧：客户端保持默认并自行
// 上传基线）。705 是连接级配置对象，不随 0x0C 切图重置，无需重推。
//
// 2026-11-09 曾停推（官服镜像）：官服 c/jn/a 三份抓包全程 705 下行恒 0
// ——官服客户端本地自存配置（switch_mode 落 systeminfo.ini），从不靠
// 服务端恢复；且真机抓包实证（49166 会话）：官服客户端
// on_parser_customizing（custom_handler.lua:3959）每收一帧 705 就强制
// game_config_info.game_control_mode=0 与 camera_value=0，把引擎控制
// 对象打回传统 3D（左键=调视角），而 operate_control_mode 仍=1（设置
// 界面照显九阴特色）——表象即「点击地面零输出、WASD 正常」。
// 2026-11-10 恢复：真机客户端已定版 fork 版（本地不落盘），键位/快捷
// 配置全靠服务端回推，停推后重登即丢。若切回官服客户端须重新停推。
func pushSavedCustomizing(conn sceneMessageConnection, config string) error {
	if config == "" {
		return nil
	}
	frame, err := customizingFrame(config)
	if err != nil {
		return fmt.Errorf("encode customizing frame: %w", err)
	}
	if err := conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send customizing frame: %w", err)
	}
	return nil
}

// pushKeybindFrames 在 post-spawn 推送存档键位域：198（当前启用）、
// 199/200（方案一/二，保存面板 check_saved 读它们判定「已保存」）、
// 223（操作模式）、201（内功栏绑定串，内功栏主窗体打开时快照）。
// 无档属性零帧。调用方以会话闸控制只推一次，
// 0x0C 切图清空玩家对象属性视图后需复位闸重推（Java completeSceneReentry
// 同款；705 连接级，不在此列）。
func pushKeybindFrames(conn sceneMessageConnection, player *playerActor) error {
	if player == nil {
		return nil
	}
	bind := player.keybindSnapshot()
	schemes := []struct {
		scheme int32
		value  string
	}{{0, bind.bind}, {1, bind.scheme1}, {2, bind.scheme2}}
	for _, scheme := range schemes {
		if scheme.value == "" {
			continue
		}
		frame, err := keybindSchemePropertyFrame(scheme.scheme, scheme.value)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("push keybind scheme %d: %w", scheme.scheme, err)
		}
	}
	if bind.operateModeSet {
		frame, err := operateModePropertyFrame(bind.operateMode)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("push operate mode %d: %w", bind.operateMode, err)
		}
	}
	if bind.neigongBind != "" {
		frame, err := neiGongBindPropertyFrame(bind.neigongBind)
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("push neigong bind: %w", err)
		}
	}
	return nil
}

// customizingSession 跟踪一条连接上的 107 时序闸（Java LoginSession
// .customizingSeenUpload 对应物）。客户端登录期会先上传默认键位转储
// （此时恢复帧未必已应用）：会话首个上传在库里有存档时跳过落库，
// 防默认覆盖玩家配置。
type customizingSession struct {
	saved      string
	hasSaved   bool
	seenUpload bool
}

// handleCustomizingCustom 处理 C2S 107 个性配置上传。
func handleCustomizingCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, session *customizingSession, store *customizingDBStore, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSaveCustomizing {
		return false, nil
	}
	if player == nil {
		// 选角前的上传无法归属角色，直接忽略（不计入首传闸）。
		log.Printf("%s: ignore customizing upload before role selected", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 6 {
		return true, fmt.Errorf("customizing expects TVarList(107, string), got %v", custom.Values)
	}
	config := custom.Values[1].Text
	if config == "" {
		return true, nil
	}
	if len(config) > customizingConfigLimit {
		return true, fmt.Errorf("customizing config length %d exceeds %d", len(config), customizingConfigLimit)
	}
	firstUpload := !session.seenUpload
	session.seenUpload = true
	if firstUpload && session.hasSaved {
		// 恢复前/中的默认键位转储：跳过落库，回发存档再确认恢复态。
		if err := pushSavedCustomizing(conn, session.saved); err != nil {
			return true, fmt.Errorf("re-assert saved customizing: %w", err)
		}
		log.Printf("%s: skipped default customizing dump (%d bytes), re-asserted saved %d bytes", remote, len(config), len(session.saved))
		return true, nil
	}
	if err := store.Save(roleID, config); err != nil {
		log.Printf("%s: persist customizing role=%d: %v", remote, roleID, err)
		return true, nil
	}
	session.saved, session.hasSaved = config, true
	if err := pushSavedCustomizing(conn, config); err != nil {
		return true, fmt.Errorf("echo customizing: %w", err)
	}
	log.Printf("%s: saved customizing config %d bytes", remote, len(config))
	return true, nil
}

// printableKeybindValue 校验键位串是可打印 ASCII（0x20-0x7E）：真机
// 引擎会发 649B 形态串，放宽自旧的 [0-9,;]+ 白名单（Java 同口径）。
func printableKeybindValue(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7E {
			return false
		}
	}
	return true
}

// handleKeybindCustom 处理 C2S 195 键位方案上传：校验 → 回推 0x10
// 属性帧（客户端属性视图以服务端下发为准，不回推则保存面板恒
// 「未保存」）→ 落库。非法 scheme/串静默拒（零帧）。
func handleKeybindCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, store *keybindDBStore, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomKeybindUpload {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: ignore keybind upload before role selected", remote)
		return true, nil
	}
	if len(custom.Values) < 3 {
		return true, fmt.Errorf("keybind upload expects TVarList(195, int scheme, string), got %v", custom.Values)
	}
	scheme, ok := custom.Values[1].exactInt32()
	bind := custom.Values[2].Text
	if !ok || custom.Values[2].Type != 6 {
		return true, fmt.Errorf("keybind upload scheme/string mismatch: %v", custom.Values)
	}
	if scheme < 0 || scheme > 2 || bind == "" || len(bind) > keybindValueLimit || !printableKeybindValue(bind) {
		log.Printf("%s: ignore invalid keybind upload scheme=%d len=%d", remote, scheme, len(bind))
		return true, nil
	}
	frame, err := keybindSchemePropertyFrame(scheme, bind)
	if err != nil {
		return true, err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("echo keybind scheme %d: %w", scheme, err)
	}
	player.applyKeybindScheme(scheme, bind)
	if err := store.SaveScheme(roleID, scheme, bind); err != nil {
		log.Printf("%s: persist keybind role=%d scheme=%d: %v", remote, roleID, scheme, err)
	}
	log.Printf("%s: keybind scheme %d uploaded (%d bytes)", remote, scheme, len(bind))
	return true, nil
}

// handleOperateModeCustom 处理 C2S 99：立即回推 0x10 OperateMode 同步
// 客户端属性缓存，再落库。非法值静默拒（零帧）。
func handleOperateModeCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, store *keybindDBStore, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSetOperateMode {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: ignore operate mode before role selected", remote)
		return true, nil
	}
	if len(custom.Values) < 2 {
		return true, fmt.Errorf("operate mode expects TVarList(99, int mode), got %v", custom.Values)
	}
	mode, ok := custom.Values[1].exactInt32()
	if !ok || (mode != 0 && mode != 1) {
		log.Printf("%s: ignore unknown operate mode %d", remote, mode)
		return true, nil
	}
	frame, err := operateModePropertyFrame(mode)
	if err != nil {
		return true, err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("send operate mode %d: %w", mode, err)
	}
	player.setOperateMode(mode)
	if err := store.SaveOperateMode(roleID, mode); err != nil {
		log.Printf("%s: persist operate mode role=%d: %v", remote, roleID, err)
	}
	log.Printf("%s: operate mode set %d (%s)", remote, mode, map[bool]string{true: "2.5D click", false: "3D WASD"}[mode == 1])
	return true, nil
}

// handleNeiGongBindCustom 处理 C2S 199 内功栏绑定上传：校验 → 回推
// 0x10 NeiGongBind 同步客户端属性缓存 → 落库。非法串静默拒（零帧）。
// Java 参考服未实现此域（docs/lua64-protocol-dict.md 仅录得协议号），
// 语义按客户端 form_main_shortcut.lua 直读。
func handleNeiGongBindCustom(conn sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, store *keybindDBStore, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomNeiGongBind {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: ignore neigong bind before role selected", remote)
		return true, nil
	}
	if len(custom.Values) != 2 || custom.Values[1].Type != 6 {
		return true, fmt.Errorf("neigong bind expects TVarList(199, string), got %v", custom.Values)
	}
	value := custom.Values[1].Text
	if !validNeiGongBind(value) {
		log.Printf("%s: ignore invalid neigong bind len=%d", remote, len(value))
		return true, nil
	}
	frame, err := neiGongBindPropertyFrame(value)
	if err != nil {
		return true, err
	}
	if err := conn.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("echo neigong bind: %w", err)
	}
	player.applyNeiGongBind(value)
	if err := store.SaveNeiGongBind(roleID, value); err != nil {
		log.Printf("%s: persist neigong bind role=%d: %v", remote, roleID, err)
	}
	bound := 0
	for _, field := range strings.Split(strings.TrimSuffix(value, ";"), ";") {
		if field != "0" {
			bound++
		}
	}
	log.Printf("%s: neigong bind uploaded (%d/%d slots bound)", remote, bound, neiGongBindSlotCount)
	return true, nil
}
