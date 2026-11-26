"""subtab.py —— 还原客户端 fxgamelogic 的 m24「子码 → 处理函数」表（本地 DLL 文件 / 外服内存镜像通用）。

  python -I subtab.py pe <fxgamelogic.dll> <out.txt>
  python -I subtab.py mem <img.bin> <基址十六进制> <out.txt>        镜像用 mdmp.py save 导出

要点：
  1) 表基址与子码上限从分发器本体读：lea eax,[rcx-1]; mov r10,rdx; cmp eax,<上限>; ja …;
     lea r9,[table]; movsxd rax,ecx; mov r9,[r9+rax*8] —— 合法子码 1..上限+1，槽 = 表基址 + 8 × 子码；
  2) 注册对按「同一寄存器」配对：lea reg,[rip+d1] … mov [rip+d2],reg —— 编译器会在两者中间夹
     别的 lea/mov，不能按相邻配对（曾因此把 61/74/78/79 四个子码的处理函数错位配对）；
  3) 每个处理函数列出其函数体内引用的全部字符串（int3 填充定界），用来判断子码语义；
     空的子码 = 处理函数内没有字符串引用。
各子码的字段含义要以抓包对拍为准，见 docs/combat/寒心恨雪与太极剑舞引导机制.md。"""
import os
import re
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式下脚本目录不在 sys.path
sys.stdout.reconfigure(encoding='utf-8', errors='replace')      # 控制台非 UTF-8 时避免打印失败

from capstone import Cs, CS_ARCH_X86, CS_MODE_64

from mempe import MemPE
from pe import PE
from x64 import RIPABS, STR

DISP = re.compile(rb'\x8d\x41\xff\x4c\x8b\xd2\x83\xf8(.)\x77.\x4c\x8d\x0d(.{4})', re.S)   # 分发器特征，共 18 字节
# 64 位 lea / mov：[REX.W] opcode modrm(mod=00, rm=101) disp32
LEA = re.compile(rb'(?P<rex>[\x48\x4c])?\x8d(?P<modrm>[\x05\x0d\x15\x1d\x25\x2d\x35\x3d])(?P<disp>.{4})', re.S)
MOV = re.compile(rb'(?P<rex>[\x48\x4c])?\x89(?P<modrm>[\x05\x0d\x15\x1d\x25\x2d\x35\x3d])(?P<disp>.{4})', re.S)


def reg_of(m):
    """modrm.reg 字段的寄存器号：lea 的目的 / mov 的源，高 8 号寄存器由 REX.R 扩展。"""
    rex = m.group('rex')
    return ((m.group('modrm')[0] >> 3) & 7) | (8 if rex == b'L' else 0)


def build(img):
    """还原结果文本：首行分发器与表基址，次行命中槽数，其后每槽一行。"""
    out, table, maxcode = [], None, None
    for base, blob in img.code():
        m = DISP.search(blob)
        if m:
            # 分发器：if ((u32)(subcode - 1) > 上限) return false; handler = table[subcode];
            # 所以上限是版本相关的立即数，合法子码 = 1..上限+1，槽地址 = 表基址 + 8 × 子码。
            maxcode, table = m.group(1)[0], base + m.end() + struct.unpack_from('<i', m.group(2))[0]
            out.append('分发器 %#x：子码 1..%#x，表基址 %#x（槽 = 基址 + 8 × 子码）' % (base + m.start(), maxcode + 1, table))
            break
    if table is None:
        sys.exit('dispatcher not found')
    slots = {}
    for base, blob in img.code():
        evs = []
        for kind, rx in (('lea', LEA), ('mov', MOV)):
            evs += [(m.start(), kind, reg_of(m), m) for m in rx.finditer(blob)]
        last = {}
        for _, kind, reg, m in sorted(evs, key=lambda e: e[0]):
            target = base + m.end() + struct.unpack_from('<i', m.group('disp'))[0]
            if kind == 'lea':
                last[reg] = target
            elif reg in last and table + 8 <= target < table + 8 * (maxcode + 2):
                slots[target] = last[reg]
    out.append('命中 %d 槽' % len(slots))
    md = Cs(CS_ARCH_X86, CS_MODE_64)
    for i in range(1, maxcode + 2):
        if table + 8 * i not in slots:
            continue
        real = img.resolve(slots[table + 8 * i])
        b, e = img.func_by_padding(real)
        strs = []
        for ins in md.disasm(img.read(b, e - b), b):
            m = RIPABS.search(ins.op_str) if ins.mnemonic == 'lea' else None
            if not m:
                continue
            t = ins.address + ins.size + (int(m.group(2), 16) if m.group(1) == '+' else -int(m.group(2), 16))
            q = STR.match(img.read(t, 100))
            if q and q.group(0) not in strs:
                strs.append(q.group(0))
        out.append('sub %3d  func=%#x  %s' % (i, real, ' | '.join(s.decode('latin1')[:52] for s in strs)))
    return '\n'.join(out)


def main():
    argv = sys.argv[1:]
    if len(argv) == 3 and argv[0] == 'pe':
        img = PE(argv[1])
    elif len(argv) == 4 and argv[0] == 'mem':
        img = MemPE(argv[1], int(argv[2], 16))
    else:
        sys.exit(__doc__)
    text = build(img)
    with open(argv[-1], 'w', encoding='utf-8') as f:
        f.write(text + '\n')
    print(text)


if __name__ == '__main__':
    main()
