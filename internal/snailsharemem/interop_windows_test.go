//go:build windows

package snailsharemem

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// pythonInterop 生成对拍脚本：按官方 0x814 布局+位图协议直接读写字节
// （等价 CSharingDataExWrite 写路径 / CreateReaderID 注册 / Read+ReadEnd
// 消费路径）。原生 SharingMemory.dll 为 32 位，64 位 python 加载失败时
// 仅跳过 GetFactory 冒烟（协议对拍不依赖原生代码）。
const pythonInterop = `
import ctypes, struct, sys, time
MODE, NAME, PAYLOAD = sys.argv[1], sys.argv[2], sys.argv[3]
DBG = r"F:\Game\Online\9yin\BGserverCJ\Debug"
k32 = ctypes.WinDLL("kernel32", use_last_error=True)
for fn in ("OpenFileMappingW", "CreateFileMappingW", "MapViewOfFile"):
    getattr(k32, fn).restype = ctypes.c_void_p  # 64 位下指针不可被 c_int 截断
k32.UnmapViewOfFile.argtypes = [ctypes.c_void_p]
k32.CloseHandle.argtypes = [ctypes.c_void_p]
PAGE_RW, FM_RW = 4, 0x06  # 官方 0xF001F 在 Win11 24H2 视图写入即 AV，取最小读写掩码
SIZE = 0x814

try:
    dll = ctypes.WinDLL(DBG + r"\SharingMemory.dll")
    dll.GetFactory.restype = ctypes.c_void_p
    p = dll.GetFactory()
    assert 0x10000000 < p < 0x10200000, hex(p)
except OSError:
    print("SMOKE: dll missing (skip)")

wname = ctypes.create_unicode_buffer("SNAILEDITOR_" + NAME)

def open_create():
    h = k32.CreateFileMappingW(ctypes.c_void_p(-1), None, PAGE_RW, 0, SIZE, wname)
    assert h, "create failed %d" % ctypes.get_last_error()
    p = k32.MapViewOfFile(h, FM_RW, 0, 0, 0)
    assert p, "map failed"
    return h, p

def open_existing():
    h = k32.OpenFileMappingW(0xF001F, False, wname)
    assert h, "open failed %d" % ctypes.get_last_error()
    p = k32.MapViewOfFile(h, FM_RW, 0, 0, 0)
    assert p, "map failed"
    return h, p

def buf_at(p):
    return (ctypes.c_char * SIZE).from_address(p)

if MODE == "write":
    h, p = open_existing()
    buf = buf_at(p)
    data = PAYLOAD.encode()
    struct.pack_into("<I", buf, 0x804, len(data))
    ctypes.memmove(ctypes.addressof(buf) + 4, data, len(data))
    reg = struct.unpack_from("<I", buf, 0x810)[0]
    struct.pack_into("<I", buf, 0x80C, reg)
    buf[0x808] = 1
    k32.UnmapViewOfFile(p); k32.CloseHandle(h)
    print("OK write reg=%d" % reg)
elif MODE == "register":
    h, p = open_create()
    buf = buf_at(p)
    reg = struct.unpack_from("<I", buf, 0x810)[0]
    rid = next(i for i in range(32) if not (reg >> i) & 1)
    struct.pack_into("<I", buf, 0x810, reg | (1 << rid))
    k32.UnmapViewOfFile(p); k32.CloseHandle(h)
    print("OK register rid=%d" % rid)
else:  # consume <payload> <rid>
    rid = int(sys.argv[4])
    h, p = open_create()
    buf = buf_at(p)
    for _ in range(200):
        if buf[0x808]:
            break
        time.sleep(0.1)
    assert buf[0x808], "no frame"
    n = struct.unpack_from("<I", buf, 0x804)[0]
    got = ctypes.string_at(ctypes.addressof(buf) + 4, n)
    assert got.decode() == PAYLOAD, got
    readable = struct.unpack_from("<I", buf, 0x80C)[0]
    assert readable & (1 << rid), "readable bit for rid %d" % rid
    struct.pack_into("<I", buf, 0x80C, readable & ~(1 << rid))
    if struct.unpack_from("<I", buf, 0x80C)[0] == 0:
        buf[0x808] = 0
    k32.UnmapViewOfFile(p); k32.CloseHandle(h)
    print("OK consume rid=%d" % rid)
`

// TestInteropPythonReaderWriter 跨进程对拍（Go↔Python 按官方协议）：
// ① Go 读端创建+注册 → Python 写端写帧 → Go 读走清算；
// ② Python 注册读者 → Go 写帧 → Python 消费+清算
// （官方语义：写帧之后注册的读者收不到该帧，故必须先注册）。
func TestInteropPythonReaderWriter(t *testing.T) {
	py, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python 不可用")
	}
	name := "SvnUpdateFinish_itest"

	rd, err := OpenRead(name)
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	id, err := rd.Box().CreateReaderID()
	if err != nil {
		t.Fatal(err)
	}
	out1 := runPython(t, py, name, "write", "frame-from-python")
	if !strings.Contains(out1, "OK write") {
		t.Fatalf("python write 失败: %s", out1)
	}
	if got := rd.Box().Read(id); string(got) != "frame-from-python" {
		t.Fatalf("Go 读端帧不符: %q", got)
	}
	rd.Box().ReadEnd(id)
	if !rd.Box().GetWritable() {
		t.Fatal("清算后应可写")
	}

	outReg := runPython(t, py, name, "register", "")
	rid := extractInt(t, outReg, "rid=")

	wr, err := OpenWrite(name)
	if err != nil {
		t.Fatal(err)
	}
	defer wr.Close()
	if _, err := wr.Box().Write([]byte("frame-from-go")); err != nil {
		t.Fatal(err)
	}
	out2 := runPythonArgs(t, py, "consume", name, "frame-from-go", strconv.Itoa(rid))
	if !strings.Contains(out2, "OK consume") {
		t.Fatalf("python consume 失败: %s", out2)
	}
	// Go 读端(id=0)仍持可读位——官方语义：python 清算后帧仍为其保留。
	if got := rd.Box().Read(id); string(got) != "frame-from-go" {
		t.Fatalf("Go 读端应仍可读: %q", got)
	}
	if wr.Box().GetWritable() {
		t.Fatal("尚有读者未清算，不应可写")
	}
	rd.Box().ReadEnd(id)
	if !wr.Box().GetWritable() {
		t.Fatal("全部清算后应可写")
	}
}

func runPython(t *testing.T, py, name, mode, payload string) string {
	t.Helper()
	return runPythonArgs(t, py, mode, name, payload)
}

func runPythonArgs(t *testing.T, py string, args ...string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "interop.py")
	if err := os.WriteFile(script, []byte(pythonInterop), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(py, append([]string{"-I", script}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, out)
	}
	return string(out)
}

func extractInt(t *testing.T, out, marker string) int {
	t.Helper()
	i := strings.Index(out, marker)
	if i < 0 {
		t.Fatalf("marker %q 缺失: %s", marker, out)
	}
	v := 0
	for j := i + len(marker); j < len(out) && out[j] >= '0' && out[j] <= '9'; j++ {
		v = v*10 + int(out[j]-'0')
	}
	return v
}
