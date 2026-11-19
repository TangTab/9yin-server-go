# -*- coding: utf-8 -*-
"""调用级提取：每个注册调用的完整实参元组 -> regs_calls.json"""
import sys, re, json, collections
sys.path.insert(0, ".")
import fdis
import capstone
from maparse import parse_map

BASE = 0x10000000
SLOTS = {
    0x0c: "AddLogicClass", 0x10: "AddClassCallback", 0x14: "AddEventCallbackInt",
    0x18: "AddEventCallback", 0x1c: "AddCommandHook", 0x20: "AddIntCommandHook",
    0x24: "AddCustomHook", 0x28: "AddIntCustomHook", 0x2c: "RunEventCallback",
    0x30: "DeclareLuaExt", 0x34: "DeclareHeartBeat", 0x38: "DeclareCritical",
    0x3c: "DeclareRecHook", 0x150: "PreloadConfig", 0x154: "PreloadConfigTxt",
}
NARGS = {  # 每槽的实参个数（thiscall: this 在 ecx 不计）
    "AddLogicClass": 3, "AddClassCallback": 4, "AddEventCallbackInt": 5,
    "AddEventCallback": 4, "AddCommandHook": 4, "AddIntCommandHook": 4,
    "AddCustomHook": 4, "AddIntCustomHook": 4, "RunEventCallback": 4,
    "DeclareLuaExt": 2, "DeclareHeartBeat": 3, "DeclareCritical": 4,
    "DeclareRecHook": 2, "PreloadConfig": 1, "PreloadConfigTxt": 2,
}
syms, _ = parse_map()
syms_by_obj = collections.defaultdict(list)
for rva, name, obj in syms:
    syms_by_obj[obj].append((rva, name))
md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
IMM = capstone.x86.X86_OP_IMM
def is_text(r): return 0x1000 <= r < 0xd66000
def printable(s): return bool(s) and all(32 <= ord(c) < 127 or c in "._/:\\" for c in s)

def decode(v):
    rva = v - BASE if v >= BASE else v
    if 0xd67000 <= rva <= 0xe95000:
        s = fdis.cstr(rva, 120)
        if printable(s): return s
    if is_text(rva): return "fn:%08x" % rva
    if 0 < v <= 0xffff: return v
    return None

def scan_func(rva, size):
    res = []
    try: code = fdis.read(rva, min(size, 0x8000))
    except Exception: return res
    insns = list(md.disasm(code, rva))
    pushes = []  # (idx, imm)
    for idx, ins in enumerate(insns):
        if ins.mnemonic == "push" and ins.operands and ins.operands[0].type == IMM:
            pushes.append((idx, ins.operands[0].imm))
        elif ins.mnemonic == "call" and ins.op_str in ("eax","ebx","ecx","edx","esi","edi"):
            disp = None
            for lb in insns[max(0, idx-6):idx]:
                if lb.mnemonic == "mov":
                    m = re.search(r"dword ptr \[(\w+) \+ (0x[0-9a-f]+)\]", lb.op_str)
                    if m and m.group(1) not in ("ebp","esp"):
                        dd = int(m.group(2), 16)
                        if dd in SLOTS: disp = dd
            if disp is not None:
                slot = SLOTS[disp]
                # 取最近 NARGS 个 push（逆序=实参序）
                args = [decode(v) for _, v in pushes[-NARGS[slot]:]]
                if any(a is not None and not isinstance(a, int) for a in args):
                    res.append((slot, rva, args))
    return res

modobjs = [o for o in syms_by_obj if re.search(r"(?i)(module|executor)\.obj$", o)]
calls = []
for obj in sorted(modobjs):
    fns = sorted([x for x in syms_by_obj[obj] if is_text(x[0]) and not x[1].startswith("__")])
    for i, (rva, name) in enumerate(fns):
        nxt = fns[i+1][0] if i+1 < len(fns) else rva + 0x40
        size = nxt - rva
        if size <= 0 or size > 0x8000: continue
        for slot, site, args in scan_func(rva, size):
            calls.append({"obj": obj, "slot": slot, "site": "%08x" % site, "args": args})
json.dump(calls, open("regs_calls.json","w"), ensure_ascii=False, indent=0)
print("calls:", len(calls))
