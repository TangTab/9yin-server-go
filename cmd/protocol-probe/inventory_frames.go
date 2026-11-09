package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// 三期物品域帧，移植 Java InventoryMessages（官方方言按名写属性，
// 这里按官方 0x09 表的玩家/视图域末次出现索引直写）：17 视图容器、
// 扩容锚点行、背包行 0x19/0x18 成对（真实槽位，禁用计数器）、
// View1 纸娃娃、武器外观 0x10。

const (
	viewDoll         uint16 = 1
	viewToolPage     uint16 = 2
	viewEquipBag     uint16 = 121
	viewMaterialPage uint16 = 123
	viewTaskPage     uint16 = 125
	bagSlotMax       uint16 = 90
	dollSlotMax      uint16 = 48
)

// pageExtView 扩容页号→扩展视图（页1=道具/页2=装备/页3=材料/页4=任务）。
var pageExtView = map[int32]uint16{1: 3, 2: 122, 3: 124, 4: 126}

// fashionMirror 时装镜像页（背包成对清位防误删用）。
var fashionMirror = map[uint16]uint16{
	2: 174, 3: 175, 121: 176, 122: 177, 123: 178, 124: 179, 125: 180, 126: 181,
}

// bagMainPages 主页顺序（BaseCap=18 的 8 页）；bagExtPages 扩展页。
var (
	bagMainPages = []uint16{121, 2, 123, 125, 176, 174, 178, 180}
	bagExtPages  = []uint16{122, 3, 124, 126, 177, 175, 179, 181}
)

// 官方 0x09 表玩家/视图域（末次出现）属性索引。
// ItemType@1889 与 player_progress.go 的 propItemType 同一索引，直接复用。
const (
	propItemConfigID   uint16 = 1440 // ConfigID String
	propItemLevel      uint16 = 1465 // Level Byte
	propColorLevel     uint16 = 1892 // ColorLevel Int32
	propUniqueID       uint16 = 1893 // UniqueID String
	propAmount         uint16 = 1894 // Amount Int32
	propMaxAmount      uint16 = 1895 // MaxAmount Int32
	propBindStatus     uint16 = 1898 // BindStatus Int32
	propHardiness      uint16 = 1935 // Hardiness Int32
	propMaxHardiness   uint16 = 1936 // MaxHardiness Int32
	propEquipType      uint16 = 1938 // EquipType String
	propMaxBuildCount  uint16 = 1950 // MaxBuildCount Int32
	propCurBuildCount  uint16 = 1951 // CurBuildCount Int32
	propPropBuilt      uint16 = 1954 // PropBuilt String
	propEffectPackID   uint16 = 1959 // EffectPackID Int32
	propRecoverFlag    uint16 = 1971 // RecoverFlag Byte
	propBeginPos       uint16 = 2013 // BeginPos Word
	propEndPos         uint16 = 2014 // EndPos Word
	propBaseCap        uint16 = 2215 // BaseCap Int32
	propPlayerWeapon   uint16 = 396  // Weapon String
	propPlayerShotWpn  uint16 = 398  // ShotWeapon String
	propPlayerActionSt uint16 = 406  // ActionSet String
	propItemPhoto      uint16 = 184  // Photo String（玩家/视图域末次出现）
	// 洗炼域（0x09 表视图域；洗炼界面 QueryProp 校验用，缺发会被
	// form_treasure_reset[_add] 拒收或报 nil 比较）。
	propLockStatus       uint16 = 1899 // LockStatus Int32
	propTersureType      uint16 = 1972 // TersureType Int32（宝物 0..4 五类池）
	propTaoLuValue       uint16 = 1976 // TaoLuValue Int32
	propSecondTaoLuCnt   uint16 = 1980 // SecondTaoLuCounts Int32
)

// serverViewRemove 编码 SERVER_VIEW_REMOVE（0x19）：u16 视图 + u16 槽位。
func serverViewRemove(viewID, objectIndex uint16) []byte {
	msg := make([]byte, 5)
	msg[0] = 0x19
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	binary.LittleEndian.PutUint16(msg[3:], objectIndex)
	return msg
}

// bagViewOf 背包行按 game_items.view_id 分类路由：1→2（道具页）2→121
// （装备页）3→123（材料）4→125（任务）；未知归 2。
func bagViewOf(viewID int32) uint16 {
	switch viewID {
	case 2:
		return viewEquipBag
	case 3:
		return viewMaterialPage
	case 4:
		return viewTaskPage
	default:
		return viewToolPage
	}
}

// bagMirrorOf 背包页视图→时装镜像页（无镜像页返回原视图）。
func bagMirrorOf(view uint16) uint16 {
	if mirror, ok := fashionMirror[view]; ok {
		return mirror
	}
	return view
}

// bagViewCreateFrames 编码 17 视图容器（真实客户端实测布局）：
// 8 主页 cap=90 + BaseCap=18、8 扩展页 cap=2、View1 纸娃娃 cap=48。
func bagViewCreateFrames() [][]byte {
	frames := make([][]byte, 0, 17)
	for _, viewID := range bagMainPages {
		frame, err := serverCreateViewWithProperties(
			serverViewSpec{ID: viewID, Capacity: bagSlotMax},
			[]serverViewProperty{viewInt(propBaseCap, 18)})
		if err != nil {
			panic(err)
		}
		frames = append(frames, frame)
	}
	for _, viewID := range bagExtPages {
		frames = append(frames, serverCreateView(serverViewSpec{ID: viewID, Capacity: 2}))
	}
	frames = append(frames, serverCreateView(serverViewSpec{ID: viewDoll, Capacity: dollSlotMax}))
	return frames
}

// itemBagFrames 编码扩容锚点行：扩展页 0x18，index=锚点位；
// BeginPos=19+Σ前序 boost、EndPos=18+累计（Word 线型）；Hardiness=
// 剩余小时（永久=720）。过期锚点不下发。
func itemBagFrames(packs []bagpackRow, now time.Time) [][]byte {
	frames := make([][]byte, 0, len(packs))
	cumulative := map[int32]int32{}
	for _, pack := range packs {
		if pack.expiresAt != nil && pack.expiresAt.Before(now) {
			continue
		}
		extView, ok := pageExtView[pack.page]
		if !ok {
			continue
		}
		before := cumulative[pack.page]
		begin := 19 + before
		end := 18 + before + pack.boost
		cumulative[pack.page] = before + pack.boost
		frame, err := serverViewAdd(extView, uint16(pack.anchor), []serverViewProperty{
			viewString(propItemConfigID, pack.configID),
			viewWord(propBeginPos, uint16(begin)),
			viewWord(propEndPos, uint16(end)),
			viewInt(propHardiness, anchorHardiness(pack.expiresAt, now)),
		})
		if err != nil {
			continue
		}
		frames = append(frames, frame)
	}
	return frames
}

// anchorHardiness 永久锚点（DB 存 9999-12-31）=720 小时；否则剩余小时。
func anchorHardiness(expiresAt *time.Time, now time.Time) int32 {
	if expiresAt == nil || expiresAt.After(time.Date(2999, 12, 31, 0, 0, 0, 0, time.UTC)) {
		return 720
	}
	hours := int32(expiresAt.Sub(now).Hours())
	if hours < 0 {
		return 0
	}
	return hours
}

// itemUID 物品行下发的统一口径：列值 > 旧行按 seq 派生
// MD5(role:kind:seq) 前 8 字节 hex > 兜底槽位方言 MD5(role:kind:slot)。
func itemUID(roleID uint64, kind, uid string, seq int64, slot int32) string {
	if uid != "" {
		return uid
	}
	if seq > 0 {
		return md5Hex16(fmt.Sprintf("%d:%s:%d", roleID, kind, seq))
	}
	return md5Hex16(fmt.Sprintf("%d:%s:%d", roleID, kind, slot))
}

func md5Hex16(seed string) string {
	digest := md5.Sum([]byte(seed))
	return hex.EncodeToString(digest[:8])
}

// weaponVisual 武器外观三属性（玩家属性 Weapon/ShotWeapon/ActionSet）。
// 官服真机 0x10 帧实证：引擎按 Weapon 裸模型名挂 obj\weapon\<名>\<名>.xmod，
// ActionSet 驱动持械动画组。
type weaponVisual struct {
	weapon     string
	shotWeapon string
	actionSet  string
}

func (w weaponVisual) armed() bool {
	return w.weapon != "" || w.shotWeapon != ""
}

// weaponVisualOf 已穿武器 → Weapon/ShotWeapon 模型名 + ActionSet
// （空手全空串）。主手（Weapon/InnerWeapon 首件）优先于副手。
func weaponVisualOf(sex byte, equips []equipRow) weaponVisual {
	var weaponConfig, shotConfig string
	var weaponItemType, shotItemType int32
	for _, equip := range equips {
		switch equip.equipType {
		case "Weapon", "InnerWeapon":
			if weaponConfig == "" {
				weaponConfig, weaponItemType = equip.configID, equip.itemType
			}
		case "ShotWeapon":
			if shotConfig == "" {
				shotConfig, shotItemType = equip.configID, equip.itemType
			}
		}
	}
	visual := weaponVisual{}
	if weaponConfig != "" {
		visual.weapon = equipmentModels.resolve(weaponConfig, sex)
	}
	if shotConfig != "" {
		visual.shotWeapon = equipmentModels.resolve(shotConfig, sex)
	}
	visual.actionSet = actionSetFor(weaponConfig, weaponItemType)
	if visual.actionSet == "" {
		visual.actionSet = actionSetFor(shotConfig, shotItemType)
	}
	return visual
}

// actionSetFor 表值（itemartstatic [ArtPack].ActionSet）优先；表缺
// （Weapon 101-108/ShotWeapon 110-117 无表值）回退 ItemType-100+"h"
// （数据面实证 101→1h..108→8h）。
func actionSetFor(configID string, itemType int32) string {
	if configID == "" {
		return ""
	}
	if fromTable := equipmentModels.resolveActionSet(configID); fromTable != "" {
		return fromTable
	}
	if itemType >= 101 && itemType <= 108 {
		return fmt.Sprintf("%dh", itemType-100)
	}
	return ""
}

// inventoryFrames 登录背包/纸娃娃整推：纸娃娃行 → 背包成对行 →
// 武器外观。背包行为 remove(槽位预清)+add(真实槽位)+remove(镜像页清位)
// 成对（计数器误删 add 行的防回归形态）。
func inventoryFrames(roleID role.RoleID, sex byte, equips []equipRow, items []bagRow) ([][]byte, error) {
	frames := make([][]byte, 0, len(equips)+len(items)*3+1)
	for _, equip := range equips {
		frame, err := equipmentViewAdd(roleID, equip)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	for _, item := range items {
		if item.slot < 1 || item.slot > int32(bagSlotMax) {
			continue // 槽位 91+ 溢出：DB 保留、显示裁剪
		}
		view := bagViewOf(item.viewID)
		frames = append(frames, serverViewRemove(view, uint16(item.slot)))
		frame, err := bagRowViewAdd(roleID, item)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
		frames = append(frames, serverViewRemove(bagMirrorOf(view), uint16(item.slot)))
	}
	weapon := weaponVisualOf(sex, equips)
	if weapon.armed() {
		frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
			{Index: propPlayerWeapon, Name: "Weapon", Value: clientdata.StringValue(weapon.weapon)},
			{Index: propPlayerShotWpn, Name: "ShotWeapon", Value: clientdata.StringValue(weapon.shotWeapon)},
			{Index: propPlayerActionSt, Name: "ActionSet", Value: clientdata.StringValue(weapon.actionSet)},
		})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// equipmentViewAdd View1 纸娃娃行（index=身体槽位）：ConfigID/数量/
// 类型/EquipType(真值)/耐久/精炼/UniqueID/Photo。
func equipmentViewAdd(roleID role.RoleID, equip equipRow) ([]byte, error) {
	stat := equipmentModels.itemStatic(equip.configID)
	props := itemCommonProps(roleID, equip.configID, equip.itemType, "equip", equip.slot, equip.uid, equip.seq)
	props = append(props, viewString(propEquipType, equip.equipType))
	maxBuild := int32(2)
	if stat != nil {
		maxBuild = stat.maxBuildCount
	}
	props = append(props,
		viewInt(propMaxBuildCount, maxBuild),
		viewInt(propCurBuildCount, maxBuild),
		viewByte(propRecoverFlag, 0),
		viewInt(propColorLevel, statColorLevel(stat)),
		viewInt(propHardiness, equip.hardiness),
		viewInt(propMaxHardiness, equip.maxHardiness),
		viewByte(propItemLevel, statLevel(stat)),
	)
	// 精炼词缀（PropBuilt）与特效包（EffectPackID）互斥；词缀非空下发。
	// 模板包物品（镇派兵器等：equipment.ini @Prop/SkillModifyPackRec）的
	// 词条走物品 record 394/395（官方方言，random_equip_push.go），PropBuilt
	// 让位特效行——旧库残留的包列表 affixes 同步吞掉。
	switch {
	case equip.propBuilt != "" && !randomEquipCatalogOf().hasTemplatePacks(equip.configID):
		props = append(props, viewString(propPropBuilt, equip.propBuilt))
	case stat != nil && stat.effectPackID != 0:
		props = append(props, viewInt(propEffectPackID, stat.effectPackID))
	}
	if stat != nil && stat.photo != "" {
		props = append(props, viewString(propItemPhoto, stat.photo))
	}
	return serverViewAdd(viewDoll, uint16(equip.slot), props)
}

// bagRowViewAdd 背包行：非装备 6 项；装备类补至 14-16 项
// （EquipType 必须真值否则不渲染）。
func bagRowViewAdd(roleID role.RoleID, item bagRow) ([]byte, error) {
	stat := equipmentModels.itemStatic(item.configID)
	props := itemCommonProps(roleID, item.configID, item.itemType, "bag", item.slot, item.uid, item.seq)
	if stat != nil || item.equipType != "" {
		equipType := item.equipType
		if stat != nil && stat.equipType != "" {
			equipType = stat.equipType
		}
		maxBuild := int32(2)
		if stat != nil {
			maxBuild = stat.maxBuildCount
		}
		hardiness, maxHardiness := item.hardiness, item.maxHardiness
		if hardiness <= 0 {
			hardiness = 100
		}
		if maxHardiness <= 0 {
			maxHardiness = 100
		}
		props = append(props,
			viewString(propEquipType, equipType),
			viewInt(propMaxBuildCount, maxBuild),
			viewInt(propCurBuildCount, maxBuild),
			viewByte(propRecoverFlag, 0),
			viewInt(propColorLevel, statColorLevel(stat)),
			viewInt(propHardiness, hardiness),
			viewInt(propMaxHardiness, maxHardiness),
			viewByte(propItemLevel, statLevel(stat)),
		)
		if stat != nil && stat.effectPackID != 0 {
			props = append(props, viewInt(propEffectPackID, stat.effectPackID))
		}
		if stat != nil && stat.photo != "" {
			props = append(props, viewString(propItemPhoto, stat.photo))
		}
		// 宝物行洗炼域属性（form_treasure_reset[_add] 放入校验：
		// TersureType/BindStatus/ColorLevel/LockStatus/SecondTaoLuCounts）。
		// BindStatus 原位覆写为 1（绑定，洗炼硬门槛；通用 6 项已带 0）；
		// LockStatus 基线 0，锁定态由 688 会话经 0x10 isView=1 增量帧翻转。
		if isTreasureEquipType(equipType) {
			tersureType := int32(0)
			if stat != nil {
				tersureType = stat.tersureType
			}
			for i := range props {
				if props[i].index == propBindStatus {
					props[i] = viewInt(propBindStatus, 1)
					break
				}
			}
			props = append(props,
				viewInt(propTersureType, tersureType),
				viewInt(propLockStatus, 0),
				viewInt(propTaoLuValue, 0),
				viewInt(propSecondTaoLuCnt, 0),
			)
		}
	}
	return serverViewAdd(bagViewOf(item.viewID), uint16(item.slot), props)
}

// isTreasureEquipType 宝物类装备（Treasure=火浣台宝物、NewTreasure=翡翠
// 娃娃；洗炼域属性按此判定）。
func isTreasureEquipType(equipType string) bool {
	return equipType == "Treasure" || equipType == "NewTreasure"
}

// itemCommonProps 通用 6 项：ConfigID/Amount=1/MaxAmount=1/ItemType/
// BindStatus=0/UniqueID（实例稳定口径，Java addCommon 同形——数量
// 固定 1，堆叠数量由客户端按 ConfigID 自行聚合显示）。
func itemCommonProps(roleID role.RoleID, configID string, itemType int32, kind string, slot int32, uid string, seq int64) []serverViewProperty {
	return []serverViewProperty{
		viewString(propItemConfigID, configID),
		viewInt(propAmount, 1),
		viewInt(propMaxAmount, 1),
		viewInt(propItemType, itemType),
		viewInt(propBindStatus, 0),
		viewString(propUniqueID, itemUID(uint64(roleID), kind, uid, seq, slot)),
	}
}

func statColorLevel(stat *itemStatic) int32 {
	if stat == nil {
		return 1
	}
	return stat.colorLevel
}

func statLevel(stat *itemStatic) byte {
	if stat == nil {
		return 0
	}
	if stat.level < 0 || stat.level > 255 {
		return 0
	}
	return byte(stat.level)
}
