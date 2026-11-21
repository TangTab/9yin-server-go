# -*- coding: utf-8 -*-
"""极简 MSF/PDB 解析器：提取模块符号流中的 PROC32/PUBSYM32 记录
（函数名 + segment + offset），输出 TSV。仅支持 MSF 7.00 明文 PDB。"""
import struct
import sys


class MSF:
    def __init__(self, path):
        d = open(path, 'rb').read()
        self.data = d
        assert d[:32] == b"Microsoft C/C++ MSF 7.00\r\n\x1aDS\x00\x00\x00", "not MSF 7.00"
        (self.block_size, self.free_map, self.num_blocks,
         self.dir_bytes, _, self.block_map_addr) = struct.unpack_from("<IIIIII", d, 32)
        nblocks_dir = (self.dir_bytes + self.block_size - 1) // self.block_size
        bm = self.block_map_addr * self.block_size
        dir_blocks = struct.unpack_from("<%dI" % nblocks_dir, d, bm)
        directory = b"".join(d[b * self.block_size:(b + 1) * self.block_size] for b in dir_blocks)[:self.dir_bytes]
        pos = 0
        num_streams = struct.unpack_from("<I", directory, pos)[0]
        pos = 4
        sizes = []
        for _ in range(num_streams):
            sizes.append(struct.unpack_from("<I", directory, pos)[0])
            pos += 4
        streams = []
        for sz in sizes:
            if sz == 0xFFFFFFFF:
                sz = 0
            nblk = (sz + self.block_size - 1) // self.block_size
            blocks = struct.unpack_from("<%dI" % nblk, directory, pos) if nblk else ()
            pos += 4 * nblk
            content = b"".join(d[b * self.block_size:(b + 1) * self.block_size] for b in blocks)[:sz] if sz else b""
            streams.append(content)
        self.streams = streams

    def stream(self, idx):
        return self.streams[idx]


def read_sz(buf, off):
    end = buf.index(b"\x00", off)
    return buf[off:end], end + 1


def parse_db_modules(msf):
    dbi = msf.stream(3)
    (magic, ver, age, gp_idx, pub_idx, _, symrec_idx,
     gp_sz, pub_sz, symrec_sz, mod_sz) = struct.unpack_from("<IHHHHHIHHHI", dbi, 0)
    assert magic == 0xFFFFFFFF, "bad dbi header"
    off = 64
    end = off + mod_sz
    modules = []
    while off < end - 8:
        name1, off = read_sz(dbi, off)
        name2, off = read_sz(dbi, off)
        vals = struct.unpack_from("<HIIII", dbi, off)
        stream_idx, cb_sym, cb_old, cb_line, extra = vals
        off += 2 + 4 * 3
        # 之后是 u16(line stream/源文件数) + 18B section contribution + 尾部标志
        # 自适应：若 stream_idx 不可信则与 extra 互换再试
        if stream_idx >= len(msf.streams) or cb_sym > 0x1000000:
            stream_idx2 = extra & 0xFFFF
            if stream_idx2 < len(msf.streams):
                stream_idx = stream_idx2
        # 跳到下一个记录：section contribution 18B + u32 fWritten + u32 fEC → 26B
        # llvm 布局: u16 SourceFileCount, SC(18B), 无尾部 → 20B；两形态都对齐 4
        skip = 20
        # 尝试对齐判定：peek 下一个记录开头应是可读 ascii 之一
        for cand in (20, 26, 24, 28):
            no = off + cand
            if no + 2 < end:
                b1, b2 = dbi[no], dbi[no + 1]
                if 0x20 <= b1 < 0x7F or (b1 == 0 and b2 == 0):
                    skip = cand
                    break
            else:
                skip = cand
                break
        off += skip
        if name1 or name2:
            modules.append((name1.decode('mbcs', 'replace'), name2.decode('mbcs', 'replace'),
                            stream_idx, cb_sym))
    return modules, gp_idx, pub_idx


def parse_sym_records(buf, base_off, size):
    out = []
    pos = base_off
    end = min(base_off + size, len(buf))
    while pos + 4 <= end:
        rec_len = struct.unpack_from("<H", buf, pos)[0]
        rec_typ = struct.unpack_from("<H", buf, pos + 2)[0]
        if rec_len <= 0:
            break
        body = pos + 4
        nxt = body + rec_len
        if nxt > end:
            break
        try:
            if rec_typ == 0x110E:  # S_PUBSYM32: flags u32, off u32, seg u16, name
                off32 = struct.unpack_from("<I", buf, body + 4)[0]
                seg16 = struct.unpack_from("<H", buf, body + 8)[0]
                nm, _ = read_sz(buf, body + 10)
                out.append((seg16, off32, nm.decode('mbcs', 'replace')))
            elif rec_typ in (0x110F, 0x1110, 0x1112, 0x1113):  # LPROC32/GPROC32/LPROC32ID/GPROC32ID
                off32 = struct.unpack_from("<I", buf, body + 24)[0]
                seg16 = struct.unpack_from("<H", buf, body + 28)[0]
                nm, _ = read_sz(buf, body + 30)
                out.append((seg16, off32, nm.decode('mbcs', 'replace')))
        except Exception:
            pass
        pos = nxt
    return out


def main(pdb_path, out_path):
    msf = MSF(pdb_path)
    print("streams:", len(msf.streams))
    modules, gp, pub = parse_db_modules(msf)
    print("modules:", len(modules), "gp:", gp, "pub:", pub)
    rows = []
    for name1, name2, idx, symsz in modules:
        if idx >= len(msf.streams) or symsz == 0 or idx == 0xFFFF:
            continue
        buf = msf.stream(idx)
        if len(buf) < 8:
            continue
        sig = struct.unpack_from("<H", buf, 0)[0]
        base = 4 if sig in (2, 12) else 0
        for seg, off, nm in parse_sym_records(buf, base, symsz):
            rows.append((seg, off, nm, name1 + "|" + name2))
    seen = set()
    with open(out_path, "w", encoding="utf-8", errors="replace") as f:
        for seg, off, nm, mod in rows:
            key = (seg, off, nm)
            if key in seen:
                continue
            seen.add(key)
            f.write("%d\t0x%X\t%s\t%s\n" % (seg, off, nm, mod))
    print("wrote", len(seen), "symbols ->", out_path)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
