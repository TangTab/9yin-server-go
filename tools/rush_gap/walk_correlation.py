# -*- coding: utf-8 -*-
"""事件 json → 冲锋按下/自走/受理簇三方时序对齐表（定版工具）。

对每次 C2S m211 出招：找其后 2.5s 内首条 m15 state=7 自走与 0.5s 内受理簇
（CurSkillID 帧到达），输出相对时延与先后——用于判定「按下即走（本地决策）
/ 簇到即走（服务端武装）/ 不走」。定版结论见
docs/冲锋自走触发链官服定版与差集.md §1.1。

用法：
  python -I walk_correlation.py <events.json>
"""
import json
import sys


def main():
    ev = json.load(open(sys.argv[1], encoding='utf-8'))
    presses = [e for e in ev if e['dir'] == 'C2S' and 'm211' in e['sig']]
    walks = [e for e in ev if e['dir'] == 'C2S' and 'm15' in e['sig'] and 'state=7' in e['desc']]
    clusters = [e for e in ev if e['dir'] == 'S2C' and 'CurSkillID=' in e['desc']
                and 'CS_' in e['desc']]
    print('%8s %-22s %9s %9s %s' % ('press_t', 'skill', 'walk_dt', 'clu_dt', 'mode'))
    counts = {}
    for p in presses:
        t = p['t']
        w = next((x['t'] for x in walks if t - 0.001 <= x['t'] <= t + 2.5), None)
        c = next((x['t'] for x in clusters if t <= x['t'] <= t + 0.5), None)
        wd = '%+.3f' % (w - t) if w is not None else '-'
        cd = '%+.3f' % (c - t) if c is not None else '-'
        if w is None:
            mode = '不走' + ('(无簇)' if c is None else '')
        elif c is None:
            mode = '无簇'
        elif w <= c:
            mode = '按下即走'
        elif w - t <= 0.08:
            mode = '簇到即走'
        else:
            mode = '延迟起走'
        counts[mode] = counts.get(mode, 0) + 1
        print('%8.3f %-22s %9s %9s %s' % (t, p['desc'].split()[2][:20], wd, cd, mode))
    print('\n模式统计:', dict(sorted(counts.items(), key=lambda kv: -kv[1])))


if __name__ == '__main__':
    main()
