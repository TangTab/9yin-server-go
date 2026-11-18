package fskit

// 模块装配（对齐官方 FxModule 8 导出 + CRT 静态构造即注册的装载协议）。
// 官方在 DLL 加载时把 4 个实体 creator 与 6 个 FuncCreator 的属性/方法链表
// 全部构建完毕，框架随后拉取。Go 侧以声明式注册表等价。

// 模块常量（官方 FxModule_GetType/GetVersion/GetSpace 的返回）。
const (
	ModuleType    = "FxModule"
	ModuleVersion = 100
	ModuleSpace   = "FsKit"
)

// EntityFactory 对齐官方 IEntCreator::Create 的构造面。
type EntityFactory func(args *VarList) any

// ExtFuncDecl 对齐官方 IFuncCreator 声明（函数名 + mid 分发）。
type ExtFuncDecl struct {
	Name string
	Call func(args *VarList, result *VarList) bool
}

// EntityDecl 对齐官方 IEntCreator（类型名 + 构造 + 属性/方法集）。
type EntityDecl struct {
	TypeName string
	New      EntityFactory
	Props    []string // 属性名（官方 property 链表）
	Methods  []string // 方法名（官方 method 链表）
}

// EntityTypes 官方 GetEntCreator 可见的 4 个实体类型。
var EntityTypes = []EntityDecl{
	{
		TypeName: ArrayListTypeName,
		New: func(args *VarList) any {
			// 官方 CreateChild 经 Core::CreateEntity("ArrayList") 建同类实体；
			// 顶层构造入口带可选名字参数（脚本 CreateEntity 习惯）。
			name := ""
			if args != nil && args.Count() > 0 {
				name = args.GetString(0)
			}
			return &ArrayList{name: name, hash: HashValueCase(name)}
		},
		Props:   []string{"Name"},
		Methods: []string{"CreateChild", "AddChild", "RemoveChild", "RemoveChildByIndex", "ClearChild", "GetChildCount", "GetChildList", "FindChild", "GetChild", "GetChildByIndex"},
	},
	{
		TypeName: FileSearchTypeName,
		New:      func(args *VarList) any { return NewFileSearch() },
		Props:    []string{"Path", "FileName"},
		Methods:  []string{"SearchDir", "SearchFile", "GetDirCount", "GetDirList", "GetFileCount", "GetFileList"},
	},
	{
		TypeName: IniDocumentTypeName,
		New: func(args *VarList) any {
			name := ""
			if args != nil && args.Count() > 0 {
				name = args.GetString(0)
			}
			return NewIniDocument(name)
		},
		Props:   []string{"Name", "FileName"},
		Methods: []string{"LoadFromFile", "SaveToFile", "AddSection", "FindSection", "DeleteSection", "GetSectionCount", "GetSectionList", "FindItem", "DeleteItem", "GetItemCount", "GetItemList", "GetItemValueList", "ReadInteger", "ReadFloat", "ReadString", "WriteInteger", "WriteFloat", "WriteString", "AddInteger", "AddFloat", "AddString"},
	},
	{
		TypeName: MainConsoleTypeName,
		New:      func(args *VarList) any { return NewMainConsole() },
		Props:   []string{"Color"},
		Methods: []string{"WriteLine"},
	},
}

// ExtFuncs 官方 GetFuncCreator 可见的 6 个全局扩展函数。
var ExtFuncs = []ExtFuncDecl{
	{Name: "ext_split_string", Call: func(args, result *VarList) bool {
		if args.Count() != 2 {
			return false
		}
		for _, s := range SplitString(args.GetString(0), args.GetString(1)) {
			result.AddString(s)
		}
		return true
	}},
	{Name: "ext_split_file_path", Call: func(args, result *VarList) bool {
		if args.Count() != 1 {
			return false
		}
		for _, s := range SplitFilePath(args.GetString(0)) {
			result.AddString(s)
		}
		return true
	}},
	{Name: "ext_split_file_name", Call: func(args, result *VarList) bool {
		if args.Count() != 1 {
			return false
		}
		for _, s := range SplitFileName(args.GetString(0)) {
			result.AddString(s)
		}
		return true
	}},
	{Name: "ext_get_file_time", Call: func(args, result *VarList) bool {
		if args.Count() != 1 {
			return false
		}
		result.AddString(GetFileTime(args.GetString(0)))
		return true
	}},
	{Name: "ext_get_full_path", Call: func(args, result *VarList) bool {
		if args.Count() != 1 {
			return false
		}
		result.AddString(GetFullPath(args.GetString(0)))
		return true
	}},
	{Name: "ext_gen_unique_name", Call: func(args, result *VarList) bool {
		result.AddString(GenUniqueName())
		return true
	}},
}
