# -*- coding: utf-8 -*-
"""Official pcapng scan: View43/View40 row-update dialect.
Usage: python -I official_view43_scan.py <pcapng>...
"""
import importlib.util, os, struct, sys
SMOKE = r"G:\pojie\ai\9yin-server\tools\smoke"
spec = importlib.util.spec_from_file_location("odt", os.path.join(SMOKE, "official_drop_trace.py"))
odt = importlib.util.module_from_spec(spec); spec.loader.exec_module(odt)

TABLE = []
with open(r"G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table.txt", "rb") as f:
    for line in f:
        parts = line.decode("utf-8", "replace").rstrip("\r\n").split("\t")
        if len(parts) >= 3 and parts[0].strip().isdigit():
            TABLE.append((parts[1], int(parts[2])))

def ptype(idx):
    return TABLE[idx][1] if idx < len(TABLE) else 3

def read_prop(fr, off, t):
    if t == 1: return fr[off], off+1
    if t == 2: return struct.unpack_from('<H', fr, off)[0], off+2
    if t == 3: return struct.unpack_from('<i', fr, off)[0], off+4
    if t == 5: return round(struct.unpack_from('<f', fr, off)[0],3), off+4
    if t == 7:
        n = struct.unpack_from('<I', fr, off)[0]; off += 4
        return fr[off:off+n].rstrip(b'\x00').decode('gbk','replace'), off+n
    return None, off

def scan(path):
    streams = odt.build(path)
    for st in streams:
        if st['kind'] != 'S2C':
            continue
        t0 = st['pairs'][0][1]
        for i, (fr, ts) in enumerate(st['pairs']):
            if not fr: continue
            op = fr[0]
            rel = (ts - t0) / 1e6
            if op == 0x16:
                v = struct.unpack_from('<H', fr, 1)[0]
                if v in (40,43):
                    print(f"{os.path.basename(path)} [{i}] {rel:8.1f}s 0x16 DEL view={v}")
            elif op == 0x18:
                v, obj = struct.unpack_from('<HH', fr, 1)
                if v in (40, 43):
                    pc = struct.unpack_from('<H', fr, 5)[0]
                    props, off = {}, 7
                    for _ in range(pc):
                        if off+2 > len(fr): break
                        idx = struct.unpack_from('<H', fr, off)[0]; off += 2
                        t = 7 if idx == 1440 else (1 if idx == 1465 else ptype(idx))
                        val, off = read_prop(fr, off, t)
                        props[idx] = val
                    print(f"{os.path.basename(path)} [{i}] {rel:8.1f}s 0x18 view={v} obj={obj} cid={props.get(1440)} lvl={props.get(1465)}")
            elif op == 0x10 and fr[1] == 1:
                oid, owner, pc = struct.unpack_from('<IIH', fr, 2)
                if oid in (40, 43):
                    props, off = {}, 12
                    for _ in range(pc):
                        if off+2 > len(fr): break
                        idx = struct.unpack_from('<H', fr, off)[0]; off += 2
                        t = 1 if idx == 1465 else ptype(idx)
                        val, off = read_prop(fr, off, t)
                        props[idx] = val
                    print(f"{os.path.basename(path)} [{i}] {rel:8.1f}s 0x10 isView=1 view={oid} row={owner} props={props}")

for p in sys.argv[1:]:
    scan(p)
