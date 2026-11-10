# -*- coding: utf-8 -*-
"""op32 会话票据三样本验证：k.pcapng 全登录流 vs c.pcapng 既有定版布局（§十二）。

验证点：
  1. k 是否有 op32（帧数/长度/时序：op04 后 60ms 级）
  2. 175B 打包字符串块结构（4 字段 NUL 分隔、头截断）
  3. @744 账号名 / @811..821 密钥（02 + 905F+8B）/ @824 4 串
  4. C2S op04 密钥 K 是否 == op32@812 的 10 字节
  5. 三样本（k/c/login2 如可得）逐字节分类：恒定区 vs 可变区
"""
import struct
import sys

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames


def analyze(path, label):
    print('=' * 20, label, path.split('\\')[-1].split('/')[-1], '=' * 20)
    streams = build_streams(path)
    for st in streams:
        if not st.s2c or not st.c2s:
            continue
        s2c = stream_frames(st.s2c)
        c2s = stream_frames(st.c2s)
        has10 = sum(1 for _, f in s2c if f and f[0] == 0x10)
        if has10 == 0:
            continue
        print('stream %s -> %s  s2c=%d c2s=%d' % (st.client, st.server, len(s2c), len(c2s)))
        t0 = min(s2c[0][0] if s2c else 0, c2s[0][0] if c2s else 0)
        op32s = [(ts, fno, f) for fno, (ts, f) in enumerate(s2c) if f and f[0] == 0x32]
        op04s = [(ts, fno, f) for fno, (ts, f) in enumerate(c2s) if f and f[0] == 0x04]
        print('op32 帧 %d 个: %s' % (len(op32s), ['%d:%dB@t=%.3f' % (fno, len(f), (ts - t0) / 1e6) for ts, fno, f in op32s]))
        print('C2S op04 帧 %d 个: %s' % (len(op04s), ['#%d:%dB@t=%.3f' % (fno, len(f), (ts - t0) / 1e6) for ts, fno, f in op04s]))
        for ts, fno, f in op32s:
            out = r'G:\pojie\ai\9yin-server-go\tmp\op32_%s.bin' % label
            open(out, 'wb').write(f)
            print('saved %s (%d B)' % (out, len(f)))
            # 结构检查
            if len(f) >= 1026:
                check_layout(f, label)
        # op04 密钥 vs op32@812
        for ts, fno, f in op04s:
            if len(f) >= 11:
                print('op04 #%d key=%s (len=%d)' % (fno, f[1:11].hex().upper(), len(f)))
        for ts, fno, f in op32s:
            if len(f) >= 822:
                print('op32 #%d @811..821=%s  @812..821=%s' % (fno, f[811:822].hex().upper(), f[812:822].hex().upper()))


def check_layout(f, label):
    print('--- 布局检查 (%s, %dB) ---' % (label, len(f)))
    block = f[1:176]
    parts = block.split(b'\x00')
    for i, p in enumerate(parts):
        try:
            s = p.decode('gbk')
        except Exception:
            s = repr(p)
        print('  字段%d (%dB): %r' % (i + 1, len(p), s))
    print('  块合计=%dB (字段+%d 分隔NUL)' % (sum(len(p) for p in parts) + len(parts) - 1, len(parts) - 1))
    # 账号 @744
    acct = f[744:744 + 16].split(b'\x00')[0]
    print('  @744 账号: %r' % acct.decode('gbk', 'replace'))
    print('  @811: %02X  密钥@812..821: %s' % (f[811], f[812:822].hex().upper()))
    s4 = f[824:824 + 6].split(b'\x00')[0]
    print('  @824 4串: %r' % s4.decode('gbk', 'replace'))
    s12 = f[472:472 + 14].split(b'\x00')[0]
    print('  @472 12串: %r' % s12.decode('gbk', 'replace'))
    # 64 位指针样貌：扫 u64 值 0x00007f_xxxx_xxxx
    ptrs = []
    for off in range(176, 1026 - 7, 8):
        v = struct.unpack_from('<Q', f, off)[0]
        if 0x7F0000000000 <= v <= 0x7FFFFFFFFFFF:
            ptrs.append((off, hex(v)))
    print('  0x7f.. 指针样貌 u64: %s' % (ptrs[:8],))
    # u16 恒量（§12.3: c=0x01FF×6 / login2=0x020A×6）
    from collections import Counter
    u16s = Counter()
    for off in range(176, 1026 - 1, 2):
        u16s[struct.unpack_from('<H', f, off)[0]] += 1
    print('  u16 高频: %s' % u16s.most_common(4))


if __name__ == '__main__':
    analyze(r'F:/Game/Online/9yin/9yin-server-go/artifacts/k.pcapng', 'k')
    analyze(r'F:/Game/Online/9yin/9yin-server-go/artifacts/c.pcapng', 'c')
