#!/usr/bin/env python3
"""从 fxgame.DMP 抠取运行时模块重建可直接 IDA 的 PE（dump 布局）。

节内容=运行时内存（VMP 已解包）；头=内存头（完好用内存的，缺则用磁盘头）；
布局=PointerToRawData:=VirtualAddress、SizeOfRawData:=VirtualSize，
ImageBase:=实际加载基址 → 重建后 VA 与 DMP 地址逐位一致。
用法: python carve_module.py <module_name> <out_path> [dmp_path]
"""
import os
import struct
import sys

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools')
from dmp_reader import Dump

DMP = r'F:/Game/Online/9yin/九阴资源/tmp_carve/fxgame.DMP'


def main(mod_name, out_path, dmp=None):
    d = Dump(dmp or os.environ.get('NINEYIN_DMP', DMP))
    mods = {n.replace('\\', '/').split('/')[-1].lower(): (n, b, s) for n, b, s in d.modules()}
    if mod_name.lower() not in mods:
        print('module not found:', mod_name)
        return
    _, base, size = mods[mod_name.lower()]
    hdr = bytearray(d.read(base, 0x400))
    if not hdr or hdr[:2] != b'MZ':
        print('no MZ in memory header')
        return
    e_lfanew = struct.unpack_from('<I', hdr, 0x3C)[0]
    assert hdr[e_lfanew:e_lfanew + 4] == b'PE\x00\x00'
    nsec = struct.unpack_from('<H', hdr, e_lfanew + 6)[0]
    opt_size = struct.unpack_from('<H', hdr, e_lfanew + 20)[0]
    machine = struct.unpack_from('<H', hdr, e_lfanew + 4)[0]
    # ImageBase(偏移 +0x18 in optional header, PE32+) := 实际加载基址
    opt = e_lfanew + 24
    struct.pack_into('<Q', hdr, opt + 0x18, base)
    # SizeOfImage 保留；Alignment 字段对 dump 布局无碍
    sec0 = opt + opt_size
    secs = []
    for i in range(nsec):
        off = sec0 + i * 40
        name, vsize, va, rsize, roff = struct.unpack_from('<8sIIII', hdr, off)
        chars = struct.unpack_from('<I', hdr, off + 36)[0]
        secs.append((name, vsize, va, off))
        # dump 布局：raw 跟 va 对齐
        struct.pack_into('<II', hdr, off + 16, (vsize + 0xFFF) & ~0xFFF if vsize else 0, va)
    # 重建文件：按最大 va+vsize 排布
    total = max(va + ((vsize + 0xFFF) & ~0xFFF) for _, vsize, va, _ in secs) if secs else 0x400
    img = bytearray(total)
    img[0:0x400] = hdr[:0x400]
    holes = 0
    for name, vsize, va, _ in secs:
        if vsize == 0:
            continue
        got = 0
        pos = 0
        while pos < vsize:
            chunk = d.read(base + va + pos, min(1 << 20, vsize - pos))
            if chunk is None:
                holes += min(1 << 20, vsize - pos)
                break
            img[va + pos:va + pos + len(chunk)] = chunk
            got += len(chunk)
            pos += len(chunk)
        print(f'{name.decode().rstrip(chr(0)):<10} va={va:#x} vsize={vsize:#x} read={got:#x}')
    print('holes(bytes):', holes, 'image size:', hex(len(img)))
    with open(out_path, 'wb') as f:
        f.write(img)
    print('written', out_path)


if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else None)
