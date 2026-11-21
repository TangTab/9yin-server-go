# -*- coding: utf-8 -*-
"""带符号注释的反汇编：解析 thunk 目标 → PDB 符号名，标注 call 与 opcode 写入。"""
import bisect
import sys
import pefile
from capstone import *

DLL = r"E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsSoloWorld.dll"
SYMS = r"C:\Users\Lizy\AppData\Local\Temp\FsSoloWorld_syms.tsv"
pe = pefile.PE(DLL, fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
sections = [(s.VirtualAddress, s.Misc_VirtualSize, s.Name.decode(errors='replace').strip('\x00'))
            for s in pe.sections]

# 符号表: va -> name（取每段排序数组做最近邻）
va_list, va_names = [], []
rows = []
with open(SYMS, encoding='utf-8') as f:
    for line in f:
        parts = line.rstrip('\n').split('\t')
        if len(parts) < 3:
            continue
        seg, off, nm = int(parts[0]), int(parts[1], 16), parts[2]
        if seg - 1 < len(sections):
            va = base + sections[seg - 1][0] + off
            rows.append((va, nm))
rows.sort()
va_list = [r[0] for r in rows]
va_names = [r[1] for r in rows]


def sym(va):
    i = bisect.bisect_right(va_list, va) - 1
    if i >= 0 and va - va_list[i] < 0x800:
        return "%s+%x" % (va_names[i], va - va_list[i])
    return ""


md = Cs(CS_ARCH_X86, CS_MODE_32)


def read(va, n):
    return pe.get_data(va - base, n)


def resolve_thunk(va, depth=0):
    if depth > 3:
        return va
    code = read(va, 8)
    for ins in md.disasm(code, va):
        if ins.mnemonic == 'jmp' and ins.op_str.startswith('0x'):
            return resolve_thunk(int(ins.op_str, 16), depth + 1)
        break
    return va


def undecorate(nm):
    # 简易 c++ 名字修饰还原：?func@Class@@... → Class::func
    if not nm.startswith('?'):
        return nm
    body = nm[1:]
    if '@@' in body:
        head = body.split('@@')[0]
        parts = head.split('@')
        if len(parts) >= 2:
            return parts[1] + '::' + parts[0]
        return parts[0]
    return body


def dis_func(seg, off, size=0x400, maxlines=400):
    idx = seg - 1
    va = base + sections[idx][0] + off
    code = read(va, size)
    count = 0
    for ins in md.disasm(code, va):
        line = "0x%08x  %-7s %s" % (ins.address, ins.mnemonic, ins.op_str)
        if ins.mnemonic == 'call' and ins.op_str.startswith('0x'):
            tgt = resolve_thunk(int(ins.op_str, 16))
            s = sym(tgt)
            if s:
                line += "   ; -> %s" % undecorate(s.split('+')[0])
        print(line)
        count += 1
        if ins.mnemonic in ('ret', 'retn') and count > 8:
            pass
        if count >= maxlines:
            break


if __name__ == "__main__":
    seg = int(sys.argv[1])
    off = int(sys.argv[2], 16)
    size = int(sys.argv[3], 0) if len(sys.argv) > 3 else 0x400
    dis_func(seg, off, size)
