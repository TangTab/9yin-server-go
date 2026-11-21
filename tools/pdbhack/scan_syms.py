# -*- coding: utf-8 -*-
"""绕过 DBI：遍历 PDB 全部流，符号流签名(2/12)解析 PROC32/PUBSYM32，
过滤合法记录输出 TSV（seg, off, rectype, name, stream）。"""
import struct
import sys

sys.path.insert(0, 'tools/pdbhack')
from msf_symbols import MSF


def parse_records(buf, base, size, align4=False):
    """CV 记录：len u16(不含自身)，type u16，body。nxt = pos+2+len。"""
    out = []
    pos = base
    end = min(base + size, len(buf))
    while pos + 4 <= end:
        rec_len = struct.unpack_from("<H", buf, pos)[0]
        rec_typ = struct.unpack_from("<H", buf, pos + 2)[0]
        if rec_len < 2 or rec_len > 0x2000:
            break
        nxt = pos + 2 + rec_len
        if align4:
            nxt = (nxt + 3) & ~3
        if nxt > end:
            break
        try:
            if rec_typ == 0x110E:  # PUBSYM32: flags u32, off u32, seg u16, name
                off32 = struct.unpack_from("<I", buf, pos + 8)[0]
                seg16 = struct.unpack_from("<H", buf, pos + 12)[0]
                e = buf.index(b"\x00", pos + 14)
                nm = buf[pos + 14:e]
                out.append((seg16, off32, rec_typ, nm))
            elif rec_typ in (0x110F, 0x1110, 0x1112, 0x1113):  # PROC32 族
                off32 = struct.unpack_from("<I", buf, pos + 28)[0]
                seg16 = struct.unpack_from("<H", buf, pos + 30)[0]
                e = buf.index(b"\x00", pos + 32)
                nm = buf[pos + 32:e]
                out.append((seg16, off32, rec_typ, nm))
        except (ValueError, struct.error):
            pass
        pos = nxt
    return out


def plausible(seg, off, typ, nm):
    if not (1 <= seg <= 16):
        return False
    if off > 0x2000000:
        return False
    if len(nm) < 2 or len(nm) > 512:
        return False
    c0 = nm[0:1]
    if c0 not in (b'?', b'_', b'$') and not (65 <= nm[0] <= 90) and not (97 <= nm[0] <= 122):
        return False
    return all(32 <= b < 127 for b in nm)


def main(pdb_path, out_path):
    msf = MSF(pdb_path)
    rows = []
    for si, buf in enumerate(msf.streams):
        if len(buf) < 16:
            continue
        best = []
        for base in (0, 4):
            for a4 in (False, True):
                got = [r for r in parse_records(buf, base, len(buf) - base, a4)
                       if plausible(*r)]
                if len(got) > len(best):
                    best = got
        for seg, off, typ, nm in best:
            rows.append((seg, off, typ, nm, si))
    seen = {}
    for seg, off, typ, nm, si in rows:
        key = (seg, off, nm)
        seen.setdefault(key, si)
    with open(out_path, "w", encoding="utf-8", errors="replace") as f:
        for (seg, off, nm), si in sorted(seen.items()):
            f.write("%d\t0x%X\t%s\t%d\n" % (seg, off, nm.decode('mbcs', 'replace'), si))
    print("streams:", len(msf.streams), "records:", len(rows), "unique:", len(seen), "->", out_path)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
