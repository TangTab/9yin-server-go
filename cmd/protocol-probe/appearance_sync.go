package main

// 装备外观挂载三模式裁决（Java AppearanceSync 的 Go 移植，docs/
// 装备外观与风物志挂载.md V2/V4 定稿）：客户端角色外观实为 3 套配置
// （0=江湖装/1=身份装/2=时装），显示模式由玩家属性 ShowEquipType(408,
// byte) 三选一，客户端引擎按服务端算好的最终模型属性渲染
// （RefreshBodyNorEquip）——服务端裁决、无新下行 handler。
//
// 槽位口径（role_equip_items 实测 + form_new_rp_arm.lua:848-881 UI 绑定表
// 钉死）：1 帽/3 衣/4 裤/8 鞋/22 武器；身份装 16=Suit、时装 17=FashionCoat
// /18=FashionHat/19=FashionShoes；新面板镜像 40-43。equipSlotFor/
// mirrorDollSlot 见 item_wear.go（同表）。

import (
	"github.com/local/9yin-go-server/internal/clientdata"
)

const (
	slotHat          int32 = 1
	slotCloth        int32 = 3
	slotPants        int32 = 4
	slotShoes        int32 = 8
	slotSuit         int32 = 16
	slotFashionCoat  int32 = 17
	slotFashionHat   int32 = 18
	slotFashionShoes int32 = 19
	slotWeapon       int32 = 22
)

// 显示类型（玩家属性 ShowEquipType / custom 61）。
const (
	showEquipGear     = 0 // 江湖装（已穿装备外观）
	showEquipIdentity = 1 // 身份装（ItemType 180/181 门派套装，槽 16）
	showEquipFashion  = 2 // 时装（槽 17-19；皮肤挂载亦强制 2）
)

// 部位名（0x10 属性与 finalModels 键）。
const (
	partCloth = "Cloth"
	partPants = "Pants"
	partShoes = "Shoes"
	partHat   = "Hat"
)

// 官方 0x09 表玩家域外观属性索引（official_property_table.txt）。
var (
	propShowEquipType = clientdata.MustPlayerProperty("ShowEquipType").Index // t1 Byte（fork=408）
	propFashionCoat   = clientdata.MustPlayerProperty("FashionCoat").Index   // FashionCoat（fork=1353）
	propFashionHat    = clientdata.MustPlayerProperty("FashionHat").Index    // FashionHat（fork=1354）
	propFashionShoes  = clientdata.MustPlayerProperty("FashionShoes").Index  // FashionShoes（fork=1355）
	propPlayerHat     = clientdata.MustPlayerProperty("Hat").Index           // addObject 档内是发型别名，0x10 才是真帽（fork=391）
	propPlayerCloth   = clientdata.MustPlayerProperty("Cloth").Index         // Cloth（fork=393）
	propPlayerPants   = clientdata.MustPlayerProperty("Pants").Index         // Pants（fork=394）
	propPlayerShoes   = clientdata.MustPlayerProperty("Shoes").Index         // Shoes（fork=395）
)

// wornModels 已穿装备按槽位 → 模型路径（后穿覆盖先穿，同槽只留一件）。
// 只有衣裤鞋帽/身份装/时装改变模型；武器走 WeaponMode 通道（weaponVisualOf）。
func wornModels(sex byte, equips []equipRow) map[int32]string {
	models := make(map[int32]string, len(equips))
	for _, equip := range equips {
		switch equip.slot {
		case slotHat, slotCloth, slotPants, slotShoes,
			slotSuit, slotFashionCoat, slotFashionHat, slotFashionShoes:
		default:
			continue
		}
		if model := equipmentModels.resolve(equip.configID, sex); model != "" {
			models[equip.slot] = model
		}
	}
	return models
}

// configAt 指定槽位当前穿戴的 configId（后穿覆盖先穿；无返回空串）。
func configAt(equips []equipRow, slot int32) string {
	config := ""
	for _, equip := range equips {
		if equip.slot == slot {
			config = equip.configID
		}
	}
	return config
}

// finalModels 三模式最终模型裁决（Cloth/Pants/Shoes/Hat → 模型路径，缺件
// 回落已穿装备）：江湖装=装备槽 1/3/4/8；身份装=槽 16 整套（主 ArtPack=衣，
// Hat/Plants/ShoesArtPack 分件有则覆盖）；时装=槽 17-19（套装式 FashionCoat
// 自带帽/鞋 ArtPack 先行，单件帽/鞋槽再覆盖；时装无裤件——套装衣模型自带
// 下身，Pants 保持装备值）。
func finalModels(showEquipType int, sex byte, equips []equipRow) map[string]string {
	worn := wornModels(sex, equips)
	finals := make(map[string]string, 4)
	putModel(finals, partCloth, worn[slotCloth])
	putModel(finals, partPants, worn[slotPants])
	putModel(finals, partShoes, worn[slotShoes])
	putModel(finals, partHat, worn[slotHat])
	switch showEquipType {
	case showEquipIdentity:
		if suit := configAt(equips, slotSuit); suit != "" {
			putModel(finals, partCloth, equipmentModels.resolve(suit, sex))
			putModel(finals, partHat, equipmentModels.resolveHat(suit, sex))
			putModel(finals, partPants, equipmentModels.resolvePants(suit, sex))
			putModel(finals, partShoes, equipmentModels.resolveShoes(suit, sex))
		}
	case showEquipFashion:
		if coat := configAt(equips, slotFashionCoat); coat != "" {
			putModel(finals, partCloth, equipmentModels.resolve(coat, sex))
			// 套装式时装自带帽/鞋包（如 FASHION_SUIT_G027_gui：HatArtPack/ShoesArtPack）。
			putModel(finals, partHat, equipmentModels.resolveHat(coat, sex))
			putModel(finals, partShoes, equipmentModels.resolveShoes(coat, sex))
		}
		putModel(finals, partHat, resolveSlotModel(equips, slotFashionHat, sex))
		putModel(finals, partShoes, resolveSlotModel(equips, slotFashionShoes, sex))
	}
	return finals
}

func resolveSlotModel(equips []equipRow, slot int32, sex byte) string {
	if config := configAt(equips, slot); config != "" {
		return equipmentModels.resolve(config, sex)
	}
	return ""
}

func putModel(finals map[string]string, part, model string) {
	if model != "" {
		finals[part] = model
	}
}

// mergeVisual 登录外观合并：已穿 Hat/Cloth/Pants/Shoes 装备的模型覆盖
// 创角初始 roleVisual（无装备保持初始装）——重登不再回落「初始屌丝装」。
// 显示模式（身份/时装）的最终模型不进 0x0D 档，登录后由 pushEquipAppearance
// 的 0x10 推送裁决结果。
func mergeVisual(base roleVisual, equips []equipRow) roleVisual {
	models := wornModels(base.sex, equips)
	merged := base
	if model, ok := models[slotCloth]; ok {
		merged.cloth = model
	}
	if model, ok := models[slotPants]; ok {
		merged.pants = model
	}
	if model, ok := models[slotShoes]; ok {
		merged.shoes = model
	}
	return merged
}

// appearanceFrame 外观属性帧（0x10）：显示开关 ShowEquipType + 三模式裁决
// 后的衣/裤/鞋/帽主通道 + 时装镜像通道（FashionCoat/Hat/Shoes 原始模型）
// + 武器三属性（Weapon/ShotWeapon/ActionSet——引擎挂 obj\weapon\<名>，
// 官服实证；空串=收武器显式清除，衣裤鞋空值仍跳过保持客户端当前值）。
// 登录期 addObject 后补帧、穿戴/切换后实时推送共用。
func appearanceFrame(objectID, ownerID uint32, showEquipType int, sex byte, equips []equipRow) ([]byte, error) {
	models := wornModels(sex, equips)
	finals := finalModels(showEquipType, sex, equips)
	weapon := weaponVisualOf(sex, equips)
	weaponActionSet := weapon.actionSet
	if !weapon.armed() {
		// 全空手回初始 1h 动作组（F 抓包方言，原 item_wear.go
		// pushWeaponVisual 校准点——武器/副手清空但 ActionSet 回基线）。
		weaponActionSet = "1h"
	}
	properties := make([]clientdata.IndexedProperty, 0, 10)
	properties = append(properties, clientdata.IndexedProperty{
		Index: propShowEquipType, Name: "ShowEquipType", Value: clientdata.ByteValue(uint8(showEquipType)),
	})
	addOptionalString(&properties, propPlayerCloth, partCloth, finals[partCloth])
	addOptionalString(&properties, propPlayerPants, partPants, finals[partPants])
	addOptionalString(&properties, propPlayerShoes, partShoes, finals[partShoes])
	addOptionalString(&properties, propPlayerHat, partHat, finals[partHat])
	addOptionalString(&properties, propFashionCoat, "FashionCoat", models[slotFashionCoat])
	addOptionalString(&properties, propFashionHat, "FashionHat", models[slotFashionHat])
	addOptionalString(&properties, propFashionShoes, "FashionShoes", models[slotFashionShoes])
	properties = append(properties,
		clientdata.IndexedProperty{Index: propPlayerWeapon, Name: "Weapon", Value: clientdata.StringValue(weapon.weapon)},
		clientdata.IndexedProperty{Index: propPlayerShotWpn, Name: "ShotWeapon", Value: clientdata.StringValue(weapon.shotWeapon)},
		clientdata.IndexedProperty{Index: propPlayerActionSt, Name: "ActionSet", Value: clientdata.StringValue(weaponActionSet)},
	)
	return sceneObjectProperties(objectID, ownerID, 0, properties)
}

// addOptionalString 空模型不下发（无装备部位保持客户端当前值，登录归档
// 已带初始装）；武器三属性走恒写通道（见 appearanceFrame）。
func addOptionalString(properties *[]clientdata.IndexedProperty, index uint16, name, value string) {
	if value == "" {
		return
	}
	*properties = append(*properties, clientdata.IndexedProperty{
		Index: index, Name: name, Value: clientdata.StringValue(value),
	})
}

// appearanceOverlayOccupied 登录补帧门槛（Java pushEquipAppearance 同口径）：
// 帽/身份装/时装任一槽有占用，或显示类型非默认江湖装，才需要 0x10 补帧；
// 否则零帧（帧序对齐 golden——addObject 已带装备合并视觉）。
func appearanceOverlayOccupied(equips []equipRow, showEquipType int) bool {
	if showEquipType != showEquipGear {
		return true
	}
	for _, equip := range equips {
		switch equip.slot {
		case slotHat, slotSuit, slotFashionCoat, slotFashionHat, slotFashionShoes:
			return true
		}
	}
	return false
}
