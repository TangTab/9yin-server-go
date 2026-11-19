# -*- coding: utf-8 -*-
"""GPKitSvr.dll 动态行为探针(加载用户自己的服务端组件,载荷为自制 ret 桩)。"""
import ctypes, os, struct, sys

os.chdir(os.path.dirname(os.path.abspath(__file__)))

dll = ctypes.CDLL(r"F:\Game\Online\9yin\BGserverCJ\Debug\GPKitSvr.dll")
fn = dll.GPKCreateSvrDynCode
fn.restype = ctypes.c_void_p
obj = fn()
print("obj =", hex(obj))
if not obj:
    sys.exit("create failed")

vtbl_addr = struct.unpack_from("<I", ctypes.string_at(obj, 4), 0)[0]
vtbl = ctypes.cast(ctypes.c_void_p(vtbl_addr), ctypes.POINTER(ctypes.c_void_p))
print("vtbl =", hex(vtbl_addr), "slots:", [hex(vtbl[i] or 0) for i in range(9)])


def call(slot, *args):
    proto = ctypes.WINFUNCTYPE(ctypes.c_int, ctypes.c_void_p, *[ctypes.c_uint32] * len(args))
    return proto(vtbl[slot])(obj, *args)


# 全局状态 dump(CSvrDynCode 0x414 布局)
def dump_obj(tag):
    raw = ctypes.string_at(obj, 0x414)
    flag = raw[4]
    n = struct.unpack_from("<H", raw, 0x3F0)[0]
    cnt = struct.unpack_from("<H", raw, 0x3F2)[0]
    slots = struct.unpack_from("<252I", raw, 8)
    used = [(i, hex(s)) for i, s in enumerate(slots[:n if n else 0]) if s]
    print("[%s] flag=%d n@3f0=%d cnt@3f2=%d slots=%s" % (tag, flag, n, cnt, used))


dump_obj("before init")
r = call(1, ctypes.c_char_p(b"DynCodeBin\\Server").value if False else int(ctypes.cast(ctypes.c_char_p(b"DynCodeBin\\Server"), ctypes.c_void_p).value),
         int(ctypes.cast(ctypes.c_char_p(b"DynCodeBin\\Client"), ctypes.c_void_p).value))
print("slot1(Init) ret =", r)
dump_obj("after init")

if r >= 0:
    r2 = call(2)
    print("slot2(RandomPick) ret =", r2)
    out = ctypes.create_string_buffer(4)
    # slot3(this, packed, out) ret 0xc
    packed = (r2 << 16) if r2 is not None and r2 >= 0 else 0
    r3 = call(3, packed, ctypes.addressof(ctypes.c_uint32.from_buffer(out)))
    print("slot3(GetData idx=%d) ret(size?)=%r out=%r" % (r2, r3, out.raw[:8]))
    # slot4: (this, a, b, codeid) ret 0x10
    r4 = call(4, 0xDEADBE01, 0xDEADBE02, r2)
    print("slot4(Call1) ret =", hex(r4 & 0xFFFFFFFF) if r4 is not None else None)
    r5 = call(5, 0xDEADBEEF, 0xC4, 0)
    print("slot5(Call2) ret =", hex(r5 & 0xFFFFFFFF) if r5 is not None else None)
