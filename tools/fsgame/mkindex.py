# -*- coding: utf-8 -*-
"""合成 241 模块完整索引: 名称串/creator ctor/Name()/Init/obj/符号数/注册面统计/源码路径。"""
import sys, re, json, collections, struct
sys.path.insert(0, ".")
import fdis
from maparse import parse_map

BASE = 0x10000000
syms, _ = parse_map()
by_name = {}
for rva, name, obj in syms:
    by_name.setdefault(name, rva)

creators = []
for line in open("creators.txt"):
    rva, cls = line.strip().split("\t")
    creators.append((int(rva, 16), cls))

# obj 符号计数
objcount = collections.Counter()
objinit = {}
for rva, name, obj in syms:
    objcount[obj] += 1

# Name() 反汇编: mov eax, imm(字符串VA)
def creator_name(cls):
    nm = by_name.get("?Name@%sModuleCreator@@UAEPBDXZ" % cls)
    if nm is None: return None
    code = fdis.read(nm, 16)
    # 习惯: b8 xx xx xx xx (mov eax, imm32)
    if code[0] == 0xB8:
        va = struct.unpack("<I", code[1:5])[0]
        return fdis.cstr(va - BASE if va >= BASE else va, 64)
    return None

# Init/LoadResource/Shut RVA
def method_rva(cls, m):
    return by_name.get("?%s@%s@@UAE_NPAVIKernel@@@Z" % (m, cls))

regs = json.load(open("regs_calls.json"))
by_obj_reg = collections.defaultdict(lambda: collections.defaultdict(list))
for c in regs:
    by_obj_reg[c["obj"]][c["slot"]].append(c["args"])

srcmap = {}
for p in open("src_files.txt").read().splitlines():
    base = p.replace("\\", "/").split("/")[-1].rsplit(".", 1)[0].lower()
    srcmap[base] = p.replace("\\", "/").split("bg_source/")[-1]

out = []
for ctor, cls in creators:
    cand = ["%s.obj" % cls, "%sModule.obj" % cls, "%s.obj" % cls.rstrip("Module")]
    obj = next((c for c in cand if c in objcount), None)
    if obj is None:
        cands = [o for o in objcount if o.lower().startswith(cls.lower()[:6]) and o.endswith(".obj")]
        cands.sort(key=lambda x: -objcount[x])
        obj = cands[0] if cands else None
    slots = by_obj_reg.get(obj, {}) if obj else {}
    ncmd = set(); nhb = []; ncrit = 0; nevent = 0; lua = []; precfg = set()
    for args in slots.get("AddIntCustomHook", []) + slots.get("AddIntCommandHook", []):
        for a in args:
            if isinstance(a, int) and a >= 3: ncmd.add(a)
    for args in slots.get("DeclareHeartBeat", []):
        for a in args:
            if isinstance(a, str) and "::" in a: nhb.append(a)
    ncrit = sum(1 for args in slots.get("DeclareCritical", []) for a in args if isinstance(a, str) and "::" in a)
    nevent = len(slots.get("AddEventCallback", [])) + len(slots.get("AddClassCallback", []))
    for args in slots.get("DeclareLuaExt", []):
        for a in args:
            if isinstance(a, str) and a.startswith("nx_"): lua.append(a)
    for args in slots.get("PreloadConfig", []):
        for a in args:
            if isinstance(a, str) and not a.startswith("fn:"): precfg.add(a)
    out.append({
        "class": cls, "creator_ctor": "%08x" % ctor,
        "name": creator_name(cls),
        "init": ("%08x" % method_rva(cls, "Init")) if method_rva(cls, "Init") else None,
        "loadres": ("%08x" % method_rva(cls, "LoadResource")) if method_rva(cls, "LoadResource") else None,
        "obj": obj, "syms": objcount.get(obj, 0),
        "ncmd": len(ncmd), "cmds": sorted(ncmd),
        "nhb": len(nhb), "hbs": nhb,
        "ncrit": ncrit, "nevent": nevent,
        "nlua": len(lua), "lua": lua,
        "cfg": sorted(precfg),
        "src": srcmap.get(cls.lower().rstrip("module"), None) or srcmap.get(cls.lower(), None),
    })
json.dump(out, open("modules_index.json", "w"), ensure_ascii=False, indent=1)
print("modules:", len(out), " named:", sum(1 for m in out if m["name"]))
for m in out[:8]:
    print(m["class"], m["name"], m["init"], m["obj"], m["syms"], "cmd", m["ncmd"], "hb", m["nhb"], "lua", m["nlua"])
