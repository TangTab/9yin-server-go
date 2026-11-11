package main

// 官服登录时间线对齐帧簇（docs/官服登录帧对照与缺口清单.md §二/§八/§十）。
//
// c.pcapng（官服 7074 线）登录窗口实证的三个补充位：
//
//  0.1s  位（0x0A 记录表后、EntryScene 前）：m366 客户端 IP 回显、
//        m623 [0,0,3]、m655 服务器/账号信息（引擎级登录信息栏数据源）。
//  4.8s  位（ClientReady 技能视图簇后）：m607 账号+角色名声明、
//        m52 服务器时间 [1,unix]、m586 [5] 状态推送簇首（官服登录
//        尾单发一次，此后只随走停迁移发四连）。
//  4.8s  位（背包视图行后、View46 前）：战斗属性全量 0x10——官服
//        逐项单帧下发人物面板数值（StrAdd/Power/Def/Damage 等 ~110
//        帧）；本服按 progress.properties() 可计算子集对齐（其余项
//        无数据源，见对照文档 §十一 未处理清单）。
//
// m924→m294 对钟应答：官服对每条 C2S m924 回 m294
// [client_ms 回显, obj(玩家), unix 秒, uptime 秒(f64)]（c.pcapng ×139
// 逐对实证）；本服保留 1s 周期推（fork 52905 方言，CD 对钟依赖），
// 叠加官服形态的逐条应答，两方言共存互不干扰。
//
// m327 任务消息应答：官服对 C2S m327[6,"任务ID串"]（CLIENT_CUSTOMMSG_
// TASK_MSG，lua64 定名）回 0x13 Task_Accepted#29 位图写格 ×9 两轮幂等
// 重推；本服以已授予的接受任务行等价回放（幂等，客户端行已存在则覆写
// 同值）。

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
)

// 官方消息号（lua64 client_custom_define.lua / c.pcapng 实证）。
const (
	officialMsgServerTime     int32 = 52  // [1, unix秒]
	officialMsgClientIP       int32 = 366 // [客户端IP]
	officialMsgServerInfo     int32 = 623 // [0,0,3]
	officialMsgAccountCard    int32 = 607 // [账号, 角色名]
	officialMsgServerIdentity int32 = 655 // [sid,sid,服务器名,sid,账号,账号ID]
	officialMsgStatePushHead  int32 = 586 // 登录簇首 [5]
	officialMsgTimeSync       int32 = 294 // 对钟应答
	officialMsgJyzDomain      int32 = 416 // 九阴志域声明 ["DomainJiuYinZhi", step]
)

// C2S 0x0A 信封新增处理的官服消息号（lua64 client_custom_define.lua）。
const (
	clientCustomMsgDelay int32 = 924 // CLIENT_CUSTOMMSG_MSG_DELAY（引擎 MessageDelay 直发）
	clientCustomTaskMsg  int32 = 327 // CLIENT_CUSTOMMSG_TASK_MSG（send_task_msg 等）
)

// activityCustomMsgKnown 是 0x0A 活动信封的 msgId 已知集：原内联白名单
// 全集 + m924/m327。集合外不再静默丢弃——main.go 记一条 unhandled 日志
// （对照文档 §五-C：官服登录期 27 种 UI 上报此前完全不可见）。
func activityCustomMsgKnown(id int32) bool {
	switch id {
	case clientCustomSetShortcut, clientCustomRemoveShortcut, clientCustomSitCross,
		clientCustomSaveCustomizing, clientCustomKeybindUpload, clientCustomSetOperateMode,
		clientCustomNeiGongBind,
		211, 215, 216, 217, 1006,
		clientCustomActiveParry,
		clientCustomSelectTarget, clientCustomAttackTarget, clientCustomBlockState,
		clientCustomMovementUpload, clientCustomRushMoveUpload, clientCustomCastDashLanding,
		clientCustomChangeWarning,
		clientCustomMoveItem, clientCustomUseItem, clientCustomDeleteItem, clientCustomArrangeBag,
		clientCustomShowEquipType, clientCustomFashionSuit, clientCustomWardrobeCard,
		clientCustomShopBuy, clientCustomShopSell, clientCustomSetTitle,
		clientCustomJingMaiCultivate, clientCustomJingMaiActive,
		clientCustomFreshManChoice, clientCustomGuideAddSchool, clientCustomSchoolMessage,
		clientCustomPickupDropItem, clientCustomCloseDropBox,
		clientCustomJyzQuery, clientCustomMapNavigate, clientCustomTersure,
		clientCustomMsgDelay, clientCustomTaskMsg:
		return true
	}
	return false
}

// officialServerIdentity 返回 m655 服务器标识四元组：服务器 ID/名称可经
// NINEYIN_OFFICIAL_SERVER_ID / NINEYIN_OFFICIAL_SERVER_NAME 覆盖（默认取
// c.pcapng 官服值 7100037/侠骨丹心——登录信息栏显示源，客户端不校验）。
func officialServerIdentity() (serverID int32, serverName string) {
	serverID = 7100037
	if v := strings.TrimSpace(os.Getenv("NINEYIN_OFFICIAL_SERVER_ID")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil {
			serverID = int32(parsed)
		}
	}
	serverName = "侠骨丹心"
	if v := strings.TrimSpace(os.Getenv("NINEYIN_OFFICIAL_SERVER_NAME")); v != "" {
		serverName = v
	}
	return serverID, serverName
}

// officialAccountLabel 返回 m655/m607 的账号显示名。2026-11-11 起 op02
// 账号槽密文经 fork 域密码逆阶梯解出明文账号（docs/fork-login-crypto.md，
// DecryptAccountFields 已滤非 ASCII——tag6 窄串只放 ASCII，官方/Java 同俗，
// 中文账号名回空防账号栏乱码），即回传真实用户名；
// NINEYIN_OFFICIAL_ACCOUNT 仍可强制覆盖。
func officialAccountLabel(loginAccount string) string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_OFFICIAL_ACCOUNT")); v != "" {
		return v
	}
	return loginAccount
}

// writeOfficialLoginInfoFrames 下发 0.1s 位的登录信息帧簇（m366/m623/m655）。
// 位置：0x0A 记录表之后、建视图之前（c.pcapng 0.097-0.099s 同位）。
func writeOfficialLoginInfoFrames(link sceneMessageConnection, remote, account string, roleName string, roleID uint32) {
	ip, _, err := splitHostPort(remote)
	if err != nil {
		ip = remote
	}
	frames := [][]byte{}
	if frame, err := serverCustomIntMessage(officialMsgClientIP, customString(ip)); err == nil {
		frames = append(frames, frame)
	}
	if frame, err := serverCustomIntMessage(officialMsgServerInfo, customInt(0), customInt(0), customInt(3)); err == nil {
		frames = append(frames, frame)
	}
	serverID, serverName := officialServerIdentity()
	accountID := int64(roleID)
	if frame, err := serverCustomIntMessage(officialMsgServerIdentity,
		customInt(serverID), customInt(serverID), customWideString(serverName),
		customInt(serverID), customString(account), customInt64(accountID)); err == nil {
		frames = append(frames, frame)
	}
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write official login info frame: %v", remote, err)
			return
		}
	}
	log.Printf("%s: pushed official login info frames (m366/m623/m655 server=%d name=%s)", remote, serverID, serverName)
}

// writeOfficialReadyInfoFrames 下发 4.8s 位的就绪信息帧簇：m607/m52/m586。
// 位置：ClientReady 技能视图（View43/40/45）授予之后（c.pcapng 4.80-4.83s 同位）。
func writeOfficialReadyInfoFrames(link sceneMessageConnection, remote, account string, roleName string, now time.Time) {
	frames := [][]byte{}
	if frame, err := serverCustomIntMessage(officialMsgAccountCard, customString(account), customWideString(roleName)); err == nil {
		frames = append(frames, frame)
	}
	if frame, err := serverCustomIntMessage(officialMsgServerTime, customInt(1), customInt64(now.Unix())); err == nil {
		frames = append(frames, frame)
	}
	if frame, err := serverCustomIntMessage(officialMsgStatePushHead, customInt(5)); err == nil {
		frames = append(frames, frame)
	}
	for _, frame := range frames {
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write official ready info frame: %v", remote, err)
			return
		}
	}
	log.Printf("%s: pushed official ready info frames (m607/m52/m586)", remote)
}

// writeOfficialCombatAttributeFrames 战斗属性全量：progress.properties() 的
// 战斗面板子集逐项单帧 0x10（官服 4.82-4.84s 形态：单属性帧，非大批）。
// 位置：背包视图行之后、View46 之前；官服含 ~110 项，本服仅发可计算项
// （五维/Power/Hit/Dodge/Va/Def/Parry/Damage/HP·MP 上限加成等 ~45 项，
// 其余 AI*/Mul 系无数据源——对照文档 §十一）。
func writeOfficialCombatAttributeFrames(link sceneMessageConnection, player *playerActor, remote string) {
	if player == nil {
		return
	}
	props := player.combatPanelProperties()
	sent := 0
	for _, prop := range props {
		if !officialCombatPanelProp(prop.Index) {
			continue
		}
		frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{prop})
		if err != nil {
			log.Printf("%s: encode combat panel prop %d: %v", remote, prop.Index, err)
			return
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write combat panel prop %d: %v", remote, prop.Index, err)
			return
		}
		sent++
	}
	log.Printf("%s: pushed official combat panel attributes (%d single-property frames)", remote, sent)
}

// officialCombatPanelProp 圈定 progress.properties() 中的战斗面板子集
// （player_progress.go 索引常量；StrAdd 系面板显示值由五维主值承担同位
// 语义：官服面板 Add 列与主值独立下发，本服无独立加成数据源时跳过）。
func officialCombatPanelProp(index uint16) bool {
	switch index {
	case propStr, propSta, propDex, propIng, propSpi,
		propMeleePower, propShotPower, propMagicPower,
		propMeleePowerAdd, propShotPowerAdd, propMagicPowerAdd,
		propPhyHit, propPhyHitAdd, propMagicHit, propMagicHitAdd,
		propDodge, propDodgeAdd,
		propPhyVa, propPhyVaAdd, propMagicVa, propMagicVaAdd,
		propMaxParry, propMaxParryAdd,
		propPhyDef, propStiDef, propNegDef, propMasDef, propJujDef,
		propMinMeleeDamage, propMaxMeleeDamage,
		propMaxHPAdd, propMaxMPAdd, propMinMagicDefAdd,
		propMaxLevel,
		propYinLevel, propYangLevel, propTaijiLevel,
		propWuJiPoint, propMaxWuJiPoint,
		propCurNeiGongWuXing:
		return true
	}
	return false
}

// combatPanelSnapshot 战斗面板子集快照：索引 → 单属性帧编码字节。
// 以真实编码字节作比较基准（值与线型变化都能命中）；编码失败的项跳过。
func combatPanelSnapshot(player *playerActor) map[uint16][]byte {
	if player == nil {
		return nil
	}
	snapshot := make(map[uint16][]byte, 32)
	for _, prop := range player.combatPanelProperties() {
		if !officialCombatPanelProp(prop.Index) {
			continue
		}
		frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{prop})
		if err != nil {
			continue
		}
		snapshot[prop.Index] = frame
	}
	return snapshot
}

// pushCombatAttributeDelta 换装后的战斗属性增量推送（官服运行期模型，
// c.pcapng 战斗期实证：单属性 0x10 小帧、只推变化项，从不重推全量——
// 全量仅在登录/对象新建时下发）。与快照逐项字节比对，变化/新增项以
// 单属性帧下发；消失项跳过（保留旧值，等待下次登录全量收敛）。
// 本服当前可计算输入 = 装备内功书 + 镇派兵器槽（普通防具加成未建模，
// 建模落地后本通道自动生效，无需再改）。
func pushCombatAttributeDelta(link sceneMessageConnection, player *playerActor, before map[uint16][]byte, remote string) {
	if player == nil || before == nil {
		return
	}
	after := combatPanelSnapshot(player)
	sent := 0
	for index, frame := range after {
		old, ok := before[index]
		if ok && bytes.Equal(old, frame) {
			continue
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write combat attr delta prop=%d: %v", remote, index, err)
			return
		}
		sent++
	}
	if sent > 0 {
		log.Printf("%s: pushed combat attribute delta after equip change (%d props)", remote, sent)
	}
}

// officialTimeSyncReplyFrame m924→m294 官服形态应答：
// [client_ms 回显(i64), obj(玩家), unix 秒(i64), uptime 秒(f64)]。
// （c.pcapng 全量时间线 PAIR ×139 实证；uptime 为服务器运行秒数浮点。）
func officialTimeSyncReplyFrame(clientMs int64, start time.Time, now time.Time) ([]byte, error) {
	uptime := float64(now.Sub(start).Milliseconds()) / 1000
	if math.IsNaN(uptime) || uptime < 0 {
		uptime = 0
	}
	return serverCustomIntMessage(officialMsgTimeSync,
		customInt64(clientMs),
		customObject(playerObjectID, playerOwnerID),
		customInt64(now.Unix()),
		customFloat64(uptime),
	)
}

// repushAcceptedTaskFrames m327 应答：重放已授予的 Task_Accepted 行
// （官服对 m327[6] 回 0x13 位图写格 ×9 两轮；本服等价物 = 接受任务
// 行重写，幂等——客户端行已存在时覆写同值）。
func repushAcceptedTaskFrames(link sceneMessageConnection, remote string) error {
	for _, taskID := range facultyAcceptedTaskIDs() {
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordTaskAccepted, []recordCell{
			recordInt(taskID), recordInt(0), recordString(""),
			recordInt(0), recordString(""), recordString(""), recordInt(0),
		})
		if err != nil {
			return fmt.Errorf("encode Task_Accepted task %d: %w", taskID, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("add Task_Accepted task %d: %w", taskID, err)
		}
	}
	return nil
}

// splitHostPort 去端口（net.SplitHostPort 语义，避免引入 net 依赖差异）。
func splitHostPort(addr string) (string, string, error) {
	if i := strings.LastIndex(addr, ":"); i > 0 {
		if strings.HasPrefix(addr, "[") { // [v6]:port
			if j := strings.Index(addr, "]"); j >= 0 && j < i {
				return addr[1:j], addr[j+2:], nil
			}
		}
		return addr[:i], addr[i+1:], nil
	}
	return addr, "", fmt.Errorf("no port in %q", addr)
}

// wuxingLevelBuckets 按「已学内功 + 已学技能」的 wuxing 分桶累计等级
// （用户口径：阴阳太极=当前修炼的内功、技能的属性）：
//   - 内功 5=太极→TaijiLevel、6=阴柔→YinLevel、7=阳刚→YangLevel
//   - 技能 3=阴→YinLevel、4=阳→YangLevel（语义直配；1=刚/2=柔 无
//     阴阳太极对应桶，不入）
//
// 数据源 wuxuebaseinfo.ini nWuXing（neigong_effects.go 目录）；目录
// 不可用时全零（技能不走 skillWuXing 的缺行回退 5——回退会误入太极桶，
// 桶计算只认目录原值，0 跳过）。官服 1089/623/299 的精确构成无样本，
// 公式为语义推导（2026-11-10 注）。
func wuxingLevelBuckets(books []learnedNeiGong, skills map[string]int32) (yin, yang, taiji int32) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return 0, 0, 0
	}
	for i := range books {
		wuXing := catalog.wuXingOf(books[i].configID)
		switch wuXing {
		case 5:
			taiji += books[i].level
		case 6:
			yin += books[i].level
		case 7:
			yang += books[i].level
		}
	}
	for configID, level := range skills {
		switch catalog.wuXingOf(configID) {
		case 3:
			yin += level
		case 4:
			yang += level
		}
	}
	return yin, yang, taiji
}

// 武技点（经脉无忌格货币）：无存档源，env 覆盖（默认 0）。
func wuJiPointEnv() int32 {
	return envInt32OrDefault("NINEYIN_WUJI_POINT", 0)
}

func wuJiPointMaxEnv() int32 {
	return envInt32OrDefault("NINEYIN_WUJI_MAX", 0)
}

func envInt32OrDefault(key string, fallback int32) int32 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(parsed)
		}
	}
	return fallback
}

// writeOfficialLoginBuffFrames 官服登录期 m169 buff（两值形态 [169,
// buff名, obj(玩家)]，c.pcapng 0.1s/4.7s 实证）：公共 buff 组标志
// buff_group_flag_101（BufferGroupPublicRec#59 同域，永久）+ 当前装备
// 内功的可见 buff（buf_ng_*，learnedNeiGong.buffID 表源，永久）。
// 限时 buff（官服 buf_baosd_02 等）无存档数据源，未发（对照文档 §11.2）。
func writeOfficialLoginBuffFrames(link sceneMessageConnection, player *playerActor, remote string) {
	if player == nil {
		return
	}
	player.mu.Lock()
	name := player.name
	var buffID string
	if book := player.progress.book(player.progress.curNeiGong); book != nil {
		buffID = book.buffID
	}
	player.mu.Unlock()
	names := []string{"buff_group_flag_101"}
	if buffID != "" {
		names = append(names, buffID)
	}
	for _, buff := range names {
		frame, err := serverCustomIntMessage(169, customString(buff), customObject(playerObjectID, playerOwnerID))
		if err != nil {
			log.Printf("%s: encode login buff %s: %v", remote, buff, err)
			return
		}
		if err := link.WriteFrame(frame); err != nil {
			log.Printf("%s: write login buff %s: %v", remote, buff, err)
			return
		}
	}
	log.Printf("%s: pushed official login buffs %v (player=%s)", remote, names, name)
}

// equipAttributeSums 普通装备属性折算：已穿戴装备的随机属性
// （RandomPropRec 快照，PropID=属性表索引）按索引求和。技能行
// （SkillRows）不进面板数值。uid 空的行（starter/无随机属性）跳过。
func equipAttributeSums(equips []equipRow, props map[string]itemRandomProps) map[uint16]int32 {
	sums := make(map[uint16]int32, 16)
	for _, equip := range equips {
		if equip.uid == "" {
			continue
		}
		snapshot, ok := props[equip.uid]
		if !ok {
			continue
		}
		for _, line := range snapshot.PropRows {
			if line.PropID <= 0 || line.PropID > 0xFFFF {
				continue
			}
			sums[uint16(line.PropID)] += int32(line.Value)
		}
	}
	return sums
}

// setEquipAttributeAdds 更新装备属性折算表（穿脱/登录装载后调用）。
func (p *playerActor) setEquipAttributeAdds(adds map[uint16]int32) {
	p.mu.Lock()
	p.equipAdds = adds
	p.mu.Unlock()
}

// combatPanelProperties 战斗面板属性权威出口：progressProperties()
// 叠加装备随机属性折算（equipAdds）。仅合并 int32 线型的属性——随机
// 属性行（RandomPropRec）的 PropID 即属性表索引，与 0x10 单属性帧同域。
// 登录全量簇（writeOfficialCombatAttributeFrames）与换装增量簇
// （combatPanelSnapshot/pushCombatAttributeDelta）必须共用本出口，
// 保证前后快照口径一致。
func (p *playerActor) combatPanelProperties() []clientdata.IndexedProperty {
	props := p.progressProperties()
	p.mu.Lock()
	adds := p.equipAdds
	skills := p.learnedSkills
	books := append([]learnedNeiGong(nil), p.progress.books...)
	p.mu.Unlock()
	if len(adds) > 0 {
		for i := range props {
			add, ok := adds[props[i].Index]
			if !ok || add == 0 {
				continue
			}
			if props[i].Value.Type != clientdata.WireInt32 {
				continue
			}
			props[i].Value = clientdata.Int32Value(props[i].Value.I32 + add)
		}
	}
	// 阴阳太极累计（内功+技能双输入）在此注入：properties() 是
	// playerProgress 域（无技能表），actor 层才是两域交汇点。登录
	// 全量簇/换装快照/换书与秘籍升级的 delta 全部经本出口，口径一致。
	yin, yang, taiji := wuxingLevelBuckets(books, skills)
	props = append(props,
		clientdata.IndexedProperty{Index: propYinLevel, Name: "YinLevel", Value: clientdata.Int32Value(yin)},
		clientdata.IndexedProperty{Index: propYangLevel, Name: "YangLevel", Value: clientdata.Int32Value(yang)},
		clientdata.IndexedProperty{Index: propTaijiLevel, Name: "TaijiLevel", Value: clientdata.Int32Value(taiji)},
	)
	return props
}
