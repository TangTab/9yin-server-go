#!/usr/bin/env python3
"""最小诊断：OpenProcess/VirtualQueryEx/RPM 三步走通否。"""
import ctypes
import struct

k32 = ctypes.WinDLL('kernel32', use_last_error=True)
k32.OpenProcess.restype = ctypes.c_void_p
k32.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_int, ctypes.c_ulong]
k32.VirtualQueryEx.restype = ctypes.c_size_t
k32.VirtualQueryEx.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_size_t]
k32.ReadProcessMemory.restype = ctypes.c_int
k32.ReadProcessMemory.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_size_t, ctypes.POINTER(ctypes.c_size_t)]


class MBI(ctypes.Structure):
    _fields_ = [('BaseAddress', ctypes.c_ulonglong), ('AllocationBase', ctypes.c_ulonglong),
                ('AllocationProtect', ctypes.c_ulong), ('PartitionId', ctypes.c_ushort),
                ('RegionSize', ctypes.c_ulonglong), ('State', ctypes.c_ulong),
                ('Protect', ctypes.c_ulong), ('Type', ctypes.c_ulong)]


pid = 300232
for acc in (0x1F0FFF, 0x410, 0x10):
    h = k32.OpenProcess(acc, False, pid)
    print(f'OpenProcess({acc:#x}) -> {h}, err={ctypes.get_last_error()}')
    if h:
        break
if not h:
    raise SystemExit('no handle')
mbi = MBI()
n = k32.VirtualQueryEx(h, 0, ctypes.byref(mbi), ctypes.sizeof(mbi))
print('VQE ret', n, 'first region', hex(mbi.BaseAddress), hex(mbi.RegionSize), 'state', hex(mbi.State), 'prot', hex(mbi.Protect))
cnt = 0
addr = 0
total = 0
while addr < 0x7FFFFFFFFFFF:
    if k32.VirtualQueryEx(h, ctypes.c_void_p(addr), ctypes.byref(mbi), ctypes.sizeof(mbi)) == 0:
        break
    if mbi.State == 0x1000 and (mbi.Protect & 0xFF) in (2, 4, 0x20, 0x40):
        cnt += 1
        total += mbi.RegionSize
    addr = mbi.BaseAddress + mbi.RegionSize
print('readable regions(no size cap):', cnt, 'total MB:', total // 1048576)
# 读 fxnet2 头
buf = ctypes.create_string_buffer(16)
got = ctypes.c_size_t()
ok = k32.ReadProcessMemory(h, ctypes.c_void_p(0x10600000), buf, 16, ctypes.byref(got))
print('RPM fxnet2 base:', ok, buf.raw[:8].hex())
