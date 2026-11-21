"""1) crack extend.ini Server=  2) find launch-arg tokens in bin64 plain DLLs  3) fxupdate cmdline"""
import re, os, struct

# ---- 1. extend.ini Server= crack ----
raw = "1cx0x4x4ax15x0x49x11x10x5ex40x10x7x40x59x18x1x7x1cx5bx4dxcx2x42x14x53x1cx0x4x4ax15x0x1ex3x2x1ex1fx2x1x1cx17x0x17x1fx8x16x16x60x10x7x1dx14x17x7fx2f"
b = bytes(int(x, 16) for x in raw.split("x") if x)
print(f"Server= blob: {len(b)} bytes: {b.hex()}")

def printable_score(s):
    return sum(1 for c in s if 32 <= c < 127 or c in (10, 13))

# single byte xor
best = []
for k in range(256):
    d = bytes(c ^ k for c in b)
    sc = printable_score(d)
    best.append((sc, k, d))
best.sort(reverse=True)
for sc, k, d in best[:5]:
    print(f"  xor {k:#04x} score={sc}/{len(b)}: {d[:48]!r}")

# xor with index
for base in range(256):
    d = bytes((c ^ ((base + i) & 0xff)) for i, c in enumerate(b))
    sc = printable_score(d)
    if sc > len(b) * 0.8:
        print(f"  xoridx base={base:#x} score={sc}: {d[:48]!r}")

# wide chars?
for k in range(256):
    d = bytes(c ^ k for c in b)
    try:
        s = d.decode("utf-16le")
        if printable_score(s) > len(s) * 0.8:
            print(f"  wide xor {k:#04x}: {s!r}")
    except Exception:
        pass

# ---- 2. arg tokens in plain bin64 DLLs ----
KW = re.compile(r"^(login|server|serv|ip|port|host|addr|account|acc|user|line|area|zone|world|auth|token|ticket|key|char|role|sw|safe|test|debug|console|log|lang|region|ver|version|update|patch|mode|fxmode|game|start|launch|enter|reconnect|disconnect|multi|single|demo|gm|admin|skip|noext|noupdate|nosound|window|full|dx|render|mini|fast)$", re.I)
targets = [r"E:\Game\9yin\9yin\bin64\fxcore.dll", r"E:\Game\9yin\9yin\bin64\fxcli.dll",
           r"E:\Game\9yin\9yin\bin64\fxnet2.dll", r"E:\Game\9yin\9yin\bin64\fxtool.dll",
           r"E:\Game\9yin\9yin\bin64\fxres.exe"]
STR = re.compile(rb"[\x20-\x7e]{3,64}")
dash = re.compile(rb"(?:^|[^A-Za-z0-9])[-/]{1,2}([A-Za-z][A-Za-z0-9_]{1,20})")
for path in targets:
    if not os.path.exists(path):
        print(f"\n--- {path}: MISSING"); continue
    d = open(path, "rb").read()
    hits = set()
    for m in dash.finditer(d):
        t = m.group(1).decode()
        if KW.match(t):
            hits.add(t)
    # also collect GetCommandLine / argv imports evidence
    gc = b"GetCommandLine" in d
    cl = b"CommandLineToArgv" in d
    print(f"\n--- {os.path.basename(path)} size={len(d)} GetCommandLine={gc} CommandLineToArgv={cl}")
    print("   tokens:", sorted(hits) if hits else "(none)")

# ---- 3. updater exe strings ----
for root in (r"E:\Game\9yin\9yin\updater_", r"E:\Game\9yin\9yin\updater", r"E:\Game\9yin\9yin\updater-9"):
    if not os.path.isdir(root):
        continue
    for fn in os.listdir(root):
        if fn.lower().endswith(".exe"):
            p = os.path.join(root, fn)
            d = open(p, "rb").read()
            ss = [m.group().decode() for m in STR.finditer(d)]
            rel = [s for s in ss if re.search(r"fxgame|bin64|\.exe|cmd|arg|param|command", s, re.I)]
            print(f"\n--- {p} ({len(d)} B)")
            for s in rel[:40]:
                print("   ", s)
