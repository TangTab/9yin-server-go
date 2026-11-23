# cddecode: pcapskill 输出 → 冷却表(记录 109)写入与相关 0x10 属性的可读时间线。只读分析。
import sys, re
sys.stdout.reconfigure(encoding='utf-8', errors='replace')
def le(b): return int.from_bytes(bytes(b), 'little')
for line in sys.stdin:
    line = line.rstrip('\n')
    m = re.match(r't=\s*([0-9.]+) (S2C|C2S) #(-?\d+) op=0x([0-9A-Fa-f]+) len=(\d+)\s+(.*)', line)
    if not m:
        print(line[:150]); continue
    t, d, no, op, ln, hx = m.groups()
    if d != 'S2C':
        print('t=%8s %s %s' % (t, d, hx[:70])); continue
    b = [int(x, 16) for x in hx.split()]
    o = op.lower()
    if o == '13' and le(b[10:12]) == 109:
        row = le(b[14:16]); col = b[16]
        if col == 0: val = str(le(b[17:21]))
        elif col == 3: val = str(le(b[17:19]))
        else:
            v = le(b[17:25]); val = str(v)
            # 与同批首行比较用：打印相对 1343934000000 的偏移
            val = '%d (rel %d ms)' % (v, v - 1343934000000)
        print('t=%8s 0x13 cooldown row=%d col=%d val=%s' % (t, row, col, val))
    elif o == '11' and le(b[10:12]) == 109:
        print('t=%8s 0x11 cooldown insert row=%d' % (t, le(b[12:14])))
    elif o == '12' and le(b[10:12]) == 109:
        print('t=%8s 0x12 cooldown delete row=%d' % (t, le(b[12:14])))
    elif o == '10':
        n = le(b[10:12]); p = 12; out = []
        for _ in range(n):
            idx = le(b[p:p+2]); p += 2
            if idx == 462:
                l2 = le(b[p:p+4]); p += 4; out.append('CurSkillID=%r' % bytes(b[p:p+l2-1]).decode('ascii', 'replace')); p += l2
            elif idx in (468, 472, 203, 474):
                name = {468: 'MP', 472: 'MPRatio', 474: 'CantParry'}.get(idx)
                if idx == 203:
                    out.append('CantMove=%d' % b[p]); p += 1
                else:
                    out.append('%s=%d' % (name, le(b[p:p+4]))); p += 4
            elif idx == 220:
                l2 = le(b[p:p+4]); p += 4; out.append('PauseSkill=%r' % bytes(b[p:p+l2-1]).decode('ascii', 'replace')); p += l2
            else:
                out.append('idx%d' % idx); break
        if out: print('t=%8s 0x10 %s' % (t, ' '.join(out)))
