package main

// 外观域上行处理（Java LoginFlow.handleShowEquipType/handleFashionSuit +
// SkinMessages 骨架，docs/装备外观与风物志挂载.md V4）：
//
//   - C2S 61 CLIENT_CUSTOMMSG_SHOWEQUIP_TYPE：form_new_rp_arm 三 radio
//     custom_showequip_type(0/1/2) → 存 role_appearance → 0x10 整刷。
//   - C2S 19 CLIENT_CUSTOMMSG_FASHION_SUIT：[19, sub, arg]——sub 1 穿上
//     （nx_string configId，须已激活）、2 脱下（EquipType 槽名）、
//     3 购买（会话级扣 1 金，Java 口径不落库）。
//   - C2S 180 CLIENT_CUSTOMMSG_CARD（风物志衣橱）见 skin_wardrobe.go。
//
// 客户端引擎按服务端算好的最终模型属性渲染（RefreshBodyNorEquip），
// 服务端裁决、无新下行 handler。

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

const (
	clientCustomShowEquipType int32 = 61 // CLIENT_CUSTOMMSG_SHOWEQUIP_TYPE
	clientCustomFashionSuit   int32 = 19 // CLIENT_CUSTOMMSG_FASHION_SUIT
	clientCustomWardrobeCard  int32 = 180
)

// appearanceSession 外观域 handler 的会话上下文（itemSession 同款参数包）。
type appearanceSession struct {
	link         sceneMessageConnection
	player       *playerActor
	inventory    *inventoryStore     // 可为 nil（纯内存降级）
	appearance   *appearanceDBStore  // 可为 nil
	fashionStore *fashionSuitDBStore // 可为 nil
	ctx          context.Context
	roleID       role.RoleID
	remote       string
}

// effectiveShowEquipType 有效显示类型：皮肤挂载 ? 2 : 存档偏好（无档 0）
// （Java LoginFlow.effectiveShowEquipType 同序）。
func effectiveShowEquipType(player *playerActor) int {
	player.mu.Lock()
	defer player.mu.Unlock()
	if len(player.skins.mounted) > 0 {
		return showEquipFashion
	}
	if player.showEquipTypeSet {
		return player.showEquipType
	}
	return showEquipGear
}

// ---- playerActor 外观态访问器（锁纪律一律 p.mu） ----

func (p *playerActor) setShowEquipTypePref(showEquipType int) {
	p.mu.Lock()
	p.showEquipType = showEquipType
	p.showEquipTypeSet = true
	p.mu.Unlock()
}

func (p *playerActor) restoreFashionSuits(suits []string) {
	p.mu.Lock()
	p.fashionSuits = suits
	p.mu.Unlock()
}

// addFashionSuit 激活集合并（去重升序），返回合并后快照。
func (p *playerActor) addFashionSuit(suitID string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fashionSuits = mergeFashionSuits(p.fashionSuits, []string{suitID})
	return append([]string(nil), p.fashionSuits...)
}

func (p *playerActor) fashionSuitSnapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.fashionSuits...)
}

func (p *playerActor) goldSnapshot() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gold
}

func (p *playerActor) setGold(value int32) {
	p.mu.Lock()
	p.gold = value
	p.mu.Unlock()
}

// goldUpdate 金币单属性 0x10（CapitalType0@416 Int64，silverUpdate 同构；
// 会话级不落库——Java applyCapital 口径）。
func (p *playerActor) goldUpdate() ([]byte, error) {
	p.mu.Lock()
	gold := p.gold
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 416, Name: "CapitalType0", Value: clientdata.Int64Value(int64(gold))},
	})
}

// ---- 0x10 外观帧推送 ----

// pushAppearanceRefresh 显示类型切换/时装穿脱后的单帧整刷（Java
// pushAppearanceRefresh：appearanceFrame 以有效显示类型裁决）。
func pushAppearanceRefresh(link sceneMessageConnection, player *playerActor) error {
	showEquipType := effectiveShowEquipType(player)
	equips := player.sessionEquips()
	sex := player.visualSexOf()
	frame, err := appearanceFrame(playerObjectID, playerOwnerID, showEquipType, sex, equips)
	if err != nil {
		return err
	}
	if err := link.WriteFrame(frame); err != nil {
		return err
	}
	log.Printf("appearance refresh player=%d-%d show_equip_type=%d equips=%d", playerObjectID, playerOwnerID, showEquipType, len(equips))
	return nil
}

// pushEquipAppearance 登录期 addObject 后的外观补帧（Java 同名门槛）：
// 帽/身份装/时装任一槽占用或显示类型非默认江湖装才发，否则零帧
// （帧序对齐 golden——0x0D 档已带装备合并视觉）。
func pushEquipAppearance(link sceneMessageConnection, player *playerActor, equips []equipRow) error {
	showEquipType := effectiveShowEquipType(player)
	if !appearanceOverlayOccupied(equips, showEquipType) {
		return nil
	}
	frame, err := appearanceFrame(playerObjectID, playerOwnerID, showEquipType, player.visualSexOf(), equips)
	if err != nil {
		return err
	}
	return link.WriteFrame(frame)
}

// ---- C2S 61 显示类型切换 ----

// handleShowEquipTypeCustom 处理 [61, type(0/1/2)]：校验 → 内存+落库 →
// 0x10 整刷。非法值静默拒（零帧）。
func handleShowEquipTypeCustom(h *appearanceSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomShowEquipType {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: ignore show equip type before role selected", h.remote)
		return true, nil
	}
	args := customIntArgs(custom)
	if len(args) < 1 {
		log.Printf("%s: show equip type expects TVarList(61, int type), got %d values", h.remote, len(custom.Values))
		return true, nil
	}
	showEquipType := int(args[0])
	if showEquipType < showEquipGear || showEquipType > showEquipFashion {
		log.Printf("%s: ignore unknown show equip type %d", h.remote, showEquipType)
		return true, nil
	}
	h.player.setShowEquipTypePref(showEquipType)
	if err := h.appearance.Save(h.roleID, showEquipType); err != nil {
		log.Printf("%s: persist show equip type role=%d: %v", h.remote, h.roleID, err)
	}
	if err := pushAppearanceRefresh(h.link, h.player); err != nil {
		return true, err
	}
	log.Printf("%s: show equip type set %d (%s)", h.remote, showEquipType, showEquipTypeName(showEquipType))
	return true, nil
}

func showEquipTypeName(showEquipType int) string {
	switch showEquipType {
	case showEquipIdentity:
		return "身份装"
	case showEquipFashion:
		return "时装"
	default:
		return "江湖装"
	}
}

// ---- C2S 19 时装穿/脱/买 ----

// handleFashionSuitCustom 处理 [19, sub, arg]：sub 1 穿上（configId 串）、
// 2 脱下（EquipType 槽名）、3 购买（扣 1 金）。未知/非法静默拒。
func handleFashionSuitCustom(h *appearanceSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomFashionSuit {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: ignore fashion suit before role selected", h.remote)
		return true, nil
	}
	if len(custom.Values) < 3 {
		log.Printf("%s: fashion suit expects TVarList(19, int sub, string arg), got %d values", h.remote, len(custom.Values))
		return true, nil
	}
	sub, ok := custom.Values[1].exactInt32()
	if !ok {
		log.Printf("%s: fashion suit sub is not an int: %v", h.remote, custom.Values[1])
		return true, nil
	}
	textArg := ""
	if custom.Values[2].Type == 6 {
		textArg = custom.Values[2].Text
	}
	switch sub {
	case 1:
		return true, wearFashionSuit(h, textArg)
	case 2:
		return true, removeFashionSuit(h, textArg)
	case 3:
		return true, buyFashionSuit(h, textArg)
	default:
		log.Printf("%s: fashion suit sub %d not wired", h.remote, sub)
		return true, nil
	}
}

// equipTypeOfConfig 解析配置的 EquipType（game_items 表值 > equipment.ini
// 表值；Java ToolItemCatalog.Info.equipType 对应物）。
func (h *appearanceSession) equipTypeOfConfig(configID string) (string, int32) {
	if h.inventory != nil {
		if entry, ok := h.inventory.lookupGrantable(h.ctx, configID); ok && entry.equipType != "" {
			return entry.equipType, entry.itemType
		}
	}
	if stat := equipmentModels.itemStatic(configID); stat != nil {
		return stat.equipType, 0
	}
	return "", 0
}

// wearFashionSuit sub1 穿上：激活集校验 → 槽位限 16-19 → upsert 落库
// → 穿戴帧组 → 穿戴即切到对应显示类型（槽16→身份装、槽17-19→时装；
// 本服口径，真机校准点见文档 §V4-1）。
func wearFashionSuit(h *appearanceSession, configID string) error {
	if configID == "" {
		log.Printf("%s: fashion wear reject: empty config id", h.remote)
		return nil
	}
	equipType, itemType := h.equipTypeOfConfig(configID)
	equips := h.player.sessionEquips()
	slot := equipSlotFor(equipType, equips)
	if slot < slotSuit || slot > slotFashionShoes {
		log.Printf("%s: fashion wear reject: %s type=%q slot=%d not a fashion/identity piece",
			h.remote, configID, equipType, slot)
		return nil
	}
	if !containsIgnoreCase(h.player.fashionSuitSnapshot(), configID) {
		log.Printf("%s: fashion wear reject: %s not activated", h.remote, configID)
		return nil
	}
	worn, err := h.inventory.upsertEquipSlot(h.ctx, h.roleID, slot, configID, itemType, equipType)
	if err != nil {
		log.Printf("%s: fashion wear persist role=%d slot=%d: %v", h.remote, h.roleID, slot, err)
		return nil
	}
	h.player.replaceEquipSlot(worn)
	equipsNow := h.player.sessionEquips()
	sex := h.player.visualSexOf()
	if err := pushWearFrames(h.link, h.roleID, sex, effectiveShowEquipType(h.player), worn, equipsNow); err != nil {
		log.Printf("%s: fashion wear push frames: %v", h.remote, err)
	}
	// 穿戴即切显示类型（与存档不同才落库）。
	wearType := showEquipFashion
	if slot == slotSuit {
		wearType = showEquipIdentity
	}
	h.player.mu.Lock()
	changed := h.player.showEquipType != wearType || !h.player.showEquipTypeSet
	h.player.showEquipType = wearType
	h.player.showEquipTypeSet = true
	h.player.mu.Unlock()
	if changed {
		if err := h.appearance.Save(h.roleID, wearType); err != nil {
			log.Printf("%s: persist show equip type role=%d: %v", h.remote, h.roleID, err)
		}
		if err := pushAppearanceRefresh(h.link, h.player); err != nil {
			log.Printf("%s: fashion wear refresh: %v", h.remote, err)
		}
	}
	log.Printf("%s: fashion wear %s -> slot=%d show_equip_type=%d", h.remote, configID, slot, wearType)
	return nil
}

// removeFashionSuit sub2 脱下：EquipType 槽名（FashionCoat 等，大小写
// 不敏感）→ 槽位限 16-19；空槽幂等静默；脱下后外观整刷。
func removeFashionSuit(h *appearanceSession, slotName string) error {
	slots := equipSlotsByType[strings.ToLower(strings.TrimSpace(slotName))]
	if len(slots) == 0 || slots[0] < slotSuit || slots[0] > slotFashionShoes {
		log.Printf("%s: fashion remove reject: slot name %q not a fashion slot", h.remote, slotName)
		return nil
	}
	slot := slots[0]
	if configAt(h.player.sessionEquips(), slot) == "" {
		return nil // 空槽幂等（Java 同口径静默）
	}
	if err := h.inventory.deleteEquipSlot(h.ctx, h.roleID, slot); err != nil {
		log.Printf("%s: fashion remove persist role=%d slot=%d: %v", h.remote, h.roleID, slot, err)
		return nil
	}
	h.player.removeEquipSlot(slot)
	if err := h.link.WriteFrame(serverViewRemove(viewDoll, uint16(slot))); err != nil {
		log.Printf("%s: fashion remove clear slot: %v", h.remote, err)
	}
	if mirror := mirrorDollSlot(slot); mirror != slot {
		if err := h.link.WriteFrame(serverViewRemove(viewDoll, uint16(mirror))); err != nil {
			log.Printf("%s: fashion remove clear mirror: %v", h.remote, err)
		}
	}
	if err := pushAppearanceRefresh(h.link, h.player); err != nil {
		log.Printf("%s: fashion remove refresh: %v", h.remote, err)
	}
	log.Printf("%s: fashion remove slot=%d (%s)", h.remote, slot, slotName)
	return nil
}

// buyFashionSuit sub3 购买：未知配置/已激活幂等返回；余额不足日志拒绝
// （客户端无失败回执通道，待真机找拒绝下行——文档 §V4-5）；扣 1 金
// （会话级 CapitalType0，货币落库属货币持久化专项）→ 激活 +
// FashionSuitRec 补行（已推过才补）。
func buyFashionSuit(h *appearanceSession, configID string) error {
	if configID == "" {
		return nil
	}
	if _, itemType := h.equipTypeOfConfig(configID); itemType == 0 && equipmentModels.itemStatic(configID) == nil {
		log.Printf("%s: fashion buy reject: unknown config %s", h.remote, configID)
		return nil
	}
	if containsIgnoreCase(h.player.fashionSuitSnapshot(), configID) {
		return nil // 已激活幂等
	}
	gold := h.player.goldSnapshot()
	if gold < 1 {
		log.Printf("%s: fashion buy reject: %s insufficient gold %d", h.remote, configID, gold)
		return nil
	}
	h.player.setGold(gold - 1)
	if frame, err := h.player.goldUpdate(); err == nil {
		if err := h.link.WriteFrame(frame); err != nil {
			log.Printf("%s: fashion buy gold update: %v", h.remote, err)
		}
	}
	if err := h.fashionStore.Activate(h.roleID, []string{configID}); err != nil {
		log.Printf("%s: fashion buy activate role=%d %s: %v", h.remote, h.roleID, configID, err)
	}
	h.player.addFashionSuit(configID)
	if h.player.fashionRowsPushed() {
		if err := pushFashionSuitRows(h.link, []string{configID}); err != nil {
			log.Printf("%s: fashion buy push row: %v", h.remote, err)
		}
	}
	log.Printf("%s: fashion buy %s (gold %d->%d)", h.remote, configID, gold, gold-1)
	return nil
}

// ---- 选角列表外观（Java LoginFlow.roleListVisual） ----

// skinCardModels 皮肤挂载的部件模型（只取 Cloth/Pants/Shoes，部件名
// 归一化首字母大写；后挂覆盖同部件）——挂载皮肤时选角模型取 card.ini
// 部件模型。
func skinCardModels(mounted []int32, sex byte) map[string]string {
	models := make(map[string]string, 3)
	for _, id := range mounted {
		entry, ok := skinCards.lookup(int(id))
		if !ok {
			continue
		}
		for _, partModel := range entry.models(sex) {
			var key string
			switch strings.ToLower(partModel.part) {
			case "cloth":
				key = partCloth
			case "pants":
				key = partPants
			case "shoes":
				key = partShoes
			default:
				continue
			}
			if partModel.model != "" {
				models[key] = partModel.model
			}
		}
	}
	return models
}

// roleListLookOf 计算选角列表外观态：f6=显示类型存档（无档 golden "1"）；
// f9/f10/f11 按 Java roleListVisual 口径——装备合并为底，身份/时装走
// finalModels（缺件回落装备合并值），皮肤挂载优先取 card.ini 部件模型。
// 全部软依赖：读库失败回 golden 形态。
func roleListLookOf(ctx context.Context, inventory *inventoryStore, appearance appearanceStores, roleID role.RoleID, appearanceValues []string) roleListLook {
	look := defaultRoleListLook(appearanceValues)
	base := look.visual
	var equips []equipRow
	if inventory != nil {
		if loaded, err := inventory.equips(ctx, roleID); err == nil {
			equips = loaded
		}
	}
	merged := mergeVisual(base, equips)
	look.visual = merged

	pref, saved := appearance.appearance.Load(roleID)
	effective := showEquipGear
	if saved {
		look.showEquipType = strconv.Itoa(pref)
		effective = pref
	}
	skins, hasSkins := appearance.skins.Load(roleID)
	if hasSkins && len(skins.mounted) > 0 {
		// 皮肤挂载优先：选角模型取 card.ini 部件模型，缺件回落装备合并值。
		cardModels := skinCardModels(skins.mounted, base.sex)
		if model, ok := cardModels[partCloth]; ok {
			merged.cloth = model
		}
		if model, ok := cardModels[partPants]; ok {
			merged.pants = model
		}
		if model, ok := cardModels[partShoes]; ok {
			merged.shoes = model
		}
		look.visual = merged
		return look
	}
	if effective == showEquipIdentity || effective == showEquipFashion {
		finals := finalModels(effective, base.sex, equips)
		if model := finals[partCloth]; model != "" {
			merged.cloth = model
		}
		if model := finals[partPants]; model != "" {
			merged.pants = model
		}
		if model := finals[partShoes]; model != "" {
			merged.shoes = model
		}
	}
	look.visual = merged
	return look
}

// ---- 登录自动发放 / FashionSuitRec 行 ----

// grantIdentitySuitsAtLogin 按角色 faction 幂等激活身份装清单（旧门派
// shaolin→[school_shaolin]，force_/newschool_ 同名段；世家/帮会/生活/
// 活动组不自动——GM grant_identity_suits 手动）。
func grantIdentitySuitsAtLogin(h *appearanceSession) {
	faction := h.player.factionOf()
	suits := fashionSuits.identitySuitsOf(faction)
	if len(suits) == 0 {
		return
	}
	activated := h.player.fashionSuitSnapshot()
	var missing []string
	for _, suit := range suits {
		if !containsIgnoreCase(activated, suit) {
			missing = append(missing, suit)
		}
	}
	if len(missing) == 0 {
		return
	}
	if err := h.fashionStore.Activate(h.roleID, missing); err != nil {
		log.Printf("%s: grant identity suits role=%d faction=%s: %v", h.remote, h.roleID, faction, err)
	}
	h.player.mu.Lock()
	h.player.fashionSuits = mergeFashionSuits(h.player.fashionSuits, missing)
	granted := len(missing)
	h.player.mu.Unlock()
	log.Printf("%s: granted %d identity suits role=%d faction=%s", h.remote, granted, h.roleID, faction)
}

// pushFashionSuitRows 推 FashionSuitRec 0x11 行（configId + 5 个零值
// 数值列——官方 schema [7,3,3,4,3,3]，数值列语义未校准保持零填充）。
func pushFashionSuitRows(conn sceneMessageConnection, suits []string) error {
	for _, suit := range suits {
		frame, err := serverRecordAddCells(playerObjectID, playerOwnerID, recordFashionSuit, []recordCell{
			recordString(suit),
			recordInt(0), recordInt(0), recordInt64(0), recordInt(0), recordInt(0),
		})
		if err != nil {
			return err
		}
		if err := conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

// ---- playerActor 装备槽内存维护（穿戴/脱下共用） ----

// replaceEquipSlot upsert 一个穿戴槽（同槽替换，保持其余行序）。
func (p *playerActor) replaceEquipSlot(row equipRow) {
	p.mu.Lock()
	defer p.mu.Unlock()
	replaced := false
	for index := range p.equip {
		if p.equip[index].slot == row.slot {
			p.equip[index] = row
			replaced = true
			break
		}
	}
	if !replaced {
		p.equip = append(p.equip, row)
	}
}

// removeEquipSlot 删除一个穿戴槽（不存在时空操作）。
func (p *playerActor) removeEquipSlot(slot int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index := range p.equip {
		if p.equip[index].slot == slot {
			p.equip = append(p.equip[:index], p.equip[index+1:]...)
			return
		}
	}
}

func (p *playerActor) fashionRowsPushed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fashionPushed
}

func (p *playerActor) skinsRowsPushed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skinsPushed
}

func (p *playerActor) setSkinsRowsPushed() {
	p.mu.Lock()
	p.skinsPushed = true
	p.mu.Unlock()
}

// resetSkinRowMirrors 0x0C 重入后调用：客户端记录视图已清空，整刷前
// 不能再对不存在的旧行发 0x12。
func (p *playerActor) resetSkinRowMirrors() {
	p.mu.Lock()
	p.skinRows, p.skinMountedRows = 0, 0
	p.mu.Unlock()
}

func (p *playerActor) setFashionRowsPushed() {
	p.mu.Lock()
	p.fashionPushed = true
	p.mu.Unlock()
}

func (p *playerActor) factionOf() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.faction
}

func (p *playerActor) visualSexOf() byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.visualSex
}
