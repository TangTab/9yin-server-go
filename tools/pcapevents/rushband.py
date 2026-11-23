# rushband: 对每次 CS_jh_cqgf01 暂存受理，列出其后 m220/m15/m212 上报到 c61 授予的逐条距离。
import math, re, sys
lines = open(sys.argv[1], 'rb').read().decode('utf-8', 'replace').splitlines()
f32 = lambda s: float(s[len('f32('):-1])
def cols(l):
    out = {}
    return l.split()
for i, l in enumerate(lines):
    if ' actenv m211 str("CS_jh_cqgf01") ' not in l or not l.startswith('A t='):
        continue
    p = l.split()
    t0 = float(p[2])
    px, py, pz = f32(p[8]), f32(p[9]), f32(p[10])
    tgt = [tok for tok in p if tok.startswith('raw(')]
    if len(tgt) < 1:
        continue
    fvals = [f32(tok) for tok in p if tok.startswith('f32(')]
    tx, ty, tz = fvals[7], fvals[8], fvals[9]
    print('== press t=%.3f dist=%.2f dy=%.2f target=%s' % (t0, math.hypot(tx - px, tz - pz), py - ty, tgt[0]))
    for j in range(i + 1, min(i + 500, len(lines))):
        l2 = lines[j]
        if not l2.startswith('A t=') or ' C2S #' not in l2 and ' S2C #' not in l2:
            continue
        q = l2.split()
        t = float(q[2])
        if t - t0 > 4.0:
            break
        if ' S2C ' in l2 and ' i32(61) ' in l2 and 'custom m24' in l2:
            fv = [f32(tok) for tok in q if tok.startswith('f32(')]
            k = q.index('i32(61)')
            flag = q[k + 1]
            print('   c61   t=%.3f (+%4dms) dest=(%.2f,%.2f,%.2f) flag=%s anim=%s dest-target=(%.2f,%.2f,%.2f)' % (
                t, round((t - t0) * 1000), fv[0], fv[1], fv[2], flag, q[-1][5:-2], fv[0] - tx, fv[1] - ty, fv[2] - tz))
            break
        if ' C2S ' in l2:
            if ' actenv m212 ' in l2:
                fv = [f32(tok) for tok in q if tok.startswith('f32(')]
                print('   m212  t=%.3f (+%4dms) pos=(%.2f,%.2f,%.2f) dist=%.2f tail=%s' % (
                    t, round((t - t0) * 1000), fv[0], fv[1], fv[2], math.hypot(tx - fv[0], tz - fv[2]), q[-2]))
            elif ' actenv m220 ' in l2:
                fv = [f32(tok) for tok in q if tok.startswith('f32(')]
                print('   m220  t=%.3f (+%4dms) dist=%.2f dy=%.2f' % (t, round((t - t0) * 1000), math.hypot(tx - fv[0], tz - fv[2]), fv[1] - ty))
            elif ' actenv m15 ' in l2 and 'i32(4)' in l2:
                fv = [f32(tok) for tok in q if tok.startswith('f32(')]
                print('   m15st t=%.3f (+%4dms) dist=%.2f' % (t, round((t - t0) * 1000), math.hypot(tx - fv[0], tz - fv[2])))
            elif ' actenv m211 ' in l2:
                print('   (new m211 %s before c61)' % q[q.index('actenv') + 1])
                break
