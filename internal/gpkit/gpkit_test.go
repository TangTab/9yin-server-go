package gpkit_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/gpkit"
)

// mkCodeBin 按官方配对规则布置目录:
// codeDir/X.bin = 代码载荷;codeDir/basename(dataDir)/X.bin = 数据。
func mkCodeBin(t *testing.T, codeDir, dataDir, name string, payload, data []byte) {
	t.Helper()
	if err := os.MkdirAll(codeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(codeDir, filepath.Base(dataDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeDir, name+".bin"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeDir, filepath.Base(dataDir), name+".bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func gpkPayload(gen, chk uint16) []byte {
	b := make([]byte, 8)
	copy(b, "GPK1")
	binary.LittleEndian.PutUint16(b[4:], gen)
	binary.LittleEndian.PutUint16(b[6:], chk)
	return b
}

func TestCreateAndDefaults(t *testing.T) {
	s := gpkit.Create()
	if s.GroupCount() != 0 {
		t.Fatalf("fresh svr group count = %d, want 0", s.GroupCount())
	}
	if s.Count() != 0 {
		t.Fatalf("fresh svr count = %d, want 0", s.Count())
	}
	if got := s.RandomPick(); got != -1 {
		t.Fatalf("RandomPick on empty = %d, want -1", got)
	}
	if _, err := s.GetData(0); err == nil {
		t.Fatal("GetData on empty should fail")
	}
}

func TestInitLoadAndChain(t *testing.T) {
	dir := t.TempDir()
	codeDir := filepath.Join(dir, "DynCodeBin", "Server")
	dataDir := filepath.Join(dir, "DynCodeBin", "Client")
	data := append([]byte{1, 2, 3, 4}, bytes.Repeat([]byte{'A'}, 60)...)
	mkCodeBin(t, codeDir, dataDir, "TestA", gpkPayload(0, 1), data)

	s := gpkit.Create()
	n := s.Init(codeDir, dataDir)
	if n != 1 {
		t.Fatalf("Init loaded %d entries, want 1", n)
	}
	if s.GroupCount() != 1 || s.Count() != 1 {
		t.Fatalf("group=%d count=%d, want 1/1", s.GroupCount(), s.Count())
	}
	idx := s.RandomPick()
	if idx != 0 {
		t.Fatalf("RandomPick = %d, want 0", idx)
	}
	got, err := s.GetData(uint32(idx) << 16)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("GetData mismatch: %x", got[:8])
	}
	// Call1/Call2 走内置检查(确定性应答);同参同码应稳定。
	v1, err := s.Call1(0xDEAD0001, 0xDEAD0002, uint32(idx)<<16)
	if err != nil {
		t.Fatal(err)
	}
	v1b, _ := s.Call1(0xDEAD0001, 0xDEAD0002, uint32(idx)<<16)
	if v1 != v1b {
		t.Fatalf("Call1 not deterministic: %x vs %x", v1, v1b)
	}
	v2, err := s.Call2(0, 0xC4, uint32(idx)<<16)
	if err != nil {
		t.Fatal(err)
	}
	if v2 == 0 {
		t.Fatal("Call2 returned 0")
	}
	// 越界索引
	if _, err := s.Call1(0, 0, 0xFF<<16); err == nil {
		t.Fatal("out-of-range codeid should fail")
	}
}

func TestInitBadDir(t *testing.T) {
	s := gpkit.Create()
	if got := s.Init(filepath.Join(t.TempDir(), "nope"), "x"); got != -1 {
		t.Fatalf("Init(bad) = %d, want -1", got)
	}
}

func TestInitRejectsBadPayload(t *testing.T) {
	dir := t.TempDir()
	codeDir := filepath.Join(dir, "S")
	dataDir := filepath.Join(dir, "C")
	mkCodeBin(t, codeDir, dataDir, "Bad", []byte("garbage!"), nil)
	s := gpkit.Create()
	if got := s.Init(codeDir, dataDir); got != 0 {
		t.Fatalf("Init(bad payload) = %d, want 0", got)
	}
}

func TestMultipleEntriesAndPick(t *testing.T) {
	dir := t.TempDir()
	codeDir := filepath.Join(dir, "S")
	dataDir := filepath.Join(dir, "C")
	for i, nm := range []string{"A", "B", "C"} {
		mkCodeBin(t, codeDir, dataDir, nm, gpkPayload(uint16(i), uint16(69-i)), []byte{byte(i)})
	}
	s := gpkit.Create()
	if n := s.Init(codeDir, dataDir); n != 3 {
		t.Fatalf("Init = %d, want 3", n)
	}
	seen := map[int]bool{}
	for i := 0; i < 50; i++ {
		idx := s.RandomPick()
		if idx < 0 || idx > 2 {
			t.Fatalf("RandomPick out of range: %d", idx)
		}
		seen[idx] = true
	}
	if len(seen) != 3 {
		t.Fatalf("RandomPick not covering: %v", seen)
	}
}

func TestCustomLoader(t *testing.T) {
	dir := t.TempDir()
	codeDir := filepath.Join(dir, "S")
	dataDir := filepath.Join(dir, "C")
	mkCodeBin(t, codeDir, dataDir, "X", []byte{1}, nil)
	s := gpkit.Create()
	s.SetLoader(func(code []byte) (gpkit.CheckFunc, gpkit.CheckFunc, error) {
		f := func(a, b uint32) uint32 { return a + b }
		return f, f, nil
	})
	if n := s.Init(codeDir, dataDir); n != 1 {
		t.Fatalf("Init = %d, want 1", n)
	}
	// 官方转发语义:fn1(b, codeid&0xFFFF) → 3+0
	v, err := s.Call1(2, 3, 0)
	if err != nil || v != 3 {
		t.Fatalf("Call1 = %d, %v; want 3, nil", v, err)
	}
}

func TestConfParse(t *testing.T) {
	ini := []byte(strings.Join([]string{
		"# comment",
		"CheckMem = 1",
		"CheckProcName=0",
		"  CheckWindow = 1  ",
		"CheckModName = 1",
		"CheckCode = 1",
		"CheckDriver = 0",
		"CheckScanMod = 1",
	}, "\n"))
	c := gpkit.ParseConf(ini)
	want := gpkit.Conf{CheckMem: true, CheckProcName: false, CheckWindow: true,
		CheckModName: true, CheckCode: true, CheckDriver: false, CheckScanMod: true}
	if c != want {
		t.Fatalf("ParseConf = %+v, want %+v", c, want)
	}
	s := gpkit.Create()
	s.LoadConf(ini)
	if s.Conf() != want {
		t.Fatal("LoadConf mismatch")
	}
}

func TestAuthDataLimit(t *testing.T) {
	s := gpkit.Create()
	s.LoadAuthData(bytes.Repeat([]byte{0xAA}, 20000))
	if got := len(s.AuthData()); got != 0x3E80 {
		t.Fatalf("AuthData len = %x, want 3E80", got)
	}
}

func TestHackWarning(t *testing.T) {
	for n := 10; n <= 79; n++ {
		w, err := gpkit.HackWarning(n)
		if err != nil {
			t.Fatalf("HackWarning(%d) err: %v", n, err)
		}
		if want := fmt.Sprintf("hack %d warning", n); w != want {
			t.Fatalf("HackWarning(%d) = %q", n, w)
		}
	}
	if _, err := gpkit.HackWarning(9); err == nil {
		t.Fatal("HackWarning(9) should fail")
	}
	if _, err := gpkit.HackWarning(80); err == nil {
		t.Fatal("HackWarning(80) should fail")
	}
}

func TestCheckAuthReply(t *testing.T) {
	mk := func(idx uint16, maskOK bool) []byte {
		pkt := make([]byte, gpkit.AuthReplySize)
		binary.LittleEndian.PutUint16(pkt[0:], idx)
		m := binary.LittleEndian.Uint32(pkt[0:4])
		if maskOK {
			binary.LittleEndian.PutUint32(pkt[4:8], ^m)
		} else {
			binary.LittleEndian.PutUint32(pkt[4:8], m)
		}
		return pkt
	}
	// 长度错误
	if _, err := gpkit.CheckAuthReply(make([]byte, 10), 1, nil); err == nil ||
		err.Error() != "packet  length error!" {
		t.Fatalf("len err = %v", err)
	}
	// 序号不符
	if _, err := gpkit.CheckAuthReply(mk(3, true), 2, nil); err == nil ||
		err.Error() != "CheckAuthReply Idx Exception!" {
		t.Fatalf("idx err = %v", err)
	}
	// 掩码错误
	if _, err := gpkit.CheckAuthReply(mk(2, false), 2, nil); err == nil ||
		err.Error() != "packet mask check error!" {
		t.Fatalf("mask err = %v", err)
	}
	// 正常:idx=2 走校验表(内置默认通过)
	if _, err := gpkit.CheckAuthReply(mk(2, true), 2, nil); err != nil {
		t.Fatalf("ok case err = %v", err)
	}
	// 窗口期直接放行(0x0A < idx < 0x15)
	if _, err := gpkit.CheckAuthReply(mk(0x12, true), 0x12, nil); err != nil {
		t.Fatalf("window case err = %v", err)
	}
	// 自定义校验表:返回非 0 → check error
	tbl := make([]gpkit.CheckFunc, 0x15)
	tbl[2] = func(a, b uint32) uint32 { return 1 }
	if _, err := gpkit.CheckAuthReply(mk(2, true), 2, tbl); err == nil ||
		err.Error() != "check error" {
		t.Fatalf("check err = %v", err)
	}
}

func TestBuiltinCheckDeterministic(t *testing.T) {
	// 同一 id 的检查对同输入必须稳定;不同 id 输出分布应不同。
	var first uint32
	for i := 0; i < 70; i++ {
		s := gpkit.Create()
		_ = s
	}
	f := builtinCheckRef(t, 0)
	a := f(0x1234, 0x5678)
	if a != f(0x1234, 0x5678) {
		t.Fatal("builtin check not deterministic")
	}
	first = a
	if first == builtinCheckRef(t, 1)(0x1234, 0x5678) {
		t.Fatal("check id 0 and 1 collide")
	}
}

// builtinCheckRef 经由默认载荷加载器取得内置检查。
func builtinCheckRef(t *testing.T, id uint16) gpkit.CheckFunc {
	t.Helper()
	fn1, _, err := gpkit.DefaultPayloadLoader(gpkPayload(id, id))
	if err != nil {
		t.Fatal(err)
	}
	return fn1
}

func TestRelease(t *testing.T) {
	dir := t.TempDir()
	codeDir := filepath.Join(dir, "S")
	dataDir := filepath.Join(dir, "C")
	mkCodeBin(t, codeDir, dataDir, "A", gpkPayload(0, 0), []byte{1})
	s := gpkit.Create()
	if n := s.Init(codeDir, dataDir); n != 1 {
		t.Fatalf("Init = %d", n)
	}
	s.Release()
	if s.Count() != 0 || s.GroupCount() != 0 {
		t.Fatal("Release should clear state")
	}
	if got := s.RandomPick(); got != -1 {
		t.Fatalf("RandomPick after Release = %d", got)
	}
}
