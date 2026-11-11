# -*- coding: utf-8 -*-
"""终局盘点：c.pcapng 登录簇 0x14→0x29 完整预载清单 + t=1 帧全 hex 判读。"""
import sys, os, struct, pickle
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'op28'))
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

RT = r'E:\Workspace\project\game\9yin-go-server1\internal\clientdata\official_record_table.bin'
d = open(RT, 'rb').read()
cnt = struct.unpack_from('<H', d, 1)[0]
off = 3
REC = {}
for i in range(cnt):
    z = d.find(b'\x00', off)
    ncols = struct.unpack_from('<H', d, z + 1)[0]
    REC[i] = (d[off:z].decode('gbk', 'replace'), d[z + 3:z + 3 + ncols])
    off = z + 3 + ncols

DIR = r'E:\Workspace\project\game\adata\official\c.pcapng'
streams = build_streams(DIR)
evs = []
for sid, st in enumerate(streams):
    for ts, fr in stream_frames(st.s2c):
        evs.append((ts, sid, fr))
evs.sort(key=lambda e: e[0])
t0 = evs[0][0]

# 时序解码（同前），同时记录 0x14→0x29/0x28/0x11 邻接
worlds = {}
okmap = {}   # frame_idx -> outlen/plain
for i, (ts, sid, fr) in enumerate(evs):
    if not fr or fr[0] not in range(0x27, 0x31):
        continue
    w = worlds.setdefault(sid, World())
    try:
        mode = fr[1]
        hdrlen = 4 if (mode & 2) else 1
        outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        if outlen > 0xFFFF:
            raise DecodeError('big')
        out = decompress(fr[1:], outlen, w)
        if fr[0] == 0x29:
            okmap[i] = (outlen, bytes(out))
    except DecodeError:
        pass

print('=== c.pcapng 登录预载通道清单（0x14 宣告 → 载体帧）===')
n29 = 0
for i, (ts, sid, fr) in enumerate(evs[:260]):
    if not fr:
        continue
    if fr[0] == 0x14 and len(fr) >= 12:
        # 紧跟的载体
        nxt = evs[i + 1][2] if i + 1 < len(evs) else None
        if nxt and nxt[0] in (0x29, 0x28):
            tag = struct.unpack_from('<H', fr, 10)[0]
            sub = fr[1]
            name = REC.get(tag, ('?',))[0] if tag < len(REC) else '?'
            carrier = 'op%02X' % nxt[0]
            if nxt[0] == 0x29 and (i + 1) in okmap:
                outlen, plain = okmap[i + 1]
                rec2 = struct.unpack_from('<H', plain, 9)[0] if plain[0] == 0 else -1
                rows = struct.unpack_from('<H', plain, 13)[0]
                bm = '位图' if rows & 0x8000 else '行'
                rows &= 0x7fff
                print('[%4d] +%.2fs 0x14 sub%d tag=%-3d %-24s → %s 解码OK rec=%d rows=%d outlen=%d' % (
                    i, (ts - t0) / 1e6, sub, tag, name, carrier, rec2, rows, outlen))
            else:
                print('[%4d] +%.2fs 0x14 sub%d tag=%-3d %-24s → %s (未解)' % (
                    i, (ts - t0) / 1e6, sub, tag, name, carrier))
    elif fr[0] == 0x29:
        n29 += 1

# t=1 帧全部
print()
print('=== t=1 形态帧（全部）===')
for i, (outlen, plain) in sorted(okmap.items()):
    if plain[0] == 1:
        print('--- [%d] outlen=%d ---' % (i, outlen))
        for off in range(0, min(len(plain), 0x90), 16):
            chunk = plain[off:off + 16]
            print('%04x: %-47s %s' % (off, ' '.join('%02x' % b for b in chunk),
                  ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)))
