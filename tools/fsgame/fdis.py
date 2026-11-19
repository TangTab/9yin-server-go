# -*- coding: utf-8 -*-
"""FsGame.dll 反汇编助手：RVA->bytes, 符号解析, capstone 线性反汇编。"""
import struct, re, sys
import capstone

DLL = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsGame.dll"
d = open(DLL, "rb").read()
pe = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, pe+6)[0]
sh = pe + 24 + struct.unpack_from("<H", d, pe+20)[0]
secs = []
for i in range(nsec):
    o = sh + i*40
    name = d[o:o+8].rstrip(b"\0").decode()
    vs, va, rs, ro = struct.unpack_from("<IIII", d, o+8)
    secs.append((name, va, vs, ro))
def r2o(rva):
    for name, va, vs, ro in secs:
        if va <= rva < va + max(vs, 0x1000):
            return rva - va + ro
    return None
def read(rva, n):
    o = r2o(rva)
    return d[o:o+n]
def cstr(rva, maxn=200):
    o = r2o(rva)
    if o is None: return None
    e = d.find(b"\0", o, o + maxn)
    try: return d[o:e].decode("gbk", "replace")
    except: return repr(d[o:e])

sys.path.insert(0, ".")
from maparse import parse_map
syms, _ = parse_map()
byrva = {}
for rva, name, obj in syms:
    if rva not in byrva:
        byrva[rva] = (name, obj)
def sym(rva):
    return byrva.get(rva)

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True

def func_size(rva):
    ks = sorted(byrva)
    import bisect
    i = bisect.bisect_right(ks, rva) - 1
    if i + 1 < len(ks) and ks[i] == rva:
        return min(ks[i+1] - rva, 0x2000)
    return 0x400

def disasm(rva, n=None, resolve=True):
    if n is None: n = func_size(rva)
    code = read(rva, n)
    out = []
    for ins in md.disasm(code, rva):
        line = "%08x  %-8s %s" % (ins.address, ins.mnemonic, ins.op_str)
        if resolve:
            # call/jmp 目标解析
            m = re.match(r"^(call|jmp|j\w+)$", ins.mnemonic)
            if m and ins.operands and ins.operands[0].type == capstone.x86.X86_OP_IMM:
                t = ins.operands[0].imm
                s = sym(t)
                if s: line += "   ; " + s[0][:80]
            if ins.mnemonic in ("push", "mov", "lea") and ins.operands:
                for op in ins.operands:
                    if op.type == capstone.x86.X86_OP_IMM and 0x1000 < op.imm < 0xebd000:
                        s = sym(op.imm)
                        if s: line += "   ; => " + s[0][:70]
                    if op.type == capstone.x86.X86_OP_MEM and op.mem.base == 0 and 0x1000 < op.mem.disp < 0xebd000:
                        s = sym(op.mem.disp)
                        if s: line += "   ; => " + s[0][:70]
        out.append(line)
    return "\n".join(out)

if __name__ == "__main__":
    rva = int(sys.argv[1], 16)
    n = int(sys.argv[2], 16) if len(sys.argv) > 2 else None
    print(disasm(rva, n))
