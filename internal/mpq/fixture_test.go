package mpq

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 用 Rust mpq 库生成的夹具做跨实现对拍：
// 读取 gen_v1..v4 / enc_v1 / mini_v1 并校验内容。

// fixture 读取 testdata/mpq 下的夹具文件。
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mpq", name))
	if err != nil {
		t.Fatalf("读取夹件 %s 失败：%v", name, err)
	}
	return data
}

// manifestSha 解析 manifest.txt（名字<TAB>sha256<TAB>大小）。
func manifestSha(t *testing.T) map[string]string {
	t.Helper()
	text := string(fixture(t, "manifest.txt"))
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(line))
		out[parts[0]] = parts[1]
	}
	return out
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestReadsRustGeneratedArchives(t *testing.T) {
	for _, archiveName := range []string{"gen_v1", "gen_v2", "gen_v3", "gen_v4"} {
		t.Run(archiveName, func(t *testing.T) {
			manifest := manifestSha(t)
			archive, err := OpenBytes(fixture(t, archiveName+".mpq"))
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()

			wantVersion := int(archiveName[5] - '0')
			if archive.Version() != wantVersion {
				t.Fatalf("版本 = %d，应为 %d", archive.Version(), wantVersion)
			}
			for name, wantSha := range manifest {
				if !archive.HasFile(name) {
					t.Fatalf("%s 应存在", name)
				}
				data, err := archive.ReadFile(name)
				if err != nil {
					t.Fatalf("读取 %s 失败：%v", name, err)
				}
				if got := sha256Hex(data); got != wantSha {
					t.Fatalf("%s sha256 = %s，应为 %s", name, got, wantSha)
				}
			}
			// 空文件读取为 0 字节。
			if data, err := archive.ReadFile("empty.dat"); err != nil || len(data) != 0 {
				t.Fatalf("empty.dat 应读出 0 字节（err=%v len=%d）", err, len(data))
			}
		})
	}
}

func TestReadsHandBuiltEncryptedArchive(t *testing.T) {
	// enc_v1 由 Rust 手工构造：名字密钥加密 / key-v2（压缩与不可压缩）、
	// sector CRC、PKWARE implode、明文单块。
	archive, err := OpenBytes(fixture(t, "enc_v1.mpq"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if archive.Version() != 1 {
		t.Fatalf("版本 = %d，应为 1", archive.Version())
	}

	if got, err := archive.ReadFile("plain.txt"); err != nil ||
		string(got) != "plain stored single-unit file\n" {
		t.Fatalf("plain.txt 内容不符（err=%v）", err)
	}

	// secret.txt：名字密钥 + zlib 单块加密。
	secret, err := archive.ReadFile("secret.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) != 3000 {
		t.Fatalf("secret.txt 长度 = %d，应为 3000", len(secret))
	}

	// fixed.bin：key-v2 加密 + 压缩 sector（偏移表 key-1、sector key+i、dword 填充）。
	fixed, err := archive.ReadFile("fixed.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed) != 13_000 {
		t.Fatalf("fixed.bin 长度 = %d，应为 13000", len(fixed))
	}

	// rand.bin：key-v2 加密 + 不可压缩固定偏移 sector。
	randData, err := archive.ReadFile("rand.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(randData) != 9_500 {
		t.Fatalf("rand.bin 长度 = %d，应为 9500", len(randData))
	}

	// crc.bin：带 sector 校验和，不应产生 CRC 告警。
	crcData, err := archive.ReadFile("crc.bin")
	if err != nil {
		t.Fatal(err)
	}
	if len(crcData) != 12_000 {
		t.Fatalf("crc.bin 长度 = %d，应为 12000", len(crcData))
	}
	if warnings := archive.ReadWarnings(); len(warnings) != 0 {
		t.Fatalf("不应有读取告警：%v", warnings)
	}

	// pk.txt：旧式 PKWARE implode 单块。
	pk, err := archive.ReadFile("pk.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(pk, fixture(t, "pk_text.orig")) {
		t.Fatal("pk.txt 内容与参考原文不一致")
	}

	// 与 Rust 夹具生成器相同的确定性内容生成器对拍。
	if !equalBytes(secret, patternBytes(3000, 0x0BADF00D)) {
		t.Fatal("secret.txt 内容与确定性生成器不符")
	}
	if !equalBytes(fixed, patternBytes(13_000, 0x0F110F11)) {
		t.Fatal("fixed.bin 内容与确定性生成器不符")
	}
	if !equalBytes(crcData, patternBytes(12_000, 0xCCCC0001)) {
		t.Fatal("crc.bin 内容与确定性生成器不符")
	}
	if !equalBytes(randData, noiseBytes(9_500, 0x5EED5EED)) {
		t.Fatal("rand.bin 内容与确定性生成器不符")
	}
}

func TestReadsMiniV1Fixture(t *testing.T) {
	archive, err := OpenBytes(fixture(t, "mini_v1.mpq"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	found := false
	for _, name := range archive.FileNames() {
		if name == "war3map.j" {
			found = true
		}
	}
	if !found {
		t.Fatalf("(listfile) 应包含 war3map.j，实得 %v", archive.FileNames())
	}
	data, err := archive.ReadFile("war3map.j")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "// dummy map script\n" {
		t.Fatalf("war3map.j 内容不符：%q", data)
	}
}

// patternBytes 与 Rust 夹具生成器一致的半随机可压缩内容（xorshift 游程）。
func patternBytes(n int, seed uint32) []byte {
	x := seed | 1
	out := make([]byte, n)
	pos := 0
	for pos < n {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		run := 1 + int(x&0x3F)
		b := byte(x >> 8)
		for i := 0; i < run && pos < n; i++ {
			out[pos] = b
			pos++
		}
	}
	return out
}

// noiseBytes 与 Rust 夹具生成器一致的不可压缩内容。
func noiseBytes(n int, seed uint32) []byte {
	x := seed | 1
	out := make([]byte, n)
	for i := range out {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		out[i] = byte(x >> 16)
	}
	return out
}
