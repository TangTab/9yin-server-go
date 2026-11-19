@echo off

setlocal EnableExtensions

title 9yin Go - cluster stop



rem 官方 kill.bat 逆序：member→entry→lister→public→store→world→log→auction→monitor

rem member 多实例共用同一 exe 名（-member N 区分 MemberID），

rem 按映像名一次回收全部；taskkill 缺进程静默；单进程形态也一并回收（幂等）。

taskkill /f /im game_solomember.exe  >nul 2>&1

taskkill /f /im game_soloentry.exe   >nul 2>&1

taskkill /f /im game_sololister.exe  >nul 2>&1

taskkill /f /im game_solopublic.exe  >nul 2>&1

taskkill /f /im game_store.exe       >nul 2>&1

taskkill /f /im game_soloworld.exe   >nul 2>&1

taskkill /f /im game_logserver.exe   >nul 2>&1

taskkill /f /im game_auction.exe     >nul 2>&1

taskkill /f /im game_monitor.exe     >nul 2>&1

taskkill /f /im game_solounite.exe   >nul 2>&1

echo Cluster stopped.

pause

