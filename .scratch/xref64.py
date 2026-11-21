"""Cross-ref Start/Server/extend.ini in BOTH builds; disassemble x64 stub refs."""
import struct, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_32, CS_MODE_64

def load(path):
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
    return d, imgbase, sections

def off2rva(sections, off):
    for nm, sa, vs, rp, rs in sections:
        if rp <= off < rp + rs:
            return sa + (off - rp), nm
    return None, None

for path in (r"E:\Game\9yin\9yin\bin\fxgame.exe", r"E:\Game\9yin\9yin\bin64\fxgame.exe"):
    d, imgbase, sections = load(path)
    is64 = imgbase > 0xffffffff or path.endswith("bin64\\fxgame.exe")
    print(f"\n########## {path} imgbase={imgbase:#x}")
    # find strings anywhere in file
    for s in (b"extend.ini\x00", b"Server\x00", b"Start\x00"):
        for m in re.finditer(re.escape(s), d):
            rva, sec = off2rva(sections, m.start())
            if rva is None: continue
            va = imgbase + rva
            enc = struct.pack("<Q" if is64 else "<I", va)
            hits = [i for i in range(len(d)) if d[i:i+len(enc)] == enc]
            # count refs per section
            refs = []
            for h in hits:
                r2, s2 = off2rva(sections, h)
                refs.append(s2 or "?")
            print(f"  {s.rstrip(chr(0).encode()).decode():<12} VA={va:#x} sec={sec} refs={refs}")

# ---- disassemble x64 stub around refs ----
path = r"E:\Game\9yin\9yin\bin64\fxgame.exe"
d, imgbase, sections = load(path)
sa, vs, rp, rs = next((sa, vs, rp, rs) for nm, sa, vs, rp, rs in sections if nm == ".text")
tblob = d[rp:rp+rs]
md = Cs(CS_ARCH_X86, CS_MODE_64)

def find_refs(va):
    enc = struct.pack("<I", va & 0xffffffff)
    out = []
    for m in re.finditer(re.escape(enc), tblob):
        # lea reg, [rip+disp32]: pattern at m requires preceding 8-byte instr; just note offset
        out.append(m.start())
    return out

# locate string VAs in rdata
target_vas = {}
for s in (b"extend.ini\x00", b"Server\x00", b"Start\x00"):
    for m in re.finditer(re.escape(s), d):
        rva, sec = off2rva(sections, m.start())
        if rva and sec in (".rdata", ".data"):
            target_vas[s.rstrip(chr(0).encode()).decode()] = imgbase + rva
print("\nx64 string VAs:", {k: hex(v) for k, v in target_vas.items()})

for name, va in target_vas.items():
    offs = find_refs(va)
    print(f"\n===== {name} @ {va:#x}: {len(offs)} raw dword matches in .text =====")
    for o in offs[:3]:
        # disasm window ending near o (lea patterns: opcode + disp32 ends at match)
        start = max(0, o - 0x90)
        code = tblob[start:start+0x220]
        base = imgbase + sa + start
        for ins in md.disasm(code, base):
            tag = " <<<<" if ins.address + ins.size >= imgbase + sa + o >= ins.address else ""
            print(f"  {ins.address:#x}  {ins.mnemonic:<8} {ins.op_str}{tag}")
