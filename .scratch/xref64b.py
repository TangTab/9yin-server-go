"""RIP-relative xref scan + disasm around Start/Server/extend.ini in x64 stub."""
import struct, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

path = r"E:\Game\9yin\9yin\bin64\fxgame.exe"
d = open(path, "rb").read()
e = struct.unpack_from("<I", d, 0x3c)[0]
imgbase = struct.unpack_from("<Q", d, e+24+24)[0]
nsec = struct.unpack_from("<H", d, e+6)[0]
secs_off = e + 24 + struct.unpack_from("<H", d, e+20)[0]
sections = []
for i in range(nsec):
    nm = d[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", d, secs_off+i*40+8)
    sections.append((nm, sa, vs, rp, rs))

def off2rva(off):
    for nm, sa, vs, rp, rs in sections:
        if rp <= off < rp + rs:
            return sa + (off - rp), nm
    return None, None

# strings
tvas = {}
for s in (b"extend.ini\x00", b"Server\x00", b"Start\x00"):
    for m in re.finditer(re.escape(s), d):
        rva, sec = off2rva(m.start())
        if rva and sec == ".rdata":
            tvas[s.rstrip(b"\x00").decode()] = imgbase + rva

sa, vs, rp, rs = next(x[1:] for x in sections if x[0] == ".text")
tb = d[rp:rp+rs]
text_va0 = imgbase + sa

# rip-relative scan: any position p where (text_va0+p) + disp == target
hits = {k: [] for k in tvas}
for p in range(0, len(tb) - 4):
    disp = struct.unpack_from("<i", tb, p)[0]
    tgt = text_va0 + p + disp
    for k, va in tvas.items():
        if tgt == va:
            hits[k].append(p)
for k, v in hits.items():
    print(k, hex(tvas[k]), "->", [hex(x) for x in v])

md = Cs(CS_ARCH_X86, CS_MODE_64)
def dis(p_ref, back=0x70, fwd=0x200, title=""):
    start = max(0, p_ref - back)
    code = tb[start:start+back+fwd]
    for ins in md.disasm(code, text_va0 + start):
        tag = " <<<<" if ins.address <= text_va0 + p_ref < ins.address + ins.size else ""
        print(f"  {ins.address:#x}  {ins.mnemonic:<8} {ins.op_str}{tag}")

for k, ps in hits.items():
    for p in ps[:2]:
        print(f"\n===== {k} ref =====")
        dis(p)
