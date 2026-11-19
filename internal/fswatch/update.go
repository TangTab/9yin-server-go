package fswatch

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// chunkSize 官方分块上限 0x10000（SendCustomFile/SendDirData 同值）。
const chunkSize = 0x10000

// FileInfo 官方 file_data_t（name/size/mtime；FileEqual 三元组比对）。
type FileInfo struct {
	Name    string
	Size    int64
	ModTime int64 // Unix 秒（官方 FILETIME 转 time_t 面）
}

// DirInfo 官方 dir_data_t（0x88 字节：name(+0)/path(+0x30)/dirs(+0x60)/
// files(+0x74)）。
type DirInfo struct {
	Name  string
	Path  string
	Dirs  []*DirInfo
	Files []FileInfo
}

// ReadDirInfo 官方 CUpdate::ReadDirInfo(0x81c80)：path\name 递归扫描，
// 跳 "."/".."，目录在前、文件在后（ FindFirstFile 迭代序；复刻按名排序
// 保证确定性）。
func ReadDirInfo(root, rel string) (*DirInfo, error) {
	d := &DirInfo{Name: rel, Path: root}
	full := root
	if rel != "" {
		full = filepath.Join(root, rel)
	}
	ents, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		if n == "." || n == ".." {
			continue
		}
		fi, err := os.Stat(filepath.Join(full, n))
		if err != nil {
			continue
		}
		if fi.IsDir() {
			sub, err := ReadDirInfo(full, n)
			if err != nil {
				continue
			}
			sub.Path = full
			d.Dirs = append(d.Dirs, sub)
		} else {
			d.Files = append(d.Files, FileInfo{Name: n, Size: fi.Size(), ModTime: fi.ModTime().Unix()})
		}
	}
	return d, nil
}

// StoreArchive 官方 CUpdate::StoreArchive(0x82850) 文本序列化形态：
// 行式 "D <rel>"/"F <rel> <size> <mtime>"，目录先序遍历。
// 官方为 TStoreArchive 二进制容器，本复刻取其外显等价（get_dir 的消费方
// 是运维控制台，行文本即可 round-trip）。
func (d *DirInfo) StoreArchive() []byte {
	var b strings.Builder
	var walk func(node *DirInfo, prefix string)
	walk = func(node *DirInfo, prefix string) {
		rel := node.Name
		if prefix != "" && rel != "" {
			rel = prefix + "\\" + rel
		} else if prefix != "" {
			rel = prefix
		}
		if rel != "" {
			fmt.Fprintf(&b, "D %s\r\n", rel)
		}
		for i := range node.Files {
			fmt.Fprintf(&b, "F %s %d %d\r\n", filepath.Join(rel, node.Files[i].Name), node.Files[i].Size, node.Files[i].ModTime)
		}
		for _, sub := range node.Dirs {
			walk(sub, rel)
		}
	}
	walk(d, "")
	return []byte(b.String())
}

// DiffDirInfo 官方 CUpdate::DiffDirInfo(0x82f10)/ParseDiffDirInfo：远端
// want 与本地 have 对比（FileEqual = 名+size+mtime），产出待建目录与
// 待更新文件（相对 root 的 \ 分隔路径）。
func DiffDirInfo(root string, want *DirInfo) (dirs []string, files []string) {
	var have *DirInfo
	if h, err := ReadDirInfo(root, ""); err == nil {
		have = h
	}
	haveFiles := map[string]FileInfo{}
	haveDirs := map[string]bool{}
	if have != nil {
		var collect func(node *DirInfo, prefix string)
		collect = func(node *DirInfo, prefix string) {
			for i := range node.Files {
				rel := node.Files[i].Name
				if prefix != "" {
					rel = prefix + "\\" + rel
				}
				haveFiles[rel] = node.Files[i]
			}
			for _, sub := range node.Dirs {
				subPrefix := sub.Name
				if prefix != "" {
					subPrefix = prefix + "\\" + sub.Name
				}
				haveDirs[subPrefix] = true
				collect(sub, subPrefix)
			}
		}
		collect(have, "")
	}
	dirs = nil
	var walk func(node *DirInfo, prefix string)
	walk = func(node *DirInfo, prefix string) {
		for i := range node.Files {
			rel := node.Files[i].Name
			if prefix != "" {
				rel = prefix + "\\" + rel
			}
			h, ok := haveFiles[rel]
			if !ok || h.Size != node.Files[i].Size || h.ModTime != node.Files[i].ModTime {
				files = append(files, rel)
			}
		}
		for _, sub := range node.Dirs {
			subPrefix := sub.Name
			if prefix != "" {
				subPrefix = prefix + "\\" + sub.Name
			}
			if !haveDirs[subPrefix] {
				dirs = append(dirs, subPrefix)
			}
			walk(sub, subPrefix)
		}
	}
	walk(want, "")
	return dirs, files
}

// sendFileBlocks 官方 SendCustomFile(0x6f630) 骨架：fopen rb → fseek/ftell
// → 64KB 循环 → varlist [头串… "succeed" total offset chunklen UserData]
// → send 回调。失败时由调用方回 "send file error"。
func sendFileBlocks(head []string, r io.Reader, send func(*VarList) bool) bool {
	data, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	total := len(data)
	for off := 0; off < total || total == 0; off += chunkSize {
		end := off + chunkSize
		if end > total {
			end = total
		}
		chunk := data[off:end]
		l := &VarList{}
		for _, h := range head {
			l.AddString(h)
		}
		l.AddString("succeed")
		l.AddInt(int64(total))
		l.AddInt(int64(off))
		l.AddInt(int64(len(chunk)))
		l.AddUserData(chunk)
		if !send(l) {
			return false
		}
		if end >= total {
			break
		}
	}
	return true
}

// sendBufferBlocks 官方 SendDirData(0x6f9d0)：StoreArchive 缓冲 → 同一分块
// 骨架下发。
func sendBufferBlocks(head []string, data []byte, send func(*VarList) bool) bool {
	total := len(data)
	for off := 0; off < total || total == 0; off += chunkSize {
		end := off + chunkSize
		if end > total {
			end = total
		}
		chunk := data[off:end]
		l := &VarList{}
		for _, h := range head {
			l.AddString(h)
		}
		l.AddString("succeed")
		l.AddInt(int64(total))
		l.AddInt(int64(off))
		l.AddInt(int64(len(chunk)))
		l.AddUserData(chunk)
		if !send(l) {
			return false
		}
		if end >= total {
			break
		}
	}
	return true
}

// UpdateSession 官方 set_file 接收侧缓冲（'file cache error' 面）：
// 同一 (server,file) 的分块拼装，齐 size 后 MD5 校验落盘。
type UpdateSession struct {
	root    string
	file    string
	size    int64
	modTime int64
	md5     string
	buf     []byte
}

func (s *UpdateSession) feed(off int64, chunk []byte) {
	end := off + int64(len(chunk))
	if end > int64(len(s.buf)) {
		grow := make([]byte, end)
		copy(grow, s.buf)
		s.buf = grow
	}
	copy(s.buf[off:end], chunk)
}

func (s *UpdateSession) complete() bool { return int64(len(s.buf)) >= s.size }

// flush 官方 set_file 收尾：md5 verify → wb 落盘 → 恢复 mtime
// （错误串 'md5 verify code not match'/'create file failed'/
// 'write file data failed'/'set file time failed' → 'succeed'）。
func (s *UpdateSession) flush(destRoot string) (string, error) {
	sum := md5.Sum(s.buf[:s.size])
	if !strings.EqualFold(hex.EncodeToString(sum[:]), s.md5) {
		return "md5 verify code not match", fmt.Errorf("md5 mismatch")
	}
	full := filepath.Join(destRoot, filepath.FromSlash(s.file))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "create file failed", err
	}
	if err := os.WriteFile(full, s.buf[:s.size], 0o644); err != nil {
		return "write file data failed", err
	}
	if s.modTime > 0 {
		t := time.Unix(s.modTime, 0)
		if err := os.Chtimes(full, t, t); err != nil {
			return "set file time failed", err
		}
	}
	return "succeed", nil
}
