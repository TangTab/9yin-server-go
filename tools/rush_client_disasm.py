#!/usr/bin/env python3
"""冲锋自走客户端侧反推：fxnet2.dll 0x21 AllDest handler 消费链反汇编。

用法:
  python rush_client_disasm.py <rva-hex> [max_bytes]     # 反汇编指定 RVA
  python rush_client_disasm.py --chain <rva-hex>         # 跟一层 call 链
DLL: G:/pojie/ai/tmp_carve/fxnet2.dll（半壳，.text 可读）
"""
import sys

import capstone
import pefile

DLL = r"G:/pojie/ai/tmp_carve/fxnet2.dll"
BASE = 0x10600000

_pe = pefile.PE(DLL, fast_load=True)
_md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_64)
_md.detail = True


def rva_to_off(rva: int) -> int:
    for sec in _pe.sections:
        if sec.VirtualAddress <= rva < sec.VirtualAddress + max(sec.Misc_VirtualSize, sec.SizeOfRawData):
            return sec.PointerToRawData + (rva - sec.VirtualAddress)
    raise ValueError(f"rva {rva:#x} not in any section")


def read(rva: int, n: int) -> bytes:
    off = rva_to_off(rva)
    with open(DLL, "rb") as f:
        f.seek(off)
        return f.read(n)


def disasm(rva: int, max_bytes: int = 400, depth: int = 0, seen: set | None = None) -> None:
    if seen is None:
        seen = set()
    code = read(rva, max_bytes)
    pad = "  " * depth
    print(f"{pad}==== sub_{BASE + rva:x} (rva {rva:#x}) ====")
    for ins in _md.disasm(code, BASE + rva):
        line = f"{pad}{ins.address:08x}  {ins.mnemonic:<8}{ins.op_str}"
        # 标注 rip 相对读的常量（LEA/MOV 立即数等）
        for op in ins.operands:
            if op.type == capstone.x86.X86_OP_IMM and ins.mnemonic in ("call", "jmp"):
                line += f"   ; -> sub_{op.imm:x} (rva {op.imm - BASE:#x})"
        print(line)
        if ins.mnemonic in ("ret", "jmp") and ins.op_str.startswith("0x"):
            break
        # 跟随直接 call 一层
        if depth < 1 and ins.mnemonic == "call" and ins.op_str.startswith("0x"):
            tgt = int(ins.op_str, 16)
            if BASE < tgt < BASE + 0x200000 and tgt not in seen:
                seen.add(tgt)
                disasm(tg - BASE if (tg := tgt) else 0, 160, depth + 1, seen)


if __name__ == "__main__":
    rva = int(sys.argv[1], 16)
    n = int(sys.argv[2]) if len(sys.argv) > 2 else 400
    disasm(rva, n)
