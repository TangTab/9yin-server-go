"""Verify candidate images inside the dump against on-disk files."""
import struct, sys, math

dump_path = r"E:\Workspace\project\game\adata\IczLoader.DMP"
f = open(dump_path, "rb")
hdr = f.read(32)
nstreams, dir_rva = struct.unpack("<II", hdr[8:16])
f.seek(dir_rva)
entries = [struct.unpack("<III", f.read(12)) for _ in range(nstreams)]

def stream(tp):
    return next((dsz, rva) for st, dsz, rva in entries if st == tp)

dsz, rva = stream(9)
f.seek(rva)
nranges, base_rva = struct.unpack("<QQ", f.read(16))
ranges = []
for _ in range(nranges):
    start, size = struct.unpack("<QQ", f.read(16))
    ranges.append((start, size, base_rva))
    base_rva += size
ranges.sort()

def read_mem(va, n):
    """Read n bytes at va from dump, b''
    for gaps."""
    out = bytearray()
    remaining = n
    pos = va
    while remaining > 0:
        # binary search range containing pos
        lo, hi, hit = 0, len(ranges) - 1, None
        while lo <= hi:
            mid = (lo + hi) // 2
            s, sz, o = ranges[mid]
            if pos < s: hi = mid - 1
            elif pos >= s + sz: lo = mid + 1
            else: hit = (s, sz, o); break
        if not hit:
            # skip to next range start
            nxt = next((s for s, sz, o in ranges if s > pos), None)
            if nxt is None or nxt >= va + n:
                out += b"\x00" * remaining
                break
            gap = min(nxt - pos, remaining)
            out += b"\x00" * gap
            pos += gap; remaining -= gap
            continue
        s, sz, o = hit
        take = min(sz - (pos - s), remaining)
        f.seek(o + (pos - s))
        out += f.read(take)
        pos += take; remaining -= take
    return bytes(out)

def entropy(b):
    if not b: return 0
    counts = [0]*256
    for x in b: counts[x] += 1
    n = len(b)
    return -sum((c/n)*math.log2(c/n) for c in counts if c)

def inspect(va, label):
    data = read_mem(va, 0x1000)
    if data[:2] != b"MZ":
        print(f"{label} @{va:#x}: no MZ"); return
    e = struct.unpack_from("<I", data, 0x3c)[0]
    mach, nsec, tds = struct.unpack_from("<HHI", data, e+4)
    opt = e + 24
    magic = struct.unpack_from("<H", data, opt)[0]
    soi = struct.unpack_from("<I", data, opt+56)[0]
    ep = struct.unpack_from("<I", data, opt+16)[0]
    imgbase = struct.unpack_from("<Q", data, opt+24)[0] if magic == 0x20b else struct.unpack_from("<I", data, opt+28)[0]
    print(f"\n=== {label} @ VA {va:#x} ===")
    print(f"  machine={mach:#x} nsec={nsec} timestamp={tds:#x} magic={magic:#x} SizeOfImage={soi:#x} EP={ep:#x} ImageBase={imgbase:#x}")
    secs_off = opt + struct.unpack_from("<H", data, e+20)[0]
    seclist = []
    for i in range(nsec):
        nm = data[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
        vs, sa, rs, rp = struct.unpack_from("<IIII", data, secs_off+i*40+8)
        ch = struct.unpack_from("<I", data, secs_off+i*40+36)[0]
        seclist.append((nm, sa, vs))
    # coverage & entropy per section (sample first 64KB of each section)
    for nm, sa, vs in seclist:
        mem = read_mem(va + sa, min(vs, 0x10000))
        ent = entropy(mem)
        # coverage of whole section
        got, total = 0, vs
        pos = va + sa
        while pos < va + sa + vs:
            hit = next(((s, sz) for s, sz, o in ranges if s <= pos < s + sz), None)
            if hit:
                s, sz = hit
                step = min(sz - (pos - s), va + sa + vs - pos)
                got += step; pos += step
            else:
                nxt = next((s for s, sz, o in ranges if s > pos), None)
                if nxt is None: break
                pos = nxt
        nz = sum(1 for x in mem[:0x1000] if x) / min(len(mem), 0x1000)
        print(f"    {nm:<10} VA={sa:#010x} VS={vs:#010x} mapped={got*100//total:>3}%  ent={ent:.2f} nonzero1k={nz*100:.0f}%")
    return tds, nsec, soi

print("on-disk fxgame.exe  : timestamp=0x6ad56db5 nsec=8 SizeOfImage=0x86e000")
inspect(0x143C0000, "candidate fxgame.exe")
inspect(0x0EDA0000, "candidate x64 2-sec")
inspect(0x12770000, "candidate x86 2-sec 17MB")

# on-disk fxgamelogic.dll header for comparison
d = open(r"E:\Game\9yin\9yin\bin64\fxgamelogic.dll","rb").read(0x400)
e = struct.unpack_from("<I", d, 0x3c)[0]
mach, nsec, tds = struct.unpack_from("<HHI", d, e+4)
soi = struct.unpack_from("<I", d, e+24+56)[0]
print(f"\non-disk fxgamelogic.dll: machine={mach:#x} nsec={nsec} timestamp={tds:#x} SizeOfImage={soi:#x}")
