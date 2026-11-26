# x64.py —— x64 代码镜像的通用分析层（磁盘 PE 与进程内存镜像共用；CLI 入口见 pe.py / mempe.py）。
# Image 的子类提供 base / size / read(va, n) / where(va) / off2va(文件偏移) / execs（可执行区间
# [(VA, 长度)]），这里实现：int3 填充定界、增量链接跳转桩穿透、RIP 相对引用、E8/E9 调用者、
# 立即数定位、capstone 反汇编，以及两个 CLI 共用的命令循环 run()。
# 只依赖标准库 + capstone（pip install capstone）。
import re
import struct
import sys

from capstone import Cs, CS_ARCH_X86, CS_MODE_64

# [REX] opcode modrm(mod=00, rm=101)：lea / mov / cmp / add / sub 的 RIP 相对形式
RIPREF = re.compile(rb'[\x40-\x4f]?[\x8d\x8b\x3b\x39\x03\x2b\x89\x83\x63][\x05\x0d\x15\x1d\x25\x2d\x35\x3d]', re.S)
RIPABS = re.compile(r'\[rip ([+-]) (0x[0-9a-f]+)\]')
STR = re.compile(rb'[\x20-\x7e]{4,}')
PAD = b'\xcc' * 12     # MSVC 把函数间填 int3 到 16 字节边界，>= 12 个连续 CC 即函数边界


class Image:
    base = 0
    size = 0
    execs = ()

    def read(self, va, n):
        """读 n 字节；落出镜像或在节间空洞处截断。"""
        raise NotImplementedError

    def where(self, va):
        """VA 所在节名；不在任何节返回 '?'。"""
        raise NotImplementedError

    def off2va(self, off):
        """镜像文件偏移 → VA；不在任何节返回 0。"""
        raise NotImplementedError

    def in_image(self, va):
        return self.base <= va < self.base + self.size

    def strings(self):
        """[(VA, 串)] 镜像里全部 >= 4 字节的 ASCII 串（带缓存）。"""
        cache = getattr(self, '_strings', None)
        if cache is None:
            cache = self._strings = [(self.off2va(m.start()), m.group(0)) for m in STR.finditer(self.d)]
        return cache

    def code(self):
        """[(VA, 字节)] 各可执行区间（带缓存）。"""
        cache = getattr(self, '_code', None)
        if cache is None:
            cache = self._code = [(va, self.read(va, n)) for va, n in self.execs]
        return cache

    def exec_of(self, va):
        """含 va 的可执行区间 (起, 止)；不在任何可执行区间返回 None。"""
        return next((b for b in ((base, base + n) for base, n in self.execs) if b[0] <= va < b[1]), None)

    def thunk(self, va):
        """增量链接跳转桩：E9 rel32 → 目标；非桩返回 None。"""
        raw = self.read(va, 5)
        if len(raw) == 5 and raw[0] == 0xE9:
            return va + 5 + struct.unpack_from('<i', raw, 1)[0]
        return None

    def resolve(self, va):
        """沿跳转桩链走到真函数（最多 4 跳）。"""
        for _ in range(4):
            t = self.thunk(va)
            if not t or t == va:
                break
            va = t
        return va

    def padding_window(self, va, span=0x8000):
        """int3 定界的搜索窗口 (下界, 上界)：va 两侧各 span 字节，不越出所在可执行区间。"""
        lo, hi = self.exec_of(va) or (va - span, va + span)
        return max(lo, va - span), min(hi, va + span)

    def func_by_padding(self, va, span=0x8000):
        """所在函数的 [起, 止)：向上找 >= 12 个连续 CC 之后的第一个字节，向下找 >= 12 个连续 CC。
        .pdata 缺项 / 被拆成碎片 / 内存镜像里不可靠时用。某一侧顶到搜索窗口边界（见 padding_window）
        即那一侧没找到填充。"""
        lo, hi = self.padding_window(va, span)
        back = self.read(lo, va - lo)
        i = len(back)
        while i >= 12 and back[i - 12:i] != PAD:
            i -= 1
        fwd = self.read(va, hi - va)
        j = fwd.find(PAD)
        return lo + i, va + (j if j >= 0 else len(fwd))

    def func_of(self, va):
        """所在函数的 [起, 止)；默认按 int3 填充，磁盘 PE 子类优先用 .pdata。"""
        return self.func_by_padding(va)

    def xref(self, va):
        """RIP 相对引用 va 的位置（disp32 之后的地址）。原始字节粗扫，会有少量伪命中。"""
        out = []
        for base, blob in self.code():
            for m in RIPREF.finditer(blob):
                o = m.end()
                if o + 4 <= len(blob) and base + o + 4 + struct.unpack_from('<i', blob, o)[0] == va:
                    out.append(base + o + 4)
        return sorted(out)

    def rel32(self, va, opcode):
        """E8 / E9 rel32 以 va 为目标的位置（opcode 所在地址）；原始字节粗扫。"""
        out = []
        for base, blob in self.code():
            for m in re.finditer(re.escape(bytes([opcode])), blob):
                o = m.start()
                if o + 5 <= len(blob) and base + o + 5 + struct.unpack_from('<i', blob, o + 1)[0] == va:
                    out.append(base + o)
        return sorted(out)

    def callers(self, va):
        """[(调用点, 经由的跳转桩或 None)]：E8 直接调用者 + 经跳转到 va 的桩的调用者。"""
        out = [(a, None) for a in self.rel32(va, 0xE8)]
        for th in self.rel32(va, 0xE9):
            out += [(a, th) for a in self.rel32(th, 0xE8)]
        return sorted(out)

    def imm(self, value):
        """代码里出现该 u32 立即数的位置。"""
        pat = re.escape(struct.pack('<I', value & 0xFFFFFFFF))
        out = []
        for base, blob in self.code():
            out += [base + m.start() for m in re.finditer(pat, blob)]
        return sorted(out)

    def cstr(self, va, n=100):
        """va 处的 ASCII C 串（不足 4 个可见字符或含非可见字节返回空）。"""
        raw = self.read(va, n)
        e = raw.find(b'\0')
        s = raw[:e if e >= 0 else n]
        return s if len(s) >= 4 and all(32 <= c < 127 for c in s) else b''

    def disasm(self, begin, end, maxins=4000):
        """反汇编 [begin, end) → 文本行。`[0x…]` 是解出来的 RIP 相对目标（后注其处的串 / qword），
        `=>` 是 call/jmp 穿透跳转桩后的真目标。"""
        md = Cs(CS_ARCH_X86, CS_MODE_64)
        rows = []
        for ins in md.disasm(self.read(begin, end - begin), begin):
            op, note = ins.op_str, ''
            m = RIPABS.search(op)
            if m:
                t = ins.address + ins.size + (int(m.group(2), 16) if m.group(1) == '+' else -int(m.group(2), 16))
                op = op.replace(m.group(0), '[0x%x]' % t)
                s = self.cstr(t)
                if s:
                    note = '  ; "%s"' % s.decode('latin1')
                else:
                    raw8 = self.read(t, 8)
                    if len(raw8) == 8:
                        q = struct.unpack('<Q', raw8)[0]
                        if self.in_image(q):
                            note = '  ; qword=%#x' % q + (' -> %#x' % self.resolve(q) if self.thunk(q) else '')
            if ins.mnemonic in ('call', 'jmp') and op.startswith('0x'):
                tgt = int(op, 16)
                real = self.resolve(tgt)
                note += '  ; => %#x%s' % (real, ' (thunk)' if real != tgt else '')
            rows.append('%012x  %-7s %s%s' % (ins.address, ins.mnemonic, op, note))
            if len(rows) >= maxins:
                rows.append('…')
                break
        return rows

    def function_rows(self, va, begin, end, maxins=4000):
        """函数反汇编文本（含首行范围说明）。va 是定位用的地址（None = 不检查填充边界）。"""
        note = ''
        if va is not None:
            lo, hi = self.padding_window(va)
            lost = [side for side, hit in (('起', begin == lo), ('止', end == hi)) if hit]
            if lost:
                note = '；%s 侧未找到 int3 填充边界（入口可能在节首，或函数被虚拟化）' % '、'.join(lost)
        return ['; function %#x .. %#x (%d bytes)%s' % (begin, end, end - begin, note)] + self.disasm(begin, end, maxins)


def run(img, args):
    """两个 CLI 共用的命令循环（info 由各自入口打印）；命令不被识别返回 False。"""
    cmd, a = args[0], args[1:]
    need = {'range': 2}.get(cmd, 1)      # 除 range 外每个命令至少 1 个参数
    if len(a) < need:
        sys.exit('%s 命令缺参数（见 pe.py / mempe.py 文件头的用法）' % cmd)
    lines = []
    if cmd == 'str':
        pat = re.compile(a[0].encode(), re.I)
        lines = ['VA=%#x %-8s %s' % (va, img.where(va), s.decode('latin1')[:170])
                 for va, s in img.strings() if va and pat.search(s)]
    elif cmd == 'refs':
        pat = re.compile(a[0].encode(), re.I)
        first = None
        for va, s in img.strings():
            if not va or not pat.search(s):
                continue
            for ref in img.xref(va):
                b, e = img.func_of(ref)
                lines.append('== %s VA=%#x  ref@%#x (func %#x..%#x)' % (s[:80].decode('latin1'), va, ref, b, e))
                first = first or (ref, va)
        if '-d' in a and first:
            b, e = img.func_of(first[0])
            lines += img.function_rows(first[0], b, e)
    elif cmd == 'xref':
        lines = ['ref@%#x  (func %#x-%#x, %s)' % (ref, *img.func_of(ref), img.where(ref))
                 for ref in img.xref(int(a[0], 16))]
    elif cmd == 'callers':
        for ref, via in img.callers(int(a[0], 16)):
            b, e = img.func_of(ref)
            lines.append('call@%#x  (in func %#x-%#x)%s' % (ref, b, e, '  via thunk %#x' % via if via else ''))
    elif cmd in ('func', 'fni'):
        va = int(a[0], 16)
        b, e = img.func_of(va) if cmd == 'func' else img.func_by_padding(va)
        lines = img.function_rows(va, b, e, int(a[1]) if len(a) > 1 else 4000)
    elif cmd == 'range':
        lines = img.function_rows(None, int(a[0], 16), int(a[1], 16), int(a[2]) if len(a) > 2 else 4000)
    elif cmd == 'ptr':
        va, n = int(a[0], 16), int(a[1]) if len(a) > 1 else 16
        if len(img.read(va, 8)) != 8:
            lines = ['%#x: 镜像里没有原始数据（运行期填充的 .data 区，或地址不在镜像内）' % va]
        for i in range(n):
            raw8 = img.read(va + 8 * i, 8)
            if len(raw8) != 8:
                break
            q = struct.unpack('<Q', raw8)[0]
            real = img.resolve(q) if img.in_image(q) else 0
            lines.append('%#x: %#018x%s' % (va + 8 * i, q, ' -> %#x (%s)' % (real, img.where(real)) if real else ''))
    elif cmd == 'imm':
        lines = ['imm@%#x  (func %#x-%#x)' % (hit, *img.func_of(hit)) for hit in img.imm(int(a[0]))]
    else:
        return False
    print('\n'.join(lines))
    return True
