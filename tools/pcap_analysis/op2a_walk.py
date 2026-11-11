# -*- coding: utf-8 -*-
"""0x2a 装备外观帧走查器：[头10B: obj/own/row/181/08][pair×N: [idx u16][0x09表列型值]]
用 official_property_table_2433.txt 列型驱动，验证 k.pcapng 全部 64 帧。"""
import sys, os, struct
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, r'E:\Workspace\project\game\9yin-go-server1\tools\op28')
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

# 0x09 属性表: idx \t name \t coltype
PT = {}
for line in open(r'E:\Workspace\project\game\9yin-go-server1\internal\clientdata\official_property_table_2433.txt',
                 encoding='utf-8', errors='replace'):
    parts = line.rstrip('\n').split('\t')
    if len(parts) >= 3 and parts[0].isdigit():
        PT[int(parts[0])] = (parts[1], int(parts[2]))
print('属性表 %d 项; 样例 1447=%s 1896=%s 1944=%s 2142=%s' % (
    len(PT), PT.get(1447), PT.get(1896), PT.get(1944), PT.get(2142)))

WIDTH = {0: 0, 1: 1, 2: 2, 3: 4, 4: 4, 5: 8, 6: 4, 9: 4, 10: 4, 11: 4}

def read_val(p, pos, t):
    if t in (7, 8):
        n = struct.unpack_from('<I', p, pos)[0]
        if n > 8192 or pos + 4 + n > len(p):
            raise ValueError('bad strlen %d @%#x' % (n, pos))
        s = p[pos + 4:pos + 4 + n]
        try:
            txt = s.rstrip(b'\x00').decode('gbk')
        except UnicodeDecodeError:
            txt = s.hex()
        return txt, pos + 4 + n
    n = WIDTH.get(t)
    if n is None:
        raise ValueError('coltype %#x' % t)
    if pos + n > len(p):
        raise ValueError('past end')
    if t == 4:
        return round(struct.unpack_from('<f', p, pos)[0], 4), pos + n
    if t == 5:
        return round(struct.unpack_from('<d', p, pos)[0], 4), pos + n
    if t == 6:
        return p[pos:pos + n].hex(), pos + n
    return int.from_bytes(p[pos:pos + n], 'little'), pos + n

def walk2a(body):
    obj, own, row, b5 = struct.unpack_from('<HHHH', body, 0)
    h8 = body[8]
    pos = 9
    pairs = []
    while pos < len(body):
        idx = struct.unpack_from('<H', body, pos)[0]
        pos += 2
        name, t = PT.get(idx, ('?', 3))
        v, pos = read_val(body, pos, t)
        pairs.append((idx, name, t, v))
    return dict(obj=obj, own=own, row=row, b5=b5, h8=h8), pairs

# k.pcapng 全 0x2a
DIR = r'E:\Workspace\project\game\adata\official\k.pcapng'
evs = []
for st in build_streams(DIR):
    for ts, fr in stream_frames(st.s2c):
        evs.append((ts, fr))
evs.sort(key=lambda e: e[0])
worlds = {}
ok47 = {}
for i, (ts, fr) in enumerate(evs):
    if not fr or fr[0] not in range(0x27, 0x31) or len(fr) < 2:
        continue
    mode = fr[1]
    if (mode & 1) == 0:
        continue
    w = worlds.setdefault(0, World())
    try:
        hdrlen = 4 if (mode & 2) else 1
        outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        if outlen > 0xFFFF:
            raise DecodeError('big')
        out = decompress(fr[1:], outlen, w)
        if fr[0] == 0x2a:
            ok47[i] = bytes(out)
    except DecodeError:
        pass

n46 = n47 = bad46 = bad47 = 0
samples = []
for i, (ts, fr) in enumerate(evs):
    if not fr or fr[0] != 0x2a or len(fr) < 2:
        continue
    mode = fr[1]
    if mode == 0x46:
        outlen = struct.unpack_from('<I', fr, 6)[0]
        body = bytes(fr[10:10 + outlen])
        src = '46'
    elif i in ok47:
        body = ok47[i]
        src = '47'
    else:
        continue
    try:
        meta, pairs = walk2a(body)
        if src == '46':
            n46 += 1
        else:
            n47 += 1
        if len(samples) < 2 and src == '46':
            samples.append((meta, pairs))
    except Exception as e:
        if src == '46':
            bad46 += 1
        else:
            bad47 += 1
        if (bad46 + bad47) <= 5:
            print('FAIL[%d] src=%s %s' % (i, src, e))
print('0x46: ok=%d bad=%d | 0x47: ok=%d bad=%d' % (n46, bad46, n47, bad47))

for meta, pairs in samples:
    print()
    print('=== 样例: obj=%#x own=%d row=%d b5=%d h8=%d ===' % (
        meta['obj'], meta['own'], meta['row'], meta['b5'], meta['h8']))
    for idx, name, t, v in pairs:
        print('  %-5d %-28s t%d = %s' % (idx, name, t, str(v)[:64]))
