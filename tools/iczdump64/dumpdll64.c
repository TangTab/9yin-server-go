/* IczDump64 v4 - pure in-process dumper (Iczelion's IczDump philosophy).
 *
 * The game is launched NORMALLY via fxlaunch (all packer gates pass
 * legitimately, nothing is patched - the stub checksums its code and
 * kills the process on any modification, see v3 experiments).
 * This DLL is attached afterwards and only OBSERVES + dumps:
 *
 *   Ctrl+Alt+D : MiniDumpWriteDump(SELF, full memory)
 *   Ctrl+Alt+T : log module list + all thread RIPs
 *   log        : E:\Workspace\project\game\adata\iczdump64.log
 *
 * build: x86_64-w64-mingw32-gcc -shared -O2 -o IczDump64.dll dumpdll64.c
 */
#include <windows.h>
#include <tlhelp32.h>
#include <stdio.h>
#include <string.h>

#define LOG_PATH "E:\\Workspace\\project\\game\\adata\\iczdump64.log"
#define OUT_DIR  "E:\\Workspace\\project\\game\\adata"
#define DUMP_FLAGS (0x02 | 0x04 | 0x20 | 0x800 | 0x1000 | 0x40000)

static FILE *g_log;

static void logf_(const char *fmt, ...)
{
    if (!g_log) return;
    SYSTEMTIME st; GetLocalTime(&st);
    fprintf(g_log, "[%02d:%02d:%02d.%03d] ",
            st.wHour, st.wMinute, st.wSecond, st.wMilliseconds);
    va_list ap; va_start(ap, fmt);
    vfprintf(g_log, fmt, ap);
    va_end(ap);
    fflush(g_log);
}

static void log_threads(void)
{
    DWORD pid = GetCurrentProcessId();
    HANDLE snap = CreateToolhelp32Snapshot(TH32CS_SNAPTHREAD, 0);
    if (snap == INVALID_HANDLE_VALUE) return;
    THREADENTRY32 te = { .dwSize = sizeof(te) };
    int n = 0;
    if (Thread32First(snap, &te)) {
        do {
            if (te.th32OwnerProcessID != pid) continue;
            HANDLE ht = OpenThread(THREAD_GET_CONTEXT | THREAD_QUERY_INFORMATION,
                                   FALSE, te.th32ThreadID);
            if (!ht) continue;
            CONTEXT ctx = { .ContextFlags = CONTEXT_CONTROL };
            if (GetThreadContext(ht, &ctx)) {
                HMODULE m = NULL;
                char modname[MAX_PATH] = "?";
                if (GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS |
                                       GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
                                       (LPCSTR)ctx.Rip, &m))
                    GetModuleFileNameA(m, modname, MAX_PATH);
                const char *base = strrchr(modname, '\\');
                logf_("tid=%-6lu rip=%#llx %s+%#llx\n",
                      (unsigned long)te.th32ThreadID,
                      (unsigned long long)ctx.Rip,
                      base ? base + 1 : modname,
                      m ? (unsigned long long)(ctx.Rip - (UINT64)m) : 0ull);
            }
            CloseHandle(ht);
            if (++n > 60) break;
        } while (Thread32Next(snap, &te));
    }
    CloseHandle(snap);
    logf_("(%d threads listed)\n", n);
}

static void do_self_dump(void)
{
    SYSTEMTIME st; GetLocalTime(&st);
    char out[MAX_PATH];
    _snprintf(out, MAX_PATH, "%s\\fxgame_self_%02d%02d%02d.dmp",
              OUT_DIR, st.wHour, st.wMinute, st.wSecond);
    HANDLE fh = CreateFileA(out, GENERIC_WRITE, 0, NULL,
                            CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (fh == INVALID_HANDLE_VALUE) {
        logf_("dump: CreateFile(%s) err=%lu\n", out, GetLastError());
        return;
    }
    HMODULE dbgh = GetModuleHandleA("dbghelp.dll");
    if (!dbgh) dbgh = LoadLibraryA("dbghelp.dll");
    BOOL (WINAPI *mdwd)(HANDLE, DWORD, HANDLE, DWORD, PVOID, PVOID, PVOID) =
        (dbgh ? (void *)GetProcAddress(dbgh, "MiniDumpWriteDump") : NULL);
    logf_("dump: writing %s ...\n", out);
    if (!mdwd || !mdwd(GetCurrentProcess(), GetCurrentProcessId(),
                       fh, DUMP_FLAGS, NULL, NULL, NULL)) {
        logf_("dump: MiniDumpWriteDump failed err=%lu\n", GetLastError());
    } else {
        LARGE_INTEGER sz = {0};
        GetFileSizeEx(fh, &sz);
        logf_("dump: OK %s (%.0f MiB)\n", out, sz.QuadPart / 1048576.0);
    }
    CloseHandle(fh);
}


static int am_i_fxgame(void)
{
    wchar_t exe[MAX_PATH];
    GetModuleFileNameW(NULL, exe, MAX_PATH);
    CharLowerW(exe);
    return (wcsstr(exe, L"fxgame.exe") != NULL);
}

/* file-trigger loop: any process holding this DLL sees the trigger file,
   but ONLY fxgame.exe instances dump (and consume the trigger). */
static void trigger_loop(void)
{
    const char *path = OUT_DIR "\\iczdump.trigger";
    for (;;) {
        if (GetFileAttributesA(path) != INVALID_FILE_ATTRIBUTES) {
            if (am_i_fxgame()) {
                logf_("trigger seen -> self dump\n");
                do_self_dump();
                DeleteFileA(path);
            } else {
                logf_("trigger seen but I am not fxgame - ignored\n");
                Sleep(3000);   /* let the game instance consume it first */
            }
        }
        Sleep(500);
    }
}

static DWORD WINAPI worker(LPVOID unused)
{
    (void)unused;
    g_log = fopen(LOG_PATH, "a");
    logf_("=== IczDump64 v4 (attach mode) pid=%lu ===\n", GetCurrentProcessId());
    logf_("modules: kernel32=%#llx ntdll=%#llx exe=%#llx\n",
          (unsigned long long)(UINT_PTR)GetModuleHandleA("kernel32.dll"),
          (unsigned long long)(UINT_PTR)GetModuleHandleA("ntdll.dll"),
          (unsigned long long)(UINT_PTR)GetModuleHandleW(NULL));

    if (!RegisterHotKey(NULL, 1, MOD_CONTROL | MOD_ALT, 'D'))
        logf_("RegisterHotKey(D) err=%lu\n", GetLastError());
    if (!RegisterHotKey(NULL, 2, MOD_CONTROL | MOD_ALT, 'T'))
        logf_("RegisterHotKey(T) err=%lu\n", GetLastError());
    logf_("hotkeys ready: Ctrl+Alt+D=dump Ctrl+Alt+T=threads\n");
    logf_("trigger: create %s\\iczdump.trigger to dump THIS game\n", OUT_DIR);

    CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)trigger_loop, NULL, 0, NULL);

    MSG msg;
    while (GetMessage(&msg, NULL, 0, 0) > 0) {
        if (msg.message == WM_HOTKEY) {
            if (msg.wParam == 1) do_self_dump();
            if (msg.wParam == 2) log_threads();
        }
    }
    return 0;
}

BOOL WINAPI DllMain(HINSTANCE h, DWORD reason, LPVOID r)
{
    (void)r;
    if (reason == DLL_PROCESS_ATTACH) {
        DisableThreadLibraryCalls(h);
        CreateThread(NULL, 0, worker, NULL, 0, NULL);
    }
    return TRUE;
}

/* exported hook proc for SetWindowsHookEx-based injection:
   the OS loads this DLL into every GUI process that pumps messages;
   DllMain starts the worker (hotkeys + self-dump) automatically. */
__declspec(dllexport) LRESULT CALLBACK hook_proc(int code, WPARAM w, LPARAM l)
{
    return CallNextHookEx(NULL, code, w, l);
}
