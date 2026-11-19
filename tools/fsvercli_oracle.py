# -*- coding: utf-8 -*-
"""FsVerCli.dll unicorn 真值 oracle。

执行官方客户端校验 DLL 的 encode_func_N（2048 个，bool 返回）：
  - 128 个 g_cond_N（恒等映射：cond_N = seed[N/8] 的第 7-(N%8) 位）
  - _CORE_API 需非空（函数入口做非空门）
用法：
    from fsvercli_oracle import CliOracle
    o = CliOracle()
    o.set_cond(seed16)
    out16 = o.encode(n, in16)
"""
import struct
import sys

sys.path.insert(0, __file__.rsplit("\\", 1)[0])
from fsverify_oracle import _build_gdt_entry  # 复用 GDT 构造

from unicorn import UC_ARCH_X86, UC_HOOK_CODE, UC_HOOK_MEM_INVALID, UC_MODE_32, Uc
from unicorn.x86_const import (
    UC_X86_REG_EAX,
    UC_X86_REG_ECX,
    UC_X86_REG_EDX,
    UC_X86_REG_EFLAGS,
    UC_X86_REG_EIP,
    UC_X86_REG_ESP,
    UC_X86_REG_FS_BASE,
)

from pdb_syms import Pdb, iter_syms, S_PUB32

DLL_PATH = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsVerCli.dll"
IMAGE_BASE = 0x10000000
STACK_TOP = 0x00300000
STACK_SIZE = 0x00100000
MAGIC_RET = 0x0F0F0F0F
TEB_ADDR = 0x7FFDF000
GDT_ADDR = 0x7FFDE000
HEAP_ADDR = 0x00400000
HEAP_SIZE = 0x00100000

VA_CORE_API = 0x102E1D28
COND_BASE_NOTE = "cond_N VA 见 cond_va_map（恒等位映射）"


def load_cond_va_map():
    import json

    with open(r"G:\pojie\ai\9yin-server-go\tmp\vercli_cond_va.json") as f:
        return {int(k, 16): v for k, v in json.load(f).items()}


class CliOracle:
    def __init__(self, dll_path=DLL_PATH):
        self.d = open(dll_path, "rb").read()
        self._parse_pe()
        self.uc = Uc(UC_ARCH_X86, UC_MODE_32)
        self._map_image()
        self._map_support()
        self._stub_iat()
        self.encode_addrs = self._load_encode_addrs()
        self.cond_va = load_cond_va_map()
        # md5 thiscall 助手（thunk 一跳）
        self.md5_ctor = self._hop(0x100EFA43)
        self.md5_update = self._hop(0x100EDA8B)
        self.md5_finalize = self._hop(0x100EF9F3)
        self.md5_digest = self._hop(0x100EDA90)
        self._setup_fake_vapi()
        self._setup_fs()
        self._disarm_rtc()

    def _setup_fs(self):
        uc = self.uc
        uc.reg_write(UC_X86_REG_FS_BASE, TEB_ADDR)

    def _disarm_rtc(self):
        """__RTC_CheckEsp（栈检查）补丁为 ret——桩函数不匹配 stdcall 清栈会误报。"""
        va = self._hop(0x100ED991)
        self.uc.mem_write(va, b"\xC3")

    # ---------- PE / 映射（同 fsverify_oracle） ----------
    def _parse_pe(self):
        d = self.d
        pe = struct.unpack_from("<I", d, 0x3C)[0]
        nsec = struct.unpack_from("<H", d, pe + 6)[0]
        optsz = struct.unpack_from("<H", d, pe + 20)[0]
        self.imagebase = struct.unpack_from("<I", d, pe + 24 + 28)[0]
        assert self.imagebase == IMAGE_BASE
        self.secs = []
        so = pe + 24 + optsz
        end = 0
        for i in range(nsec):
            o = so + i * 40
            name = d[o : o + 8].rstrip(b"\0").decode()
            vsz, va, rsz, raw = struct.unpack_from("<IIII", d, o + 8)
            self.secs.append((name, IMAGE_BASE + va, vsz, raw, rsz))
            end = max(end, IMAGE_BASE + va + max(vsz, rsz))
        self.image_end = (end + 0xFFF) & ~0xFFF

    def rva2off(self, rva):
        for n, sva, vsz, raw, rsz in self.secs:
            if raw and sva - IMAGE_BASE <= rva < sva - IMAGE_BASE + max(vsz, rsz):
                return raw + (rva - (sva - IMAGE_BASE))
        return None

    def _map_image(self):
        uc = self.uc
        uc.mem_map(IMAGE_BASE, self.image_end - IMAGE_BASE)
        for name, sva, vsz, raw, rsz in self.secs:
            if raw and rsz:
                uc.mem_write(sva, self.d[raw : raw + rsz])

    def _map_support(self):
        uc = self.uc
        uc.mem_map(STACK_TOP - STACK_SIZE, STACK_SIZE)
        uc.mem_map(HEAP_ADDR, HEAP_SIZE)
        uc.mem_map(TEB_ADDR, 0x1000)
        uc.mem_map(GDT_ADDR, 0x1000)
        uc.mem_write(TEB_ADDR, struct.pack("<I", 0xFFFFFFFF))

    def _stub_iat(self):
        uc = self.uc
        stub = IMAGE_BASE + (self.image_end - IMAGE_BASE) - 0x10
        uc.mem_write(stub, b"\xC3")
        d = self.d
        pe = struct.unpack_from("<I", d, 0x3C)[0]
        imp_rva = struct.unpack_from("<I", d, pe + 24 + 96 + 8)[0]  # import dir rva
        off = self.rva2off(imp_rva)
        i = 0
        while True:
            oft, ts, fc, name_rva, iat_rva = struct.unpack_from("<IIIII", d, off + i * 20)
            if not any((oft, ts, name_rva, iat_rva)):
                break
            j = 0
            while True:
                thunk = struct.unpack_from("<I", d, self.rva2off(oft or iat_rva) + j * 4)[0]
                if not thunk:
                    break
                slot = IMAGE_BASE + iat_rva + j * 4
                if slot < self.image_end:
                    uc.mem_write(slot, struct.pack("<I", stub))
                j += 1
            i += 1

    def _hop(self, va):
        off = self.rva2off(va - IMAGE_BASE)
        if off is not None and self.d[off] == 0xE9:
            return va + 5 + struct.unpack_from("<i", self.d, off + 1)[0]
        return va

    def _load_encode_addrs(self):
        pdb = Pdb(DLL_PATH.replace(".dll", ".pdb"))
        addrs = {}
        for rectype, off, seg, ln, name in iter_syms(pdb.streams[pdb.dbi_header()["symrec"]]):
            if rectype != S_PUB32 or not name.startswith("?encode_func_"):
                continue
            try:
                idx = int(name.split("_")[2].split("@")[0])
            except ValueError:
                continue
            if 1 <= seg <= len(self.secs):
                addrs[idx] = IMAGE_BASE + (self.secs[seg - 1][1] - IMAGE_BASE) + off
        return addrs

    # ---------- 业务 ----------
    def set_cond(self, seed16):
        """seed16 → 128 个 g_cond_N（恒等映射，LSB 先：cond_N = seed[N/8]>>(N%8)&1）。"""
        assert len(seed16) == 16
        for va, bit in self.cond_va.items():
            v = (seed16[bit >> 3] >> (bit & 7)) & 1
            self.uc.mem_write(va, struct.pack("<I", v))

    def _run(self, va, args, timeout=4_000_000):
        uc = self.uc
        sp = STACK_TOP - 0x1000
        for a in reversed(args):
            sp -= 4
            uc.mem_write(sp, struct.pack("<I", a))
        sp -= 4
        uc.mem_write(sp, struct.pack("<I", MAGIC_RET))
        uc.reg_write(UC_X86_REG_ESP, sp)
        uc.emu_start(va, MAGIC_RET, timeout=timeout)
        return uc.reg_read(UC_X86_REG_EAX)

    def _setup_fake_vapi(self):
        """假 VAPI：所有槽返回 4KB 堆缓冲（vt[2] 是临时 scratch 分配；其余需非零跳垃圾字节）。"""
        uc = self.uc
        obj = HEAP_ADDR + 0x800
        vt = HEAP_ADDR + 0xC00
        stub = HEAP_ADDR + 0xE00
        self.vapi_buf = HEAP_ADDR + (HEAP_SIZE - 0x4000)  # 堆区尾部，无需新映射
        # mov eax, vapi_buf; ret
        uc.mem_write(stub, b"\xB8" + struct.pack("<I", self.vapi_buf) + b"\xC3")
        for i in range(64):
            uc.mem_write(vt + i * 4, struct.pack("<I", stub))
        uc.mem_write(obj, struct.pack("<I", vt))
        self.vapi_obj = obj
        # g_isdp（IsDebuggerPresent 指针，EncodeMessageInit 才初始化）→ ret0 桩
        uc.mem_write(0x102E1980, struct.pack("<I", stub))

    def encode(self, idx, in16):
        uc = self.uc
        inbuf = HEAP_ADDR + 0x100
        outbuf = HEAP_ADDR + 0x200
        uc.mem_write(inbuf, bytes(in16[:16]))
        uc.mem_write(outbuf, b"\xEE" * 16)
        uc.mem_write(self.vapi_buf, b"\x00" * 0x1000)  # 清空 VAPI scratch
        # 非空门：in/out/_CORE_API
        uc.mem_write(VA_CORE_API, struct.pack("<I", self.vapi_obj))
        ok = self._run(self.encode_addrs[idx], [inbuf, outbuf])
        return ok, bytes(uc.mem_read(outbuf, 16))


if __name__ == "__main__":
    o = CliOracle()
    print("encode funcs:", len(o.encode_addrs))
    o.set_cond(bytes(16))
    ok, out = o.encode(0, bytes(range(16)))
    print("func_0 cond=0:", ok, out.hex())
    o.set_cond(b"\xff" * 16)
    ok, out = o.encode(0, bytes(range(16)))
    print("func_0 cond=ff:", ok, out.hex())
