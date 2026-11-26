"""stateconst.py —— 从 minidump 导出客户端玩家状态机的状态注册表 / VM 常量区字符串。

客户端的玩家状态机（motion_func）逻辑被 VMProtect 虚拟化，读不出来；但运行期它把「状态索引 →
状态名」注册在 FxProtect.dll 的堆上，状态名 / 属性名 / 日志格式串（player_state_<状态> PLAYER_INPUT_<输入>）
也以明文留在同模块的 VM 常量区。两种导出：

  python -I stateconst.py <dmp> <out.txt>                  状态注册表（默认）：索引 → 状态名 → 入口 → 标志
  python -I stateconst.py <dmp> <out.txt> --strings        常量区字符串（可配 --lo / --hi / --min / --re）
  可选：--module FxProtect（模块名子串）

注册表每项 0x28 字节 {u64 索引; char* 状态名; void* 入口; u32 标志A; u32 标志B; u64 扩展}，按索引顺序连续排列。
m24 子码 71 的处理函数把施法者切到 123 号状态 player_flash_attack（已在 fxgamelogic 反汇编里核对），
124 号是 player_flash_stop。纯标准库。"""
import argparse
import os
import re
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式下脚本目录不在 sys.path
sys.stdout.reconfigure(encoding='utf-8', errors='replace')      # 控制台非 UTF-8 时避免打印失败

from mdmp import Dmp

STRIDE = 0x28


def short_name(path):
    return path.replace('/', '\\').split('\\')[-1]


def state_table(dmp, module='FxProtect'):
    """(模块, 表起偏移, 表止偏移, [注册项,…])：注册项 = (偏移, 索引, 状态名, 入口 VA, 标志A, 标志B, 扩展)。
    镜像里 8 字节对齐处凡满足「首 qword 是小整数、次 qword 指向模块内的可打印短串、再次 qword 落在模块内」
    的位置算候选，取按步长 0x28 连续的最长一段。"""
    m, image = dmp.module_image(module)
    base, cand = m[0], []
    for o in range(0, len(image) - STRIDE, 8):
        idx, name, entry = struct.unpack_from('<QQQ', image, o)
        if idx >= 0x400 or not base < name < base + len(image) or not base < entry < base + len(image):
            continue
        s = image[name - base:name - base + 64]
        end = s.find(b'\0')
        s = s[:end if end >= 0 else 64]
        if 2 <= len(s) <= 48 and all(32 <= c < 127 for c in s):
            cand.append(o)
    runs, cur = [], cand[:1]
    for a, b in zip(cand, cand[1:]):
        if b - a == STRIDE:
            cur.append(b)
        else:
            runs.append(cur)
            cur = [b]
    runs.append(cur)
    best = max(runs, key=len)
    rows = []
    for o in best:
        idx, name, entry, fa, fb, ext = struct.unpack_from('<QQQIIQ', image, o)
        s = image[name - base:name - base + 64]
        rows.append((o, idx, s[:s.find(b'\0')].decode('latin1'), entry, fa, fb, ext))
    return m, best[0], best[-1] + STRIDE, rows


def export_strings(dmp, module='FxProtect', lo=0x1ad6b8, hi=0x1b9380, minlen=3, rx=None):
    """(模块, [常量区串行,…])，窗口是相对模块基址的偏移；默认窗口即外服 fxgame.DMP 里
    player_state_* 日志格式串所在的那一段。"""
    m, image = dmp.module_image(module)
    rows = []
    for mt in re.finditer(rb'[\x20-\x7e]{%d,}' % minlen, image[lo:hi]):
        s = mt.group(0).decode('latin1')
        if rx is None or rx.search(s):
            rows.append('   +0x%08x  %s' % (lo + mt.start(), s))
    return m, rows


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('dmp')
    ap.add_argument('out')
    ap.add_argument('--module', default='FxProtect', help='模块名子串')
    ap.add_argument('--strings', action='store_true', help='导出常量区字符串而不是状态注册表')
    ap.add_argument('--lo', type=lambda s: int(s, 16), default=0x1ad6b8, help='--strings：窗口起（相对模块基址的偏移）')
    ap.add_argument('--hi', type=lambda s: int(s, 16), default=0x1b9380, help='--strings：窗口止')
    ap.add_argument('--min', type=int, default=3, help='--strings：最短串长')
    ap.add_argument('--re', help='--strings：只保留匹配该正则的串')
    ns = ap.parse_args()
    dmp = Dmp(ns.dmp)
    if ns.strings:
        m, rows = export_strings(dmp, ns.module, ns.lo, ns.hi, ns.min, re.compile(ns.re) if ns.re else None)
        head = '%s (base %#x) 内 VM 常量区 [+%#x, +%#x)，按偏移排列；状态名→属性名→日志格式串成组' % (
            short_name(m[4]), m[0], ns.lo, ns.hi)
    else:
        m, lo, hi, table = state_table(dmp, ns.module)
        head = ('%s (base %#x) 玩家状态机状态注册表：%d 项，[+%#x, +%#x)，步长 %#x；'
                '每项 {u64 索引; char* 状态名; void* 入口; u32 标志A; u32 标志B; u64 扩展}' % (
                    short_name(m[4]), m[0], len(table), lo, hi, STRIDE))
        rows = ['%4d  +0x%06x  %-24s entry=%#x  flags=%#010x/%#010x  ext=%d' % (
            idx, o, name, entry, fa, fb, ext) for o, idx, name, entry, fa, fb, ext in table]
    with open(ns.out, 'w', encoding='utf-8', newline='\n') as f:
        f.write('\n'.join([head] + rows) + '\n')
    print('%s：导出 %d 条 -> %s' % (head, len(rows), ns.out))


if __name__ == '__main__':
    main()
