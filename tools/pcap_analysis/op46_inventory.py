# -*- coding: utf-8 -*-
"""盘点 0x27-0x30 族 mode 分布：0x46/0x44 等 memcpy 明文路径帧的 op/包/长度，
并核对 0x46 帧与同 outlen 的 0x47 帧明文是否同构。"""
import sys, os, struct, glob
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames
sys.path.insert(0, r'E:\Workspace\project\game\9yin-go-server1\tools\op28')
from lzdec import World, decompress, DecodeError
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

OFFICIAL = r'E:\Workspace\project\game\adata\official'

for path in sorted(glob.glob(os.path.join(OFFICIAL, '*.pcapng'))):
    name = os.path.basename(path)
    evs = []
    for st in build_streams(path):
        for ts, fr in stream_frames(st.s2c):
            evs.append((ts, fr))
    evs.sort(key=lambda e: e[0])
    fam = {}
    plain46 = []   # (i, op, outlen, body_bytes)
    for i, (ts, fr) in enumerate(evs):
        if not fr or fr[0] not in range(0x27, 0x31) or len(fr) < 2:
            continue
        op, mode = fr[0], fr[1]
        fam[mode] = fam.get(mode, 0) + 1
        # memcpy 路径: bit0=0
        if (mode & 1) == 0:
            try:
                hdrlen = 4 if (mode & 2) else 1
                outlen = struct.unpack_from('<I', fr, 2 + hdrlen)[0] if len(fr) >= 2 + hdrlen + 4 else -1
            except struct.error:
                outlen = -1
            # memcpy: 明文直接在 outlen 字段后
            body = bytes(fr[2 + hdrlen + 4:2 + hdrlen + 4 + (outlen if 0 < outlen < 8192 else 0)])
            plain46.append((i, op, mode, outlen, len(fr), body))
    if fam:
        print('%-12s 族mode分布: %s' % (name, {hex(k): v for k, v in sorted(fam.items())}))
        for i, op, mode, outlen, flen, body in plain46:
            print('    [%d] op=%#x mode=%#x 帧长=%d outlen=%s 体前24B=%s' % (
                i, op, mode, flen, outlen, body[:24].hex(' ') if body else '(超长/未取)'))
