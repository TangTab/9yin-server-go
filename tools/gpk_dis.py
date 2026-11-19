#!/usr/bin/env python
"""GPKitSvr.dll / GPKitSvr64.dll 反汇编辅助(纯静态分析)。
用法: python -I gpk_dis.py <dll> <rva-hex> [count] [--str]
  --str 顺带 dump .rdata 字符串引用
"""
import struct, sys
from capstone import *

def load(path):
    d = open(path, 'rb').read()
    e = struct.unpack_from('<I', d, 0x3C)[0]
    coff = e + 4
    mach, nsec, tds, ps, ns, optsz, ch = struct.unpack_from('<HHIIIHH', d, coff)
    opt = coff + 20
    magic = struct.unpack_from('<H', d, opt)[0]
    arch = 'x86' if magic == 0x10b else 'x64'
    secs = []
    sh = opt + optsz
    for i in range(nsec):
        name, vsize, va, rsize, ra, *_ = struct.unpack_from('<8sIIIIIIHHI', d, sh + i * 40)
        secs.append((name.rstrip(b'\x00').decode(), vsize, va, rsize, ra))
    def r2o(rva):
        for n, vs, vva, rs, ra in secs:
            if vva <= rva < vva + max(vs, rs):
                return ra + (rva - vva)
        return None
    def o2r(off):
        for n, vs, vva, rs, ra in secs:
            if ra <= off < ra + rs:
                return vva + (off - ra)
        return None
    imgbase = struct.unpack_from('<I' if arch == 'x86' else '<Q', d, opt + (28 if arch == 'x86' else 24))[0]
    return d, secs, r2o, o2r, arch, imgbase

class Dis:
    def __init__(self, path):
        self.path = path
        (self.d, self.secs, self.r2o, self.o2r, self.arch, self.imgbase) = load(path)
        self.md = Cs(CS_ARCH_X86, CS_MODE_32 if self.arch == 'x86' else CS_MODE_64)

    def str_at(self, rva):
        o = self.r2o(rva)
        if o is None: return None
        try:
            e = self.d.index(b'\x00', o, o + 260)
        except ValueError:
            return None
        s = self.d[o:e]
        if len(s) >= 3 and all(0x20 <= c < 0x7f for c in s):
            return s.decode()
        return None

    def disasm(self, rva, count=200, stop_ret=True):
        d = self.d
        out = []
        fo = self.r2o(rva)
        code = d[fo:fo + count * 16]
        n = 0
        for ins in self.md.disasm(code, rva):
            extra = ''
            # annotate immediate/push imm referencing rdata strings
            if ins.mnemonic in ('push', 'mov', 'lea', 'cmp') and '0x' in ins.op_str:
                import re as _re
                m = _re.findall(r'0x([0-9a-f]{6,8})', ins.op_str)
                for h in m:
                    v = int(h, 16)
                    if v > 0x10000000 or v > self.imgbase:
                        s = self.str_at(v - self.imgbase if v >= self.imgbase else v)
                        if s: extra += '  ; "%s"' % s
            out.append('%08x  %-24s %-8s %s%s' % (ins.address, ins.bytes.hex(), ins.mnemonic, ins.op_str, extra))
            n += 1
            if stop_ret and ins.mnemonic in ('ret',) and n > 3:
                break
            if n >= count:
                break
        return '\n'.join(out)

if __name__ == '__main__':
    dll = sys.argv[1]
    rva = int(sys.argv[2], 16)
    cnt = int(sys.argv[3]) if len(sys.argv) > 3 else 200
    dz = Dis(dll)
    print(dz.disasm(rva, cnt))
