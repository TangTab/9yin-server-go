# -*- coding: utf-8 -*-
"""Scan a session dump pair for 内功/技能 upgrade moments:
C2S: custom 1006 subcommands (修炼), item-use (skill book)
S2C: 0x18 view43/view40 rows, 0x10 isView=1 obj=43/40 updates, custom 1006 replies
Usage: python -I scan_upgrade_frames.py <dir-with-bins>
"""
import importlib.util, os, struct, sys

SMOKE = r"G:\pojie\ai\9yin-server\tools\smoke"
_spec = importlib.util.spec_from_file_location("capture_deep", os.path.join(SMOKE, "capture_deep.py"))
_cd = importlib.util.module_from_spec(_spec); _spec.loader.exec_module(_cd)

TABLE = None
def load_table():
    global TABLE
    p = r"G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table.txt"
    entries = []
    with open(p, "rb") as f:
        for line in f:
            parts = line.decode("utf-8", "replace").rstrip("\r\n").split("\t")
            if len(parts) >= 3 and parts[0].strip().isdigit():
                entries.append((parts[1], int(parts[2])))
    TABLE = entries

def prop_type(idx):
    if TABLE and idx < len(TABLE):
        return TABLE[idx][1]
    return 3

def read_prop(frame, off, ptype):
    if ptype == 1: return frame[off], off + 1
    if ptype == 2: return struct.unpack_from('<H', frame, off)[0], off + 2
    if ptype == 3: return struct.unpack_from('<i', frame, off)[0], off + 4
    if ptype == 5: return round(struct.unpack_from('<f', frame, off)[0], 3), off + 4
    if ptype == 7:
        n = struct.unpack_from('<I', frame, off)[0]; off += 4
        return frame[off:off+n].rstrip(b'\x00').decode('gbk','replace'), off + n
    return None, off

def scan(path, direction):
    frames = _cd.load(path)
    print(f"== {os.path.basename(path)} ({direction}) {len(frames)} frames")
    if direction == "s2c":
        for idx, opcode, msg_id, vals in _cd.s2c_customs(frames):
            if msg_id == 1006:
                args = " ".join(_cd.fmt_val(v) for v in vals[1:4])
                print(f"[{idx}] 0x{opcode:02X} custom 1006 {args}")
    for i, fr in enumerate(frames):
        if not fr: continue
        op = fr[0]
        if op == 0x16:
            view = struct.unpack_from('<H', fr, 1)[0]
            if view in (40, 43, 45, 46):
                print(f"[{i}] 0x16 DELETE view={view}")
        if op == 0x18:
            view, obj = struct.unpack_from('<HH', fr, 1)
            if view in (40, 43, 45, 46):
                pc = struct.unpack_from('<H', fr, 5)[0]
                props = {}
                off = 7
                for _ in range(pc):
                    if off + 2 > len(fr): break
                    idx = struct.unpack_from('<H', fr, off)[0]; off += 2
                    if idx == 1440: ptype = 7
                    elif idx == 1465: ptype = 1
                    else: ptype = prop_type(idx)
                    val, off = read_prop(fr, off, ptype)
                    props[idx] = val
                cid = props.get(1440); lvl = props.get(1465)
                print(f"[{i}] 0x18 view={view} obj={obj} ConfigID={cid} Level={lvl} props={ {k:v for k,v in props.items() if k in (1440,1465,747,1512,1513)} }")
        elif op == 0x10:
            isview = fr[1]
            if isview == 1:
                oid, owner, pc = struct.unpack_from('<IIH', fr, 2)
                if oid in (40, 43, 45):
                    props = {}
                    off = 12
                    for _ in range(pc):
                        if off + 2 > len(fr): break
                        idx = struct.unpack_from('<H', fr, off)[0]; off += 2
                        ptype = 1 if idx == 1465 else prop_type(idx)
                        val, off = read_prop(fr, off, ptype)
                        props[idx] = val
                    print(f"[{i}] 0x10 isView=1 view={oid} row={owner} props={props}")

def main():
    load_table()
    d = sys.argv[1]
    for name in sorted(os.listdir(d)):
        if name.endswith("_c2s.bin"):
            c2s = os.path.join(d, name)
            s2c = c2s.replace("_c2s.bin", "_s2c.bin")
            if os.path.exists(s2c):
                scan(s2c, "s2c")
main()
