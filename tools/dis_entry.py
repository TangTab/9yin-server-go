# -*- coding: utf-8 -*-
"""Fs 系 DLL/EXE 脱机反汇编：map 符号 + 字符串注释（capstone x86）。

对 BGserverCJ 全符号快照（F:\\Game\\Online\\9yin\\BGserverCJ\\Debug）任意
模块可用；imagebase 从 PE 头读，.map 用 MSVC linker map（FsSoloEntry.map 等）。

用法：
  python dis_entry.py --dll X.dll --map X.map syms <regex>     符号普查（粗解 C++ 名）
  python dis_entry.py --dll X.dll --map X.map str <regex>      字符串普查（.rdata/.data，ASCII+UTF16）
  python dis_entry.py --dll X.dll --map X.map hex <hexbytes>   字节模式定位（如 DES 键）
  python dis_entry.py --dll X.dll --map X.map xref <va>        E8 调用点普查（含 ILT thunk 一跳解析）
  python dis_entry.py --dll X.dll --map X.map dis <va> [n]     反汇编注释

不传 --dll/--map 时默认 FsSoloEntry.dll（登录接入服）。坑（BGserverCJ 实测）：
.textbss 无 raw 数据须跳过；0x10043xxx~0x10045xxx = ILT thunk 表，call 目标
须二次 jmp 解析（xref 模式已内置）。
"""
import argparse
import bisect
import re
import sys

from capstone import Cs, CS_ARCH_X86, CS_MODE_32

DEBUG_DIR = r"F:\Game\Online\9yin\BGserverCJ\Debug"
DEFAULT_DLL = DEBUG_DIR + r"\FsSoloEntry.dll"
DEFAULT_MAP = DEBUG_DIR + r"\FsSoloEntry.map"


def load(dll_path, map_path):
    d = open(dll_path, "rb").read()
    pe = int.from_bytes(d[0x3c:0x40], "little")
    nsec = int.from_bytes(d[pe+6:pe+8], "little")
    optsz = int.from_bytes(d[pe+20:pe+22], "little")
    imagebase = int.from_bytes(d[pe+24+28:pe+24+32], "little")
    secs = []
    for i in range(nsec):
        o = pe + 24 + optsz + i * 40
        name = d[o:o+8].rstrip(b"\0").decode()
        va = int.from_bytes(d[o+12:o+16], "little")
        raw = int.from_bytes(d[o+20:o+24], "little")
        vsz = int.from_bytes(d[o+8:o+12], "little")
        secs.append((name, imagebase + va, vsz, raw))
    syms = {}
    for line in open(map_path, errors="ignore"):
        m = re.match(r"\s*\d+:[0-9a-fA-F]+\s+(\S+)\s+([0-9a-fA-F]{8})\s+f", line)
        if m:
            syms[int(m.group(2), 16)] = m.group(1)
    addrs = sorted(syms)
    return d, secs, syms, addrs, imagebase


def va2off(secs, va):
    for name, sva, vsz, raw in secs:
        if raw and sva <= va < sva + max(vsz, raw):
            return raw + (va - sva), name
    return None, None


def demangle(n):
    m = re.match(r"\?(\w+)@(\w+)@@", n)
    if m:
        return f"{m.group(2)}::{m.group(1)}"
    m = re.match(r"\?\?0(\w+)@@.*", n)
    if m:
        return f"{m.group(1)}::ctor"
    m = re.match(r"\?\?1(\w+)@@.*", n)
    if m:
        return f"{m.group(1)}::dtor"
    m = re.match(r"\?(\w+)@(\w+)@(\w+)@@", n)
    if m:
        return f"{m.group(3)}::{m.group(2)}::{m.group(1)}"
    return n


def build_strings(d, secs):
    strs = {}
    for name, sva, vsz, raw in secs:
        if name not in (".rdata", ".data") or not raw:
            continue
        blob = d[raw:raw+vsz]
        for m in re.finditer(rb"[\x20-\x7e]{4,}", blob):
            strs[sva+m.start()] = m.group().decode("latin1")
        for m in re.finditer(rb"(?:[\x20-\x7e]\x00){4,}", blob):
            strs.setdefault(sva+m.start(), m.group().decode("utf-16le"))
    return strs


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dll", default=DEFAULT_DLL, help="目标 DLL/EXE 路径")
    ap.add_argument("--map", default=DEFAULT_MAP, help="MSVC linker .map 路径")
    ap.add_argument("mode", choices=["syms", "str", "hex", "xref", "dis"])
    ap.add_argument("args", nargs="+")
    ns = ap.parse_args()

    d, secs, syms, addrs, imagebase = load(ns.dll, ns.map)
    print(f"# {ns.dll} imagebase={imagebase:#x} syms={len(syms)}", file=sys.stderr)
    strs = build_strings(d, secs)

    def sym_at(va):
        return demangle(syms[va]) if va in syms else None

    def near(va):
        i = bisect.bisect_right(addrs, va) - 1
        return demangle(syms[addrs[i]]) if i >= 0 else "?"

    if ns.mode == "syms":
        pat = re.compile(ns.args[0], re.I)
        for a in addrs:
            nm = demangle(syms[a])
            if pat.search(nm):
                print(f"{a:08x}  {nm}")
        return

    if ns.mode == "str":
        pat = re.compile(ns.args[0], re.I)
        for va in sorted(strs):
            if pat.search(strs[va]):
                print(f"{va:08x}  {strs[va][:160]}")
        return

    if ns.mode == "hex":
        needle = bytes.fromhex(ns.args[0].replace(" ", ""))
        start = 0
        while True:
            i = d.find(needle, start)
            if i < 0:
                break
            for name, sva, vsz, raw in secs:
                if raw and raw <= i < raw + max(vsz, raw):
                    va = sva + (i - raw)
                    print(f"file {i:#x} -> va {va:#x} ({name}) near {near(va)}")
            start = i + 1
        return

    if ns.mode == "xref":
        # 全部代码段 E8 rel32 call 调用点（目标经 ILT thunk 一跳解析）
        target = int(ns.args[0], 16)
        thunk = {}
        for name, sva, vsz, raw in secs:
            if not raw:
                continue
            blob = d[raw:raw+vsz]
            for i in range(len(blob) - 5):
                if blob[i] == 0xE9:
                    rel = int.from_bytes(blob[i+1:i+5], "little", signed=True)
                    thunk[sva + i] = sva + i + 5 + rel
        for name, sva, vsz, raw in secs:
            if ".text" not in name or not raw:
                continue
            blob = d[raw:raw+vsz]
            for i in range(len(blob) - 5):
                if blob[i] == 0xE8:
                    rel = int.from_bytes(blob[i+1:i+5], "little", signed=True)
                    va = sva + i
                    dst = thunk.get(va + 5 + rel, va + 5 + rel)
                    if dst == target:
                        print(f"call @ {va:08x}  in {near(va)}")
        return

    if ns.mode == "dis":
        start = int(ns.args[0], 16)
        count = int(ns.args[1]) if len(ns.args) > 1 else 400
        off, sec = va2off(secs, start)
        if off is None:
            print(f"!! va {start:#x} unmapped")
            return
        md = Cs(CS_ARCH_X86, CS_MODE_32)
        print(f"==== {near(start)} @ {start:#x} ({sec}) ====")
        code = d[off:off+0x2000]
        n = 0
        for ins in md.disasm(code, start):
            line = f"{ins.address:08x}  {ins.mnemonic:<8} {ins.op_str}"
            note = []
            for tok in re.findall(r"0x[0-9a-f]{6,8}", ins.op_str):
                t = int(tok, 16)
                s = sym_at(t)
                if s:
                    note.append(f"-> {s}")
                elif t in strs:
                    note.append(f'-> "{strs[t][:70]}"')
            for tok in re.findall(r"(?:dword ptr )?\[?0x([0-9a-f]{6,8})\]?", ins.op_str):
                t = int(tok, 16)
                if t in strs:
                    note.append(f'[{t:#x}]="{strs[t][:70]}"')
            if note:
                line += "   ; " + "  ".join(dict.fromkeys(note))
            print(line)
            n += 1
            if ins.mnemonic == "int3":
                break
            if n >= count:
                break
            if n > 8 and ins.address in syms and ins.address != start:
                break


if __name__ == "__main__":
    main()
