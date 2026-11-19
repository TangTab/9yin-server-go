# -*- coding: utf-8 -*-
"""批量函数轨迹：kernel 槽调用 + 直接 call 符号 + 字符串实参。"""
import sys, re
sys.path.insert(0, ".")
import fdis

SLOT = {}
for ln in open("ckernel_slots.txt"):
    m = re.match(r"\[\s*(\d+)\] \+0x([0-9a-f]+) (?:[0-9a-f]+) \?(\w+)@", ln)
    if m: SLOT[int(m.group(2), 16)] = m.group(3)

def trace(rva, size, label, maxlines=60):
    print("==== %s @ %06x (size~%x)" % (label, rva, size))
    code = fdis.read(rva, size)
    insns = list(fdis.md.disasm(code, rva))
    n = 0
    for idx, ins in enumerate(insns):
        if ins.mnemonic == "call":
            if ins.op_str.startswith("0x"):
                t = int(ins.op_str, 16)
                s = fdis.sym(t)
                if s and not s[0].startswith("__RTC"):
                    print("  %06x call   %s" % (ins.address, s[0][:80]))
                    n += 1
            else:
                disp = None
                for lb in insns[max(0, idx-6):idx]:
                    m2 = re.search(r"mov\s+\w+, dword ptr \[\w+ \+ (0x[0-9a-f]+)\]", lb.mnemonic + " " + lb.op_str)
                    if m2 and int(m2.group(1), 16) in SLOT:
                        disp = int(m2.group(1), 16)
                if disp is not None:
                    # 回看最近 push 的字符串实参
                    args = []
                    for lb2 in insns[max(0, idx-12):idx]:
                        m3 = re.match(r"push (0x[0-9a-f]{9})", lb2.mnemonic + " " + lb2.op_str)
                        if m3:
                            v = int(m3.group(1), 16)
                            s2 = fdis.cstr(v - 0x10000000, 64)
                            if s2: args.append(s2)
                    print("  %06x k:%-22s %s" % (ins.address, SLOT[disp], (" ; " + ",".join(args[:3])) if args else ""))
                    n += 1
        if n >= maxlines:
            print("  ...")
            break

if __name__ == "__main__":
    import json
    idx = {m["class"]: m for m in json.load(open("modules_index.json"))}
    FNS = [
        ("PlayerBaseModule", [("OnEntry", 0x1615d0, 0x3b0), ("OnReady", 0x161980, 0x180),
                               ("OnEntryGame", 0x161c50, 0xf0), ("OnStore", 0x161b00, 0x150),
                               ("OnDisconnect", 0x161d40, 0xa0), ("Cmd0x0C", 0x162490, 0x100)]),
        ("CreateRoleModule", [("Cmd0x375", 0x1b47a0, 0x50), ("OnCreateRole", 0x1b47f0, 0x400)]),
    ]
    for mod, fns in FNS:
        print("#" * 15, mod)
        for lab, rva, size in fns:
            trace(rva, size, lab)
