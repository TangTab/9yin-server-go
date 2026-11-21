# -*- coding: utf-8 -*-
"""全 .text 扫描 `mov byte ptr [dst], imm8`（C6 /0 opcode 形态），
映射到 PDB 符号函数，按 opcode 值分组输出。"""
import bisect
import sys
import pefile
from capstone import *

DLL = r"E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsSoloMember.dll"
SYMS = r"C:\Users\Lizy\AppData\Local\Temp\FsSoloMember_syms.tsv"
pe = pefile.PE(DLL, fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
sections = [(s.VirtualAddress, s.Misc_VirtualSize, s.Name.decode(errors='replace').strip('\x00'))
            for s in pe.sections]

rows = []
with open(SYMS, encoding='utf-8') as f:
    for line in f:
        parts = line.rstrip('\n').split('\t')
        if len(parts) < 3:
            continue
        seg, off, nm = int(parts[0]), int(parts[1], 16), parts[2]
        if seg - 1 < len(sections):
            rows.append((base + sections[seg - 1][0] + off, nm))
rows.sort()
va_list = [r[0] for r in rows]
va_names = [r[1] for r in rows]


def sym(va):
    i = bisect.bisect_right(va_list, va) - 1
    if i >= 0 and va - va_list[i] < 0x1000:
        return va_names[i]
    return "?unknown"


def undecorate(nm):
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


md = Cs(CS_ARCH_X86, CS_MODE_32)
target_ops = {int(x, 0) for x in sys.argv[1:]} or set(range(0x100))

for sva, vsz, sname in sections:
    if not sname.startswith('.text') or sname == '.textbss':
        continue
    data = pe.get_data(sva, vsz)
    pos = 0
    while pos < len(data):
        got = False
        for ins in md.disasm(data[pos:], base + sva + pos):
            got = True
            if ins.mnemonic == 'mov' and ins.op_str.startswith('byte ptr [') and ', 0x' in ins.op_str:
                try:
                    imm = int(ins.op_str.split(', 0x')[1], 16)
                except (IndexError, ValueError):
                    imm = None
                if imm is not None and imm in target_ops:
                    print("op=0x%02x  %-52s  %s" % (imm, undecorate(sym(ins.address)), ins.op_str))
            pos = ins.address + ins.size - (base + sva)
        if not got:
            pos += 1
