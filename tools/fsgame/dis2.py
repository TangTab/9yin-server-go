# -*- coding: utf-8 -*-
"""二期分析助手：RVA→反汇编 + 字符串/符号解析（DLL 路径改为 E: 盘真档）。"""
import struct, sys, bisect
import capstone

import os
DLL = os.environ.get("FSGAME_DLL", r"E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsGame.dll")
d = open(DLL, "rb").read()
pe = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, pe+6)[0]
sh = pe + 24 + struct.unpack_from("<H", d, pe+20)[0]
secs = []
for i in range(nsec):
    o = sh + i*40
    name = d[o:o+8].rstrip(b"\0").decode()
    vs, va, rs, ro = struct.unpack_from("<IIII", d, o+8)
    secs.append((name, va, vs, ro, rs))

def r2o(rva):
    for name, va, vs, ro, rs in secs:
        if va <= rva < va + max(vs, rs):
            return rva - va + ro
    return None

def read(rva, n):
    o = r2o(rva)
    return d[o:o+n] if o is not None else b""

def cstr(rva, maxn=120):
    o = r2o(rva)
    if o is None:
        return None
    e = d.find(b"\0", o, o + maxn)
    if e < 0:
        return None
    raw = d[o:e]
    if len(raw) < 2:
        return None
    if all(32 <= b < 127 for b in raw):
        return raw.decode()
    try:
        s = raw.decode("gbk")
        if s.isprintable():
            return s
    except Exception:
        pass
    return None

sys.path.insert(0, r"E:\Workspace\project\game\9yin-go-server1\tools\fsgame")
from maparse import parse_map
syms, _ = parse_map()
byrva = {}
for rva, name, obj in syms:
    if rva not in byrva:
        byrva[rva] = (name, obj)
ks = sorted(byrva)

def sym(rva):
    return byrva.get(rva)

def symname(rva):
    s = byrva.get(rva)
    if not s:
        return ""
    n = s[0]
    return n

def func_size(rva):
    i = bisect.bisect_right(ks, rva) - 1
    if i + 1 < len(ks) and ks[i] == rva:
        return min(ks[i+1] - rva, 0x3000)
    return 0x400

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True

def disasm(rva, n=None, maxins=4000):
    if n is None:
        n = func_size(rva)
    code = read(rva, n)
    out = []
    for ins in md.disasm(code, rva):
        line = "%08x  %-7s %s" % (ins.address, ins.mnemonic, ins.op_str)
        # call/jmp 目标解析
        if ins.mnemonic in ("call", "jmp") and ins.operands and ins.operands[0].type == capstone.x86.X86_OP_IMM:
            t = ins.operands[0].imm
            nm = symname(t)
            if nm:
                line += "   ; " + nm[:110]
        # push imm → 字符串尝试
        if ins.mnemonic == "push" and ins.operands and ins.operands[0].type == capstone.x86.X86_OP_IMM:
            v = ins.operands[0].imm
            if 0x00400000 < v < 0x10000000:
                s = cstr(v)
                if s:
                    line += "   ; \"%s\"" % s[:80]
        out.append(line)
        if len(out) >= maxins:
            break
    return out

def strings_in(rva, n=None):
    """函数体内引用的全部字符串（按出现序）。VA=0x10xxxxxx（RVA+基址）。"""
    if n is None:
        n = func_size(rva)
    code = read(rva, n)
    res = []
    for ins in md.disasm(code, rva):
        for op in ins.operands:
            if op.type == capstone.x86.X86_OP_IMM and (0x00400000 < op.imm < 0x10000000 or 0x10d67000 <= op.imm < 0x10ea0000):
                v = op.imm - 0x10000000 if op.imm >= 0x10000000 else op.imm
                s = cstr(v)
                if s and len(s) >= 2:
                    res.append((ins.address, s))
            if op.type == capstone.x86.X86_OP_MEM and op.mem.base == 0 and (0x00400000 < op.mem.disp < 0x10000000 or 0x10d67000 <= op.mem.disp < 0x10ea0000):
                v = op.mem.disp - 0x10000000 if op.mem.disp >= 0x10000000 else op.mem.disp
                s = cstr(v)
                if s and len(s) >= 2:
                    res.append((ins.address, s))
    return res

if __name__ == "__main__":
    rva = int(sys.argv[1], 16)
    n = int(sys.argv[2], 16) if len(sys.argv) > 2 else None
    print("\n".join(disasm(rva, n)))
