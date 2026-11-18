package main

// 梳妆阁图纸（attire blend）——server2 方案（docs/reference/梳妆阁图纸
// 功能说明.md）在 official-2433 血统上的移植，四处按本分支实测校准：
//
//   - 记录 blend_collect_rec 用官方 0x0A 表下标 179（server2 的 25 是其
//     自协商方言表位，本仓库明确拒绝该做法），列型 [7,3,3,1]——第四列
//     ACTIVE 是 byte（string+int×3 编码器会拒收）。
//   - 图纸条目 = tool_item.ini [Tab_b_*]（男）/[Tab_g_*]（女）段，取
//     ItemType（601 帽/602 衣/603 裤/604 鞋）与 ArtPack（数字美术 id，
//     如 14233）；模型 = itemartstatic.ini 数字节的 MaleModel/FemaleModel
//     ——该表已被 loadEquipmentResolver 整表装进 equipmentModels.models，
//     零新增装载（官方抓包 1967(ReplacePack)=14233 与图纸 ArtPack 同值
//     实证此链）。
//   - 绑定 key 用物品视图 UniqueID 同口径（itemUID：列值 > MD5(role:
//     kind:seq) > MD5(role:kind:slot)），客户端 692 原样回传，双方一致。
//   - 存档 MySQL role_attire_blends（0022 迁移）——本分支外观域刻意
//     MySQL-only，不用 server2 的 data/梳妆阁数据.json。
//
// 语义（server2 终态「晚4」）：图纸绑定的是具体装备件（uid），穿着被
// 绑定件才替换模型，卸下即恢复；图标（ReplacePack/ReplaceID）跟物品走
// （纸娃娃/背包行均替换，绑定行撤直发 Photo——直发 Photo 优先级最高会
// 遮住 ReplacePack）；优先级 时装槽17-19 > 图纸融合 > 身份装套装16 >
// 真实装备。保存不收费（use_type 0-3 全按直存处理）。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/role"
)

// ---- 图纸目录 ----

// attireBlueprint 一张梳妆阁图纸（tool_item.ini Tab_* 段）。
type attireBlueprint struct {
	configID string
	artPack  string // 数字美术 id（ArtPack 原值字符串，如 "14233"）
	itemType int32  // 601 帽/602 衣/603 裤/604 鞋
	male     bool   // Tab_b_=true / Tab_g_=false
}

// attireCatalogData 图纸目录（升序 configID，行序稳定可测）。
type attireCatalogData struct {
	order []string
	byID  map[string]attireBlueprint
}

func (c *attireCatalogData) lookup(configID string) (attireBlueprint, bool) {
	entry, ok := c.byID[strings.ToLower(configID)]
	return entry, ok
}

// attireCatalogOnce / attireCatalogLoaded 测试注入点（cardItemCatalogOnce
// 同款；nil 时按 defaultModernShareRoot 装载真实 tool_item.ini）。
var (
	attireCatalogOnce   func() *attireCatalogData
	attireCatalogLoaded *attireCatalogData
)

// attireCatalogOf 图纸目录单例（首次调用装载，目录缺失按空目录降级）。
func attireCatalogOf() *attireCatalogData {
	if attireCatalogOnce != nil {
		return attireCatalogOnce()
	}
	if attireCatalogLoaded == nil {
		attireCatalogLoaded = attireCatalogFromTable(
			loadOptionalINI("tool_item.ini", defaultModernShareRoot, "item"))
		log.Printf("attire blend catalog ready: %d blueprints", len(attireCatalogLoaded.order))
	}
	return attireCatalogLoaded
}

// attireCatalogFromTable 纯函数装载：Tab_b_*/Tab_g_* 段取 ItemType
// （601-604 过滤）与 ArtPack；order 升序（行序稳定可测）。
func attireCatalogFromTable(table iniTable) *attireCatalogData {
	loaded := &attireCatalogData{byID: map[string]attireBlueprint{}}
	for section, fields := range table {
		lower := strings.ToLower(section)
		male := strings.HasPrefix(lower, "tab_b_")
		if !male && !strings.HasPrefix(lower, "tab_g_") {
			continue
		}
		itemType, artPack := int32(0), ""
		for _, field := range fields {
			if strings.EqualFold(field.key, "ItemType") {
				if parsed, err := strconv.Atoi(strings.TrimSpace(field.value)); err == nil {
					itemType = int32(parsed)
				}
				continue
			}
			if strings.EqualFold(field.key, "ArtPack") {
				artPack = strings.TrimSpace(field.value)
			}
		}
		if !isValidBlendSlot(blendSlotOf(itemType)) || artPack == "" {
			continue
		}
		loaded.byID[lower] = attireBlueprint{
			configID: section, artPack: artPack, itemType: itemType, male: male,
		}
	}
	loaded.order = make([]string, 0, len(loaded.byID))
	for configID := range loaded.byID {
		loaded.order = append(loaded.order, configID)
	}
	sort.Strings(loaded.order)
	return loaded
}

// blendSlotOf 图纸 ItemType → 装备槽（601→1 帽/602→3 衣/603→4 裤/604→8 鞋；
// 其余 0=非图纸）。
func blendSlotOf(itemType int32) int32 {
	switch itemType {
	case 601:
		return slotHat
	case 602:
		return slotCloth
	case 603:
		return slotPants
	case 604:
		return slotShoes
	}
	return 0
}

// isValidBlendSlot 槽是否图纸可绑定的四件之一（帽/衣/裤/鞋）。
func isValidBlendSlot(slot int32) bool {
	switch slot {
	case slotHat, slotCloth, slotPants, slotShoes:
		return true
	}
	return false
}

// blendModelOf 图纸按性别解析模型（itemartstatic 数字节 → Male/Female）。
func blendModelOf(bp attireBlueprint, sex byte) string {
	return equipmentModels.resolveArt(bp.artPack, sex)
}

// ---- blendDBStore（role_attire_blends，skinDBStore 同款降级语义） ----

// marshalBlends 写库 JSON 对象形态（key 升序——encoding/json 对 map 键
// 恒排序，确定性好测）。
func marshalBlends(blends map[string]string) string {
	if len(blends) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(blends)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

// parseBlends 解析 JSON 列（空/坏行返回 nil）。
func parseBlends(raw string) map[string]string {
	trimmed := trimSpaceASCII(raw)
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var blends map[string]string
	if err := json.Unmarshal([]byte(trimmed), &blends); err != nil {
		return nil
	}
	return blends
}

type blendDBStore struct {
	db *sql.DB
}

func newBlendDBStore(db *sql.DB) *blendDBStore {
	return &blendDBStore{db: db}
}

// Load 读取绑定表；无行/DB 错误按「无存档」处理（空态）。
func (store *blendDBStore) Load(roleID role.RoleID) (map[string]string, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	var blends string
	err := store.db.QueryRowContext(ctx,
		`SELECT blends FROM role_attire_blends WHERE role_id = ?`, roleID).Scan(&blends)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		log.Printf("attire blend store: load role=%d: %v", roleID, err)
		return nil, false
	}
	return parseBlends(blends), true
}

// Save upsert 绑定表（整表替换；version 逐次 +1）。
func (store *blendDBStore) Save(roleID role.RoleID, blends map[string]string) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("attire blend store: no database")
	}
	if roleID == 0 {
		return errors.New("attire blend store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), appearanceQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`INSERT INTO role_attire_blends (role_id, blends, version, updated_at)
VALUES (?, CAST(? AS JSON), 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE blends = VALUES(blends),
  version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, marshalBlends(blends)); err != nil {
		return fmt.Errorf("persist attire blends role=%d: %w", roleID, err)
	}
	return nil
}

// ---- playerActor 图纸态访问器（锁纪律 p.mu） ----

func (p *playerActor) restoreBlends(blends map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.attireBlends = cloneBlends(blends)
}

// applyBlends 原子替换绑定表并返回快照（692 保存/GM 清空后）。
func (p *playerActor) applyBlends(blends map[string]string) map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.attireBlends = blends
	return cloneBlends(p.attireBlends)
}

func (p *playerActor) blendSnapshot() map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneBlends(p.attireBlends)
}

// blendRowsPushedFlag / setBlendRowsPushed blend_collect_rec 行的
// ClientReady 一次性闸（静态行内容，无镜像计数）。
func (p *playerActor) blendRowsPushedFlag() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.blendRowsPushed
}

func (p *playerActor) setBlendRowsPushed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blendRowsPushed = true
}

func cloneBlends(blends map[string]string) map[string]string {
	if blends == nil {
		return nil
	}
	return mapsClone(blends)
}

func mapsClone(source map[string]string) map[string]string {
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

// ---- blend_collect_rec 记录推送（登录/0x0C 重入） ----

// pushBlendCollectRows 全解锁推送：目录序逐行 OWN=1/USED=0/ACTIVE=1
// （私服口径——梳妆阁面板全部可用；行内容静态，无需镜像记账，0x0C
// 随玩家对象清空后整体重放即可）。空目录零帧。
func pushBlendCollectRows(conn sceneMessageConnection) error {
	catalog := attireCatalogOf()
	for _, configID := range catalog.order {
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordBlendCollect, []recordCell{
			recordString(configID),
			recordInt(1),  // OWNNUM
			recordInt(0),  // USEDNUM
			recordByte(1), // ACTIVE（byte 列）
		})
		if err != nil {
			return fmt.Errorf("encode blend_collect_rec %s: %w", configID, err)
		}
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

// ---- 外观/图标覆盖助手 ----

// roleIDOf 读取会话角色 id（创建期一次写入，读侧仍走锁保持纪律）。
func (p *playerActor) roleIDOf() role.RoleID {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.roleID
}

// blendOverlay 玩家当前图纸模型覆盖（外观帧用；等价
// blendOverlayFor(roleID, blendSnapshot(), sex, equips)）。
func (p *playerActor) blendOverlay(sex byte, equips []equipRow) map[int32]string {
	return blendOverlayFor(p.roleIDOf(), p.blendSnapshot(), sex, equips)
}

// blendOverlayFor 已穿装备的图纸模型覆盖（槽→模型；仅帽/衣/裤/鞋，绑定
// key = 该穿着件下发的 UniqueID——itemUID 同口径，legacy 无 uid 行按
// role:equip:seq 派生）。未穿着被绑定件不产生条目（卸下即恢复语义）。
func blendOverlayFor(roleID role.RoleID, blends map[string]string, sex byte, equips []equipRow) map[int32]string {
	if len(blends) == 0 {
		return nil
	}
	catalog := attireCatalogOf()
	var overlay map[int32]string
	for _, equip := range equips {
		if !isValidBlendSlot(equip.slot) {
			continue
		}
		configID, bound := blends[itemUID(uint64(roleID), "equip", equip.uid, equip.seq, equip.slot)]
		if !bound {
			continue
		}
		bp, known := catalog.lookup(configID)
		if !known || blendSlotOf(bp.itemType) != equip.slot {
			continue // 陈旧绑定/部位不符：忽略（存档残留防御）
		}
		if model := blendModelOf(bp, sex); model != "" {
			if overlay == nil {
				overlay = make(map[int32]string, 4)
			}
			overlay[equip.slot] = model
		}
	}
	return overlay
}

// blendBindingFor 视图行属性查询：行 uid 键命中绑定返回图纸条目。
func blendBindingFor(roleID role.RoleID, kind string, uid string, seq int64, slot int32, blends map[string]string) (attireBlueprint, bool) {
	if len(blends) == 0 {
		return attireBlueprint{}, false
	}
	configID, bound := blends[itemUID(uint64(roleID), kind, uid, seq, slot)]
	if !bound {
		return attireBlueprint{}, false
	}
	return attireCatalogOf().lookup(configID)
}

// blendReplaceViewProps 视图行的图纸图标替换属性（ReplacePack=数字
// 美术 id、ReplaceID=图纸 ConfigID）。命中时调用方须撤直发 Photo
// （优先级最高会遮住 ReplacePack）；未命中返回 nil、Photo 照发。
func blendReplaceViewProps(roleID role.RoleID, kind string, uid string, seq int64, slot int32, blends map[string]string) []serverViewProperty {
	bp, bound := blendBindingFor(roleID, kind, uid, seq, slot, blends)
	if !bound {
		return nil
	}
	pack, err := strconv.Atoi(bp.artPack)
	if err != nil || pack <= 0 {
		return nil
	}
	return []serverViewProperty{
		viewInt(propReplacePack, int32(pack)),
		viewString(propReplaceID, bp.configID),
	}
}

// ---- C2S 692 梳妆阁保存 ----

// attireBlendPair 692 info 载荷的一对「图纸ConfigID,装备uid」。
type attireBlendPair struct {
	configID string
	uid      string
}

// parseAttireBlendInfo 解析 "cfg,uid,cfg,uid,..."（套装路径的 ";index;
// flag" 尾段忽略；奇数残项丢弃）。
func parseAttireBlendInfo(info string) []attireBlendPair {
	if index := strings.IndexByte(info, ';'); index >= 0 {
		info = info[:index]
	}
	parts := strings.Split(info, ",")
	pairs := make([]attireBlendPair, 0, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		configID := strings.TrimSpace(parts[i])
		uid := strings.TrimSpace(parts[i+1])
		if configID == "" || uid == "" {
			continue
		}
		pairs = append(pairs, attireBlendPair{configID: configID, uid: uid})
	}
	return pairs
}

// handleAttireEquipBlendCustom 处理 [692, use_type(int), info(string)]：
// 图纸对逐项校验（目录存在/性别匹配/uid 对得上穿着或背包装备件、穿着
// 时部位匹配）→ 合并进绑定表 → 落库 → 绑定穿着槽逐槽重推（纸娃娃行
// 含新图标 + 0x10 外观帧）→ sysinfo 377 保存成功提示（server2 校准：
// a1[769]→ID=(索引-15)/2=377，string_id 16596「保存形象成功」）。
// use_type 0-3 全按免费直存（私服口径）。
func handleAttireEquipBlendCustom(h *appearanceSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomAttireEquipBlend {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: ignore attire blend before role selected", h.remote)
		return true, nil
	}
	if len(custom.Values) < 3 {
		log.Printf("%s: attire blend expects TVarList(692, int use_type, string info), got %d values", h.remote, len(custom.Values))
		return true, nil
	}
	useType, ok := custom.Values[1].exactInt32()
	if !ok {
		log.Printf("%s: attire blend use_type is not an int: %v", h.remote, custom.Values[1])
		return true, nil
	}
	info := ""
	if custom.Values[2].Type == 6 {
		info = custom.Values[2].Text
	}
	pairs := parseAttireBlendInfo(info)
	if len(pairs) == 0 {
		log.Printf("%s: attire blend use_type=%d rejected: no cfg,uid pairs in %q", h.remote, useType, info)
		return true, nil
	}
	sex := h.player.visualSexOf()
	equips := h.player.sessionEquips()
	bag := h.player.sessionBag()
	catalog := attireCatalogOf()
	merged := h.player.blendSnapshot()
	if merged == nil {
		merged = map[string]string{}
	}
	applied := 0
	for _, pair := range pairs {
		bp, known := catalog.lookup(pair.configID)
		if !known {
			log.Printf("%s: attire blend ignore unknown blueprint %s", h.remote, pair.configID)
			continue
		}
		if bp.male != (sex == 0) {
			log.Printf("%s: attire blend ignore sex mismatch %s (player sex=%d)", h.remote, pair.configID, sex)
			continue
		}
		worn, wornHere := equipUIDAt(equips, h.roleID, pair.uid)
		if wornHere {
			if blendSlotOf(bp.itemType) != worn.slot {
				log.Printf("%s: attire blend ignore part mismatch %s -> slot=%d", h.remote, pair.configID, worn.slot)
				continue
			}
		} else if !bagUIDKnown(bag, h.roleID, pair.uid) {
			log.Printf("%s: attire blend ignore unknown equip uid %s", h.remote, pair.uid)
			continue
		}
		merged[pair.uid] = bp.configID
		applied++
	}
	if applied == 0 {
		return true, nil
	}
	blendsNow := h.player.applyBlends(merged)
	if err := h.blends.Save(h.roleID, blendsNow); err != nil {
		log.Printf("%s: persist attire blends role=%d: %v", h.remote, h.roleID, err)
	}
	// 绑定且当前穿着的槽位逐槽重推：纸娃娃行（含 ReplacePack/ReplaceID
	// 新图标）+ 0x10 外观帧（pushWearFrames 内含）。仅背包绑定无视觉
	// 变化，靠背包行下次同步携带图标。
	for _, equip := range equips {
		if !isValidBlendSlot(equip.slot) {
			continue
		}
		if _, bound := blendsNow[itemUID(uint64(h.roleID), "equip", equip.uid, equip.seq, equip.slot)]; !bound {
			continue
		}
		if err := pushWearFrames(h.link, h.roleID, sex, effectiveShowEquipType(h.player), equip, equips, blendsNow); err != nil {
			log.Printf("%s: attire blend re-push slot=%d: %v", h.remote, equip.slot, err)
		}
	}
	sendAttireBlendSavedSysinfo(h.link, h.remote)
	log.Printf("%s: attire blend saved use_type=%d pairs=%d/%d bindings=%d", h.remote, useType, applied, len(pairs), len(blendsNow))
	return true, nil
}

// equipUIDAt 按 692 回传 uid 找穿着件（key 口径 = itemUID("equip")）。
func equipUIDAt(equips []equipRow, roleID role.RoleID, uid string) (equipRow, bool) {
	for _, equip := range equips {
		if itemUID(uint64(roleID), "equip", equip.uid, equip.seq, equip.slot) == uid {
			return equip, true
		}
	}
	return equipRow{}, false
}

// bagUIDKnown 按 692 回传 uid 找背包装备件（key 口径 = itemUID("bag")）。
func bagUIDKnown(bag []bagRow, roleID role.RoleID, uid string) bool {
	for _, item := range bag {
		if itemUID(uint64(roleID), "bag", item.uid, item.seq, item.slot) == uid {
			return true
		}
	}
	return false
}

// sendAttireBlendSavedSysinfo custom_specific_sysinfo（ID=377）保存成功
// 提示：[msg_type=0, tips_type=0, string_id="16596", 金额=0]（server2 从
// 客户端注册表 a1[769] 校准；私服免费固定 0）。best-effort：编码/发送
// 失败仅日志。
func sendAttireBlendSavedSysinfo(link sceneMessageConnection, remote string) {
	frame, err := serverCustomIntMessage(377,
		customInt(0), customInt(0), customString("16596"), customInt(0))
	if err != nil {
		log.Printf("%s: attire blend sysinfo encode: %v", remote, err)
		return
	}
	if err := link.WriteFrame(frame); err != nil {
		log.Printf("%s: attire blend sysinfo send: %v", remote, err)
	}
}

// clearAttireBlendsForRole GM 清空绑定（clear_attire_blends）：落库空表 +
// 内存清空 + 全外观重推（模型回落）+ 绑定穿着槽图标还原（重推纸娃娃行）。
func clearAttireBlendsForRole(link sceneMessageConnection, store *blendDBStore, player *playerActor, roleID role.RoleID, remote string) error {
	blendsNow := player.applyBlends(map[string]string{})
	if err := store.Save(roleID, blendsNow); err != nil {
		log.Printf("%s: persist attire blends clear role=%d: %v", remote, roleID, err)
	}
	for _, equip := range player.sessionEquips() {
		if isValidBlendSlot(equip.slot) {
			if err := pushWearFrames(link, roleID, player.visualSexOf(), effectiveShowEquipType(player), equip, player.sessionEquips(), blendsNow); err != nil {
				log.Printf("%s: attire blend clear re-push slot=%d: %v", remote, equip.slot, err)
			}
		}
	}
	if err := pushAppearanceRefresh(link, player); err != nil {
		log.Printf("%s: attire blend clear refresh: %v", remote, err)
	}
	return nil
}
