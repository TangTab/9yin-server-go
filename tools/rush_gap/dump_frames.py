# -*- coding: utf-8 -*-
"""frames pkl 按时间窗 dump 帧字节（16 字节 hexdump，附字符串列）。

用法：
  python -I dump_frames.py <frames.pkl> <t1秒> <t2秒> [op列表 0x28,0x21] [s2c|c2s]

时间基准与 pcap_frames 输出一致（首帧=0）。
"""
import pickle
import sys


def hexdump(fr, limit=None):
    n = len(fr) if limit is None else min(len(fr), limit)
    lines = []
    for off in range(0, n, 16):
        chunk = fr[off:off + 16]
        hexs = ' '.join('%02x' % b for b in chunk)
        asc = ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)
        lines.append('%04x: %-48s %s' % (off, hexs, asc))
    if limit is not None and len(fr) > limit:
        lines.append('... (%d more bytes)' % (len(fr) - limit))
    return '\n'.join(lines)


def main():
    if len(sys.argv) < 4:
        print(__doc__)
        return
    data = pickle.load(open(sys.argv[1], 'rb'))
    t_lo, t_hi = float(sys.argv[2]), float(sys.argv[3])
    ops = set(int(x, 0) for x in sys.argv[4].split(',')) if len(sys.argv) > 4 and sys.argv[4] else None
    d_flt = sys.argv[5] if len(sys.argv) > 5 else None
    rows = []
    for st in data['streams']:
        if d_flt and st['dir'] != d_flt:
            continue
        for ts, fr in st['frames']:
            rel = ts / 1e6
            if fr and t_lo <= rel <= t_hi and (ops is None or fr[0] in ops):
                rows.append((rel, st['dir'], fr))
    rows.sort(key=lambda r: r[0])
    for rel, d, fr in rows:
        print('=== %10.4f %s op=%02x len=%d ===' % (rel, d.upper(), fr[0], len(fr)))
        print(hexdump(fr))


if __name__ == '__main__':
    main()
