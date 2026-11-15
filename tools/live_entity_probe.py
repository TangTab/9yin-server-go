#!/usr/bin/env python3
"""活体读 fxgame.exe 客户端内存：定位玩家 fxnet2 实体并读 AllDest 存储槽。

实体判据（dump 反汇编定版）：
  [+0x58] = "obj-own" 名字串指针（%u-%u 格式）
  [+0x70] = 名字串 131 进制哈希 u32
  [+0x98..0xb4] = AllDest 8 floats（x,y,z,slot3,speed,orient,5.5,9.8）
  [+0xb8] = intent u32
用法: python live_entity_probe.py <pid> <obj> <own> [--watch]
"""
import ctypes
import ctypes.wintypes as wt
import struct
import sys
import time

k32 = ctypes.windll.kernel32


class MEMORY_BASIC_INFORMATION(ctypes.Structure):
    _fields_ = [('BaseAddress', ctypes.c_ulonglong), ('AllocationBase', ctypes.c_ulonglong),
                ('AllocationProtect', wt.DWORD), ('RegionSize', ctypes.c_ulonglong),
                ('State', wt.DWORD), ('Protect', wt.DWORD), ('Type', wt.DWORD)]


def open_proc(pid):
    PROCESS_VM_READ = 0x10
    PROCESS_QUERY_INFORMATION = 0x400
    h = k32.OpenProcess(PROCESS_VM_READ | PROCESS_QUERY_INFORMATION, False, pid)
    if not h:
        raise OSError('OpenProcess failed', ctypes.get_last_error())
    return h


def rpm(h, addr, n):
    buf = ctypes.create_string_buffer(n)
    got = ctypes.c_size_t()
    if not k32.ReadProcessMemory(h, ctypes.c_void_p(addr), buf, n, ctypes.byref(got)):
        return None
    return buf.raw[:got.value]


def regions(h):
    addr = 0
    mbi = MEMORY_BASIC_INFORMATION()
    MEM_COMMIT = 0x1000
    READABLE = {0x02, 0x04, 0x20, 0x40}  # R, RW, RX, RWX
    while addr < 0x7FFFFFFFFFFF:
        if k32.VirtualQueryEx(h, ctypes.c_void_p(addr), ctypes.byref(mbi), ctypes.sizeof(mbi)) == 0:
            break
        if mbi.State == MEM_COMMIT and (mbi.Protect & 0xFF) in READABLE and mbi.Protect != 0x01:
            yield mbi.BaseAddress, mbi.RegionSize
        addr = mbi.BaseAddress + mbi.RegionSize


def fnv131(s: bytes) -> int:
    h = 0
    for c in s:
        h = h * 131 + c
    return h & 0xFFFFFFFF


def find_entities(h, needle: bytes):
    """扫堆找名字串，反推实体并校验 +0x70 哈希。yield (entity_va, name)。"""
    want_hash = fnv131(needle)
    for base, size in regions(h):
        if size > 0x20000000:
            continue
        off = 0
        while off < size:
            chunk = rpm(h, base + off, min(1 << 20, size - off))
            if chunk is None:
                break
            idx = 0
            while True:
                i = chunk.find(needle, idx)
                if i < 0:
                    break
                str_va = base + off + i
                ent_va = str_va - 0x58
                head = rpm(h, ent_va, 0xC0)
                if head and head[:len(needle)] == needle or True:
                    # 校验：实体内 +0x58 处应有指针 == str_va，且 +0x70 == hash
                    if len(head or b'') >= 0xBC:
                        p58 = struct.unpack_from('<Q', head, 0x58)[0]
                        h70 = struct.unpack_from('<I', head, 0x70)[0]
                        if p58 == str_va and h70 == want_hash:
                            yield ent_va, head
                idx = i + 1
            off += len(chunk)
    return


def decode(ent_va, head):
    fl = struct.unpack_from('<8f', head, 0x98)
    intent = struct.unpack_from('<I', head, 0xB8)[0]
    return ent_va, fl, intent


if __name__ == '__main__':
    pid = int(sys.argv[1])
    obj, own = sys.argv[2], sys.argv[3]
    name = f'{obj}-{own}'.encode()
    watch = '--watch' in sys.argv
    h = open_proc(pid)
    while True:
        found = list(find_entities(h, name))
        print(f'[{time.strftime("%H:%M:%S")}] entities named {name!r}: {len(found)}')
        for ent_va, head in found:
            _, fl, intent = decode(ent_va, head)
            print(f'  entity @ {ent_va:#x}: floats={[round(f,3) for f in fl]} intent={intent:#x}')
        if not watch:
            break
        time.sleep(2)
