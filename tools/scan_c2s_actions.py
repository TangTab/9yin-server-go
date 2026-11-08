# -*- coding: utf-8 -*-
"""Scan c2s dumps for custom actions: 1006 subcommands (cultivation), 31 use-item, 150 jingmai.
Usage: python -I scan_c2s_actions.py <dir-with-bins>
"""
import importlib.util, os, sys
SMOKE = r"G:\pojie\ai\9yin-server\tools\smoke"
spec = importlib.util.spec_from_file_location("capture_deep", os.path.join(SMOKE, "capture_deep.py"))
cd = importlib.util.module_from_spec(spec); spec.loader.exec_module(cd)

def main():
    d = sys.argv[1]
    for name in sorted(os.listdir(d)):
        if not name.endswith("_c2s.bin"):
            continue
        path = os.path.join(d, name)
        frames = cd.load(path)
        hits = []
        for idx, seq, msg_id, vals in cd.c2s_customs(frames):
            if not vals:
                continue
            if msg_id == 1006 and len(vals) >= 2:
                extra = ""
                if len(vals) >= 3:
                    extra = cd.fmt_val(vals[2])
                if len(vals) >= 4:
                    extra += "," + cd.fmt_val(vals[3])
                hits.append(f"[{idx}] 1006/{vals[1]} seq={seq} {extra}")
            elif msg_id == 31:
                args = " ".join(cd.fmt_val(v) for v in vals[1:6])
                hits.append(f"[{idx}] use-item seq={seq} {args}")
            elif msg_id == 150:
                args = " ".join(cd.fmt_val(v) for v in vals[1:4])
                hits.append(f"[{idx}] jingmai seq={seq} {args}")
        if hits:
            print(f"== {name}")
            for h in hits[:60]:
                print("  " + h)
main()
