"""1) bin\\fxgame.exe sections+strings  2) constraint-solve extend.ini repeating-XOR"""
import struct, re

# ---- 1. 32-bit twin ----
p = r"E:\Game\9yin\9yin\bin\fxgame.exe"
d = open(p, "rb").read()
e = struct.unpack_from("<I", d, 0x3c)[0]
mach, nsec, tds = struct.unpack_from("<HHI", d, e+4)
magic = struct.unpack_from("<H", d, e+24)[0]
soi = struct.unpack_from("<I", d, e+24+56)[0]
print(f"bin\\fxgame.exe: machine={mach:#x} nsec={nsec} ts={tds:#x} magic={magic:#x} SizeOfImage={soi:#x}")
secs = e + 24 + struct.unpack_from("<H", d, e+20)[0]
for i in range(nsec):
    nm = d[secs+i*40:secs+i*40+8].rstrip(b"\x00").decode("ascii", "replace")
    vs, sa, rs, rp = struct.unpack_from("<IIII", d, secs+i*40+8)
    print(f"   {nm:<10} VA={sa:#010x} VS={vs:#010x} RS={rs:#010x}")

STR = re.compile(rb"[\x20-\x7e]{3,120}")
kw = re.compile(r"(cmd|arg|param|command|ip|port|server|serv|host|addr|login|line|area|zone|gate|auth|user|acc|token|ticket|extend|start|enter|lang|ver|update|mode|gameid|sn|http|url|path)", re.I)
print("\n--- cmd-related strings in bin\\fxgame.exe ---")
seen = set()
for m in STR.finditer(d):
    s = m.group().decode()
    if kw.search(s) and s not in seen:
        seen.add(s)
        if len(seen) <= 80:
            print("   ", s)

# ---- 2. extend.ini constraint solve ----
raw = "1cx0x4x4ax15x0x49x11x10x5ex40x10x7x40x59x18x1x7x1cx5bx4dxcx2x42x14x53x1cx0x4x4ax15x0x1ex3x2x1ex1fx2x1x1cx17x0x17x1fx8x16x16x60x10x7x1dx14x17x7fx2f"
C = bytes(int(x, 16) for x in raw.split("x") if x)
ALLOWED = set(b"0123456789.:;|=,_-+/ abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\x00")
print(f"\n--- extend.ini solve: {len(C)} bytes ---")
solutions = []
for L in range(1, 29):
    # key candidates per residue
    cand = [set(range(256)) for _ in range(L)]
    ok = True
    for i, c in enumerate(C):
        poss = {k for k in range(256) if (c ^ k) in ALLOWED}
        cand[i % L] &= poss
        if not cand[i % L]:
            ok = False; break
    if ok and all(cand):
        # count total key space
        total = 1
        for s in cand: total *= len(s)
        if total < 200000:
            solutions.append((L, cand))
            print(f"  keylen={L} keyspace={total}")
            # emit a few full decodes with ip-biased choice: prefer decode containing digits/dots
            import itertools
            shown = 0
            for combo in itertools.product(*cand):
                key = bytes(combo)
                P = bytes(c ^ key[i % L] for i, c in enumerate(C))
                txt = P.decode("latin1")
                if re.search(r"\d+\.\d+\.\d+\.\d+", txt) or txt.count(".") >= 2:
                    print(f"    key={key.hex()} -> {txt!r}")
                    shown += 1
                    if shown >= 6: break
if not solutions:
    print("  no repeating-XOR solution with that charset; transform is not plain repeating XOR")
