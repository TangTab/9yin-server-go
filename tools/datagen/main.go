// datagen：服务端数据构建器——九阴客户端资源 + 黑金参考资源 → 服务端
// XML。生成物先以散文件落在输出根（缺省 testdata）的对应目录，验证无误后
// 再并入资源层（go run ./cmd/mpq store pack）。
//
// 用法: go run ./tools/datagen [构建器…] [选项]
//
//	npc     NPC 模板 / 场景摆放 / 任务域 / 战斗模板 / 任务文案
//	patrol  NPC 巡逻路线 mappath
//	item    物品配置 ini/Item
//	skill   技能配置 ini/Skill
//	action  招式判定节拍 ini/Action
//
// 2026-11-25 起只有 item / skill 必须构建（模型重建）。NPC、任务域、任务文案、
// 巡逻路线的装载器可以直接读数据根里的客户端 share 树（开关
// NINEYIN_DATA_SOURCE，见 docs/official/NPC与任务数据来源直读落地.md）；npc 与
// patrol 构建器保留，产物是回退路径与 A/B 对拍基线——直读的键名 / 取值口径
// 以这里的输出为准，两边的差异由对拍测试兜住，改这里的列映射时要同步改
// internal/npcfeed/load_share.go 的 npcColumnBindings。
//
// 不带构建器名即构建全部。任务数据（ini/Quest 的任务域 XML）由 npc 构建
// 器一并产出。只需要两类输入：
//
//	-client  客户端资源解压根（含 share/、text/、lua64/），参数传入
//	-heijin  黑金参考资源根（层文件 h001.rep 所在目录），缺省仓内
//	         testdata/references
//
// 各构建器的输入在两类根下的位置见对应子包的文件头说明。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/action"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/item"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/npc"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/patrol"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/skill"
)

// defaultClientDirectory 客户端资源解压根的本机缺省（别的机器用 -client 指定）。
const defaultClientDirectory = `E:\Workspace\project\game\resources`

// inputs 一次运行的两类输入与公共选项。
type inputs struct {
	Client string // 客户端资源解压根
	Heijin string // 黑金参考资源根（空串 = 不用黑金样本）
	Out    string // 输出根：生成物以散文件落在其下的对应目录
	Repo   string // 仓库根（审计报告落在 artifacts/local）
	Verify bool   // 生成后用服务端装载器回读校验
}

// builder 一个构建器：名字（子命令）、说明、执行函数。切片序即缺省构建序。
type builder struct {
	name string
	help string
	run  func(in inputs) error
}

var builders = []builder{
	{"npc", "NPC 模板 / 场景摆放 / 任务域 / 战斗模板 / 任务文案", buildNpc},
	{"patrol", "NPC 巡逻路线 mappath", buildPatrol},
	{"item", "物品配置 ini/Item", buildItem},
	{"skill", "技能配置 ini/Skill", buildSkill},
	{"action", "招式判定节拍 ini/Action", buildAction},
}

func main() {
	flag.Usage = usage
	client := flag.String("client", defaultClientDirectory, "客户端资源解压根（含 share/、text/、lua64/）")
	heijin := flag.String("heijin", "", "黑金参考资源根（层文件 h001.rep 所在目录）；缺省 <仓库>/testdata/references，显式置空则不用黑金样本")
	out := flag.String("out", "", "输出根：生成物先以散文件落在这里；缺省 <仓库>/testdata")
	verify := flag.Bool("verify", true, "生成后用服务端装载器回读校验")
	args := reorderArgs(os.Args[1:])
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}

	repo := repoRoot()
	in := inputs{Client: filepath.Clean(*client), Out: *out, Repo: repo, Verify: *verify}
	in.Heijin = *heijin
	if in.Heijin == "" && !flagWasPassed(args, "heijin") {
		in.Heijin = filepath.Join(repo, "testdata", "references")
	}
	if in.Heijin != "" {
		in.Heijin = filepath.Clean(in.Heijin)
	}
	if in.Out == "" {
		in.Out = filepath.Join(repo, "testdata")
	}
	in.Out = filepath.Clean(in.Out)

	selected, err := selectBuilders(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "datagen:", err)
		flag.Usage()
		os.Exit(2)
	}
	if err := checkInputs(in, selected); err != nil {
		fmt.Fprintln(os.Stderr, "datagen:", err)
		os.Exit(1)
	}

	fmt.Printf("datagen: 客户端资源 %s\n", in.Client)
	if in.Heijin != "" {
		fmt.Printf("datagen: 黑金参考 %s\n", in.Heijin)
	}
	fmt.Printf("datagen: 输出根   %s（先落散文件，验证无误后再并入资源层）\n", in.Out)

	failed := 0
	for _, b := range selected {
		fmt.Printf("datagen: 构建 %s —— %s\n", b.name, b.help)
		if err := b.run(in); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "datagen: %s 失败: %v\n", b.name, err)
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "datagen: %d/%d 个构建器失败\n", failed, len(selected))
		os.Exit(1)
	}
	fmt.Printf("datagen: %d 个构建器全部完成\n", len(selected))
	if samePath(in.Out, filepath.Join(repo, "testdata")) {
		fmt.Println(`datagen: 散文件验证无误后并入资源层：go run ./cmd/mpq store pack testdata -prefix b -prune -exclude mpq -exclude README.md`)
	}
}

// selectBuilders 把位置参数（构建器名，可多个、可重复）按缺省构建序去重；
// 不带参数即全部。名字不认识时报错并列出可选项。
func selectBuilders(names []string) ([]builder, error) {
	if len(names) == 0 {
		return builders, nil
	}
	wanted := map[string]bool{}
	var unknown []string
	for _, name := range names {
		if findBuilder(name) == nil {
			unknown = append(unknown, name)
			continue
		}
		wanted[name] = true
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("未知构建器 %s（可选：%s）",
			strings.Join(unknown, " "), builderNames())
	}
	var selected []builder
	for _, b := range builders {
		if wanted[b.name] {
			selected = append(selected, b)
		}
	}
	return selected, nil
}

func findBuilder(name string) *builder {
	for i := range builders {
		if builders[i].name == name {
			return &builders[i]
		}
	}
	return nil
}

func builderNames() string {
	names := make([]string, len(builders))
	for i, b := range builders {
		names[i] = b.name
	}
	return strings.Join(names, " / ")
}

// checkInputs 开工前核对两类输入，报错时说清缺什么、该用哪个参数。
func checkInputs(in inputs, selected []builder) error {
	if info, err := os.Stat(in.Client); err != nil || !info.IsDir() {
		return fmt.Errorf("客户端资源根不可用 %s（-client 指定；应含 share/、text/、lua64/）", in.Client)
	}
	if info, err := os.Stat(filepath.Join(in.Client, "share")); err != nil || !info.IsDir() {
		return fmt.Errorf("客户端资源根 %s 下没有 share/ 目录（-client 应指向解压根，而非其子目录）", in.Client)
	}
	for _, b := range selected {
		if b.name != "patrol" {
			continue
		}
		// 巡逻路线源在客户端 share/creator/mappath（与 npc 构建器的
		// npc_creator 摆放树同级）。
		if info, err := res.Stat(filepath.Join(in.Client, "share", "creator", "mappath")); err != nil || !info.IsDir() {
			return fmt.Errorf("客户端资源根 %s 下没有 share/creator/mappath（patrol 构建器的输入）", in.Client)
		}
	}
	needsHeijin := false
	for _, b := range selected {
		if b.name == "item" || b.name == "skill" {
			needsHeijin = true
		}
	}
	if !needsHeijin || in.Heijin == "" {
		return nil
	}
	// 经资源仓读：黑金参考是层文件（h001.rep），磁盘上没有散文件。
	if info, err := res.Stat(filepath.Join(in.Heijin, "ini")); err != nil || !info.IsDir() {
		return fmt.Errorf("黑金参考资源根不可用 %s（-heijin 指定；应含层文件 h001.rep）", in.Heijin)
	}
	return nil
}

// buildNpc ini/Npc、ini/Scene、ini/Quest、ini/Text。
func buildNpc(in inputs) error {
	return npc.Run(npc.Options{
		Client:     in.Client,
		Out:        filepath.Join(in.Out, "ini", "Npc"),
		ScenesOut:  filepath.Join(in.Out, "ini", "Scene"),
		QuestOut:   filepath.Join(in.Out, "ini", "Quest"),
		TextOut:    filepath.Join(in.Out, "ini", "Text"),
		ReportPath: filepath.Join(in.Repo, "artifacts", "local", "npcgen-report.json"),
		Verify:     in.Verify,
	})
}

// buildPatrol 数据根 share/creator/mappath：NPC 巡逻路线表，客户端
// share/creator/mappath 原样透传（装载器 internal/npcfeed/patrol.go 按场景
// 短名前缀读；输出与客户端资源树同路径）。
func buildPatrol(in inputs) error {
	return patrol.Run(patrol.Options{
		In:         filepath.Join(in.Client, "share", "creator", "mappath"),
		Out:        filepath.Join(in.Out, "share", "creator", "mappath"),
		ReportPath: filepath.Join(in.Repo, "artifacts", "local", "patrolgen-report.json"),
		Verify:     in.Verify,
	})
}

// buildItem ini/Item：设计态表与两个 lua 定义表取自客户端，黑金样本与
// struct schema 取自参考资源根。
func buildItem(in inputs) error {
	return item.Run(item.Options{
		In:             filepath.Join(in.Client, "share", "item"),
		Out:            filepath.Join(in.Out, "ini", "Item"),
		Heijin:         in.Heijin,
		LuaDefine:      filepath.Join(in.Client, "lua64", "share", "item_static_data_define.lua"),
		ItemTypeDefine: filepath.Join(in.Client, "lua64", "share", "itemtype_define.lua"),
		ReportPath:     filepath.Join(in.Repo, "artifacts", "local", "itemgen-report.json"),
		Verify:         in.Verify,
	})
}

// buildSkill ini/Skill：设计态表与流程包取自客户端 share/，黑金样本与
// struct schema 取自参考资源根。
func buildSkill(in inputs) error {
	return skill.Run(skill.Options{
		In:         filepath.Join(in.Client, "share", "skill"),
		Rule:       filepath.Join(in.Client, "share", "rule"),
		Out:        filepath.Join(in.Out, "ini", "Skill"),
		Heijin:     in.Heijin,
		LuaDefine:  filepath.Join(in.Client, "lua64", "share", "skill_static_data_define.lua"),
		ReportPath: filepath.Join(in.Repo, "artifacts", "local", "skillgen-report.json"),
		Verify:     in.Verify,
	})
}

// buildAction ini/Action：客户端招式表（ini/action/zhaoshi_*.ini）离线编译为
// 判定节拍产物——战斗链每击结算的时刻表。
func buildAction(in inputs) error {
	return action.Run(action.Options{
		Client:     in.Client,
		Out:        in.Out,
		ReportPath: filepath.Join(in.Repo, "artifacts", "local", "actiongen-report.json"),
		Verify:     in.Verify,
	})
}

// valueFlagNames 带取值的选项（值跟在下一个参数里；`-name=value` 形式不需要）。
var valueFlagNames = map[string]bool{"client": true, "heijin": true, "out": true}

// reorderArgs 把命令行重排成「选项在前、构建器名在后」再交给 flag 解析：
// 标准库的 flag 遇到第一个位置参数就停止解析，不重排的话
// `datagen npc -client X` 会把 -client 当成构建器名。`--` 之后一律按位置参数。
func reorderArgs(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			continue
		}
		if valueFlagNames[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positional...)
}

// flagWasPassed 报告参数里是否显式给出了某个开关（区别于取缺省值）。
func flagWasPassed(args []string, name string) bool {
	for _, arg := range args {
		if arg == "-"+name || arg == "--"+name || strings.HasPrefix(arg, "-"+name+"=") ||
			strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
	}
	return false
}

// repoRoot 自工作目录与可执行文件所在目录向上定位仓库根（go.mod 所在层）。
func repoRoot() string {
	var starts []string
	if workingDirectory, err := os.Getwd(); err == nil {
		starts = append(starts, workingDirectory)
	}
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}
	for _, start := range starts {
		for current := filepath.Clean(start); ; current = filepath.Dir(current) {
			if info, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && !info.IsDir() {
				return current
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	return "."
}

func samePath(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && rel == "."
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `用法: go run ./tools/datagen [构建器…] [选项]

构建器（不带名字即全部）：
`)
	for _, b := range builders {
		fmt.Fprintf(flag.CommandLine.Output(), "  %-6s %s\n", b.name, b.help)
	}
	fmt.Fprintf(flag.CommandLine.Output(), `
选项：
`)
	flag.PrintDefaults()
}
