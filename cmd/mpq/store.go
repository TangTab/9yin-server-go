package main

// 多层资源仓子命令。数据根下的层文件 bNNN.rep（测试仓）/ dataNNN.rep
// （正式服）/ hNNN.rep（参考资源，如黑金原档）按层号覆盖，根下散文件盖过
// 一切；资源名即数据根下的相对路径。

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

func storeUsage() {
	fmt.Fprint(os.Stderr, `用法: mpq store <子命令> <数据根> [参数]
  info                        层清单与合并视图统计
  list   [前缀]               合并视图：所在层 / 大小 / 名字
  read   <名字>               命中层与内容预览
  pack   [选项]               把散文件打成新层（默认只收与下层不同的 → 增量层）
  unpack [目标目录] [前缀]    合并视图（散文件 > 高层 > 低层）解包成散文件树
  delete <名字>…              在最高层登记删除（根下同名散文件须先移走）
选项:
  -prefix b|data|h  层文件族：b 测试仓 / data 正式服 / h 参考资源（多族并存时必填；建首个层时必填）
  pack 专用：
  -layer N           目标层号（缺省 = 现有最大层号 + 1）
  -replace           允许整层重建已存在的目标层
  -from <目录>       源目录（缺省 = 数据根自身的散文件）
  -exclude <规则>    排除规则，/ 分隔 glob，可重复；命中文件或其任一级上级目录即排除
  -v 1-4             MPQ 格式版本（缺省 4）
  -full              关闭去重（与下层相同的文件也写入）
  -delete-missing    源中缺失而下层存在的名字登记删除（源须是完整树）
  -prune             成功并回读校验后删除已并入的散文件
读取顺序：根下散文件 > <前缀>N.rep > <前缀>N-1.rep > … > <前缀>001.rep
`)
}

func store(args []string) error {
	if len(args) < 4 {
		storeUsage()
		os.Exit(2)
	}
	cmd := args[2]
	flags, err := parseFlags(args[3:])
	if err != nil {
		return err
	}
	if len(flags.pos) == 0 {
		storeUsage()
		os.Exit(2)
	}
	dir := flags.pos[0]
	opt := res.Options{Prefix: flags.first("prefix")}

	switch cmd {
	case "info":
		return storeInfo(dir, opt)
	case "list":
		return storeList(dir, opt, argAt(flags.pos, 1))
	case "read":
		if len(flags.pos) < 2 {
			return fmt.Errorf("用法: mpq store read <数据根> <名字>")
		}
		return storeRead(dir, opt, flags.pos[1])
	case "pack":
		return storePack(dir, flags)
	case "unpack":
		return storeUnpack(dir, opt, argAt(flags.pos, 1), argAt(flags.pos, 2))
	case "delete":
		if len(flags.pos) < 2 {
			return fmt.Errorf("用法: mpq store delete <数据根> <名字>…")
		}
		return storeDelete(dir, opt, flags.pos[1:])
	}
	fmt.Fprintln(os.Stderr, "未知子命令："+cmd)
	storeUsage()
	os.Exit(2)
	return nil
}

// storeFlags 是 store 子命令解析后的参数：位置参数 + 选项。
type storeFlags struct {
	pos   []string
	flags map[string][]string
}

// valueFlags 是必须带取值的选项；switchFlags 是开关；此外一律报错，
// 免得拼错的选项被静默忽略。
var valueFlags = map[string]bool{"prefix": true, "layer": true, "from": true, "exclude": true, "v": true}

var switchFlags = map[string]bool{"replace": true, "full": true, "delete-missing": true, "prune": true}

func parseFlags(args []string) (*storeFlags, error) {
	f := &storeFlags{flags: map[string][]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			f.pos = append(f.pos, a)
			continue
		}
		name := strings.TrimLeft(a, "-")
		if switchFlags[name] {
			f.flags[name] = []string{""}
			continue
		}
		if !valueFlags[name] {
			return nil, fmt.Errorf("未知选项 %s", a)
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
			return nil, fmt.Errorf("选项 -%s 缺少取值", name)
		}
		i++
		f.flags[name] = append(f.flags[name], args[i])
	}
	return f, nil
}

// first 返回选项的第一个取值；未给出为空串。
func (f *storeFlags) first(name string) string {
	if v := f.flags[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// on 报告开关是否给出。
func (f *storeFlags) on(name string) bool {
	_, ok := f.flags[name]
	return ok
}

// intOf 解析取值为整数；空串返回 0。
func (f *storeFlags) intOf(name string, what string) (int, error) {
	raw := f.first(name)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("无效的%s %q", what, raw)
	}
	return n, nil
}

func argAt(pos []string, i int) string {
	if i < len(pos) {
		return pos[i]
	}
	return ""
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func storeInfo(dir string, opt res.Options) error {
	st, err := res.OpenWith(dir, opt)
	if err != nil {
		return err
	}
	defer st.Close()
	files, err := st.List()
	if err != nil {
		return err
	}
	local := 0
	for _, f := range files {
		if f.Layer == "local" {
			local++
		}
	}

	abs, _ := filepath.Abs(dir)
	fmt.Println("数据根    :", abs)
	if p := st.Prefix(); p != "" {
		fmt.Println("层文件族  :", p)
	} else {
		fmt.Println("层文件族  : 无（纯散文件）")
	}
	layers := st.Layers()
	if len(layers) == 0 {
		fmt.Println("层        : 无")
	} else {
		fmt.Printf("%-6s %-9s %-8s %-6s %-10s %-19s %s\n",
			"层号", "层名", "条目", "删除", "大小", "修改时间", "文件")
		for _, l := range layers {
			fmt.Printf("%-6d %-9s %-8d %-6d %-10s %-19s %s\n", l.Number, l.Name, l.Entries,
				l.Deleted, humanSize(l.Size), l.ModTime.Format("2006-01-02 15:04:05"), l.File)
			for _, w := range l.Warnings {
				fmt.Println("告警      :", l.Name, w)
			}
		}
	}
	fmt.Printf("合并清单  : %d 个文件（层内 %d，散文件 %d）\n", len(files), len(files)-local, local)
	return nil
}

// storePrefix 把 / 或 \ 分隔的前缀参数规范化为小写、\ 分隔。
func storePrefix(arg string) string {
	return strings.ToLower(strings.ReplaceAll(strings.Trim(arg, "/\\"), "/", `\`))
}

func storeList(dir string, opt res.Options, prefixArg string) error {
	st, err := res.OpenWith(dir, opt)
	if err != nil {
		return err
	}
	defer st.Close()
	files, err := st.List()
	if err != nil {
		return err
	}
	prefix := storePrefix(prefixArg)
	shown := 0
	for _, f := range files {
		if prefix == "" || strings.HasPrefix(strings.ToLower(f.Name), prefix) {
			fmt.Printf("%-9s %10d  %s\n", f.Layer, f.Size, f.Name)
			shown++
		}
	}
	fmt.Printf("共 %d 个（全量 %d）\n", shown, len(files))
	return nil
}

func storeRead(dir string, opt res.Options, name string) error {
	st, err := res.OpenWith(dir, opt)
	if err != nil {
		return err
	}
	defer st.Close()
	layer := st.Layer(name)
	if layer == "" {
		fmt.Println("不存在:", name)
		os.Exit(1)
		return nil
	}
	data, err := st.Read(name)
	if err != nil {
		return err
	}
	fmt.Printf("层: %s，大小: %d 字节\n", layer, len(data))
	// 文本文件预览首 200 字节内的第一行；控制字符显示为 ?。
	preview := string(data[:min(200, len(data))])
	preview = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' {
			return '?'
		}
		return r
	}, preview)
	if idx := strings.IndexByte(preview, '\n'); idx >= 0 {
		preview = preview[:idx]
	}
	fmt.Println("预览:", preview)
	return nil
}

func storePack(dir string, f *storeFlags) error {
	opt := res.BuildOptions{
		Prefix:        f.first("prefix"),
		Source:        f.first("from"),
		Exclude:       f.flags["exclude"],
		Replace:       f.on("replace"),
		Full:          f.on("full"),
		DeleteMissing: f.on("delete-missing"),
		Prune:         f.on("prune"),
	}
	var err error
	if opt.Number, err = f.intOf("layer", "层号"); err != nil {
		return err
	}
	if opt.Format, err = f.intOf("v", "格式版本"); err != nil {
		return err
	}
	result, err := res.BuildLayer(dir, opt)
	if err != nil {
		return err
	}

	abs, _ := filepath.Abs(result.File)
	if !result.Changed {
		fmt.Printf("无变化    : 未写出层文件（目标层 %s）\n", abs)
	} else {
		verb := "写出"
		if result.Replaced {
			verb = "整层重建"
		}
		fmt.Printf("%s层    : %s（%s）\n", verb, abs, humanSize(result.Bytes))
	}
	if result.Added > 0 || result.Replaced {
		fmt.Printf("收录      : %d 个文件\n", result.Added)
	}
	if result.Skipped > 0 {
		fmt.Printf("去重      : %d 个与下层内容相同，未写入\n", result.Skipped)
	}
	if result.Excluded > 0 {
		fmt.Printf("排除      : %d 个\n", result.Excluded)
	}
	for _, name := range result.Deleted {
		fmt.Println("登记删除  :", name)
	}
	if result.Pruned > 0 {
		fmt.Printf("修剪散文件: %d 个\n", result.Pruned)
	}
	return nil
}

func storeUnpack(dir string, opt res.Options, destArg, prefixArg string) error {
	st, err := res.OpenWith(dir, opt)
	if err != nil {
		return err
	}
	defer st.Close()

	// 目标目录缺省放在数据根旁：落在数据根之内的话，解出的散文件就成了覆盖层。
	dest := destArg
	if dest == "" {
		dest = filepath.Join(filepath.Dir(st.Dir()), filepath.Base(st.Dir())+"-unpacked")
	}
	dest, err = filepath.Abs(dest)
	if err != nil {
		return err
	}
	if rel, err := filepath.Rel(st.Dir(), dest); err == nil && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("目标目录在数据根之内，解出的散文件会成为覆盖层：%s", dest)
	}
	n, err := res.Unpack(st, dest, prefixArg)
	if err != nil {
		return err
	}
	fmt.Printf("共解出 %d 个文件到 %s\n", n, dest)
	return nil
}

func storeDelete(dir string, opt res.Options, names []string) error {
	shadowed, err := res.Delete(dir, opt, names...)
	if err != nil {
		return err
	}
	for _, p := range shadowed {
		fmt.Println("警告: 根下仍有同名散文件，登记对其不生效，请先移走：", p)
	}
	fmt.Printf("已在最高层登记删除 %d 个名字\n", len(names))
	return nil
}
