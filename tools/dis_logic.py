#!/usr/bin/env python3
"""fxgamelogic_rebuilt.dll 注解反汇编：rip 相对引用自动解出字符串。
用法: python dis_logic.py <va-hex> [条数] [尾va-hex]
"""
import re
import sys

import capstone

IMG = open(r'G:/pojie/ai/tmp_carve/fxgamelogic_rebuilt.dll', 'rb').read()
BASE = 0x3AF0000
END = BASE + len(IMG)
MD = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_64)


def s_at(va, n=64):
    off = va - BASE
    if off < 0 or off >= len(IMG):
        return None
    b = IMG[off:off + n].split(b'\x00')[0]
    if len(b) >= 2 and all(32 <= c < 127 for c in b):
        return b.decode()
    return None


def dis(va, count=120, stop_va=None):
    off = va - BASE
    code = IMG[off:off + count * 8 + 64]
    n = 0
    for ins in MD.disasm(code, va):
        note = ''
        m = re.search(r'rip ([+-]) (0x[0-9a-f]+)', ins.op_str)
        if m:
            disp = int(m.group(2), 16) * (1 if m.group(1) == '+' else -1)
            tgt = ins.address + ins.size + disp
            s = s_at(tgt)
            note = f'  ; {tgt:#x}' + (f' "{s}"' if s else '')
        if ins.mnemonic in ('call', 'jmp') and ins.op_str.startswith('0x'):
            note += f'  ; -> {ins.op_str}'
        print(f'{ins.address:08x}  {ins.mnemonic:<8}{ins.op_str}{note}')
        n += 1
        if stop_va and ins.address >= stop_va:
            break
        if n >= count:
            break


if __name__ == '__main__':
    va = int(sys.argv[1], 16)
    cnt = int(sys.argv[2]) if len(sys.argv) > 2 else 120
    dis(va, cnt)
