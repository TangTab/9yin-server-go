# -*- coding: utf-8 -*-
"""FsVerify.dll unicorn 真值 oracle。

在 x86-32 模拟器里直接执行官方 DLL 的函数（不经 DllMain/CLR）：
encode_func_N（2048 个编码器）、DecodeAccount/DecodePassword、des2key/D2des，
并直接读内存提取 g_encode_order/g_cond_value/g_client_code 表。

用法（库）：
    from fsverify_oracle import Oracle
    o = Oracle()
    o.encode(n, bytes16) -> bytes16
    o.decode_account(ct) -> bytes
    o.decode_password(ct) -> bytes

用法（命令行自检）：
    python fsverify_oracle.py selftest
"""
import struct
import sys

from unicorn import (
    UC_ARCH_X86,
    UC_HOOK_CODE,
    UC_HOOK_MEM_INVALID,
    UC_MODE_32,
    Uc,
)
from unicorn.x86_const import (
    UC_X86_REG_EAX,
    UC_X86_REG_EBP,
    UC_X86_REG_ECX,
    UC_X86_REG_EDX,
    UC_X86_REG_EFLAGS,
    UC_X86_REG_EIP,
    UC_X86_REG_ESP,
    UC_X86_REG_FS,
)

DLL_PATH = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsVerify.dll"
IMAGE_BASE = 0x10000000
STACK_TOP = 0x00300000
STACK_SIZE = 0x00100000
MAGIC_RET = 0x0F0F0F0F
TEB_ADDR = 0x7FFDF000
GDT_ADDR = 0x7FFDE000
HEAP_ADDR = 0x00400000
HEAP_SIZE = 0x00100000

# PDB 符号解析的函数地址（rva）
RVA = {
    "encode_dispatcher": 0x200A50,
    "des2key": 0xD69C0,
    "Ddes": 0xD6A60,
    "D2des": 0xD6B40,
    "DecodeAccount": 0x210B10,
    "DecodePassword": 0x210D50,
    "md5_ctor": 0xD7980,
    "md5_update": 0xD7CB0,
    "md5_finalize": 0xD7E20,
    "md5_digest": 0xD79E0,
}
ENCODE_FUNC_BASE_RVA = None  # 由 PDB 符号给出，见 load_encode_addrs


def _build_gdt_entry(base, limit, access, flags):
    return struct.pack(
        "<IHBBLHBBH",
        limit & 0xFFFF,
        base & 0xFFFF,
        (base >> 16) & 0xFF,
        access,
        flags | ((limit >> 16) & 0x0F),
        (base >> 24) & 0xFF,
        0,
        0,
        0,
    )


class Oracle:
    def __init__(self, dll_path=DLL_PATH):
        self.d = open(dll_path, "rb").read()
        self._parse_pe()
        self.uc = Uc(UC_ARCH_X86, UC_MODE_32)
        self._map_image()
        self._map_support()
        self._setup_fs()
        self._stub_iat()
        self._hook()
        self.encode_addrs = self._load_encode_addrs()

    # ---------- PE / 映射 ----------
    def _parse_pe(self):
        d = self.d
        pe = struct.unpack_from("<I", d, 0x3C)[0]
        nsec = struct.unpack_from("<H", d, pe + 6)[0]
        optsz = struct.unpack_from("<H", d, pe + 20)[0]
        self.imagebase = struct.unpack_from("<I", d, pe + 24 + 28)[0]
        assert self.imagebase == IMAGE_BASE, hex(self.imagebase)
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
        # TEB: ExceptionList=0xFFFFFFFF（空链），其余清零
        uc.mem_write(TEB_ADDR, struct.pack("<I", 0xFFFFFFFF))

    def _setup_fs(self):
        # 不动 GDTR（破坏默认平坦段），直接写 FS base → 假 TEB
        uc = self.uc
        from unicorn.x86_const import UC_X86_REG_FS_BASE

        uc.reg_write(UC_X86_REG_FS_BASE, TEB_ADDR)

    def _stub_iat(self):
        # IAT 槽全部指向 ret 桩，杜绝未解析导入被调用时崩飞
        uc = self.uc
        stub = IMAGE_BASE + (self.image_end - IMAGE_BASE) - 0x10
        uc.mem_write(stub, b"\xC3")
        d = self.d
        pe = struct.unpack_from("<I", d, 0x3C)[0]
        optsz = struct.unpack_from("<H", d, pe + 20)[0]
        ddoff = pe + 24 + 96
        imp_rva, imp_sz = struct.unpack_from("<II", d, ddoff)
        if not imp_rva:
            return
        # 找 .idata 节内偏移
        raw = None
        for name, sva, vsz, praw, rsz in self.secs:
            if name == ".idata" and praw:
                raw = praw
                break
        if raw is None:
            return
        iat_rva = struct.unpack_from("<I", d, ddoff + 8)[0]  # IAT dir
        base = imp_rva
        while True:
            ent = struct.unpack_from("<I", d, raw + (base - imp_rva) + 0)[0] if False else None
            off = raw + (base - imp_rva)
            fields = struct.unpack_from("<5I", d, off)
            if fields[0] == 0 and fields[1] == 0:
                break
            orig_first, tds, fwd, name_rva, first_thunk = fields
            iat_va = IMAGE_BASE + first_thunk
            iat_raw = None
            for name, sva, vsz, praw, rsz in self.secs:
                if name == ".idata" and praw:
                    iat_raw = praw + (first_thunk - (sva - IMAGE_BASE))
                    break
            n = 0
            while True:
                v = struct.unpack_from("<I", d, iat_raw + n * 4)[0]
                if v == 0:
                    break
                uc.mem_write(iat_va + n * 4, struct.pack("<I", stub))
                n += 1
            base += 20

    def _hook(self):
        uc = self.uc
        self.hookHandles = []

        def on_invalid(uc_, access, address, size, value, user_data):
            eip = uc_.reg_read(UC_X86_REG_EIP)
            import unicorn as _u

            kind = {16: "FETCH", 17: "READ", 18: "WRITE", 19: "RW"}.get(access, str(access))
            print("!! invalid %s at %#x size=%d value=%#x eip=%#x" % (kind, address, size, value, eip))
            return False

        uc.hook_add(UC_HOOK_MEM_INVALID, on_invalid)

    # ---------- 符号 ----------
    def _load_encode_addrs(self):
        sys.path.insert(0, __file__.rsplit("\\", 1)[0])
        from pdb_syms import Pdb, iter_syms, S_PUB32

        pdb = Pdb(DLL_PATH.replace(".dll", ".pdb"))
        addrs = {}
        for rectype, off, seg, ln, name in iter_syms(pdb.streams[pdb.dbi_header()["symrec"]]):
            if rectype != S_PUB32:
                continue
            n = name.split("@")[0]
            if n.startswith("?encode_func_"):
                try:
                    idx = int(n[len("?encode_func_") :])
                except ValueError:
                    continue
                # seg 偏移 → rva
                rva = self._seg2rva(seg, off)
                if rva is not None:
                    addrs[idx] = IMAGE_BASE + rva
        return addrs

    def _seg2rva(self, seg, off):
        if 1 <= seg <= len(self.secs):
            name, sva, vsz, raw, rsz = self.secs[seg - 1]
            return sva - IMAGE_BASE + off
        return None

    # ---------- 执行原语 ----------
    def _run(self, eip, ecx=None, args=(), max_instr=2_000_000):
        uc = self.uc
        esp = STACK_TOP - 0x2000
        uc.reg_write(UC_X86_REG_ESP, esp)
        uc.reg_write(UC_X86_REG_EBP, STACK_TOP - 0x100)
        if ecx is not None:
            uc.reg_write(UC_X86_REG_ECX, ecx)
        # cdecl 栈布局（低→高）：返回地址 → arg1 → arg2…
        for a in reversed(args):
            esp -= 4
            uc.mem_write(esp, struct.pack("<I", a))
        esp -= 4
        uc.mem_write(esp, struct.pack("<I", MAGIC_RET))
        uc.reg_write(UC_X86_REG_ESP, esp)

        state = {"done": False}

        def on_code(uc_, addr, size, user):
            if addr == MAGIC_RET:
                state["done"] = True
                uc_.emu_stop()

        h = uc.hook_add(UC_HOOK_CODE, on_code, begin=MAGIC_RET, end=MAGIC_RET)
        try:
            uc.emu_start(eip, MAGIC_RET, count=max_instr)
        finally:
            uc.hook_del(h)
        # emu 在 until 地址停（fetch 时停，不走 hook）——以 EIP 判定成败
        return state["done"] or uc.reg_read(UC_X86_REG_EIP) == MAGIC_RET

    # ---------- 公开 API ----------
    def encode(self, idx, data16):
        assert len(data16) == 16
        uc = self.uc
        inbuf = HEAP_ADDR
        outbuf = HEAP_ADDR + 0x100
        uc.mem_write(inbuf, bytes(data16))
        uc.mem_write(outbuf, b"\0" * 32)
        ok = self._run(self.encode_addrs[idx], args=(inbuf, outbuf))
        if not ok:
            raise RuntimeError("encode_func_%d did not return cleanly" % idx)
        return bytes(uc.mem_read(outbuf, 16))

    def _decode(self, rva, ct):
        uc = self.uc
        buf = HEAP_ADDR
        uc.mem_write(buf, bytes(ct) + b"\0" * 16)
        fake_this = HEAP_ADDR + 0x800
        ok = self._run(IMAGE_BASE + rva, ecx=fake_this, args=(buf, len(ct)))
        if not ok:
            raise RuntimeError("decode @%x did not return cleanly" % rva)
        return bytes(uc.mem_read(buf, len(ct)))

    def decode_account(self, ct):
        return self._decode(RVA["DecodeAccount"], ct)

    def decode_password(self, ct):
        return self._decode(RVA["DecodePassword"], ct)

    def encode_dispatch(self, idx, data16):
        uc = self.uc
        inbuf = HEAP_ADDR
        outbuf = HEAP_ADDR + 0x100
        uc.mem_write(inbuf, bytes(data16))
        uc.mem_write(outbuf, b"\0" * 32)
        ok = self._run(IMAGE_BASE + RVA["encode_dispatcher"], args=(idx, inbuf, outbuf))
        if not ok:
            raise RuntimeError("encode dispatcher did not return cleanly")
        return bytes(uc.mem_read(outbuf, 16))

    def read_rva(self, rva, n):
        return bytes(self.uc.mem_read(IMAGE_BASE + rva, n))


def selftest():
    o = Oracle()
    print("encode funcs loaded:", len(o.encode_addrs))
    inp = bytes(range(16))
    out = o.encode(0, inp)
    print("encode_func_0(range16) =", out.hex())
    out2 = o.encode_dispatch(0, inp)
    print("dispatcher(0)          =", out2.hex())
    assert out == out2, "dispatcher mismatch"
    # 已知样本（反汇编手工推导的 2044 号）
    o4 = o.encode(2044, inp)
    print("encode_func_2044(range16) =", o4.hex())
    # DecodeAccount：官方 golden 账号槽
    import hashlib

    key1 = bytes.fromhex("66f928e8a1009b7629a01f8def458919")
    # 用 golden 密文验证需读取仓库文件，此处用自造 32B
    ct = bytes(range(32))
    print("decode_account(range32) =", o.decode_account(ct).hex())
    print("decode_password(range32) =", o.decode_password(ct).hex())
    print("OK")


if __name__ == "__main__":
    selftest()
