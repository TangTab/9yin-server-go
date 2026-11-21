"""Find parent-process check: search dump image for FxUpdater strings + disasm refs."""
import struct, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

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
        lo, hi, hit = 0, len(ranges) - 1, None
        while lo <= hi:
            mid = (lo + hi) // 2; s, sz, o = ranges[mid]
            if pos < s: hi = mid - 1
            elif pos >= s + sz: lo = mid + 1
            else: hit = (s, sz, o); break
        if not hit:
            nxt = next((s for s, sz, o in ranges if s > pos), None)
            if nxt is None or nxt >= va + n: out += b"\x00" * rem; break
            gap = min(nxt - pos, rem); out += b"\x00" * gap; pos += gap; rem -= gap; continue
        s, sz, o = hit
        take = min(sz - (pos - s), rem)
        f.seek(o + (pos - s)); out += f.read(take); pos += take; rem -= take
    return bytes(out)

IMG = 0x143C0000          # where the copy sits in IczLoader
ORIG = 0x140000000        # original ImageBase (pointers/instr use this)
# section VAs from PE header
hdr = read_mem(IMG, 0x1000)
e = struct.unpack_from("<I", hdr, 0x3c)[0]
nsec = struct.unpack_from("<H", hdr, e + 6)[0]
secs_off = e + 24 + struct.unpack_from("<H", hdr, e + 20)[0]
sections = []
for i in range(nsec):
    nm = hdr[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", hdr, secs_off+i*40+8)
    sections.append((nm, sa, vs))
print("sections:", sections)

def orig2copy(rva):
    return IMG + rva

# 1. find FxUpdater/handshake strings in the image copy (rdata/data/text)
print("\n=== strings inside dump image copy ===")
found = {}
for nm, sa, vs in sections:
    blob = read_mem(IMG + sa, vs)
    for pat in (b"FxUpdater", b"FxUpdate", b"fxupdate", b"Please use", b"updater", b"Parent", b"parent", b"ppid", b"NtQueryInformationProcess", b"ProcessBasicInformation", b"CreateMutex", b"FxLaunch", b"Mutex"):
        for m in re.finditer(re.escape(pat), blob):
            va = ORIG + sa + m.start()
            ctx = blob[m.start():m.start()+90].split(b"\x00")[0]
            found.setdefault(pat, []).append((va, ctx))
for pat, lst in found.items():
    for va, ctx in lst[:4]:
        print(f"  {pat.decode():<28} @ {va:#x}: {ctx[:80]!r}")

# 2. disasm .text refs to 'Please use FxUpdater' string address (rip-relative, original addresses)
text_sa, text_vs = next((sa, vs) for nm, sa, vs in sections if nm == ".text")
tb = read_mem(IMG + text_sa, text_vs)
text_va0 = ORIG + text_sa
md = Cs(CS_ARCH_X86, CS_MODE_64)

def rip_refs(target):
    out = []
    for p in range(len(tb) - 4):
        disp = struct.unpack_from("<i", tb, p)[0]
        if text_va0 + p + disp == target:
            out.append(p)
    return out

for pat, lst in found.items():
    if pat not in (b"Please use", b"FxUpdate\x00"): continue
    for va, ctx in lst[:2]:
        refs = rip_refs(va)
        print(f"\n===== refs to {ctx[:40]!r} @ {va:#x}: {[hex(r) for r in refs]} =====")
        for r in refs[:2]:
            start = max(0, r - 0x120)
            code = tb[start:start+0x380]
            for ins in md.disasm(code, text_va0 + start):
                tag = " <<<<" if ins.address <= text_va0 + r < ins.address + ins.size else ""
                print(f"  {ins.address:#x}  {ins.mnemonic:<8} {ins.op_str}{tag}")
