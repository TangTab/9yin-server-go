package fsdumper

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// 列类型（dump_role_dbinfo.xml type= 语义；ParseRoleInfo switch 1..4）。
const (
	colKindProp    = 1 // type="prop"：按 prop 名直取档案值
	colKindRecord  = 2 // type="record"：记录展开（dump_role_record.xml）
	colKindDefault = 3 // type="default"：固定值（prop 属性给字面量）
)

// dbtype→VarType（官方常量表 0x1016D0A8：int→1、varchar→4、nvarchar→5、
// datetime→7、decimal→1、longtext→4）。复刻按 dbtype 决定 SQL 值处理。
var dbTypeQuoted = map[string]bool{
	"varchar": true, "nvarchar": true, "datetime": true, "longtext": true,
}

// RoleColumn 对应 dump_role_dbinfo.xml 的 <property> 行。
type RoleColumn struct {
	Desc      string `xml:"desc,attr"`
	Col       string `xml:"col,attr"`
	DBType    string `xml:"dbtype,attr"`
	Type      string `xml:"type,attr"` // prop / default / record
	Prop      string `xml:"prop,attr"`
	Child     string `xml:"child,attr"`
	CreateSQL string `xml:"createsql,attr"`
}

// RoleTable 对应 <table> 根。
type RoleTable struct {
	Name    string       `xml:"name,attr"`
	Primary string       `xml:"primary,attr"`
	Index   string       `xml:"index,attr"`
	Columns []RoleColumn `xml:"property"`
}

// RoleRecordDef 对应 dump_role_record.xml 的 <record> 行。
type RoleRecordDef struct {
	Desc    string   `xml:"desc,attr"`
	RecName string   `xml:"rec_name,attr"`
	ColNum  int      `xml:"col_num,attr"`
	Cols    []string `xml:",any"` // col_0/col_1/... 顺序属性
}

type roleRecordXML struct {
	XMLName xml.Name        `xml:"Object"`
	Records []roleRecordRow `xml:"record"`
}

type roleRecordRow struct {
	Desc    string `xml:"desc,attr"`
	RecName string `xml:"rec_name,attr"`
	ColNum  int    `xml:"col_num,attr"`
	Col0    string `xml:"col_0,attr"`
	Col1    string `xml:"col_1,attr"`
	Col2    string `xml:"col_2,attr"`
	Col3    string `xml:"col_3,attr"`
	Col4    string `xml:"col_4,attr"`
}

// DumpRole 对应 FsDumpLogic DumpRole 模块（0x6C 字节单例）。
type DumpRole struct {
	table     *RoleTable
	records   map[string]*RoleRecordDef // prop 名（如 capital_rec）→ 定义
	dumpVer   int64                     // m_dumpVersion（this+0x1A0，-1=未定）
	tableName string
	logf      func(string, ...any)
}

// NewDumpRole 解析两份配置 XML（官方 LoadDumpRolePropInfo/RecordInfo，
// 路径 = GetResourcePath()+dumpconfig/）。
func NewDumpRole(dbinfoXML, recordXML []byte, logf func(string, ...any)) (*DumpRole, error) {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	d := &DumpRole{records: map[string]*RoleRecordDef{}, dumpVer: -1, logf: logf}
	var t RoleTable
	if err := xml.Unmarshal([]byte(normalizeXML(dbinfoXML)), &t); err != nil {
		return nil, fmt.Errorf("fsdumper: dump_role_dbinfo.xml: %w", err)
	}
	if t.Name == "" || len(t.Columns) == 0 {
		return nil, fmt.Errorf("fsdumper: dump_role_dbinfo.xml 无表定义")
	}
	d.table = &t
	d.tableName = t.Name

	var rr roleRecordXML
	if err := xml.Unmarshal([]byte(normalizeXML(recordXML)), &rr); err != nil {
		return nil, fmt.Errorf("fsdumper: dump_role_record.xml: %w", err)
	}
	for _, r := range rr.Records {
		cols := []string{r.Col0, r.Col1, r.Col2, r.Col3, r.Col4}[:max(r.ColNum, 0)]
		d.records[r.RecName] = &RoleRecordDef{Desc: r.Desc, RecName: r.RecName, ColNum: r.ColNum, Cols: cols}
	}
	return d, nil
}

func (d *DumpRole) kind(c RoleColumn) int {
	switch c.Type {
	case "prop":
		return colKindProp
	case "record":
		return colKindRecord
	default:
		return colKindDefault
	}
}

// normalizeXML 官方 XML 为 gb2312 声明 + GBK 内容；encoding/xml 只认
// UTF-8，先统一转码并改声明。
func normalizeXML(b []byte) string {
	s := decodeGBK(b)
	s = strings.Replace(s, `encoding="gb2312"`, `encoding="UTF-8"`, 1)
	s = strings.Replace(s, `encoding="GBK"`, `encoding="UTF-8"`, 1)
	return s
}

// EnsureTable 官方 GetTableSchem("output", 表名)→OnTableColNames 的
// 建表路径：列不存在则按 createsql 逐列建表（主键/索引同建）。
func (d *DumpRole) EnsureTable(ctx context.Context, db DB) error {
	var sb strings.Builder
	sb.WriteString("CREATE TABLE IF NOT EXISTS `")
	sb.WriteString(d.table.Name)
	sb.WriteString("` (")
	for i, c := range d.table.Columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strings.TrimSpace(c.CreateSQL))
	}
	if d.table.Primary != "" {
		fmt.Fprintf(&sb, ", PRIMARY KEY (`%s`)", d.table.Primary)
	}
	if d.table.Index != "" {
		fmt.Fprintf(&sb, ", KEY `IX_%s` (`%s`)", d.table.Index, d.table.Index)
	}
	sb.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8")
	if _, err := db.Exec(ctx, sb.String()); err != nil {
		return fmt.Errorf("fsdumper: 建表 %s: %w", d.table.Name, err)
	}
	d.dumpVer = templateVersion(d.table)
	return nil
}

// templateVersion 版本号 = 模板指纹（官方 m_dumpVersion 由模板内容推导；
// Go 以 列数*31+列名哈希 折叠，模板变更即变版）。
func templateVersion(t *RoleTable) int64 {
	var h int64 = 1469598103934665603
	for _, c := range t.Columns {
		for _, ch := range c.Col + c.DBType + c.Type + c.Prop {
			h = (h ^ int64(ch)) * 1099511628211
		}
	}
	if h < 0 {
		h = -h
	}
	return h%900000 + 100000
}

// BuildVersionMsg 官方 SendDumpVersionInfo 组帧：
// ["dump_role_msg", 1, version, table, (kind, col, prop)*]。
func (d *DumpRole) BuildVersionMsg() *textVarList {
	v := newTextVarList()
	v.addString("dump_role_msg").addInt(1).addInt(d.dumpVer).addString(d.table.Name)
	for _, c := range d.table.Columns {
		v.addInt(int64(d.kind(c))).addString(c.Col).addString(c.Prop)
	}
	return v
}

// HandleGameServerMsg 官方 DumpRole::OnRecvGameServerMsg。
// 返回需回发的消息（nil 无）。
func (d *DumpRole) HandleGameServerMsg(k *Kernel, clientID int, msg *textVarList) {
	if msg.Len() < 3 || msg.String(0) != "dump_role_msg" {
		return
	}
	switch msg.Int(1) {
	case 1: // 版本报：ver==本地 → 无事；不符 → 回发模板纠偏
		if msg.Int(2) != d.dumpVer {
			k.SendServerMsg(clientID, d.BuildVersionMsg())
		}
	case 2: // 角色快照：[2]=字段数，其后按模板列序给值
		d.ingestRole(k, clientID, msg)
	}
}

// ingestRole 官方 ParseRoleInfo+InitWriteProp+WriteRecord：
// 字段数不符丢弃并告警；合法则拼 upsert 写 output 库。
func (d *DumpRole) ingestRole(k *Kernel, clientID int, msg *textVarList) {
	fields := int(msg.Int(2))
	if fields != len(d.table.Columns) {
		d.logf("parse role info error, server fields=%d, template fields=%d", fields, len(d.table.Columns))
		return
	}
	cols := d.table.Columns
	vals := msg.vals[3:]
	vals = vals[:min(fields, len(vals))]

	var sb strings.Builder
	var args []any
	sb.WriteString("INSERT INTO `")
	sb.WriteString(d.table.Name)
	sb.WriteString("` (")
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "`%s`", c.Col)
	}
	sb.WriteString(") VALUES (")
	pkIdx := -1
	for i := range cols {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
		if cols[i].Col == d.table.Primary {
			pkIdx = i
		}
	}
	sb.WriteString(")")
	if pkIdx >= 0 {
		var ups []string
		for i, c := range cols {
			if i == pkIdx {
				continue
			}
			ups = append(ups, fmt.Sprintf("`%s`=VALUES(`%s`)", c.Col, c.Col))
		}
		if len(ups) > 0 {
			sb.WriteString(" ON DUPLICATE KEY UPDATE ")
			sb.WriteString(strings.Join(ups, ", "))
		}
	}
	for i := range cols {
		args = append(args, columnValue(cols[i], vals[i]))
	}
	db, err := k.Access("output")
	if err != nil {
		d.logf("%v", err)
		return
	}
	if _, err := db.Exec(context.Background(), sb.String(), args...); err != nil {
		d.logf("write role record failed: %v", err)
		return
	}
	name := "<unnamed>"
	if pkIdx >= 0 && (vals[pkIdx].kind == 's' || vals[pkIdx].kind == 'w') {
		name = vals[pkIdx].s
	}
	d.logf("role [%s] dumped", name)
}

// columnValue 列值方言：数字列直取；带引 dbtype 字符串化（数值亦转
// 文本）；record 列接「行间 ';'、列间 ','」序列化串（复刻决策，见包文档）。
func columnValue(c RoleColumn, v textValue) any {
	if c.Type == "record" {
		return v.s // 已是序列化文本
	}
	if v.kind == 's' || v.kind == 'w' {
		return v.s
	}
	if dbTypeQuoted[c.DBType] {
		if v.kind == 'd' {
			return strconv.FormatFloat(v.f, 'g', -1, 64)
		}
		return strconv.FormatInt(v.n, 10)
	}
	switch v.kind {
	case 'd':
		return v.f
	default:
		return v.n
	}
}
