package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf16"
)

func createRoleStrings(msg []byte) []string {
	values := make([]string, 0, 10)
	for offset := 0x49; offset+5 <= len(msg); {
		if msg[offset] != 6 {
			offset++
			continue
		}
		size := int(binary.LittleEndian.Uint32(msg[offset+1:]))
		if size < 1 || size > 4096 || offset+5+size > len(msg) || msg[offset+5+size-1] != 0 {
			offset++
			continue
		}
		values = append(values, string(msg[offset+5:offset+5+size-1]))
		offset += 5 + size
	}
	return values
}

func createRoleName(msg []byte) string {
	if len(msg) < 7 {
		return ""
	}
	units := make([]uint16, 0, 32)
	for offset := 5; offset+1 < len(msg) && offset < 5+72; offset += 2 {
		unit := binary.LittleEndian.Uint16(msg[offset:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	return string(utf16.Decode(units))
}

// roleListLook 选角列表的外观态（Java LoginFlow.roleListVisual 对应物）：
// f6=显示类型存档（无档保持 golden "1"——字节兼容），f9/f10/f11=按显示
// 模式合并后的衣/裤/鞋模型（装备合并 > 身份/时装 finalModels > 皮肤
// card.ini 部件模型，缺件回落装备合并值）。
type roleListLook struct {
	showEquipType string
	visual        roleVisual
}

// defaultRoleListLook 无 DB 外观态时的回退（golden 兼容形态）。
func defaultRoleListLook(appearance []string) roleListLook {
	return roleListLook{showEquipType: "1", visual: resolveRoleVisual(appearance)}
}

// oneRoleLoginSuccess 选角列表 0x04 帧（无外观态的 golden 形态，建角回
// 路与既有测试沿用）。带 DB 外观态的选角走 oneRoleLoginSuccessWithLook。
func oneRoleLoginSuccess(now time.Time, name string, appearance []string, sceneConfig, sceneResource string) []byte {
	return oneRoleLoginSuccessWithLook(now, name, appearance, sceneConfig, sceneResource, defaultRoleListLook(appearance))
}

// oneRoleLoginSuccessWithLook 选角列表 0x04 帧。f23/f24/f25 为选人界面的
// 实力称号/所在场景/剧本标题（Java LoginMessages.oneRoleLoginSuccess 生产
// 重载）：f23=roleListTitle 档位键、f24=sceneDisplayName 真实场景名、
// f25=bookIntroKey 剧本标题键，客户端 form_roles.lua 分别按 desc_ 前缀、
// 原键、原键查表本地化，查不到才回退显示原码。
func oneRoleLoginSuccessWithLook(now time.Time, name string, appearance []string, sceneConfig, sceneResource string, look roleListLook) []byte {
	msg := noRoleLoginSuccess(now)
	binary.LittleEndian.PutUint32(msg[0x25:], 1)
	appendByte := func(v byte) { msg = append(msg, v) }
	appendWord := func(v uint16) {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], v)
		msg = append(msg, b[:]...)
	}
	appendInt := func(v uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], v)
		msg = append(msg, b[:]...)
	}
	appendUCS2 := func(text string) {
		units := utf16.Encode([]rune(text))
		appendInt(uint32((len(units) + 1) * 2))
		for _, unit := range units {
			appendWord(unit)
		}
		appendWord(0)
	}
	appendDouble := func(v float64) {
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
		msg = append(msg, b[:]...)
	}
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	visual := look.visual
	fields := make([]string, 36)
	fields[0] = fmt.Sprint(visual.sex)
	fields[1] = roleFaction(appearance)
	fields[2], fields[3], fields[4] = "0", "0", "1"
	fields[5] = get(2, "")
	fields[6] = look.showEquipType
	fields[7] = visual.hair
	fields[9], fields[10], fields[11] = visual.cloth, visual.pants, visual.shoes
	fields[21], fields[22] = "stand", "127.0.0.1"
	fields[23] = roleListTitle(roleListPowerLevel)
	fields[24] = sceneDisplayName(sceneConfig, sceneResource)
	fields[25] = bookIntroKey(get(0, "book1"))
	fields[31] = "0"
	fields[32] = get(0, "book1")
	fields[33] = visual.hair // 客户端 form_roles.lua [34]=hair，随性别取默认发型
	fields[35] = "0"
	rolePara := strings.Join(fields, ",")
	appendWord(6)
	appendByte(2)
	appendInt(0)
	appendByte(2)
	appendInt(0)
	appendByte(7)
	appendUCS2(name)
	appendByte(7)
	appendUCS2(rolePara)
	appendByte(2)
	appendInt(0)
	appendByte(5)
	appendDouble(0)
	return msg
}

func worldInfo(infoType uint16, text string) []byte {
	units := utf16.Encode([]rune(text))
	msg := make([]byte, 3+(len(units)+1)*2)
	msg[0] = 0x05
	binary.LittleEndian.PutUint16(msg[1:], infoType)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(msg[3+i*2:], unit)
	}
	return msg
}

// serverMenu 编码现代 FxNet2 SERVER_MENU（0x1C）包。其布局
// 直接取自当前客户端的 sub_10617490 接收器：
// object/owner、u16 条目数，随后每个条目为（u8 type、u16 mark、
// u32 含 NUL 的 UTF-16LE 字节数、含 NUL 的 UTF-16LE 内容）。
type serverMenuItem struct {
	Type    byte
	Mark    uint16
	Content string
}

type serverViewSpec struct {
	ID       uint16
	Capacity uint16
}

// serverViewProperty 是 SERVER_CREATE_VIEW 的属性归档条目。
// 其取值编码由 opcode 0x09 中已协商的属性类型决定；
// 在线缆格式上并不自描述。
type serverViewProperty struct {
	index  uint16
	byte1  *byte
	word   *uint16
	text   *string
	int32  *int32
	real32 *float32
}

func viewByte(index uint16, value byte) serverViewProperty {
	return serverViewProperty{index: index, byte1: &value}
}

// viewWord 编码官方线型 2（u16，如扩容锚点 BeginPos/EndPos@2013/2014）。
func viewWord(index uint16, value uint16) serverViewProperty {
	return serverViewProperty{index: index, word: &value}
}

func viewString(index uint16, value string) serverViewProperty {
	return serverViewProperty{index: index, text: &value}
}

func viewInt(index uint16, value int32) serverViewProperty {
	return serverViewProperty{index: index, int32: &value}
}

func viewFloat(index uint16, value float32) serverViewProperty {
	return serverViewProperty{index: index, real32: &value}
}

func serverMenu(objectID, ownerID uint32, items []serverMenuItem) ([]byte, error) {
	if len(items) > math.MaxUint16 {
		return nil, fmt.Errorf("menu item count %d exceeds u16", len(items))
	}
	msg := make([]byte, 11, 11+len(items)*16)
	msg[0] = 0x1C
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	binary.LittleEndian.PutUint16(msg[9:], uint16(len(items)))
	for _, item := range items {
		units := utf16.Encode([]rune(item.Content))
		byteLength := uint64(len(units)+1) * 2
		if byteLength > math.MaxUint32 {
			return nil, fmt.Errorf("menu content exceeds u32 byte length")
		}
		msg = append(msg, item.Type)
		var mark [2]byte
		binary.LittleEndian.PutUint16(mark[:], item.Mark)
		msg = append(msg, mark[:]...)
		var length [4]byte
		binary.LittleEndian.PutUint32(length[:], uint32(byteLength))
		msg = append(msg, length[:]...)
		for _, unit := range units {
			var encoded [2]byte
			binary.LittleEndian.PutUint16(encoded[:], unit)
			msg = append(msg, encoded[:]...)
		}
		msg = append(msg, 0, 0)
	}
	return msg, nil
}

// serverCustomValue 是自定义消息外层 opcode 下的一个类型化
// TVarList 值。当前客户端同时接受 0x1E 及其现代别名 0x27。
type serverCustomValue struct {
	text     *string
	wideText *string
	int32    *int32
	int64    *int64
	float32  *float32
	float64  *float64
	object   *serverCustomObject
}

func customString(value string) serverCustomValue { return serverCustomValue{text: &value} }
func customInt(value int32) serverCustomValue     { return serverCustomValue{int32: &value} }
func customInt64(value int64) serverCustomValue   { return serverCustomValue{int64: &value} }
func customFloat32(value float32) serverCustomValue {
	return serverCustomValue{float32: &value}
}

// customFloat64 是 TVarList tag5（f64）：msg24 c1=71 武技连击段的 30.0/0.0
// 双精度对（Java CustomMessages.ofFloat64 / FightMessages.wujiComboFloat）。
func customFloat64(value float64) serverCustomValue {
	return serverCustomValue{float64: &value}
}

// customWideString 是 TVarList tag7 宽字符串：u32(字节数+2) +
// UTF-16LE 文本 + u16 NUL（F 盘抓包 msg169 双角色名实证，
// Java CustomMessages.ofWideString 同形）。
func customWideString(value string) serverCustomValue {
	return serverCustomValue{wideText: &value}
}

// serverCustomObject 是 TVarList 类型 8 携带的 PERSISTID。其两个
// u32 字就是 SceneObj 标识中所用的同一对 object/owner。
type serverCustomObject struct {
	objectID uint32
	ownerID  uint32
}

func customObject(objectID, ownerID uint32) serverCustomValue {
	return serverCustomValue{object: &serverCustomObject{objectID: objectID, ownerID: ownerID}}
}

// serverCustomStringMessage 保留旧版 0x1E 字符串命令外壳。
func serverCustomStringMessage(messageName string, values ...serverCustomValue) ([]byte, error) {
	return serverCustomStringMessageWithOpcode(0x1E, messageName, values...)
}

// serverModernCustomStringMessage 使用 FxNet2 的 0x27 外壳，同时把
// 命令名保留为 TVarList[0]。现代原生动作接收器仍使用诸如
// "action" 的名字；其注册表槽位并不是线缆 ID。
func serverModernCustomStringMessage(messageName string, values ...serverCustomValue) ([]byte, error) {
	return serverCustomStringMessageWithOpcode(0x27, messageName, values...)
}

func serverCustomStringMessageWithOpcode(opcode byte, messageName string, values ...serverCustomValue) ([]byte, error) {
	if messageName == "" {
		return nil, fmt.Errorf("custom string message name is empty")
	}
	args := make([]serverCustomValue, 0, len(values)+1)
	args = append(args, customString(messageName))
	args = append(args, values...)
	return encodeServerCustomValuesWithOpcode(opcode, args, messageName)
}

// serverCustomIntMessage 只在其激活注册和参数契约都得到核实之后，
// 才编码数值型 C++ GameMessageHandler 路由。
// 它不是 Buff 之类原生对象状态的兜底方案。
func serverCustomIntMessage(messageID int32, values ...serverCustomValue) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x1E, messageID, values...)
}

// serverModernCustomIntMessage 使用现代 FxNet2 自定义消息别名。
// 已在该外壳上核实过的路由保留旧的 0x1E 编码器；新的
// 原生 GameMessageHandler 路由在客户端验证后应使用 0x27。
func serverModernCustomIntMessage(messageID int32, values ...serverCustomValue) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x27, messageID, values...)
}

func serverCustomIntMessageWithOpcode(opcode byte, messageID int32, values ...serverCustomValue) ([]byte, error) {
	if messageID < 1 || messageID >= 765 {
		return nil, fmt.Errorf("custom message id %d outside client range [1,764]", messageID)
	}
	args := make([]serverCustomValue, 0, len(values)+1)
	args = append(args, customInt(messageID))
	args = append(args, values...)
	return encodeServerCustomValuesWithOpcode(opcode, args, fmt.Sprintf("%d", messageID))
}

func encodeServerCustomValues(args []serverCustomValue, label string) ([]byte, error) {
	return encodeServerCustomValuesWithOpcode(0x1E, args, label)
}

func encodeServerCustomValuesWithOpcode(opcode byte, args []serverCustomValue, label string) ([]byte, error) {
	if opcode != 0x1E && opcode != 0x27 {
		return nil, fmt.Errorf("custom message %s has unsupported outer opcode %#x", label, opcode)
	}
	if len(args) > math.MaxUint16 {
		return nil, fmt.Errorf("custom message %s has too many arguments: %d", label, len(args))
	}
	msg := make([]byte, 3, 64)
	msg[0] = opcode
	binary.LittleEndian.PutUint16(msg[1:], uint16(len(args)))
	appendText := func(text string) {
		msg = append(msg, 6)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len(text)+1))
		msg = append(msg, text...)
		msg = append(msg, 0)
	}
	for _, value := range args {
		switch {
		case value.text != nil:
			appendText(*value.text)
		case value.wideText != nil:
			units := utf16.Encode([]rune(*value.wideText))
			data := make([]byte, 0, len(units)*2+2)
			for _, unit := range units {
				data = binary.LittleEndian.AppendUint16(data, unit)
			}
			msg = append(msg, 7)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(data)+2))
			msg = append(msg, data...)
			msg = binary.LittleEndian.AppendUint16(msg, 0)
		case value.int32 != nil:
			msg = append(msg, 2)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(*value.int32))
		case value.int64 != nil:
			msg = append(msg, 3)
			msg = binary.LittleEndian.AppendUint64(msg, uint64(*value.int64))
		case value.float32 != nil:
			msg = append(msg, 4)
			msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(*value.float32))
		case value.float64 != nil:
			msg = append(msg, 5)
			msg = binary.LittleEndian.AppendUint64(msg, math.Float64bits(*value.float64))
		case value.object != nil:
			msg = append(msg, 8)
			msg = binary.LittleEndian.AppendUint32(msg, value.object.objectID)
			msg = binary.LittleEndian.AppendUint32(msg, value.object.ownerID)
		default:
			return nil, fmt.Errorf("custom %s has an empty argument", label)
		}
	}
	return msg, nil
}

// serverCreateView 编码当前 FxNet2 SERVER_CREATE_VIEW（0x15）
// 布局：u16 view id、u16 capacity、u16 属性数，随后是属性归档。
// 空容器视图没有视图级属性，因此其归档
// 恰好就是这七字节头部。
func serverCreateView(spec serverViewSpec) []byte {
	msg, err := serverCreateViewWithProperties(spec, nil)
	if err != nil {
		panic(err)
	}
	return msg
}

// serverCreateViewWithProperties 编码经核实的 0x15 头部，其后是
// 协商属性归档。空容器视图保留由 serverCreateView
// 输出的七字节表示。
func serverCreateViewWithProperties(spec serverViewSpec, properties []serverViewProperty) ([]byte, error) {
	if len(properties) > math.MaxUint16 {
		return nil, fmt.Errorf("view %d property count %d exceeds u16", spec.ID, len(properties))
	}
	msg := make([]byte, 7)
	msg[0] = 0x15
	binary.LittleEndian.PutUint16(msg[1:], spec.ID)
	binary.LittleEndian.PutUint16(msg[3:], spec.Capacity)
	binary.LittleEndian.PutUint16(msg[5:], uint16(len(properties)))
	for _, property := range properties {
		switch {
		case property.byte1 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = append(msg, *property.byte1)
		case property.word != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint16(msg, *property.word)
		case property.text != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*property.text)+1))
			msg = append(msg, (*property.text)...)
			msg = append(msg, 0)
		case property.int32 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(*property.int32))
		case property.real32 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(*property.real32))
		default:
			return nil, fmt.Errorf("view %d property %d has no value", spec.ID, property.index)
		}
	}
	return msg, nil
}

// serverViewAdd 编码 SERVER_VIEW_ADD（0x18）。View 对象使用与其
// 父视图相同的协商属性归档。商店窗体直接读取这些
// 对象；它刻意不加载 share/trade/shop.ini。
func serverViewAdd(viewID, objectIndex uint16, properties []serverViewProperty) ([]byte, error) {
	if len(properties) > math.MaxUint16 {
		return nil, fmt.Errorf("view %d object %d property count %d exceeds u16", viewID, objectIndex, len(properties))
	}
	msg := make([]byte, 7)
	msg[0] = 0x18
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	binary.LittleEndian.PutUint16(msg[3:], objectIndex)
	binary.LittleEndian.PutUint16(msg[5:], uint16(len(properties)))
	for _, property := range properties {
		switch {
		case property.byte1 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = append(msg, *property.byte1)
		case property.word != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint16(msg, *property.word)
		case property.text != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*property.text)+1))
			msg = append(msg, (*property.text)...)
			msg = append(msg, 0)
		case property.int32 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(*property.int32))
		case property.real32 != nil:
			msg = binary.LittleEndian.AppendUint16(msg, property.index)
			msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(*property.real32))
		default:
			return nil, fmt.Errorf("view %d object %d property %d has no value", viewID, objectIndex, property.index)
		}
	}
	return msg, nil
}

func noRoleLoginSuccess(now time.Time) []byte {
	msg := make([]byte, 0x29)
	msg[0] = 0x04
	put := func(offset int, value uint32) { binary.LittleEndian.PutUint32(msg[offset:], value) }
	put(0x01, 1)
	put(0x09, uint32(now.Year()))
	put(0x0D, uint32(now.Month()))
	put(0x11, uint32(now.Day()))
	put(0x15, uint32(now.Hour()))
	put(0x19, uint32(now.Minute()))
	put(0x1D, uint32(now.Second()))
	return msg
}

// heartbeatReply 把客户端 12B 0x18 ping 转成 0x38 回显帧：
// 整帧拷贝仅换 opcode（fork 49840 抓包 326 对逐帧验证；
// 客户端用回显的 tick/seq 做 RTT 配对，缺帧致 HUD 延时封顶）。
func heartbeatReply(ping []byte) []byte {
	reply := append([]byte(nil), ping...)
	if len(reply) > 0 {
		reply[0] = 0x38
	}
	return reply
}

// timeSyncFrame 即 msg294 对钟帧（Java LoginFlow.timeSyncFrame 同源，
// 52905 抓包 1s 一发、计数器恒 +1000）：[294, i32 服务器运行毫秒,
// i64 unix毫秒+FILETIME 纪元偏移, i64 unix 秒, f64 46327.464 常量]。
// cooldown_rec 的 start/end 是绝对 unix 毫秒，客户端靠本帧对齐本机
// 时钟偏移；尾值 46327.464 为官方抓包常量（游戏世界天数口径，语义未定）。
func timeSyncFrame(uptimeMs int32, now time.Time) ([]byte, error) {
	const fileTimeEpochOffsetMs = int64(11644473600000) // 1601-01-01 相对 1970 的毫秒差
	return serverCustomIntMessage(294,
		customInt(uptimeMs),
		customInt64(now.UnixMilli()+fileTimeEpochOffsetMs),
		customInt64(now.Unix()),
		customFloat64(46327.464),
	)
}
