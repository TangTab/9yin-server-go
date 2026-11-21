"""扫描 rip 相对引用: python xref.py <dll> <target-hex>...  (文件偏移==RVA)"""
import sys, struct, re
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

dll = sys.argv[1]
targets = {int(t, 0) for t in sys.argv[2:]}
d = open(dll, 'rb').read()
pe_off = struct.unpack_from('<I', d, 0x3c)[0]
nsec = struct.unpack_from('<H', d, pe_off + 6)[0]
opt_size = struct.unpack_from('<H', d, pe_off + 20)[0]
text = None
for i in range(nsec):
    o = pe_off + 24 + opt_size + i * 40
    name = d[o:o+8].rstrip(b'\0').decode()
    vsz, va = struct.unpack_from('<II', d, o + 8)
    rsz, ro = struct.unpack_from('<II', d, o + 16)
    print("sec %-8s va=%06x vsz=%06x raw=%06x rsz=%06x" % (name, va, vsz, ro, rsz), file=sys.stderr)
    if name == '.text':
        text = (va, ro, rsz)
va, ro, rsz = text
md = Cs(CS_ARCH_X86, CS_MODE_64)
count = 0
for ins in md.disasm(d[ro:ro+rsz], va):
    op = ins.op_str
    if 'rip' in op:
        m = re.search(r'rip ([+-]) (0x[0-9a-f]+)', op)
        if m:
            disp = int(m.group(2), 16) * (1 if m.group(1) == '+' else -1)
            tgt = ins.address + ins.size + disp
            if tgt in targets:
                print("xref @%06x: %s %s  -> %06x" % (ins.address, ins.mnemonic, op, tgt))
                count += 1
print("# %d xrefs" % count, file=sys.stderr)
