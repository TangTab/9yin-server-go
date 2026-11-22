@echo off
setlocal EnableExtensions
title Build Nine Yin Go Server
cd /d "%~dp0"

where go >nul 2>&1
if errorlevel 1 (
  echo ERROR: Go was not found in PATH.
  echo Install Go 1.24 or newer, then run this script again.
  goto wait_exit
)

if not exist "%~dp0build" mkdir "%~dp0build"

if /i "%~1"=="fast" goto do_build
echo Running tests...
go test ./...
if errorlevel 1 (
  echo ERROR: tests failed; server was not rebuilt.
  echo        ^(use "build.bat fast" to skip tests during iteration, or
  echo        "build.bat fast <pkg>" to build a single package^)
  goto wait_exit
)

:do_build
rem 目标参数（可选）：增量模式只编指定包，如
rem   build.bat fast solounite npcprobe   （跳测试+只编两个包）
rem   build.bat solounite                 （先跑测试+只编一个包）
rem 名字可带 game_ 前缀，按 cmd\ 与 tools\ 下目录名定位。无目标=全量。
set "TARGETS="
for %%A in (%*) do call :add_target %%~A
if defined TARGETS goto selective_build

echo Building all packages ^(single go build invocation^)...
rem 单次 go build 把 cmd\ 与 tools\ 全部包编译到 build\_stage\，再改名落位
rem build\game_<包名>.exe。新增目录无需改本脚本，自动纳入。
rem 单发省去每包一次 go 进程的固定开销：39 包约 62s 降到 10s 内；
rem 编译走 GOCACHE 增量，未改动的包不重编。
set "OUT_LIST="
if exist "%~dp0build\_stage" rmdir /s /q "%~dp0build\_stage"
mkdir "%~dp0build\_stage"
go build -trimpath -o "%~dp0build\_stage" ./cmd/... ./tools/...
if errorlevel 1 (
  echo ERROR: build failed.
  rmdir /s /q "%~dp0build\_stage"
  goto wait_exit
)
for %%F in ("%~dp0build\_stage\*.exe") do call :place_one "%%~nxF" || goto wait_exit
rmdir /s /q "%~dp0build\_stage"
if not defined OUT_LIST (
  echo ERROR: no buildable packages found under cmd\ or tools\.
  goto wait_exit
)

rem 清理陈旧产物：删除本轮没有产出的 *.exe（旧入口改名后残留在 build\，
rem 会被 deploy.bat 原样发走；如早期未加前缀的 entry.exe / solo.exe）。
rem 经 call 子例程传参比对——for 变量不过管道（右侧是子 cmd，%%~nxF 失效）。
for %%F in ("%~dp0build\*.exe") do call :cleanup_one "%%~nxF"
goto build_summary

:selective_build
echo Building selected packages:%TARGETS%
set "OUT_LIST="
for %%A in (%TARGETS%) do call :build_target %%~A || goto wait_exit
if not defined OUT_LIST (
  echo ERROR: nothing built - no matching package under cmd\ or tools\.
  goto wait_exit
)
rem 增量模式不做陈旧清理：本轮 OUT_LIST 只含指定包，清理会误删其余产物。

:build_summary
echo.
echo Build completed:
echo %~dp0build\
for %%F in (%OUT_LIST%) do echo   %%F

echo.
echo Startup:
echo   server-cmd\start-single.bat   单进程合一（game_solounite）
echo   server-cmd\start-cluster.bat  集群（官方启停序九进程；set MEMBERS=12 多场景服）
echo   server-cmd\stop-cluster.bat   集群停机（官方 kill.bat 逆序）

rem Success: short countdown so a double-clicked window keeps the result
rem readable, then exit 0. Must NOT fall through into :wait_exit (= exit 1).
echo Build OK - closing in 5 seconds, press any key to close now...
timeout /t 5
exit /b 0

:cleanup_one
rem %1 = 产物名；不在本轮 OUT_LIST（首尾空格定界）即陈旧产物，删除。
echo %OUT_LIST% | findstr /i /l /c:" %~1 " >nul
if errorlevel 1 (
  echo   cleanup stale %~1
  del /f /q "%~dp0build\%~1" >nul 2>&1
)
exit /b 0

:build_one
rem %1 = 包目录绝对路径, %2 = 包名；产物统一 game_<包名>.exe。
if not exist "%~1\*.go" exit /b 0
echo   building game_%~2.exe ...
go build -trimpath -o "%~dp0build\game_%~2.exe" "%~1"
if errorlevel 1 (
  echo ERROR: build failed for %~2.
  exit /b 1
)
set "OUT_LIST=%OUT_LIST% game_%~2.exe "
exit /b 0
:add_target
rem %1 = 命令行参数；fast 是开关不算目标。
if /i not "%~1"=="fast" set "TARGETS=%TARGETS% %~1"
exit /b 0

:place_one
rem %1 = _stage 产物 <包名>.exe；改名落位 build\game_<包名>.exe 并记入 OUT_LIST。
move /y "%~dp0build\_stage\%~1" "%~dp0build\game_%~1" >nul
if errorlevel 1 (
  rem 占用（如 game_panel.exe 在跑）不阻塞整体：保留旧产物继续。
  echo   [WARN] cannot update game_%~1 - file in use? previous exe kept.
)
rem 无论成功与否都记入 OUT_LIST：失败的包保留旧 exe，否则会被陈旧清理误删。
set "OUT_LIST=%OUT_LIST% game_%~1 "
exit /b 0
set "OUT_LIST=%OUT_LIST% game_%~1 "
exit /b 0

:build_target
rem %1 = 目标名；剥掉 game_ 前缀后在 cmd\ 与 tools\ 下定位，单包构建。
set "TNAME=%~1"
if /i "%TNAME:~0,5%"=="game_" set "TNAME=%TNAME:~5%"
if exist "%~dp0cmd\%TNAME%\*.go" goto target_in_cmd
if exist "%~dp0tools\%TNAME%\*.go" goto target_in_tools
echo ERROR: target not found under cmd\ or tools\: %TNAME%
exit /b 1
:target_in_cmd
call :build_one "%~dp0cmd\%TNAME%" "%TNAME%"
exit /b
:target_in_tools
call :build_one "%~dp0tools\%TNAME%" "%TNAME%"
exit /b

:wait_exit
echo Build failed - closing in 15 seconds, press any key to close now...
timeout /t 15
exit /b 1
