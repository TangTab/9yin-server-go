# m337check: 对每条带目标的 m211，计算「指向目标」朝向 atan2(dx,dz)，与官服紧随的 m337 朝向比对。
import math, re, sys
lines = open(sys.argv[1], 'rb').read().decode('utf-8', 'replace').splitlines()
num = r'f32\((-?[0-9.eE]+)\)'
rx211 = re.compile(r'^A t=\s*([0-9.]+) C2S #\d+ actenv m211 str\("([^"]*)"\) ' + ' '.join([num]*4) + r' f32\(0\) f32\(0\) f32\(0\) str\(""\)(.*)$')
rx337 = re.compile(r'^A t=\s*([0-9.]+) S2C #\d+ custom m337 raw\((\w+) ')
ev = []
for i, l in enumerate(lines):
    m = rx211.match(l)
    if m:
        ev.append((float(m.group(1)), i, m))
for t, i, m in ev:
    skill = m.group(2)
    px, py, pz, po = [float(m.group(k)) for k in (3, 4, 5, 6)]
    rest = m.group(7)
    tgt = re.search(r'raw\((\w{2}) (\w{2}) (\w{2}) (\w{2}) (\w{2}) (\w{2}) (\w{2}) (\w{2})\) ' + ' '.join([num]*3), rest)
    exp = None
    if tgt:
        tx, ty, tz = [float(tgt.group(k)) for k in (9, 10, 11)]
        dx, dz = tx - px, tz - pz
        dist = math.hypot(dx, dz)
        exp = math.atan2(dx, dz)
    else:
        dist = float('nan')
    # 紧随 120ms 内的 m337
    got = None
    for j in range(i + 1, min(i + 60, len(lines))):
        m2 = rx337.match(lines[j])
        if m2 and float(m2.group(1)) <= t + 0.12:
            vals = re.findall(num, lines[j])
            if len(vals) >= 4:
                got = (float(m2.group(1)), float(vals[3]))
            break
    def fmt(v):
        return '   ---  ' if v is None else '%7.3f' % v
    print('t=%8.3f %-14s dist=%5.2f pressOrient=%7.3f aimOrient=%s m337Orient=%s (dt=%s)' % (
        t, skill, dist, po, fmt(exp), fmt(got[1]) if got else '   (none)', ('%dms' % round((got[0]-t)*1000)) if got else ''))
