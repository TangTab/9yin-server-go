"""Is in-memory .text decrypted real code? + fix PEB cmdline & region parse."""
import struct, math

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
    out = bytearray(); pos, remaining = va, n
    while remaining > 0:
        lo, hi, hit = 0, len(ranges) - 1, None
        while lo <= hi:
            mid = (lo + hi) // 2
            s, sz, o = ranges[mid]
            if pos < s: hi = mid - 1
            elif pos >= s + sz: lo = mid + 1
            else: hit = (s, sz, o); break
        if not hit:
            nxt = next((s for s, sz, o in ranges if s > pos), None)
            if nxt is None or nxt >= va + n:
                out += b"\x00" * remaining; break
            gap = min(nxt - pos, remaining)
            out += b"\x00" * gap; pos += gap; remaining -= gap; continue
        s, sz, o = hit
        take = min(sz - (pos - s), remaining)
        f.seek(o + (pos - s)); out += f.read(take)
        pos += take; remaining -= take
    return bytes(out)

def entropy(b):
    if not b: return 0
    c = [0]*256
    for x in b: c[x] += 1
    n = len(b)
    return -sum((k/n)*math.log2(k/n) for k in c if k)

# ---- disk vs memory entropy per section ----
disk = open(r"E:\Game\9yin\9yin\bin64\fxgame.exe", "rb").read()
e = struct.unpack_from("<I", disk, 0x3c)[0]
secs_off = e + 24 + struct.unpack_from("<H", disk, e+20)[0]
nsec = struct.unpack_from("<H", disk, e+6)[0]
IMG = 0x143C0000
print("sec      disk_ent  mem_ent   mem_head(32B hex)")
for i in range(nsec):
    nm = disk[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii","replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", disk, secs_off+i*40+8)
    dchunk = disk[rp:rp+0x10000]
    mchunk = read_mem(IMG + sa, min(vs, 0x10000))
    print(f"{nm:<8} {entropy(dchunk):7.2f}  {entropy(mchunk):7.2f}   {mchunk[:32].hex(' ')}")

# x64 code sanity: int3 padding runs + common prologues in memory .text
text = read_mem(IMG + 0x1000, 0x62000)
cc = text.count(b"\xcc\xcc\xcc\xcc")
prologues = sum(text.count(p) for p in (b"\x48\x89\x5c\x24", b"\x48\x83\xec", b"\x55\x48\x8b\xec", b"\x40\x53", b"\x48\x8b\xec"))
print(f"\nmem .text: len={len(text):#x} int3runs={cc} prologue_hits={prologiques if False else prologues}")
print(f"disk .text: len={min(0x62000, len(disk)):#x}")

# ---- command line: dump wide string around 0x7c60c32 ----
def dump_wide(va, back, fwd):
    b = read_mem(va - back, back + fwd)
    try:
        return b.decode("utf-16le", "replace")
    except Exception:
        return repr(b)
print("\nwide context @0x7c60be8:", repr(dump_wide(0x7c60be8, 32, 700)))

# ---- all threads -> find a valid PEB ----
dsz, rva = stream(3)
f.seek(rva)
nthreads = struct.unpack("<I", f.read(4))[0]
for t in range(nthreads):
    b = f.read(48)
    tid = struct.unpack_from("<I", b, 0)[0]
    teb = struct.unpack_from("<Q", b, 16)[0]
    peb = struct.unpack("<I", read_mem(teb + 0x30, 4))[0]
    if 0x10000 < peb < 0x7fff0000:
        pp = struct.unpack("<I", read_mem(peb + 0x10, 4))[0]
        if 0x10000 < pp < 0x7fff0000:
            ln, buf = struct.unpack("<II", read_mem(pp + 0x40, 8))
            cl = read_mem(buf, ln).decode("utf-16le", "replace")
            ln2, buf2 = struct.unpack("<II", read_mem(pp + 0x38, 8))
            ip = read_mem(buf2, ln2).decode("utf-16le", "replace")
            print(f"\nthread {tid:#x}: PEB={peb:#x}")
            print(f"  ImagePathName: {ip}")
            print(f"  CommandLine:   {cl}")
            break

# ---- debug memory info parse ----
dsz, rva = stream(16)
f.seek(rva)
szh, sze, nent = struct.unpack("<III", f.read(12))
print(f"\nmeminfo: SizeOfHeader={szh} SizeOfEntry={sze} entries={nent}")
b = f.read(sze)
ba, aba, aprot, a1, rsz, state, prot, mtype, a2 = struct.unpack("<QQIIQIIII", b)
print(f"first entry: ba={ba:#x} rsz={rsz:#x} state={state:#x} prot={prot:#x} type={mtype:#x}")
