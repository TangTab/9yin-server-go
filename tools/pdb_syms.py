# -*- coding: utf-8 -*-
"""最小 MSF/PDB 符号提取器（无第三方依赖）。

针对 BGserverCJ 快照无 .map 只有 .pdb 的模块（FsVerify/FsLister/FsGame…）：
解析 MSF 容器 → DBI 头 → globals/symrec 流，提取
S_PUB32(0x110E) / S_GPROC32(0x1110) / S_LPROC32(0x110F) 符号（地址+函数长度+名字）。

用法：
  python pdb_syms.py <x.pdb> list [regex]     全部符号（粗解 C++ 名）
  python pdb_syms.py <x.pdb> gproc [regex]    仅函数（含长度）
  python pdb_syms.py <x.pdb> pub [regex]      仅公共符号
  python pdb_syms.py <x.pdb> tsv              va\tsize\tname 纯表（供脚本消费）
"""
import re
import struct
import sys

S_PUB32 = 0x110E
S_GPROC32 = 0x1110
S_LPROC32 = 0x110F
S_PUB32_ST = 0x1103  # ST 变体：u8 前缀长度名（少见，兜底解析）
S_GPROC32_ST = 0x1125
S_LPROC32_ST = 0x1124


class Pdb:
    def __init__(self, path):
        d = open(path, "rb").read()
        assert d[:20] == b"Microsoft C/C++ MSF ", d[:32]
        self.block_size, free, self.num_blocks, ndir, unk, bmap = struct.unpack_from(
            "<6I", d, 32
        )
        # 目录块链
        dir_blocks = []
        off = bmap * self.block_size
        while len(dir_blocks) * self.block_size < ndir:
            dir_blocks.append(struct.unpack_from("<I", d, off)[0])
            off += 4
        raw = b"".join(
            d[b * self.block_size : (b + 1) * self.block_size] for b in dir_blocks
        )[:ndir]
        (nstream,) = struct.unpack_from("<I", raw, 0)
        sizes = struct.unpack_from("<%dI" % nstream, raw, 4)
        pos = 4 + 4 * nstream
        self.streams = []
        for sz in sizes:
            if sz > self.num_blocks * self.block_size:
                sz = 0  # 哨兵 size（自由块图流），视为空流
            nblk = (sz + self.block_size - 1) // self.block_size
            blks = struct.unpack_from("<%dI" % nblk, raw, pos) if sz else []
            pos += 4 * nblk
            self.streams.append(
                b"".join(
                    d[b * self.block_size : (b + 1) * self.block_size] for b in blks
                )[:sz]
            )
        self.d = d

    def dbi_header(self):
        s = self.streams[3]
        if len(s) < 64:
            return None
        # 0 sig(0xFFFFFFFF) 4 ver 8 age 12 globalsIdx 14 build 16 publicsIdx
        # 18 pdbVer 20 symrecIdx 22 rbVer 24.. 子流大小
        sig, ver, age, gidx, build, pidx, _, sidx = struct.unpack_from("<IIIHHHHH", s, 0)
        return {
            "sig": sig,
            "ver": ver,
            "globals": gidx,
            "publics": pidx,
            "symrec": sidx,
        }


def iter_syms(stream):
    """产出 (rectype, va_off, seg, size_or_0, name)。PROC32 带 len。"""
    pos = 0
    n = len(stream)
    while pos + 4 <= n:
        (reclen, rectype) = struct.unpack_from("<HH", stream, pos)
        if reclen < 2 or pos + 2 + reclen > n:
            break
        body = pos + 4
        end = pos + 2 + reclen
        if rectype in (S_GPROC32, S_LPROC32):
            # pParent pEnd pNext len dbgStart dbgEnd typind off seg flags name
            ln, off, seg = struct.unpack_from("<IxxxxxxIxxxxxxH", stream, body - 4 + 20)
            # 精确布局：reclen,rectype 后 = pParent,pEnd,pNext(12B) len(4) DbgStart(4) DbgEnd(4) typind(4) off(4) seg(2) flags(1) name
            ln = struct.unpack_from("<I", stream, body + 12)[0]
            off = struct.unpack_from("<I", stream, body + 28)[0]
            seg = struct.unpack_from("<H", stream, body + 32)[0]
            name = stream[body + 37 : end].split(b"\0", 1)[0]
            yield rectype, off, seg, ln, name.decode("latin1")
        elif rectype == S_PUB32:
            off = struct.unpack_from("<I", stream, body + 4)[0]
            seg = struct.unpack_from("<H", stream, body + 8)[0]
            name = stream[body + 10 : end].split(b"\0", 1)[0]
            yield rectype, off, seg, 0, name.decode("latin1")
        elif rectype == S_PUB32_ST:
            off = struct.unpack_from("<I", stream, body + 4)[0]
            seg = struct.unpack_from("<H", stream, body + 8)[0]
            ln = stream[body + 10]
            name = stream[body + 11 : body + 11 + ln]
            yield rectype, off, seg, 0, name.decode("latin1")
        pos = end


def demangle(n):
    m = re.match(r"\?(\w+)@(\w+)@@", n)
    if m:
        return "%s::%s" % (m.group(2), m.group(1))
    m = re.match(r"\?\?0(\w+)@@.*", n)
    if m:
        return m.group(1) + "::ctor"
    m = re.match(r"\?\?1(\w+)@@.*", n)
    if m:
        return m.group(1) + "::dtor"
    m = re.match(r"\?(\w+)@(\w+)@(\w+)@@", n)
    if m:
        return "%s::%s::%s" % (m.group(3), m.group(2), m.group(1))
    m = re.match(r"\?(\w+)@(\w+)@@.*", n)
    if m:
        return m.group(2) + "::" + m.group(1)
    m = re.match(r"\?(\w+)@@YAPBDPBD", n)
    if m:
        return m.group(1) + "(char*)"
    return n


def main():
    path = sys.argv[1]
    mode = sys.argv[2] if len(sys.argv) > 2 else "list"
    pat = re.compile(sys.argv[3], re.I) if len(sys.argv) > 3 else None
    pdb = Pdb(path)
    hdr = pdb.dbi_header()
    syms = {}
    # symrec 流 = 全部 PUB32（含数据符号）；各模块流 = GPROC32（带函数长度）。
    # 模块流不做 DBI 子流定位，直接全流扫描——垃圾流会快速失败，无副作用。
    scan = [hdr["symrec"]] + list(range(len(pdb.streams)))
    for idx in scan:
        s = pdb.streams[idx]
        if not s or len(s) < 8:
            continue
        try:
            recs = list(iter_syms(s))
        except Exception:
            continue
        for rectype, off, seg, ln, name in recs:
            if not name:
                continue
            key = (off, seg, name)
            if key not in syms or (ln and not syms[key][2]):
                syms[key] = (rectype, off, ln, name, seg)
    rows = sorted(syms.values(), key=lambda r: r[1])
    imagebase = 0x10000000  # BGserverCJ DLL 均为默认基址；seg>1 时减段基
    out = []
    for rectype, off, ln, name, seg in rows:
        rva = off  # seg==1 时 off 即 RVA；其他段按 DBI secContr 换算（本快照主要 seg1）
        disp = demangle(name)
        if pat and not (pat.search(disp) or pat.search(name)):
            continue
        if mode == "tsv":
            out.append("%x\t%d\t%s" % (rva, ln, disp))
        elif mode == "gproc":
            if rectype in (S_GPROC32, S_LPROC32):
                out.append("%08x len=%-6d %s" % (rva, ln, disp))
        elif mode == "pub":
            if rectype in (S_PUB32, S_PUB32_ST):
                out.append("%08x %s" % (rva, disp))
        else:
            tag = "PROC" if rectype in (S_GPROC32, S_LPROC32) else "pub "
            out.append("%08x %s len=%-6d %s" % (rva, tag, ln, disp))
    print("\n".join(out))
    print("# total %d" % len(out), file=sys.stderr)


if __name__ == "__main__":
    main()
