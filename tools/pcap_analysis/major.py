# -*- coding: utf-8 -*-
"""骨干时间线：仅保留主报告级交互（技能/目标/轻功/对话/面板/提示/未知一次性），
周期与海量通道在表头聚合为计数行。"""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
from timeline import pair_events, runs, wall_start  # noqa: E402

# 表内保留（前缀匹配）
KEEP = (
    'C2S:m211', 'C2S:m212', 'C2S:m216', 'C2S:m217', 'C2S:m220', 'C2S:m431', 'C2S:m915',
    'C2S:m914', 'C2S:m990', 'C2S:m440', 'C2S:m107', 'C2S:m197', 'C2S:m240', 'C2S:m298',
    'C2S:m770', 'C2S:m3:', 'C2S:m721', 'C2S:m82', 'C2S:m1016', 'C2S:m819', 'C2S:m187',
    'C2S:m858', 'C2S:m876', 'C2S:m865', 'C2S:m350', 'C2S:m353', 'C2S:m356', 'C2S:m357',
    'C2S:m366', 'C2S:m423', 'C2S:m293', 'C2S:m302', 'C2S:m151', 'C2S:m965', 'C2S:m8:',
    'C2S:m421', 'C2S:m760', 'C2S:m758', 'C2S:m61', 'C2S:m949', 'C2S:m986', 'C2S:op04',
    'C2S:op07', 'C2S:op31', 'C2S:op02',
    'S2C:m4:', 'S2C:m5:', 'S2C:m6:', 'S2C:m7:', 'S2C:m8:', 'S2C:m25:', 'S2C:m26:',
    'S2C:m707', 'S2C:m19:', 'S2C:m20:', 'S2C:m337', 'S2C:m1:', 'S2C:m446', 'S2C:m249',
    'S2C:m189', 'S2C:m284', 'S2C:m603', 'S2C:m566', 'S2C:m681', 'S2C:m287', 'S2C:m639',
    'S2C:m758', 'S2C:m759', 'S2C:m98', 'S2C:m52', 'S2C:m2:', 'S2C:m64', 'S2C:m100',
    'S2C:m1016', 'S2C:m229', 'S2C:m386', 'S2C:m349', 'S2C:m250', 'S2C:m252', 'S2C:m256',
    'S2C:m259', 'S2C:m199', 'S2C:m200', 'S2C:m228', 'S2C:m235', 'S2C:m281', 'S2C:m366',
    'S2C:m368', 'S2C:m382', 'S2C:m390', 'S2C:m395', 'S2C:m414', 'S2C:m416', 'S2C:m419',
    'S2C:m430', 'S2C:m438', 'S2C:m458', 'S2C:m477', 'S2C:m496', 'S2C:m504', 'S2C:m505',
    'S2C:m561', 'S2C:m580', 'S2C:m586:', 'S2C:m59', 'S2C:m596', 'S2C:m597', 'S2C:m607',
    'S2C:m614', 'S2C:m623', 'S2C:m634', 'S2C:m640', 'S2C:m644', 'S2C:m647', 'S2C:m648',
    'S2C:m655', 'S2C:m672', 'S2C:m673', 'S2C:m679', 'S2C:m683', 'S2C:m705', 'S2C:m726',
    'S2C:m735', 'S2C:m740', 'S2C:m751', 'S2C:m756', 'S2C:m770', 'S2C:m772', 'S2C:m915',
    'S2C:pair:token', 'pair:token', 'op15:', 'op16:', 'op31', 'op0D', 'op32', 'op0A',
    'op09', 'op14', 'op29', 'op2D?:', 'op1F:', 'op2C', 'op2E',
)

AGG_LABELS = [
    ('op10:', 'S2C 0x10 属性更新'),
    ('op13:', 'S2C 0x13 记录写格'),
    ('op11:', 'S2C 0x11 记录加行'),
    ('op12:', 'S2C 0x12 记录删行'),
    ('op14', 'S2C 0x14 快照伴随帧'),
    ('op18:', 'S2C 0x18 视图加行'),
    ('op21:', 'S2C 0x21 对象同步'),
    ('op28:', 'S2C 0x28 批量帧(0x47)'),
    ('op29:', 'S2C 0x29 批量块(0x47)'),
    ('op28:', 'S2C 0x28 批量帧(0x47)'),
    ('C2S:m15:', 'C2S m15 移动上报'),
    ('C2S:m54:', 'C2S m54 格挡键上报'),
    ('C2S:m924', 'C2S m924 延迟心跳'),
    ('C2S:m9:', 'C2S m9 登录握手(6s)'),
    ('pair:ping', 'C2S 0x18 ping→0x38'),
    ('pair:delay', 'C2S m924→S2C m294'),
    ('cluster:status', 'S2C m586/645/646/750 簇'),
    ('S2C:m169:', 'S2C m169 buff 添加/移除'),
    ('S2C:m24:c1=17', 'S2C m24 c1=17 伤害飘字'),
    ('S2C:m24:c1=10', 'S2C m24 c1=10 怒气段'),
    ('S2C:m24:c1=29', 'S2C m24 c1=29 击退'),
    ('S2C:m24:c1=', 'S2C m24 其他子码'),
    ('S2C:m205:', 'S2C m205 伤害目标列表'),
    ('S2C:m18:', 'S2C m18 战斗伤害帧'),
    ('lobby-hb', '6001 大厅心跳'),
]


def main(tags):
    out = []
    for tag in tags:
        ev = json.load(open(os.path.join(HERE, 'events', tag + '.json'), encoding='utf-8'))
        ev2 = pair_events(ev)
        rr = runs(ev2)
        agg = {}
        for r in rr:
            for p, label in AGG_LABELS:
                if r['sig'].startswith(p):
                    a = agg.setdefault(label, 0)
                    agg[label] = a + r['n']
                    break
        keep = [r for r in rr if r['sig'].startswith(KEEP)]
        out.append('### %s.pcapng — %s 起' % (tag, wall_start(tag)))
        out.append('')
        if agg:
            out.append('> 已聚合通道：' + '；'.join('%s ×%d' % (k, v) for k, v in sorted(agg.items(), key=lambda kv: -kv[1])))
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
    path = os.path.join(HERE, 'major.md')
    open(path, 'w', encoding='utf-8').write(md)
    print('written', path, len(md))
    import re
    for sec in md.split('### ')[1:]:
        print(sec.split('\n')[0], '| rows:', sec.count('\n| '))
