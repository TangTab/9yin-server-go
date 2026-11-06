// mpq 命令行工具：查看 / 列表 / 解包 / 打包 / 增删文件 / 版本转换 / 三层资源仓。
//
// 用法: mpq <命令> [参数]
//
//	info    <mpq>                     头部与条目概览
//	list    <mpq>                     列出文件（名字 / 大小 / 压缩后 / 标志）
//	extract <mpq> [destDir] [名字…]   解包全部或指定文件（默认 ./<mpq名>/）
//	pack    <dir> <out.mpq> [-v 1-4]  目录递归打包为新归档（默认 v1）
//	add     <mpq> <本地文件=归档名>…  添加 / 替换文件（写回原 mpq）
//	remove  <mpq> <名字>…            删除文件（写回原 mpq）
//	convert <mpq> <out.mpq> -v 1-4   转换头部版本
//	store   info|list|read <资源根>…  三层资源仓（本地 > patch > base）
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/local/9yin-go-server/internal/mpq"
	"github.com/local/9yin-go-server/internal/mpq/res"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "错误："+err.Error())
		os.Exit(1)
	}
}

func usage(out *os.File) {
	fmt.Fprint(out, `用法: mpq <命令> [参数]
  info    <mpq>                     头部与条目概览
  list    <mpq>                     列出文件
  extract <mpq> [destDir] [名字…]   解包全部或指定文件
  pack    <dir> <out.mpq> [-v 1-4]  目录递归打包为新归档（默认 v1）
  add     <mpq> <本地文件=归档名>…  添加 / 替换文件
  remove  <mpq> <名字>…            删除文件
  convert <mpq> <out.mpq> -v 1-4   转换头部版本
  store   info|list|read <资源根>…  三层资源仓（本地 > patch > base）
`)
}

func run(args []string) error {
	switch args[1] {
	case "info":
		return info(args[2])
	case "list":
		return list(args[2])
	case "extract":
		return extract(args[2], args)
	case "pack":
		return pack(args)
	case "add":
		return add(args)
	case "remove":
		return remove(args)
	case "convert":
		return convert(args)
	case "store":
		return store(args)
	default:
		usage(os.Stderr)
		os.Exit(2)
		return nil
	}
}

func info(path string) error {
	archive, err := mpq.Open(path)
	if err != nil {
		return err
	}
	defer archive.Close()
	fmt.Println("文件        :", path)
	fmt.Println("格式版本    : v" + strconv.Itoa(archive.Version()))
	fmt.Println("sector 大小 :", archive.SectorSize())
	fmt.Printf("归档偏移    : 0x%x\n", archive.ArchiveOffset())
	fmt.Println("文件条目    :", len(archive.Entries()))
	fmt.Println("已知名字    :", len(archive.FileNames()))
	for _, w := range archive.Warnings() {
		fmt.Println("警告        :", w)
	}
	return nil
}

func list(path string) error {
	archive, err := mpq.Open(path)
	if err != nil {
		return err
	}
	defer archive.Close()
	fmt.Printf("%-10s %-10s %-8s %s\n", "大小", "压缩后", "标志", "名字")
	for _, e := range archive.Entries() {
		fmt.Printf("%-10d %-10d %-8s %s\n", e.Size, e.CompressedSize, flagsSummary(&e), e.DisplayName())
	}
	for _, w := range archive.Warnings() {
		fmt.Println("警告:", w)
	}
	return nil
}

func flagsSummary(e *mpq.FileEntry) string {
	var sb strings.Builder
	if !e.Exists() {
		sb.WriteByte('!')
	}
	if e.IsSingleUnit() {
		sb.WriteByte('U')
	}
	if e.IsCompressed() {
		sb.WriteByte('C')
	}
	if e.IsEncrypted() {
		sb.WriteByte('E')
	}
	return sb.String()
}

func extract(path string, args []string) error {
	var dest string
	var wanted []string
	if len(args) > 3 && !strings.HasPrefix(args[3], "-") {
		dest = args[3]
		wanted = args[4:]
	} else {
		name := filepath.Base(path)
		if dot := strings.LastIndex(name, "."); dot >= 0 {
			name = name[:dot]
		}
		dest = name
		wanted = args[3:]
	}

	archive, err := mpq.Open(path)
	if err != nil {
		return err
	}
	defer archive.Close()

	var entries []mpq.FileEntry
	for _, e := range archive.Entries() {
		// (listfile) 是格式内部元数据，解包时跳过。
		if strings.EqualFold(e.DisplayName(), "(listfile)") {
			continue
		}
		if len(wanted) > 0 {
			match := false
			for _, w := range wanted {
				if mpq.SameName(w, e.DisplayName()) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		entries = append(entries, e)
	}
	for _, e := range entries {
		data, err := archive.ReadEntry(e)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(strings.ReplaceAll(e.DisplayName(), `\`, "/")))
		if dir := filepath.Dir(target); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("写出 %s (%d 字节)\n", target, len(data))
	}
	fmt.Printf("共解包 %d 个文件到 %s\n", len(entries), dest)
	return nil
}

func pack(args []string) error {
	dir, out := args[2], args[3]
	version := mpq.V1
	for i := 4; i < len(args)-1; i++ {
		if args[i] == "-v" {
			v, err := versionOf(args[i+1])
			if err != nil {
				return err
			}
			version = v
		}
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("不是目录：%s", dir)
	}

	var files []mpq.Addition
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取 %s 失败：%w", path, err)
		}
		name, err := mpq.SanitizeName(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		files = append(files, mpq.Addition{Name: name, Data: data})
		return nil
	})
	if err != nil {
		return err
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	if len(files) == 0 {
		return fmt.Errorf("目录为空：%s", dir)
	}

	base, err := mpq.Create(mpq.CreateOptionsOfVersion(version))
	if err != nil {
		return err
	}
	result, err := mpq.Modify(base, nil, files)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, result.Bytes, 0o644); err != nil {
		return err
	}
	fmt.Printf("打包 %d 个文件 → %s（v%d）\n", len(files), out, int(version)+1)
	return nil
}

func add(args []string) error {
	path := args[2]
	var additions []mpq.Addition
	for _, arg := range args[3:] {
		eq := strings.LastIndex(arg, "=")
		if eq <= 0 {
			return fmt.Errorf("参数应为 <本地文件=归档名>：%s", arg)
		}
		data, err := os.ReadFile(arg[:eq])
		if err != nil {
			return err
		}
		additions = append(additions, mpq.Addition{Name: arg[eq+1:], Data: data})
	}
	return modifyInPlace(path, nil, additions)
}

func remove(args []string) error {
	path := args[2]
	var deletions []mpq.Deletion
	for _, name := range args[3:] {
		deletions = append(deletions, mpq.DeleteByName(name))
	}
	return modifyInPlace(path, deletions, nil)
}

func modifyInPlace(path string, deletions []mpq.Deletion, additions []mpq.Addition) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result, err := mpq.Modify(data, deletions, additions)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, result.Bytes, 0o644); err != nil {
		return err
	}
	report(result.Report)
	return nil
}

func convert(args []string) error {
	in, out := args[2], args[3]
	version := mpq.V3
	for i := 4; i < len(args)-1; i++ {
		if args[i] == "-v" {
			v, err := versionOf(args[i+1])
			if err != nil {
				return err
			}
			version = v
		}
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	result, err := mpq.Convert(data, version)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, result.Bytes, 0o644); err != nil {
		return err
	}
	fmt.Printf("转换 %s → %s（v%d）\n", in, out, int(version)+1)
	return nil
}

func versionOf(s string) (mpq.FormatVersion, error) {
	switch strings.ToLower(s) {
	case "1", "v1":
		return mpq.V1, nil
	case "2", "v2":
		return mpq.V2, nil
	case "3", "v3":
		return mpq.V3, nil
	case "4", "v4":
		return mpq.V4, nil
	default:
		return 0, fmt.Errorf("无效版本 %q（应为 1–4）", s)
	}
}

// == 三层资源仓 ==============================================================

func store(args []string) error {
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "用法: mpq store info|list|read <资源根目录> [名字/前缀]")
		os.Exit(2)
	}
	dir, cmd := args[3], args[2]
	st, err := res.Open(dir)
	if err != nil {
		return err
	}
	defer st.Close()

	switch cmd {
	case "info":
		abs, _ := filepath.Abs(dir)
		fmt.Println("资源根    :", abs)
		fmt.Printf("base 条目 : %d（版本 %s）\n", st.BaseCount(), st.BaseVersion())
		if st.HasPatch() {
			fmt.Printf("patch     : %d 条目（版本 %s）\n", st.PatchCount(), st.PatchVersion())
		} else {
			fmt.Println("patch     : 无")
		}
		files, err := st.List()
		if err != nil {
			return err
		}
		fmt.Println("合并清单  :", len(files), "个文件")
		for _, w := range st.Warnings() {
			fmt.Println("警告      :", w)
		}
	case "list":
		files, err := st.List()
		if err != nil {
			return err
		}
		prefix := ""
		if len(args) > 4 {
			prefix = strings.ToLower(strings.ReplaceAll(args[4], "/", `\`))
		}
		shown := 0
		for _, f := range files {
			if prefix == "" || strings.HasPrefix(strings.ToLower(strings.ReplaceAll(f.Name, "/", `\`)), prefix) {
				fmt.Printf("%-6s %10d  %s\n", f.Layer, f.Size, f.Name)
				shown++
			}
		}
		fmt.Printf("共 %d 个（全量 %d）\n", shown, len(files))
	case "read":
		if len(args) < 5 {
			return fmt.Errorf("用法: mpq store read <资源根目录> <名字>")
		}
		name := args[4]
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
		// 文本文件预览前 200 字节；二进制只显示首行。
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
	default:
		fmt.Fprintln(os.Stderr, "未知子命令："+cmd)
		os.Exit(2)
	}
	return nil
}

func report(r mpq.ModifyReport) {
	for _, n := range r.Added {
		fmt.Println("新增", n)
	}
	for _, n := range r.Replaced {
		fmt.Println("替换", n)
	}
	for _, n := range r.Deleted {
		fmt.Println("删除", n)
	}
	for _, n := range r.Dropped {
		fmt.Println("丢弃", n)
	}
	for _, n := range r.ReEncoded {
		fmt.Println("重编码", n)
	}
}
