# -*- coding: utf-8 -*-
import sys, re, collections
sys.path.insert(0, ".")
from maparse import parse_map
syms, secs = parse_map()
pat = sys.argv[1] if len(sys.argv) > 1 else "."
limit = int(sys.argv[2]) if len(sys.argv) > 2 else 40
rx = re.compile(pat, re.I)
out = [(rva, name, obj) for rva, name, obj in syms if rx.search(name)]
print("match:", len(out))
for rva, name, obj in out[:limit]:
    print("%08x  %-110s %s" % (rva, name[:110], obj))
