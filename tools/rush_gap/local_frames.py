# -*- coding: utf-8 -*-
r"""本地服 tap 裸流（captures/local/<prefix>_{c2s,s2c}.bin）→ 事件 json。

tap 写的是线上原字节流（EE 转义未去），帧解码与 pcaplib 同源。裸流无
时间戳——事件 t 用帧序伪时间（c2s/s2c 各自独立递增，1 帧=1ms 伪刻度），
只可看帧序与内容，不可做毫秒级时序结论。

用法：
  python -I local_frames.py <captures/local/127-0-0-1-51859 前缀> <out.json>
"""
import argparse
import json
import os
import pickle
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, '..', 'pcap_analysis'))
import pcaplib  # noqa: E402
import decode  # noqa: E402


def load_frames(path):
    data = open(path, 'rb').read()
    if not data:
        return []
    return [fr for fr in pcaplib.decode_frames(data) if fr]


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('prefix', help='会话前缀，如 E:\\...\\captures\\local\\127-0-0-1-51859')
    ap.add_argument('out_json')
    ap.add_argument('--pkl', default='', help='附带输出 frames pkl（ts=帧序）')
    args = ap.parse_args()
    events = []
    streams = []
    for d, label in (('c2s', 'C2S'), ('s2c', 'S2C')):
        frames = load_frames(args.prefix + '_' + d + '.bin')
        if not frames:
            continue
        streams.append({'dir': d, 'frames': [(i * 1000, fr) for i, fr in enumerate(frames)]})
        if label == 'S2C':
            tbl = decode.Table09()
            for fr in frames:
                if fr and fr[0] == 0x09:
                    tbl.feed(fr)
            for i, fr in enumerate(frames):
                try:
                    desc, sig = decode.describe_s2c(fr, tbl)
                except Exception as exc:  # noqa: BLE001
                    desc, sig = 'S2C 解码异常 op=%02x (%s)' % (fr[0], exc), 'S2C:err'
                events.append({'t': i / 1000, 'dir': label, 'desc': desc, 'sig': sig, 'len': len(fr)})
        else:
            for i, fr in enumerate(frames):
                try:
                    desc, sig = decode.describe_c2s(fr)
                except Exception as exc:  # noqa: BLE001
                    desc, sig = 'C2S 解码异常 op=%02x (%s)' % (fr[0], exc), 'C2S:err'
                events.append({'t': i / 1000, 'dir': label, 'desc': desc, 'sig': sig, 'len': len(fr)})
    with open(args.out_json, 'w', encoding='utf-8') as f:
        json.dump(events, f, ensure_ascii=False)
    if args.pkl:
        pickle.dump({'t0': 0, 'streams': streams}, open(args.pkl, 'wb'))
    from collections import Counter
    c = Counter(e['dir'] for e in events)
    print('%s: %d 帧 c2s=%d s2c=%d' % (args.prefix, len(events), c['C2S'], c['S2C']))
    print(Counter(e['sig'].split(':')[0] for e in events).most_common(12))


if __name__ == '__main__':
    main()
