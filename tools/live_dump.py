import ctypes
from ctypes import wintypes as wt
k32 = ctypes.WinDLL('kernel32', use_last_error=True)
adv = ctypes.WinDLL('advapi32', use_last_error=True)
dbg = ctypes.WinDLL('DbgHelp.dll', use_last_error=True)
k32.OpenProcess.restype = wt.HANDLE
k32.OpenProcess.argtypes = [wt.DWORD, wt.BOOL, wt.DWORD]
adv.OpenProcessToken.argtypes = [wt.HANDLE, wt.DWORD, ctypes.POINTER(wt.HANDLE)]
adv.LookupPrivilegeValueW.argtypes = [wt.LPCWSTR, wt.LPCWSTR, ctypes.c_void_p]
adv.AdjustTokenPrivileges.argtypes = [wt.HANDLE, wt.BOOL, ctypes.c_void_p, wt.DWORD, ctypes.c_void_p, ctypes.c_void_p]
class LUID(ctypes.Structure):
    _fields_=[('Lo',wt.DWORD),('Hi',ctypes.c_long)]
class TOKEN_PRIVS(ctypes.Structure):
    _fields_=[('Count',wt.DWORD),('Luid',LUID),('Attr',wt.DWORD)]
hTok = wt.HANDLE()
ok = adv.OpenProcessToken(k32.GetCurrentProcess(), 0x20, ctypes.byref(hTok))
luid = LUID()
adv.LookupPrivilegeValueW(None, 'SeDebugPrivilege', ctypes.byref(luid))
tp = TOKEN_PRIVS(1, luid, 2)
r = adv.AdjustTokenPrivileges(hTok, False, ctypes.byref(tp), 0, None, None)
print('adjust', r, 'err', ctypes.get_last_error())
h = k32.OpenProcess(0x1F0FFF, False, 300232)
print('handle', h, 'err', ctypes.get_last_error())
dbg.MiniDumpWriteDump.argtypes = [wt.HANDLE, wt.DWORD, wt.HANDLE, wt.DWORD, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p]
dbg.MiniDumpWriteDump.restype = wt.BOOL
out = r'G:\pojie\ai\tmp_carve\fxgame_live_rush.DMP'
f = k32.CreateFileW(out, 0x40000000, 0, None, 2, 0x80, None)
ok = dbg.MiniDumpWriteDump(h, 300232, f, 2, None, None, None)
print('dump', ok, 'err', ctypes.get_last_error())
k32.CloseHandle(f)
