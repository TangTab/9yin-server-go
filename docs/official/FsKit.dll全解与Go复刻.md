# FsKit.dll 官方全解与 Go 复刻

> 权威快照：`BGserverCJ\Debug\FsKit.dll`（663,552 B，2014-03-22，
> PDB 时间戳 51d6353d = 2013-07-05，源码树 `c:\wuyf\svn\serveriii\branches\publish20130508\server\`）。
> 带 `.map`（1669 符号）+ `.pdb`。Debug 无优化增量链接版（.textbss + ILK thunk 表）。
> 本文所有结论均出自 .map 符号 + .rdata 字符串 + capstone 注释反汇编
> （工具 `tmp_research/fskit_dis.py` / `fskit_dump.py`，产物 `tmp_research/fskit_disasm.txt` 35k 行）。
> Go 复刻：`internal/fskit`（本文 §8 为逐项验收清单）。

## 1. 模块定位与导出协议

FsKit = Fs 框架的**工具箱模块**：INI 解析（两套）、字符串/路径工具、文件搜索、
动态数组实体、主控制台。不含网络/业务。源码文件：

| obj | 源码 | 函数数 | 角色 |
|---|---|---|---|
| IniFile.obj | `server\utils\inifile.cpp` | 40 | CIniFile INI 引擎（非实体） |
| IniDocument.obj | `server\fskit\inidocument.cpp` | 52 | IniDocument 实体（包装 CIniFile） |
| ArrayList.obj | `server\fskit\arraylist.cpp` | 32 | ArrayList 实体（子项树） |
| FileSearch.obj | `server\fskit\filesearch.cpp` | 21 | FileSearch 实体（目录/文件枚举） |
| MainConsole.obj | `server\fskit\mainconsole.cpp` | 20 | MainConsole 实体 + FxModule 8 导出 |
| Global.obj | （fskit/global.cpp） | 18 | 6 个全局扩展函数 |
| DllMain.obj | （fskit/dllmain.cpp） | 7 | CLeakChecker 内存泄漏检查 |

公共库引用（.map 断言路径）：`public\inlines.h`（GetHashValue/GetHashValueCase）、
`public\faststr.h`（TFastStr）、`public\varlist.h`（IVarList/TVarList）、
`public\automem.h`（TAutoMem）、`utils\arraypod.h`（TArrayPod）。

### 导出表（8 个，全模块同构）

| RVA | 导出 | 行为 |
|---|---|---|
| 0x3e56e | FxModule_GetEntCreator | `return *(void**)0x100dee68`（CRT 静态构造好的 creator） |
| 0x3f4a5 | FxModule_GetFuncCreator | 同上（全局 FuncCreator 链） |
| 0x3e861 | FxModule_GetIntCreator | NULL（不提供） |
| 0x3f99b | FxModule_GetLogicCreator | NULL |
| 0x3f4c3 | FxModule_GetSpace | 字符串常量（模块域） |
| 0x3ffb3 | FxModule_GetType | "FxModule" |
| 0x3f6d0 | FxModule_GetVersion | 版本常量 |
| 0x3f57c | FxModule_Init | `g_core = arg`（存 0x100dee64，唯一动作） |

**装载协议定案**：CRT `.CRT$XCU` 动态初始化（0x100bb0f0 起 12 个 init）在
DLL_PROCESS_ATTACH 时执行：`ArrayListEntCreator_Init(0x100dee68)` 等构造函数
把全局 creator 及其全部属性/方法链表（`IEntCreator::Property<T>()` /
`Method<T>()` inline → Get/SetPropertyLink 链表）**在加载时就构建完毕**。
框架随后只调用 Get*Creator 拉链表读数据——不在 Init 里做注册。
DllMain(0x1004d2b0)：DLL_PROCESS_ATTACH → `CLeakChecker::Initialize()`；
DLL_PROCESS_DETACH → `DumpMemory()` + `Shutdown()`。

### IEntCreator 属性/方法描述结构（0x1004caa0 模式）

```
struct IEntProperty { const char* name; int type; getter_t; setter_t; link; }
struct IEntMethod  { const char* name; int paramCount/ret; mid_t; link; }
```
属性 type=6（string）等。Go 复刻不逐字模拟链表，改为声明式注册表（§7）。

## 2. CIniFile —— INI 引擎（inifile.cpp，40 函数）

### 2.1 数据布局

```
+0x28 m_fileName TFastStr<0xCA>
+0x30 m_useCoreFile bool（构造 CIniFile(name, 1) 时置位：走 core_file VFS 包装）
+0x34 m_buffer char*（整份文本，Build 后所有字符串都活在这块里）
+0x38 m_bufferSize u32
+0x3c m_sections TArrayPod<section_t,8>
+0xcc m_items     TArrayPod<item_t,8>

section_t { u32 nameOff; u32 nameHash; u32 firstItem; u32 itemCount; }
item_t    { u32 keyOff; u32 keyHash; u32 valueOff; }   // offset 均相对 m_buffer
```

### 2.2 哈希方言（public\inlines.h）

```c
// GetHashValue：大小写不敏感（tolower 表在 0x100d3020，257 项 CRT _tolower）
u32 GetHashValue(const char* s) { u32 h=0; for(;*s;s++) h = h*131 + tolower_tbl[(u8)*s]; return h; }
// GetHashValueCase：大小写敏感（arraylist.cpp 用）
u32 GetHashValueCase(const char* s){ u32 h=0; for(;*s;s++) h = h*131 + (s8)*s; return h; }
```
tolower 表 0x80..0xFF 全为恒等（GBK 字节不折叠）。查找 = hash 预筛 + `_stricmp`
二次确认 → **section/item 查找全大小写不敏感**；ArrayList 子项哈希则是**大小写敏感**版。

### 2.3 LoadFromFile（0x10042880）

1. 释放旧 buffer、clear sections/items。
2. `fopen(name, "rb")`（m_useCoreFile 时走 core_file::fopen → get_standard_filename
   归一化 + 框架 VFS vtable+0x60/+0x38）。失败 → false。
3. fseek END → ftell = size → rewind；`new char[size+2]`；fread size 字节，
   短读 → fclose/free/false。
4. **尾部哨兵**：`buf[size]='\n'; buf[size+1]=0`（保证最后一行无换行也能收行）。
5. `Build(buf, size+2)`。
LoadFromString(str, len) 同 Build 直调。

### 2.4 Build 解析器（0x10043200）——方言核心

逐字符状态机（lineStart/tokenLen/bInBracket/curSection）：

- `\r`/`\n`：置 NUL 收行，bInBracket=0；tokenLen==0 → 空行跳过。
  收行时：
  - `*lineStart==';'` → 整行注释，丢弃；
  - tokenLen>=2 且前两字符 `//` → 注释，丢弃；
  - curSection==NULL → **隐式空名 section**（name 指向末尾 NUL，即 ""）；
  - **先整行 rtrim**（尾部 ' '/'\t' 逐位置 NUL——key/value 侧行尾空白都削），
    再找 `=`（strchr）：
    - 无 `=`：key=整行，value=""（指向末尾 NUL）；
    - 有 `=`：`*eq=0`；key 侧再 rtrim（'=' 前的空白，如 `k =v`）；
      value 侧 **ltrim**（前导 ' '/'\t' 逐位置 NUL 跳过；行内空白保留）；
  - push item{keyOff,keyHash,valueOff}；curSection->itemCount++；tokenLen=0。
- `[`：tokenLen==0 时才算节头（行中 `[` 是普通字符）——bInBracket=1、置 NUL、
  lineStart=pos+1。
- `]`：bInBracket 时收节——置 NUL、push section{name=lineStart,hash,
  firstItem=items.size(),count=0}、tokenLen=0、bInBracket=0。
  **`]` 之后的同行残余字符会累积成 token，行尾时作为新节的 item 落库**。
- ' '/'\t' 且 tokenLen==0：前导空白跳过（lineStart 不动）。
- 其余：tokenLen==0 时 lineStart=pos；tokenLen++。

行为速查（复刻验收用）：

| 输入行 | 结果 |
|---|---|
| `[Sec]` | 节 Sec |
| `; cmt` / `// cmt` | 丢弃 |
| `k = v` | key=`k`，value=`v` |
| `k =v w ` | key=`k`，value=`v w`（整行 rtrim 削尾；行内空白保留） |
| `noeq` | key=`noeq`，value=`` |
| 段前裸 item | 归入隐式 `""` 节（SaveToFile 输出 `[]`） |
| `[a] tail` | 节 a + 节 a 的 item key=`tail` |

### 2.5 SaveToFile（0x10043850）

`fopen("wb")`；每节：
```
fprintf(f, "[%s]\r\n", name)
for item: fprintf(f, "%s=%s\r\n", key, value)
fprintf(f, "\r\n")          ← 每节之后一个空行（含最后一节）
```
fclose → true。写出的是 trim 后的串；隐式节名 "" 输出 `[]`。

### 2.6 读 API

- GetData(s,k)：GetSection+GetItem（hash+stricmp），miss → NULL。
- ReadInteger(s,k,def) = atoi(GetData) ：def 兜底。
- ReadFloat(s,k,def) = atof。
- ReadString(s,k,def) = 静态 buffer strcpy 返回（两次调用覆盖）。
- ReadString(s,k,buf,maxlen)：n=strlen(v)；`n>=maxlen → n=maxlen-1`；memcpy+NUL；
  miss → strcpy(buf,def)。返回 buf。
- GetSectionCount/GetSectionList（节名 AddString 进 IVarList）；
  GetItemCount(s)/GetItemList(s)/GetItemValueList(s,k)。

### 2.7 写 API（方言重点）

- `SetData(s,k,v)`（WriteInteger/WriteString/WriteFloat 底层）：
  1. FindSectionIndex miss → AddToBuffer(节名) + push_back 新节（firstItem=items.size()）。
  2. GetItem miss → AddToBuffer(key)、新 item（valueOff=末尾 NUL=""）**insert 到
     本节末**（firstItem+itemCount），itemCount++，其后所有节的 firstItem++ 修正。
  3. 命中：`strcmp(旧值,新值)!=0` 才 **AddToBuffer(新值)（append 到 buffer 尾，
     旧串成为孤儿数据）** 并改 valueOff；同值则完全不动（保序保内容）。
- `AddData(s,k,v)`（AddInteger/AddString/AddFloat）：同 miss 路径但**不查重**，
  每次必 append 一行（同 key 可重复）。
- AddToBuffer(str)（0x10044c00）：realloc buffer（size+strlen+1）、memcpy 到旧 size
  处、返回**旧 size**（即新串 offset）。
- AddSection(s)：查重后 append；FindSection/DeleteSection（删节连同 items 搬移，
  后续节 firstItem 修正）；FindItem/DeleteItem（删后节 itemCount--、后续节 firstItem--）。
- GetMemoryUsage：bufferSize+容量。

数值写格式：WriteInteger → `"%d"`；WriteFloat → `"%g"`；AddFloat 同。

### 2.8 NewInstance/Release

`CIniFile::NewInstance(name)` = `new CIniFile(name)`（debug new 带行号）；
`Release()` = `delete this`。无全局表/引用计数。

## 3. IniDocument 实体（inidocument.cpp）

- 实体类型名 `"IniDocument"`；Init(args)：`args[0]`=文件名 → `new CIniFile(name)`
  存 this+0x28。**Init 不自动 Load**（LoadFromFile 显式调）。
- 属性：Name（get：实体名"IniDocument"？getter 返回固定串）；FileName（get/set，
  set 即 CIniFile::SetFileName）。
- 方法（21 个 mid，全部校验 args 数量不符返回 false）：
  LoadFromFile()/SaveToFile()/AddSection(s)/FindSection(s)/DeleteSection(s)/
  GetSectionCount()/GetSectionList()/FindItem(s,k)/DeleteItem(s,k)/
  GetItemCount(s)/GetItemList(s)/GetItemValueList(s,k)/ReadInteger(s,k,def)/
  ReadFloat(s,k,def)/ReadString(s,k,def)/WriteInteger(s,k,v)/WriteFloat(s,k,v)/
  WriteString(s,k,v)/AddInteger(s,k,v)/AddFloat(s,k,v)/AddString(s,k,v)。

## 4. ArrayList 实体（arraylist.cpp）

- 布局：+0x28 m_name TFastStr<16>；+0x48 m_hash u32；+0x4c m_children
  `TArrayPod< ArrayList*, 1 >`。
- 属性：Name（get/set → SetName + SetHash）。
- 方法：CreateChild(name)→PERSISTID / AddChild(id) / RemoveChild(name) /
  RemoveChildByIndex(i) / ClearChild() / GetChildCount() / GetChildList() /
  FindChild(name) / GetChild(name)→PERSISTID / GetChildByIndex(i)→PERSISTID。
- CreateChild（0x100490d0）：`Core->CreateEntity("ArrayList", varlist)`（vtable+0x94）
  → 新实体 SetName(name)、SetHash(**GetHashValueCase**，大小写敏感)、push_back、
  返回 GetID()。失败返回空 PERSISTID。
- 子项查找 FindChildIndex：按 name+hashCase 精确匹配。
- GetChildList：result.Clear() + 逐项 AddString(name)。

## 5. FileSearch 实体（filesearch.cpp）

- 布局：+0x28 m_path TFastStr<16>；+0x2c m_fileName；+0x68 结果容器（带 vtable 的
  字符串数组）。**两份结果容器**（dir/file）。
- 属性：Path（get）、FileName（get）。
- SearchDir(path)（0x1004e430）：清 dir 容器 → Port_FindFirst(&fs, path, "") →
  循环 `GetFindIsDir && name!="." && name!=".."` → AddString(name) →
  FindNext/FindClose；m_path=path；true。找不到 → false（容器已清空）。
- SearchFile(path, filename)：同构，取 `GetFindIsFile`（非目录即文件）。
- GetDirCount/GetFileCount/GetDirList/GetFileList：GetDirList = result.Clear() +
  `result.Assign(dirList, 0, dirList.Count())`（纯文件名，**无路径前缀**）。
- Port_FindFirst(search, dir, filename)（0x1004e630）通配方言：
  - filename 空 → `snprintf(pattern, 0x1ff, "%s*.*", dir)`
  - filename 含 `.` → `"%s%s"`（原样）
  - 否则 → `"%s*.%s"`（**无点=扩展名，自动加 `*.` 前缀**）
  - `FindFirstFileA(pattern, &search->data)`；`port_file_search_t{handle, WIN32_FIND_DATA}`。
- GetFindIsDir = `attrs & 0x10`；GetFindIsFile = `!(attrs & 0x10)`。

## 6. MainConsole 实体 + Global 扩展函数

### 6.1 MainConsole

- Init（0x100561f0）：`SetThreadExecutionState(0x80000003)`（防休眠）+
  `SetConsoleCtrlHandler(handler, TRUE)`。handler（0x10056270）对 CTRL_C(0)/
  BREAK(1)/CLOSE(2)/LOGOFF(5)/SHUTDOWN(6) 返回 TRUE（忽略），其余 FALSE。
- WriteLine(line)（0x10056330）：`printf("%s\r\n", line)` → true。**CRLF**。
- SetColor(color)（0x100563c0）：stricmp 颜色名 → attr：
  `Red=4 Blue=1 Green=2 Yellow=6 Purple=5 Cyan=3 LightRed=0xC LightBlue=9
  LightGreen=0xA LightYellow=0xE LightPurple=0xD LightCyan=0xB Gray=7 White=0xF`
  命中 → `GetStdHandle(STD_OUTPUT_HANDLE)` + `SetConsoleTextAttribute(h, attr)`
  再存名；**未知名只存名不改色**。GetColor 返回存的名字。
- 属性 Color（get/set）。

### 6.2 Global 六个扩展函数（extfunc，IVarList 进出）

| 函数 | 语义 |
|---|---|
| `ext_split_string(str, sep)` | sep 取 args[1] 首字符，空 → `','`。分隔符逐位置 NUL；**连续分隔符的空 token 跳过**；**尾随分隔符产生一个空串 token**（p!=buf 判据）；空输入零输出。 |
| `ext_split_file_path(path)` | `GetFullPathNameA` → filePart 非空：`[*filePart=0` 截断的全路径（目录含尾 `\`）, 文件名] 两元素；filePart 空 → [全路径] 一元素；失败 → 零输出。 |
| `ext_split_file_name(name)` | 从尾向前找 `.`：命中 → 截断，返回 [base, ext]；遇 `\` 或 `/` 先出现 → [整名, ""]。".hidden" → ["", "hidden"]。 |
| `ext_get_file_time(path)` | CreateFileA(OPEN_EXISTING)→GetFileTime(**最后写时间**)→FileTimeToSystemTime（**UTC，无本地化**）→ `"%04d-%02d-%02d %02d:%02d:%02d"`。失败 → ""。 |
| `ext_get_full_path(path)` | GetFullPathNameA → 全路径；失败 → ""。 |
| `ext_gen_unique_name()` | `CoCreateGuid` → `"%08X%04X%04X%02X"×8`（Data1,2,3 + Data4[0..7] 正序）= **32 位大写 hex 无连字符**；失败 NULL。 |

### 6.3 IVarList vtable 速查（Debug thiscall，参数压栈）

`+0x08 Assign(other, start, count)`；`+0x0c Clear()`；`+0x14 GetCount()`；
`+0x30 AddString(const char*)`；`+0x5c GetString(index)→const char*`（TVarGetter）。

## 7. CLeakChecker（DllMain.obj）

Initialize：`_CrtSetDbgFlag` + `OurReportingFunction`（_CrtSetReportHook），
输出重定向 `mem_fs_kit.log` / `assert.log`（`_AssertionFail`："Assert(%s), file: %s,
line: %d"）。DETACH 时 DumpMemory（CRT 泄漏转储）+ Shutdown。Go 复刻以
`runtime.MemStats`/pprof 替代，语义等价记录。

## 8. Go 复刻对齐清单（internal/fskit）

| 官方 | Go | 对齐点 |
|---|---|---|
| CIniFile 全 API | `inifile.go IniFile` | 同 offset-free 实现（[]byte buffer + section/item 索引），Build/Save 逐字节方言（含哨兵 \n、隐式节、trim 置零语义、CRLF、节后空行） |
| （Go 扩展）分词统一入口 | `inifile.go ParseItems` | 按文件序回调 (节,键,值)；全仓 18 处配置解析器（login/solo 家族/fsmember 家族/taskdata 等）已收编，不再各写行循环。'#' 注释官方 Build 不吃（漏成 key-only 条目）由各消费方过滤；taskdata/text.go 的 idres 文本表有意不收编（值不 trim） |
| GetHashValue/Case | `HashValue/HashValueCase` | h*131 + tolower 表（0x80+ 恒等）/ 原字节 |
| 查找 | `GetSection/GetItem` | hash 预筛 + EqualFold(ASCII) 等价 stricmp |
| IniDocument 实体 | `inidocument.go` | Init(args[0]) 不自动 Load；21 方法 |
| ArrayList | `arraylist.go` | 大小写敏感 131 哈希；CreateChild→id |
| FileSearch | `filesearch.go` | `%s*.*`/`%s%s`/`%s*.%s` 通配方言；`.`/`..` 排除；纯名字列表 |
| MainConsole | `mainconsole.go` | 14 色表；CRLF WriteLine；SetOutput 等 Go 化 |
| ext_* 六件 | `global.go` | 全部含边界（尾分隔符空 token、UTC 时间、GUID 大写 32 hex） |
| FxModule 8 导出 | `module.go` | 声明式注册表（EntityType/ExtFunc），供宿主装配 |
| CLeakChecker | doc 说明 | Go 侧以测试+pprof 替代，不复刻 CRT |

**已知不复刻细节**（记录在案）：core_file VFS 包装（m_useCoreFile 路径）——Go 直用
os 包；CRT debug 堆统计（GetMemoryUsage 返回逻辑对齐但容量口径为 Go slice cap）；
`SetConsoleTextAttribute` 的 Win32 控制台着色（Go 侧提供 ANSI 等价层）。

## 9. 复刻验证状态

- `go test ./internal/fskit/...` 全绿（见 `fskit_test.go`：Build 方言 12 例、
  Save 回放、Write/Add 家族、split 三兄弟、颜色表、FileSearch、GUID 形态等）。
- `cmd/fskit`：可执行自检入口（round-trip 演示）。
