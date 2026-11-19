# -*- coding: utf-8 -*-
"""FsGame.map 解析器： publics -> (rva, name, obj) 表 + 类/命名空间统计。"""
import re, sys, struct, collections

MAP = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsGame.map"
DLL = r"F:\Game\Online\9yin\BGserverCJ\Debug\FsGame.dll"

def section_rvas():
    d = open(DLL, "rb").read()
    pe = struct.unpack_from("<I", d, 0x3C)[0]
    nsec = struct.unpack_from("<H", d, pe + 6)[0]
    opt = struct.unpack_from("<H", d, pe + 20)[0]
    base = pe + 24 + opt
    out = {}
    for i in range(nsec):
        off = base + i * 40
        name = d[off:off+8].rstrip(b"\0").decode()
        vsize, rva = struct.unpack_from("<II", d, off + 8)
        out[i + 1] = (name, rva, vsize)
    return out

def parse_map():
    secs = section_rvas()
    syms = []
    pat = re.compile(r"^ (\d{4}):([0-9a-f]{8})\s+(\S+)\s+([0-9a-f]{8})\s+(.*)$")
    with open(MAP, "r", errors="replace") as f:
        in_pub = False
        for line in f:
            if "Publics by Value" in line:
                in_pub = True
                continue
            if not in_pub:
                continue
            m = pat.match(line.rstrip("\n"))
            if not m:
                continue
            sec, off, name, va, rest = m.groups()
            si = int(sec)
            if si not in secs:
                continue
            _, rva0, _ = secs[si]
            rva = rva0 + int(off, 16)
            obj = rest.split()[-1] if rest else ""
            syms.append((rva, name, obj))
    return syms, secs

def demangle_simple(n):
    """粗解 ?name@class@@... -> class::name；只做外层。"""
    if not n.startswith("?"):
        return None
    m = re.match(r"\?([\?$@][^\s]*)", n)
    return n

if __name__ == "__main__":
    syms, secs = parse_map()
    print("sections:", {k: (v[0], hex(v[1]), hex(v[2])) for k, v in secs.items()})
    print("total publics:", len(sys.argv) and len(syms))
    # 统计 C++ 类符号: ?xx@Class@@ 形式
    cls = collections.Counter()
    func_in_cls = collections.Counter()
    pat_cls = re.compile(r"^\?[^\?].*?@([A-Z][A-Za-z0-9_]+)@@")
    for rva, name, obj in syms:
        m = pat_cls.match(name)
        if m:
            cls[m.group(1)] += 1
    print("distinct top classes:", len(cls))
    for c, n in cls.most_common(80):
        print("%6d  %s" % (n, c))
