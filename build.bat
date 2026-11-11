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

echo Building server...
rem 同一编译时间戳打进两个进程：gm-panel 会与游戏服 /internal/v1/health
rem 的 version 比对，提醒两边是否漏部署了其中一个。
for /f %%i in ('powershell.exe -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss"') do set "BUILD_TS=%%i"
go build -trimpath -ldflags "-X main.gmInternalVersion=%BUILD_TS%" -o "%~dp0build\9yin-game-server.exe" ./cmd/protocol-probe
if errorlevel 1 (
  echo ERROR: build failed.
  goto wait_exit
)

echo Building standalone GM panel...
go build -trimpath -ldflags "-X main.panelVersion=%BUILD_TS%" -o "%~dp0build\gm-panel.exe" ./cmd/gm-panel
if errorlevel 1 (
  echo ERROR: gm-panel build failed.
  goto wait_exit
)

echo Building sql-replay tool...
rem deploy.bat 会用它把 pending 迁移滚动重放到 MySQL（-embedded 用编译期
rem 嵌入的 migrations，目标机不需要 migrations 目录）。
go build -trimpath -o "%~dp0build\sql-replay.exe" ./cmd/sql-replay
if errorlevel 1 (
  echo ERROR: sql-replay build failed.
  goto wait_exit
)

echo.
echo Build completed:
echo %~dp0build\9yin-game-server.exe
echo %~dp0build\gm-panel.exe
echo %~dp0build\sql-replay.exe

rem Success: short countdown so a double-clicked window keeps the result
rem readable, then exit 0. Must NOT fall through into :wait_exit (= exit 1).
echo Build OK - closing in 5 seconds, press any key to close now...
timeout /t 5
exit /b 0

:wait_exit
echo Build failed - closing in 15 seconds, press any key to close now...
timeout /t 15
exit /b 1
