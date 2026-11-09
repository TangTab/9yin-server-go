# -*- coding: utf-8 -*-
"""时间线构建：事件流 → 连续同类合并的交互时间线（markdown）。

合并规则：
1. 配对阶段：C2S 请求与已知应答合并为单事件（ping→0x38、m924→m294、
   m9→m586/645/646/750 登录应答簇、m721→m566）。
2. 游程合并：连续同签名且间隔 ≤3s 的事件合并为一条（×N，t0–t1）。
"""
import datetime
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
EV = os.path.join(HERE, 'events')
GAP = 3.0  # 秒


def pair_events(ev):
    """请求-应答配对：请求后 window 内向前扫描应答（允许中间隔着其他帧），配中即双双移除。"""
    used = [False] * len(ev)
    out = []
    for i, e in enumerate(ev):
        if used[i]:
            continue
        t0 = e['t'] / 1e6
        pair = None
        window = 0
        if e['dir'] == 'C2S' and e['sig'] == 'C2S:op18':
            pair, window = 'pair:ping', 0.6
        elif e['dir'] == 'C2S' and e['sig'] == 'C2S:m924':
            pair, window = 'pair:delay', 1.0
        elif e['dir'] == 'C2S' and e['sig'] == 'C2S:m721':
            pair, window = 'pair:token', 0.5
        if pair:
            want = {'pair:ping': 'op38', 'pair:delay': 'S2C:m294', 'pair:token': 'S2C:m566'}[pair]
            for j in range(i + 1, len(ev)):
                if ev[j]['t'] / 1e6 - t0 > window:
                    break
                if used[j]:
                    continue
                if ev[j]['dir'] == 'S2C' and ev[j]['sig'] == want:
                    used[j] = True
                    used[i] = True
                    if pair == 'pair:ping':
                        desc = 'C2S ping(0x18) → S2C 0x38 回显'
                    elif pair == 'pair:delay':
                        desc = 'C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …]'
                    else:
                        desc = 'C2S m721 令牌串(40字符) ↔ S2C m566 应答串(32字符)（握手对，m566 实先于 m721 到达）'
                    out.append({'t': e['t'], 'dir': 'PAIR', 'sig': pair, 'desc': desc})
                    break
            else:
                out.append(e)
                continue
            continue
        if e['dir'] == 'S2C' and e['sig'] == 'S2C:m586':
            # m586/645/646/750 四连状态推送簇捆绑为单事件（与 m9 无关：m9 为 6s 周期请求）
            grp = [e['desc']]
            used[i] = True
            for j in range(i + 1, len(ev)):
                if ev[j]['t'] / 1e6 - t0 > 0.2:
                    break
                if used[j]:
                    continue
                if ev[j]['dir'] == 'S2C' and ev[j]['sig'] in ('S2C:m645', 'S2C:m646', 'S2C:m750'):
                    used[j] = True
                    grp.append(ev[j]['desc'])
            if len(grp) >= 2:
                out.append({'t': e['t'], 'dir': 'S2C', 'sig': 'cluster:status',
                            'desc': '状态推送簇 m586/645/646/750: %s' % ' | '.join(g.split('] ')[-1] for g in grp)})
                continue
            used[i] = False
        out.append(e)
    return out


# 周期噪声签名（前缀匹配）：各自聚成游程，且不打断主流程游程
NOISE_PREFIX = ('pair:ping', 'pair:delay', 'lobby-hb', 'op21:')


def is_noise(sig):
    return sig.startswith(NOISE_PREFIX)


def _run_merge(ev):
    """连续同签名 ≤GAP 合并（朴素游程）。"""
    out = []
    cur = None
    for e in ev:
        t = e['t'] / 1e6
        if cur and e['sig'] == cur['sig'] and t - cur['t1'] <= GAP:
            cur['n'] += 1
            cur['t1'] = t
            continue
        if cur:
            out.append(cur)
        cur = {'sig': e['sig'], 'dir': e['dir'], 'desc': e['desc'],
               't0': t, 't1': t, 'n': 1, 'lens': [e.get('len', 0)]}
    if cur:
        out.append(cur)
    return out


def runs(ev):
    """三路合并：每个噪声签名各自独立游程合并；非噪声流同签名游程 ≤GAP 合并。"""
    noise_runs = []
    seen = set()
    for e in ev:
        if is_noise(e['sig']) and e['sig'] not in seen:
            seen.add(e['sig'])
            noise_runs.extend(_run_merge([x for x in ev if x['sig'] == e['sig']]))
    main_runs = _run_merge([e for e in ev if not is_noise(e['sig'])])
    return sorted(noise_runs + main_runs, key=lambda r: r['t0'])


def fmt_run(r):
    span = '%.1fs' % r['t0'] if r['t1'] - r['t0'] < 0.15 else '%.1f–%.1fs' % (r['t0'], r['t1'])
    tail = ' ×%d' % r['n'] if r['n'] > 1 else ''
    return '| %s | %s | %s%s |' % (span, r['dir'], r['desc'], tail)


META = {
    'a': ('49.4.14.150:7074 侠骨丹心', '冲锋/范围技打怪 + 区域boss进度 + 打坐 + 末尾设置上报'),
    'b': ('103.143.19.173:7071', 'kfkj02/03/04/05 冲锋连打（含空中/无目标形态）'),
    'c': ('49.4.14.150:7074（与 a 同客户端口 60179，重连会话）', '重连全量快照 + UI/系统功能遍历探测（属性商城等表单事件）'),
    'd': ('103.143.19.173:7071（与 b/e 同客户端口 52396）', '短会话：mzjf02/06 出招两次'),
    'e': ('103.143.19.173:7071', '长战斗会话：mzjf/kfkj/lyfh/hyjf 全家桶 + 冲锋落地循环'),
    'f': ('49.4.14.150:7074', 'kfkj02 十连按拉回剧本（docs 已详析：选怪→受理→拉回→冲刺→命中）'),
    'g': ('103.143.19.173:7072（与 i 同端口，前半段）', 'kfkj 战斗 + 击杀 + NPC 对话（ui_npcclonetl）'),
    'i': ('103.143.19.173:7072（g 的续段，中途回抓）', 'hyjf07 十二次连发 + 冲刺/伤害/怒气全链'),
}


def wall_start(tag):
    sys.path.insert(0, HERE)
    from pcaplib import read_pcapng
    path = r'E:\Workspace\project\game\adata\official\%s.pcapng' % tag
    for ts, lt, pkt in read_pcapng(path):
        if lt is not None:
            # pcapng 时间戳按本机时区解读（抓包机=UTC+8）
            return datetime.datetime.fromtimestamp(ts / 1e6).strftime('%Y-%m-%d %H:%M:%S')
    return ''


def main(tags):
    lines = []
    for tag in tags:
        ev = json.load(open(os.path.join(EV, tag + '.json'), encoding='utf-8'))
        wall = '抓包起点 %s（本地墙钟）' % wall_start(tag)
        ev2 = pair_events(ev)
        rr = runs(ev2)
        lines.append('## %s.pcapng — %s' % (tag, META.get(tag, ('', ''))[0]))
        lines.append('')
        lines.append('场景：%s。事件 %d 条 → 合并为 %d 段。%s' % (META.get(tag, ('', ''))[1], len(ev), len(rr), wall))
        lines.append('')
        lines.append('| 时间 | 方向 | 交互 |')
        lines.append('|---|---|---|')
        for r in rr:
            lines.append(fmt_run(r))
        lines.append('')
    return '\n'.join(lines)


if __name__ == '__main__':
    tags = sys.argv[1:] or list('abcdefgi')
    md = main(tags)
    out = os.path.join(HERE, 'timeline.md')
    with open(out, 'w', encoding='utf-8') as f:
        f.write(md)
    print('written', out, len(md), 'bytes')
