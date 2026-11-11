# -*- coding: utf-8 -*-
"""池种子溯源：fail 帧 miss idx → 反查注册源（表/早期帧/0x29 自身）。
哈希: idx = ((v>>12) ^ v) & 0xfff, v=任意 4B 对齐窗口的 u32。"""
import sys, os, struct
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'op28'))
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

DIR = r'E:\Workspace\project\game\adata\official\c.pcapng'

# 1) 收集 c.pcapng 流帧
streams = build_streams(DIR)
evs = []
for sid, st in enumerate(streams):
    for ts, fr in stream_frames(st.s2c):
        evs.append((ts, sid, fr))
evs.sort(key=lambda e: e[0])

# 2) 时序走 0x27-0x30 族，抓 0x29 首 fail 的 miss idx 集
worlds = {}
miss_samples = []   # (frame_idx, miss_idx, miss_len, out_pos)
fail_frames = []
for i, (ts, sid, fr) in enumerate(evs):
    if not fr or fr[0] not in range(0x27, 0x31):
        continue
    w = worlds.setdefault(sid, World())
    trace = []
    try:
        mode = fr[1]
        hdrlen = 4 if (mode & 2) else 1
        outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        if outlen > 0xFFFF:
            raise DecodeError('big')
        decompress(fr[1:], outlen, w, trace)
    except DecodeError as e:
        if fr[0] == 0x29:
            for t in trace:
                if t[0] == 'POOLMISS':
                    miss_samples.append((i, t[1], t[2], t[3]))
            fail_frames.append(i)

print('0x29 fail 帧数=%d, miss 样本=%d' % (len(fail_frames), len(miss_samples)))
from collections import Counter
idx_c = Counter(m[1] for m in miss_samples)
print('miss idx Top20:', [(hex(k), v) for k, v in idx_c.most_common(20)])

# 3) 候选源字节
sources = {}
# a) 0x09/0x0A 表帧原始字节
for ts, sid, fr in evs:
    if fr and fr[0] in (0x09, 0x0A):
        sources['op%02X_raw' % fr[0]] = bytes(fr)
        break
# b) op32 票据
for ts, sid, fr in evs:
    if fr and fr[0] == 0x20:
        sources['op32_raw'] = bytes(fr)
        break
# c) 前置小帧 (op10/11/14/1E) 合并
pre = bytearray()
for ts, sid, fr in evs[:124]:
    if fr and fr[0] in (0x10, 0x11, 0x14, 0x1E, 0x0B):
        pre += bytes(fr)
sources['pre_small_frames'] = bytes(pre)
# d) 全部 S2C 帧原始（0x123 前）合并
allraw = bytearray()
for ts, sid, fr in evs[:124]:
    if fr:
        allraw += bytes(fr)
sources['all_s2c_before_first29'] = bytes(allraw)

def idx_stream(data):
    """全部 4B 滑窗（步进1）的 idx → 位置列表"""
    m = {}
    for pos in range(len(data) - 3):
        v = struct.unpack_from('<I', data, pos)[0]
        idx = ((v >> 12) ^ v) & 0xfff
        m.setdefault(idx, []).append(pos)
    return m

print()
for m in miss_samples[:0]:
    pass
# 对每个 miss idx：各源是否含注册窗口
src_idx = {}
for name, data in sources.items():
    src_idx[name] = idx_stream(data)

probe = idx_c.most_common(12)
print('%-8s %-28s %s' % ('miss_idx', '命中源(次数)', '源内首位置示例'))
for idx, n in probe:
    hits = []
    for name, m in src_idx.items():
        if idx in m:
            hits.append('%s×%d@%#x' % (name, len(m[idx]), m[idx][0]))
    print('%-#6x(n=%-3d) %s' % (idx, n, ' | '.join(hits) if hits else '(全源未命中)'))
