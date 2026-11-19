# -*- coding: utf-8 -*-
"""GPKitSvr64.dll 动态行为探针(64 位,自制无害载荷)。"""
import ctypes, os, struct, sys

os.chdir(os.path.dirname(os.path.abspath(__file__)))

dll = ctypes.CDLL(r"F:\Game\Online\9yin\BGserverCJ\Debug\GPKitSvr64.dll")
fn = dll.GPKCreateSvrDynCode
fn.restype = ctypes.c_void_p
obj = fn()
print("obj =", hex(obj or 0))
if not obj:
    sys.exit("create failed")

vtbl = ctypes.cast(ctypes.c_void_p(struct.unpack_from("<Q", ctypes.string_at(obj, 8), 0)[0]),
                   ctypes.POINTER(ctypes.c_uint64))
print("vtbl slots:", [hex(vtbl[i]) for i in range(9)])


def call(slot, *args):
    proto = ctypes.CFUNCTYPE(ctypes.c_int64, ctypes.c_void_p, *[ctypes.c_uint64] * len(args))
    return proto(int(vtbl[slot]))(obj, *args)


def dump(tag):
    raw = ctypes.string_at(obj, 0x820)
    flag = raw[8]
    n = struct.unpack_from("<I", raw, 0x808)[0]
    cnt = struct.unpack_from("<I", raw, 0x80C)[0]
    seq = struct.unpack_from("<I", raw, 0x810)[0]
    slots = struct.unpack_from("<256Q", raw, 0x10)
    used = [(i, hex(s)) for i, s in enumerate(slots) if s]
    print("[%s] flag=%d n@808=%d cnt@80c=%d seq@810=%d slots=%s" % (tag, flag, n, cnt, seq, used[:5]))


dump("before")
pSrv = ctypes.cast(ctypes.c_char_p(b"abs:\\DynCodeBin\\Server"), ctypes.c_void_p).value or 0
pClt = ctypes.cast(ctypes.c_char_p(b"abs:\\DynCodeBin\\Client"), ctypes.c_void_p).value or 0
srv = ctypes.create_string_buffer(rb"G:\pojie\ai\9yin-server-go\tmp_gpk\DynCodeBin\Server")
clt = ctypes.create_string_buffer(rb"G:\pojie\ai\9yin-server-go\tmp_gpk\DynCodeBin\Client")
r = call(1, ctypes.addressof(srv), ctypes.addressof(clt))
print("slot1(Init srv, clt) ret =", r)
dump("after")

if r >= 0:
    idx = call(2)
    print("slot2(RandomPick) =", idx)
    if idx >= 0:
        out = ctypes.c_uint64(0)
        proto = ctypes.CFUNCTYPE(ctypes.c_int64, ctypes.c_void_p, ctypes.c_uint64, ctypes.POINTER(ctypes.c_uint64))
        sz = proto(int(vtbl[3]))(obj, idx << 16, ctypes.byref(out))
        print("slot3(GetData) size=%d ptr=%x" % (sz, out.value))
        if out.value and sz > 0:
            print("  data:", ctypes.string_at(out.value, min(sz, 16)).hex())
        a = call(4, 0xDEAD0001, 0xDEAD0002, idx << 16)
        print("slot4(Call1) ret =", hex(a & 0xFFFFFFFFFFFFFFFF))
        b = call(5, 0, 0, idx << 16)
        print("slot5(Call2) ret =", hex(b & 0xFFFFFFFFFFFFFFFF))
