# combatscan —— 旁观者视角群战类抓包的战斗结构化扫描（tools/pcapevents dumpall 文本 → 结论表）。
#
# 观测端角色不参战时，别人的出招/受击/受控/位移都在这些帧里：
#   m20[施法者, 动作, 目标, …]        出招广播           m205[攻击者, 技能, 受击者…]  命中节拍
#   m18[受击者, 动作, 0, 攻击者]      受击动作           m24 子码 40/56/78/79/71…      受控/特效
#   0x10 / 0x2D                      属性推送（State/LogicState/CurSkillID/CurIsAirSkill…）
#   0x1F / 0x2C                      位置（单个 / 压缩批量）  0x21  移动意图（目的地+速度）
#
# 用法：
#   python -I combatscan.py <dumpall> skills                  按招式聚合：出招数/命中节拍/施法者位移
#   python -I combatscan.py <dumpall> hits [N=40]             命中时受击者收到的帧（按技能聚合反应形态）
#   python -I combatscan.py <dumpall> control                 受控逐例：恢复时长 + 位置轨迹 + 位移流
#   python -I combatscan.py <dumpall> cast <t> <obj十六进制>  某施法者距 t 最近一次出招的逐帧时间线
import re
import statistics
import sys

T = re.compile(r'^A t=\s*([0-9.]+) (C2S|S2C) #\d+ (.*)$')
OBJ = re.compile(r'^raw\(([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2}) ([0-9a-f]{2})\)$')
VAL = re.compile(r'raw\([0-9a-f ]+\)|str\("(?:[^"\\]|\\.)*"\)|obj\([^)]*\)|[a-z0-9]+\([^)]*\)')
E21 = re.compile(r' \{([0-9a-f]{8}) dst=([0-9.,-]+) o=([0-9.,-]+) spd=([0-9.,-]+) rot=([0-9.,-]+) m2=([0-9.,-]+) m3=([0-9.,-]+) f=(\d+)\}')
E2C = re.compile(r' \{([0-9a-f]{8}) (\S+),(\S+),(\S+) o=(\S+)\}')
LOC = re.compile(r'obj=([0-9a-f]{8}) own=[0-9a-f]{8} x=(\S+) y=(\S+) z=(\S+) orient=(\S+)')
E2D = re.compile(r' \{([0-9a-f]{8}) ((?: \S+)*?)\}(?= \{|\s*$)')

CONTROL = {56: 'c56', 78: 'c78', 79: 'c79'}
WATCH_PROPS = ('State', 'LogicState', 'CurSkillID', 'CurIsAirSkill', 'InParry', 'CantMove', 'CantUseSkill', 'CantSkill', 'JumpTimes', 'HitHPRatio')


def parse_token(tok):
    """dumpall 的一个值 token → (类型, 值)。"""
    m = OBJ.match(tok)
    if m:
        bs = [int(g, 16) for g in m.groups()]
        return 'obj', (int.from_bytes(bs[0:4], 'little'), int.from_bytes(bs[4:8], 'little'))
    kind = tok.split('(', 1)[0]
    body = tok[len(kind) + 1:-1]
    if kind == 'str':
        return 'str', body[1:-1]
    if kind == 'obj':
        return 'obj', tuple(int(v, 16) for v in body.split('/'))
    return kind, body


def load(path):
    """逐行 → 事件序列（时间升序）。k：msg / prop / mprop / dest / loc / pos / other。"""
    evs = []
    for l in open(path, encoding='utf-8', errors='replace'):
        m = T.match(l)
        if not m:
            continue
        b = m.group(3)
        e = {'t': float(m.group(1)), 's2c': m.group(2) == 'S2C', 'raw': b}
        c = re.match(r'^(?:custom|actenv) m(\d+)(.*)$', b)
        if c:
            e['k'], e['n'] = 'msg', int(c.group(1))
            e['v'] = [parse_token(x) for x in VAL.findall(c.group(2))]
        elif b.startswith('0x10 '):
            e['k'] = 'prop'
            e['obj'] = int(re.search(r'obj=([0-9a-f]{8})', b).group(1), 16)
            e['props'] = [(p.split('=', 1)[0], parse_token(p.split('=', 1)[1])) for p in b.split(':', 1)[1].split(' ') if '=' in p]
        elif b.startswith('0x2D '):
            e['k'] = 'mprop'
            e['items'] = [(int(g[0], 16), [(p.split('=', 1)[0], parse_token(p.split('=', 1)[1])) for p in g[1].split(' ') if '=' in p]) for g in E2D.findall(b)]
        elif b.startswith('op=0x21 '):
            e['k'] = 'dest'
            e['entries'] = [(int(g[0], 16), tuple(float(x) for x in g[1].split(',')), float(g[3]), int(g[7])) for g in E21.findall(b)]
        elif b.startswith('op=0x1F '):
            e['k'] = 'loc'
            g = LOC.search(b)
            e['obj'], e['xyz'], e['o'] = int(g.group(1), 16), tuple(float(g.group(i)) for i in (2, 3, 4)), float(g.group(5))
        elif b.startswith('op=0x2C '):
            e['k'] = 'pos'
            e['entries'] = [(int(g[0], 16), tuple(float(x) for x in g[1:4]), float(g[4])) for g in E2C.findall(b)]
        else:
            e['k'] = 'other'
        evs.append(e)
    return evs


def msg(e, n):
    return e['k'] == 'msg' and e['n'] == n and e['s2c']


def oid(tok):
    return tok[1][0] if tok and tok[0] == 'obj' else None


def iv(e, i):
    return int(e['v'][i][1]) if len(e['v']) > i and e['v'][i][0] == 'i32' else None


def sv(e, i):
    return e['v'][i][1] if len(e['v']) > i and e['v'][i][0] == 'str' else None


def dist(a, b):
    return sum((x - y) ** 2 for x, y in zip(a, b)) ** 0.5


def track(evs):
    """每对象的位置采样序列 [(t, xyz, orient)]：0x2C 批量 + 0x1F 单帧。"""
    tr = {}
    for e in evs:
        if e['k'] == 'pos':
            for o, xyz, ori in e['entries']:
                tr.setdefault(o, []).append((e['t'], xyz, ori))
        elif e['k'] == 'loc':
            tr.setdefault(e['obj'], []).append((e['t'], e['xyz'], e['o']))
    # 本地玩家的位置来自它的上行 m15[标志, 状态, x, y, z, 朝向, …]（服务端不向本人
    # 广播 0x2C）。本地玩家 = 0x10 属性帧最多的对象。
    own = {}
    for e in evs:
        if e['k'] == 'prop':
            own[e['obj']] = own.get(e['obj'], 0) + 1
    if own:
        self = max(own.items(), key=lambda kv: kv[1])[0]
        for e in evs:
            if e['k'] == 'msg' and e['n'] == 15 and not e['s2c'] and len(e['v']) >= 6 and e['v'][2][0] == 'f32':
                tr.setdefault(self, []).append((e['t'], tuple(float(e['v'][i][1]) for i in (2, 3, 4)), float(e['v'][5][1])))
    return tr


def sample(tr, o, t0, t1):
    """对象 o 在 [t0, t1] 的位置采样。"""
    return [s for s in tr.get(o, ()) if t0 <= s[0] <= t1]


def props_on(evs, o, t0, t1, names=WATCH_PROPS):
    """对象 o 在窗口内被推送的受关注属性 [(相对时刻, 名, 值)]（0x10 与 0x2D）。"""
    out = []
    for e in evs:
        if not (t0 <= e['t'] <= t1):
            continue
        if e['k'] == 'prop' and e['obj'] == o:
            out += [(e['t'], n, v) for n, v in e['props'] if n in names]
        elif e['k'] == 'mprop':
            for oo, ps in e['items']:
                if oo == o:
                    out += [(e['t'], n, v) for n, v in ps if n in names]
    return out


def casts(evs):
    """每次出招 = 一条广播 m20[施法者, 动作, 目标, …]。"""
    return [{'i': i, 't': e['t'], 'caster': e['v'][0][1][0], 'action': e['v'][1][1], 'target': oid(e['v'][2]) if len(e['v']) > 2 else None}
            for i, e in enumerate(evs) if msg(e, 20) and len(e['v']) >= 2 and e['v'][0][0] == 'obj' and e['v'][1][0] == 'str']


def burst(evs, t, pred):
    """与时刻 t 同毫秒风暴内满足 pred 的事件。"""
    return [e for e in evs if abs(e['t'] - t) < 0.0015 and pred(e)]


# ---------------------------------------------------------------- skills / cast

def cmd_skills(evs, tr):
    cs = casts(evs)
    nxt = {}
    for c in cs:
        nxt.setdefault(c['caster'], []).append(c['t'])
    by = {}
    for c in cs:
        ts = nxt[c['caster']]
        end = min([x for x in ts if x > c['t']] + [c['t'] + 3.0])
        # 命中节拍：本施法者在窗口内的 m205
        hits = [e for e in evs[c['i']:] if e['t'] <= end and msg(e, 205) and e['v'] and oid(e['v'][0]) == c['caster']]
        s = sample(tr, c['caster'], c['t'] - 0.3, end)
        p0 = s[0][1] if s else None
        move = max((dist(p0, x[1]) for x in s), default=0.0) if p0 else 0.0
        dy = (max(x[1][1] for x in s) - s[0][1][1]) if s else 0.0
        dests = [g for e in evs[c['i']:] if e['t'] <= end and e['k'] == 'dest' for g in e['entries'] if g[0] == c['caster']]
        air = any(v not in ('0', 'byte(0)') for _, n, v in props_on(evs, c['caster'], c['t'], end, ('CurIsAirSkill',)))
        by.setdefault(c['action'], []).append((c, len(hits), move, dy, dests, air))
    rows = sorted(by.items(), key=lambda kv: -len(kv[1]))
    print('出招广播 m20 共 %d 次，%d 个招式动作，%d 个施法者' % (len(cs), len(by), len({c['caster'] for c in cs})))
    print('%-30s %4s %4s %-14s %7s %7s %6s %5s  %s' % ('招式动作', '次数', '施法', '命中节拍 中/大', '中位位移', '最大位移', '升空中', '有0x21', '出招窗口内下发的属性（前 7）'))
    for action, lst in rows[:int(sys.argv[3]) if len(sys.argv) > 3 else 40]:
        nh = [h for _, h, *_ in lst]
        mv = [m for _, _, m, *_ in lst]
        dy = [d for _, _, _, d, *_ in lst]
        d21 = sum(1 for x in lst if any(g[2] > 0 for g in x[4]))
        air = sum(1 for x in lst if x[5] or x[3] > 0.6)
        names = {}
        for c, *_ in lst:
            for _, n, _ in props_on(evs, c['caster'], c['t'] - 0.02, c['t'] + 0.08):
                names[n] = names.get(n, 0) + 1
        top = ','.join(n for n, _ in sorted(names.items(), key=lambda kv: -kv[1])[:7])
        print('%-30s %4d %4d %-14s %7.2f %7.2f %6d %5d  %s' % (action, len(lst), len({x[0]['caster'] for x in lst}),
              '%d/%d' % (statistics.median(nh), max(nh)), statistics.median(mv), max(mv), air, d21, top))


def cmd_cast(evs, tr):
    o, t = int(sys.argv[4], 16), float(sys.argv[3])
    cs = [c for c in casts(evs) if c['caster'] == o]
    c = min(cs, key=lambda c: abs(c['t'] - t))
    end = min([x['t'] for x in cs if x['t'] > c['t']] + [c['t'] + 3.0])
    print('出招 t=%.3f 施法者=%08x 动作=%s 目标=%s 窗口至 %.3f' % (c['t'], c['caster'], c['action'], ('%08x' % c['target']) if c['target'] else None, end))
    rows = []
    for e in evs:
        if not (c['t'] - 0.3 <= e['t'] <= end):
            continue
        dt = e['t'] - c['t']
        if msg(e, 205) and e['v'] and oid(e['v'][0]) == o:
            rows.append((dt, '命中', '%s -> %s' % (sv(e, 1), ' '.join('%08x' % x for x in map(oid, e['v'][2:]) if x))))
        elif msg(e, 19) and len(e['v']) > 2 and oid(e['v'][0]) == o:
            rows.append((dt, 'm19 锁定', sv(e, 2)))
        elif msg(e, 18) and len(e['v']) > 3 and oid(e['v'][3]) == o:
            rows.append((dt, '受击动作', '受击者 %08x %s' % (oid(e['v'][0]), sv(e, 1))))
        elif msg(e, 24) and len(e['v']) >= 2 and iv(e, 1) in CONTROL and oid(e['v'][0]) == c['target']:
            rows.append((dt, CONTROL[iv(e, 1)], '目标 %08x %s %s' % (c['target'], sv(e, 3) or '', sv(e, 4) or '')))
        elif e['k'] == 'prop' and e['obj'] == o:
            rows.append((dt, '属性', ' '.join('%s=%s' % (n, v) for n, v in e['props'])[:150]))
        elif e['k'] == 'dest':
            for g in e['entries']:
                if g[0] == o:
                    rows.append((dt, '0x21 意图', 'dst=%.1f,%.1f,%.1f spd=%.2f f=%d' % (g[1] + (g[2], g[3]))))
        elif e['k'] in ('loc', 'pos'):
            for s in ([e] if e['k'] == 'loc' else []):
                rows.append((dt, '0x1F 位置', '%.1f,%.1f,%.1f' % s['xyz']))
            for oo, xyz, ori in e.get('entries', ()):
                if oo == o:
                    rows.append((dt, '0x2C 位置', '%.1f,%.1f,%.1f o=%.2f' % (xyz + (ori,))))
    last = None
    for dt, kind, text in sorted(rows):
        if kind in ('0x2C 位置',) and last == (kind, text):
            continue
        last = (kind, text)
        print('%+7.3f  %-10s %s' % (dt, kind, text))


# ---------------------------------------------------------------- hits

def cmd_hits(evs):
    """每次命中（m205）时受击者收到的帧：按技能聚合成反应形态。"""
    sig = {}
    total = 0
    for i, e in enumerate(evs):
        if not (msg(e, 205) and len(e['v']) >= 3 and e['v'][1][0] == 'str'):
            continue
        a, skill = oid(e['v'][0]), sv(e, 1)
        for vt in e['v'][2:]:
            v = oid(vt)
            if not v:
                continue
            total += 1
            st = []
            for f in burst(evs, e['t'], lambda f: f['k'] == 'msg'):
                if msg(f, 18) and len(f['v']) > 3 and oid(f['v'][0]) == v and oid(f['v'][3]) == a:
                    st.append('m18=%s' % sv(f, 1))
                elif msg(f, 20) and len(f['v']) > 1 and oid(f['v'][0]) == v and f['v'][1][0] == 'str':
                    st.append('m20=%s' % sv(f, 1))
                elif msg(f, 24) and len(f['v']) >= 2 and oid(f['v'][0]) == v and iv(f, 1) in (40, 56, 78, 79, 71):
                    code = iv(f, 1)
                    if code in (78, 79) and len(f['v']) > 2 and oid(f['v'][2]) not in (None, a):
                        continue  # 同毫秒别的攻击者造成的受控
                    st.append('%s%s' % (CONTROL.get(code, 'c%d' % code), '=%s' % sv(f, 2) if sv(f, 2) else ''))
            for tt, n, val in props_on(evs, v, e['t'] - 0.0005, e['t'] + 0.0005, ('State', 'LogicState')):
                st.append('%s=%s' % (n, val[1]))
            key = ' '.join(st) or '(无受击帧)'
            d = sig.setdefault(skill, {})
            d[key] = d.get(key, 0) + 1
    print('命中事件 m205 的（技能, 受击者）对共 %d；下表为每种技能下受击者同毫秒收到的帧（次数）' % total)
    top = int(sys.argv[3]) if len(sys.argv) > 3 else 30
    for skill, d in sorted(sig.items(), key=lambda kv: -sum(kv[1].values()))[:top]:
        print('%-28s ×%3d' % (skill, sum(d.values())))
        for s, n in sorted(d.items(), key=lambda kv: -kv[1])[:3]:
            print('      %3d  %s' % (n, s[:230]))


# ---------------------------------------------------------------- control

def cmd_control(evs, tr):
    """受控逐例：c78/c79 与 State=hurt_* 推送 → 恢复时长、位置轨迹、0x21 位移流。"""
    events = []
    for e in evs:
        if msg(e, 24) and len(e['v']) >= 2 and iv(e, 1) in (78, 79):
            events.append((e['t'], oid(e['v'][0]), ('c78' if iv(e, 1) == 78 else 'c79') + ('' if not sv(e, 3) else '=%s' % sv(e, 3)), oid(e['v'][2])))
        elif e['k'] == 'prop' and any(n == 'State' and v[1].startswith('hurt') for n, v in e['props']):
            st = [v[1] for n, v in e['props'] if n == 'State'][0]
            events.append((e['t'], e['obj'], 'State=%s' % st, None))
        elif e['k'] == 'mprop':
            for oo, ps in e['items']:
                st = [v[1] for n, v in ps if n == 'State' and v[1].startswith('hurt')]
                if st:
                    events.append((e['t'], oo, 'State=%s' % st[0], None))
    kinds = {}
    print('%7s %-10s %-9s %-26s %6s %7s %7s %6s %5s %4s' % ('t', '受控者', '形态', '动画', '恢复(s)', '水平位移', '最大|位移|', '升幅', '采样', '0x21'))
    for t, v, kind, a in events:
        rec = None
        for tt, n, val in props_on(evs, v, t + 0.001, t + 6.0, ('State', 'LogicState')):
            if (n == 'State' and val == ('str', '')) or (n == 'LogicState' and val[1] in ('120',)):
                rec = tt - t
                break
        s = sample(tr, v, t - 0.35, t + 3.0)
        base = ([x for x in s if x[0] <= t] or s)[-1] if s else None
        if base:
            ds = [dist(base[1], x[1]) for x in s]
            horiz = max(((x[1][0] - base[1][0]) ** 2 + (x[1][2] - base[1][2]) ** 2) ** 0.5 for x in s)
            rise = max(x[1][1] - base[1][1] for x in s)
        else:
            ds, horiz, rise = [0.0], 0.0, 0.0
        n21 = len([g for e in evs if t <= e['t'] <= t + 2.0 and e['k'] == 'dest' for g in e['entries'] if g[0] == v and g[2] > 0])
        print('%7.3f %08x  %-9s %-26s %6s %7.2f %7.2f %6.2f %5d %4d' % (t, v, kind.split('=')[0], kind.split('=')[1] if '=' in kind else (a or ''), '%.2f' % rec if rec is not None else '>6', horiz, max(ds), rise, len(s), n21))
        kinds.setdefault(kind, []).append((rec, horiz, max(ds), rise, n21))
    print('--- 按形态汇总 ---')
    for kind, lst in sorted(kinds.items()):
        recs = [r for r, *_ in lst if r is not None]
        print('%-26s ×%2d  恢复 %s  水平位移中位 %.2f 最大 %.2f  升幅最大 %.2f  带 0x21 的 %d' % (
            kind, len(lst), ('中位 %.2f 范围 %.2f~%.2f' % (statistics.median(recs), min(recs), max(recs))) if recs else '未见恢复',
            statistics.median([x[1] for x in lst]), max(x[1] for x in lst), max(x[3] for x in lst), sum(1 for x in lst if x[4])))


# ---------------------------------------------------------------- track

def cmd_track(evs, tr):
    """某对象在窗口内的位置采样（相对基准点的增量）+ 受控/受击/属性事件 + 0x21 意图。
    用法: track <t> <obj十六进制> [前秒=0.5] [后秒=3.0]"""
    o, t = int(sys.argv[4], 16), float(sys.argv[3])
    before = float(sys.argv[5]) if len(sys.argv) > 5 else 0.5
    after = float(sys.argv[6]) if len(sys.argv) > 6 else 3.0
    rows = []
    for e in evs:
        if not (t - before <= e['t'] <= t + after):
            continue
        dt = e['t'] - t
        if e['k'] == 'pos':
            rows += [(dt, '0x2C', '%.2f,%.2f,%.2f o=%.2f' % (xyz + (ori,))) for oo, xyz, ori in e['entries'] if oo == o]
        elif e['k'] == 'loc' and e['obj'] == o:
            rows.append((dt, '0x1F', '%.2f,%.2f,%.2f o=%.2f' % (e['xyz'] + (e['o'],))))
        elif e['k'] == 'dest':
            rows += [(dt, '0x21', 'dst=%.2f,%.2f,%.2f spd=%.2f f=%d' % (g[1] + (g[2], g[3]))) for g in e['entries'] if g[0] == o]
        elif msg(e, 24) and len(e['v']) >= 2 and oid(e['v'][0]) == o and iv(e, 1) in (40, 56, 78, 79, 71, 4, 17):
            args = ' '.join(str(x[1]) if x[0] != 'obj' else '%08x' % x[1][0] for x in e['v'][2:])
            rows.append((dt, 'm24/%d' % iv(e, 1), args[:120]))
        elif msg(e, 18) and len(e['v']) > 1 and oid(e['v'][0]) == o:
            rows.append((dt, 'm18', '%s 攻击者=%08x' % (sv(e, 1), oid(e['v'][3]) if len(e['v']) > 3 else 0)))
        elif msg(e, 20) and len(e['v']) > 1 and oid(e['v'][0]) == o:
            rows.append((dt, 'm20', sv(e, 1)))
        elif msg(e, 205) and e['v'] and o in [oid(x) for x in e['v'][2:]]:
            rows.append((dt, '被命中', '%s by %08x' % (sv(e, 1), oid(e['v'][0]))))
        elif msg(e, 169) and len(e['v']) > 2 and oid(e['v'][2]) == o:
            rows.append((dt, 'm169', '%s %s by %08x' % ('加' if iv(e, 0) == 0 else '删', sv(e, 1), oid(e['v'][3]))))
        elif msg(e, 15) and False:
            pass
        elif msg(e, 188) and len(e['v']) > 1 and oid(e['v'][0]) == o:
            rows.append((dt, 'm188', str(e['v'][1][1]) if len(e['v']) > 1 else ''))
        elif msg(e, 189) and len(e['v']) > 1 and oid(e['v'][0]) == o:
            rows.append((dt, 'm189', ''))
    rows += [(tt - t, '属性', ' '.join('%s=%s' % (n, v[1]) for n, v in [(n, v)])) for tt, n, v in props_on(evs, o, t - before, t + after)]
    s = sample(tr, o, t - before, t)
    base = s[-1][1] if s else None
    print('=== 对象 %08x  基准位置（t 前最近采样）%s ===' % (o, '%.2f,%.2f,%.2f' % base if base else '无'))
    for dt, kind, text in sorted(rows):
        if kind in ('0x2C', '0x1F') and base:
            xyz = tuple(float(x) for x in text.split(' o=')[0].split(','))
            text += '   Δ=(%+.2f,%+.2f,%+.2f) 水平 %.2f' % (xyz[0] - base[0], xyz[1] - base[1], xyz[2] - base[2], dist(xyz, base))
        print('%+7.3f  %-7s %s' % (dt, kind, text[:170]))


def main():
    evs = load(sys.argv[1])
    cmd = sys.argv[2] if len(sys.argv) > 2 else 'skills'
    if cmd == 'skills':
        cmd_skills(evs, track(evs))
    elif cmd == 'hits':
        cmd_hits(evs)
    elif cmd == 'control':
        cmd_control(evs, track(evs))
    elif cmd == 'cast':
        cmd_cast(evs, track(evs))
    elif cmd == 'track':
        cmd_track(evs, track(evs))
    elif cmd == 'push':
        cmd_push(evs, track(evs))
    elif cmd == 'stacks':
        cmd_stacks(evs)
    else:
        sys.exit('未知子命令: %s' % cmd)


# ---------------------------------------------------------------- stacks

def cmd_stacks(evs):
    """某受控者在时间窗内的 buff 加/删时间线，并排标出 c78 的 N 与此刻在身上的层数
    （检验「N = 受控 buff 的在体层数/叠加计数」）。
    用法: stacks <t0> <t1> <obj十六进制> [buff 名,逗号分隔；缺省 buf_skill_musejuben_41,buf_CS_wd_tjq08]"""
    t0, t1, o = float(sys.argv[3]), float(sys.argv[4]), int(sys.argv[5], 16)
    names = sys.argv[6].split(',') if len(sys.argv) > 6 else ['buf_skill_musejuben_41', 'buf_CS_wd_tjq08']
    live = {}
    for e in evs:
        if not (t0 <= e['t'] <= t1):
            continue
        if msg(e, 169) and len(e['v']) > 2 and oid(e['v'][2]) == o and sv(e, 1) in names:
            b = sv(e, 1)
            live[b] = live.get(b, 0) + (1 if iv(e, 0) == 0 else -1)
            print('%8.3f  m169 %s %-28s 在体=%d' % (e['t'], '加' if iv(e, 0) == 0 else '删', b, live[b]))
        elif msg(e, 24) and len(e['v']) >= 6 and iv(e, 1) in (78, 79) and oid(e['v'][0]) == o:
            print('%8.3f  c%d  N=%s 尾=%s   在体层数 %s' % (e['t'], iv(e, 1), iv(e, 5), e['v'][6][1] if len(e['v']) > 6 else '', ' '.join('%s=%d' % kv for kv in live.items()) or '(0)'))
        elif e['k'] == 'prop' and e['obj'] == o and any(n == 'State' for n, _ in e['props']):
            print('%8.3f  State=%s' % (e['t'], [v[1] for n, v in e['props'] if n == 'State'][0]))


# ---------------------------------------------------------------- push

def cmd_push(evs, tr):
    """击退帧 c78[受控者, 78, 攻击者, "", "", N, 尾值] 逐例：N/尾值 对照受控者随后的位移
    （世界方位角、偏离「攻击者→受控者」连线的角度、水平距离）与同刻命中技能。
    用法: push            检验「N=方位角」假说；按受控者列 N 的先后序列（叠加计数假说）。"""
    import math
    rows, seq = [], {}
    for e in evs:
        if not (msg(e, 24) and len(e['v']) >= 6 and iv(e, 1) == 78):
            continue
        v, a, n = oid(e['v'][0]), oid(e['v'][2]), iv(e, 5)
        tail = e['v'][6][1] if len(e['v']) > 6 and e['v'][6][0] == 'f32' else ''
        skill = next((sv(f, 1) for f in burst(evs, e['t'], lambda f: msg(f, 205))
                      if oid(f['v'][0]) == a and v in [oid(x) for x in f['v'][2:]]), '')
        vb = sample(tr, v, e['t'] - 1.0, e['t'] + 0.05)
        vn = sample(tr, v, e['t'] + 0.06, e['t'] + 1.3)
        ab = sample(tr, a, e['t'] - 1.5, e['t'] + 0.3)
        base = vb[-1] if vb else (vn[0] if vn else None)
        # 终点取窗口内水平离起点最远的采样：受控结束后目标会自己走回来，
        # 取「最后一个采样」会把回程当成击退方向。
        nxt = max(vn, key=lambda s: dist(s[1], base[1])) if (base and vn) else None
        d = ang = off = gap = float('nan')
        if base and nxt:
            dx, dy, dz = (nxt[1][i] - base[1][i] for i in range(3))
            d, gap = (dx * dx + dz * dz) ** 0.5, nxt[0] - base[0]
            if d > 0.3:
                ang = math.degrees(math.atan2(dx, dz)) % 360.0
                if ab:
                    ax, az = base[1][0] - ab[-1][1][0], base[1][2] - ab[-1][1][2]
                    if ax * ax + az * az > 0.01:
                        cosv = (dx * ax + dz * az) / (d * (ax * ax + az * az) ** 0.5)
                        off = math.degrees(math.acos(max(-1.0, min(1.0, cosv))))
        rows.append((e['t'], v, a, skill, n, tail, gap, d, ang, off))
        seq.setdefault(v, []).append(n)
    print('%7s %-9s %-9s %-26s %4s %-7s %5s %6s %7s %7s' % ('t', '受控者', '攻击者', '同刻命中技能', 'N', '尾值', '采样差', '水平距', '位移方位', '偏离攻击线'))
    for t, v, a, skill, n, tail, gap, d, ang, off in rows:
        print('%7.3f %08x  %08x  %-26s %4s %-7s %5.2f %6.2f %7s %7s' % (t, v, a, skill[:26], n, tail[:7], gap, d, '%.0f' % ang if ang == ang else '-', '%.0f' % off if off == off else '-'))
    ok = [r for r in rows if r[8] == r[8]]
    if len(ok) >= 3:
        for label, f in (('位移方位角 atan2(dx,dz)', lambda r: r[8]), ('偏离攻击线角', lambda r: r[9])):
            pts = [(r[4], f(r)) for r in ok if f(r) == f(r) and isinstance(r[4], int)]
            if len(pts) >= 3:
                mx = sum(p[0] for p in pts) / len(pts)
                my = sum(p[1] for p in pts) / len(pts)
                num = sum((p[0] - mx) * (p[1] - my) for p in pts)
                den = (sum((p[0] - mx) ** 2 for p in pts) * sum((p[1] - my) ** 2 for p in pts)) ** 0.5
                print('相关系数 N ↔ %s: %.2f（%d 例）' % (label, num / den if den else float('nan'), len(pts)))
    print('--- 按受控者的 N 先后序列 ---')
    for v, ns in sorted(seq.items(), key=lambda kv: -len(kv[1]))[:12]:
        print('%08x  %s' % (v, ' '.join(str(x) for x in ns)))


if __name__ == '__main__':
    main()
