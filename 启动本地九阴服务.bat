@echo off
setlocal EnableExtensions
title Nine Yin Portable Server

set "ROOT=%~dp0"
set "NINEYIN_SERVER_ROOT=%ROOT%"
set "SERVER=%ROOT%9yin-game-native-menu.exe"
set "LOGDIR=%ROOT%logs"
set "LISTER=%ROOT%loopback-lister.ps1"

if not exist "%SERVER%" (
  echo ERROR: server executable not found:
  echo %SERVER%
  pause
  exit /b 1
)
if not exist "%LISTER%" (
  echo ERROR: lister script not found:
  echo %LISTER%
  pause
  exit /b 1
)
if not exist "%ROOT%resources\modern\share" (
  echo ERROR: runtime resources not found:
  echo %ROOT%resources\modern\share
  pause
  exit /b 1
)
if not exist "%ROOT%data\roles.json" (
  echo ERROR: role data not found:
  echo %ROOT%data\roles.json
  pause
  exit /b 1
)
if not exist "%LOGDIR%" mkdir "%LOGDIR%"

powershell.exe -NoProfile -Command "$listeners = @(Get-NetTCPConnection -State Listen -LocalPort 19061,19062 -ErrorAction SilentlyContinue); if ($listeners.Count -eq 0) { exit 0 }; $owners = @($listeners.OwningProcess | Sort-Object -Unique); if ($owners.Count -eq 1) { $process = Get-CimInstance Win32_Process -Filter ('ProcessId=' + $owners[0]); if ($process.ExecutablePath -and [IO.Path]::GetFullPath($process.ExecutablePath) -eq [IO.Path]::GetFullPath($env:SERVER)) { exit 2 } }; exit 1"
set "PORT_STATE=%ERRORLEVEL%"
if "%PORT_STATE%"=="2" (
  echo Nine Yin server copy is already running.
  start "Nine Yin GM Panel" http://127.0.0.1:19062/
  exit /b 0
)
if "%PORT_STATE%"=="1" (
  echo ERROR: game or GM port is already in use.
  echo Close the other Nine Yin game server before starting this copy.
  pause
  exit /b 1
)

echo.
echo Starting portable Nine Yin local services...
echo Server list : 127.0.0.1:4000
echo Game  : 127.0.0.1:19061
echo GM UI : http://127.0.0.1:19062/
echo.

powershell.exe -NoProfile -Command "if (Get-NetTCPConnection -State Listen -LocalPort 4000 -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }"
if errorlevel 1 (
  start "Nine Yin Server List 4000" /D "%ROOT%" powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%LISTER%" -Port 4000
) else (
  echo Server list 4000 is already running; reusing it.
)

start "Nine Yin Game 19061" /D "%ROOT%" "%SERVER%" -listen 127.0.0.1:19061 -gm-listen 127.0.0.1:19062 -log-file "%LOGDIR%\protocol-probe-live.log" 1>>"%LOGDIR%\game-19061.current.log" 2>>&1

timeout /t 2 /nobreak >nul
start "Nine Yin GM Panel" http://127.0.0.1:19062/

echo Launcher finished. Keep the two server windows open.
pause
