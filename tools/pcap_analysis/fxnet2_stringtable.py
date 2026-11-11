# -*- coding: utf-8 -*-
"""fxnet2_x64.dll GameReceiver 错误串表提取：RVA 0x402b8-0x41600 → 处理器名/on_* 事件名。"""
import struct, sys
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

DLL = r'E:\Workspace\project\game\adata\official\bin\fxnet2.dll'
data = open(DLL, 'rb').read()

# PE: RVA → file offset
pe = struct.unpack_from('<I', data, 0x3c)[0]
assert data[pe:pe+4] == b'PE\x00\x00'
nsec = struct.unpack_from('<H', data, pe + 6)[0]
opt = struct.unpack_from('<H', data, pe + 20)[0]
sec_off = pe + 24 + opt
secs = []
for i in range(nsec):
    o = sec_off + i * 40
    name = data[o:o+8].rstrip(b'\x00').decode()
    vsize, vaddr, rsize, roff = struct.unpack_from('<IIII', data, o + 8)
    secs.append((name, vaddr, vsize, roff, rsize))
    print('节 %-8s va=%#x vsz=%#x raw=%#x' % (name, vaddr, vsize, roff))

def rva2off(rva):
    for name, vaddr, vsize, roff, rsize in secs:
        if vaddr <= rva < vaddr + max(vsize, rsize):
            return roff + (rva - vaddr)
    return None

lo, hi = 0x402b8, 0x41600
o1, o2 = rva2off(lo), rva2off(hi)
print('rva %#x-%#x -> file %#x-%#x' % (lo, hi, o1, o2))
blob = data[o1:o2]

# 连续可打印串提取（含 GBK 中文）
cur = bytearray()
start = 0
out = []
for i, b in enumerate(blob):
    if 32 <= b < 127 or b >= 0x80:
        if not cur:
            start = i
        cur.append(b)
    else:
        if len(cur) >= 4:
            try:
                s = bytes(cur).decode('gbk')
            except UnicodeDecodeError:
                s = bytes(cur).decode('latin1')
            out.append((lo + start, s))
        cur = bytearray()
for off, s in out:
    print('%#06x  %s' % (off, s))
