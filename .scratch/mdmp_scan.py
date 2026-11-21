"""Scan minidump Memory64 data for PE images and fxgame strings."""
import struct, sys

path = sys.argv[1]
f = open(path, "rb")

hdr = f.read(32)
nstreams, dir_rva = struct.unpack("<II", hdr[8:16])
f.seek(dir_rva)
entries = [struct.unpack("<III", f.read(12)) for _ in range(nstreams)]

def stream(tp):
    return next((dsz, rva) for st, dsz, rva in entries if st == tp)

# --- Memory64 ranges (NumberOfMemoryRanges first, then BaseRva) ---
dsz, rva = stream(9)
f.seek(rva)
nranges, base_rva = struct.unpack("<QQ", f.read(16))
ranges = []
for _ in range(nranges):
    start, size = struct.unpack("<QQ", f.read(16))
    ranges.append((start, size, base_rva))
    base_rva += size
tot = sum(r[1] for r in ranges)
print(f"memory64: {nranges} ranges, {tot/1024/1024:.1f} MiB")

# --- module list (to distinguish mapped-but-unlisted images) ---
dsz, rva = stream(4)
f.seek(rva)
cnt = struct.unpack("<I", f.read(4))[0]
mod_bases = {}
for _ in range(cnt):
    b = f.read(108)
    base, size, csum, tds, name_rva = struct.unpack("<QIIII", b[:24])
    f2pos = f.tell()
    f.seek(name_rva); n = struct.unpack("<I", f.read(4))[0]
    name = f.read(n).decode("utf-16le", "replace")
    f.seek(f2pos)
    mod_bases[base] = (size, name)

# --- memory info list for region type ---
info = []
if any(st == 16 for st, _, _ in entries):
    dsz, rva = stream(16)
    f.seek(rva)
    szh, sze, nent = struct.unpack("<III", f.read(12))
    for _ in range(nent):
        b = f.read(sze)
        ba, aba, aprot, _a1, rsz, state, prot, mtype, _a2 = struct.unpack("<QQIIQIIII", b)
        info.append((ba, rsz, mtype))
info.sort()

def region_type(va):
    lo, hi = 0, len(info) - 1
    while lo <= hi:
        mid = (lo + hi) // 2
        ba, rsz, mtype = info[mid]
        if va < ba: hi = mid - 1
        elif va >= ba + rsz: lo = mid + 1
        else: return mtype
    return None

MEM_PRIVATE, MEM_MAPPED, MEM_IMAGE = 0x20000, 0x40000, 0x1000000
TYPE_S = {MEM_PRIVATE: "PRIVATE", MEM_MAPPED: "MAPPED", MEM_IMAGE: "IMAGE", None: "?"}

# --- scan every range: PE headers at 64K alignment, plus fxgame strings ---
def is_pe(buf, off):
    if buf[off:off+2] != b"MZ": return None
    try:
        e_lfanew = struct.unpack_from("<I", buf, off + 0x3c)[0]
        if not (0x40 <= e_lfanew <= 0x400): return None
        p = off + e_lfanew
        if buf[p:p+4] != b"PE\x00\x00": return None
        machine = struct.unpack_from("<H", buf, p + 4)[0]
        nsec = struct.unpack_from("<H", buf, p + 6)[0]
        magic = struct.unpack_from("<H", buf, p + 24)[0]
        size_of_image = struct.unpack_from("<I", buf, p + 24 + 56)[0]
        if not (0 < size_of_image <= 0x40000000): return None
        return machine, nsec, magic, size_of_image
    except Exception:
        return None

pe_images = []
str_hits = []
for start, size, off0 in ranges:
    f.seek(off0)
    data = f.read(size)
    # PE headers: images are at least page-aligned; check every 0x1000 boundary
    for i in range(0, len(data) - 0x400, 0x1000):
        if data[i:i+2] != b"MZ":
            continue
        r = is_pe(data, i)
        if r:
            va = start + i
            pe_images.append((va, r))
    # string hits
    needle_a = b"fxgame"
    needle_w = "fxgame".encode("utf-16le")
    for needle, kind in ((needle_a, "ascii"), (needle_w, "utf16")):
        pos = 0
        while True:
            j = data.find(needle, pos)
            if j < 0: break
            str_hits.append((start + j, kind, data[max(0,j-40):j+60]))
            pos = j + 1
            if sum(1 for h in str_hits if h[1] == kind) > 400: break

print(f"\n--- PE images found in memory: {len(pe_images)} ---")
for va, (machine, nsec, magic, soi) in sorted(pe_images):
    listed = mod_bases.get(va)
    tag = f"module: {listed[1]}" if listed else "*** NOT IN MODULE LIST ***"
    m = {0x14c: "x86", 0x8664: "x64"}.get(machine, hex(machine))
    mg = {0x10b: "PE32", 0x20b: "PE32+"}.get(magic, hex(magic))
    print(f"  {va:>12x} {mg}/{m} secs={nsec:<3} SizeOfImage={soi:>8x} {TYPE_S.get(region_type(va))}  {tag}")

print(f"\n--- fxgame string hits: {len(str_hits)} ---")
for va, kind, ctx in str_hits[:60]:
    print(f"  {va:>12x} {kind}: {ctx!r}")
