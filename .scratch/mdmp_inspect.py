"""Inspect a Windows minidump: list streams, modules, and hunt for fxgame.exe image data."""
import struct, sys, io

path = sys.argv[1]
f = open(path, "rb")

hdr = f.read(32)
sig, version, nstreams, dir_rva, checksum, tstamp = struct.unpack("<IIIIII", hdr[:24])
flags = struct.unpack("<Q", hdr[24:32])[0]
assert sig == 0x504D444D, "not MDMP"
print(f"streams={nstreams} dir_rva={dir_rva:#x} flags={flags:#x}")

STREAM_NAMES = {0:"Unused",3:"ThreadList",4:"ModuleList",5:"MemoryList",6:"Exception",
                7:"SystemInfo",8:"ThreadExList",9:"Memory64List",10:"CommentA",11:"CommentW",
                12:"HandleData",13:"FunctionTable",14:"UnloadedModuleList",15:"MiscInfo",
                16:"MemoryInfoList",17:"ThreadInfoList",18:"HandleOperationList",22:"SystemMemoryInfo",
                23:"ProcessVmCounters",21:"TokenStream"}

f.seek(dir_rva)
entries = []
for i in range(nstreams):
    st, dsz, rva = struct.unpack("<III", f.read(12))
    entries.append((st, dsz, rva))
    print(f"  stream type={st}({STREAM_NAMES.get(st,'?')}) size={dsz:#x} rva={rva:#x}")

def mdmp_string(rva):
    if rva == 0: return ""
    pos = f.tell()
    f.seek(rva)
    n = struct.unpack("<I", f.read(4))[0]
    s = f.read(n).decode("utf-16le", "replace")
    f.seek(pos)
    return s

def parse_modules(st):
    st, dsz, rva = next(e for e in entries if e[0] == st)
    f.seek(rva)
    cnt = struct.unpack("<I", f.read(4))[0]
    mods = []
    for i in range(cnt):
        b = f.read(108)
        base, size, csum, tds, name_rva = struct.unpack("<QIIII", b[:24])
        mods.append((base, size, mdmp_string(name_rva)))
    return mods

def parse_unloaded():
    e = next((e for e in entries if e[0] == 14), None)
    if not e: return []
    f.seek(e[2])
    cnt = struct.unpack("<I", f.read(4))[0]
    out = []
    for i in range(cnt):
        b = f.read(24)
        base, size, csum, tds, name_rva = struct.unpack("<QIIII", b[:24])
        out.append((base, size, mdmp_string(name_rva)))
    return out

def parse_memory64():
    e = next((e for e in entries if e[0] == 9), None)
    if not e: return None, []
    f.seek(e[2])
    base_rva, nranges = struct.unpack("<QQ", f.read(16))
    ranges = []
    for i in range(nranges):
        start, dsz = struct.unpack("<QQ", f.read(16))
        ranges.append([start, dsz, base_rva])
        base_rva += dsz
    return base_rva, ranges

print("\n--- modules ---")
mods = parse_modules(4)
for base, size, name in sorted(mods):
    print(f"  {base:>12x} {size:>9x}  {name}")
print(f"total {len(mods)} modules")

un = parse_unloaded()
if un:
    print("\n--- unloaded modules ---")
    for base, size, name in un:
        print(f"  {base:>12x} {size:>9x}  {name}")

print("\n--- memory64 ranges ---")
tail, ranges = parse_memory64()
tot = sum(r[1] for r in ranges)
print(f"  {len(ranges)} ranges, total {tot:#x} ({tot/1024/1024:.1f} MiB)")

# check module coverage: how many bytes of each module's image are inside memory ranges
def coverage(base, size):
    got = 0
    for start, dsz, _ in ranges:
        s, e = start, start + dsz
        lo, hi = max(s, base), min(e, base + size)
        if hi > lo: got += hi - lo
    return got

print("\n--- module memory coverage ---")
for base, size, name in sorted(mods):
    got = coverage(base, size)
    pct = 100.0 * got / size if size else 0
    print(f"  {name:<50} {got:>10x}/{size:<8x} {pct:5.1f}%")
