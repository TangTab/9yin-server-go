package login

import (
	"encoding/binary"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// 建角/选角列表方言（golden 源 = 旧分支 oneRoleLoginSuccess 源码，
// Java LoginMessages 同源）。追加段是 6 值 TVarList：
//
//	[u16 6] int32 0 · int32 0 · widestr 角色名 · widestr rolePara ·
//	int32 0 · double 0
//
// rolePara 为 36 字段逗号串（选人界面展示），关键位：
// f0=性别、f1=门派、f5=脸型、f6=显示类型存档、f7/f33=发型、f9/f10/f11=
// 衣/裤/鞋、f21/f22=stand/回显IP、f23=实力称号键、f24=场景显示名、
// f25=剧本标题键、f32=剧本。

// Role 是账号下的一名角色（当前为内存形态；persist 里程碑接 DB）。
type Role struct {
	Name       string
	Appearance []string // 建角请求的 tag6 串列表（索引语义见 roleVisualOf）
}

// RoleStore 账号→角色的内存仓（每账号一角色，与旧分支无角色/单角色
// 应答形态对齐；多角色列表方言待抓包扩展）。
type RoleStore struct {
	mu    sync.Mutex
	roles map[string]*Role
}

// NewRoleStore 创建空仓。
func NewRoleStore() *RoleStore { return &RoleStore{roles: make(map[string]*Role)} }

// Get 取账号下角色（无则 nil）。
func (s *RoleStore) Get(account string) *Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.roles[account]; r != nil {
		cp := *r
		return &cp
	}
	return nil
}

// Create 建角（覆盖同账号旧角色——单角色形态）。
func (s *RoleStore) Create(account, name string, appearance []string) *Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &Role{Name: name, Appearance: append([]string(nil), appearance...)}
	s.roles[account] = r
	return r
}

// parseCreateRole 解析 op05 建角请求：名字 = [5:] 起的 UTF-16LE（最长
// 72B 至 NUL）；外观串 = [0x49:] 起 tag 6 窄字符串序列（u32 长度含 NUL）。
func parseCreateRole(frame []byte) (name string, appearance []string) {
	if len(frame) < 5 {
		return "", nil
	}
	units := make([]uint16, 0, 32)
	for off := 5; off+1 < len(frame) && off < 5+72; off += 2 {
		unit := binary.LittleEndian.Uint16(frame[off:])
		if unit == 0 {
			break
		}
		units = append(units, unit)
	}
	name = string(utf16.Decode(units))

	values := make([]string, 0, 10)
	for off := 0x49; off+5 <= len(frame); {
		if frame[off] != 6 { // tag 6 = 窄字符串
			off++
			continue
		}
		size := int(binary.LittleEndian.Uint32(frame[off+1:]))
		if size < 1 || size > 4096 || off+5+size > len(frame) || frame[off+5+size-1] != 0 {
			off++
			continue
		}
		values = append(values, string(frame[off+5:off+5+size-1]))
		off += 5 + size
	}
	return name, values
}

// buildCreateRoleFrame 构造 op05 请求（probe/测试用）：名字 + 若干
// tag6 外观串，与 parseCreateRole 对偶。
func buildCreateRoleFrame(name string, appearance []string) []byte {
	units := utf16.Encode([]rune(name))
	frame := make([]byte, 5, 5+len(units)*2+16*len(appearance))
	frame[0] = OpClientRoleNew
	for _, u := range units {
		frame = binary.LittleEndian.AppendUint16(frame, u)
	}
	frame = append(frame, 0, 0) // 名字 NUL
	for len(frame) < 0x49 {
		frame = append(frame, 0)
	}
	for _, s := range appearance {
		frame = append(frame, 6)
		frame = binary.LittleEndian.AppendUint32(frame, uint32(len(s)+1))
		frame = append(frame, s...)
		frame = append(frame, 0)
	}
	return frame
}

// roleVisual 建角外观的展示形态（简化：衣裤鞋不走装备表解析，
// 取请求值或按性别默认裸配置；装备合并留 ini 接入）。
type roleVisual struct {
	sex                byte
	photo, face, hair  string
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
	clothDefault, pantsDefault, shoesDefault := "cloth_b0001", "pants_b0001", "shoes_b0001"
	if sex == 1 {
		photoDefault = `icon\players\girl01_create.png`
		hairDefault = `obj\char\g_hair\g_hair1`
		clothDefault, pantsDefault, shoesDefault = "cloth_g0001", "pants_g0001", "shoes_g0001"
	}
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

// inferSex 性别推断（旧分支线上实证口径）：photo > 发型路径 g_/b_ >
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

// roleParaFields 组装 36 字段逗号串（选人界面展示）。
// 简化项（与旧分支差异，均有注释依据）：f23 新手恒 title001（实力=0）；
// f24 无场景表回退 scene_name_017；f25 未知剧本回退 book 原值。
func roleParaFields(r *Role) []string {
	appearance := r.Appearance
	get := func(index int, fallback string) string {
		if index >= 0 && index < len(appearance) && appearance[index] != "" {
			return appearance[index]
		}
		return fallback
	}
	visual := roleVisualOf(appearance)
	faction := "0"
	if len(appearance) > 8 && appearance[8] != "" {
		faction = appearance[8]
	}
	fields := make([]string, 36)
	fields[0] = strconv.Itoa(int(visual.sex))
	fields[1] = faction
	fields[2], fields[3], fields[4] = "0", "0", "1"
	fields[5] = get(2, "")
	fields[6] = "1" // 显示类型存档（无档 golden "1"）
	fields[7] = visual.hair
	fields[9], fields[10], fields[11] = visual.cloth, visual.pants, visual.shoes
	fields[21], fields[22] = "stand", "127.0.0.1"
	fields[23] = "title001" // 新手实力档（faculty 表接入后按实力阶梯）
	fields[24] = "scene_name_017" // stringname 表接入后按场景查名
	fields[25] = get(0, "book1")  // 剧本标题键：未知回退 book 原值
	fields[31] = "0"
	fields[32] = get(0, "book1")
	fields[33] = visual.hair
	fields[35] = "0"
	return fields
}

// oneRoleLoginSuccessFrame 生成有角色 0x04：41B 基座 + [0x25]u32=1 +
// 6 值 TVarList（0,0,名字,rolePara,0,0.0）。
func oneRoleLoginSuccessFrame(now time.Time, r *Role) []byte {
	msg := noRoleLoginSuccessFrame(now)
	binary.LittleEndian.PutUint32(msg[0x25:], 1)
	body := (&proto.VarList{}).
		AddInt32(0).
		AddInt32(0).
		AddWide(r.Name).
		AddWide(strings.Join(roleParaFields(r), ",")).
		AddInt32(0).
		AddDouble(0)
	return append(msg, body.Encode()...)
}
