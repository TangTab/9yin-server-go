@echo off
setlocal EnableExtensions
title NPC Res Probe
chcp 65001 >nul

cd /d "%~dp0"

where go >nul 2>&1
if errorlevel 1 (
  echo ERROR: Go was not found in PATH.
  echo Install Go 1.24 or newer, then run this script again.
  goto wait_exit
)

rem ============================================================
rem  NPC 目录装载探针（tools/npcprobe 只读包装，不占互斥体）。
rem
rem  用法: build-res.bat [数据根] [npcprobe 余参...]
rem    %1 = 数据根（须含 ini\Npc\NpcConfig 与 ini\Scene），
rem         缺省 = 正式部署 go2 的 Res；
rem    %2..%5 透传 npcprobe（如 -scene born01 -radius 800 -limit 50）。
rem  例: build-res.bat
rem      build-res.bat F:/Game/Online/9yin/9yin-server-go2/Res -scene born01
rem      build-res.bat ..\testdata            （仓内测试档）
rem
rem  本脚本只做装载探针。构建数据（NPC / 任务 / 物品 / 技能）走统一入口：
rem      go run ./tools/datagen            （先落散文件到 testdata，再并入资源层）
rem ============================================================

set "RES_ROOT=%~1"
if not defined RES_ROOT set "RES_ROOT=F:/Game/Online/9yin/9yin-server-go2/Res"

if not exist "%RES_ROOT%/ini/Npc/NpcConfig" (
  echo ERROR: invalid data root - need ini\Npc\NpcConfig : %RES_ROOT%
  goto wait_exit
)

echo npc probe  root=%RES_ROOT%  extra: %2 %3 %4 %5
go run ./tools/npcprobe -root "%RES_ROOT%" %2 %3 %4 %5
if errorlevel 1 (
  echo ERROR: npcprobe failed for %RES_ROOT%.
  goto wait_exit
)

echo.
echo Probe OK - closing in 5 seconds, press any key to close now...
timeout /t 5 >nul
exit /b 0

:wait_exit
echo Probe failed - closing in 15 seconds, press any key to close now...
timeout /t 15
exit /b 1
