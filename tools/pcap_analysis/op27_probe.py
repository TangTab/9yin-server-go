# -*- coding: utf-8 -*-
"""0x27 全包解码走查：共享池时序 + 0x46 明文帧，对 0x09 属性表列型试探走查。"""
import sys, os, struct, glob
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, r'E:\Workspace\project\game\9yin-go-server1\tools\op28')
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

PT = {}
for line in open(r'E:\Workspace\project\game\9yin-go-server1\internal\clientdata\official_property_table_2433.txt', encoding='utf-8', errors='replace'):
    parts = line.rstrip('\n').split('\t')
    if len(parts) >= 3 and parts[0].isdigit():
        PT[int(parts[0])] = (parts[1], int(parts[2]))
WIDTH = {0: 0, 1: 1, 2: 2, 3: 4, 4: 4, 5: 8, 6: 4, 9: 4, 10: 4, 11: 4}

def try_walk_pairs(p, pos):
    """按 [idx u16][列型值] 走查到帧尾；返回 (pairs, ok)"""
    pairs = []
    try:
        while pos < len(p):
            idx = struct.unpack_from('<H', p, pos)[0]
            pos += 2
            name, t = PT.get(idx, (None, None))
            if name is None:
                return pairs, False
            if t in (7, 8):
                n = struct.unpack_from('<I', p, pos)[0]
                if n > 8192 or pos + 4 + n > len(p):
                    return pairs, False
                v = p[pos + 4:pos + 4 + n].rstrip(b'\x00').decode('gbk', 'replace')
                pos += 4 + n
            else:
                n = WIDTH.get(t)
                if n is None or pos + n > len(p):
                    return pairs, False
                v = int.from_bytes(p[pos:pos + n], 'little')
                pos += n
            pairs.append((idx, name, v))
    except struct.error:
        return pairs, False
    return pairs, pos == len(p)

OFFICIAL = r'E:\Workspace\project\game\adata\official'
for fname in ['c.pcapng', 'k.pcapng', 'l.pcapng', '登陆.pcapng']:
    path = os.path.join(OFFICIAL, fname)
    evs = []
    for st in build_streams(path):
        for ts, fr in stream_frames(st.s2c):
            evs.append((ts, fr))
    evs.sort(key=lambda e: e[0])
    w = World()
    items = []
    for i, (ts, fr) in enumerate(evs):
        if not fr or fr[0] != 0x27 or len(fr) < 2:
            continue
        mode = fr[1]
        if mode == 0x46:
            outlen = struct.unpack_from('<I', fr, 6)[0]
            items.append((i, ts, bytes(fr[10:10 + outlen]), '46'))
        elif mode == 0x47:
            try:
                outlen = struct.unpack_from('<I', fr, 6)[0]
                if outlen > 0xFFFF:
                    raise DecodeError('big')
                out = decompress(fr[1:], outlen, w)
                items.append((i, ts, bytes(out), '47'))
            except DecodeError:
                pass
    print('=== %s: 0x27 可解 %d 帧 ===' % (fname, len(items)))
    shown = 0
    for i, ts, body, src in items:
        # 头尝试: [obj u32][own u32] (0x28 风格) + pairs @8
        pairs8, ok8 = try_walk_pairs(body, 8)
        pairs9, ok9 = try_walk_pairs(body, 9)
        tag = 'PAIRS@8' if ok8 else ('PAIRS@9' if ok9 else '?')
        print('[%5d] +%.1fs %-2s len=%-4d 头=%s %s 首对=%s' % (
            i, 0, src, len(body), body[:12].hex(' '), tag,
            '%d=%s' % (pairs8[0][0], pairs8[0][2]) if pairs8 and ok8 else
            ('%d=%s' % (pairs9[0][0], pairs9[0][2]) if pairs9 and ok9 else '')))
        if shown == 0 and (ok8 or ok9):
            pr = pairs8 if ok8 else pairs9
            for idx, name, v in pr[:14]:
                print('        %-5d %-26s = %s' % (idx, name, str(v)[:56]))
        shown += 1
        if shown >= 6:
            break
