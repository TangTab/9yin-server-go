package main

// 假人内功 buff 全池（server2 dummy_neigong_buffs.go 移植；资源读取改走
// server1 的 parseINI/openResourceReader——MPQ 三层存储优先、resources/
// 本地文件兜底）。
//
// 数据链（与真玩家装备内功的可见操作 buff 同一条路线）：
//   resources/modern/share/skill/neigong/neigong.ini 的 QName（ng_*，
//   有效 ng_ 前缀约 269 条）↔ resources/modern/share/skill/buff_new.ini 的
//   同名节 [buf_<QName>]（StaticData>0）。
//   基准实证：ng_wd_001→buf_ng_wd_001(5954)、ng_yh_001→buf_ng_yh_001(13073)、
//   ng_th_001→buf_ng_th_001(13434)。
//
// 用户观感要求「全部内功中随机添加一个，每个假人身上有不同的内功 buff」：
// 配置列 neigong_buff 填 QName，三表交叉即全量池。

import (
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// dummyNeiGongBuffEntry 单条内功 buff：QName → 显示名 / buff configID /
// 静态 ID。
type dummyNeiGongBuffEntry struct {
	Name       string
	BuffID     string
	StaticData uint32
}

var (
	dummyNeiGongBuffOnce sync.Once
	dummyNeiGongBuffPool map[string]dummyNeiGongBuffEntry
)

// decodeIniText ini 内混编 GBK/UTF-8：已是合法 UTF-8 原样保留，仅对非法
// 序列按 GBK 解码（server2 同策略）。
func decodeIniText(raw []byte) string {
	if len(raw) == 0 || utf8.Valid(raw) {
		return string(raw)
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), string(raw))
	if err != nil {
		return string(raw)
	}
	return decoded
}

// loadDummyNeiGongBuffPool 进程级懒加载全内功 buff 池。文件缺失/解析失败
// 时回退三条基准内功（ng_wd_001/ng_yh_001/ng_th_001），保证 buff 栏始终
// 有可下发的档位。
func loadDummyNeiGongBuffPool() map[string]dummyNeiGongBuffEntry {
	dummyNeiGongBuffOnce.Do(func() {
		pool := map[string]dummyNeiGongBuffEntry{}
		names := map[string]string{} // QName -> 显示名
		loadNeiGongNames(&names)
		loadBuffNewSections(names, &pool)
		if len(pool) == 0 {
			// 回退：三条基准可见内功（真玩家 learnedNeiGong 实证）。
			pool = map[string]dummyNeiGongBuffEntry{
				"ng_wd_001": {Name: "两仪护心功", BuffID: "buf_ng_wd_001", StaticData: 5954},
				"ng_yh_001": {Name: "明玉功", BuffID: "buf_ng_yh_001", StaticData: 13073},
				"ng_th_001": {Name: "碧波心法", BuffID: "buf_ng_th_001", StaticData: 13434},
			}
		}
		dummyNeiGongBuffPool = pool
	})
	return dummyNeiGongBuffPool
}

// readResourceFile 经 openResourceReader 读资源文件（MPQ 优先）。
func readResourceFile(path string) ([]byte, error) {
	file, err := openResourceReader(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// loadNeiGongNames 解析 neigong.ini：section → (QName, Name)。
func loadNeiGongNames(names *map[string]string) {
	path := defaultSkillINIPath("neigong", "neigong.ini")
	raw, err := readResourceFile(path)
	if err != nil {
		return
	}
	section := ""
	qname := ""
	for _, rawLine := range strings.Split(decodeIniText(raw), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			qname = ""
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 || section == "" {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		switch key {
		case "QName":
			qname = value
		case "Name":
			if strings.HasPrefix(qname, "ng_") && value != "" {
				(*names)[qname] = value
			}
		}
	}
}

// loadBuffNewSections 解析 buff_new.ini：节名 [buf_ng_xxx] ↔ neigong QName
// 交叉，StaticData>0 才入池。
func loadBuffNewSections(names map[string]string, pool *map[string]dummyNeiGongBuffEntry) {
	path := defaultSkillINIPath("buff_new.ini")
	raw, err := readResourceFile(path)
	if err != nil {
		return
	}
	section := ""
	static := uint32(0)
	flush := func() {
		if section == "" || static == 0 {
			return
		}
		buffID := section
		if !strings.HasPrefix(buffID, "buf_") {
			buffID = "buf_" + buffID
		}
		qname := strings.TrimPrefix(section, "buf_")
		name, ok := names[qname]
		if !ok {
			return
		}
		(*pool)[qname] = dummyNeiGongBuffEntry{Name: name, BuffID: buffID, StaticData: static}
	}
	for _, rawLine := range strings.Split(decodeIniText(raw), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			section = line[1 : len(line)-1]
			static = 0
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 || section == "" {
			continue
		}
		if strings.TrimSpace(line[:eq]) == "StaticData" {
			static = parseDummyUint32(strings.TrimSpace(line[eq+1:]))
		}
	}
	flush()
}

// defaultSkillINIPath skill 域资源路径（share/skill，与 loadEquipmentResolver
// 的 share/item 同根约定）。
func defaultSkillINIPath(parts ...string) string {
	return filepath.Join(append([]string{defaultModernShareRoot, "skill"}, parts...)...)
}

// parseDummyUint32 宽松解析 ini 十进制数值（空串/非法 → 0）。
func parseDummyUint32(s string) uint32 {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0
	}
	return uint32(n)
}
