package res

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq"
)

// 层的构建、删除登记与合并视图解包。

// BuildOptions 配置 BuildLayer。
type BuildOptions struct {
	// Prefix 层文件名族（PrefixTest / PrefixData），必填。
	Prefix string
	// Number 目标层号；0 = 现有最大层号 + 1（无层则 1）。
	Number int
	// Replace 允许整层重建已存在的目标层。未开启而目标层已存在时报错——
	// 重建后该层只含本次源提供的内容。
	Replace bool
	// Source 源目录；"" = 数据根自身（收集根下散文件）。
	Source string
	// Exclude 排除规则：/ 分隔的 glob（path.Match 语法），命中文件本身
	// 或其任一级上级目录即排除。层容器文件始终不是资源。
	Exclude []string
	// Format MPQ 格式版本 1–4；0 = 4（v4 带全量 MD5 校验和）。
	Format int
	// Full 关闭去重：与下层内容相同的文件也写入本层。
	Full bool
	// DeleteMissing 源中缺失而下层存在的名字在本层登记删除（源须是
	// 完整树；只提供改动文件的源不要开）。
	DeleteMissing bool
	// Prune 成功并回读校验后，删除已并入的源散文件（含与下层相同而被
	// 去重的那些）。仅源为数据根自身时有效；被排除的文件不动。
	Prune bool
}

// BuildResult 描述一次 BuildLayer 的结果。
type BuildResult struct {
	File     string // 目标层文件路径
	Number   int
	Replaced bool // 整层重建了既有同号层
	Changed  bool // 是否写出了层文件（无变化时为 false）
	Added    int
	Skipped  int // 与下层内容相同而未写入
	Excluded int // 被排除规则挡掉
	Deleted  []string
	Pruned   int
	Bytes    int64 // 层文件大小
}

// BuildLayer 把源目录下的文件打成一个层：默认只收与下层（层号更小的
// 各层）内容不同的文件，所以"根下散文件 → 下一层"得到的就是增量层；
// 没有任何下层时即全量层。写出前回读校验，原子替换目标文件。
func BuildLayer(root string, opt BuildOptions) (*BuildResult, error) {
	prefix := canonicalPrefix(opt.Prefix)
	if prefix == "" {
		return nil, fmt.Errorf("res: 构建层须指定前缀（%s 之一）", strings.Join(Families(), " / "))
	}
	if !IsPrefix(prefix) {
		return nil, fmt.Errorf("res: 无效的层文件前缀 %q（应为 %s 之一）",
			opt.Prefix, strings.Join(Families(), " / "))
	}
	if opt.Number < 0 || opt.Number > maxLayerCount {
		return nil, fmt.Errorf("res: 无效的层号 %d", opt.Number)
	}
	if opt.Format < 0 || opt.Format > 4 {
		return nil, fmt.Errorf("res: 无效的格式版本 %d（应为 1–4）", opt.Format)
	}
	format := opt.Format
	if format == 0 {
		format = 4
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(abs)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("res: 数据根不是目录：%s", root)
	}
	// 层文件可能被路径式访问入口（同进程）打开着；Windows 上占用时无法替换。
	ResetPathCache()

	families, err := scanLayers(root)
	if err != nil {
		return nil, err
	}
	number := opt.Number
	if number == 0 {
		for _, f := range families[prefix] {
			number = max(number, f.number)
		}
		number++
	}
	target := filepath.Join(root, layerFileName(prefix, number))
	res := &BuildResult{File: target, Number: number}
	if _, err := os.Stat(target); err == nil {
		if !opt.Replace {
			return nil, fmt.Errorf("res: 层 %s 已存在；整层重建须显式开启 Replace", filepath.Base(target))
		}
		res.Replaced = true
	}

	// 下层视图：去重与缺失比对的基准（不含散文件层）。
	below, err := openStore(root, prefix, number, true)
	if err != nil {
		return nil, err
	}
	defer below.Close()

	src := root
	if opt.Source != "" {
		if src, err = filepath.Abs(opt.Source); err != nil {
			return nil, err
		}
		src = filepath.Clean(src)
	}
	prunable := opt.Prune && samePath(src, root)

	var adds []mpq.Addition
	var prune []string
	seen := map[string]string{} // 小写资源名 → 首见路径：防仅大小写不同的同名
	source := map[string]bool{} // 源内名字（小写键）
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || isLayerFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		relName := filepath.ToSlash(rel)
		if excluded(relName, opt.Exclude) {
			res.Excluded++
			return nil
		}
		name, err := mpq.SanitizeName(relName)
		if err != nil {
			return fmt.Errorf("res: %s 不能作为资源名：%w", p, err)
		}
		if !mpq.SameName(name, Normalize(relName)) {
			return fmt.Errorf("res: %s 越出源目录，不能作为资源名", p)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		key := NormalizeLower(name)
		source[key] = true
		if !opt.Full {
			if size, _, ok := below.Lookup(name); ok && size == int64(len(data)) {
				if old, _, err := below.read(name); err == nil && bytes.Equal(old, data) {
					res.Skipped++
					if prunable {
						prune = append(prune, p)
					}
					return nil
				}
			}
		}
		if first, dup := seen[key]; dup {
			return fmt.Errorf("res: %s 与 %s 仅大小写不同，归档按名寻址无法区分", relName, first)
		}
		seen[key] = relName
		adds = append(adds, mpq.Addition{Name: name, Data: data})
		if prunable {
			prune = append(prune, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 删除名单：保留本层原有的，并入源中缺失的，去掉本次重新收录的。
	tombstones := map[string]bool{}
	if res.Replaced {
		if old, err := mpq.Open(target); err == nil {
			tombstones, _ = loadDeleted(old, filepath.Base(target))
			old.Close()
		}
	}
	if opt.DeleteMissing {
		idx := below.index()
		for _, key := range idx.keys {
			if !source[key] {
				tombstones[key] = true
			}
		}
	}
	for _, add := range adds {
		delete(tombstones, NormalizeLower(add.Name))
	}
	if len(tombstones) > 0 {
		res.Deleted = sortedKeys(tombstones)
	}

	if len(adds) == 0 && len(tombstones) == 0 {
		// 没有可写的内容（MPQ 归档不能为空）：保留原层，不落新文件。
		// 与下层相同的散文件已是冗余，照常修剪。
		res.Pruned, err = pruneLoose(root, prune)
		return res, err
	}

	additions := append([]mpq.Addition(nil), adds...)
	if len(tombstones) > 0 {
		additions = append(additions, mpq.Addition{Name: deletedFile, Data: deletedList(res.Deleted)})
	}
	base, err := mpq.Create(mpq.CreateOptionsOfVersion(mpq.FormatVersion(format - 1)))
	if err != nil {
		return nil, err
	}
	result, err := mpq.Modify(base, nil, additions)
	if err != nil {
		return nil, err
	}
	if err := writeFileVerified(target, result.Bytes, additions); err != nil {
		return nil, err
	}
	res.Changed, res.Added, res.Bytes = true, len(adds), int64(len(result.Bytes))
	res.Pruned, err = pruneLoose(root, prune)
	return res, err
}

// pruneLoose 删除已并入层（或与下层内容相同）的源散文件，并清掉因此
// 变空的目录。
func pruneLoose(root string, files []string) (int, error) {
	n := 0
	for _, p := range files {
		if err := os.Remove(p); err != nil {
			return n, fmt.Errorf("res: 删除散文件失败：%w", err)
		}
		n++
	}
	if n > 0 {
		pruneEmptyDirs(root, files)
	}
	return n, nil
}

// Delete 在最高层登记删除名字：合并视图中自此不存在。散文件层不受
// 登记约束——根下若仍有同名散文件，须先移走，返回值 shadowed 会列出
// 它们。没有任何层文件时无处登记，报错。本函数改写最高层文件：调用方
// 须先关闭自己持有的该数据根的 Store（Windows 上占用着无法替换）。
func Delete(root string, opt Options, names ...string) (shadowed []string, err error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("res: 未给出要删除的名字")
	}
	norms := make([]string, 0, len(names))
	for _, name := range names {
		norm, err := mpq.SanitizeName(name)
		if err != nil {
			return nil, err
		}
		norms = append(norms, norm)
	}

	// 开仓只为定位最高层与被散文件遮蔽的名字，用完即关。
	store, err := OpenWith(root, opt)
	if err != nil {
		return nil, err
	}
	layers := store.Layers()
	if len(layers) == 0 {
		store.Close()
		return nil, fmt.Errorf("res: %s 没有层文件，无处登记删除", store.Dir())
	}
	top := layers[len(layers)-1]
	for _, norm := range norms {
		if local, ok := store.localFileOf(norm); ok {
			shadowed = append(shadowed, local)
		}
	}
	store.Close()
	ResetPathCache()

	deleted := map[string]bool{}
	if archive, err := mpq.Open(top.Path); err == nil {
		if old, err := loadDeleted(archive, top.File); err == nil {
			for key := range old {
				deleted[key] = true
			}
		}
		archive.Close()
	}
	var deletions []mpq.Deletion
	for _, norm := range norms {
		deleted[NormalizeLower(norm)] = true
		deletions = append(deletions, mpq.DeleteByName(norm))
	}
	deletions = append(deletions, mpq.DeleteByName(deletedFile))

	raw, err := os.ReadFile(top.Path)
	if err != nil {
		return nil, err
	}
	lines := sortedKeys(deleted)
	result, err := mpq.Modify(raw, deletions,
		[]mpq.Addition{{Name: deletedFile, Data: deletedList(lines)}})
	if err != nil {
		return nil, err
	}
	if err := writeFileVerified(top.Path, result.Bytes,
		[]mpq.Addition{{Name: deletedFile, Data: deletedList(lines)}}); err != nil {
		return nil, err
	}
	return shadowed, nil
}

// Unpack 把合并视图（散文件 > 高层 > 低层）写成散文件树。prefix 非空时
// 只写以它开头的名字（不分大小写，/ 或 \ 分隔）。返回写出个数。
func Unpack(s *Store, dest string, prefix string) (int, error) {
	files, err := s.List()
	if err != nil {
		return 0, err
	}
	prefix = strings.ToLower(Normalize(prefix))
	written := 0
	for _, f := range files {
		if prefix != "" && !strings.HasPrefix(NormalizeLower(f.Name), prefix) {
			continue
		}
		data, err := s.Read(f.Name)
		if err != nil {
			return written, err
		}
		target := filepath.Join(dest, filepath.FromSlash(strings.ReplaceAll(f.Name, `\`, "/")))
		// 归档里的名字来自外部：越出目标目录的一律拒绝，不做任何写出。
		if rel, err := filepath.Rel(dest, target); err != nil || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return written, fmt.Errorf("res: 资源名越出目标目录：%s", f.Name)
		}
		if dir := filepath.Dir(target); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return written, err
			}
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// -- 内部 ----------------------------------------------------------------------

// deletedList 生成 (deleted) 内容：每行一个名字，CRLF 结尾。
func deletedList(names []string) []byte {
	return []byte(strings.Join(names, "\r\n") + "\r\n")
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// excluded 报告相对路径（/ 分隔）是否命中任一排除规则：规则对路径
// 本身及其每一级上级目录做 path.Match。
func excluded(rel string, rules []string) bool {
	for p := rel; ; {
		for _, rule := range rules {
			if ok, err := path.Match(strings.Trim(rule, "/"), p); err == nil && ok {
				return true
			}
		}
		i := strings.LastIndexByte(p, '/')
		if i < 0 {
			return false
		}
		p = p[:i]
	}
}

func samePath(a, b string) bool {
	rel, err := filepath.Rel(a, b)
	return err == nil && rel == "."
}

// writeFileVerified 先写临时文件，重新打开逐个回读比对后才原子替换
// 目标；任何一步失败都清理临时文件并保留原目标。
func writeFileVerified(target string, data []byte, additions []mpq.Addition) error {
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	archive, err := mpq.Open(tmp)
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("res: 写出的层 %s 自检打不开：%w", filepath.Base(tmp), err)
	}
	for _, add := range additions {
		got, err := archive.ReadFile(add.Name)
		if err != nil || !bytes.Equal(got, add.Data) {
			archive.Close()
			os.Remove(tmp)
			return fmt.Errorf("res: 写出的层 %s 自检不一致：%s", filepath.Base(target), add.Name)
		}
	}
	archive.Close()
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("res: 替换 %s 失败（文件被占用？）：%w", filepath.Base(target), err)
	}
	return nil
}

// pruneEmptyDirs 自下而上清掉因删除散文件而变空的目录（不含数据根）。
func pruneEmptyDirs(root string, files []string) {
	dirs := map[string]bool{}
	for _, f := range files {
		for d := filepath.Dir(f); !dirs[d] && !samePath(d, root); d = filepath.Dir(d) {
			dirs[d] = true
		}
	}
	ordered := sortedKeys(dirs)
	// 深的先删：子目录空了父目录才可能空。
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, d := range ordered {
		if !samePath(d, root) {
			os.Remove(d) // 非空目录删不掉，忽略错误
		}
	}
}
