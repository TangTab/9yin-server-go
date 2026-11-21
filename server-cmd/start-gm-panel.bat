@echo off
setlocal EnableExtensions
title 9yin Go - gm panel

rem ============================================================
rem  启动维护 game_panel.exe（http://127.0.0.1:19064）
rem  - 独立模式：独立窗口运行，安全退出，不影响集群
rem    子进程 stdout/stderr 重定向 bin\logs\<日期>-<时间>.log
rem  - 观察模式：start-cluster.bat 已拉起集群也没关系，端口探测自动识别
rem  需要 env.server 提供 NINEYIN_MYSQL_DSN（角色查询/告警页面）
rem ============================================================

cd /d "%~dp0"

rem env.server (NINEYIN_MYSQL_DSN etc) - missing file is not fatal
if exist "%~dp0env.server" for /f "usebackq delims=" %%L in (`findstr /b /l /c:"set " "%~dp0env.server"`) do call %%L

set "BIN="
if exist "bin\game_panel.exe" set "BIN=%~dp0bin"
if not defined BIN if exist "game_panel.exe" set "BIN=%~dp0"
if not defined BIN if exist "%~dp0..\build\game_panel.exe" set "BIN=%~dp0..\build"
if not defined BIN (
  echo [ERROR] game_panel.exe not found in bin\, .\ or ..\build
  echo         Run build.bat first.
  pause
  exit /b 1
)

start "gm panel" /min /d "%BIN%" "game_panel.exe"
echo GM panel: http://127.0.0.1:19064  (closed window = panel exit; cluster survives)
endlocal
