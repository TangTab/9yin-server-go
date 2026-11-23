/* IczLoader64 - modern x64 rework of Iczelion's IczLoader.
 *
 * Two modes:
 *   IczLoader64.exe                spawn target suspended, inject, resume
 *   IczLoader64.exe -attach        inject into the RUNNING fxgame.exe
 *                                  (launched via fxlaunch - all gates pass
 *                                   legitimately, we patch nothing)
 *
 * build: x86_64-w64-mingw32-gcc -O2 -o IczLoader64.exe loader64.c
 */
#include <windows.h>
#include <tlhelp32.h>
#include <stdio.h>
#include <string.h>

#define DEFAULT_TARGET "E:\\Game\\9yin\\9yin\\bin64\\fxgame.exe"
#define MIN_THREADS 8
#define LOG_PATH "E:\\Workspace\\project\\game\\adata\\iczdump64.log"

static void logline(const char *fmt, ...)
{
    FILE *f = fopen(LOG_PATH, "a");
    if (!f) return;
    SYSTEMTIME st; GetLocalTime(&st);
    fprintf(f, "[%02d:%02d:%02d] [loader] ", st.wHour, st.wMinute, st.wSecond);
    va_list ap; va_start(ap, fmt);
    vfprintf(f, fmt, ap);
    va_end(ap);
    fprintf(f, "\n");
    fclose(f);
}

static void pause_(void)
{
    printf("\npress ENTER to close ...\n");
    getchar();
}

/* SeDebugPrivilege bypasses the restrictive DACL fxprotect.dll installs
   on the game process object. Requires admin elevation. */
static int enable_se_debug(void)
{
    HANDLE tok;
    if (!OpenProcessToken(GetCurrentProcess(),
                          TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &tok))
        return 0;
    LUID luid;
    if (!LookupPrivilegeValueA(NULL, "SeDebugPrivilege", &luid)) {
        CloseHandle(tok); return 0;
    }
    TOKEN_PRIVILEGES tp = { .PrivilegeCount = 1,
                            .Privileges[0].Luid = luid,
                            .Privileges[0].Attributes = SE_PRIVILEGE_ENABLED };
    BOOL ok = AdjustTokenPrivileges(tok, FALSE, &tp, sizeof(tp), NULL, NULL);
    DWORD err = GetLastError();
    CloseHandle(tok);
    return (ok && err != ERROR_NOT_ALL_ASSIGNED);
}

static DWORD find_game_pid(void)
{
    HANDLE snap = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if (snap == INVALID_HANDLE_VALUE) return 0;
    PROCESSENTRY32 pe = { .dwSize = sizeof(pe) };
    DWORD best = 0, best_threads = 0;
    if (Process32First(snap, &pe)) {
        do {
            if (_stricmp(pe.szExeFile, "fxgame.exe") == 0 &&
                pe.cntThreads > best_threads) {
                best = pe.th32ProcessID;
                best_threads = pe.cntThreads;
            }
        } while (Process32Next(snap, &pe));
    }
    CloseHandle(snap);
    return (best_threads >= MIN_THREADS) ? best : 0;
}

static int inject(HANDLE hProc, const char *dllpath)
{
    SIZE_T need = strlen(dllpath) + 1;
    LPVOID remote = VirtualAllocEx(hProc, NULL, need,
                                   MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (!remote) {
        printf("[!] VirtualAllocEx err=%lu\n", GetLastError());
        return 1;
    }
    SIZE_T written = 0;
    if (!WriteProcessMemory(hProc, remote, dllpath, need, &written)) {
        printf("[!] WriteProcessMemory err=%lu\n", GetLastError());
        return 1;
    }
    HMODULE k32 = GetModuleHandleA("kernel32.dll");
    LPTHREAD_START_ROUTINE loadlib =
        (LPTHREAD_START_ROUTINE)GetProcAddress(k32, "LoadLibraryA");
    HANDLE th = CreateRemoteThread(hProc, NULL, 0, loadlib, remote, 0, NULL);
    if (!th) {
        printf("[!] CreateRemoteThread err=%lu\n", GetLastError());
        return 1;
    }
    WaitForSingleObject(th, 10000);
    DWORD hmod = 0;
    GetExitCodeThread(th, &hmod);
    CloseHandle(th);
    VirtualFreeEx(hProc, remote, 0, MEM_RELEASE);
    if (!hmod) {
        printf("[!] LoadLibrary in target returned NULL\n");
        return 1;
    }
    printf("[+] injected (hmodule=%#lx)\n", (unsigned long)hmod);
    return 0;
}

static int real_main(int argc, char **argv)
{
    char dllpath[MAX_PATH] = {0};
    if (argc > 2) strncpy(dllpath, argv[2], MAX_PATH - 1);
    else {
        GetModuleFileNameA(NULL, dllpath, MAX_PATH);
        char *slash = strrchr(dllpath, '\\');
        if (slash) strcpy(slash + 1, "IczDump64.dll");
    }
    printf("[.] dll : %s\n", dllpath);
    if (GetFileAttributesA(dllpath) == INVALID_FILE_ATTRIBUTES) {
        printf("[!] dll not found\n"); return 1;
    }

    if (enable_se_debug())
        printf("[+] SeDebugPrivilege enabled\n");
    else {
        printf("[!] SeDebugPrivilege NOT granted - run as administrator!\n");
        logline("SeDebugPrivilege not granted (not elevated?)");
    }

    /* ---- attach mode: game already running via official chain ---- */
    if (argc < 2 || strcmp(argv[1], "-attach") == 0) {
        printf("[.] looking for running fxgame.exe (%d+ threads)...\n", MIN_THREADS);
        DWORD pid = find_game_pid();
        if (!pid) {
            printf("[!] game not running - launch it via fxlaunch.lnk first\n");
            return 1;
        }
        HANDLE h = OpenProcess(PROCESS_ALL_ACCESS, FALSE, pid);
        if (!h) {
            printf("[!] OpenProcess(%lu) err=%lu (run as admin)\n",
                   (unsigned long)pid, GetLastError());
            return 1;
        }
        printf("[+] target pid=%lu\n", (unsigned long)pid);
        int rc = inject(h, dllpath);
        CloseHandle(h);
        if (rc == 0)
            printf("[+] hotkeys in game: Ctrl+Alt+D=dump Ctrl+Alt+T=threads\n");
        return rc;
    }

    /* ---- spawn mode: explicit -spawn only (hits the packer gate!) ---- */
    if (argc > 1 && strcmp(argv[1], "-spawn") != 0) {
        printf("[!] unknown arg %s\n", argv[1]);
        return 1;
    }
    const char *target = (argc > 2) ? argv[2] : DEFAULT_TARGET;
    printf("[.] target : %s\n", target);
    if (GetFileAttributesA(target) == INVALID_FILE_ATTRIBUTES) {
        printf("[!] target not found\n"); return 1;
    }
    char dir[MAX_PATH] = {0};
    strncpy(dir, target, MAX_PATH - 1);
    char *slash = strrchr(dir, '\\');
    if (slash) *slash = 0;
    STARTUPINFOA si = { .cb = sizeof(si) };
    PROCESS_INFORMATION pi = {0};
    char cmdline[MAX_PATH + 4];
    _snprintf(cmdline, MAX_PATH + 3, "\"%s\"", target);
    if (!CreateProcessA(target, cmdline, NULL, NULL, FALSE,
                        CREATE_SUSPENDED, NULL, dir, &si, &pi)) {
        printf("[!] CreateProcess err=%lu\n", GetLastError());
        return 1;
    }
    printf("[+] process pid=%lu (suspended)\n", (unsigned long)pi.dwProcessId);
    if (inject(pi.hProcess, dllpath) == 0)
        printf("[+] hotkeys in game: Ctrl+Alt+D=dump Ctrl+Alt+T=threads\n");
    ResumeThread(pi.hThread);
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    return 0;
}

int main(int argc, char **argv)
{
    int rc = real_main(argc, argv);
    pause_();
    return rc;
}
