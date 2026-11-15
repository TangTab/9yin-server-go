#!/usr/bin/env python3
"""常驻提权 worker v2：tmp/elev_cmd.txt（首行=脚本，其余=参数行），
tmp/elev_kill.txt 存在则退出。子进程限时 90s（探针类脚本自带轮询落盘，
不依赖被 worker 回收）。输出追加 tmp/elev_out.txt。"""
import os
import subprocess
import sys
import time

ROOT = r'G:\pojie\ai\9yin-server-go\tmp'
CMD = os.path.join(ROOT, 'elev_cmd.txt')
OUT = os.path.join(ROOT, 'elev_out.txt')
KILL = os.path.join(ROOT, 'elev_kill.txt')

with open(OUT, 'a', encoding='utf-8') as f:
    f.write(f'=== worker2 start {time.strftime("%H:%M:%S")} pid={os.getpid()}\n')

while True:
    if os.path.exists(KILL):
        os.remove(KILL)
        with open(OUT, 'a', encoding='utf-8') as f:
            f.write('=== worker2 exit by kill file\n')
        break
    if os.path.exists(CMD):
        try:
            lines = open(CMD, encoding='utf-8').read().splitlines()
            os.replace(CMD, CMD + '.done')
            if lines and lines[0].strip():
                with open(OUT, 'a', encoding='utf-8') as f:
                    f.write(f'--- run {time.strftime("%H:%M:%S")}: {lines}\n')
                p = subprocess.run([sys.executable] + [l.strip() for l in lines if l.strip()],
                                   capture_output=True, timeout=90)
                with open(OUT, 'ab') as f:
                    f.write(p.stdout)
                    f.write(p.stderr)
                    f.write(b'\n--- exit %d\n' % p.returncode)
        except subprocess.TimeoutExpired:
            with open(OUT, 'a', encoding='utf-8') as f:
                f.write('--- timeout 90s (child keeps running detached)\n')
        except Exception as e:
            with open(OUT, 'a', encoding='utf-8') as f:
                f.write(f'!!! {e}\n')
    time.sleep(1)
