# -*- coding: utf-8 -*-
"""从新版服务端 exe 的抓包中导出官方 0x09 属性表。

数据源:F:\\Game\\Online\\9yin\\9yin_server1\\build\\9yin-game-native-menu.exe 自抓的
s2c 捕获(该 exe 登录链下发的 0x09 帧即其线上方言,2026-11-06 反汇编 + 三方比对
确认与 Java 参考服 official_property_table.txt 逐项一致)。

用法(建议 -I 隔离运行):
  python -I tools/export_official_table.py <s2c.bin> <out.txt> [--check java_table.txt]

输出:index\tName\tType 三列 ASCII TSV,2370 行,行号即帧内位置(0x09 只带
名字+线型,客户端按位置绑定索引)。自校验:重组帧与捕获帧逐字节一致 + 锚点断言。
"""
import struct
import sys

KEY = bytes([0xCC, 0x3B, 0x6D, 0xFA])

ANCHORS = {
    34: ("NpcType", 3),
    36: ("Prop36", 8),
    88: ("InParry", 3),
    100: ("LastObject", 9),
    178: ("Name", 8),
    390: ("SpeedRatio", 3),
    414: ("LastObject", 9),
    415: ("AutoSelectObject", 9),
    463: ("LogicState", 1),
    465: ("HP", 3),
    467: ("SP", 3),
    470: ("MPRatio", 3),
    471: ("InParry", 3),
    474: ("InManualParry", 3),
    475: ("ParryValue", 3),
    580: ("MaxSP", 3),
    583: ("MaxSPAdd", 3),
    629: ("BufferInfo1", 7),
    652: ("BufferInfo24", 7),
}


def decode_frames(buf):
    """EE EE 切帧 → 去转义(EE 00→EE) → 帧内偏移对齐 KEY 异或。"""
    frames, body = [], bytearray()
    i = 0
    while i < len(buf):
        if buf[i] == 0xEE and i + 1 < len(buf):
            nxt = buf[i + 1]
            if nxt == 0x00:
                body.append(0xEE)
                i += 2
            elif nxt == 0xEE:
                frames.append(bytes(b ^ KEY[j % 4] for j, b in enumerate(body)))
                body = bytearray()
                i += 2
            else:
                break
        else:
            body.append(buf[i])
            i += 1
    return frames


def parse_09(frame):
    """0x09 | u16 count | (name\\0 u8 type)* → [(name, type)]"""
    count = struct.unpack_from('<H', frame, 1)[0]
    entries, off = [], 3
    for _ in range(count):
        end = frame.index(b'\x00', off)
        entries.append((frame[off:end].decode('latin-1'), frame[end + 1]))
        off = end + 2
    return count, entries, off


def main():
    args = [a for a in sys.argv[1:] if not a.startswith('--')]
    if len(args) < 2:
        print(__doc__)
        sys.exit(2)
    s2c_path, out_path = args[0], args[1]
    check_path = None
    for i, a in enumerate(sys.argv):
        if a == '--check':
            check_path = sys.argv[i + 1]

    frames = decode_frames(open(s2c_path, 'rb').read())
    table_frames = [f for f in frames if f and f[0] == 0x09]
    if not table_frames:
        print('no 0x09 frame in capture')
        sys.exit(1)
    count, entries, consumed = parse_09(table_frames[0])
    for other in table_frames[1:]:
        c2, e2, _ = parse_09(other)
        assert (c2, e2) == (count, entries), 'multiple 0x09 frames disagree'

    assert count == len(entries) == 2370, f'entry count {count} != 2370'
    for idx, (name, typ) in ANCHORS.items():
        got = entries[idx]
        assert got == (name, typ), f'anchor {idx}: {got} != {(name, typ)}'

    rebuilt = bytearray(b'\x09')
    rebuilt += struct.pack('<H', count)
    for name, typ in entries:
        rebuilt += name.encode('latin-1') + b'\x00' + bytes([typ])
    assert bytes(rebuilt) == table_frames[0], 'rebuilt frame != captured frame'

    with open(out_path, 'w', encoding='ascii', newline='\n') as fh:
        for i, (name, typ) in enumerate(entries):
            fh.write(f'{i}\t{name}\t{typ}\n')

    print(f'exported {count} entries -> {out_path}')
    if check_path:
        official = [ln.strip().split('\t') for ln in open(check_path, encoding='ascii') if ln.strip()]
        diffs = [(i, entries[i], (official[i][1], int(official[i][2])))
                 for i in range(min(len(entries), len(official)))
                 if entries[i][0] != official[i][1] or entries[i][1] != int(official[i][2])]
        print(f'check vs {check_path}: entries={len(official)} diffs={len(diffs)}')
        for d in diffs[:10]:
            print('  ', d)


if __name__ == '__main__':
    main()
