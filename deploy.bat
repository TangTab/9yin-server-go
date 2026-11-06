@echo off
setlocal EnableExtensions
title Nineyin Deploy

rem ============================================================
rem  Nine Yin Go server deploy script.
rem
rem  Steps:
rem    [1] Load NINEYIN_SERVER_ROOT from env.server
rem        (an already-defined environment variable takes priority)
rem    [2] Copy build\* into the target root, overwriting existing files
rem    [3] Copy server-cmd\* into the target root only when missing,
rem        so local edits on the target side survive redeploys
rem
rem  Usage: deploy.bat
rem    Run build.bat first; this script only ships what is in build\.
rem ============================================================

cd /d "%~dp0"

rem ---------- [1/3] Load NINEYIN_SERVER_ROOT from env.server ----------
if defined NINEYIN_SERVER_ROOT goto root_loaded
if not exist "env.server" (
    echo [ERROR] NINEYIN_SERVER_ROOT is not defined and env.server was not found.
    echo         Copy env.server.default to env.server and fill in the target path.
    pause
    exit /b 1
)
rem Execute only the lines that start with "set " so comments (::) are skipped.
for /f "usebackq delims=" %%L in (`findstr /b /l /c:"set " "env.server"`) do call %%L
:root_loaded
if not defined NINEYIN_SERVER_ROOT (
    echo [ERROR] NINEYIN_SERVER_ROOT is not set in env.server.
    pause
    exit /b 1
)
if not exist "%NINEYIN_SERVER_ROOT%" (
    echo [ERROR] Target folder does not exist: %NINEYIN_SERVER_ROOT%
    echo         Check NINEYIN_SERVER_ROOT in env.server.
    pause
    exit /b 1
)

echo ==========================================
echo   Nine Yin Go server deploy
echo ==========================================
echo   Target : %NINEYIN_SERVER_ROOT%
echo ==========================================
echo.

rem ---------- [2/3] build\* -> target, overwrite ----------
echo [2/3] Deploying build\* (overwrite)...
if not exist "build\*" (
    echo [ERROR] build\ is empty. Run build.bat first.
    pause
    exit /b 1
)
set /a BUILD_N=0
for %%f in ("%~dp0build\*") do (
    call :deploy_build_file "%%~ff" "%%~nxf"
    if errorlevel 1 goto deploy_fail
    set /a BUILD_N+=1
)

rem ---------- [3/3] server-cmd\* -> target, only if missing ----------
echo [3/3] Deploying server-cmd\* (skip existing)...
set /a CMD_NEW=0
set /a CMD_SKIP=0
for %%f in ("%~dp0server-cmd\*") do (
    call :deploy_cmd_file "%%~ff" "%%~nxf"
    if errorlevel 1 goto deploy_fail
)

echo.
echo ==========================================
echo Deploy finished.
echo   build     : %BUILD_N% file(s) copied (existing overwritten)
echo   server-cmd: %CMD_NEW% added, %CMD_SKIP% skipped (already present)
echo   Target    : %NINEYIN_SERVER_ROOT%
echo   Start with: %NINEYIN_SERVER_ROOT%\start-9yin.bat
echo ==========================================
endlocal
exit /b 0

:deploy_fail
echo.
echo [ERROR] Deploy aborted.
pause
exit /b 1

rem ================= helpers =================

rem Copy one build artifact into the target root, overwriting.
:deploy_build_file
if not exist "%~1" exit /b 0
if exist "%NINEYIN_SERVER_ROOT%\%~2" (set "TAG=overwrite") else set "TAG=new"
copy /y "%~1" "%NINEYIN_SERVER_ROOT%\" >nul
if errorlevel 1 (
    echo [ERROR] Failed to copy %~2. If the server exe is running, close it first.
    exit /b 1
)
echo   [build] %~2 (%TAG%)
exit /b 0

rem Copy one launcher script into the target root only when missing.
:deploy_cmd_file
if not exist "%~1" exit /b 0
if exist "%NINEYIN_SERVER_ROOT%\%~2" (
    echo   [server-cmd] %~2 - exists, skipped
    set /a CMD_SKIP+=1
    exit /b 0
)
copy /y "%~1" "%NINEYIN_SERVER_ROOT%\" >nul
if errorlevel 1 (
    echo [ERROR] Failed to copy %~2.
    exit /b 1
)
echo   [server-cmd] %~2 - copied
set /a CMD_NEW+=1
exit /b 0
