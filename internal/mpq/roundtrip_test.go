package mpq

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

// 创建 / 修改 / 转换的往返测试。

func sampleFiles() []Addition {
	return []Addition{
		{Name: "hello.txt", Data: []byte("hello mpq\r\nsecond line\r\n")},
		{Name: "empty.dat", Data: nil},
		{Name: "docs/readme.md", Data: bytes.Repeat([]byte("## readme\n资源打包测试文本\r\n"), 80)},
		{Name: "data/pattern.bin", Data: patternBytes(300_000, 0x12345678)},
		{Name: "blob/noise.bin", Data: noiseBytes(200_000, 0xDEADBEEF)},
		{Name: "deep/dir/tree/leaf.txt", Data: bytes.Repeat([]byte("deep leaf"), 500)},
	}
}

func TestCreateModifyReadRoundtrip(t *testing.T) {
	for _, version := range []FormatVersion{V1, V2, V3, V4} {
		t.Run("v"+strings.Repeat("I", int(version)+1), func(t *testing.T) {
			base, err := Create(CreateOptionsOfVersion(version))
			if err != nil {
				t.Fatal(err)
			}
			empty, err := OpenBytes(base)
			if err != nil {
				t.Fatal(err)
			}
			if empty.Version() != int(version)+1 || len(empty.Entries()) != 0 {
				t.Fatalf("空归档校验失败：version=%d entries=%d", empty.Version(), len(empty.Entries()))
			}
			empty.Close()

			samples := sampleFiles()
			result, err := Modify(base, nil, samples)
			if err != nil {
				t.Fatal(err)
			}
			// 导入名经 sanitize 规范化：/ 分隔符转为 MPQ 惯用的 \。
			var expectedAdded []string
			for _, s := range samples {
				name, err := SanitizeName(s.Name)
				if err != nil {
					t.Fatal(err)
				}
				expectedAdded = append(expectedAdded, name)
			}
			if strings.Join(result.Report.Added, "|") != strings.Join(expectedAdded, "|") {
				t.Fatalf("added = %v，应为 %v", result.Report.Added, expectedAdded)
			}

			archive, err := OpenBytes(result.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			if archive.Version() != int(version)+1 {
				t.Fatalf("版本 = %d，应为 %d", archive.Version(), int(version)+1)
			}
			for _, s := range samples {
				data, err := archive.ReadFile(s.Name)
				if err != nil {
					t.Fatalf("读取 %s 失败：%v", s.Name, err)
				}
				if !equalBytes(data, s.Data) {
					t.Fatalf("%s 内容不一致", s.Name)
				}
			}
			// 6 个导入文件 + 重建的 (listfile)。
			if got := len(archive.Entries()); got != len(samples)+1 {
				t.Fatalf("条目数 = %d，应为 %d", got, len(samples)+1)
			}
		})
	}
}

func TestReplaceAndDeleteRoundtrip(t *testing.T) {
	base, err := Create(DefaultCreateOptions())
	if err != nil {
		t.Fatal(err)
	}
	v1, err := Modify(base, nil, []Addition{
		{Name: "a.txt", Data: []byte("version 1")},
		{Name: "b.bin", Data: patternBytes(9000, 1)},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 替换 a.txt、删除 b.bin。
	result, err := Modify(v1.Bytes, []Deletion{DeleteByName("b.bin")}, []Addition{
		{Name: "a.txt", Data: []byte("version 2 with more text")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Report.Replaced, "|") != "a.txt" {
		t.Fatalf("replaced = %v，应为 [a.txt]", result.Report.Replaced)
	}
	if strings.Join(result.Report.Deleted, "|") != "b.bin" {
		t.Fatalf("deleted = %v，应为 [b.bin]", result.Report.Deleted)
	}

	archive, err := OpenBytes(result.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if got, _ := archive.ReadFile("a.txt"); string(got) != "version 2 with more text" {
		t.Fatalf("a.txt = %q", got)
	}
	if archive.HasFile("b.bin") {
		t.Fatal("b.bin 应已删除")
	}
}

func TestConvertAcrossVersions(t *testing.T) {
	base, err := Create(DefaultCreateOptions())
	if err != nil {
		t.Fatal(err)
	}
	keep := noiseBytes(30_000, 7)
	v1, err := Modify(base, nil, []Addition{
		{Name: "keep.bin", Data: keep},
		{Name: "text/a.txt", Data: []byte("convert me")},
	})
	if err != nil {
		t.Fatal(err)
	}

	current := v1.Bytes
	for _, target := range []FormatVersion{V1, V2, V3, V4} {
		converted, err := Convert(current, target)
		if err != nil {
			t.Fatalf("转换到 v%d 失败：%v", int(target)+1, err)
		}
		archive, err := OpenBytes(converted.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if archive.Version() != int(target)+1 {
			t.Fatalf("版本 = %d，应为 %d", archive.Version(), int(target)+1)
		}
		if got, err := archive.ReadFile("keep.bin"); err != nil || !equalBytes(got, keep) {
			t.Fatalf("keep.bin 内容不一致（err=%v）", err)
		}
		if got, _ := archive.ReadFile("text/a.txt"); string(got) != "convert me" {
			t.Fatalf("text/a.txt = %q", got)
		}
		archive.Close()
		current = converted.Bytes
	}
	// 转换回源自身版本为无操作。
	again, err := Convert(current, V4)
	if err != nil {
		t.Fatal(err)
	}
	if !equalBytes(again.Bytes, current) {
		t.Fatal("转换回源自身版本应为无操作（字节不变）")
	}
}

func TestKeyV2ReencryptedEncodingIsSelfConsistent(t *testing.T) {
	// 手工验证 encodeFreshReencrypted 的产物可按 readEntry 的磁盘布局
	// 约定解码回原文（offset/flags 来自编码时约定）。
	content := patternBytes(20_000, 0x0ABCDEF0)
	const pos = 0x100
	enc := encodeFreshReencrypted("fixed.bin", content, 4096, pos)
	if enc.flags&fileKeyV2 == 0 {
		t.Fatal("编码产物应带 KEY_V2 标志")
	}
	decoded := decodeKeyV2(t, enc.bytes, len(content), 4096, pos, "fixed.bin")
	if !equalBytes(decoded, content) {
		t.Fatal("key-v2 编码解码后内容不一致")
	}
}

// decodeKeyV2 按 readEntry 的磁盘布局约定解码 encodeFreshReencrypted 的产物。
func decodeKeyV2(t *testing.T, block []byte, fsize, sectorSize int, pos int64, name string) []byte {
	t.Helper()
	key := FileKey(name, true, uint64(pos), uint32(fsize))
	nsec := (fsize + sectorSize - 1) / sectorSize
	tableLen := (nsec + 1) * 4
	table := append([]byte(nil), block[:tableLen]...)
	DecryptBytes(table, key-1)
	offsets := make([]int, nsec+1)
	for i := range offsets {
		offsets[i] = int(binary.LittleEndian.Uint32(table[i*4:]))
	}
	if offsets[0] != tableLen {
		t.Fatalf("偏移表首项 = %d，应为 %d", offsets[0], tableLen)
	}

	out := make([]byte, fsize)
	outPos := 0
	for i := 0; i < nsec; i++ {
		raw := append([]byte(nil), block[offsets[i]:offsets[i+1]]...)
		DecryptBytes(raw, key+uint32(i))
		expected := min(fsize-i*sectorSize, sectorSize)
		sector := raw
		if len(raw) < expected {
			var err error
			sector, err = decompressMasked(raw, expected)
			if err != nil {
				t.Fatal(err)
			}
		}
		take := min(len(sector), expected)
		copy(out[outPos:], sector[:take])
		outPos += take
	}
	if outPos != fsize {
		t.Fatalf("解码长度 = %d，应为 %d", outPos, fsize)
	}
	return out
}

func TestSanitizeName(t *testing.T) {
	for _, bad := range []string{"(listfile)", "", ".."} {
		if _, err := SanitizeName(bad); err == nil {
			t.Fatalf("SanitizeName(%q) 应报错", bad)
		}
	}
	vectors := map[string]string{
		"a/b/./c.txt": `a\b\c.txt`,
		"/../x.txt":   "x.txt",
	}
	for in, want := range vectors {
		got, err := SanitizeName(in)
		if err != nil {
			t.Fatalf("SanitizeName(%q) 报错：%v", in, err)
		}
		if got != want {
			t.Fatalf("SanitizeName(%q) = %q，应为 %q", in, got, want)
		}
	}
}

func TestNotAnArchiveIsRejected(t *testing.T) {
	garbage := bytes.Repeat([]byte{0xAB}, 64)
	if _, err := OpenBytes(garbage); err == nil {
		t.Fatal("垃圾字节应被拒绝")
	}
}
