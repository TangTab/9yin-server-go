# -*- coding: utf-8 -*-
"""登陆-all.pcapng 玩家档案探针（v2）。

pcaplib.build_streams 是到达序裸拼接——本包 s2c 有 ①同四元组端口复用
多连接混流（ISN 大跳变）②2824B 迟到重传插花，直接切帧会把 3203B 的
玩家档案 0x28 帧撑成 6027B 假帧。故先按 seq 重组再切帧。

流程：seq 重组（epoch=ISN 空间，重叠去重）→ EE 切帧 → 会话内 0x09 表
→ 0x27-0x30 族共享 LZ 池时序解码（tools/op28/lzdec.World）→ 大 0x28
明文按 0x46 布局逐字段解码（296 字段玩家档案全图）。

用法:
  python -I op46_player_profile.py <登陆-all.pcapng> [--dump out.pkl]
"""
import os
import pickle
import struct
import sys
from collections import Counter

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
from pcaplib import read_pcapng, parse_eth_ip_tcp, decode_frames  # noqa: E402

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\op28')
from lzdec import World, decompress, DecodeError  # noqa: E402

TYPE_SIZE = {1: 1, 2: 2, 3: 4, 4: 8, 5: 4, 9: 4}  # 7/8 串变长
TYPE_NAME = {1: 'u8', 2: 'u16', 3: 'u32', 4: 'u64', 5: 'f32', 7: 'str', 8: 'wstr', 9: 'obj'}
FAMILY = set(range(0x27, 0x31))


# ---------- TCP 重组 ----------

def collect(path, s_ip, s_port):
    """按方向收集 (ts, seq, payload)，返回 (c2s, s2c)。"""
    c2s, s2c = [], []
    for ts, lt, pkt in read_pcapng(path):
        r = parse_eth_ip_tcp(pkt, lt)
        if r is None:
            continue
        src, sport, dst, dport, seq, payload = r
        if not payload:
            continue
        if src == s_ip and sport == s_port:
            s2c.append((ts, seq, payload))
        elif dst == s_ip and dport == s_port:
            c2s.append((ts, seq, payload))
    return c2s, s2c


def split_epochs(pkts):
    """到达序扫：seq 相对期望大倒退(>1MB)视为新连接 epoch。"""
    epochs, cur = [], []
    exp = None
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


def reassemble(pkts):
    """epoch 内按 seq 排序 + 重叠去重 → 连续字节流（空洞补零保位）。"""
    if not pkts:
        return b''
    segs = sorted(pkts, key=lambda x: x[1])
    out = bytearray()
    next_seq = segs[0][1]
    for _, seq, pl in segs:
        rel = seq - next_seq
        if rel < 0:  # 纯重传已被覆盖
            overlap = -rel
            pl = pl[overlap:] if overlap < len(pl) else b''
            rel = 0
        if rel > 0:  # 捕包空洞：补零保位（超大洞截断防炸）
            out += b'\x00' * min(rel, 1 << 22)
        if pl:
            out += pl
            next_seq = seq + len(pl)
    return bytes(out)


# ---------- 帧级解析 ----------

def parse_09_table(frame):
    if not frame or frame[0] != 0x09 or len(frame) < 3:
        return None
    (n,) = struct.unpack_from('<H', frame, 1)
    off, out = 3, []
    for _ in range(n):
        try:
            end = frame.index(b'\x00', off)
        except ValueError:
            return None
        out.append((frame[off:end].decode('latin-1'), frame[end + 1]))
        off = end + 2
    return out


def walk_fields(body, fo, count, table):
    fields = []
    for _ in range(count):
        if fo + 2 > len(body):
            return None
        idx = struct.unpack_from('<H', body, fo)[0]
        if idx >= len(table):
            return None
        name, tc = table[idx]
        fo += 2
        if tc in (7, 8):
            if fo + 4 > len(body):
                return None
            (slen,) = struct.unpack_from('<I', body, fo)
            fo += 4
            if fo + slen > len(body):
                return None
            raw = bytes(body[fo:fo + slen])
            fo += slen
            val = raw.rstrip(b'\x00').decode('gbk', 'replace') if tc == 7 else raw.decode('utf-16le', 'replace')
            fields.append((idx, name, tc, val, raw))
        else:
            sz = TYPE_SIZE.get(tc)
            if sz is None or fo + sz > len(body):
                return None
            fmt = {1: '<B', 2: '<H', 3: '<I', 4: '<Q', 5: '<f', 9: '<I'}[tc]
            fields.append((idx, name, tc, struct.unpack_from(fmt, body, fo)[0], None))
            fo += sz
    return fields, fo


def lz_decode(p, world):
    """op 级压缩值消息：mode@[1]，4B 长头位形读 outlen@[6:10]。"""
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    return bytes(decompress(p[1:], outlen, world)), outlen


def parse_046_body(body, table):
    """明文体（obj/own/变换/字段区）→ dict 或 None。锚位自校验扫描。"""
    if len(body) < 64:
        return None
    obj, own = struct.unpack_from('<II', body, 0)
    for anchor in (60, 58, 56, 62, 64, 66, 68, 70, 72, 54):
        if anchor + 2 > len(body):
            continue
        (cnt,) = struct.unpack_from('<H', body, anchor)
        if not (3 <= cnt <= 600):
            continue
        r = walk_fields(body, anchor + 2, cnt, table)
        if r and r[1] == len(body):
            fl = struct.unpack_from('<%df' % ((anchor - 8) // 4), body, 8)
            return dict(obj=obj, own=own, floats=fl, count=cnt, fields=r[0], anchor=anchor)
    return None


def main():
    path = sys.argv[1]
    s_ip = '49.4.14.150'
    s_port = 7074
    if len(sys.argv) > 2 and sys.argv[2] == '--ip':
        s_ip = sys.argv[3]
    c2s, s2c = collect(path, s_ip, s_port)
    epochs = split_epochs(s2c)
    print('s2c pkts=%d epochs=%d sizes=%s' % (len(s2c), len(epochs), [sum(len(p) for _, _, p in e) for e in epochs]))

    table = None
    world = World()
    big_frames = []  # (epoch, idx, outlen, plaintext)
    stats = Counter()
    for ei, ep in enumerate(epochs):
        stream = reassemble(ep)
        frames = decode_frames(stream)
        print('epoch%d bytes=%d frames=%d' % (ei, len(stream), len(frames)))
        for fi, fr in enumerate(frames):
            if not fr:
                stats['empty'] += 1
                continue
            op = fr[0]
            stats['op%02x' % op] += 1
            if op == 0x09 and table is None:
                t = parse_09_table(fr)
                if t and len(t) > 100:
                    table = t
                    print('  0x09 table @e%d#%d cols=%d' % (ei, fi, len(t)))
            if op == 0x28:
                len1 = struct.unpack_from('<I', fr, 2)[0]
                outlen_hdr = struct.unpack_from('<I', fr, 6)[0]
                ok = (len1 == len(fr) - 1)
                print('  0x28 @e%d#%d len=%d len-1=%d(%s) outlen_hdr=%d mode=%#x'
                      % (ei, fi, len(fr), len1, 'OK' if ok else 'BAD', outlen_hdr, fr[1]))
                if not ok:
                    continue  # 仍粘连/残缺的帧不进 LZ 池
            if op in FAMILY:
                try:
                    out, outlen = lz_decode(fr, world)
                    stats['lz_ok'] += 1
                    if op == 0x28 and outlen >= 1000:
                        big_frames.append((ei, fi, outlen, out))
                except DecodeError as e:
                    stats['lz_fail'] += 1
    print('stats:', dict(stats))
    if table is None:
        print('NO 0x09 TABLE — cannot field-map')
        return

    print('\n===== big 0x28 plaintexts: %d =====' % len(big_frames))
    for ei, fi, outlen, out in big_frames:
        print('\n--- e%d#%d outlen=%d head=%s' % (ei, fi, outlen, out[:32].hex()))
        parsed = parse_046_body(out, table)
        if not parsed:
            print('  parse fail; first 64B:', out[:64].hex())
            continue
        print('  obj=%#x own=%#x fields=%d anchor=%d scale=%s' % (
            parsed['obj'], parsed['own'], parsed['count'], parsed['anchor'],
            '%.4g' % parsed['floats'][8] if len(parsed['floats']) > 8 else '?'))
        print('  floats:', ['%.4g' % x for x in parsed['floats']])
        for n, f in enumerate(parsed['fields']):
            idx, name, tc, val, raw = f
            v = repr(val) if tc in (7, 8) else (('%.6g' % val) if tc == 5 else str(val))
            print('  f#%03d idx=%4d %-26s %-4s = %s' % (n, idx, name, TYPE_NAME.get(tc, tc), v))
    if '--dump' in sys.argv:
        dp = sys.argv[sys.argv.index('--dump') + 1]
        with open(dp, 'wb') as fh:
            pickle.dump({'table': table, 'big': big_frames}, fh)
        print('dumped ->', dp)


if __name__ == '__main__':
    main()
