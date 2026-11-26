@echo off
setlocal EnableExtensions
title 9yin Go - single (solounite)

rem ============================================================
rem  单进程合一形态：game_solounite 一个进程承载官方
rem  game_unite 七实体（world/store/public/member/auction/logger/entry）
rem  官方拓扑见 docs/official/game_unite与game_world全解与Go复刻.md。
rem
rem  二进制定位顺序：本目录 bin\ → 本目录 → 仓库 build\（开发机直跑）。
rem  工作目录 = 二进制所在目录（ini 缺失时由进程释放内置官方快照）。
rem ============================================================

cd /d "%~dp0"

rem env.server (NINEYIN_MYSQL_DSN etc) - missing file is not fatal
if exist "%~dp0env.server" for /f "usebackq delims=" %%L in (`findstr /b /l /c:"set " "%~dp0env.server"`) do call %%L
set "BIN="
if exist "bin\game_solounite.exe" set "BIN=%~dp0bin"
if not defined BIN if exist "game_solounite.exe" set "BIN=%~dp0"
if not defined BIN if exist "%~dp0..\build\game_solounite.exe" set "BIN=%~dp0..\build"
if not defined BIN (
  echo [ERROR] game_solounite.exe not found in bin\, .\ or ..\build\
  echo         Run build.bat first.
  pause
  exit /b 1
)

echo Target : %BIN%\game_solounite.exe
echo Workdir: %BIN%

rem 可选伴随区服列表服：官方合一形态七实体不含 lister（客户端直连
rem entry 2001）；但官方客户端编死 4000 取列表——需要时
rem `set WITHLISTER=1` 后再跑本脚本，附带拉起 game_sololister(4000)。
set WITHLISTER=1
if /i "%WITHLISTER%"=="1" (
  if exist "%BIN%\game_sololister.exe" (
    echo Sidecar: game_sololister.exe ^(Outer 4000^)
    rem /b 不开新窗：sidecar 与主进程共用本控制台（无控制台宿主下也能存活；
    rem 退场由下方 taskkill 收口）
    start "" /b /d "%BIN%" "game_sololister.exe"
  ) else (
    echo [WARN] WITHLISTER=1 但 %BIN%\game_sololister.exe 不存在，跳过
  )
) else (
  echo Lister  : 未拉起（官方合一形态无 lister；需要 4000 列表面请 set WITHLISTER=1）
)

echo.
rem 场景配置缺省 ..\res\scenes.ini（官方方言）；无则空场景，用 -scenes 覆盖。
rem Optional client-line wire capture: any non-empty NINEYIN_CAPTURE_DIR
rem dumps both directions of every client session to
rem <dir>\{ip-dashes}-{port}_c2s.bin / _s2c.bin (raw wire bytes, same
rem layout as the reference captures\local). CAPTURE=1 picks that dir here.
if /i "%CAPTURE%"=="1" if not defined NINEYIN_CAPTURE_DIR set "NINEYIN_CAPTURE_DIR=%~dp0logs\capture"
if defined NINEYIN_CAPTURE_DIR echo Capture: %NINEYIN_CAPTURE_DIR%

rem  -capture-dir "%~dp0logs\capture"
cd /d "%BIN%"
"game_solounite.exe" %*
echo.
echo solounite exited.
if /i "%WITHLISTER%"=="1" taskkill /f /im game_sololister.exe >nul 2>&1
pause
