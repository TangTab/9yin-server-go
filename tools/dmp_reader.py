#!/usr/bin/env python3
"""fxgame.DMP minidump 读取器：VA→文件偏移映射 + 模块表。

用法（库）:
  from dmp_reader import Dump
  d = Dump(r'G:/pojie/ai/tmp_carve/fxgame.DMP')
  d.modules()                  # [(name, base, size)]
  data = d.read(va, n)         # 进程虚地址读
  hits = d.scan_pattern(b'..') # 全内存扫（yield va）
"""
import struct

STREAM_TYPES = {3: 'ThreadList', 4: 'ModuleList', 5: 'MemoryList',
                6: 'Exception', 7: 'SystemInfo', 9: 'Memory64List', 16: 'MemoryInfoList'}


class Dump:
    def __init__(self, path):
        self.f = open(path, 'rb')
        hdr = self.f.read(32)
        assert hdr[:4] == b'MDMP', 'not a minidump'
        self.num_streams, self.dir_rva = struct.unpack_from('<II', hdr, 8)
        self.f.seek(self.dir_rva)
        self.streams = {}
        raw = self.f.read(self.num_streams * 12)
        for i in range(self.num_streams):
            st, size, rva = struct.unpack_from('<III', raw, i * 12)
            self.streams.setdefault(st, (size, rva))
        self._mem = None      # [(start_va, file_off, size)]
        self._mods = None

    def memory_map(self):
        """Memory64ListStream → [(start_va, file_off, size)] 升序。"""
        if self._mem is None:
            size, rva = self.streams[9]
            self.f.seek(rva)
            n, base_rva = struct.unpack('<QQ', self.f.read(16))
            self.f.seek(rva + 16)
            ranges = self.f.read(n * 16)
            self._mem = []
            off = base_rva
            for i in range(n):
                va, sz = struct.unpack_from('<QQ', ranges, i * 16)
                self._mem.append((va, off, sz))
                off += sz
        return self._mem

    def modules(self):
        """ModuleListStream → [(name, base, size)]。"""
        if self._mods is None:
            size, rva = self.streams[4]
            self.f.seek(rva)
            n, = struct.unpack('<I', self.f.read(4))
            self._mods = []
            for _ in range(n):
                ent = self.f.read(108)
                base, = struct.unpack_from('<Q', ent, 0)
                sz, = struct.unpack_from('<I', ent, 8)
                name_rva, = struct.unpack_from('<I', ent, 0x14)
                self.f.seek(name_rva)
                ln, = struct.unpack('<I', self.f.read(4))
                ln = min(ln, 1024)
                name = self.f.read(ln).decode('utf-16-le', errors='ignore').rstrip('\x00')
                self._mods.append((name, base, sz))
                self.f.seek(rva + 4 + (_ + 1) * 108)
        return self._mods

    def va_to_off(self, va):
        lo, hi = 0, len(self.memory_map()) - 1
        m = self.memory_map()
        while lo <= hi:
            mid = (lo + hi) // 2
            s, o, sz = m[mid]
            if va < s:
                hi = mid - 1
            elif va >= s + sz:
                lo = mid + 1
            else:
                return o + (va - s)
        return None

    def read(self, va, n):
        off = self.va_to_off(va)
        if off is None:
            return None
        self.f.seek(off)
        return self.f.read(n)

    def scan_pattern(self, pat, mask=None, spans=None, chunk=1 << 22):
        """全内存（或指定 spans=[(va,size)]）扫字节模式。mask 同长 b'?'=跳过。yield va。"""
        m = self.memory_map()
        for s, o, sz in m:
            if spans and not any(s < e and a < s + sz for a, e in spans):
                continue
            end = o + sz
            pos = o
            carry = b''
            carry_va = s
            while pos < end:
                self.f.seek(pos)
                buf = self.f.read(min(chunk, end - pos))
                if not buf:
                    break
                hay = carry + buf
                start = 0
                while True:
                    i = self._match(hay, pat, mask, start)
                    if i < 0:
                        break
                    yield carry_va + i - len(carry)
                    start = i + 1
                carry = buf[-(len(pat) - 1):] if len(buf) >= len(pat) else hay
                carry_va = s + (pos - o) + len(buf) - (len(carry) - (len(buf) >= len(pat) and 0)) if False else None
                # 简化：下一块起始 VA
                pos += len(buf)
                carry_va = s + (pos - o) - len(carry)
                if len(carry) >= len(pat):
                    pass
                if pos >= end:
                    # 处理尾部 carry
                    i = self._match(carry, pat, mask, 0)
                    if i >= 0:
                        yield (s + sz - len(carry)) + i
                    break

    @staticmethod
    def _match(hay, pat, mask, start):
        if mask is None:
            return hay.find(pat, start)
        n = len(pat)
        for i in range(start, len(hay) - n + 1):
            ok = True
            for j in range(n):
                if mask[j] != 0x3F and hay[i + j] != pat[j]:
                    ok = False
                    break
            if ok:
                return i
        return -1
