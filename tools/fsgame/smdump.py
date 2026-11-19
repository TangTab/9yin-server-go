# -*- coding: utf-8 -*-
"""FsSoloMember.dll CKernel vtable 转储：VA -> 符号（FsSoloMember.map）。"""
import struct, re

DLL = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloMember.dll"
MAP = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloMember.map"

d = open(DLL, "rb").read()
pe = struct.unpack_from("<I", d, 0x3C)[0]
nsec = struct.unpack_from("<H", d, pe+6)[0]
sh = pe + 24 + struct.unpack_from("<H", d, pe+20)[0]
secs = []
for i in range(nsec):
    o = sh + i*40
    vs, va, rs, ro = struct.unpack_from("<IIII", d, o+8)
    secs.append((va, vs, ro))
def va2o(va):
    va -= 0x10000000
    for va0, vs, ro in secs:
        if va0 <= va < va0 + max(vs, 0x1000):
            return va - va0 + ro
# map 符号 (VA)
syms = {}
pat = re.compile(r"^ \d{4}:([0-9a-f]{8})\s+(\S+)\s+([0-9a-f]{8})")
with open(MAP, errors="replace") as f:
    for line in f:
        m = pat.match(line)
        if m:
            off, name, va = m.groups()
            syms.setdefault(int(va, 16), name)

def dump_vtable(va, label, n=90):
    print("==", label, hex(va))
    o = va2o(va)
    for i in range(n):
        (fn,) = struct.unpack_from("<I", d, o + 4*i)
        s = syms.get(fn)
        print("  [%3d] +%04x  %08x  %s" % (i, 4*i, fn, s if s else "?"))

if __name__ == "__main__":
    import sys
    dump_vtable(0x10351c0c, "CKernel vtable", int(sys.argv[1]) if len(sys.argv)>1 else 90)
