"""mempe.py —— 进程内存镜像（模块按基址整段落盘的 .bin，用 mdmp.py save 导出）的 x64 分析 CLI，
命令与 pe.py 对齐。内存里 .pdata 不可靠，函数定界一律用 int3 填充（func 与 fni 等价）。

  python -I mempe.py <img.bin> <基址十六进制> info
  python -I mempe.py <img.bin> <基址> str <正则>               搜 ASCII 串 → 所在节与 VA
  python -I mempe.py <img.bin> <基址> refs <关键字> [-d]
  python -I mempe.py <img.bin> <基址> xref <VA>
  python -I mempe.py <img.bin> <基址> callers <VA>
  python -I mempe.py <img.bin> <基址> func <VA> [最大行数]
  python -I mempe.py <img.bin> <基址> range <起VA> <止VA> [最大行数]
  python -I mempe.py <img.bin> <基址> ptr <VA> [条数]
  python -I mempe.py <img.bin> <基址> imm <u32>
  python -I mempe.py <img.bin> <基址> sub <十六进制字节串>[,字节串…]   原始字节子串搜索（不限可打印边界）

外服客户端的镜像里 .text 起始处仍是增量链接跳转桩表，逻辑被 VMProtect 搬进 .vmp0/.vmp1，
这些节同样算可执行区间。"""
import os
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式下脚本目录不在 sys.path
sys.stdout.reconfigure(encoding='utf-8', errors='replace')      # 控制台非 UTF-8 时避免打印失败

from x64 import Image, run


class MemPE(Image):
    def __init__(self, path, base):
        self.d = open(path, 'rb').read()
        d = self.d
        self.base, self.size = base, len(d)
        pe = struct.unpack_from('<I', d, 0x3C)[0]
        nsec = struct.unpack_from('<H', d, pe + 6)[0]
        optsz = struct.unpack_from('<H', d, pe + 20)[0]
        # (名, RVA, 虚拟大小, 属性)；虚拟大小截到镜像实际长度
        self.secs = []
        so = pe + 24 + optsz
        for i in range(nsec):
            name, vsize, va, rsize, rptr = struct.unpack_from('<8sIIII', d, so + i * 40)
            chars = struct.unpack_from('<I', d, so + i * 40 + 36)[0]
            self.secs.append((name.rstrip(b'\0').decode('latin1'), va, min(vsize, len(d) - va), chars))
        # 一切可执行节都算代码：VMProtect 的 .vmp0/.vmp1 也是代码（变异 / 虚拟化后的函数体）
        self.execs = [(base + va, vsize) for _, va, vsize, chars in self.secs if chars & 0x20000000 and vsize > 0]

    def read(self, va, n):
        o = va - self.base
        return self.d[o:o + n] if 0 <= o < len(self.d) else b''

    def where(self, va):
        rva = va - self.base
        return next((name for name, sva, vsize, _ in self.secs if sva <= rva < sva + vsize), '?')

    def off2va(self, off):
        return self.base + off


def main():
    argv = sys.argv[1:]
    if len(argv) < 3:
        sys.exit(__doc__)
    img = MemPE(argv[0], int(argv[1], 16))
    if argv[2] == 'info':
        print('base=%#x size=%#x' % (img.base, img.size))
        for name, va, vsize, chars in img.secs:
            print('  %-8s VA=%#x vsize=%#x chars=%#010x' % (name, img.base + va, vsize, chars))
    elif argv[2] == 'sub':
        for part in argv[3].split(','):
            key = bytes.fromhex(part)
            hits, i = [], img.d.find(key)
            while i >= 0:
                hits.append(i)
                i = img.d.find(key, i + 1)
            print('%s: %s' % (key.hex(), ', '.join('%#x(%s)' % (img.base + h, img.where(img.base + h))
                                                   for h in hits[:14]) or '无'))
    elif not run(img, argv[2:]):
        sys.exit(__doc__)


if __name__ == '__main__':
    main()
