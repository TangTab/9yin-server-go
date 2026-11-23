/* hooker64 - SetWindowsHookEx injector (no process handle needed).
 *
 * Installs a global WH_GETMESSAGE hook backed by IczDump64.dll.
 * Every GUI process that pumps messages - including fxgame.exe -
 * gets the DLL loaded by the OS itself. Handle-stripping drivers
 * cannot see this (there is no OpenProcess at all).
 *
 * build: x86_64-w64-mingw32-gcc -mwindows -O2 -o hooker64.exe hooker64.c
 * usage: hooker64.exe [seconds-to-stay-hooked, default 60]
 */
#include <windows.h>
#include <stdio.h>
#include <stdlib.h>

#define LOG_PATH "E:\\Workspace\\project\\game\\adata\\iczdump64.log"

int main(int argc, char **argv)
{
    int seconds = argc > 1 ? atoi(argv[1]) : 60;

    char dllpath[MAX_PATH] = {0};
    GetModuleFileNameA(NULL, dllpath, MAX_PATH);
    char *slash = strrchr(dllpath, '\\');
    if (slash) strcpy(slash + 1, "IczDump64b.dll");

    HMODULE dll = LoadLibraryA(dllpath);   /* map into self so hook is valid */
    if (!dll) {
        printf("[!] cannot load %s err=%lu\n", dllpath, GetLastError());
        return 1;
    }
    HOOKPROC proc = (HOOKPROC)GetProcAddress(dll, "hook_proc");
    if (!proc) {
        printf("[!] hook_proc not found\n");
        return 1;
    }

    printf("[.] installing global WH_GETMESSAGE hook ...\n");
    HHOOK hhk = SetWindowsHookExW(WH_GETMESSAGE, proc, dll, 0);
    if (!hhk) {
        printf("[!] SetWindowsHookEx err=%lu\n", GetLastError());
        return 1;
    }
    printf("[+] hook installed - nudging all windows to pump messages\n");
    PostMessageW(HWND_BROADCAST, WM_NULL, 0, 0);
    SendMessageTimeoutW(HWND_BROADCAST, WM_NULL, 0, 0, SMTO_ABORTIFHUNG, 2000, NULL);

    printf("[+] staying hooked %ds (game should pick it up; then Ctrl+Alt+D)\n", seconds);
    printf("[.] (fxgame only loads the hook while pumping messages - give it a few seconds)\n");
    Sleep(seconds * 1000);

    UnhookWindowsHookEx(hhk);
    printf("[.] unhooked\n");
    return 0;
}
