@echo off
set "ROOT=%~dp0"
set "NINEYIN_GM_TOKEN="
findstr /b /r /c:"set .*NINEYIN_GM_TOKEN" "%ROOT%..\env.server"
for /f "usebackq delims=" %%L in (`findstr /b /r /c:"set .*NINEYIN_GM_TOKEN" "%ROOT%..\env.server"`) do call %%L
echo loaded-def=%NINEYIN_GM_TOKEN:~0,3%
