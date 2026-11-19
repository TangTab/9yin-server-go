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
  echo        ^(use "build.bat fast" to skip tests during iteration^)
  goto wait_exit
)

:do_build
echo Building all packages...
rem 扫描 cmd\ 与 tools\ 下所有含 Go 源码的目录，逐个编译为
rem build\game_<包名>.exe。新增目录无需改本脚本，自动纳入。
set "OUT_LIST="
for /d %%D in ("%~dp0cmd\*") do call :build_one "%%~fD" "%%~nxD"
for /d %%D in ("%~dp0tools\*") do call :build_one "%%~fD" "%%~nxD"

if not defined OUT_LIST (
  echo ERROR: no buildable packages found under cmd\ or tools\.
  goto wait_exit
)

rem 清理陈旧产物：删除本轮没有产出的 *.exe（旧入口改名后残留在 build\，
rem 会被 deploy.bat 原样发走；如早期未加前缀的 entry.exe / solo.exe）。
rem 经 call 子例程传参比对——for 变量不过管道（右侧是子 cmd，%%~nxF 失效）。
for %%F in ("%~dp0build\*.exe") do call :cleanup_one "%%~nxF"

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
  goto wait_exit
)
set "OUT_LIST=%OUT_LIST% game_%~2.exe "
exit /b 0

:wait_exit
echo Build failed - closing in 15 seconds, press any key to close now...
timeout /t 15
exit /b 1
