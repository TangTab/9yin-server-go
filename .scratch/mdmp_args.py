"""Hunt for fxgame.exe launch parameters inside the dump + disk stub."""
import struct, re

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

# ---- 1. full wide context around every "fxgame.exe" wide hit ----
print("=== command-line-shaped wide strings ===")
needle = "fxgame.exe".encode("utf-16le")
seen = set()
for start, size, off0 in ranges:
    f.seek(off0)
    data = f.read(min(size, 0x800000))
    pos = 0
    while True:
        j = data.find(needle, pos)
        if j < 0: break
        pos = j + 1
        chunk = read_mem(start + j - 0x80, 0x220)
        # decode wide, printable-ish
        s = chunk.decode("utf-16le", "replace")
        # look for quote/cmdline shapes
        if '"' in s or ' -' in s or ' /' in s:
            key = s
            if key not in seen:
                seen.add(key)
                print("@%#x: %r" % (start+j-0x80, s.encode("ascii", "backslashreplace").decode()))

# ---- 2. ASCII option-token scan inside image copy + whole dump (dedup) ----
token_re = re.compile(rb"(?:^|[\x00\x01\x02\x03\n\r\t \"'])((?:-{1,2}|/)[A-Za-z][A-Za-z0-9_]{1,20})(?=[\x00=:\s\"']|$)")
kv_re = re.compile(rb"(?:^|[\x00\n\r\t \"'])([A-Za-z][A-Za-z0-9_]{1,16})=(?:\"|)([^\x00\n\r\"]{0,40})")

from collections import Counter
tok_hits = Counter()
kv_hits = Counter()
IMG_LO, IMG_HI = 0x143C0000, 0x143C0000 + 0x86E000
img_tokens, img_kv = Counter(), Counter()
for start, size, off0 in ranges:
    f.seek(off0)
    data = f.read(size)
    for m in token_re.finditer(data):
        t = m.group(1).decode()
        tok_hits[t] += 1
        if IMG_LO <= start + m.start() < IMG_HI:
            img_tokens[t] += 1
    for m in kv_re.finditer(data):
        k = m.group(1).decode(); v = m.group(2).decode("latin1")
        if any(c in v for c in "=") or len(v) > 35: continue
        kv_hits[(k, v[:30])] += 1
        if IMG_LO <= start + m.start() < IMG_HI:
            img_kv[(k, v[:30])] += 1

INTEREST = re.compile(r"(user|pass|account|acc|server|serv|ip|port|host|addr|login|token|key|auth|line|world|zone|area|mode|ver|version|channel|launch|start|update|patch|lang|region|sw|test|debug|console|log|full|window|dx|render|gpu|config|setting|game|map|scene|role|char|auto|skip|nosound|safe)", re.I)
def show(counter, label, n=60):
    print(f"\n=== {label} ===")
    rows = [(t, c) for t, c in counter.most_common() if INTEREST.search(str(t))]
    for t, c in rows[:n]:
        print(f"  {c:>5}x {t}")
show(img_tokens, "option tokens inside image copy")
show(tok_hits, "option tokens whole dump")
show(img_kv, "key=value inside image copy")
show(kv_hits, "key=value whole dump")
