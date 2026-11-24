// mpq 命令行工具：查看 / 列表 / 解包 / 打包 / 增删文件 / 版本转换 / 多层资源仓。
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
//	store   <子命令> <数据根> …       多层资源仓，见 storeUsage
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
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
  store   <子命令> <数据根> …       多层资源仓（mpq store 查看子命令）
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
		// (listfile) 与 (deleted) 是格式 / 资源仓的内部元数据，解包时跳过。
		name := e.DisplayName()
		if strings.EqualFold(name, "(listfile)") || strings.EqualFold(name, "(deleted)") {
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
