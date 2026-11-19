// Package gpkit 是官方 GPKitSvr.dll / GPKitSvr64.dll / GPKitSvrD.dll
// (BGserverCJ 2014 快照,无 PDB;GPKitSvrD 内嵌 e:\GPKBuild\src\server\GPKitSvr
// 构建路径,独立源码树的反外挂 SDK)的 Go 复刻。
//
// 定位(全解见 docs/official/GPKitSvr全解与Go复刻.md):
//
//   - 全套服务端唯一消费者是 release/FsGame.dll(game_world 进程)的
//     PlayerBaseModule:g_pGpk = GPKCreateSvrDynCode() 后按
//     gpk_code(玩家属性,存随机码 idx)/ gpk_msg(挑战与回执字符串)/
//     NpMisCount / NpState 驱动检测,命中经 AddBuffToPlayerForGpk 惩罚。
//   - 服务器侧"动态码"机制:Init("DynCodeBin\Server","DynCodeBin\Client")
//     枚举 Server\*.bin(代码载体,官方为 RWX 机器码,call(1)/call(2)
//     自举出 Encode/Check 两个入口),数据配对规则(动态验证定案)为
//     Server\Client\同名.bin;RandomPick = rand()%count 随机选码;
//     GetData 按 hiword 索引取挑战数据;Call1/Call2 转调载荷入口。
//   - 一次性初始化:gpkconf.ini 七开关(CheckMem/CheckProcName/CheckWindow/
//     CheckModName/CheckCode/CheckDriver/CheckScanMod)+ AuthData.dat
//     0x3E80 字节密钥素材 + 两张 70 项检查函数表(表 A 生成器/表 B 校验器,
//     对应 hack 10..79 共 70 条警告文案)。
//   - x64(2013)新增 CCSAuth::CheckAuthReply:回执包固定 0x80 字节,
//     校验 u16 期望序号(防重放)与 pkt[0:4]==~pkt[4:8] 掩码后分派校验表。
//
// Go 复刻载荷 ABI:官方 .bin 是 x86 机器码载荷(不随包分发,无二进制兼容
// 对象),本包用 PayloadLoader 注入检查闭包;默认 DefaultPayloadLoader
// 识别 "GPK1|u16 genId|u16 checkId" 签名格式并映射到内置 70 检查。
package gpkit

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// CheckFunc 是动态码载荷入口(官方 entry.fn1/fn2)。
// 官方 ABI:调用方 push 两参且不清栈(载荷 ret 8);此处以 Go 闭包表达,
// 参数 a/b 即槽 4/5 转发后的 (arg, codeid 低 16 位)。
type CheckFunc func(a, b uint32) uint32

// PayloadLoader 把代码文件内容解析为 fn1/fn2(官方:VirtualAlloc RWX 后
// call(1)/call(2) 取返回的两个函数指针)。
type PayloadLoader func(code []byte) (fn1, fn2 CheckFunc, err error)

const (
	maxGroups  = 250 // CSvrDynCode 码组槽数(0x3F0/4)
	maxEntries = 256 // CMainDynCode 条目槽数
	maxBinSize = 0x2800
	authMax    = 0x3E80 // AuthData.dat 读取上限(16000)
)

// Conf 是 gpkconf.ini 的七开关(官方全局 0x10022FCC..0x10022FE4)。
type Conf struct {
	CheckMem      bool
	CheckProcName bool
	CheckWindow   bool
	CheckModName  bool
	CheckCode     bool
	CheckDriver   bool
	CheckScanMod  bool
}

// CodeEntry 对应官方 new 0x58 的条目结构。
type CodeEntry struct {
	Name string // 代码文件全路径(官方 strncpy 0x3F)
	Code []byte // 代码文件内容
	Data []byte // 数据文件内容(Server\Client\同名.bin)
	Fn1  CheckFunc
	Fn2  CheckFunc
}

// MainDynCode 对应官方 CMainDynCode(0x414,256 槽)。
type MainDynCode struct {
	entries []*CodeEntry
	seq     uint32 // 调用序号,官方每个接口调用 ++(防重放侧信道)
}

func (m *MainDynCode) count() int { return len(m.entries) }

// Svr 对应官方 CSvrDynCode(0x414),实现 IGPKSvrDynCode 九槽语义。
type Svr struct {
	mu       sync.Mutex
	inited   bool // 官方 +0x04 flag:构造=1,Init 成功=0
	groups   []*MainDynCode
	groupN   int
	cur      *MainDynCode // 官方 +0x3F4 活跃码组
	conf     Conf
	authData []byte
	loader   PayloadLoader
	rng      *rand.Rand
}

// Create 复刻 GPKCreateSvrDynCode():返回新服务对象。
func Create() *Svr {
	return &Svr{
		inited: true, // 官方构造 flag=1
		groups: make([]*MainDynCode, 0, maxGroups),
		loader: DefaultPayloadLoader,
		rng:    rand.New(rand.NewSource(rand.Int63())),
	}
}

// SetLoader 替换载荷解析器(测试/自定义载荷用)。
func (s *Svr) SetLoader(l PayloadLoader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loader = l
}

// LoadAuthData 复刻 0x14A10:读 AuthData.dat ≤0x3E80 字节密钥素材。
func (s *Svr) LoadAuthData(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(b) > authMax {
		b = b[:authMax]
	}
	s.authData = append([]byte(nil), b...)
}

// AuthData 返回密钥素材副本。
func (s *Svr) AuthData() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.authData...)
}

// Conf 返回当前开关。
func (s *Svr) Conf() Conf {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conf
}

// ParseConf 复刻官方自制 INI 解析器(0x15020):'#' 行注释,
// key = value,忽略键值两侧空白。
func ParseConf(ini []byte) Conf {
	var c Conf
	set := func(key string, val string) {
		on := strings.TrimSpace(val) != "" && strings.TrimSpace(val) != "0"
		switch strings.TrimSpace(key) {
		case "CheckMem":
			c.CheckMem = on
		case "CheckProcName":
			c.CheckProcName = on
		case "CheckWindow":
			c.CheckWindow = on
		case "CheckModName":
			c.CheckModName = on
		case "CheckCode":
			c.CheckCode = on
		case "CheckDriver":
			c.CheckDriver = on
		case "CheckScanMod":
			c.CheckScanMod = on
		}
	}
	for _, line := range strings.Split(string(ini), "\n") {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		eq := strings.IndexByte(t, '=')
		if eq < 0 {
			continue
		}
		set(t[:eq], t[eq+1:])
	}
	return c
}

// LoadConf 解析并保存 gpkconf.ini。
func (s *Svr) LoadConf(ini []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conf = ParseConf(ini)
}

// RandomPick 复刻 slot2:rand()%count;无条目返回 -1。
func (s *Svr) RandomPick() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.active()
	if m == nil || m.count() == 0 {
		return -1
	}
	m.seq++
	return s.rng.Intn(m.count())
}

// GetData 复刻 slot3:packed 的 hiword 为条目索引,
// 返回挑战数据(官方返回指针+大小,Go 直接给切片)。
func (s *Svr) GetData(packed uint32) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.active()
	if m == nil {
		return nil, errors.New("gpkit: not initialized")
	}
	m.seq++
	idx := packed >> 16
	if idx >= uint32(m.count()) {
		return nil, fmt.Errorf("gpkit: code index %d out of range (%d)", idx, m.count())
	}
	e := m.entries[idx]
	if e == nil {
		return nil, errors.New("gpkit: empty entry")
	}
	return e.Data, nil
}

// Call1 复刻 slot4:转调 entry.Fn1(b, codeid&0xFFFF)。
// 官方返回 bool;这里带回执值与错误。
func (s *Svr) Call1(a, b uint32, codeid uint32) (uint32, error) {
	return s.call(4, a, b, codeid)
}

// Call2 复刻 slot5:转调 entry.Fn2(b, codeid&0xFFFF)。
// FsGame 用法:Call2(gpk_codeStrPtr, 0xC4, packed) → 输出 196 字节,
// 首字节 0x69 为正常签名。
func (s *Svr) Call2(a, b uint32, codeid uint32) (uint32, error) {
	return s.call(5, a, b, codeid)
}

func (s *Svr) call(slot int, a, b, codeid uint32) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.active()
	if m == nil {
		return 0, errors.New("gpkit: not initialized")
	}
	m.seq++
	idx := codeid >> 16
	if idx >= uint32(m.count()) {
		return 0, fmt.Errorf("gpkit: code index %d out of range (%d)", idx, m.count())
	}
	e := m.entries[idx]
	if e == nil {
		return 0, errors.New("gpkit: empty entry")
	}
	var fn CheckFunc
	if slot == 4 {
		fn = e.Fn1
	} else {
		fn = e.Fn2
	}
	if fn == nil {
		return 0, errors.New("gpkit: entry has no payload entry point")
	}
	return fn(b, codeid&0xFFFF), nil
}

// Release 复刻 slot6:清空全部码组与素材。
func (s *Svr) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups = nil
	s.groupN = 0
	s.cur = nil
	s.authData = nil
	s.inited = true
}

func (s *Svr) active() *MainDynCode {
	if s.cur != nil {
		return s.cur
	}
	if s.groupN == 0 || len(s.groups) == 0 {
		return nil
	}
	return s.groups[0]
}
