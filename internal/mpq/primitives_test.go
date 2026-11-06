package mpq

import (
	"bytes"
	"testing"
)

// 基础算法对拍：storm 加密/哈希常量、Jenkins 参考向量、PKWARE 参考向量。
// 向量均取自参考实现的单元测试（自身来自 Rust mpq 库）。

func TestStormHashMatchesStormlibConstants(t *testing.T) {
	// 命中 StormLib 文档常量即证明 crypt table 与哈希函数逐位精确。
	if got := HashString("(hash table)", hashFileKey); got != 0xC3AF3770 {
		t.Fatalf("(hash table) 哈希 = %#x，应为 0xC3AF3770", got)
	}
	if got := HashString("(block table)", hashFileKey); got != 0xEC83B3A3 {
		t.Fatalf("(block table) 哈希 = %#x，应为 0xEC83B3A3", got)
	}
}

func TestStormEncryptDecryptRoundtrip(t *testing.T) {
	original := make([]byte, 100)
	for i := range original {
		original[i] = byte(i*7 + 3)
	}
	buf := append([]byte(nil), original...)
	EncryptBytes(buf, 0x12345678)
	if bytes.Equal(buf, original) {
		t.Fatal("加密后数据不应与原文相同")
	}
	DecryptBytes(buf, 0x12345678)
	if !bytes.Equal(buf, original) {
		t.Fatal("解密后应还原为原文")
	}
}

func TestStormNameNormalization(t *testing.T) {
	want := []byte{'A', 'B', 'C', '\\', 'D', 'E', 'F', '.', 'T', 'X', 'T'}
	if got := NormalizeName("abc/def.txt"); !bytes.Equal(got, want) {
		t.Fatalf("规范化结果 = %q，应为 %q", got, want)
	}
	if HashString(`ABC\DEF.txt`, 1) != HashString("abc/DEF.txt", 1) {
		t.Fatal("等价名字的哈希应一致")
	}
}

func TestJenkinsReferenceValues(t *testing.T) {
	// 与 StormLib HashStringJenkins 一致（lower 表、pc=2、pb=1）。
	vectors := map[string]uint64{
		"(listfile)":    0x3bc43bb0b2f3866a,
		"":              0xdeadbef1deadbef2,
		"abc":           0xcbc05d12b47025a5,
		"war3map.j":     0x5252117e4beda79d,
		"exactly12chr":  0x97b18367a6e9e2a8,
		"exactly13chrs": 0x4e7480ccc6359c94,
	}
	for name, want := range vectors {
		if got := FileNameHash(name); got != want {
			t.Fatalf("FileNameHash(%q) = %#x，应为 %#x", name, got, want)
		}
	}
}

func TestPkwareExplodesReferenceVectors(t *testing.T) {
	vectors := [][2]string{
		{"pk_text.bin", "pk_text.orig"},
		{"pk_rand.bin", "pk_rand.orig"},
		{"pk_runs.bin", "pk_runs.orig"},
	}
	for _, v := range vectors {
		stream := fixture(t, v[0])
		plain := fixture(t, v[1])
		out := bytes.NewBuffer(make([]byte, 0, len(plain)))
		if err := Explode(stream, out, len(plain)); err != nil {
			t.Fatalf("%s: %v", v[0], err)
		}
		if !bytes.Equal(out.Bytes(), plain) {
			t.Fatalf("%s 解压结果与参考原文不一致", v[0])
		}
	}
}

func TestPkwareRejectsBadStreams(t *testing.T) {
	bad := [][]byte{
		{},
		{0, 9, 0, 0, 0}, // 非法字典大小
		{7, 4, 0, 0, 0}, // 非法模式
	}
	for _, b := range bad {
		if err := Explode(b, &bytes.Buffer{}, 64); err == nil {
			t.Fatalf("坏流 %v 应报错", b)
		}
	}
}

func TestZlibRoundtripViaMask(t *testing.T) {
	plain := bytes.Repeat([]byte("hello hello hello compressed data 1234567890"), 4)
	z := zlibCompress(plain)
	block := append([]byte{mZlib}, z...)
	got, err := decompressMasked(block, len(plain))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("带方法字节解压结果与原文不一致")
	}

	// 缺方法字节回退：整个块是普通 zlib 流。
	got, err = decompressMasked(z, len(plain))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("回退路径解压结果与原文不一致")
	}
}
