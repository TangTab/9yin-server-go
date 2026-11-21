# -*- coding: utf-8 -*-
"""FsSoloWorld.dll 反汇编助手：按 PDB seg/off 定位函数，capstone 输出。"""
import sys
import pefile
from capstone import *

DLL = r"E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsSoloWorld.dll"
pe = pefile.PE(DLL, fast_load=True)
base = pe.OPTIONAL_HEADER.ImageBase
# PDB seg N → PE 第 N-1 节（seg1=头之后第一节）
sections = []
for s in pe.sections:
    sections.append((s.VirtualAddress, s.Misc_VirtualSize, s.Name.decode().strip('\x00')))


def seg_off_to_rva(seg, off):
    # seg 从 1 计：seg2 = 第一个含代码的节（.text）
    idx = seg - 1
    if 0 <= idx < len(sections):
        va, vsz, name = sections[idx]
        return va + off, name
    return None, None


def read_code(rva, size):
    return pe.get_data(rva, size)


md = Cs(CS_ARCH_X86, CS_MODE_32)
md.detail = False


def dis(rva, size, show=True):
    code = read_code(rva, size)
    out = []
    for ins in md.disasm(code, base + rva):
        line = "0x%08x  %-8s %s" % (ins.address, ins.mnemonic, ins.op_str)
        out.append(line)
        if show:
            print(line)
    return out


if __name__ == "__main__":
    seg = int(sys.argv[1])
    off = int(sys.argv[2], 16)
    size = int(sys.argv[3], 0) if len(sys.argv) > 3 else 0x120
    rva, name = seg_off_to_rva(seg, off)
    print("; section:", name, "rva:", hex(rva), "va:", hex(base + rva))
    dis(rva, size)
