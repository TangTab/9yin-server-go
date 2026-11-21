# -*- coding: utf-8 -*-
"""记录挂载计划提取器：从 enter_golden.bin 提取 0x14 挂载帧清单
（顺序/sub/记录名），按名映射生成 internal/entergen/mountplan.go。

golden 是 Blue Dragon 方言（344 schema），官方 447 表按名对映——
生成器运行时 MustRecordIndexByName 在当前方言解析 idx。

用法: python -I tools/gen_mount_plan.py
"""
import os
import struct

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BIN = os.path.join(ROOT, 'internal', 'fssoloworld', 'enter_golden.bin')
GOLDEN_TABLE = os.path.join(ROOT, 'internal', 'clientdata', 'record_table_golden.bin')
OUT = os.path.join(ROOT, 'internal', 'entergen', 'mountplan.go')


def pack_frames(path):
    raw = open(path, 'rb').read()
    out, off = [], 0
    while off + 4 <= len(raw):
        ln = struct.unpack_from('<I', raw, off)[0]
        if ln <= 0 or off + 4 + ln > len(raw):
            break
        out.append(raw[off+4:off+4+ln])
        off += 4 + ln
    return out


def schema_names(path):
    d = open(path, 'rb').read()
    cnt = struct.unpack_from('<H', d, 1)[0]
    off, names = 3, {}
    for i in range(cnt):
        z = d.find(b'\x00', off)
        names[i] = d[off:z].decode('gbk', 'replace')
        ncols = struct.unpack_from('<H', d, z+1)[0]
        off = z + 3 + ncols
    return names


def main():
    gnames = schema_names(GOLDEN_TABLE)
    rows = []
    for fr in pack_frames(BIN):
        if fr[0] != 0x14 or len(fr) != 12:
            continue
        sub = fr[1]
        rec = struct.unpack_from('<H', fr, 10)[0]
        # 后一帧 op 定行推通道（11=纯数字列 / 29=字符串列批量 / 其它=挂载即止）
        rows.append((gnames.get(rec, '?%d' % rec), sub))
    lines = ['package entergen', '',
             '// MountPlan 官方登录挂载计划（enter_golden.bin 0x14 帧提取，',
             '// tools/gen_mount_plan.py 生成——顺序/sub/记录名与官方抓包一致；',
             '// idx 运行时按当前方言表名解析）。sub：0=玩家域（obj/own=玩家），',
             '// 2=全局域（obj/own=0）。',
             '',
             'var MountPlan = []MountSpec{']
    for name, sub in rows:
        lines.append('\t{Name: %s, Sub: %d},' % (go_quote(name), sub))
    lines.append('}')
    with open(OUT, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines) + '\n')
    print('written', OUT, len(rows), 'mounts')


def go_quote(s):
    return '"' + s.replace('\\', '\\\\').replace('"', '\\"') + '"'


if __name__ == '__main__':
    main()
