package fssoloworld

// 建角链数据面（官方通路）：op05 客户端帧解析（双读法）→ store 0x0A
// store_forw_create_t 构造 → 0x06 回执后有角 0x04 组帧。
//
// 数据来源：rolePara 36 字段与有角 0x04 布局移植 internal/login/role.go
// （fork 线真机钉死 golden；官方口径差异：f1 门派恒 "0"——建角无门派，
// docs/official/九阴建角初始数据归纳.md §2）。出生点取官方权威表
// testdata/location.ini BornCJ000003（烟雨庄 born03_YanYuZhuang），
// scene_name_017 与 rolePara f24 回退值互证。

import (
	"math"
	"encoding/binary"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// createReq 建角在途请求（pendingCreate 值；store 回执前暂存外观，
// 0x06 到达后组 rolePara 下行）。
type createReq struct {
	name       string
	appearance []string
}

// 出生场景常量已上收 rolestate（2026-11-26 建号剧本化）：首登场景=
// rolestate.GuideScene（新手副本），出生地=books.ini 剧本表数据驱动
//（DefaultBook/DefaultBirthSection），选人 0x04 的 f24 发出生地节名。

// parseCreateRoleBody 解析 op05 建角帧载荷（msgtype 字节之后的 body）。
// 双读法兼容两种形态，**裸扫 NUL 优先**（fork 线真机实证读法，且天然
// 兼容 archive 形态——名字区定长槽零填充即 NUL 终止）：
//   - fork 真机方言：[4B 保留域跳过][裸 UTF-16LE 扫 NUL，≤72B]；
//   - archive 宽串：[u32 码元数][UTF-16LE]（solo 族 CLoadArchive 方言，
//     兜底——真机 [0:4] 若非码元计数，优先读法会截出半截名）。
//
// 外观串固定从 body[0x48:] 起扫 tag6 窄串序列（fork 帧内 0x49 折算；
// 名字区 34 wchar 槽定长铺垫，两种读法同位）。
func parseCreateRoleBody(body []byte) (name string, appearance []string) {
	if len(body) < 4 {
		return "", nil
	}
	units := make([]uint16, 0, 16) // 读法一：裸 UTF-16 扫 NUL
	for off := 4; off+1 < len(body) && off < 4+72; off += 2 {
		unit := binary.LittleEndian.Uint16(body[off:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	name = string(decodeUTF16(units))
	if name == "" { // 读法二：archive 宽串（[0:4]=码元计数形态）
		if v, n, _ := readArchiveWidestr(body); n > 4 {
			name = v
		}
	}
	if len(body) <= 0x48 {
		return name, nil
	}
	values := make([]string, 0, 10)
	for off := 0x48; off+5 <= len(body); {
		if body[off] != 6 { // tag 6 = 窄字符串
			off++
			continue
		}
		size := int(binary.LittleEndian.Uint32(body[off+1:]))
		if size < 1 || size > 4096 || off+5+size > len(body) || body[off+5+size-1] != 0 {
			off++
			continue
		}
		values = append(values, string(body[off+5:off+5+size-1]))
		off += 5 + size
	}
	return name, values
}

// buildCreateRoleStore 构造 0x0A store_forw_create_t（solo 族 ≥0xD4）：
// {client_id@0, serial@4, server_id@8, account char[33]@0xC,
// name wchar 34槽@0x2D, para wchar@0x71}。名字@0x2D 依网络版分片键
// hashw(data+0x2D) 定位（FsSql与FsStore §1.5）。
func buildCreateRoleStore(clientID, serial, serverID uint32, account, name, para string) []byte {
	b := make([]byte, 0xD4)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	binary.LittleEndian.PutUint32(b[8:], serverID)
	copy(b[0xC:], account)
	units := encodeUTF16([]rune(name))
	for i, u := range units {
		if i >= 34 {
			break
		}
		binary.LittleEndian.PutUint16(b[0x2D+i*2:], u)
	}
	pu := encodeUTF16([]rune(para))
	if need := 0x71 + (len(pu)+1)*2; need > len(b) {
		b = append(b, make([]byte, need-len(b))...)
	}
	for i, u := range pu {
		binary.LittleEndian.PutUint16(b[0x71+i*2:], u)
	}
	return b
}

// buildDeleteRoleStore 构造 0x0B store_forw_delete_t（solo ==0xAC）：
// 与 create 同头（account@0xC/name@0x2D），uid char[33]@0x71。
func buildDeleteRoleStore(clientID, serial, serverID uint32, account, name, uid string) []byte {
	b := make([]byte, 0xAC)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	binary.LittleEndian.PutUint32(b[8:], serverID)
	copy(b[0xC:], account)
	units := encodeUTF16([]rune(name))
	for i, u := range units {
		if i >= 34 {
			break
		}
		binary.LittleEndian.PutUint16(b[0x2D+i*2:], u)
	}
	copy(b[0x71:], uid)
	return b
}

// wideNulNameOf UTF-16LE 扫 NUL 解码（op04 名字区/回执 para）。
func wideNulNameOf(p []byte) string {
	end := 0
	for end+1 < len(p) {
		if p[end] == 0 && p[end+1] == 0 {
			break
		}
		end += 2
	}
	u := make([]uint16, end/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(p[i*2:])
	}
	return string(decodeUTF16(u))
}

// buildLoadRoleStore 构造 0x02 store_forw_load_t（solo ==0xBC）：与
// create 同头（无 uid/para 尾段）。
func buildLoadRoleStore(clientID, serial, serverID uint32, account, name string) []byte {
	b := make([]byte, 0xBC)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	binary.LittleEndian.PutUint32(b[8:], serverID)
	copy(b[0xC:], account)
	units := encodeUTF16([]rune(name))
	for i, u := range units {
		if i >= 34 {
			break
		}
		binary.LittleEndian.PutUint16(b[0x2D+i*2:], u)
	}
	return b
}

// buildSaveStore 构造 0x04 存档包（官方锚点：name 宽串@0x08 0x48 区、
// charge_info@0x50 0x30 区、location 4×double@0x80、定长头 0xA0=官方
// 最小 160）。0xA0 后为 TLV 段序列（Go 两端自洽方言）：
//
//	tag 1 account（LoadRole/merge 定位需要）
//	tag 2 server_id（u32 裸值）
//	tag 3 RST1 json（role_state_json——GM 热改后的权威快照）
//	tag 4 save_data blob（主树无官方 0xD0B blob 生成器，恒空）
//
// 官方 meta 列（total_sec/letter_max/scene_id/…）world 不可知，不编码——
// store 端 merge 时以库内基底保留，避免 0 值冲掉真值。
func buildSaveStore(clientID, serial uint32, account, name string, pos [4]float64, rst1 []byte) []byte {
	b := make([]byte, 0xA0, 0x100)
	binary.LittleEndian.PutUint32(b[0:], clientID)
	binary.LittleEndian.PutUint32(b[4:], serial)
	units := encodeUTF16([]rune(name))
	for i, u := range units {
		if i >= 0x24 { // 0x48 区 = 36 码元
			break
		}
		binary.LittleEndian.PutUint16(b[0x08+i*2:], u)
	}
	// charge_info char[0x30] @0x50 保持零
	for i := 0; i < 4; i++ {
		bits := math.Float64bits(pos[i])
		binary.LittleEndian.PutUint64(b[0x80+i*8:], bits)
	}
	appTLV := func(tag uint32, data []byte) {
		var hdr [8]byte
		binary.LittleEndian.PutUint32(hdr[0:], tag)
		binary.LittleEndian.PutUint32(hdr[4:], uint32(len(data)))
		b = append(b, hdr[:]...)
		b = append(b, data...)
	}
	appTLV(1, []byte(account))
	var sid [4]byte // server_id 由 store 端按连接会话补（此处 0）
	appTLV(2, sid[:])
	appTLV(3, rst1)
	appTLV(4, nil) // save_data：主树恒空（空档=官方已有语义）
	return b
}

// joinAppearance/splitAppearance store save_para1 串形态（逗号拼）。
func joinAppearance(values []string) string { return strings.Join(values, ",") }

func splitAppearance(s string) []string {
	if s == "" {
		return nil
	}
	// 槽位归位：捏脸串可能含逗号（para join/split 碎片化，2026-11-27 真机
	// 「没头发」根因），未知段并回捏脸槽。
	return rolestate.NormalizeAppearance(strings.Split(s, ","))
}

// roleVisual 建角外观的展示形态（移植 internal/login/role.go）。衣裤鞋
// 为**初始布衣三件模型路径**（2433 方言定版 docs/official/
// 初始号无装备身体模型定版.md——无装备≠空槽，fork 线装备码 cloth0002 族
// 是旧分支方言，captures/local 分支客户端不认，模型缺失）。
type roleVisual struct {
	sex                 byte
	photo, face, hair   string
	cloth, pants, shoes string
}

func roleVisualOf(appearance []string) roleVisual {
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	sex := inferSex(appearance)
	photoDefault := `icon\players\boy03_create.png`
	hairDefault := `obj\char\b_hair\b_hair4`
	clothDefault, pantsDefault, shoesDefault := `obj\char\b_jianghu001\b_cloth001`,
		`obj\char\b_jianghu003\b_pants003`, `obj\char\b_jianghu003\b_shoes003`
	if sex == 1 {
		photoDefault = `icon\players\girl01_create.png`
		hairDefault = `obj\char\g_hair\g_hair4`
		clothDefault, pantsDefault, shoesDefault = `obj\char\g_jianghu001\g_cloth001`,
			`obj\char\g_jianghu003\g_pants003`, `obj\char\g_jianghu003\g_shoes003`
	}
	return roleVisual{
		sex:   sex,
		photo: get(1, photoDefault),
		face:  get(2, "e/330$33,'51\x1a4>=3333333333?5=<==\x19=33!F!1'6F3\t\x01"),
		hair:  get(4, hairDefault),
		// 布衣三件固定（定版：固定数据，不走档内槽）——52626 会话
		// rolePara f9/f10/f11 实证值。
		cloth: clothDefault,
		pants: pantsDefault,
		shoes: shoesDefault,
	}
}

// inferSex 性别推断（fork 线实证口径）：photo > 发型路径 g_/b_ >
// 衣裤鞋 _g/_b；全无证据 → 0（男）。
func inferSex(appearance []string) byte {
	value := func(index int) string {
		if index >= 0 && index < len(appearance) {
			return appearance[index]
		}
		return ""
	}
	if byPhoto := photoSex(value(1)); byPhoto >= 0 {
		return byte(byPhoto)
	}
	if byHair := pathSex(value(4)); byHair >= 0 {
		return byte(byHair)
	}
	for index := 5; index <= 7; index++ {
		part := value(index)
		byPart := pathSex(part)
		if byPart < 0 && (strings.HasPrefix(part, "cloth_") || strings.HasPrefix(part, "pants_") || strings.HasPrefix(part, "shoes_")) {
			switch {
			case strings.Contains(part, "_g"):
				byPart = 1
			case strings.Contains(part, "_b"):
				byPart = 0
			}
		}
		if byPart >= 0 {
			return byte(byPart)
		}
	}
	return 0
}

func photoSex(photo string) int {
	lower := strings.ToLower(photo)
	switch {
	case strings.Contains(lower, "girl"):
		return 1
	case strings.Contains(lower, "boy"):
		return 0
	}
	return -1
}

func pathSex(path string) int {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "/g_"), strings.Contains(lower, `\g_`):
		return 1
	case strings.Contains(lower, "/b_"), strings.Contains(lower, `\b_`):
		return 0
	}
	return -1
}

// bookIntroKeys 五书剧本标题键（b-line scene_names.go bookIntroCodeByNumber
// 同源：qdz 千灯镇/yyz 烟雨庄/cd 成都/jm 剑门/wmp 悟明坡）。
var bookIntroKeys = map[string]string{
	"book1": "introduction_qdz01_title",
	"book2": "introduction_yyz01_title",
	"book4": "introduction_cd01_title",
	"book5": "introduction_jm01_title",
	"book8": "introduction_wmp01_title",
}

// birthSceneDisplayName/sceneDisplayName 已随出生地节名口径（建号.pcapng
// f24=ChushengCJ000002）移除：选人界面不再发场景中文直接量。

// roleParaFields 组装 36 字段逗号串（选人界面展示；52626 会话真机
// rolePara 36 字段对拍：f6=0、f9/f10/f11=布衣三件模型路径、f24 位为
// 场景/出生地段）。官方口径（2026-11-26 建号.pcapng 0x04 实证）：出生地
// 段发**节名**（ChushengCJ000002）非场景中文名、剧本段发 book key
//（book2）。f1 门派空串（School=""，归纳文档 §2）；f25 剧本标题映射客户
// 端本地化键（b-line 同源，未知 book 回退原值）。位次沿用 52626 抓包
//（fork/official 方言位次差异待真机专项对拍，勿轻改）。
func roleParaFields(appearance []string, book, birthSection string) []string {
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	visual := roleVisualOf(appearance)
	if book == "" {
		book = get(0, rolestate.DefaultBook)
	}
	intro := book
	if key, ok := bookIntroKeys[book]; ok {
		intro = key
	}
	if birthSection == "" {
		birthSection = rolestate.DefaultBirthSection
	}
	fields := make([]string, 36)
	fields[0] = strconv.Itoa(int(visual.sex))
	fields[1] = "" // School 空串：建角无门派（拜师由 origin 系统驱动）
	fields[2], fields[3], fields[4] = "0", "0", "1"
	fields[5] = get(2, "")
	fields[6] = "0" // 显示类型存档（52626 真机 f6="0"；旧 fork 无档 golden "1" 为异分支方言）
	fields[7] = visual.hair
	fields[9], fields[10], fields[11] = visual.cloth, visual.pants, visual.shoes
	fields[21], fields[22] = "stand", "127.0.0.1"
	fields[23] = "title001"      // 新手实力档（faculty 接入后按实力阶梯；b-line title027 为 151 实力私服口径）
	fields[24] = birthSection    // 出生地节名（官服口径 ChushengCJ000002；建号态角色恒在出生地语义下）
	fields[25] = intro
	fields[31] = "0"
	fields[32] = book
	fields[33] = visual.hair
	fields[35] = "0"
	return fields
}

// roleEntry 有角 0x04 的角色行。
type roleEntry struct {
	name       string
	appearance []string
}

// rolesFrame 生成 0x04 ServerPlayerRoles（有角/无角同形）：41B 基座
// （serial@0x21=登录会话 serial——官服抓包实证与 0x0B 的 [1:5] 同值，
// 如男号会话两者均为 0x94B）+ [0x25]u32=count + 每角色 6 值 TVarList（int32 0 · int32 0 ·
// widestr 名字 · widestr rolePara · int32 0 · double 0）。单角色
// 形态与 internal/login.oneRoleLoginSuccessFrame golden 一致；多角色
// 按 docs/fork-era/id-system-analysis.md 推导（真机校准点）。
func rolesFrame(serial uint32, now time.Time, roles []roleEntry) []byte {
	msg := make([]byte, 0x29)
	msg[0] = 0x04
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(msg[off:], v) }
	put(0x01, 1)
	put(0x09, uint32(now.Year()))
	put(0x0D, uint32(now.Month()))
	put(0x11, uint32(now.Day()))
	put(0x15, uint32(now.Hour()))
	put(0x19, uint32(now.Minute()))
	put(0x1D, uint32(now.Second()))
	put(0x21, serial)
	put(0x25, uint32(len(roles)))
	for _, r := range roles {
		// 剧本/出生地从存档外观串推导（0x04 是未进场名册帧，档未载入；
		// 建号态外观串含 book key，官服 0x04 实证 f24=出生地节名）。
		book := rolestate.InferBookFromPara(strings.Join(r.appearance, ","))
		body := (&proto.VarList{}).
			AddInt32(0).
			AddInt32(0).
			AddWide(r.name).
			AddWide(strings.Join(roleParaFields(r.appearance, book, rolestate.BirthSectionOf(book)), ",")).
			AddInt32(0).
			AddDouble(0)
		msg = append(msg, body.Encode()...)
	}
	return msg
}

// sendRolesFrame 下行 0x04：serial 取会话连接键（Conn2）。
func (s *Server) sendRolesFrame(pl *Player, roles []roleEntry) {
	s.sendToClient(pl, rolesFrame(pl.Conn2, time.Now(), roles))
}
