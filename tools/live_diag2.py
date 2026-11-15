import ctypes, ctypes.wintypes as wt
k32 = ctypes.WinDLL('kernel32', use_last_error=True)
adv = ctypes.WinDLL('advapi32', use_last_error=True)
k32.OpenProcess.restype = ctypes.c_void_p
k32.VirtualQueryEx.restype = ctypes.c_size_t
k32.ReadProcessMemory.restype = ctypes.c_int
class LUID(ctypes.Structure):
    _fields_=[('Lo',wt.DWORD),('Hi',wt.LONG)]
class TP(ctypes.Structure):
    _fields_=[('PrivilegeCount',wt.DWORD),('Luid',LUID),('Attributes',wt.DWORD)]
hTok = wt.HANDLE()
adv.OpenProcessToken(k32.GetCurrentProcess(), 0x20, ctypes.byref(hTok))
luid = LUID()
adv.LookupPrivilegeValueW(None, 'SeDebugPrivilege', ctypes.byref(luid))
tp = TP(1, luid, 2)
print('adjust:', adv.AdjustTokenPrivileges(hTok, False, ctypes.byref(tp), 0, None, None), 'err', ctypes.get_last_error())
h = k32.OpenProcess(0x1F0FFF, False, 300232)
print('handle', h)
class MBI(ctypes.Structure):
    _fields_=[('B',ctypes.c_ulonglong),('A',ctypes.c_ulonglong),('AP',ctypes.c_ulong),('P',ctypes.c_ushort),('S',ctypes.c_ulonglong),('St',ctypes.c_ulong),('Pr',ctypes.c_ulong),('T',ctypes.c_ulong)]
m = MBI()
n = k32.VirtualQueryEx(h, ctypes.c_void_p(0x10600000), ctypes.byref(m), ctypes.sizeof(m))
print('VQE', n, hex(m.B) if n else '')
buf = ctypes.create_string_buffer(8); got = ctypes.c_size_t()
ok = k32.ReadProcessMemory(h, ctypes.c_void_p(0x10600000), buf, 8, ctypes.byref(got))
print('RPM', ok, buf.raw.hex())
