@echo off
setlocal EnableExtensions
title 9yin Go - cluster

rem ============================================================
rem  集群形态：官方 solo 多进程拓扑（对齐 BGserverCJ\start.bat
rem  与 game_unite main.lua 启动序 logger→world→store→entry→
rem  public→auction→member 的依赖序合并）：
rem    1 logger   game_logserver   （日志聚合，LogShareMem 生产者）
rem    2 world    game_soloworld   （世界服）
rem    3 store    game_store       （存储枢纽）
rem    4 public   game_solopublic  （公共数据服）
rem    5 lister   game_sololister  （区服列表，客户端入口）
rem    6 entry    game_soloentry   （接入网关）
rem    7 member×N game_solomember  （场景服；N 默认 1，set MEMBERS=12 拉满）
rem    8 auction  game_auction    （拍卖行）
rem    9 monitor  game_monitor     （监控服）
rem  停机用 stop-cluster.bat（官方 kill.bat 逆序）。
rem
rem  二进制定位顺序：本目录 bin\ → 本目录 → 仓库 build\（开发机直跑）。
rem ============================================================

cd /d "%~dp0"

rem env.server (NINEYIN_MYSQL_DSN etc) - missing file is not fatal
if exist "%~dp0env.server" for /f "usebackq delims=" %%L in (`findstr /b /l /c:"set " "%~dp0env.server"`) do call %%L
set "BIN="
if exist "bin\game_soloworld.exe" set "BIN=%~dp0bin"
if not defined BIN if exist "game_soloworld.exe" set "BIN=%~dp0"
if not defined BIN if exist "%~dp0..\build\game_soloworld.exe" set "BIN=%~dp0..\build"
if not defined BIN (
  echo [ERROR] server exes not found in bin\, .\ or ..\build\
  echo         Run build.bat first.
  pause
  exit /b 1
)

if not defined MEMBERS set "MEMBERS=1"

echo Target : %BIN%
echo Members: %MEMBERS%
echo.

rem ---- 每进程独立窗口（官方 start.bat 形态），/d 指定工作目录使
rem      ini 释放/日志落在二进制同侧 ----
rem Optional client-line wire capture: any non-empty NINEYIN_CAPTURE_DIR
rem dumps both directions of every client session to
rem <dir>\{ip-dashes}-{port}_c2s.bin / _s2c.bin (raw wire bytes, same
rem layout as the reference captures\local). CAPTURE=1 picks that dir here.
if /i "%CAPTURE%"=="1" if not defined NINEYIN_CAPTURE_DIR set "NINEYIN_CAPTURE_DIR=%~dp0..\captures\local"
if defined NINEYIN_CAPTURE_DIR echo Capture: %NINEYIN_CAPTURE_DIR%

cd /d "%BIN%"
start "game_log"      /d "%BIN%" "game_logserver.exe"
timeout /t 1 /nobreak >nul
start "game_world"    /d "%BIN%" "game_soloworld.exe"
timeout /t 1 /nobreak >nul
start "game_store"    /d "%BIN%" "game_store.exe"
timeout /t 1 /nobreak >nul
start "game_public"   /d "%BIN%" "game_solopublic.exe"
timeout /t 1 /nobreak >nul
start "game_lister"   /d "%BIN%" "game_sololister.exe"
timeout /t 1 /nobreak >nul
start "game_entry"    /d "%BIN%" "game_soloentry.exe"
timeout /t 1 /nobreak >nul

rem ---- member 0..N-1（官方 member/member1..11 == MemberID 0..11）----
rem for /l 集合不做算术：end 写 %M_N%-1 会被按数字前缀解析成 M_N 多拉一个，
rem 先算好 M_END 再进循环。
set /a M_N=MEMBERS
if %M_N% LSS 1 set /a M_N=1
set /a M_END=M_N-1
for /l %%i in (0,1,%M_END%) do (
  start "game_member%%i" /d "%BIN%" "game_solomember.exe" -member %%i
  timeout /t 1 /nobreak >nul
)

start "game_auction"  /d "%BIN%" "game_auction.exe"
timeout /t 1 /nobreak >nul
start "game_monitor"  /d "%BIN%" "game_monitor.exe"

echo.
echo Cluster launched: logger world store public lister entry member×%MEMBERS% auction monitor
echo Stop with: stop-cluster.bat
endlocal
