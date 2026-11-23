"""fxgame.exe 全量内存转储器（等价 procdump -ma，零依赖）。

用法:
  python dump_fxgame.py login          # 等待 fxgame.exe 出现，再等 75 秒(到登录界面)后抓
  python dump_fxgame.py world          # 游戏已在运行，立即抓(进世界后用)
  python dump_fxgame.py world 120      # 等游戏出现后等 120 秒再抓

输出: E:\Workspace\project\game\adata\fxgame_<stage>_<HHMMSS>.dmp
注意: 必须由 fxlaunch.exe 官方链启动游戏（fxupdate 握手），裸启动会被壳自杀。
"""
import ctypes
import ctypes.wintypes as w
import os
import sys
import time
from datetime import datetime

TARGET = "fxgame.exe"
OUT_DIR = r"E:\Workspace\project\game\adata"
WAIT_AFTER_SPAWN = 75  # 进程出现后默认等待秒数(模块链加载+到登录)

PROCESS_ALL_ACCESS = 0x1F0FFF
TH32CS_SNAPPROCESS = 0x2

# MiniDumpWithFullMemory|WithHandleData|WithUnloadedModules|WithFullMemoryInfo|WithThreadInfo|WithTokenInformation
DUMP_FLAGS = 0x00000002 | 0x00000004 | 0x00000020 | 0x00000800 | 0x00001000 | 0x00040000

k32 = ctypes.WinDLL("kernel32", use_last_error=True)


class PROCENTRY32(ctypes.Structure):
    _fields_ = [
        ("dwSize", w.DWORD), ("cntUsage", w.DWORD), ("th32ProcessID", w.DWORD),
        ("th32DefaultHeapID", ctypes.POINTER(ctypes.c_ulong)), ("th32ModuleID", w.DWORD),
        ("cntThreads", w.DWORD), ("th32ParentProcessID", w.DWORD),
        ("pcPriClassBase", ctypes.c_long), ("dwFlags", w.DWORD), ("szExeFile", ctypes.c_char * 260),
    ]


def find_pid(name):
    snap = k32.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
    if snap == w.HANDLE(-1).value or not snap:
        return None
    pe = PROCENTRY32()
    pe.dwSize = ctypes.sizeof(PROCENTRY32)
    pid = None
    if k32.Process32First(snap, ctypes.byref(pe)):
        while True:
            if pe.szExeFile.decode("ascii", "replace").lower() == name:
                # 多实例时取线程数最多的(主进程；壳的看门进程只有 1 线程)
                if pid is None or pe.cntThreads > pid[1]:
                    pid = (pe.th32ProcessID, pe.cntThreads)
            if not k32.Process32Next(snap, ctypes.byref(pe)):
                break
    k32.CloseHandle(snap)
    return pid[0] if pid else None


def main():
    stage = sys.argv[1] if len(sys.argv) > 1 else "login"
    wait = int(sys.argv[2]) if len(sys.argv) > 2 else None

    pid = find_pid(TARGET)
    if pid is None:
        print(f"[.] waiting for {TARGET} ... (launch game via fxlaunch.exe)")
        while pid is None:
            time.sleep(2)
            pid = find_pid(TARGET)
    if wait is None and stage == "login":
        wait = WAIT_AFTER_SPAWN
    if wait:
        print(f"[+] {TARGET} pid={pid}, sleeping {wait}s before dump")
        time.sleep(wait)
        pid = find_pid(TARGET) or pid  # 壳可能换了实例，重找
    if not find_pid(TARGET):
        print("[!] process exited before dump")
        sys.exit(1)

    h = k32.OpenProcess(PROCESS_ALL_ACCESS, False, pid)
    if not h:
        # 降级: 只读权限也够 full-memory dump
        h = k32.OpenProcess(0x1F0FFF ^ 0xF0000, False, pid)
    if not h:
        print(f"[!] OpenProcess({pid}) failed: {ctypes.get_last_error()}")
        print("    请用管理员权限运行本脚本(右键->以管理员身份运行)")
        sys.exit(1)

    os.makedirs(OUT_DIR, exist_ok=True)
    out = os.path.join(OUT_DIR, f"fxgame_{stage}_{datetime.now():%H%M%S}.dmp")
    fh = k32.CreateFileW(out, 0x40000000, 0, None, 2, 0x80, None)  # GENERIC_WRITE, CREATE_ALWAYS
    if fh == w.HANDLE(-1).value or not fh:
        print(f"[!] CreateFile({out}) failed: {ctypes.get_last_error()}")
        sys.exit(1)

    print(f"[.] dumping pid={pid} -> {out} (full memory, may take a minute)")
    dbghelp = ctypes.WinDLL("dbghelp.dll")
    ok = dbghelp.MiniDumpWriteDump(
        w.HANDLE(h), w.DWORD(pid), w.HANDLE(fh), w.DWORD(DUMP_FLAGS),
        None, None, None,
    )
    k32.CloseHandle(fh)
    k32.CloseHandle(h)
    if not ok:
        print(f"[!] MiniDumpWriteDump failed: {ctypes.get_last_error()}")
        sys.exit(1)
    size_mb = os.path.getsize(out) / 1024 / 1024
    print(f"[OK] {out} ({size_mb:.0f} MiB)")


if __name__ == "__main__":
    main()
