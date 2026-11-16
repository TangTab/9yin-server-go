"""pdata 函数表: python funcs.py <dll> [contains-hex] [邻域数]"""
import sys, struct
dll = sys.argv[1]
d = open(dll, 'rb').read()
pe = struct.unpack_from('<I', d, 0x3c)[0]
opt = struct.unpack_from('<H', d, pe+20)[0]
# 目录 pdata
n = struct.unpack_from('<I', d, pe+24+opt+4*3+0)  # NumberOfRvaAndSizes 后各目录; pdata 不是标准目录 — 直接按节找
# .pdata 在 0x60000 (已知)
import re
# 找 .pdata 节
nsec = struct.unpack_from('<H', d, pe+6)[0]
pd = None
for i in range(nsec):
    o = pe+24+opt+i*40
    nm = d[o:o+8].rstrip(b'\0').decode()
    vsz, va = struct.unpack_from('<II', d, o+8)
    rsz, ro = struct.unpack_from('<II', d, o+16)
    if nm == '.pdata':
        pd = (va, ro, rsz)
va, ro, rsz = pd
funcs = []
for o in range(ro, ro+rsz, 12):
    s, e, u = struct.unpack_from('<III', d, o)
    if s == 0: break
    funcs.append((s, e))
funcs.sort()
print("functions: %d" % len(funcs), file=sys.stderr)
if len(sys.argv) > 2:
    tgt = int(sys.argv[2], 0)
    idx = next((i for i, (s, e) in enumerate(funcs) if s <= tgt < e), None)
    ctx = int(sys.argv[3]) if len(sys.argv) > 3 else 4
    for i in range(max(0, idx-ctx), min(len(funcs), idx+ctx+1)):
        s, e = funcs[i]
        mark = " <<<<" if i == idx else ""
        print("func %06x..%06x (%5d B)%s" % (s, e, e-s, mark))
