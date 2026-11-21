"""Extract printable strings from: memory image copy .rdata/.data/.text, disk stub .rdata/.data."""
import struct, re, sys

dump_path = r"E:\Workspace\project\game\adata\IczLoader.DMP"
f = open(dump_path, "rb")
h = f.read(32); ns, dr = struct.unpack("<II", h[8:16])
f.seek(dr); ents = [struct.unpack("<III", f.read(12)) for _ in range(ns)]
dsz, rva = next((d, r) for s, d, r in ents if s == 9)
f.seek(rva); nr, br = struct.unpack("<QQ", f.read(16))
ranges = []
for _ in range(nr):
    st, sz = struct.unpack("<QQ", f.read(16))
    ranges.append((st, sz, br)); br += sz
ranges.sort()

def read_mem(va, n):
    out = bytearray(); pos, rem = va, n
    while rem > 0:
        lo, hi, hit = 0, len(ranges)-1, None
        while lo <= hi:
            mid = (lo+hi)//2; s, sz, o = ranges[mid]
            if pos < s: hi = mid-1
            elif pos >= s+sz: lo = mid+1
            else: hit = (s, sz, o); break
        if not hit:
            nxt = next((s for s, sz, o in ranges if s > pos), None)
            if nxt is None or nxt >= va+n: out += b"\x00"*rem; break
            gap = min(nxt-pos, rem); out += b"\x00"*gap; pos += gap; rem -= gap; continue
        s, sz, o = hit
        take = min(sz-(pos-s), rem)
        f.seek(o+(pos-s)); out += f.read(take); pos += take; rem -= take
    return bytes(out)

STR = re.compile(rb"[\x20-\x7e]{4,}")
def strings_of(data, minlen=4):
    return [m.group().decode() for m in STR.finditer(data) if len(m.group()) >= minlen]

IMG = 0x143C0000
sections = [(".text", 0x1000, 0x6253a), (".rdata", 0x64000, 0x10d80), (".data", 0x75000, 0x4884)]
print("=== memory image copy strings ===")
for nm, sa, vs in sections:
    ss = strings_of(read_mem(IMG+sa, vs))
    print(f"--- {nm} ({len(ss)} strings) ---")
    for s in ss:
        print("   ", s)

print("\n=== disk stub strings (.rdata/.data/.rsrc) ===")
disk = open(r"E:\Game\9yin\9yin\bin64\fxgame.exe", "rb").read()
e = struct.unpack_from("<I", disk, 0x3c)[0]
secs_off = e + 24 + struct.unpack_from("<H", disk, e+20)[0]
nsec = struct.unpack_from("<H", disk, e+6)[0]
for i in range(nsec):
    nm = disk[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", disk, secs_off+i*40+8)
    if nm not in (".rdata", ".data", ".rsrc", ".text"):
        continue
    ss = strings_of(disk[rp:rp+rs])
    print(f"--- disk {nm} ({len(ss)} strings) ---")
    for s in ss:
        print("   ", s)
