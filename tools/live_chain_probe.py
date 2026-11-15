#!/usr/bin/env python3
"""活体全链探针：玩家实体存储槽 + receiver→consumer→vtable+0x220 链。

每遍以 append+close 落盘（缓冲免疫）。输出 tmp/live_chain_out.txt。
"""
import ctypes
import struct
import sys
import time

k32 = ctypes.WinDLL('kernel32', use_last_error=True)
k32.OpenProcess.restype = ctypes.c_void_p
k32.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_int, ctypes.c_ulong]
k32.VirtualQueryEx.restype = ctypes.c_size_t
k32.VirtualQueryEx.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_size_t]
k32.ReadProcessMemory.restype = ctypes.c_int
k32.ReadProcessMemory.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_size_t, ctypes.POINTER(ctypes.c_size_t)]
OUT = r'G:\pojie\ai\9yin-server-go\tmp\live_chain_out.txt'


class MBI(ctypes.Structure):
    _fields_ = [('BaseAddress', ctypes.c_ulonglong), ('AllocationBase', ctypes.c_ulonglong),
                ('AllocationProtect', ctypes.c_ulong), ('PartitionId', ctypes.c_ushort),
                ('RegionSize', ctypes.c_ulonglong), ('State', ctypes.c_ulong),
                ('Protect', ctypes.c_ulong), ('Type', ctypes.c_ulong)]


def log(msg):
    with open(OUT, 'a', encoding='utf-8') as f:
        f.write(msg + '\n')


def rpm(h, addr, n):
    buf = ctypes.create_string_buffer(n)
    got = ctypes.c_size_t()
    if not k32.ReadProcessMemory(h, ctypes.c_void_p(addr), buf, n, ctypes.byref(got)):
        return None
    return buf.raw[:got.value]


def regions(h):
    addr = 0
    mbi = MBI()
    while addr < 0x7FFFFFFFFFFF:
        if k32.VirtualQueryEx(h, ctypes.c_void_p(addr), ctypes.byref(mbi), ctypes.sizeof(mbi)) == 0:
            break
        if mbi.State == 0x1000 and (mbi.Protect & 0xFF) in (2, 4, 0x20, 0x40) and mbi.RegionSize <= 0x8000000:
            yield mbi.BaseAddress, mbi.RegionSize
        addr = mbi.BaseAddress + mbi.RegionSize


def q(h, va):
    b = rpm(h, va, 8)
    return struct.unpack('<Q', b)[0] if b else None


def u32(h, va):
    b = rpm(h, va, 4)
    return struct.unpack('<I', b)[0] if b else None


def fnv(s: bytes) -> int:
    h = 0
    for c in s:
        h = (h * 131 + c) & 0xFFFFFFFF
    return h


def name_of(h, ent):
    """[ent+0x58]：长名=堆指针，短名=内联字符。"""
    p = q(h, ent + 0x58)
    for tgt in (p, ent + 0x58):
        if not tgt or tgt > 0x7FFFFFFFFFFF or tgt < 0x10000:
            continue
        b = rpm(h, tgt, 24)
        if not b:
            continue
        s = b.split(b'\x00')[0]
        if 3 <= len(s) <= 23 and all(48 <= c <= 57 or c == 45 for c in s):
            return s, tgt
    return None, None


def find_player_entity(h, name: bytes):
    want = fnv(name)
    for base, size in regions(h):
        off = 0
        while off < size:
            chunk = rpm(h, base + off, min(1 << 20, size - off))
            if not chunk:
                break
            idx = 0
            while True:
                i = chunk.find(name, idx)
                if i < 0:
                    break
                str_va = base + off + i
                # 反查实体：扫该页附近谁的 +0x58 == str_va 且 +0x70 == hash —— 直接验证法：
                # 实体在堆任何位置，改为全局找指针 str_va
                idx = i + 1
            off += len(chunk)
    # 简化：全内存找 8 字节指针 → str_va 需要多次……此函数退化为：先收集串址，再全局找指针
    return []


def scan_ptr(h, target):
    """全内存找 8 字节值 == target。"""
    pat = struct.pack('<Q', target)
    hits = []
    for base, size in regions(h):
        off = 0
        while off < size:
            chunk = rpm(h, base + off, min(1 << 22, size - off))
            if not chunk:
                break
            idx = 0
            while True:
                i = chunk.find(pat, idx)
                if i < 0:
                    break
                hits.append(base + off + i)
                idx = i + 1
            off += len(chunk)
    return hits


def main(pid, obj, own):
    name = f'{obj}-{own}'.encode()
    h = k32.OpenProcess(0x1F0FFF, False, pid)
    if not h:
        h = k32.OpenProcess(0x10 | 0x400, False, pid)
    if not h:
        log(f'OpenProcess failed err={ctypes.get_last_error()}')
        return
    log(f'=== probe start pid={pid} name={name!r}')
    want = fnv(name)
    while True:
        t0 = time.time()
        # 1) 找名字串
        str_vas = []
        for base, size in regions(h):
            off = 0
            while off < size and len(str_vas) < 8:
                chunk = rpm(h, base + off, min(1 << 22, size - off))
                if not chunk:
                    break
                idx = 0
                while True:
                    i = chunk.find(name, idx)
                    if i < 0:
                        break
                    str_vas.append(base + off + i)
                    idx = i + 1
                off += len(chunk)
        ents = []
        for sv in str_vas:
            for pva in scan_ptr(h, sv):
                e = pva - 0x58
                if u32(h, e + 0x70) == want:
                    ents.append(e)
        ents = sorted(set(ents))
        lines = [f'[{time.strftime("%H:%M:%S")}] pass {time.time()-t0:.1f}s strs={len(str_vas)} ents={len(ents)}']
        for e in ents[:6]:
            b = rpm(h, e + 0x90, 0x34)
            if b:
                fl = struct.unpack_from('<8f', b, 8)
                intent = struct.unpack_from('<I', b, 0x28)[0]
                lines.append(f'  ent {e:#x} dest=({fl[0]:.2f},{fl[1]:.2f},{fl[2]:.2f}) speed={fl[4]:.2f} orient={fl[5]:.3f} intent={intent:#010x}')
        log('\n'.join(lines))
        time.sleep(3)


if __name__ == '__main__':
    main(int(sys.argv[1]), sys.argv[2], sys.argv[3])
