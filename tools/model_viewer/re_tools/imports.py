"""枚举导入: python imports.py <dll>"""
import struct, sys
d = open(sys.argv[1], 'rb').read()
pe = struct.unpack_from('<I', d, 0x3c)[0]
opt = struct.unpack_from('<H', d, pe+20)[0]
secs = []
for i in range(struct.unpack_from('<H', d, pe+6)[0]):
    o = pe+24+opt+i*40
    nm = d[o:o+8].rstrip(b'\0').decode()
    vsz, va = struct.unpack_from('<II', d, o+8)
    rsz, ro = struct.unpack_from('<II', d, o+16)
    secs.append((nm, va, vsz, ro, rsz))
def rva2off(rva):
    for nm, va, vsz, ro, rsz in secs:
        if va <= rva < va + max(vsz, rsz):
            return ro + (rva - va)
    return None
imp_rva = struct.unpack_from('<I', d, pe+24+112+8)[0]
io = rva2off(imp_rva)
thunks = {}
while True:
    oft, ts, fc, nameRva, fthunk = struct.unpack_from('<IIIII', d, io)
    if nameRva == 0: break
    dll = d[rva2off(nameRva):].split(b'\0')[0].decode(errors='replace')
    t = rva2off(oft or fthunk)
    idx = 0
    while True:
        val = struct.unpack_from('<Q', d, t + idx*8)[0]
        if val == 0: break
        if val & (1<<63):
            nm2 = "ord#%d" % (val & 0xffff)
        else:
            o2 = rva2off(val & 0x7fffffff)
            nm2 = d[o2+2:].split(b'\0')[0].decode(errors='replace') if o2 else "?"
        thunks[fthunk + idx*8] = dll + "!" + nm2
        idx += 1
    io += 20
print("# IAT %d 项" % len(thunks), file=sys.stderr)
if len(sys.argv) > 2:
    for t in sys.argv[2:]:
        a = int(t, 0)
        print("IAT %06x = %s" % (a, thunks.get(a, "?")))
else:
    for a, nm in sorted(thunks.items()):
        print("IAT %06x %s" % (a, nm))
