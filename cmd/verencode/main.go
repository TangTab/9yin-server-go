// verencode 是官方 GameEncode.bat（FsLoader.exe + FsKit+FsComm+FsVerEnc.dll +
// ..\Lua\GameEncode 脚本）的 Go 等价宿主：调用 internal/fsverenc 生成
// FsVerify/FsVerCli 同族源码三件套。
//
// 用法：
//
//	verencode -id 1 -complexity 8 -seed 0x5eed -out ver1
//
// seed 缺省取当前时间（对应官方 srand(time(NULL))）。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fsverenc"
)

func main() {
	id := flag.Int("id", 1, "版本编号（ver%%d 目录与文件名前缀）")
	complexity := flag.Int("complexity", 8, "复杂度 4..0x80（GameEncode::Encode 门）")
	seed := flag.Uint("seed", uint(time.Now().Unix()), "rand 种子（官方为 time(NULL)，复现用）")
	out := flag.String("out", "", "输出目录（默认 ver<id>）")
	flag.Parse()

	if err := fsverenc.Encode(*id, *complexity); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dir := *out
	if dir == "" {
		dir = fmt.Sprintf("ver%d", *id)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	g := fsverenc.NewGenerator(uint32(*seed))
	var accountKey [16]byte
	copy(accountKey[:], g.Cond16(*id)) // 结构级：官方账号键由发射器随机产出

	files := map[string]string{
		fmt.Sprintf("%dMes.h", *id):      g.EmitMesH(accountKey),
		fmt.Sprintf("%dEncode.h", *id):   g.EmitEncodeH(),
		fmt.Sprintf("%dEncode.cpp", *id): g.EmitEncodeCpp(),
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", p)
	}
	fmt.Printf("cond16[%d] = %s\n", *id, g.Cond16(*id))
}
