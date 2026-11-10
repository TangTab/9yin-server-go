import ida_hexrays
import idaapi
import re

cf = ida_hexrays.decompile(0x12A2A6E0)
src = str(cf)

# a1[N] = "string" | &byte_12CD27C1 | &sub_XXXX | word_XXX | byte_XXX | 0
re_assign = re.compile(r'^\s*a1\[(\d+)\]\s*=\s*(.+?);\s*(?://\s*"(.*)")?\s*$')
slots = {}
for ln in src.split("\n"):
    m = re_assign.match(ln)
    if m:
        n = int(m.group(1))
        val = m.group(2).strip()
        cmt = m.group(3)
        slots[n] = (val, cmt)

print("slots:", len(slots), "min N:", min(slots), "max N:", max(slots))

# what is byte_12CD27C1?
b = idaapi.get_bytes(0x12CD27C1, 8)
print("byte_12CD27C1 =", b)

# build id -> (name, native)
rows = []
for n in sorted(slots):
    if (n - 15) % 2 != 0:
        continue
    mid = (n - 15) // 2
    val, cmt = slots[n]
    hval, _ = slots.get(n + 1, ("?", None))
    # name resolution
    if cmt:
        name = cmt
    elif 'byte_12CD27C1' in val:
        name = ""  # empty filler
    elif val == '0':
        name = None  # slot not a name
    else:
        name = val
    native = ''
    if 'sub_' in hval:
        native = hval.replace('&', '')
    elif hval not in ('0', '?'):
        native = hval.replace('&', '')
    rows.append((mid, name or '', native))

# check pair structure: even slots (handlers) may be absent
rows = [r for r in rows if r[1] != None or True]
with open(r"G:\pojie\ai\9yin-server-go\tmp\ida_scripts\custommsg_table.txt", "w", encoding="utf-8") as f:
    for mid, name, native in rows:
        f.write("m%d\t%s\t%s\n" % (mid, name, native))
print("total ids:", len(rows))

want = [1, 2, 3, 9, 10, 11, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 52, 98, 99, 100, 136, 137, 169, 228, 235, 259, 281, 293, 294, 366, 382, 386, 390, 395, 438, 458, 496, 504, 505, 561, 596, 607, 623, 647, 648, 655, 680, 681, 683, 756, 758, 430, 416, 419]
d = dict((r[0], (r[1], r[2])) for r in rows)
for i in want:
    v = d.get(i)
    print("m%d\t%s\t%s" % (i, v[0] if v else '<absent>', v[1] if v else ''))
