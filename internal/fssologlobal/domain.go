package fssologlobal

// DomainTable 复刻官方域表（sologlobal.cpp SetDomain@0x1002ccc0 /
// GetDomainTable@0x1002cdc0）：this+0xE0 起定长数组
// `struct { char name[0x24]; char table[0x24]; } domains[8]`。
//
// 官方语义：index>=8 直接 false；name/table 以 0x24 定长拷入（超长
// 截断）；查名线性扫 8 槽 strcmp，未命中返回 ""（调用方以空串判
// "no domain table"）。main.lua 实配：0=role→nx_role_names、
// 1=guild→nx_guild_names。
type DomainTable struct {
	entries [domainCount]domainEntry
}

// domainCount 官方槽位上限；domainFieldMax 官方单字段定长 0x24
// （有效内容 0x23 字节 + NUL）。
const (
	domainCount    = 8
	domainFieldMax = 0x24
)

type domainEntry struct {
	name  string
	table string
}

// NewDomainTable 建域表（可带初始槽位，main.lua 语义）。
func NewDomainTable(initial map[int][2]string) *DomainTable {
	d := &DomainTable{}
	for idx, nt := range initial {
		d.Set(idx, nt[0], nt[1])
	}
	return d
}

// DefaultDomains 是官方 main.lua 的实配（role/guild 两域）。
func DefaultDomains() *DomainTable {
	return NewDomainTable(map[int][2]string{
		0: {"role", "nx_role_names"},
		1: {"guild", "nx_guild_names"},
	})
}

// Set 官方 SetDomain：index 越界 false；字段超 0x23 字节截断。
func (d *DomainTable) Set(index int, name, table string) bool {
	if index < 0 || index >= domainCount {
		return false
	}
	d.entries[index] = domainEntry{
		name:  truncateField(name),
		table: truncateField(table),
	}
	return true
}

// Lookup 官方 GetDomainTable：线性扫描，未命中返回 ""。
func (d *DomainTable) Lookup(name string) string {
	for i := 0; i < domainCount; i++ {
		if d.entries[i].name != "" && d.entries[i].name == name {
			return d.entries[i].table
		}
	}
	return ""
}

// Entries 呈现用槽位快照（idx → name/table，空槽跳过）。
func (d *DomainTable) Entries() map[int][2]string {
	out := map[int][2]string{}
	for i := 0; i < domainCount; i++ {
		if d.entries[i].name != "" || d.entries[i].table != "" {
			out[i] = [2]string{d.entries[i].name, d.entries[i].table}
		}
	}
	return out
}

// truncateField 官方 0x24 定长拷贝语义（按字节截断到 0x23）。
func truncateField(s string) string {
	if len(s) < domainFieldMax {
		return s
	}
	return s[:domainFieldMax-1]
}
