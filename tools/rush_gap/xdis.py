# -*- coding: utf-8 -*-
"""fxgamelogic.dll 32 位明文 .text 线性反汇编小工具（capstone）。

用途：按文件偏移窗口反汇编并自动注释指向 .rdata 字符串的立即数（属性名/
配置路径 xref 定位，如 GRushRange/CurSkillTarget/CanRush 族）。file offset
与 RVA 同值（.text/.rdata va==raw）。加壳段（'.).%' 与 '.^Fi'，熵 7.3+）
不可静态解——接近状态机主体在壳内，本工具只能覆盖明文带（字符串 xref/
技能表加载/轻功闸）。

用法：
  python -I xdis.py 0xc891f5 [back=0] [count=120] [--dll 路径]
  # 找字符串 xref：先用 python -c 在 dll 里 find(b'GRushRange\\x00') 得偏移，
  # 再搜 struct.pack('<I', BASE+off) 在 .text 的落点作起点。
"""
import argparse
import struct

import capstone

DEFAULT_DLL = r'E:\Workspace\project\game\adata\official\bin\fxgamelogic.dll'


def image_base(data):
    pe = struct.unpack_from('<I', data, 0x3c)[0]
    return struct.unpack_from('<I', data, pe + 24 + 28)[0]


def rdata_str(data, off, n=60):
    z = data.find(b'\x00', off)
    txt = data[off:min(z, off + n)]
    if len(txt) >= 4 and all(32 <= c < 127 for c in txt[:4]):
        return txt.decode(errors='replace')
    return None


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('off', type=lambda x: int(x, 0))
    ap.add_argument('back', nargs='?', type=int, default=0)
    ap.add_argument('count', nargs='?', type=int, default=120)
    ap.add_argument('--dll', default=DEFAULT_DLL)
    args = ap.parse_args()
    data = open(args.dll, 'rb').read()
    base = image_base(data)
    md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
    start = args.off - args.back
    emitted = 0
    for ins in md.disasm(data[start:start + args.count * 8], base + start):
        line = '%08x  %-8s %s' % (ins.address, ins.mnemonic, ins.op_str)
        for tok in ins.op_str.replace(',', ' ').split():
            if not tok.startswith('0x'):
                continue
            try:
                v = int(tok, 16)
            except ValueError:
                continue
            if 0x1082000 <= v - base < 0x123F000:
                txt = rdata_str(data, v - base)
                if txt:
                    line += '   ; "%s"' % txt
        print(line)
        emitted += 1
        if ins.address - (base + start) > args.count * 6:
            break
    if not emitted:
        print('(无有效指令——该偏移可能是数据/对齐错位，调整 back 相位重试)')


if __name__ == '__main__':
    main()
