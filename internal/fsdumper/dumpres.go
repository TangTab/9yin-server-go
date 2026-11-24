package fsdumper

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// dumpResConfig 对应 dumpconfig/dump_config.xml。
type dumpResConfig struct {
	Diction    dictionElem   `xml:"diction"`
	Statics    []staticSQL   `xml:"static>sql"`
	LogDefElem logDefElem    `xml:"logdefine"`
	Move       []dumpResFile `xml:"move>file"`
	LogFile    string        // LogDefine.h 实际路径（装载方注入或默认）
}

type dictionElem struct {
	Path string `xml:"path,attr"`
}

type logDefElem struct {
	Path string `xml:"path,attr"`
}
type staticSQL struct {
	Path string `xml:"path,attr"`
}
type dumpResFile struct {
	BFile bool   `xml:"bfile,attr"`
	Type  int    `xml:"type,attr"`
	Table string `xml:"table,attr"`
	Path  string `xml:"path,attr"`
}

// resMoveType 官方类型码语义（dump_config.xml 注释全表）。
var resMoveType = map[int]string{
	1: "道具", 5: "普通任务", 7: "场景", 8: "技能", 10: "PVP任务",
	14: "NPC", 15: "配置包", 17: "普通怪掉落", 19: "复活点",
}

// resTimerFirst/Interval/巡检 官方节拍常量（0x1007D739 push 1,1 与
// 0x493E0=300000ms 实证）。
const (
	resStartDelay    = time.Second
	resCheckInterval = 300000 * time.Millisecond
)

// DumpRes 对应 FsDumpLogic DumpRes 模块（0x1A8 字节单例）。
type DumpRes struct {
	resDir  string // Res 根（GetResourcePath）
	cfg     *dumpResConfig
	logf    func(string, ...any)
	dumping bool
}

// NewDumpRes 从 Res 根装载 dump_config.xml（官方 LoadConfigFile）。
func NewDumpRes(resDir string, logf func(string, ...any)) (*DumpRes, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	raw, err := res.ReadFile(filepath.Join(resDir, "dumpconfig", "dump_config.xml"))
	if err != nil {
		return nil, fmt.Errorf("fsdumper: dump_config.xml: %w", err)
	}
	var cfg dumpResConfig
	if err := xml.Unmarshal([]byte(normalizeXML(raw)), &cfg); err != nil {
		return nil, fmt.Errorf("fsdumper: dump_config.xml: %w", err)
	}
	if len(cfg.Statics) == 0 && len(cfg.Move) == 0 {
		return nil, fmt.Errorf("fsdumper: dump_config.xml 无 static/move 节点")
	}
	cfg.LogFile = filepath.Join(resDir, "dumpconfig", "DumpHFile", "LogDefine.h")
	return &DumpRes{resDir: resDir, cfg: &cfg, logf: logf}, nil
}

// OnDumperStart 官方 DumpRes::OnDumperStart：AddTimer("TimerStartDumpConfig",
// 1s, 单发)。
func (d *DumpRes) OnDumperStart(k *Kernel) {
	k.AddTimer("TimerStartDumpConfig", d, resStartDelay, 0, d.timerStart)
}

// timerStart 官方 TimerStartDumpConfig 回调（0x1007D460）：
// 1) ExecuteSql("Manage", "DELETE FROM nx_config WHERE (op_type=0 OR ISNULL(op_type))")
// 2) QuerySql nx_config id 基线
// 3) 静态 SQL + LogDefine + move 清单
// 4) AddTimer("TimerCheckDumpConfig", 5min, 巡检)
func (d *DumpRes) timerStart(k *Kernel, _ any, beat int) {
	if beat != 1 { // 单发
		return
	}
	if db, err := k.Access("manage"); err == nil {
		if _, err := db.Exec(context.Background(),
			"DELETE FROM nx_config WHERE (op_type = 0 OR ISNULL(op_type))"); err != nil {
			d.logf("(DumpRes)nx_config 清理失败: %v", err)
		}
	} else {
		d.logf("%v", err)
	}
	d.RunDumpOnce(k)
	k.AddTimer("TimerCheckDumpConfig", d, resCheckInterval, resCheckInterval, d.timerCheck)
	d.logf("All dump over ..............")
}

// timerCheck 官方 TimerCheckDumpConfig：对 <move> 清单做 mtime 巡检。
func (d *DumpRes) timerCheck(k *Kernel, _ any, _ int) { d.RunDumpOnce(k) }

// RunDumpOnce 一轮完整转储：静态 SQL → LogDefine → move 变更检测。
func (d *DumpRes) RunDumpOnce(k *Kernel) {
	if d.dumping {
		return
	}
	d.dumping = true
	defer func() { d.dumping = false }()

	d.runStaticSQLs(k)
	d.dumpLogDefine(k)
	d.dumpMoveFiles(k)
	d.logf("Object dump complete.")
}

// runStaticSQLs 官方 <static> 逐文件执行（;; 双分号语句切分）。
func (d *DumpRes) runStaticSQLs(k *Kernel) {
	db, err := k.Access("manage")
	if err != nil {
		d.logf("%v", err)
		return
	}
	for _, st := range d.cfg.Statics {
		raw, err := res.ReadFile(filepath.Join(d.resDir, filepath.FromSlash(st.Path)))
		if err != nil {
			d.logf("(DumpRes)静态 SQL 读失败 %s: %v", st.Path, err)
			continue
		}
		text := decodeGBK(raw)
		for _, stmt := range strings.Split(text, ";;") {
			stmt = strings.TrimSpace(stripSQLComment(stmt))
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(context.Background(), stmt); err != nil {
				d.logf("(DumpRes)静态 SQL 失败 %s: %v", firstLine(stmt), err)
			}
		}
	}
	d.logf("All sql run over ...........")
}

// dumpLogDefine 官方 DumpLogDefine：解析 LogDefine.h → 重建
// log_enumtype/log_enumvalue。
func (d *DumpRes) dumpLogDefine(k *Kernel) {
	if _, err := os.Stat(d.cfg.LogFile); err != nil {
		d.logf("(DumpRes)logDefine not found: %v", err)
		return
	}
	raw, err := os.ReadFile(d.cfg.LogFile)
	if err != nil {
		d.logf("(DumpRes)LogDefine.h 读失败: %v", err)
		return
	}
	groups, values := ParseLogDefine(decodeGBK(raw))
	db, err := k.Access("manage")
	if err != nil {
		d.logf("%v", err)
		return
	}
	ctx := context.Background()
	d.logf("dump LogDefine start!")
	if _, err := db.Exec(ctx, "DROP TABLE IF EXISTS `log_enumtype`"); err != nil {
		d.logf("(DumpRes)log_enumtype drop 失败: %v", err)
		return
	}
	if _, err := db.Exec(ctx, "CREATE TABLE `log_enumtype` (`type` int(11) NOT NULL DEFAULT '0', "+
		"`desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL, PRIMARY KEY (`type`)"+
		") ENGINE=InnoDB DEFAULT CHARSET=utf8"); err != nil {
		d.logf("(DumpRes)log_enumtype 建表失败: %v", err)
		return
	}
	if _, err := db.Exec(ctx, "DROP TABLE IF EXISTS `log_enumvalue`"); err != nil {
		d.logf("(DumpRes)log_enumvalue drop 失败: %v", err)
		return
	}
	if _, err := db.Exec(ctx, "CREATE TABLE `log_enumvalue` (`type` int(11) NOT NULL, "+
		"`value` varchar(64) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL, "+
		"`text` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL, "+
		"`desc` varchar(128) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT NULL, "+
		"PRIMARY KEY (`value`,`type`)) ENGINE=InnoDB DEFAULT CHARSET=utf8"); err != nil {
		d.logf("(DumpRes)log_enumvalue 建表失败: %v", err)
		return
	}
	for _, g := range groups {
		if _, err := db.Exec(ctx, "INSERT INTO log_enumtype VALUES (?, ?)", g.Type, g.Desc); err != nil {
			d.logf("(DumpRes)log_enumtype 插入失败: %v", err)
		}
	}
	for _, e := range values {
		if _, err := db.Exec(ctx, "INSERT INTO log_enumvalue VALUES (?, ?, ?, ?)",
			e.Type, e.Value, e.Text, e.Desc); err != nil {
			d.logf("(DumpRes)log_enumvalue 插入失败: %v", err)
		}
	}
	d.logf("dump LogDefine success!")
}

// dumpMoveFiles 官方 <move> 清单变更检测：FindFileTime(mtime) vs DB
// op_version，变化才转储；行不存在也转储。
func (d *DumpRes) dumpMoveFiles(k *Kernel) {
	db, err := k.Access("manage")
	if err != nil {
		d.logf("%v", err)
		return
	}
	ctx := context.Background()
	for _, f := range d.cfg.Move {
		full := filepath.Join(d.resDir, filepath.FromSlash(f.Path))
		fi, err := res.Stat(full)
		if err != nil {
			continue // 官方 FindFileTime 失败跳过
		}
		mtime := fi.ModTime().Unix()
		var opVer sql.NullInt64
		err = db.QueryRow(ctx, "SELECT op_version FROM `"+f.Table+"` WHERE id=?", resRowID(f)).Scan(&opVer)
		if err == nil && opVer.Valid && opVer.Int64 == mtime {
			continue // 未变更
		}
		d.dumpOneRes(k, db, f, full, mtime)
	}
}

// resRowID 行主键：目录形态取目录名、文件形态取去扩展名基名（官方
// nx_res_dumpinfo id 列语义：文件标识）。
func resRowID(f dumpResFile) string {
	base := filepath.Base(filepath.FromSlash(f.Path))
	if f.BFile {
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	return base
}

// dumpOneRes 单文件转储（官方 FileAssign+DoDumpResource 语义）：
// `<Object><Property .../>×N</Object>` 形态逐实体一行（id=ID 属性、
// name=Name 属性、property=属性保序重序列化），非 XML/无行文件退化为
// 整档一行（官方 TextGrid/ini 支路的近似）。另写文件级行（property
// 空）作 mtime 基线标记，供巡检跳过未变更文件。
func (d *DumpRes) dumpOneRes(k *Kernel, db DB, f dumpResFile, full string, mtime int64) {
	raw, err := res.ReadFile(full)
	if err != nil {
		d.logf("(DumpRes)读资源失败 %s: %v", f.Path, err)
		return
	}
	text := normalizeXML(raw) // 解码 GBK 并改写 xml 声明（Decoder 拒收 gb2312）
	rows := extractResRows(text)
	fileID := resRowID(f)
	rows = append(rows, resRow{ID: fileID, Name: orDefault(resMoveType[f.Type], fileID), Prop: ""})
	if len(rows) == 1 { // 无实体行 → 整档一行
		rows = []resRow{{ID: fileID, Name: orDefault(resMoveType[f.Type], fileID), Prop: strings.TrimSpace(text)}}
	}
	ctx := context.Background()
	const upsert = "INSERT INTO `%s` (id,name,type,property,status,op_version) VALUES (?,?,?,?,1,?) " +
		"ON DUPLICATE KEY UPDATE name=VALUES(name), type=VALUES(type), property=VALUES(property), op_version=VALUES(op_version)"
	for _, r := range rows {
		if _, err := db.Exec(ctx, fmt.Sprintf(upsert, f.Table), r.ID, r.Name, f.Type, r.Prop, mtime); err != nil {
			d.logf("(DumpRes)资源转储失败 %s [%s]: %v", f.Path, r.ID, err)
			return
		}
	}
	d.logf("dumped %s (type=%d, rows=%d)", f.Path, f.Type, len(rows))
}

type resRow struct {
	ID, Name, Prop string
}

// extractResRows 解析 `<Object><Property .../>…</Object>`：root 直接
// 子元素每行一条，属性序保留（官方 FileAssign 读出同序）。
func extractResRows(src string) []resRow {
	dec := xml.NewDecoder(strings.NewReader(src))
	depth := 0
	var rows []resRow
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 1 {
				rows = append(rows, buildResRow(t))
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return rows
}

func buildResRow(se xml.StartElement) resRow {
	var id, name string
	var b strings.Builder
	b.WriteString("<" + se.Name.Local)
	for _, a := range se.Attr {
		b.WriteString(fmt.Sprintf(` %s=%q`, a.Name.Local, a.Value))
		switch a.Name.Local {
		case "ID", "Id", "ConfigID":
			if id == "" {
				id = a.Value
			}
		case "Name":
			name = a.Value
		}
	}
	b.WriteString("/>")
	if id == "" {
		id = name
	}
	return resRow{ID: id, Name: name, Prop: b.String()}
}

func orDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

func stripSQLComment(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "--") || strings.HasPrefix(t, "//") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ---- LogDefine.h 解析（官方 DumpHFile/TrimWord 同义） ----

// LogEnumType log_enumtype 行。
type LogEnumType struct {
	Type int
	Desc string
}

// LogEnumValue log_enumvalue 行。
type LogEnumValue struct {
	Type  int
	Value string
	Text  string
	Desc  string
}

var (
	reLogGroup = regexp.MustCompile(`^\s*//\$@\s*(\d+)\s*=\s*(.*)$`)
	reLogEntry = regexp.MustCompile(`^\s*(\w+)\s*=\s*(-?\d+)\s*,?\s*//\$\s*(-?\d+)\s*=\s*(.*)$`)
	reQuoted   = regexp.MustCompile(`\|\s*"(.*)"\s*$`)
)

// ParseLogDefine 解析官方 LogDefine.h 方言：
//
//	//$@ <type> = <组描述>              → log_enumtype 行
//	    IDENT = N, //$ N = <文本>|"描述" → log_enumvalue 行
//
// （官方 DumpRes::DumpLogDefine 同款语料，47KB 实物验证。）
func ParseLogDefine(src string) ([]LogEnumType, []LogEnumValue) {
	var groups []LogEnumType
	var values []LogEnumValue
	curType := -1
	for _, line := range strings.Split(src, "\n") {
		if m := reLogGroup.FindStringSubmatch(line); m != nil {
			t, _ := strconv.Atoi(m[1])
			curType = t
			groups = append(groups, LogEnumType{Type: t, Desc: strings.TrimSpace(m[2])})
			continue
		}
		if curType < 0 {
			continue
		}
		m := reLogEntry.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[4])
		desc := ""
		if q := reQuoted.FindStringSubmatch(text); q != nil {
			desc = q[1]
			text = strings.TrimSpace(reQuoted.ReplaceAllString(text, ""))
		}
		values = append(values, LogEnumValue{
			Type: curType, Value: m[3], Text: text, Desc: desc,
		})
	}
	return groups, values
}
