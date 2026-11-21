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
	"encoding/binary"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// createReq 建角在途请求（pendingCreate 值；store 回执前暂存外观，
// 0x06 到达后组 rolePara 下行）。
type createReq struct {
	name       string
	appearance []string
}

// 出生场景（官方口径：建角维度 出生地×性别，默认烟雨庄；归纳文档 §2）。
const (
	birthSceneID       = 17 // scene_name_017（rolePara f24 互证）
	birthScenePath     = `ini\scene\born03_YanYuZhuang`
	birthSceneResource = "born03"
)

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
func buildLoadRoleStore(clientID, serial, serverID uint32, account, name string) []byte {	b := make([]byte, 0xBC)
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

// joinAppearance/splitAppearance store save_para1 串形态（逗号拼）。
func joinAppearance(values []string) string { return strings.Join(values, ",") }

func splitAppearance(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// roleVisual 建角外观的展示形态（移植 internal/login/role.go；衣裤鞋
// 不走装备表解析，取请求值或按性别默认裸配置）。
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
	photoDefault := `icon\players\boy01_create.png`
	hairDefault := `obj\char\b_hair\b_hair1`
	if sex == 1 {
		photoDefault = `icon\players\girl01_create.png`
		hairDefault = `obj\char\g_hair\g_hair1`
	}
	// 建角初始装备男女同款三件（官服建角口径：侠士布衣 cloth0002 /
	// 侠士下衣 hc_2400177 / 靴子 cf_composite4）
	clothDefault, pantsDefault, shoesDefault := "cloth0002", "hc_2400177", "cf_composite4"
	return roleVisual{
		sex:    sex,
		photo:  get(1, photoDefault),
		face:   get(2, "333333333333333333333333333333333333333333X38\x01"),
		hair:   get(4, hairDefault),
		cloth:  get(5, clothDefault),
		pants:  get(6, pantsDefault),
		shoes:  get(7, shoesDefault),
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

// roleParaFields 组装 36 字段逗号串（选人界面展示；fork golden）。
// 官方口径：f1 门派空串（School=""，归纳文档 §2——不走 fork 线移花宫
// 出家号 faction 口径）；f25 剧本标题映射客户端本地化键（b-line 同源，
// 未知 book 回退原值）。
func roleParaFields(appearance []string) []string {
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	visual := roleVisualOf(appearance)
	book := get(0, "book1")
	intro := book
	if key, ok := bookIntroKeys[book]; ok {
		intro = key
	}
	fields := make([]string, 36)
	fields[0] = strconv.Itoa(int(visual.sex))
	fields[1] = ""              // School 空串：建角无门派（拜师由 origin 系统驱动）
	fields[2], fields[3], fields[4] = "0", "0", "1"
	fields[5] = get(2, "")
	fields[6] = "1" // 显示类型存档（无档 golden "1"）
	fields[7] = visual.hair
	fields[9], fields[10], fields[11] = visual.cloth, visual.pants, visual.shoes
	fields[21], fields[22] = "stand", "127.0.0.1"
	fields[23] = "title001"       // 新手实力档（faculty 接入后按实力阶梯；b-line title027 为 151 实力私服口径）
	fields[24] = "scene_name_017" // 出生场景显示名（stringname 表接入后按场景查名）
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

// rolesFrame 生成有角色 0x04 ServerPlayerRoles：41B 基座 +
// [0x25]u32=count + 每角色 6 值 TVarList（int32 0 · int32 0 ·
// widestr 名字 · widestr rolePara · int32 0 · double 0）。单角色
// 形态与 internal/login.oneRoleLoginSuccessFrame golden 一致；多角色
// 按 docs/fork-era/id-system-analysis.md 推导（真机校准点）。
func rolesFrame(now time.Time, roles []roleEntry) []byte {
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
	put(0x25, uint32(len(roles)))
	for _, r := range roles {
		body := (&proto.VarList{}).
			AddInt32(0).
			AddInt32(0).
			AddWide(r.name).
			AddWide(strings.Join(roleParaFields(r.appearance), ",")).
			AddInt32(0).
			AddDouble(0)
		msg = append(msg, body.Encode()...)
	}
	return msg
}
