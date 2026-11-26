"""mdmp.py —— Windows minidump（.DMP 全内存转储）只读访问：按需 seek，不整载。纯标准库。

  python -I mdmp.py <dmp> info                            流 / 模块 / 内存区间概览
  python -I mdmp.py <dmp> find <模块名子串> <ASCII串>       在某模块镜像内搜 ASCII 串 → VA
  python -I mdmp.py <dmp> read <VA 十六进制> <字节数>      读内存（跨区间，空洞补零）
  python -I mdmp.py <dmp> save <模块名子串> <out.bin>      把模块镜像按基址整段落盘（供 mempe.py / subtab.py）
  python -I mdmp.py <dmp> gfind <关键字>…                  全 dump 搜字节串 → 命中数与所属模块

gfind 的关键字带 `hex:` 前缀按十六进制字节串，否则按 ASCII；每个关键字最多列 40 处。
模块镜像落盘后，VA = 基址 + 文件偏移，可直接交给 mempe.py。"""
import bisect
import mmap
import os
import struct
import sys

sys.stdout.reconfigure(encoding='utf-8', errors='replace')      # 控制台非 UTF-8 时避免打印失败

STREAM_NAMES = {0: 'Unused', 3: 'ThreadList', 4: 'ModuleList', 5: 'MemoryList', 6: 'Exception', 7: 'SystemInfo',
                8: 'ThreadExList', 9: 'Memory64List', 10: 'CommentA', 11: 'CommentW', 12: 'HandleData',
                13: 'FunctionTable', 14: 'UnloadedModuleList', 15: 'MiscInfo', 16: 'MemoryInfoList',
                17: 'ThreadInfoList', 18: 'HandleOperationList', 21: 'TokenVp', 22: 'SystemMemoryInfo'}
MAX_SHOW = 40


class Dmp:
    def __init__(self, path):
        self.f = open(path, 'rb')
        hdr = self.f.read(32)
        if struct.unpack_from('<I', hdr)[0] != 0x504D444D:       # 'MDMP'
            raise ValueError('not a minidump: %s' % path)
        self.nstreams, dir_rva = struct.unpack_from('<II', hdr, 8)
        self.flags = struct.unpack_from('<Q', hdr, 24)[0]
        self.f.seek(dir_rva)
        self.entries = [struct.unpack('<III', self.f.read(12)) for _ in range(self.nstreams)]   # (类型, 大小, RVA)
        self.modules = self._modules()       # [(基址, 大小, 校验和, 时间戳, 路径)] 按基址升序
        self.ranges = self._ranges()         # [(起始 VA, 长度, 文件偏移)] 按起始 VA 升序
        self.starts = [r[0] for r in self.ranges]

    def _mdstr(self, rva):
        """MINIDUMP_STRING（UTF-16LE，长度按字节计）。"""
        if not rva:
            return ''
        self.f.seek(rva)
        n = struct.unpack('<I', self.f.read(4))[0]
        return self.f.read(n).decode('utf-16le', 'replace')

    def _stream(self, kind):
        return next((e for e in self.entries if e[0] == kind), None)

    def _modules(self):
        e = self._stream(4)
        if not e:
            return []
        self.f.seek(e[2])
        out = []
        for _ in range(struct.unpack('<I', self.f.read(4))[0]):
            pos = self.f.tell()
            base, size, csum, tds, name_rva = struct.unpack_from('<QIIII', self.f.read(108))   # MINIDUMP_MODULE = 108 字节
            out.append((base, size, csum, tds, self._mdstr(name_rva)))
            self.f.seek(pos + 108)           # _mdstr 会 seek，循环内要回到记录流
        return sorted(out)

    def _ranges(self):
        """全内存转储用 Memory64ListStream（区间数据首尾相接自 BaseRva 起）；普通转储用 MemoryListStream。"""
        e = self._stream(9)
        out = []
        if e:
            self.f.seek(e[2])
            n, base_rva = struct.unpack('<QQ', self.f.read(16))
            for _ in range(n):
                start, size = struct.unpack('<QQ', self.f.read(16))
                out.append((start, size, base_rva))
                base_rva += size
        else:
            e = self._stream(5)
            if e:
                self.f.seek(e[2])
                for _ in range(struct.unpack('<I', self.f.read(4))[0]):
                    start, size, rva = struct.unpack('<QII', self.f.read(16))
                    out.append((start, size, rva))
        return sorted(out)

    def read(self, va, n):
        """跨区间读 n 字节；不在任何内存区间的空洞补零。"""
        out, pos, rem = bytearray(), va, n
        while rem > 0:
            i = bisect.bisect_right(self.starts, pos) - 1
            if i >= 0 and pos < self.ranges[i][0] + self.ranges[i][1]:
                s, sz, off = self.ranges[i]
                take = min(sz - (pos - s), rem)
                self.f.seek(off + (pos - s))
                out += self.f.read(take)
                pos, rem = pos + take, rem - take
                continue
            nxt = self.starts[i + 1] if i + 1 < len(self.starts) else None
            gap = rem if nxt is None or nxt - pos > rem else nxt - pos
            out += bytes(gap)
            pos, rem = pos + gap, rem - gap
        return bytes(out)

    def module(self, sub):
        """名字含子串的第一个模块（不分大小写）；没有返回 None。"""
        return next((m for m in self.modules if sub.lower() in m[4].lower()), None)

    def module_image(self, sub):
        """(模块, 镜像字节)：镜像按模块大小读出，缺页补零。"""
        m = self.module(sub)
        if not m:
            raise KeyError('module not found: %s' % sub)
        return m, self.read(m[0], m[1])

    def search(self, lo, hi, pat):
        """在 VA 区间 [lo, hi) 内逐内存区间搜字节串 → [VA]。"""
        out = []
        for s, sz, off in self.ranges:
            a, b = max(s, lo), min(s + sz, hi)
            if a >= b:
                continue
            self.f.seek(off + (a - s))
            data = self.f.read(b - a)
            i = data.find(pat)
            while i >= 0:
                out.append(a + i)
                i = data.find(pat, i + 1)
        return out

    def file_off_to_va(self, off):
        """Memory64 文件偏移 → VA（二分找区间；不在任何区间返回 None）。"""
        lo, hi = 0, len(self.ranges)
        while lo < hi:
            mid = (lo + hi) // 2
            s, sz, o = self.ranges[mid]
            if off < o:
                hi = mid
            elif off >= o + sz:
                lo = mid + 1
            else:
                return s + (off - o)
        return None

    def owner(self, va):
        """VA 所在模块的短名；不在模块里返回 '(非模块内存)'。"""
        m = next((m for m in self.modules if m[0] <= va < m[0] + m[1]), None)
        return '%s+%#x' % (m[4].replace('/', '\\').split('\\')[-1], va - m[0]) if m else '(非模块内存)'


def gfind(d, keys):
    """全 dump 搜字节串：分窗 mmap + find（Windows 上单次 mmap 有 2GB 上限）。"""
    size = os.fstat(d.f.fileno()).st_size
    win, overlap = 1 << 30, 64
    for key in keys:
        total, shown = 0, []
        for off in range(0, size, win):
            ln = min(win + overlap, size - off)
            with mmap.mmap(d.f.fileno(), ln, offset=off, access=mmap.ACCESS_READ) as mm:
                i = mm.find(key)
                while i >= 0:
                    if i < win or off + win >= size:          # 重叠区里的重复命中只记一次
                        total += 1
                        if len(shown) < MAX_SHOW:
                            shown.append(d.file_off_to_va(off + i))
                    i = mm.find(key, i + 1)
        text = key.decode('ascii') if all(32 <= c < 127 for c in key) else key.hex()
        print('== %s 共 %d 处%s' % (text, total, '（仅列前 %d 处）' % MAX_SHOW if total > MAX_SHOW else ''))
        for va in shown:
            print('   VA=%s  %s' % ('%#x' % va if va is not None else '?', d.owner(va) if va is not None else '(非内存区间)'))


def main():
    argv = sys.argv[1:]
    if len(argv) < 2:
        sys.exit(__doc__)
    d = Dmp(argv[0])
    cmd, a = argv[1], argv[2:]
    if cmd == 'info':
        print('flags=%#x streams=%d' % (d.flags, d.nstreams))
        for st, dsz, rva in d.entries:
            print('  stream %-2d %-20s size=%#x rva=%#x' % (st, STREAM_NAMES.get(st, '?'), dsz, rva))
        print('memory ranges=%d total=%.1f MiB' % (len(d.ranges), sum(r[1] for r in d.ranges) / 1048576))
        print('--- modules (%d) ---' % len(d.modules))
        for base, size, csum, tds, name in d.modules:
            print('  %012x size=%#010x ts=%08x cs=%08x %s' % (base, size, tds, csum, name))
    elif cmd == 'find':
        m = d.module(a[0])
        if not m:
            sys.exit('module not found: %s' % a[0])
        for va in d.search(m[0], m[0] + m[1], a[1].encode()):
            print('VA=%#x (rva %#x)' % (va, va - m[0]))
    elif cmd == 'read':
        va, b = int(a[0], 16), d.read(int(a[0], 16), int(a[1]))
        for o in range(0, len(b), 32):
            print('%012x: %s' % (va + o, b[o:o + 32].hex(' ')))
    elif cmd == 'save':
        m, image = d.module_image(a[0])
        with open(a[1], 'wb') as f:
            f.write(image)
        print('saved %d bytes of %s @%#x -> %s' % (len(image), m[4], m[0], a[1]))
    elif cmd == 'gfind':
        keys = [bytes.fromhex(k[4:]) if k.startswith('hex:') else k.encode() for k in a]
        gfind(d, keys)
    else:
        sys.exit(__doc__)


if __name__ == '__main__':
    main()
