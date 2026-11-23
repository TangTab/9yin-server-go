/* fxgame.exe dumper v2: rights probing + MiniDumpWriteDump + manual fallback.
 *
 * usage: dump_fxgame.exe [stage] [wait_secs]     (run as ADMINISTRATOR)
 *   rights probe: opens the target once per access right, prints granted/denied
 *   if full minidump fails -> manual Memory64 minidump via ReadProcessMemory
 */
#include <windows.h>
#include <tlhelp32.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

#define TARGET "fxgame.exe"
#define OUT_DIR "E:\\Workspace\\project\\game\\adata"
#define MIN_THREADS 8

static DWORD find_pid(const char *name, DWORD *threads_out)
{
    HANDLE snap = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if (snap == INVALID_HANDLE_VALUE) return 0;
    PROCESSENTRY32 pe = { .dwSize = sizeof(pe) };
    DWORD best = 0, best_threads = 0;
    if (Process32First(snap, &pe)) {
        do {
            if (_stricmp(pe.szExeFile, name) == 0 && pe.cntThreads > best_threads) {
                best = pe.th32ProcessID; best_threads = pe.cntThreads;
            }
        } while (Process32Next(snap, &pe));
    }
    CloseHandle(snap);
    if (best_threads < MIN_THREADS) return 0;
    if (threads_out) *threads_out = best_threads;
    return best;
}

static int enable_se_debug(void)
{
    HANDLE tok;
    if (!OpenProcessToken(GetCurrentProcess(),
                          TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &tok)) return 0;
    LUID luid;
    if (!LookupPrivilegeValueA(NULL, "SeDebugPrivilege", &luid)) { CloseHandle(tok); return 0; }
    TOKEN_PRIVILEGES tp = { .PrivilegeCount = 1,
                            .Privileges[0].Luid = luid,
                            .Privileges[0].Attributes = SE_PRIVILEGE_ENABLED };
    BOOL ok = AdjustTokenPrivileges(tok, FALSE, &tp, sizeof(tp), NULL, NULL);
    DWORD err = GetLastError();
    CloseHandle(tok);
    return (ok && err != ERROR_NOT_ALL_ASSIGNED);
}

static void probe_rights(DWORD pid)
{
    struct { DWORD right; const char *name; } t[] = {
        { PROCESS_VM_READ,           "VM_READ" },
        { PROCESS_VM_OPERATION,      "VM_OPERATION" },
        { PROCESS_VM_WRITE,          "VM_WRITE" },
        { PROCESS_QUERY_INFORMATION, "QUERY_INFO" },
        { PROCESS_DUP_HANDLE,        "DUP_HANDLE" },
        { PROCESS_CREATE_THREAD,     "CREATE_THREAD" },
        { PROCESS_ALL_ACCESS,        "ALL_ACCESS" },
    };
    printf("[.] rights probe:\n");
    for (int i = 0; i < 7; i++) {
        SetLastError(0);
        HANDLE h = OpenProcess(t[i].right, FALSE, pid);
        DWORD err = GetLastError();
        printf("    %-14s %s (err=%lu)\n", t[i].name, h ? "GRANTED" : "DENIED ", err);
        if (h) CloseHandle(h);
    }
}

/* manual Memory64 minidump: MDMP header + Memory64ListStream */
#pragma pack(push,1)
typedef struct { DWORD sig; DWORD version; DWORD nstreams; DWORD dirrva;
                 DWORD checksum; DWORD timestamp; DWORD __pad; unsigned long long flags; } HDR;
typedef struct { DWORD type; DWORD size; DWORD rva; } DIRENT;
#pragma pack(pop)

static int manual_dump(HANDLE h, DWORD pid, const char *out)
{
    /* 1. enumerate committed regions */
    MEMORY_BASIC_INFORMATION mbi;
    UINT64 addr = 0;
    DWORD nregions = 0;
    UINT64 total = 0;
    while (VirtualQueryEx(h, (LPCVOID)addr, &mbi, sizeof(mbi)) == sizeof(mbi)) {
        UINT64 next = (UINT64)mbi.BaseAddress + mbi.RegionSize;
        if (next <= addr) break;
        if (mbi.State == MEM_COMMIT &&
            (mbi.Protect & (PAGE_NOACCESS | PAGE_GUARD)) == 0 &&
            mbi.RegionSize < 0x10000000ULL) {
            nregions++; total += mbi.RegionSize;
        }
        addr = next;
        if (addr >= 0x7FFFFFFFFFFFULL) break;
    }
    printf("[.] manual dump: %lu regions, %.0f MiB committed\n",
           (unsigned long)nregions, total / 1048576.0);
    if (!nregions) return 0;

    FILE *f = fopen(out, "wb");
    if (!f) { printf("[!] fopen(%s) err=%lu\n", out, GetLastError()); return 0; }

    DWORD dir_off = sizeof(HDR);                       /* 1 stream */
    UINT64 list_off = dir_off + sizeof(DIRENT);
    UINT64 desc_off = list_off + 16;                   /* nranges + baserva */
    UINT64 data_off = desc_off + (UINT64)nregions * 16;

    HDR hdr = { 0x504D444D, 0xA793, 1, dir_off, 0, (DWORD)time(NULL), 0, 0x0002 };
    fwrite(&hdr, sizeof(hdr), 1, f);
    DIRENT de = { 9 /*Memory64List*/, (DWORD)(16 + (UINT64)nregions * 16), (DWORD)list_off };
    fwrite(&de, sizeof(de), 1, f);
    unsigned long long nr = nregions, base = data_off;
    fwrite(&nr, 8, 1, f); fwrite(&base, 8, 1, f);

    /* write descriptors while streaming data */
    UINT64 addr2 = 0; DWORD n = 0; UINT64 written_bytes = 0;
    unsigned char *buf = malloc(1 << 20);
    while (VirtualQueryEx(h, (LPCVOID)addr2, &mbi, sizeof(mbi)) == sizeof(mbi)) {
        UINT64 next = (UINT64)mbi.BaseAddress + mbi.RegionSize;
        if (next <= addr2) break;
        if (mbi.State == MEM_COMMIT &&
            (mbi.Protect & (PAGE_NOACCESS | PAGE_GUARD)) == 0 &&
            mbi.RegionSize < 0x10000000ULL) {
            unsigned long long start = (UINT64)mbi.BaseAddress;
            unsigned long long size = mbi.RegionSize;
            fwrite(&start, 8, 1, f); fwrite(&size, 8, 1, f);
            n++;
        }
        addr2 = next;
        if (addr2 >= 0x7FFFFFFFFFFFULL) break;
    }
    /* now data */
    addr2 = 0; n = 0;
    while (VirtualQueryEx(h, (LPCVOID)addr2, &mbi, sizeof(mbi)) == sizeof(mbi)) {
        UINT64 next = (UINT64)mbi.BaseAddress + mbi.RegionSize;
        if (next <= addr2) break;
        if (mbi.State == MEM_COMMIT &&
            (mbi.Protect & (PAGE_NOACCESS | PAGE_GUARD)) == 0 &&
            mbi.RegionSize < 0x10000000ULL) {
            UINT64 got = 0;
            while (got < mbi.RegionSize) {
                SIZE_T want = (SIZE_T)((mbi.RegionSize - got) > (1 << 20) ? (1 << 20) : (mbi.RegionSize - got));
                SIZE_T rd = 0;
                if (!ReadProcessMemory(h, (LPCVOID)((UINT64)mbi.BaseAddress + got), buf, want, &rd)) {
                    /* partial: zero-fill rest of region */
                    memset(buf, 0, want);
                    rd = want;
                }
                fwrite(buf, 1, rd, f);
                written_bytes += rd;
                got += rd;
            }
            n++;
            if (n % 50 == 0) printf("    ... %lu regions, %.0f MiB\n", (unsigned long)n, written_bytes / 1048576.0);
        }
        addr2 = next;
        if (addr2 >= 0x7FFFFFFFFFFFULL) break;
    }
    free(buf);
    fclose(f);
    printf("[OK] manual dump %s (%.0f MiB, %lu regions)\n",
           out, written_bytes / 1048576.0, (unsigned long)n);
    return 1;
}

int main(int argc, char **argv)
{
    const char *stage = argc > 1 ? argv[1] : "login";
    int wait = argc > 2 ? atoi(argv[2]) : (strcmp(stage, "login") == 0 ? 75 : 0);

    if (enable_se_debug()) printf("[+] SeDebugPrivilege enabled\n");
    else printf("[!] SeDebugPrivilege NOT granted - run as administrator\n");

    printf("[.] looking for %s (%d+ threads)...\n", TARGET, MIN_THREADS);
    DWORD threads = 0;
    DWORD pid = find_pid(TARGET, &threads);
    while (!pid) {
        printf("[.] waiting (launch game via fxlaunch.exe)...\n");
        Sleep(2000);
        pid = find_pid(TARGET, &threads);
    }
    printf("[+] pid=%lu threads=%lu\n", (unsigned long)pid, (unsigned long)threads);
    if (wait > 0) {
        Sleep(wait * 1000);
        pid = find_pid(TARGET, &threads);
        if (!pid) { printf("[!] process exited\n"); return 1; }
    }

    probe_rights(pid);

    HANDLE h = OpenProcess(PROCESS_ALL_ACCESS, FALSE, pid);
    if (!h) h = OpenProcess(PROCESS_QUERY_INFORMATION | PROCESS_VM_READ | PROCESS_DUP_HANDLE, FALSE, pid);
    if (!h) { printf("[!] OpenProcess failed\n"); return 1; }

    /* read sanity test */
    unsigned char peek[16]; SIZE_T rd = 0;
    if (ReadProcessMemory(h, (LPCVOID)0x140000000, peek, 16, &rd) && rd == 16 && peek[0] == 'M' && peek[1] == 'Z')
        printf("[+] ReadProcessMemory OK (MZ at 0x140000000)\n");
    else
        printf("[!] ReadProcessMemory failed/short (err=%lu rd=%lu)\n", GetLastError(), (unsigned long)rd);

    SYSTEMTIME st; GetLocalTime(&st);
    char out[MAX_PATH];
    _snprintf(out, MAX_PATH, "%s\\fxgame_%s_%02d%02d%02d.dmp",
              OUT_DIR, stage, st.wHour, st.wMinute, st.wSecond);

    /* try system minidump first */
    HMODULE dbgh = LoadLibraryA("dbghelp.dll");
    BOOL (WINAPI *mdwd)(HANDLE, DWORD, HANDLE, DWORD, PVOID, PVOID, PVOID) =
        (void *)GetProcAddress(dbgh, "MiniDumpWriteDump");
    HANDLE fh = CreateFileA(out, GENERIC_WRITE, 0, NULL, CREATE_ALWAYS, 0, NULL);
    if (fh != INVALID_HANDLE_VALUE) {
        printf("[.] MiniDumpWriteDump ...\n");
        if (mdwd && mdwd(h, pid, fh, 0x41822, NULL, NULL, NULL)) {
            LARGE_INTEGER sz = {0}; GetFileSizeEx(fh, &sz);
            printf("[OK] %s (%.0f MiB)\n", out, sz.QuadPart / 1048576.0);
            CloseHandle(fh); return 0;
        }
        printf("[!] MiniDumpWriteDump failed err=%lu -> manual fallback\n", GetLastError());
        CloseHandle(fh);
    }
    DeleteFileA(out);
    _snprintf(out, MAX_PATH, "%s\\fxgame_%s_%02d%02d%02d_manual.dmp",
              OUT_DIR, stage, st.wHour, st.wMinute, st.wSecond);
    return manual_dump(h, pid, out) ? 0 : 1;
}
