@echo off
setlocal EnableExtensions
title Build Nine Yin Go Server
cd /d "%~dp0"

where go >nul 2>&1
if errorlevel 1 (
  echo ERROR: Go was not found in PATH.
  echo Install Go 1.24 or newer, then run this script again.
  pause
  exit /b 1
)

if not exist "%~dp0build" mkdir "%~dp0build"

echo Running tests...
go test ./...
if errorlevel 1 (
  echo ERROR: tests failed; server was not rebuilt.
  pause
  exit /b 1
)

echo Building server...
rem 同一编译时间戳打进两个进程：gm-panel 会与游戏服 /internal/v1/health
rem 的 version 比对，提醒两边是否漏部署了其中一个。
for /f %%i in ('powershell.exe -NoProfile -Command "Get-Date -Format yyyyMMdd-HHmmss"') do set "BUILD_TS=%%i"
go build -trimpath -ldflags "-X main.gmInternalVersion=%BUILD_TS%" -o "%~dp0build\9yin-game-server.exe" ./cmd/protocol-probe
if errorlevel 1 (
  echo ERROR: build failed.
  pause
  exit /b 1
)

echo Building standalone GM panel...
go build -trimpath -ldflags "-X main.panelVersion=%BUILD_TS%" -o "%~dp0build\gm-panel.exe" ./cmd/gm-panel
if errorlevel 1 (
  echo ERROR: gm-panel build failed.
  pause
  exit /b 1
)

echo.
echo Build completed:
echo %~dp0build\9yin-game-server.exe
echo %~dp0build\gm-panel.exe
