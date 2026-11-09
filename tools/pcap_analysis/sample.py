# -*- coding: utf-8 -*-
"""抽样查看未知消息/opcode 的实际内容。用法: python -I sample.py <tag> <sig前缀>..."""
import json
import os
import sys

EV = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'events')
tag = sys.argv[1]
pats = sys.argv[2:]
ev = json.load(open(os.path.join(EV, tag + '.json'), encoding='utf-8'))
shown = {}
for e in ev:
    for p in pats:
        if p in e['sig'] and shown.get(e['sig'], 0) < 3:
            shown[e['sig']] = shown.get(e['sig'], 0) + 1
            print('%9.3fs %-3s %s' % (e['t'] / 1e6, e['dir'], e['desc']))
            break
