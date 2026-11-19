package fsloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// CoreConfig 复刻官方 CCoreConfig（app_load_core_ini 解析产物）。
// 字段顺序对应官方写入序（DebugLog→Sleep→Reload→ScriptPath→ResourcePath→
// Modules→FileSystem→FileConfig→MainEntity→MainScript→MainLogic）。
type CoreConfig struct {
	// Sleep 主循环休眠毫秒数（[System] Sleep，缺省 0；<0 不休眠全速）。
	Sleep int
	// DebugLog 是否启用调试日志系统（[System] DebugLog，缺省开）。
	DebugLog bool
	// ScriptReload 脚本是否即时重新加载（[Script] Reload，缺省开）。
	ScriptReload bool
	// ScriptPath 脚本目录（[Script] Path；相对路径以工作目录归一，
	// 保证尾分隔符——官方追加 '\\'）。
	ScriptPath string
	// ResourcePath 资源目录（[Resource] Path；同上归一）。
	ResourcePath string
	// Modules 指定装载的组件模块（[Module] 裸行表；空=全部装载）。
	Modules []string
	// FileSystem 文件系统名（[System] FileSystem；空=内置 OS 文件系统）。
	FileSystem string
	// FileConfig 文件系统配置（[System] FileConfig；空=不加载）。
	FileConfig string
	// MainEntity 主实体名称（[System] MainEntity，如 "MainCon"）。
	MainEntity string
	// MainScript 启动脚本名称（[System] MainScript，如 "main"）。
	MainScript string
	// MainLogic 启动逻辑类名称（[System] MainLogic；空=不绑定）。
	MainLogic string
	// ExtraArgs argv[2..] 附加参数（官方追加进配置尾段随 Startup 传入）。
	ExtraArgs []string
	// IniName 实际装载的 ini 文件名（Run 发现序产物；角色装配重读用
	// ——官方 main.lua 亦以 IniDocument 二次打开同一文件）。
	IniName string
	// Raw 原始 ini 视图（官方 CCoreConfig 仅取白名单键，脚本/角色装配
	// 需要访问其余节——[Main]/[Inner]/[Echo]/[Broadcast]/[Logic]）。
	Raw *fskit.IniFile
}

// LoadCoreConfig 复刻官方 app_load_core_ini(ini_name, CCoreConfig&)：
// ini 装载失败返回 error（壳层文案 "Load core ini failed"）；
// 脚本/资源目录不存在为致命错误（官方文案 "Script/Resource path not found, %s"）。
func LoadCoreConfig(ini *fskit.IniFile) (*CoreConfig, error) {
	if !ini.LoadFromFile() {
		return nil, fmt.Errorf("fsloader: ini %q 装载失败", ini.GetFileName())
	}
	cfg := &CoreConfig{
		DebugLog:     ini.ReadInteger("System", "DebugLog", 1) != 0,
		Sleep:        ini.ReadInteger("System", "Sleep", 0),
		ScriptReload: ini.ReadInteger("Script", "Reload", 1) != 0,
	}

	// 脚本/资源目录：官方语义——非空时必须存在（工作目录归一 + 尾分隔符），
	// 不存在即致命；空值跳过校验（FsLoader.ini 生产配置均给出）。
	var err error
	if cfg.ScriptPath, err = resolveDir(ini, "Script", "Path"); err != nil {
		return nil, err
	}
	if cfg.ResourcePath, err = resolveDir(ini, "Resource", "Path"); err != nil {
		return nil, err
	}

	// [Module] 裸行表：官方 GetItemList 取 key（无 '=' 行 key=整行）。
	if ini.FindSection("Module") {
		cfg.Modules = ini.GetItemList("Module")
	}

	cfg.FileSystem = ini.ReadString("System", "FileSystem", "")
	cfg.FileConfig = ini.ReadString("System", "FileConfig", "")
	cfg.MainEntity = ini.ReadString("System", "MainEntity", "")
	cfg.MainScript = ini.ReadString("System", "MainScript", "")
	cfg.MainLogic = ini.ReadString("System", "MainLogic", "")

	// 原始视图随行（角色装配读其余节；官方由脚本自行二开 ini）。
	cfg.Raw = ini
	return cfg, nil
}

// resolveDir 官方目录键归一：GetFullPathName 绝对化 → 存在性校验
// （Port_DirExists）→ 追加尾分隔符。
func resolveDir(ini *fskit.IniFile, section, key string) (string, error) {
	raw := ini.ReadString(section, key, "")
	if raw == "" {
		return "", nil
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("fsloader: %s path 归一失败: %s", section, raw)
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("%s path not found, %s", section, raw)
	}
	if !strings.HasSuffix(abs, string(os.PathSeparator)) {
		abs += string(os.PathSeparator)
	}
	return abs, nil
}

// IniNameFromExe 复刻官方 app_get_ini_name：exe 路径 → "<去扩展名基名>.ini"。
// 派生失败返回空串（壳层回退字面量 "FsLoader.ini"）。
func IniNameFromExe(exePath string) string {
	if exePath == "" {
		return ""
	}
	base := filepath.Base(exePath)
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" {
		return ""
	}
	return base + ".ini"
}
