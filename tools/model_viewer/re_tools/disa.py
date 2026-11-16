"""反汇编 fx*rebuilt.dll (文件偏移==RVA), 用法: python dis.py <dll> <start> [end] [count]"""
import sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_64
dll, start = sys.argv[1], int(sys.argv[2], 0)
end = int(sys.argv[3], 0) if len(sys.argv) > 3 else start + 256
d = open(dll, 'rb').read()
md = Cs(CS_ARCH_X86, CS_MODE_64)
md.detail = False
code = d[start:end]
for ins in md.disasm(code, start):
    print("0x%08x  %-24s %s %s" % (ins.address, ins.bytes.hex(), ins.mnemonic, ins.op_str))
