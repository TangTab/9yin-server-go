# -*- coding: utf-8 -*-
"""0x28 四包全景（帧数/对象/字段分布）+ 0x27 变长体细化走查。"""
import sys, os, struct
from collections import Counter
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

def walk28(p):
    """0x28 v3 语法: [obj u32][own u32][52B 块][n u16][[idx u16][列型值]×n]"""
    obj, own = struct.unpack_from('<II', p, 0)
    n = struct.unpack_from('<H', p, 60)[0]
    pos = 62
    pairs = []
    for _ in range(n):
        idx = struct.unpack_from('<H', p, pos)[0]
        pos += 2
        name, t = PT.get(idx, (None, None))
        if name is None:
            raise ValueError('bad idx %d @%#x' % (idx, pos))
        if t in (7, 8):
            ln = struct.unpack_from('<I', p, pos)[0]
            if ln > 8192 or pos + 4 + ln > len(p):
                raise ValueError('bad len')
            v = p[pos+4:pos+4+ln].rstrip(b'\x00').decode('gbk', 'replace')
            pos += 4 + ln
        else:
            w = WIDTH.get(t)
            if w is None or pos + w > len(p):
                raise ValueError('bad col')
            v = int.from_bytes(p[pos:pos+w], 'little')
            pos += w
        pairs.append((idx, name, v))
    return (obj, own), pairs, pos == len(p)

OFFICIAL = r'E:\Workspace\project\game\adata\official'
print('=== ① 0x28 四包全景 ===')
for fname in ['c.pcapng', 'k.pcapng', 'l.pcapng', '登陆.pcapng']:
    evs = []
    for st in build_streams(os.path.join(OFFICIAL, fname)):
        for ts, fr in stream_frames(st.s2c):
            evs.append((ts, fr))
    evs.sort(key=lambda e: e[0])
    w = World()
    n_all = n_dec = n_ok = 0
    objs = Counter()
    idx_c = Counter()
    fail_ex = None
    for i, (ts, fr) in enumerate(evs):
        if not fr or fr[0] != 0x28 or len(fr) < 2:
            continue
        n_all += 1
        mode = fr[1]
        try:
            if mode == 0x46:
                outlen = struct.unpack_from('<I', fr, 6)[0]
                body = bytes(fr[10:10+outlen])
            elif mode == 0x47:
                outlen = struct.unpack_from('<I', fr, 6)[0]
                if outlen > 0xFFFF:
                    raise DecodeError('big')
                body = bytes(decompress(fr[1:], outlen, w))
            else:
                continue
            n_dec += 1
            (obj, own), pairs, exact = walk28(body)
            if exact:
                n_ok += 1
                objs[(obj, own)] += 1
                for idx, name, v in pairs:
                    idx_c[idx] += 1
        except (DecodeError, ValueError, struct.error) as e:
            if fail_ex is None:
                fail_ex = str(e)
    print('%s: 0x28×%d 解码%d 走查精确%d | 对象 top4 %s | 字段idx top6 %s' % (
        fname, n_all, n_dec, n_ok,
        [('%#x/%#x' % k, v) for k, v in objs.most_common(4)],
        [(PT[i][0], v) for i, v in idx_c.most_common(6) if i in PT]))
    if fail_ex:
        print('   首个失败: %s' % fail_ex)
