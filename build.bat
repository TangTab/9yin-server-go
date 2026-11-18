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

echo Running tests...
go test ./...
if errorlevel 1 (
  echo ERROR: tests failed; server was not rebuilt.
  goto wait_exit
)

echo Building all packages...
rem 扫描 cmd\ 与 tools\ 下所有含 Go 源码的目录，逐个编译为
rem build\game-<包名>.exe。新增目录无需改本脚本，自动纳入。
set "OUT_LIST="
for /d %%D in ("%~dp0cmd\*") do call :build_one "%%~fD" "%%~nxD"
for /d %%D in ("%~dp0tools\*") do call :build_one "%%~fD" "%%~nxD"

if not defined OUT_LIST (
  echo ERROR: no buildable packages found under cmd\ or tools\.
  goto wait_exit
)

echo.
echo Build completed:
echo %~dp0build\
for %%F in (%OUT_LIST%) do echo   %%F

rem Success: short countdown so a double-clicked window keeps the result
rem readable, then exit 0. Must NOT fall through into :wait_exit (= exit 1).
echo Build OK - closing in 5 seconds, press any key to close now...
timeout /t 5
exit /b 0

:build_one
rem %1 = 包目录绝对路径, %2 = 包名；产物统一 game-<包名>.exe。
if not exist "%~1\*.go" exit /b 0
echo   building game_%~2.exe ...
go build -trimpath -o "%~dp0build\game_%~2.exe" "%~1"
if errorlevel 1 (
  echo ERROR: build failed for %~2.
  goto wait_exit
)
set "OUT_LIST=%OUT_LIST% game_%~2.exe"
exit /b 0

:wait_exit
echo Build failed - closing in 15 seconds, press any key to close now...
timeout /t 15
exit /b 1
