# cdsim: 复原官服冷却表(记录 109)的逐次状态（只跟玩家自己的技能施放）。只读分析。
# 输入：pcapskill 的 op11/op12/op13/op10 输出。表行 [类别, 起, 止, 队]；
# 0x11=在行号处插入，0x12=删行号，0x13=逐格写。中途起算时删除越界行按「缺」记。
import sys, re
sys.stdout.reconfigure(encoding='utf-8', errors='replace')
TAOLU = {'zs_normal_00': 'tl_normal', 'CS_jh_cqgf01': 'CS_jh_cqgf', 'CS_jh_cqgf04': 'CS_jh_cqgf',
         'CS_jh_cqgf06': 'CS_jh_cqgf', 'hw_normal_fb': 'hw_normal'}
rows = []
events = []   # 当前施放的表事件（按帧序）
def le(b): return int.from_bytes(bytes(b), 'little')
def flush():
    global events
    if events:
        print('   ' + '  '.join(events))
    events = []
for line in sys.stdin:
    line = line.rstrip('\n')
    m = re.match(r't=\s*([0-9.]+) S2C #(-?\d+) op=0x([0-9A-Fa-f]+) len=(\d+)\s+(.*)', line)
    if not m:
        continue
    t, no, op, ln, hx = m.groups()
    b = [int(x, 16) for x in hx.split()]
    o = op.lower()
    if o == '10' and le(b[10:12]) >= 1 and le(b[12:14]) == 462:
        l2 = le(b[14:18]); name = bytes(b[18:18+l2-1]).decode('ascii', 'replace')
        if name in TAOLU:
            flush()
            print('[%s] 施放 %s（taolu=%s）' % (t, name, TAOLU[name]))
            for i, r in enumerate(rows):
                print('      表[%d] 类别=%s 窗=%dms 队=%s' % (i, r[0], (r[2] - r[1]) if r[1] else 0, r[3]))
        continue
    if o == '11' and le(b[10:12]) == 109:
        n = le(b[12:14]); rows.insert(n, [None, 0, 0, None]); events.append('插入@%d' % n)
    elif o == '12' and le(b[10:12]) == 109:
        n = le(b[12:14])
        if n < len(rows):
            events.append('删@%d(类别%s)' % (n, rows.pop(n)[0]))
        else:
            events.append('删@%d(越界)' % n)
    elif o == '13' and le(b[10:12]) == 109:
        r = le(b[14:16]); c = b[16]
        if r < len(rows):
            if c == 0:
                rows[r][0] = le(b[17:21]); events.append('行%d.类别=%s' % (r, rows[r][0]))
            elif c == 3:
                rows[r][3] = le(b[17:19])
            else:
                v = le(b[17:25])
                if c == 1:
                    rows[r][1] = v
                else:
                    rows[r][2] = v
                    events.append('行%d窗=%dms' % (r, rows[r][2] - rows[r][1]))
flush()
print()
