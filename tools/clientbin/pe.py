"""pe.py —— x64 PE 文件（客户端 DLL）的只读逆向 CLI：.pdata 定位函数边界、穿透增量链接跳转桩、
RIP 相对引用与直接调用者、capstone 反汇编。只依赖标准库 + capstone。

  python -I pe.py <pe> info
  python -I pe.py <pe> str <正则>               搜 ASCII 串 → 所在节与 VA
  python -I pe.py <pe> refs <关键字> [-d]       含关键字的串 → 引用它的代码位置；-d 反汇编第一个引用者
  python -I pe.py <pe> xref <VA 十六进制>       代码里 RIP 相对引用该 VA 的位置
  python -I pe.py <pe> callers <VA 十六进制>    E8 直接调用者 + 经跳转桩的调用者
  python -I pe.py <pe> func <VA> [最大行数]     反汇编所在函数（.pdata 定界，缺项回退 int3 填充）
  python -I pe.py <pe> fni <VA> [最大行数]      反汇编所在函数（int3 填充定界；.pdata 被拆成碎片时用）
  python -I pe.py <pe> range <起VA> <止VA> [最大行数]
  python -I pe.py <pe> ptr <VA> [条数]          按 qword 列数据，标注是否为跳转桩及其真目标
  python -I pe.py <pe> imm <u32>                代码里出现该立即数的位置

VA 一律用完整虚拟地址（ImageBase + RVA）。反汇编输出里 `[0x…]` 是解出来的 RIP 相对目标，其后的
字符串 / qword 注释是目标处的内容；`=>` 是 call/jmp 穿透跳转桩后的真目标。
m24 子码处理表（子码 → 处理函数）的还原见同目录 subtab.py。"""
import bisect
import os
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))  # -I 模式下脚本目录不在 sys.path
sys.stdout.reconfigure(encoding='utf-8', errors='replace')      # 控制台非 UTF-8 时避免打印失败

from x64 import Image, run


class PE(Image):
    def __init__(self, path):
        self.d = open(path, 'rb').read()
        d = self.d
        pe = struct.unpack_from('<I', d, 0x3C)[0]
        nsec = struct.unpack_from('<H', d, pe + 6)[0]
        optsz = struct.unpack_from('<H', d, pe + 20)[0]
        self.base = struct.unpack_from('<Q', d, pe + 24 + 24)[0]     # PE32+ 的 ImageBase
        self.size = struct.unpack_from('<I', d, pe + 24 + 56)[0]     # SizeOfImage
        # (名, RVA, 虚拟大小, 文件偏移, 原始大小, 属性)
        self.secs = []
        so = pe + 24 + optsz
        for i in range(nsec):
            name, vsize, va, rsize, rptr = struct.unpack_from('<8sIIII', d, so + i * 40)
            chars = struct.unpack_from('<I', d, so + i * 40 + 36)[0]
            self.secs.append((name.rstrip(b'\0').decode('latin1'), va, vsize, rptr, rsize, chars))
        # 文件里有原始数据的区间 [(RVA 起, RVA 止, 文件偏移)]，供 VA ↔ 文件偏移二分
        self.raws = sorted((va, va + rsize, rptr) for _, va, _, rptr, rsize, _ in self.secs if rsize)
        self.raw_starts = [r[0] for r in self.raws]
        self.execs = [(self.base + va, rsize) for _, va, _, _, rsize, chars in self.secs
                      if chars & 0x20000000 and rsize]
        # 异常表（函数定界）。按 .pdata 节解析（与数据目录口径不一致）；会混入越界的垃圾条目，
        # 按「起 < 止 且长度 < 0x40000」过滤。
        off, psz = next(((rptr, rsize) for name, _, _, rptr, rsize, _ in self.secs if name == '.pdata'), (0, 0))
        self.pdata = sorted(x for x in struct.iter_unpack('<III', d[off:off + psz - psz % 12])
                            if x[0] < x[1] and x[1] - x[0] < 0x40000) if off and psz else []
        self.pdata_begins = [x[0] for x in self.pdata]

    def rva2off(self, rva):
        i = bisect.bisect_right(self.raw_starts, rva) - 1
        lo, hi, ptr = self.raws[i] if i >= 0 else (0, 0, 0)
        return ptr + rva - lo if i >= 0 and rva < hi else None

    def read(self, va, n):
        rva, end = va - self.base, va - self.base + n
        out = bytearray()
        while rva < end:
            o = self.rva2off(rva)
            if o is None:
                break       # 落在只存在于内存的区段（虚拟大小 > 原始大小）或节间空洞
            take = min(end - rva, self.raws[bisect.bisect_right(self.raw_starts, rva) - 1][1] - rva)
            out += self.d[o:o + take]
            rva += take
        return bytes(out)

    def where(self, va):
        rva = va - self.base
        return next((name for name, sva, vsize, _, _, _ in self.secs if sva <= rva < sva + vsize), '?')

    def off2va(self, off):
        for _, va, _, rptr, rsize, _ in self.secs:
            if rptr <= off < rptr + rsize:
                return self.base + va + off - rptr
        return 0

    def func_of(self, va):
        """所在函数的 [起, 止)：.pdata 命中用 .pdata，否则按 int3 填充。"""
        rva = va - self.base
        i = bisect.bisect_right(self.pdata_begins, rva) - 1
        if 0 <= i < len(self.pdata) and self.pdata[i][0] <= rva < self.pdata[i][1]:
            return self.base + self.pdata[i][0], self.base + self.pdata[i][1]
        return self.func_by_padding(va)


def main():
    argv = sys.argv[1:]
    if len(argv) < 2:
        sys.exit(__doc__)
    pe = PE(argv[0])
    if argv[1] == 'info':
        print('base=%#x size=%#x pdata=%d' % (pe.base, pe.size, len(pe.pdata)))
        for name, va, vsize, rptr, rsize, _ in pe.secs:
            tail = '（虚拟大小 > 原始大小 %+#x：尾部运行期填充）' % (vsize - rsize) if vsize > rsize else ''
            print('  %-8s VA=%#x vsize=%#x raw=%#x..%#x%s' % (name, pe.base + va, vsize, rptr, rptr + rsize, tail))
    elif not run(pe, argv[1:]):
        sys.exit(__doc__)


if __name__ == '__main__':
    main()
