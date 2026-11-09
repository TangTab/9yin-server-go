# -*- coding: utf-8 -*-
"""主报告生成：周期通道聚合 + 每会话精选时间线（未知消息保留）。"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from timeline import pair_events, runs, wall_start, GAP  # noqa: E402

# 纯周期通道：主报告按会话聚合为一行，附录保留逐段
PERIODIC = {
    'pair:ping': 'C2S 0x18 ping → S2C 0x38 回显',
    'pair:delay': 'C2S m924 MSG_DELAY → S2C m294 回显',
    'op21:': 'S2C 0x21 对象同步帧',
    'op28:': 'S2C 0x28 批量帧(子码0x47)',
    'C2S:m54:': 'C2S m54 格挡键上报（按下/抬起）',
    'C2S:m15:state=4': 'C2S m15 移动上报·静止(state=4)',
    'cluster:status': 'S2C m586/645/646/750 状态推送簇',
    'C2S:m9': 'C2S m9 登录握手(6s 周期)',
}


def is_periodic(sig):
    return any(sig.startswith(p) or sig == p for p in PERIODIC)


def main(tags):
    out = []
    for tag in tags:
        ev = json.load(open(os.path.join(HERE, 'events', tag + '.json'), encoding='utf-8'))
        ev2 = pair_events(ev)
        rr = runs(ev2)
        # 周期通道聚合
        agg = {}
        for r in rr:
            for p, label in PERIODIC.items():
                if r['sig'].startswith(p) or r['sig'] == p:
                    a = agg.setdefault(p, {'label': label, 'n': 0, 't0': r['t0'], 't1': r['t1']})
                    a['n'] += r['n']
                    a['t0'] = min(a['t0'], r['t0'])
                    a['t1'] = max(a['t1'], r['t1'])
                    break
        keep = [r for r in rr if not is_periodic(r['sig'])]
        out.append('### %s.pcapng（%s 起，%d 条事件 → 精选 %d 段）' % (
            tag, wall_start(tag), len(ev2), len(keep)))
        out.append('')
        if agg:
            out.append('**周期通道（全时段，已合并）**：')
            for p, a in sorted(agg.items(), key=lambda kv: kv[1]['t0']):
                out.append('- %s ×%d（%.0fs–%.0fs）' % (a['label'], a['n'], a['t0'], a['t1']))
            out.append('')
        out.append('| 时间 | 方向 | 交互 |')
        out.append('|---|---|---|')
        for r in keep:
            span = '%.1fs' % r['t0'] if r['t1'] - r['t0'] < 0.15 else '%.1f–%.1fs' % (r['t0'], r['t1'])
            tail = ' ×%d' % r['n'] if r['n'] > 1 else ''
            out.append('| %s | %s | %s%s |' % (span, r['dir'], r['desc'], tail))
        out.append('')
    return '\n'.join(out)


if __name__ == '__main__':
    tags = sys.argv[1:] or list('abcdefgi')
    md = main(tags)
    path = os.path.join(HERE, 'curated.md')
    open(path, 'w', encoding='utf-8').write(md)
    print('written', path, len(md))
