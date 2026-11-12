# -*- coding: utf-8 -*-
"""frames pkl → 事件 json（复用 tools/pcap_analysis/decode.py 描述器，加固版）。

对每条流：S2C 先喂 0x09 表帧再用全表重解属性帧；describe 异常/空帧容忍
（老 decode.py 对登陆-all 双连接包会 IndexError 崩）。

输出事件：{t(相对秒), dir, desc, sig, len}。

用法：
  python -I pcap_events.py <frames.pkl> <out.json>
"""
import argparse
import json
import os
import pickle
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, '..', 'pcap_analysis'))
import decode  # noqa: E402  (自带 REC_NAMES/Table09 回退表)


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('in_pkl')
    ap.add_argument('out_json')
    args = ap.parse_args()
    data = pickle.load(open(args.in_pkl, 'rb'))
    events = []
    for st in data['streams']:
        label = 'S2C' if st['dir'] == 's2c' else 'C2S'
        frames = [(ts / 1e6, fr) for ts, fr in st['frames']]
        events.append({'t': frames[0][0], 'dir': 'META',
                       'desc': '%s 流 %s:%d <-> %s:%d %d帧' % (
                           label, st['key'][0], st['key'][1], st['key'][2], st['key'][3], len(frames)),
                       'sig': 'meta', 'len': 0})
        if label == 'S2C':
            tbl = decode.Table09()
            for _, fr in frames:
                if fr and fr[0] == 0x09:
                    tbl.feed(fr)
            for t, fr in frames:
                try:
                    d, s = decode.describe_s2c(fr, tbl)
                except Exception as exc:  # noqa: BLE001
                    d, s = 'S2C 解码异常 op=%02x (%s)' % (fr[0], exc), 'S2C:err'
                events.append({'t': t, 'dir': label, 'desc': d, 'sig': s, 'len': len(fr)})
        else:
            for t, fr in frames:
                try:
                    d, s = decode.describe_c2s(fr)
                except Exception as exc:  # noqa: BLE001
                    d, s = 'C2S 解码异常 op=%02x (%s)' % (fr[0], exc), 'C2S:err'
                events.append({'t': t, 'dir': label, 'desc': d, 'sig': s, 'len': len(fr)})
    events.sort(key=lambda e: e['t'])
    with open(args.out_json, 'w', encoding='utf-8') as f:
        json.dump(events, f, ensure_ascii=False)
    from collections import Counter
    c = Counter(e['sig'].split(':')[0] for e in events if e['dir'] != 'META')
    print('%s -> %d events' % (args.out_json, len(events)))
    print(c.most_common(14))


if __name__ == '__main__':
    main()
