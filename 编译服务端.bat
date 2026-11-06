@echo off
setlocal EnableExtensions
title Build Nine Yin Go Server
cd /d "%~dp0"

where go >nul 2>&1
if errorlevel 1 (
  echo ERROR: Go was not found in PATH.
  echo Install Go 1.23 or newer, then run this script again.
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
go build -trimpath -o "%~dp0build\9yin-game-native-menu.exe" ./cmd/protocol-probe
if errorlevel 1 (
  echo ERROR: build failed.
  pause
  exit /b 1
)

echo.
echo Build completed:
echo %~dp0build\9yin-game-native-menu.exe
pause
