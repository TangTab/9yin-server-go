# -*- coding: utf-8 -*-
"""扫描 FsGame.dll 全部模块 obj 函数中的 kernel vtable 槽调用，抓注册实参。
输出 modules_regs.json: {obj: {slot: [arg,...]}}
"""
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

syms, _ = parse_map()
syms_by_obj = collections.defaultdict(list)
for rva, name, obj in syms:
    syms_by_obj[obj].append((rva, name))

md = capstone.Cs(capstone.CS_ARCH_X86, capstone.CS_MODE_32)
md.detail = True
X86_OP_IMM = capstone.x86.X86_OP_IMM

def is_text(rva): return 0x1000 <= rva < 0xd66000

def printable(s): return bool(s) and all(32 <= ord(c) < 127 or c in "._/:\\" for c in s)

def scan_func(rva, size):
    res = []
    try:
        code = fdis.read(rva, min(size, 0x8000))
    except Exception:
        return res
    insns = list(md.disasm(code, rva))
    if not insns: return res
    pushes = []
    last_push_idx = -99
    for idx, ins in enumerate(insns):
        if ins.mnemonic == "push" and ins.operands and ins.operands[0].type == X86_OP_IMM:
            pushes.append(ins.operands[0].imm)
            if len(pushes) > 8: pushes.pop(0)
            last_push_idx = idx
        elif ins.mnemonic == "call" and ins.op_str in ("eax","ebx","ecx","edx","esi","edi"):
            disp = None
            for lb in insns[max(0, idx-6):idx]:
                if lb.mnemonic == "mov":
                    m = re.search(r"dword ptr \[(\w+) \+ (0x[0-9a-f]+)\]", lb.op_str)
                    if m and m.group(1) not in ("ebp", "esp"):
                        d = int(m.group(2), 16)
                        if d in SLOTS: disp = d
            if disp is not None and idx - last_push_idx <= 8:
                res.append((SLOTS[disp], list(pushes)))
    return res

def imm_str(v):
    rva = v - BASE if v >= BASE else v
    if 0xd67000 <= rva <= 0xe95000:
        s = fdis.cstr(rva, 120)
        if printable(s): return s
    return None

def imm_fn(v):
    rva = v - BASE if v >= BASE else v
    if is_text(rva):
        sym = fdis.sym(rva)
        if sym: return "%08x" % rva
    return None

modobjs = [o for o in syms_by_obj if re.search(r"(?i)(module|executor|wrap)\.obj$", o)]
out = {}
for obj in sorted(modobjs):
    agg = collections.defaultdict(set)
    fns = sorted([x for x in syms_by_obj[obj] if is_text(x[0]) and not x[1].startswith("__")])
    for i, (rva, name) in enumerate(fns):
        nxt = fns[i+1][0] if i+1 < len(fns) else rva + 0x40
        size = nxt - rva
        if size <= 0 or size > 0x8000: continue
        for slot, imms in scan_func(rva, size):
            for v in imms:
                s = imm_str(v)
                if s is not None: agg[slot].add("s:" + s); continue
                f = imm_fn(v)
                if f: agg[slot].add("f:" + f); continue
                if 0 < v <= 0xffff: agg[slot].add("i:0x%x" % v)
    if agg:
        out[obj] = {k: sorted(v) for k, v in agg.items()}

json.dump(out, open("modules_regs.json", "w"), ensure_ascii=False, indent=1)
print("objs scanned:", len(modobjs), "with regs:", len(out))
