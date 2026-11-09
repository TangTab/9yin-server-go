# -*- coding: utf-8 -*-
"""普查 8 个 pcapng 的 TCP 连接拓扑：谁连谁、端口、字节数、包数、持续时间。"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams

DIR = r'E:\Workspace\project\game\adata\official'

for name in sorted(os.listdir(DIR)):
    if not name.endswith('.pcapng'):
        continue
    path = os.path.join(DIR, name)
    streams = build_streams(path)
    print('=' * 78)
    print(name, '- %d TCP streams' % len(streams))
    for st in sorted(streams, key=lambda s: min([t for t, _ in s.c2s] + [t for t, _ in s.s2c] or [0])):
        t0 = min([t for t, _ in st.c2s] + [t for t, _ in st.s2c])
        t1 = max([t for t, _ in st.c2s] + [t for t, _ in st.s2c])
        c2s_bytes = sum(len(p) for _, p in st.c2s)
        s2c_bytes = sum(len(p) for _, p in st.s2c)
        print('  %-21s -> %-21s c2s %6dB/%4dpk  s2c %7dB/%4dpk  %6.1fs'
              % (st.client, st.server, c2s_bytes, len(st.c2s), s2c_bytes, len(st.s2c), (t1 - t0) / 1e6))
        # 首包预览
        if st.s2c:
            first = st.s2c[0][1][:48]
            print('      s2c head: %s' % first[:24].hex())
        if st.c2s:
            first = st.c2s[0][1][:48]
            print('      c2s head: %s' % first[:24].hex())
