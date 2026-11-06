@echo off
title MySQL Server

rem 切换到脚本所在目录，保证 my.ini 和 data 相对路径生效
cd /d "%~dp0"

set "CONFIG=%~dp0mysql\my.ini"
set "DATA_DIR=%~dp0mysql\data"

if not exist "%CONFIG%" (
    echo [错误] 找不到配置文件: %CONFIG%
    pause
    exit /b 1
)

if not exist "%DATA_DIR%" (
    echo [错误] 找不到数据目录: %DATA_DIR%
    pause
    exit /b 1
)

echo [信息] 配置文件: %CONFIG%
echo [信息] 数据目录: %DATA_DIR%
echo [信息] 正在启动 MySQL，关闭本窗口即停止...
echo.

rem --defaults-file 必须是第一个参数，--console 让日志输出到当前窗口
"%~dp0mysql\bin\mysqld.exe" --defaults-file="%CONFIG%" --datadir="%DATA_DIR%" --console

echo.
echo [信息] MySQL 进程已退出。
pause
