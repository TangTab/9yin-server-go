@echo off
setlocal EnableExtensions
title Nine Yin GM Panel

set "ROOT=%~dp0"
rem --- 独立启动 gm-panel(19064)，不重启游戏服 ---
rem 部署根目录：本脚本与 gm-panel.exe/env.server 同级（deploy.bat 下发）。
rem 开发仓库：本脚本在 server-cmd\ 下，exe 与 env.server 在上一级。

rem --- token 优先级与 start-9yin.bat 一致：系统环境变量 > 本目录 env.server > 仓库 env.server > 部署目录 env.server ---
rem findstr 用正则 set .*NINEYIN_GM_TOKEN：同时命中带引号(set "VAR=..")与不带引号两种写法
if not defined NINEYIN_GM_TOKEN if exist "%ROOT%env.server" for /f "usebackq delims=" %%L in (`findstr /b /r /c:"set .*NINEYIN_GM_TOKEN" "%ROOT%env.server"`) do call %%L
if not defined NINEYIN_GM_TOKEN if exist "%ROOT%..\env.server" for /f "usebackq delims=" %%L in (`findstr /b /r /c:"set .*NINEYIN_GM_TOKEN" "%ROOT%..\env.server"`) do call %%L
rem 仓库 env.server 的 NINEYIN_SERVER_ROOT 指向部署目录，那里可能有 start-9yin.bat 已生成的 token
if not defined NINEYIN_SERVER_ROOT if exist "%ROOT%..\env.server" for /f "usebackq delims=" %%L in (`findstr /b /r /c:"set .*NINEYIN_SERVER_ROOT" "%ROOT%..\env.server"`) do call %%L
if not defined NINEYIN_GM_TOKEN if defined NINEYIN_SERVER_ROOT if exist "%NINEYIN_SERVER_ROOT%\env.server" for /f "usebackq delims=" %%L in (`findstr /b /r /c:"set .*NINEYIN_GM_TOKEN" "%NINEYIN_SERVER_ROOT%\env.server"`) do call %%L

rem --- 19063 上已有游戏服：先试着从它的进程命令行恢复 -gm-internal-token ---
rem 恢复不到（token 来自环境变量或根本没配）就不能瞎生成，否则面板永远
rem 401/503；报错并给出重启路径。PowerShell 里只用单引号，避免 for /f
rem 反引号层的双引号转义问题。
if not defined NINEYIN_GM_TOKEN (
  powershell.exe -NoProfile -Command "$c=New-Object Net.Sockets.TcpClient; try{$c.Connect('127.0.0.1',19063); exit 0}catch{exit 1}finally{$c.Close()}"
  if not errorlevel 1 (
    for /f "usebackq delims=" %%T in (`powershell.exe -NoProfile -Command "$p = Get-CimInstance Win32_Process | Where-Object { $_.Name -eq '9yin-game-server.exe' -and $_.CommandLine -match '-gm-internal-token' }; if ($p) { if ($p.CommandLine -match '(?i)-gm-internal-token(?:=|\s)[^\w-]?([0-9a-zA-Z_-]{8,})') { $Matches[1] } }"`) do set "NINEYIN_GM_TOKEN=%%T"
    if defined NINEYIN_GM_TOKEN (
      echo GM token recovered from the running game server command line.
    ) else (
      echo ERROR: a game server is listening on 19063, but its token could
      echo not be recovered from its command line. Restart it via
      echo start-9yin.bat ^(shares NINEYIN_GM_TOKEN via env.server^),
      echo then run this script again.
      pause
      exit /b 1
    )
  )
)

rem --- 首次运行：生成 token 并幂等追加到最近的 env.server（与 start-9yin.bat 同款）---
rem 注意：生成块里只 set 变量；落盘放到块外的顶层语句做——括号块内的
rem %NINEYIN_GM_TOKEN% 在解析期展开（此刻还是空），会写入空 token。
set "GM_TOKEN_FRESH="
if not defined NINEYIN_GM_TOKEN (
  for /f "delims=" %%G in ('powershell.exe -NoProfile -Command "[guid]::NewGuid().ToString('N')"') do (
    set "NINEYIN_GM_TOKEN=%%G"
    set "GM_TOKEN_FRESH=1"
  )
)
if defined GM_TOKEN_FRESH (
  if exist "%ROOT%env.server" (
    >>"%ROOT%env.server" echo set "NINEYIN_GM_TOKEN=%NINEYIN_GM_TOKEN%"
    echo GM token generated and appended to env.server ^(script directory^).
  ) else if exist "%ROOT%..\env.server" (
    >>"%ROOT%..\env.server" echo set "NINEYIN_GM_TOKEN=%NINEYIN_GM_TOKEN%"
    echo GM token generated and appended to env.server ^(repo root^).
  ) else (
    echo GM token generated per-run only ^(env.server not found^).
  )
  echo The game server must run with the SAME token: see NINEYIN_GM_TOKEN in env.server.
)

rem --- locate gm-panel.exe: deploy root first, then dev-repo build\ ---
set "PANEL=%ROOT%gm-panel.exe"
if not exist "%PANEL%" set "PANEL=%ROOT%..\build\gm-panel.exe"
if not exist "%PANEL%" (
  echo ERROR: gm-panel.exe not found:
  echo %ROOT%gm-panel.exe
  echo %ROOT%..\build\gm-panel.exe
  echo Run build.bat first.
  pause
  exit /b 1
)
for %%I in ("%PANEL%") do set "EXEDIR=%%~dpI"
if not exist "%EXEDIR%logs" mkdir "%EXEDIR%logs"

rem --- already running? reuse it, just open the page ---
rem Real TCP connect probe: Get-NetTCPConnection misses some loopback
rem listeners (same quirk start-9yin.bat works around for MySQL).
powershell.exe -NoProfile -Command "$c=New-Object Net.Sockets.TcpClient; try{$c.Connect('127.0.0.1',19064); exit 0}catch{exit 1}finally{$c.Close()}"
if not errorlevel 1 (
  echo gm-panel is already running on 19064.
  start "Nine Yin GM Panel" http://127.0.0.1:19064/
  exit /b 0
)

rem --- informational probe: panel tolerates a down game server, users do not ---
powershell.exe -NoProfile -Command "$c=New-Object Net.Sockets.TcpClient; try{$c.Connect('127.0.0.1',19063); exit 0}catch{exit 1}finally{$c.Close()}"
if errorlevel 1 echo NOTE: game server internal API 19063 is not reachable yet; the panel will keep serving and retry per request.

echo.
echo Starting standalone GM panel...
echo GM UI : http://127.0.0.1:19064/
echo Game  : 127.0.0.1:19063 (internal API)
echo Log   : %EXEDIR%logs\gm-panel.current.log
echo.

start "Nine Yin GM Panel 19064" /D "%EXEDIR%" "%PANEL%" -listen 127.0.0.1:19064 -game-addr 127.0.0.1:19063 -game-token "%NINEYIN_GM_TOKEN%" 1>>"%EXEDIR%logs\gm-panel.current.log" 2>>&1

timeout /t 1 /nobreak >nul
rem start "Nine Yin GM Panel" http://127.0.0.1:19064/

echo Launcher finished. Keep the panel window open.
rem pause
