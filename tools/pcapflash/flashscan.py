# flashscan.py —— 寒心恨雪（闪击通道技）的抓包分析：逐次施放的位移、逐帧时间线、
# m914 位置回报与回显。纯标准库；只读抓包与解码文本，不写任何文件。
#
# 两步用法：
#   1) 解码（仓库 tools/pcapevents，开全帧 hex；需要带 trimIPTotal 的版本，见
#      子命令 padding 的说明）：
#        NINEYIN_PCAP_HEXALL=1 NINEYIN_PCAP_HEX=4000 \
#          go run ./tools/pcapevents <抓包.pcapng> <服务器端口> dumprec > dump.txt
#   2) 分析：
#        python -I flashscan.py flows    <抓包.pcapng>...        抓包里的 TCP 流（找服务器端口）
#        python -I flashscan.py padding  <抓包.pcapng> <端口>     纯 ACK 以太网填充体检（dump 可信度）
#        python -I flashscan.py casts    <dump.txt>               寒心恨雪 m211 总表
#        python -I flashscan.py waves    <dump.txt>               逐跳位移表（落点 / 朝向 / 时延 / 回显）
#        python -I flashscan.py cast     <dump.txt> <t> [秒数=4]  某次施放的帧时间线（相对 m211 的毫秒）
#        python -I flashscan.py timeline <dump.txt> <t0> <t1> [正则] [行宽=400]  窗口内逐帧行
#        python -I flashscan.py m914     <dump.txt>...            全部 m914 的类型码 / 计数位
#        python -I flashscan.py echo     <dump.txt> [t0 t1]       m914 之后玩家位置被回显的容器与时延
#
# 其它闪击技加 --skill <技能 id>（默认 CS_jh_hyjf08）。定版与取证见
# docs/combat/寒心恨雪与太极剑舞引导机制.md。
#
# dump 行格式（tools/pcapevents dumprec）：
#   H t=%9.3f C2S|S2C #序号 op=0x%02X len=%d hex=<整帧明文，去掉 EE 转义与终止符>
#   A t=%9.3f C2S|S2C #序号 custom m<号> <值…> / actenv m<号> <值…>
#   A … 0x10 obj=%08x: 名=值 …            属性帧
#   A … 0x2D k=N {obj 名=值 …} …         多对象属性帧
#   A … op=0x21 len=N n=K {obj dst=x,y,z o=… spd=… rot=… m2=… m3=… f=N} …
#   A … op=0x2C len=N n=K {obj x,y,z o=…} … phex=<明文体 hex>
#   A … op=0x1F len=N obj=%08x own=%08x x=… y=… z=… orient=…
#   A … rec op=0x1N len=N obj=%08X/%04X <记录名> …  记录族帧
import math
import re
import struct
import sys
from collections import Counter
from datetime import datetime

sys.stdout.reconfigure(encoding='utf-8', errors='replace')  # 控制台非 UTF-8 时避免打印失败

WAVE_SUBCODE = 71  # m24 子码 71 = 客户端 player_flash_attack
EPS = 0.002        # 位置相等的容差（米）：float32 坐标经文本格式化后的舍入量级

T = re.compile(r'^A t=\s*([0-9.]+) (C2S|S2C) #(\d+) (.*)$')
H = re.compile(r'^H t=\s*([0-9.]+) \S+ #(\d+) op=0x([0-9A-F]{2}) len=(\d+) hex=((?:[0-9a-f]{2} ?)+)$')
PH = re.compile(r' phex=((?:[0-9a-f]{2} ?)+)')
VAL = re.compile(r'raw\([0-9a-f ]+?\)|str\("(?:[^"\\]|\\.)*"\)|[a-z0-9]+\([^)]*\)')


# ---------------------------------------------------------------- 抓包层

def pcapng_ipv4_tcp(path):
    """逐个 EPB 产出 (到达秒, 源端口, 目的端口, ip 头长, ip 总长, 整个 IPv4 包)。"""
    d = open(path, 'rb').read()
    off, link = 0, 1
    while off + 12 <= len(d):
        bt, bl = struct.unpack_from('<II', d, off)
        if bl < 12 or off + bl > len(d):
            break
        body = d[off + 8:off + bl - 4]
        if bt == 1:
            link = struct.unpack_from('<H', body)[0]
        elif bt == 6 and link == 1:
            cap = struct.unpack_from('<I', body, 12)[0]
            pkt = body[20:20 + cap]
            if len(pkt) >= 34 and struct.unpack_from('>H', pkt, 12)[0] == 0x0800:
                ip = pkt[14:]
                if len(ip) >= 20 and ip[0] >> 4 == 4 and ip[9] == 6:
                    yield ((struct.unpack_from('<I', body, 4)[0] << 32 | struct.unpack_from('<I', body, 8)[0]) / 1e6,
                           struct.unpack_from('>H', ip, 2)[0], (ip[0] & 0xf) * 4, ip)
        off += bl


def cmd_flows(paths):
    """每个抓包的 TCP 流（包数 / 载荷字节 / 首末时间），按载荷字节降序。"""
    for path in paths:
        flows, span = {}, [None, None]
        for ts, total, ihl, ip in pcapng_ipv4_tcp(path):
            doff = (ip[ihl + 12] >> 4) * 4
            key = (struct.unpack_from('>H', ip, ihl)[0], struct.unpack_from('>H', ip, ihl + 2)[0])
            f = flows.setdefault(key, [0, 0, ts, ts])
            f[0], f[1], f[2], f[3] = f[0] + 1, f[1] + max(0, total - ihl - doff), min(f[2], ts), max(f[3], ts)
            span = [ts if span[0] is None else min(span[0], ts), ts if span[1] is None else max(span[1], ts)]
        fmt = lambda t: datetime.fromtimestamp(t).strftime('%H:%M:%S.%f')[:-3]
        print('== %s  %s ~ %s（%.0fs）' % (path, fmt(span[0]) if flows else '-', fmt(span[1]) if flows else '-',
                                      span[1] - span[0] if flows else 0))
        for (sp, dp), (n, b, t0, t1) in sorted(flows.items(), key=lambda kv: -kv[1][1])[:6]:
            print('   %5d ↔ %5d  包=%-6d 载荷=%-8d %s ~ %s' % (sp, dp, n, b, fmt(t0), fmt(t1)))


def cmd_padding(path, port):
    """纯 ACK 以太网填充体检：IPv4 总长之外还捕获到零字节的报文数。
    以太网最小帧填充不属于 IP 包。重组若不按 IP 总长裁剪，纯 ACK 的填充会被当成
    载荷，与紧随其后的真数据同 seq，真数据的头几字节被当重传砍掉（旧版
    tools/pcapevents 的 op=0xCC「零头帧」即此）。数量非零的抓包，解码必须用带
    trimIPTotal 的 pcapevents。"""
    stats = {'S2C': [0, 0, 0, 0], 'C2S': [0, 0, 0, 0]}  # 报文 / 纯 ACK / 带零填充的纯 ACK / 带零填充的有载荷包
    for ts, total, ihl, ip in pcapng_ipv4_tcp(path):
        sp, dp = struct.unpack_from('>H', ip, ihl)[0], struct.unpack_from('>H', ip, ihl + 2)[0]
        direction = 'S2C' if sp == port else 'C2S' if dp == port else None
        if direction is None:
            continue
        doff = (ip[ihl + 12] >> 4) * 4
        payload = max(0, total - ihl - doff)
        s = stats[direction]
        s[0] += 1
        s[1] += payload == 0
        if len(ip) - ihl - doff > payload and not any(ip[ihl + doff + payload:]):
            s[2 if payload == 0 else 3] += 1
    for direction, (frames, acks, padded_acks, padded_data) in stats.items():
        print('%s：报文 %d，纯 ACK %d；带零填充 %d（纯 ACK %d + 有载荷的小包 %d）' % (
            direction, frames, acks, padded_acks + padded_data, padded_acks, padded_data))
    if any(s[2] + s[3] for s in stats.values()):
        print('→ 有填充：解码须用带 trimIPTotal 的 tools/pcapevents，否则纯 ACK 之后的真数据帧会丢头 N 字节。')


# ---------------------------------------------------------------- 解码文本层

def tag(a, i):
    """TVarList 单值 → (类型, 值, 下一偏移)。类型：i32/i64/f32/f64/str/wide/obj。"""
    t = a[i]
    i += 1
    if t in (2, 4):
        v = struct.unpack_from('<i' if t == 2 else '<f', a, i)[0]
        return ('i32' if t == 2 else 'f32'), v, i + 4
    if t in (3, 5):
        v = struct.unpack_from('<q' if t == 3 else '<d', a, i)[0]
        return ('i64' if t == 3 else 'f64'), v, i + 8
    if t in (6, 7):
        n = struct.unpack_from('<I', a, i)[0]
        i += 4
        raw = a[i:i + n]
        if t == 7:
            units = struct.unpack_from('<%dH' % (len(raw) // 2 - 1), raw, 0)
            return 'wide', ''.join(chr(u) for u in units), i + n
        return 'str', raw[:-1].decode('gbk', 'replace'), i + n
    if t == 8:
        return 'obj', struct.unpack_from('<II', a, i), i + 8
    raise ValueError('bad tag %d' % t)


def tvarlist(a, off):
    n = struct.unpack_from('<H', a, off)[0]
    off += 2
    out = []
    for _ in range(n):
        k, v, off = tag(a, off)
        out.append((k, v))
    return out


def dest_entries(a, off):
    """0x21 / 0x2C 的 44 字节记录：[obj][own][x][y][z][朝向][速度][rot][m2][m3][标志]。"""
    if len(a) < off + 2:
        return []
    n = struct.unpack_from('<H', a, off)[0]
    out, o = [], off + 2
    for _ in range(n):
        if o + 44 > len(a):
            break
        obj, own, x, y, z, ori, spd, rot, m2, m3, fl = struct.unpack_from('<IIffffffffI', a, o)
        out.append({'obj': obj, 'xyz': (x, y, z), 'o': ori, 'spd': spd})
        o += 44
    return out


def load(path):
    """解码文本 → 事件列表。自定义消息直接从 H 行的整帧明文原生解 TVarList（全精度）。
    事件键：t / dir / idx / op / raw / body / k（prop·rec·dest·pos·loc·other）；
    自定义消息另有 n（消息号）与 v（值表，不含消息号）。"""
    evs, pending = [], None
    for line in open(path, encoding='utf-8', errors='replace'):
        m = H.match(line)
        if m:
            pending = (int(m.group(2)), int(m.group(3), 16), bytes.fromhex(m.group(5)))
            continue
        m = T.match(line)
        if not m:
            continue
        ev = {'t': float(m.group(1)), 'dir': m.group(2), 'idx': int(m.group(3)), 'body': m.group(4)}
        # H 行紧挨在它的 A 行前面（两方向序号各自从 0 起，只能顺序配对）
        if pending and pending[0] == ev['idx']:
            ev['op'], ev['raw'], pending = pending[1], pending[2], None
        else:
            ev['op'], ev['raw'] = -1, b''
        b = ev['body']
        if b.startswith('0x10 '):
            ev['k'] = 'prop'
            ev['obj'] = int(re.search(r'obj=([0-9a-f]{8})', b).group(1), 16)
            ev['props'] = [(p.split('=', 1)[0], p.split('=', 1)[1]) for p in b.split(':', 1)[1].split(' ') if '=' in p]
        elif b.startswith('rec '):
            ev['k'] = 'rec'
        elif b.startswith('op=0x21 ') and ev['raw']:
            ev['k'], ev['entries'] = 'dest', dest_entries(ev['raw'], 1)
        elif b.startswith('op=0x2C ') and PH.search(b):
            ev['k'], ev['entries'] = 'pos', dest_entries(bytes.fromhex(PH.search(b).group(1)), 0)
        elif b.startswith('op=0x1F '):
            ev['k'] = 'loc'
            g = re.search(r'obj=([0-9a-f]{8}) own=([0-9a-f]{8}) x=(\S+) y=(\S+) z=(\S+) orient=(\S+)', b)
            ev['obj'], ev['xyz'], ev['o'] = int(g.group(1), 16), tuple(float(g.group(i)) for i in (3, 4, 5)), float(g.group(6))
        else:
            ev['k'] = 'other'
        if ev['raw'] and ev['op'] in (0x1E, 0x27, 0x0A):
            off = 21 if ev['op'] == 0x0A else 1  # 0x0A = 活动信封：16B 会话区 + u32 序列号 + 值表
            if len(ev['raw']) > off + 2:
                try:
                    vl = tvarlist(ev['raw'], off)
                    if vl and vl[0][0] in ('i32', 'i64'):
                        ev['n'], ev['v'] = vl[0][1], vl[1:]
                except (ValueError, struct.error):
                    pass
        evs.append(ev)
    return evs


def fmt(ev, width=360):
    """一帧的可读行；raw(8 字节) 的对象对解成 obj(id/own)。"""
    if ev.get('n') is not None:
        def one(k, v):
            if k == 'obj':
                return 'obj(%08x/%07x)' % v
            if k == 'str':
                return repr(v)
            return '%.6g' % v if isinstance(v, float) else str(v)
        s = 'm%d %s' % (ev['n'], ' '.join(one(k, v) for k, v in ev['v']))
    else:
        def dec(m):
            bs = [int(x, 16) for x in m.group(0)[4:-1].split(' ')]
            return 'obj(%08x/%07x)' % (int.from_bytes(bs[:4], 'little'), int.from_bytes(bs[4:], 'little')) if len(bs) == 8 else m.group(0)
        s = VAL.sub(lambda m: dec(m) if m.group(0).startswith('raw(') else m.group(0), ev['body'])
    s = re.sub(r'\s+', ' ', s)
    return '%8.3f %s #%-5d %s' % (ev['t'], 'S' if ev['dir'] == 'S2C' else 'C', ev['idx'], s if len(s) <= width else s[:width] + ' …')


# ---------------------------------------------------------------- 通道技

def casts(evs, skill):
    """全部 C2S m211 [技能, x, y, z, 朝向, 0, 0, 0, "", [目标, 目标位 xyz]]。"""
    return [e for e in evs if e['dir'] == 'C2S' and e.get('n') == 211 and any(v == skill for _, v in e['v'])]


def waves_of(evs, cast):
    """一次施放的波帧：S2C m24 子码 71。值表 = [主体, 71, 目标, "", 1, 施法者, 速度, 0, 动作…]。"""
    return [e for e in evs if cast['t'] < e['t'] <= cast['t'] + 8 and e['dir'] == 'S2C' and e.get('n') == 24
            and len(e['v']) > 1 and e['v'][1][1] == WAVE_SUBCODE and e['v'][0][1][0] != 0]


def wave_victim(wave):
    """波帧里的目标对象：首波 11 值 [目标, 71, 目标, "", 1, 施法者, 速度, 0, "", 0, ""]，
    其后 8 值 [上一目标, 71, 目标, "", 1, 施法者, 速度, 0, 动作名]——第 4 值是字符串，
    目标在它前面。"""
    v = wave['v']
    return v[3][1][0] if v[3][0] == 'obj' else v[2][1][0]


def track(evs):
    """对象实位采样 {obj: [(t, xyz)]}：0x1F / 0x2C 实位 + 0x21 目的地。"""
    tr = {}
    for e in evs:
        if e['k'] == 'loc':
            tr.setdefault(e['obj'], []).append((e['t'], e['xyz']))
        elif e['k'] in ('pos', 'dest'):
            for g in e['entries']:
                tr.setdefault(g['obj'], []).append((e['t'], g['xyz']))
    return tr


def cmd_casts(evs, skill):
    cs = casts(evs, skill)
    print('%s 的 m211 共 %d 次' % (skill, len(cs)))
    print('%-9s %-6s %-30s %-9s %-40s %-7s %s' % ('t', '序号', '施放位', '朝向', '锁定目标 @ 目标位', '波/上报', '首波 / 末波（相对 m211，含往返时延）'))
    for c in cs:
        v = c['v']
        w = waves_of(evs, c)
        reports = [e for e in evs if c['t'] < e['t'] <= c['t'] + 8 and e['dir'] == 'C2S' and e.get('n') == 914]
        target = '%08x @ (%.3f,%.3f,%.3f)' % (v[9][1][0], v[10][1], v[11][1], v[12][1]) if len(v) > 12 else '（无）'
        span = '%s / %s' % (('%+dms' % round((w[0]['t'] - c['t']) * 1000)) if w else '吞',
                            ('%+dms' % round((w[-1]['t'] - c['t']) * 1000)) if w else '-')
        print('%-9.3f #%-5d (%.3f,%.3f,%.3f) %+8.4f  %-40s %d / %d     %s' % (
            c['t'], c['idx'], v[1][1], v[2][1], v[3][1], v[4][1], target, len(w), len(reports), span))


def cmd_waves(evs, skill):
    """逐跳位移表 + 汇总：m914 时延对跳距的最小二乘拟合（斜率的倒数 = 客户端冲刺速度）。"""
    tr = track(evs)
    samples, gaps, echoes = [], [], Counter()
    print('%-3s %-6s %-9s %-20s %-7s %-32s %6s %6s %6s  %s' % ('波', 'dt', '目标', '动作名', 'm914', 'm914 位置（落点）', '跳距', 'Δyaw', '偏差', '回显'))
    for c in casts(evs, skill):
        waves = waves_of(evs, c)
        if not waves:
            continue
        v = c['v']
        print('施放 %.3f：自身 (%.3f,%.3f,%.3f)  锁定目标 %08x' % (c['t'], v[1][1], v[2][1], v[3][1], v[9][1][0] if len(v) > 9 else 0))
        prev = (v[1][1], v[2][1], v[3][1])
        for i, wave in enumerate(waves):
            wv = wave['v']
            victim = wave_victim(wave)
            caster = wv[5][1][0] if len(wv) > 5 and wv[5][0] == 'obj' else 0
            # 动作名 = tm_action 槽（第 9 个值）：后续波在这里，首波是空串、其后还跟着
            # tm_back_dist(0.0) 与 tm_behit_action("")。
            action = wv[8][1] if len(wv) > 8 and wv[8][0] == 'str' else None
            label = repr(action) if action else "'' + 0 + ''" if len(wv) > 9 else "''"
            report = next((x for x in evs if wave['t'] - 0.05 <= x['t'] <= wave['t'] + 0.6 and x['dir'] == 'C2S' and x.get('n') == 914), None)
            head = '%-3d %-6s %08x %-20s' % (i + 1, '%dms' % round((wave['t'] - c['t']) * 1000), victim, label[:20])
            if report is None:
                print(head + ' (无 m914)')
                continue
            p = tuple(x[1] for x in report['v'][:3])
            jump = math.dist(prev, p)
            dyaw = (math.atan2(p[0] - prev[0], p[2] - prev[2]) - report['v'][3][1] + math.pi) % (2 * math.pi) - math.pi
            known = [s for s in tr.get(victim, ()) if s[0] <= wave['t']]
            gap = math.dist(p, known[-1][1]) if known else float('nan')
            hits = [y for y in evs if wave['t'] <= y['t'] <= wave['t'] + 1.5 and y['dir'] == 'S2C' and y['k'] in ('dest', 'pos')
                    and any(g['obj'] == caster and all(abs(a - b) < EPS for a, b in zip(g['xyz'], p)) for g in y['entries'])]
            kind = '+'.join(sorted({'0x21' if y['k'] == 'dest' else '0x2C' for y in hits})) or '无'
            delay = round((report['t'] - wave['t']) * 1000)
            samples.append((jump, delay))
            gaps.append(gap)
            echoes[kind] += 1
            print('%s %+5dms (%.3f,%.3f,%.3f) o=%+.4f %6.3f %+6.4f %6.3f  %s' % (
                head + ' ', delay, p[0], p[1], p[2], report['v'][3][1], jump, dyaw, gap,
                kind + ('@%d~%dms' % (round((hits[0]['t'] - wave['t']) * 1000), round((hits[-1]['t'] - wave['t']) * 1000)) if hits else '')))
            prev = p
    if len(samples) >= 3:
        n = len(samples)
        mx, my = sum(s[0] for s in samples) / n, sum(s[1] for s in samples) / n
        den = sum((s[0] - mx) ** 2 for s in samples)
        slope = sum((s[0] - mx) * (s[1] - my) for s in samples) / den if den else float('nan')
        finite = sorted(g for g in gaps if g == g)
        distances = sorted(s[0] for s in samples)
        median = lambda xs: (xs[(len(xs) - 1) // 2] + xs[len(xs) // 2]) / 2
        print('--- %d 跳：m914 时延(ms) ≈ %.1f + %.2f × 跳距(m)，即 %.1f m/s；跳距中位 %.2f m、最大 %.2f m；'
              '落点与目标实位偏差中位 %.4f m；回显容器 %s' % (
                  n, my - slope * mx, slope, 1000 / slope if slope else float('nan'),
                  median(distances), distances[-1], median(finite) if finite else float('nan'),
                  '、'.join('%s×%d' % kv for kv in echoes.most_common())))


def cmd_cast(evs, skill, t, seconds):
    c = min(casts(evs, skill), key=lambda c: abs(c['t'] - t))
    for e in evs:
        if c['t'] - 0.15 <= e['t'] <= c['t'] + seconds:
            print('%+8.1f %s' % ((e['t'] - c['t']) * 1000, fmt(e)))


def cmd_timeline(evs, t0, t1, pattern, width):
    flt = re.compile(pattern) if pattern else None
    for e in evs:
        if t0 <= e['t'] <= t1 and (not flt or flt.search(e['body'])):
            print(fmt(e, width))


def cmd_m914(paths):
    """全部 m914 [x, y, z, 朝向, 类型, 计数…]：类型 12 = 技能位移到达（计数位是角色当时的
    skill_lock_count），类型 2 另带上报对象与名字。"""
    for path in paths:
        evs = load(path)
        tails, counts = Counter(), []
        for e in evs:
            if e['dir'] == 'C2S' and e.get('n') == 914:
                tail = tuple('%s:%s' % kv for kv in e['v'][4:])
                tails[tail] += 1
                if e['v'][4] == ('i32', 12) and len(e['v']) > 5:
                    counts.append(e['v'][5][1])
        print('== %s：m914 × %d' % (path, sum(tails.values())))
        for tail, n in tails.most_common(12):
            print('   尾参 %s ×%d' % (' '.join(tail), n))
        if counts:
            print('   类型 12 的计数位序列：%s' % ' '.join(map(str, counts[:80])))


def cmd_echo(evs, t0, t1):
    """每次 m914 之后 1.2s 内，玩家带着「同一位置」出现在哪些 S2C 容器里（0x21 移动
    广播 / 0x2C 压缩批量位置）。玩家 = S2C 0x10 属性帧最多的对象。"""
    player = Counter(e['obj'] for e in evs if e['k'] == 'prop').most_common(1)[0][0]
    stat, delays = Counter(), []
    for e in evs:
        if e['dir'] != 'C2S' or e.get('n') != 914 or (t0 is not None and not t0 <= e['t'] <= t1):
            continue
        p = tuple(x[1] for x in e['v'][:3])
        hits = [y for y in evs if e['t'] < y['t'] <= e['t'] + 1.2 and y['dir'] == 'S2C' and y['k'] in ('dest', 'pos')
                and any(g['obj'] == player and all(abs(a - b) < EPS for a, b in zip(g['xyz'], p)) for g in y['entries'])]
        kind = '+'.join(sorted({'0x21' if y['k'] == 'dest' else '0x2C' for y in hits}))
        stat[kind or '(无)'] += 1
        if hits:
            delays.append(round((hits[0]['t'] - e['t']) * 1000))
    print('玩家 %08x；m914 × %d' % (player, sum(stat.values())))
    for kind, n in stat.most_common():
        print('   %-10s %d' % (kind, n))
    if delays:
        ordered = sorted(delays)
        print('   首个回显的时延：中位 %dms，范围 %d~%dms' % (ordered[len(ordered) // 2], ordered[0], ordered[-1]))


def main():
    args = sys.argv[1:]
    skill = 'CS_jh_hyjf08'
    if '--skill' in args:
        i = args.index('--skill')
        skill, args = args[i + 1], args[:i] + args[i + 2:]
    if len(args) < 2:
        sys.exit(__doc__)
    cmd, rest = args[0], args[1:]
    if cmd == 'flows':
        cmd_flows(rest)
    elif cmd == 'padding':
        cmd_padding(rest[0], int(rest[1]))
    elif cmd == 'm914':
        cmd_m914(rest)
    elif cmd in ('casts', 'waves', 'cast', 'timeline', 'echo'):
        evs = load(rest[0])
        if cmd == 'casts':
            cmd_casts(evs, skill)
        elif cmd == 'waves':
            cmd_waves(evs, skill)
        elif cmd == 'cast':
            cmd_cast(evs, skill, float(rest[1]), float(rest[2]) if len(rest) > 2 else 4.0)
        elif cmd == 'timeline':
            cmd_timeline(evs, float(rest[1]), float(rest[2]), rest[3] if len(rest) > 3 else None,
                         int(rest[4]) if len(rest) > 4 else 400)
        else:
            cmd_echo(evs, float(rest[1]) if len(rest) > 2 else None, float(rest[2]) if len(rest) > 2 else None)
    else:
        sys.exit('未知子命令: %s\n%s' % (cmd, __doc__))


if __name__ == '__main__':
    main()
