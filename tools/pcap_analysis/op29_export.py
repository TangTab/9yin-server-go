# -*- coding: utf-8 -*-
"""导出 4 包全部 0x29 ok 明文 + 上下文元数据（前帧 0x14 tag / outlen / 流号）→ pkl。"""
import sys, os, struct, pickle
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'op28'))
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

OFFICIAL = r'E:\Workspace\project\game\adata\official'
OUT = os.path.dirname(os.path.abspath(__file__))

def export(path):
    streams = build_streams(path)
    evs = []
    for sid, st in enumerate(streams):
        for ts, fr in stream_frames(st.s2c):
            evs.append((ts, sid, fr))
        for ts, fr in stream_frames(st.c2s):
            evs.append((ts, -1 - sid, fr))   # C2S 负 sid
    evs.sort(key=lambda e: e[0])
    worlds = {}
    ok, fail = [], []
    prev_s2c = None
    for ts, sid, fr in evs:
        if sid < 0 or not fr:
            continue
        if fr[0] not in range(0x27, 0x31):
            prev_s2c = fr
            continue
        op = fr[0]
        w = worlds.setdefault(sid, World())
        try:
            mode = fr[1]
            hdrlen = 4 if (mode & 2) else 1
            outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
            if outlen > 0xFFFF:
                raise DecodeError('outlen big')
            out = decompress(fr[1:], outlen, w)
            if op == 0x29:
                ptag = None
                if prev_s2c and prev_s2c[0] == 0x14 and len(prev_s2c) >= 12:
                    ptag = struct.unpack_from('<H', prev_s2c, 10)[0]
                ok.append(dict(ts=ts, sid=sid, outlen=outlen, ptag=ptag,
                               plain=bytes(out), raw_head=bytes(fr[:16])))
        except DecodeError as e:
            if op == 0x29:
                fail.append(dict(ts=ts, sid=sid, err=str(e), partial=bytes(getattr(e, 'partial', b''))))
        prev_s2c = fr
    return ok, fail

ALLOK = {}
for name in ['c.pcapng', 'k.pcapng', 'l.pcapng', '登陆.pcapng']:
    ok, fail = export(os.path.join(OFFICIAL, name))
    ALLOK[name] = ok
    print('%s: ok=%d fail=%d' % (name, len(ok), len(fail)))
    with open(os.path.join(OUT, 'op29_ctx_%s.pkl' % name.split('.')[0]), 'wb') as f:
        pickle.dump((ok, fail), f)

# 明文头 16B 多样性（跨 4 包）
print()
print('=== 明文 [0:16] 分布 ===')
from collections import Counter
c = Counter()
for name, ok in ALLOK.items():
    for rec in ok:
        c[bytes(rec['plain'][:16])] += 1
for head, n in c.most_common(12):
    print('%-3d × %s' % (n, head.hex(' ')))

# outlen 分布
print()
print('=== outlen 分布 ===')
c2 = Counter()
for name, ok in ALLOK.items():
    for rec in ok:
        c2[rec['outlen']] += 1
print(sorted(c2.items()))
