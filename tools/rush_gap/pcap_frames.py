# -*- coding: utf-8 -*-
"""pcapng → 重组抽帧 → frames pkl（冲锋自走差集取证工具链第一步）。

TCP 按 seq 重组（重复/重叠去重、洞以零填充封顶 4MB）+ ISN 回退 >1MB 切
epoch（同四元组两条连接混流，登陆-all.pcapng 7074 实证）。帧解码与
pcaplib 同源（EE EE 切帧 → 去转义 → KEY 异或）。

输出 pkl 结构：
  {'t0': 首帧绝对us,
   'streams': [{'key': [srv_ip, srv_port, cli_ip, cli_port], 'dir': 's2c'|'c2s',
                'frames': [(相对us, frame_bytes)]}]}

用法：
  python -I pcap_frames.py <capture.pcapng> <out.pkl> [--ports 7071,7072,7074]
"""
import argparse
import os
import pickle
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, '..', 'pcap_analysis'))
import pcaplib  # noqa: E402

LOBBY_PORTS = {6001, 7061, 7064, 7070, 8030, 8031, 32001}


def split_epochs(pkts):
    """seq 向后回退 >1MB 视为新连接（op46_profile_stitch 同款语义）。"""
    epochs, cur, exp = [], [], None
    for ts, seq, pl in pkts:
        if exp is not None and seq < exp - (1 << 20):
            epochs.append(cur)
            cur, exp = [], None
        cur.append((ts, seq, pl))
        if exp is None or seq >= exp:
            exp = max(exp or 0, seq + len(pl))
    if cur:
        epochs.append(cur)
    return epochs


def reassemble_ts(pkts):
    """epoch 内按 seq 排序重组；重叠去重取新尾部；洞以零填充。

    返回 (buf, [(start_off, ts)])——marks 供帧时间戳归属。
    """
    if not pkts:
        return b'', []
    out = bytearray()
    marks = []
    next_seq = None
    for ts, seq, pl in sorted(pkts, key=lambda x: x[1]):
        if next_seq is None:
            next_seq = seq
        rel = seq - next_seq
        if rel < 0:
            overlap = -rel
            pl = pl[overlap:] if overlap < len(pl) else b''
            rel = 0
        if rel > 0:
            out += b'\x00' * min(rel, 1 << 22)
            next_seq += rel
        if pl:
            marks.append((len(out), ts))
            out += pl
            next_seq = seq + len(pl)
    return bytes(out), marks


def frames_with_ts(buf, marks):
    """重组缓冲上切帧（pcaplib.decode_frames 同语义），时间戳=覆盖帧尾的 segment。"""
    frames = []
    body = bytearray()
    mi = 0
    i = 0
    ln = len(buf)
    while i < ln:
        b = buf[i]
        if b == 0xEE and i + 1 < ln:
            nxt = buf[i + 1]
            if nxt == 0x00:
                body.append(0xEE)
                i += 2
                continue
            # 帧尾（EE EE 占 i..i+1）。下一包起点必须严格落在帧尾之前才认领
            # ——帧恰好收在包边界时（terminator 即包末两字节）应取本包 ts。
            while mi < len(marks) - 1 and marks[mi + 1][0] < i + 2:
                mi += 1
            frames.append((marks[mi][1], bytes(x ^ pcaplib.KEY[j % 4] for j, x in enumerate(body))))
            body = bytearray()
            i += 2
            continue
        body.append(b)
        i += 1
    if body:
        frames.append((marks[-1][1] if marks else 0,
                       bytes(x ^ pcaplib.KEY[j % 4] for j, x in enumerate(body))))
    return frames


def collect(path, keep_ports):
    """pcapng → 每(连接,方向)的 [(ts, seq, payload)]。低端口侧记为服务器。

    返回 (streams, t0)；t0=保留流的最早 TCP 载荷包到达时刻（首包锚点，
    与重组顺序无关——抓包开段乱序时「首帧锚点」会漂移几十 ms）。
    """
    per = {}
    t0 = None
    for ts, lt, pkt in pcaplib.read_pcapng(path):
        r = pcaplib.parse_eth_ip_tcp(pkt, lt)
        if r is None:
            continue
        src, sport, dst, dport, seq, payload = r
        if not payload:
            continue
        key = (src, sport, dst, dport) if sport < dport else (dst, dport, src, sport)
        if keep_ports and key[1] not in keep_ports:
            continue
        if not keep_ports and key[1] in LOBBY_PORTS:
            continue
        if t0 is None or ts < t0:
            t0 = ts
        d = 'c2s' if (src, sport) == (key[2], key[3]) else 's2c'
        per.setdefault(key, {}).setdefault(d, []).append((ts, seq, payload))
    streams = []
    for key, dirs in sorted(per.items(), key=lambda kv: (kv[0][1], kv[0][3])):
        for d in ('c2s', 's2c'):
            pkts = dirs.get(d)
            if not pkts:
                continue
            frames = []
            for ep in split_epochs(pkts):
                buf, marks = reassemble_ts(ep)
                frames.extend(f for f in frames_with_ts(buf, marks) if f[1])
            if frames:
                streams.append({'key': list(key), 'dir': d, 'frames': frames})
    return streams, t0


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('pcapng')
    ap.add_argument('out_pkl')
    ap.add_argument('--ports', default='', help='只保留这些服务器端口（逗号分隔）；缺省=全部非大厅口')
    args = ap.parse_args()
    keep = set(int(x) for x in args.ports.split(',') if x.strip()) if args.ports else None
    streams, t0 = collect(args.pcapng, keep)
    for st in streams:
        st['frames'] = [((ts - t0), fr) for ts, fr in st['frames']]
    with open(args.out_pkl, 'wb') as f:
        pickle.dump({'t0': t0, 'streams': streams}, f)
    total = sum(len(st['frames']) for st in streams)
    print('%s -> %s: %d 流 %d 帧 (t0=%.3fs)' % (
        args.pcapng, args.out_pkl, len(streams), total, 0.0))
    for st in streams:
        head = st['frames'][0][1][:1].hex() if st['frames'] else '--'
        print('  %s:%d <-> %s:%d %-3s %5d 帧 首op=%s' % (
            st['key'][0], st['key'][1], st['key'][2], st['key'][3],
            st['dir'], len(st['frames']), head))


if __name__ == '__main__':
    main()
