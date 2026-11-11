# -*- coding: utf-8 -*-
"""Dump zhaoshi ini trigger_custom + action segments for given skills (GBK)."""
import re
import sys

ROOT = r'F:\Game\Online\9yin\九阴资源\resources\ini\action'
FILES = ('zhaoshi_player.ini', 'zhaoshi_player_2.ini')

skills = sys.argv[1:]
for fname in FILES:
    data = open(ROOT + '\\' + fname, 'rb').read().decode('gbk', 'replace').replace('\r\n', '\n')
    for skill in skills:
        m = re.search(r'^\[' + re.escape(skill) + r'\]$(.*?)(?=^\[|\Z)', data, re.M | re.S)
        if not m:
            continue
        print('=== %s [%s]' % (fname, skill))
        for line in m.group(1).strip().splitlines():
            line = line.strip()
            if 'trigger' in line.lower() or re.match(r'^\d+=', line):
                print('  ', line)
