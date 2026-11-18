package fskit

// IniDocument 复刻官方 fskit/inidocument.cpp 实体：包装一个 CIniFile。
// Init(args[0]=文件名) 只构造引擎，不自动 Load（官方 0x10053b30）。
type IniDocument struct {
	ini *IniFile
}

// NewIniDocument 对齐官方 IniDocument::Init(IVarList&)：args[0] 为文件名。
func NewIniDocument(fileName string) *IniDocument {
	return &IniDocument{ini: NewIniFile(fileName)}
}

// EntityName 官方实体类型名（EntCreator::GetEntityTypeName）。
const IniDocumentTypeName = "IniDocument"

func (d *IniDocument) Inner() *IniFile { return d.ini }

// SetFileName / GetFileName 属性 FileName（get/set 直通引擎）。
func (d *IniDocument) SetFileName(name string) { d.ini.SetFileName(name) }
func (d *IniDocument) GetFileName() string     { return d.ini.GetFileName() }

// ---- 21 个 mid 方法（官方逐个校验参数个数，Go 侧以强类型签名等价收敛）----

func (d *IniDocument) LoadFromFile() bool { return d.ini.LoadFromFile() }
func (d *IniDocument) SaveToFile() bool   { return d.ini.SaveToFile() }

func (d *IniDocument) AddSection(s string) bool        { return d.ini.AddSection(s) }
func (d *IniDocument) FindSection(s string) bool       { return d.ini.FindSection(s) }
func (d *IniDocument) DeleteSection(s string) bool     { return d.ini.DeleteSection(s) }
func (d *IniDocument) GetSectionCount() int            { return d.ini.GetSectionCount() }
func (d *IniDocument) GetSectionList() []string        { return d.ini.GetSectionList() }
func (d *IniDocument) FindItem(s, k string) bool       { return d.ini.FindItem(s, k) }
func (d *IniDocument) DeleteItem(s, k string) bool     { return d.ini.DeleteItem(s, k) }
func (d *IniDocument) GetItemCount(s string) int       { return d.ini.GetItemCount(s) }
func (d *IniDocument) GetItemList(s string) []string   { return d.ini.GetItemList(s) }
func (d *IniDocument) GetItemValueList(s string) []string {
	return d.ini.GetItemValueList(s)
}

func (d *IniDocument) ReadInteger(s, k string, def int) int { return d.ini.ReadInteger(s, k, def) }
func (d *IniDocument) ReadFloat(s, k string, def float64) float64 {
	return d.ini.ReadFloat(s, k, def)
}
func (d *IniDocument) ReadString(s, k, def string) string {
	return d.ini.ReadString(s, k, def)
}
func (d *IniDocument) WriteInteger(s, k string, v int) bool {
	return d.ini.WriteInteger(s, k, v)
}
func (d *IniDocument) WriteFloat(s, k string, v float64) bool {
	return d.ini.WriteFloat(s, k, v)
}
func (d *IniDocument) WriteString(s, k, v string) bool { return d.ini.WriteString(s, k, v) }
func (d *IniDocument) AddInteger(s, k string, v int) bool {
	return d.ini.AddInteger(s, k, v)
}
func (d *IniDocument) AddFloat(s, k string, v float64) bool { return d.ini.AddFloat(s, k, v) }
func (d *IniDocument) AddString(s, k, v string) bool        { return d.ini.AddString(s, k, v) }
