"""Find code referencing Start/Server/extend.ini strings in bin fxgame.exe, disassemble."""
import struct, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_32

path = r"E:\Game\9yin\9yin\bin\fxgame.exe"
d = open(path, "rb").read()
e = struct.unpack_from("<I", d, 0x3c)[0]
imgbase = struct.unpack_from("<I", d, e+24+28)[0]
nsec = struct.unpack_from("<H", d, e+6)[0]
secs_off = e + 24 + struct.unpack_from("<H", d, e+20)[0]
sections = []
for i in range(nsec):
    nm = d[secs_off+i*40:secs_off+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", d, secs_off+i*40+8)
    sections.append((nm, sa, vs, rp, rs))
print("imagebase", hex(imgbase))
for nm, sa, vs, rp, rs in sections:
    print(f"  {nm:<8} VA={imgbase+sa:#x} VS={vs:#x} raw={rp:#x} RS={rs:#x}")

def rva2off(rva):
    for nm, sa, vs, rp, rs in sections:
        if sa <= rva < sa + max(vs, rs):
            return rp + (rva - sa)
    return None

def off2rva(off):
    for nm, sa, vs, rp, rs in sections:
        if rp <= off < rp + rs:
            return sa + (off - rp)
    return None

# locate strings in whole file
targets = {}
for s in (b"extend.ini\x00", b"Server\x00", b"Start\x00"):
    for m in re.finditer(re.escape(s), d):
        rva = off2rva(m.start())
        if rva:
            targets.setdefault(s.rstrip(b"\x00").decode(), []).append(imgbase + rva)
for k, v in targets.items():
    print(k, [hex(x) for x in v])

# scan .text (plain) for imm32 refs
text = next((sa, vs, rp, rs) for nm, sa, vs, rp, rs in sections if nm == ".text")
tblob = d[rp:rp+rs]
refs = {}
for tname, addrs in targets.items():
    for a in addrs:
        pat = struct.pack("<I", a)
        for m in re.finditer(re.escape(pat), tblob):
            rva = sa + m.start()  # note: ref may be at offset-1 for 0xB8 style opcodes etc.
            refs.setdefault(tname, []).append(m.start())
for tname, offs in refs.items():
    print(tname, "raw .text offsets:", [hex(o) for o in offs])

md = Cs(CS_ARCH_X86, CS_MODE_32)
md.detail = False

def disasm_around(t_off, back=64, fwd=320, label=""):
    start = max(0, t_off - back)
    code = tblob[start:start+fwd+back]
    rva0 = sa + start
    print(f"\n----- {label} (.text off {t_off:#x}, VA {imgbase+sa+t_off:#x}) -----")
    for ins in md.disasm(code, imgbase + rva0):
        marker = " <<<" if ins.address <= imgbase+sa+t_off < ins.address + ins.size else ""
        print(f"  {ins.address:#010x}  {ins.mnemonic:<8} {ins.op_str}{marker}")
        if ins.address > imgbase + sa + t_off + fwd:
            break

for tname, offs in refs.items():
    for o in offs[:2]:
        disasm_around(o, label=tname)
