// fskit 自检入口：对官方 FsKit.dll 复刻（internal/fskit）做一轮可运行演示，
// 覆盖全解文档 §8 验收清单的核心面（INI 方言 round-trip、split 三兄弟、
// GUID 形态、FileSearch 通配、MainConsole CRLF）。
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

func main() {
	fmt.Println("== FsKit replica self-check ==")

	// 1) INI round-trip（含隐式空名节 / 注释 / 无 '=' 行）
	doc := fskit.NewIniDocument(filepath.Join(os.TempDir(), "fskit_selfcheck.ini"))
	doc.Inner().LoadFromString("[Sec]\r\n; cmt\r\nk = v \r\nnoeq\r\n")
	fmt.Printf("read k=%q noeq=%q sections=%v\n",
		doc.ReadString("SEC", "K", ""), doc.ReadString("Sec", "noeq", "MISS"), doc.GetSectionList())
	doc.WriteInteger("Sec", "n", 7)
	doc.AddString("Sec", "dup", "1")
	doc.AddString("Sec", "dup", "2")
	fmt.Printf("save=%v items=%v values=%v\n",
		doc.SaveToFile(), doc.GetItemList("Sec"), doc.GetItemValueList("Sec"))

	// 2) split 三兄弟
	fmt.Printf("split_string(\"a,b,,c,\", ',') = %v\n", fskit.SplitString("a,b,,c,", ","))
	fmt.Printf("split_file_name(\"a.b.c\") = %v\n", fskit.SplitFileName("a.b.c"))
	fmt.Printf("split_file_path(\"cmd/fskit/main.go\") = %v\n", fskit.SplitFilePath("cmd/fskit/main.go"))

	// 3) 全局函数
	fmt.Printf("gen_unique_name() = %s\n", fskit.GenUniqueName())
	fmt.Printf("get_full_path(\".\") = %s\n", fskit.GetFullPath("."))
	if t, ok := fskit.GetFileTimeStruct(os.Args[0]); ok {
		fmt.Printf("get_file_time(self) = %04d-%02d-%02d %02d:%02d:%02d (UTC)\n",
			t.Year, t.Month, t.Day, t.Hour, t.Minute, t.Second)
	}

	// 4) FileSearch 通配方言
	dir := os.TempDir()
	fs := fskit.NewFileSearch()
	if fs.SearchFile(dir+string(os.PathSeparator), "ini") {
		fmt.Printf("search *.ini in tmp: %d hit(s)\n", fs.GetFileCount())
	}

	// 5) MainConsole
	mc := fskit.NewMainConsole()
	mc.SetColor("LightGreen")
	mc.WriteLine("MainConsole WriteLine (CRLF)")

	fmt.Println("== all checks printed ==")
}
