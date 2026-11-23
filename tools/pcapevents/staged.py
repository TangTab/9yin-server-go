# staged: 每次暂存受理（无 m337 直收）到下一条 m211 之间的上报流（m220/m15 行走）与服务端应答摘要。
import sys, math
lines = open(sys.argv[1], 'rb').read().decode('utf-8', 'replace').splitlines()
def f32(s): return float(s[len('f32('):-1])
for i, l in enumerate(lines):
    if not l.startswith('A t=') or ' C2S ' not in l or ' actenv m211 ' not in l:
        continue
    p = l.split(); t0 = float(p[2])
    # 直收判定：120ms 内有 m337
    direct = False
    for j in range(i + 1, min(i + 40, len(lines))):
        if not lines[j].startswith('A t='): continue
        q = lines[j].split()
        if float(q[2]) > t0 + 0.12: break
        if ' S2C ' in lines[j] and ' custom m337 ' in lines[j]:
            direct = True; break
    if direct:
        continue
    print('== staged? t=%.3f %s' % (t0, p[7]))
    for j in range(i + 1, min(i + 600, len(lines))):
        l2 = lines[j]
        if not l2.startswith('A t='): continue
        q = l2.split(); t = float(q[2])
        if ' C2S ' in l2 and ' actenv m211 ' in l2:
            print('   next m211 t=%.3f (+%dms) %s' % (t, round((t - t0) * 1000), q[7]))
            break
        if t - t0 > 6.0:
            print('   (no next m211 within 6s)'); break
        if ' C2S ' in l2 and (' actenv m15 ' in l2 and 'i32(7)' in l2 or ' actenv m220 ' in l2 or ' actenv m212 ' in l2):
            print('   %s t=%.3f (+%dms)' % (q[6], t, round((t - t0) * 1000)))
        if ' S2C ' in l2 and (' i32(61) ' in l2 and 'custom m24' in l2 or 'CurSkillID' in l2 or 'custom m26' in l2 or 'm707' in l2 or 'PauseSkill' in l2):
            print('   S2C %s' % ' '.join(q[5:12]))
