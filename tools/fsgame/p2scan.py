# -*- coding: utf-8 -*-
"""二期定向扫描 v2：esp 栈槽仿真，按 [esp+N] 还原 thiscall 实参。"""
import sys, re
sys.path.insert(0, ".")
import capstone
import dis2

BASE = 0x10000000
SLOTS = {
    0x0c: "AddLogicClass", 0x10: "AddClassCallback", 0x14: "AddEventCallbackInt",
    0x18: "AddEventCallback", 0x1c: "AddCommandHook", 0x20: "AddIntCommandHook",
    0x24: "AddCustomHook", 0x28: "AddIntCustomHook", 0x2c: "RunEventCallback",
    0x30: "DeclareLuaExt", 0x34: "DeclareHeartBeat", 0x38: "DeclareCritical",
    0x3c: "DeclareRecHook", 0x150: "PreloadConfig", 0x154: "PreloadConfigTxt",
    0x214: "AddRecord",
}
NARGS = {
    "AddLogicClass": 3, "AddClassCallback": 4, "AddEventCallbackInt": 5,
    "AddEventCallback": 4, "AddCommandHook": 4, "AddIntCommandHook": 4,
    "AddCustomHook": 4, "AddIntCustomHook": 4, "RunEventCallback": 4,
    "DeclareLuaExt": 2, "DeclareHeartBeat": 3, "DeclareCritical": 4,
    "DeclareRecHook": 2, "PreloadConfig": 1, "PreloadConfigTxt": 2, "AddRecord": 6,
}
# thiscall 实参（不含 this/ecx）——CKernel 槽签名
TARGS = {
    "AddLogicClass": 3, "AddClassCallback": 3, "AddEventCallbackInt": 4,
    "AddEventCallback": 3, "AddCommandHook": 3, "AddIntCommandHook": 3,
    "AddCustomHook": 3, "AddIntCustomHook": 3, "RunEventCallback": 4,
    "DeclareLuaExt": 2, "DeclareHeartBeat": 2, "DeclareCritical": 3,
    "DeclareRecHook": 2, "PreloadConfig": 1, "PreloadConfigTxt": 2, "AddRecord": 6,
}
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
IMM = capstone.x86.X86_OP_IMM

def decode(v):
    if v is None:
        return "?"
    rva = v - BASE if v >= BASE else v
    if 0x400000 <= rva < 0xd66000 and dis2.symname(rva):
        return "fn:%08x" % rva
    if 0xd67000 <= rva <= 0xe95000:
        s = dis2.cstr(rva, 120)
        if s:
            return '"%s"' % s
    if -0x10000 < v < 0x10000:
        return str(v)
    return "0x%08x" % v

def scan(rva, n=None, verbose=False):
    if n is None:
        n = dis2.func_size(rva)
    code = dis2.read(rva, n)
    insns = list(md.disasm(code, rva))
    esp = 0                      # 相对函数入口
    mem = {0: None}              # 相对 esp 槽 → 值
    regs = {}
    for idx, ins in enumerate(insns):
        op = ins.op_str
        if ins.mnemonic == "push":
            v = None
            if ins.operands[0].type == IMM:
                v = ins.operands[0].imm
            elif op in regs:
                v = regs[op]
            esp -= 4
            mem[esp] = v
            for k in list(mem):
                if k < esp - 0x40:
                    del mem[k]
        elif ins.mnemonic == "pop":
            esp += 4
            regs[op] = mem.get(esp)
        elif ins.mnemonic == "mov":
            m = re.match(r"^(e[a-d]x|esi|edi|ebx|ebp), (.+)$", op)
            if m:
                dst, src = m.groups()
                if src.isdigit() or (src.startswith("0x") or src.startswith("-")):
                    try:
                        regs[dst] = int(src, 0)
                        continue
                    except ValueError:
                        pass
                regs[dst] = None if "," in src else regs.get(src)
                continue
            m = re.match(r"^dword ptr \[esp( \+ (0x[0-9a-f]+)| - (0x[0-9a-f]+))?\], (.+)$", op)
            if m:
                off = 0
                if m.group(2):
                    off = int(m.group(2), 16)
                elif m.group(3):
                    off = -int(m.group(3), 16)
                src = m.group(4)
                if src.isdigit() or src.startswith("0x"):
                    try:
                        mem[esp + off] = int(src, 0)
                        continue
                    except ValueError:
                        pass
                mem[esp + off] = regs.get(src.strip())
        elif ins.mnemonic == "add" and op.startswith("esp, "):
            esp += int(op.split(", ")[1], 0)
        elif ins.mnemonic == "sub" and op.startswith("esp, "):
            esp -= int(op.split(", ")[1], 0)
        elif ins.mnemonic == "call":
            # 目标判定：寄存器间接 → 前置 mov reg,[reg2+slot]
            disp = None
            if op in ("eax", "ebx", "ecx", "edx", "esi", "edi"):
                for lb in insns[max(0, idx - 6):idx]:
                    if lb.mnemonic == "mov":
                        mm = re.search(r"dword ptr \[(\w+) \+ (0x[0-9a-f]+)\]", lb.op_str)
                        if mm and mm.group(1) not in ("ebp", "esp"):
                            dd = int(mm.group(2), 16)
                            if dd in SLOTS:
                                disp = dd
            if disp is not None:
                slot = SLOTS[disp]
                na = TARGS.get(slot, 4)
                args = [mem.get(esp + 4 * i) for i in range(na)]
                if verbose:
                    print("   [esp=%d memkeys=%s]" % (esp, sorted(mem)))
                print("%08x  %-18s (%s)" % (ins.address, slot, ", ".join(decode(v) for v in args)))
                esp += 4 * na  # thiscall：被调方清栈
                for k in list(mem):
                    if k < esp:
                        del mem[k]
            else:
                # 未知调用：保守 cdecl（不调整），清深度记录
                for k in list(mem):
                    if k < esp - 0x200:
                        del mem[k]

if __name__ == "__main__":
    rva = int(sys.argv[1], 16)
    n = int(sys.argv[2], 16) if len(sys.argv) > 2 else None
    verbose = len(sys.argv) > 3
    print("== %08x %s" % (rva, dis2.symname(rva) or ""))
    scan(rva, n, verbose)
