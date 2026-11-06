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

func oneRoleLoginSuccess(now time.Time, name string, appearance []string) []byte {
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
	visual := resolveRoleVisual(appearance)
	fields := make([]string, 36)
	fields[0] = fmt.Sprint(visual.sex)
	fields[1] = roleFaction(appearance)
	fields[2], fields[3], fields[4] = "0", "0", "1"
	fields[5] = get(2, "")
	fields[6] = "1"
	fields[7] = visual.hair
	fields[9], fields[10], fields[11] = visual.cloth, visual.pants, visual.shoes
	fields[21], fields[22], fields[23], fields[24] = "stand", "127.0.0.1", "title001", "scene_name_017"
	fields[25] = get(0, "book1")
	fields[31] = "0"
	fields[32] = get(0, "book1")
	fields[33] = get(4, `obj\char\b_hair\b_hair1`)
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

// serverMenu encodes the modern FxNet2 SERVER_MENU (0x1C) packet. Its layout
// is taken directly from the current client's sub_10617490 receiver:
// object/owner, a u16 item count, then (u8 type, u16 mark, u32 UTF-16LE byte
// length including NUL, UTF-16LE content including NUL) for each item.
type serverMenuItem struct {
	Type    byte
	Mark    uint16
	Content string
}

type serverViewSpec struct {
	ID       uint16
	Capacity uint16
}

// serverViewProperty is a property archive entry for SERVER_CREATE_VIEW.  The
// value encoding is selected by the property's type already negotiated in
// opcode 0x09; it is not self-describing on the wire.
type serverViewProperty struct {
	index  uint16
	byte1  *byte
	text   *string
	int32  *int32
	real32 *float32
}

func viewByte(index uint16, value byte) serverViewProperty {
	return serverViewProperty{index: index, byte1: &value}
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

// serverCustomValue is one typed TVarList value for a custom-message outer
// opcode. The current client accepts both 0x1E and its modern 0x27 alias.
type serverCustomValue struct {
	text    *string
	int32   *int32
	int64   *int64
	float32 *float32
	object  *serverCustomObject
}

func customString(value string) serverCustomValue { return serverCustomValue{text: &value} }
func customInt(value int32) serverCustomValue     { return serverCustomValue{int32: &value} }
func customInt64(value int64) serverCustomValue   { return serverCustomValue{int64: &value} }
func customFloat32(value float32) serverCustomValue {
	return serverCustomValue{float32: &value}
}

// serverCustomObject is the PERSISTID carried by TVarList type 8. Its two
// u32 words are the same object/owner pair used in SceneObj identifiers.
type serverCustomObject struct {
	objectID uint32
	ownerID  uint32
}

func customObject(objectID, ownerID uint32) serverCustomValue {
	return serverCustomValue{object: &serverCustomObject{objectID: objectID, ownerID: ownerID}}
}

// serverCustomStringMessage retains the legacy 0x1E string-command shell.
func serverCustomStringMessage(messageName string, values ...serverCustomValue) ([]byte, error) {
	return serverCustomStringMessageWithOpcode(0x1E, messageName, values...)
}

// serverModernCustomStringMessage uses the FxNet2 0x27 shell while preserving
// the command name as TVarList[0]. Modern native action receivers still use
// names such as "action"; their registration-table slot is not a wire ID.
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

// serverCustomIntMessage encodes a numeric C++ GameMessageHandler route only
// after its active registration and argument contract have been verified.
// It is not a fallback for native object state such as Buffs.
func serverCustomIntMessage(messageID int32, values ...serverCustomValue) ([]byte, error) {
	return serverCustomIntMessageWithOpcode(0x1E, messageID, values...)
}

// serverModernCustomIntMessage uses the modern FxNet2 custom-message alias.
// Keep the older 0x1E encoder for routes already verified on that shell; new
// native GameMessageHandler routes should use 0x27 when proven by the client.
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
		case value.int32 != nil:
			msg = append(msg, 2)
			msg = binary.LittleEndian.AppendUint32(msg, uint32(*value.int32))
		case value.int64 != nil:
			msg = append(msg, 3)
			msg = binary.LittleEndian.AppendUint64(msg, uint64(*value.int64))
		case value.float32 != nil:
			msg = append(msg, 4)
			msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(*value.float32))
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

// serverCreateView encodes the current FxNet2 SERVER_CREATE_VIEW (0x15)
// layout: u16 view id, u16 capacity, u16 property count, then that archive.
// Empty container views have no view-level properties, so their archive is
// exactly the seven-byte header.
func serverCreateView(spec serverViewSpec) []byte {
	msg, err := serverCreateViewWithProperties(spec, nil)
	if err != nil {
		panic(err)
	}
	return msg
}

// serverCreateViewWithProperties encodes the proven 0x15 header followed by
// a negotiated property archive. Empty container views retain the seven-byte
// representation emitted by serverCreateView.
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

// serverViewAdd encodes SERVER_VIEW_ADD (0x18). View objects use the same
// negotiated property archive as their parent view. The shop form reads these
// objects directly; it deliberately does not load share/trade/shop.ini.
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

func starterBagViews() []serverViewSpec {
	// IDs are defined by the current client share/view_define.lua. The four
	// primary views back the visible bag tabs; the four AddToolBox views back
	// their expansion slots. Capacity is an upper bound in the client view;
	// the GUI itself supplies the fixed grid layout.
	return []serverViewSpec{
		{ID: 121, Capacity: 132}, // VIEWPORT_EQUIP_TOOL: 36 base + 16*6 expansion
		{ID: 2, Capacity: 132},   // VIEWPORT_TOOL
		{ID: 123, Capacity: 132}, // VIEWPORT_MATERIAL_TOOL
		{ID: 125, Capacity: 132}, // VIEWPORT_TASK_TOOL
		{ID: 122, Capacity: 16},  // VIEWPORT_EQUIP_ADDTOOLBOX
		{ID: 3, Capacity: 16},    // VIEWPORT_ADDTOOLBOX
		{ID: 124, Capacity: 16},  // VIEWPORT_MATERIAL_ADDTOOLBOX
		{ID: 126, Capacity: 16},  // VIEWPORT_TASK_ADDTOOLBOX
	}
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
