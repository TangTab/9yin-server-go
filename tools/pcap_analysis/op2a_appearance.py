# -*- coding: utf-8 -*-
"""0x2a 装备外观帧挖掘：k.pcapng 全部 0x2a（0x46 明文 + 0x47 共享池解码）+
邻接上下文 + 跨帧对齐 diff → 语法。"""
import sys, os, struct
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, r'E:\Workspace\project\game\9yin-go-server1\tools\op28')
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

DIR = r'E:\Workspace\project\game\adata\official\k.pcapng'
evs = []
for sid, st in enumerate(build_streams(DIR)):
    for ts, fr in stream_frames(st.s2c):
        evs.append((ts, sid, fr))
evs.sort(key=lambda e: e[0])
t0 = evs[0][0]

# 流内共享池解码全族（拿到 0x2a 的 0x47 明文）
worlds = {}
plain47 = {}   # idx -> (op, outlen, plain)
for i, (ts, sid, fr) in enumerate(evs):
    if not fr or fr[0] not in range(0x27, 0x31):
        continue
    op, mode = fr[0], fr[1]
    if (mode & 1) == 0:
        continue  # 0x46 类后面单独取
    w = worlds.setdefault(sid, World())
    try:
        hdrlen = 4 if (mode & 2) else 1
        outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        if outlen > 0xFFFF:
            raise DecodeError('big')
        out = decompress(fr[1:], outlen, w)
        if op == 0x2a:
            plain47[i] = (outlen, bytes(out))
    except DecodeError:
        pass

# 全部 0x2a 帧：明文体统一收集
items = []   # (idx, ts, sid, kind, body)
for i, (ts, sid, fr) in enumerate(evs):
    if not fr or fr[0] != 0x2a or len(fr) < 2:
        continue
    mode = fr[1]
    if mode == 0x46:
        hdrlen = 4
        outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        body = bytes(fr[2 + hdrlen + 4:2 + hdrlen + 4 + outlen])
        items.append((i, ts, sid, '46', body, outlen))
    elif i in plain47:
        outlen, p = plain47[i]
        items.append((i, ts, sid, '47', p, outlen))

print('0x2a 总数=%d（0x46×%d + 0x47 已解×%d）' % (
    len(items), sum(1 for x in items if x[3] == '46'), sum(1 for x in items if x[3] == '47')))

print()
print('=== ① 全部 0x46 明文帧（头部 0x30 + 串摘要）===')
for i, ts, sid, kind, body, outlen in items:
    if kind != '46':
        continue
    import re
    strs = re.findall(rb'[ -~]{4,}', body)
    print('[%5d] +%.1fs len=%d 头=%s | %s' % (i, (ts - t0) / 1e6, outlen,
          body[:20].hex(' '), b' '.join(strs[:3]).decode('ascii', 'replace')[:60]))

print()
print('=== ② 第一帧全文 hex（走查用）===')
first = next(x for x in items if x[3] == '46')
body = first[4]
for off in range(0, min(len(body), 0x140), 16):
    chunk = body[off:off + 16]
    print('%04x: %-47s %s' % (off, ' '.join('%02x' % b for b in chunk),
          ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)))

print()
print('=== ③ 邻接上下文（每 0x46 帧前 2 帧）===')
seen_ts = set()
for i, ts, sid, kind, body, outlen in items:
    if kind != '46':
        continue
    ctx = []
    for j in range(max(0, i - 2), i):
        ts2, sid2, fr2 = evs[j]
        if fr2:
            ctx.append('op%02X len=%d %s' % (fr2[0], len(fr2), fr2[:16].hex()))
    print('[%d] <- %s' % (i, ' || '.join(ctx)))

print()
print('=== ④ 0x47 已解 0x2a 样例（前 4 帧）对照 ===')
n = 0
for i, ts, sid, kind, body, outlen in items:
    if kind != '47':
        continue
    print('[%d] outlen=%d 头=%s' % (i, outlen, body[:24].hex(' ')))
    n += 1
    if n >= 4:
        break
