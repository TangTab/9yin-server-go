# -*- coding: utf-8 -*-
import struct
d = open(r"F:\Game\Online\9yin\BGserverCJ\Debug\FsGame.dll","rb").read()
pe = struct.unpack_from("<I", d, 0x3C)[0]
opt = pe + 24
nsec = struct.unpack_from("<H", d, pe+6)[0]
sh = pe + 24 + struct.unpack_from("<H", d, pe+20)[0]
secs = []
for i in range(nsec):
    o = sh + i*40
    name = d[o:o+8].rstrip(b"\0").decode()
    vs, va, rs, ro = struct.unpack_from("<IIII", d, o+8)
    secs.append((name, va, vs, ro))
def r2o(rva):
    for name, va, vs, ro in secs:
        if va <= rva < va + max(vs, 0x1000):
            return rva - va + ro
    return None
exp_rva = 0xe94730
o = r2o(exp_rva)
chars, ts, mj, mn, name_rva, base, nf, nn, afuncs, anames, aord = struct.unpack_from("<IIHHIIIIIII", d, o)
no = r2o(name_rva); print("dllname:", d[no:d.index(b"\0",no)].decode(), "funcs:", nf, "names:", nn)
for i in range(nn):
    (nrva,) = struct.unpack_from("<I", d, r2o(anames)+4*i)
    no2 = r2o(nrva); nm = d[no2:d.index(b"\0",no2)].decode()
    (frva,) = struct.unpack_from("<I", d, r2o(afuncs)+4*i)
    print("EXPORT:", nm, "rva", hex(frva))
imp_rva = 0xe93aa4
o = r2o(imp_rva)
while True:
    oft, ts2, fc, dll_rva, ft = struct.unpack_from("<IIIII", d, o)
    if dll_rva == 0: break
    no = r2o(dll_rva); dll = d[no:d.index(b"\0",no)].decode()
    fo = r2o(oft or ft); fns = []
    ii = 0
    while True:
        (th,) = struct.unpack_from("<I", d, fo+4*ii)
        if th == 0: break
        if not (th & 0x80000000):
            ho = r2o(th); fo2 = ho+2
            fns.append(d[fo2:d.index(b"\0",fo2)].decode())
        ii += 1
    print("IMPORT:", dll, len(fns), "fns")
    for f in fns: print("   ", f)
    o += 20
