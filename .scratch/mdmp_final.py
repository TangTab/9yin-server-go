"""Final forensics: PEB cmdline, image-vs-disk byte compare, private RWX regions."""
import struct

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

# ---- 1. main thread TEB -> PEB -> ProcessParameters ----
dsz, rva = stream(3)
f.seek(rva)
nthreads = struct.unpack("<I", f.read(4))[0]
# MINIDUMP_THREAD: ThreadId(4) SuspendCount(4) PriorityClass(4) Priority(4) Teb(8) Stack(16) Context(8) = 48
tebs = []
for _ in range(nthreads):
    b = f.read(48)
    tid = struct.unpack_from("<I", b, 0)[0]
    teb = struct.unpack_from("<Q", b, 16)[0]
    tebs.append((tid, teb))
print(f"threads={nthreads}")
main_tid, main_teb = tebs[0]
peb = struct.unpack("<I", read_mem(main_teb + 0x30, 4))[0]
print(f"main thread tid={main_tid:#x} teb={main_teb:#x} peb={peb:#x}")
pp = struct.unpack("<I", read_mem(peb + 0x10, 4))[0]
# 32-bit RTL_USER_PROCESS_PARAMETERS: ImagePathName @0x38, CommandLine @0x40
for name, off in (("ImagePathName", 0x38), ("CommandLine", 0x40)):
    ln, buf = struct.unpack("<II", read_mem(pp + off, 8))
    s = read_mem(buf, ln).decode("utf-16le", "replace")
    print(f"  {name}: {s}")
# PEB->ImageBase / Ldr hint
ib = struct.unpack("<I", read_mem(peb + 8, 4))[0]
print(f"  PEB.ImageBase = {ib:#x}")

# ---- 2. compare image sections with on-disk file ----
disk = open(r"E:\Game\9yin\9yin\bin64\fxgame.exe", "rb").read()
e = struct.unpack_from("<I", disk, 0x3c)[0]
secs_off = e + 24 + struct.unpack_from("<H", disk, e+20)[0]
nsec = struct.unpack_from("<H", disk, e+6)[0]
IMG = 0x143C0000
print("\nsection compare (memory @0x143C0000 vs disk):")
for i in range(nsec):
    nm = disk[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii","replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", disk, secs_off+i*40+8)
    n = min(rs, 0x4000)
    mem = read_mem(IMG + sa, n)
    dsk = disk[rp:rp+n]
    same = sum(1 for a, b in zip(mem, dsk) if a == b)
    print(f"  {nm:<8} first{len(dsk):#6x}: identical={same*100//max(len(dsk),1):>3}%")

# ---- 3. big PRIVATE regions and their protection ----
dsz, rva = stream(16)
f.seek(rva)
szh, sze, nent = struct.unpack("<III", f.read(12))
regions = []
for _ in range(nent):
    b = f.read(sze)
    ba, aba, aprot, _a1, rsz, state, prot, mtype, _a2 = struct.unpack("<QQIIQIIII", b)
    regions.append((ba, rsz, state, prot, mtype))
P = {0x10:"NOACCESS",0x01:"R",0x02:"RW",0x04:"RWX",0x20:"R+X",0x40:"RWX",0x08:"WC",0x02|0x100:"RW(guard)",0x10|0x100:"NOACCESS(g)"}
def ps(p):
    return P.get(p, hex(p))
print("\ntop PRIVATE committed regions by size:")
priv = sorted([r for r in regions if r[4] == 0x20000 and r[2] == 0x1000], key=lambda r: -r[1])
for ba, rsz, state, prot, mtype in priv[:20]:
    print(f"  {ba:>12x} size={rsz:>10x} ({rsz/1024/1024:8.1f} MiB) prot={ps(prot)}")
tot = sum(r[1] for r in priv)
print(f"  total private committed: {tot/1024/1024:.0f} MiB")
