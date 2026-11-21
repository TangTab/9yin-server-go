"""Who references fxgame.exe; full strings of fxcli.dll + fxupdate around key hits."""
import os, re

STR = re.compile(rb"[\x20-\x7e]{4,120}")
roots = [r"E:\Game\9yin\9yin", r"E:\Game\9yin\9yin\bin", r"E:\Game\9yin\9yin\bin64",
         r"E:\Game\9yin\9yin\updater", r"E:\Game\9yin\9yin\updater_"]

print("=== binaries containing 'fxgame' (case-insens) ===")
for root in roots:
    if not os.path.isdir(root): continue
    for fn in sorted(os.listdir(root)):
        if not fn.lower().endswith((".exe", ".dll")): continue
        p = os.path.join(root, fn)
        d = open(p, "rb").read()
        if b"fxgame" in d.lower():
            print(f"  {p} ({len(d)} B)")

# fxcli.dll full strings, filtered to config/cmd-ish
print("\n=== fxcli.dll strings (cmd/config related) ===")
d = open(r"E:\Game\9yin\9yin\bin64\fxcli.dll", "rb").read()
ss = [m.group().decode() for m in STR.finditer(d)]
kw = re.compile(r"(^|\b)(ip|port|server|serv|host|addr|login|line|area|zone|gate|auth|user|acc|token|ticket|key|cmd|arg|param|command|start|enter|ver|update|patch|lang|path|dir|url|http|extend|ini|fxgame|bin64|sw|mode|gameid|gametype|sn|mac|machine)", re.I)
seen = set()
for s in ss:
    if kw.search(s) and s not in seen:
        seen.add(s)
        print("   ", s)
