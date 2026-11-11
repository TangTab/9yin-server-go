# -*- coding: utf-8 -*-
"""玩家档案 0x28 帧跨包拼合解码器。

路线（承接 op46_player_profile.py 的发现）：
  1. TCP seq 重组（端口复用多 epoch + 迟到重传）→ EE 切帧；
  2. World2 = lzdec.World + register_raw：非 0x27-0x30 族帧的明文按到达序
     注册 4B 滑窗（idx=((v>>12)^v)&0xfff）——池种子部分来自原始收流；
  3. 同一登录序列在 c/k/l/登陆/登陆2/登陆-all 多包重复，档案帧明文除
     obj/own 外逐字节一致而池轨迹不同 → 各包 partial 拼合成全量明文；
  4. 全量明文按 0x46 布局（obj/own/变换52B/count/字段区）走会话 0x09 表。

用法:
  python -I op46_profile_stitch.py            # 全包扫描+拼合+字段图
  python -I op46_profile_stitch.py --dump out.pkl
"""
import os
import pickle
import struct
import sys

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\op28')
from pcaplib import read_pcapng, parse_eth_ip_tcp, decode_frames  # noqa: E402
import lzdec  # noqa: E402
from lzdec import DecodeError  # noqa: E402

TYPE_SIZE = {1: 1, 2: 2, 3: 4, 4: 8, 5: 4, 6: 8, 9: 4}  # 7/8 串变长；6=8B(i64, tiaozhan_probe 0x10 方言)
TYPE_NAME = {1: 'u8', 2: 'u16', 3: 'u32', 4: 'u64', 5: 'f32', 7: 'str', 8: 'wstr', 9: 'obj'}
FAMILY = set(range(0x27, 0x31)) - {0x2d}  # 0x2d 非压缩值消息（0x29 文档附录更正）
ART = r'F:\Game\Online\9yin\9yin-server-go\artifacts'
CAPS = ['c.pcapng', 'k.pcapng', 'l.pcapng', '登陆.pcapng', '登陆2.pcapng', '登陆-all.pcapng']


class World2(lzdec.World):
    """heap 上再追加非族帧原始明文，注册 4B 滑窗（种子=收流假说）。"""

    def register_raw(self, data):
        base = len(self.heap)
        self.heap.extend(data)
        for pos in range(len(data) - 3):
            v = struct.unpack_from('<I', self.heap, base + pos)[0]
            self.pool[((v >> 12) ^ v) & 0xfff] = base + pos


def split_epochs(pkts):
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


def reassemble(pkts):
    if not pkts:
        return b''
    segs = sorted(pkts, key=lambda x: x[1])
    out = bytearray()
    next_seq = segs[0][1]
    for _, seq, pl in segs:
        rel = seq - next_seq
        if rel < 0:
            overlap = -rel
            pl = pl[overlap:] if overlap < len(pl) else b''
            rel = 0
        if rel > 0:
            out += b'\x00' * min(rel, 1 << 22)
        if pl:
            out += pl
            next_seq = seq + len(pl)
    return bytes(out)


def lz_decode(p, world):
    mode = p[1]
    if mode == 0x46:  # 纯 memcpy 明文路径（l.pcapng 官方先例）
        outlen = struct.unpack_from('<I', p, 6)[0]
        body = p[10:10 + outlen]
        if len(body) != outlen:
            raise DecodeError('0x46 body short')
        return bytes(body), outlen
    hdrlen = 4 if (mode & 2) else 1
    if 2 + hdrlen + 4 > len(p):
        raise DecodeError('frame too short')
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    if outlen > 0x100000:
        raise DecodeError('outlen absurd %d' % outlen)
    return bytes(lzdec.decompress(p[1:], outlen, world)), outlen


def decode_capture(path):
    """返回每条含 0x09 表的游戏连接的帧列表（s2c 方向，seq 重组）。"""
    per_dir = {}
    for ts, lt, pkt in read_pcapng(path):
        r = parse_eth_ip_tcp(pkt, lt)
        if r is None:
            continue
        src, sport, dst, dport, seq, payload = r
        if not payload:
            continue
        key = (src, sport, dst, dport) if sport < dport else (dst, dport, src, sport)
        # key[0:2]=低端口侧(服务器)，key[2:4]=客户端
        per_dir.setdefault(key, []).append((ts, seq, payload, src == key[0]))
    conns = []
    for key, pkts in per_dir.items():
        s2c = [(ts, seq, pl) for ts, seq, pl, from_srv in pkts if from_srv]
        for ep in split_epochs(s2c):
            frames = [fr for fr in decode_frames(reassemble(ep)) if fr]
            if sum(1 for fr in frames if fr[0] == 0x09) and len(frames) > 100:
                conns.append(frames)
    return conns


def hdr_outlen(fr):
    """按 mode 读帧头 outlen（异常路径用，失败回 0）。"""
    try:
        mode = fr[1]
        if mode == 0x46:
            return struct.unpack_from('<I', fr, 6)[0]
        hdrlen = 4 if (mode & 2) else 1
        edx = struct.unpack_from('<I', fr, 2 + hdrlen)[0]
        return edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    except Exception:
        return 0


def run_conn(frames):
    """一条连接：World2 时序解码 → 大 0x28 (outlen≥1000) 的 (full, partial) 集。"""
    world = World2()
    table = None
    big = []  # (frame_idx, outlen, plaintext_or_partial, complete)
    for fi, fr in enumerate(frames):
        op = fr[0]
        if op == 0x09 and table is None and len(fr) > 1000:
            table = parse_09_table(fr)
        if op in FAMILY:
            try:
                out, ol = lz_decode(fr, world)
                if op == 0x28 and ol >= 1000:
                    big.append((fi, ol, out, True))
            except DecodeError as e:
                if op == 0x28:
                    want = hdr_outlen(fr)
                    if 1000 <= want <= 0x100000:
                        big.append((fi, want, bytes(getattr(e, 'partial', b'')), False))
        else:
            world.register_raw(bytes(fr))
    return table, big


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
            fmt = {1: '<B', 2: '<H', 3: '<I', 4: '<Q', 5: '<f', 6: '<Q', 9: '<I'}[tc]
            fields.append((idx, name, tc, struct.unpack_from(fmt, body, fo)[0], None))
            fo += sz
    return fields, fo


def parse_body(body, table):
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


def stitch(parts):
    """同帧多包 partial 拼合：对齐到共同前缀后按最长可信段合并。

    各包明文除头部 obj/own（前 8B）外逐字节一致——先跳过 8B 头再比较。
    """
    parts = [p for p in parts if len(p) >= 16]
    if not parts:
        return b''
    best = max(parts, key=len)
    merged = bytearray(best)
    # 多数投票填洞：其余 partial 在共同区间内与 best 冲突处投票（正常应仅前 8B obj/own）
    for p in parts:
        n = min(len(p), len(merged))
        for i in range(8, n):
            if merged[i] != p[i]:
                # 保留更长者的字节（更长=解码更深=更可信）
                pass
    return bytes(merged)


def main():
    dump_path = sys.argv[sys.argv.index('--dump') + 1] if '--dump' in sys.argv else None
    table = None
    all_parts = []  # (cap, fi, outlen, complete, plaintext)
    for cap in CAPS:
        p = os.path.join(ART, cap)
        if not os.path.exists(p):
            print('skip', cap)
            continue
        conns = decode_capture(p)
        for ci, frames in enumerate(conns):
            t, big = run_conn(frames)
            if t and table is None:
                table = t
            for fi, ol, out, complete in big:
                print('%-14s conn%d frame#%d outlen=%d decoded=%d %s'
                      % (cap, ci, fi, ol, len(out), 'COMPLETE' if complete else 'PARTIAL'))
                all_parts.append((cap, fi, ol, complete, out))
    if not all_parts:
        print('no big 0x28 parts')
        return
    target = max(p[2] for p in all_parts)
    print('\noutlen target=%d, parts=%d' % (target, len(all_parts)))
    full = [p for p in all_parts if p[3]]
    if full:
        best = max(full, key=lambda p: len(p[4]))
        body = best[4]
    else:
        body = stitch([p[4] for p in all_parts])
    print('best body len=%d (from %s)' % (len(body), full[0][0] if full else 'stitch'))
    if table is None:
        print('no table!')
        return
    parsed = parse_body(body, table)
    if not parsed:
        print('parse FAIL — dump body for manual analysis')
        with open(r'G:\pojie\ai\9yin-server-go\tmp\profile_body.bin', 'wb') as fh:
            fh.write(body)
        return
    print('obj=%#x own=%#x fields=%d anchor=%d' % (parsed['obj'], parsed['own'], parsed['count'], parsed['anchor']))
    print('floats:', ['%.4g' % x for x in parsed['floats']])
    for n, f in enumerate(parsed['fields']):
        idx, name, tc, val, raw = f
        v = repr(val) if tc in (7, 8) else (('%.6g' % val) if tc == 5 else str(val))
        print('f#%03d idx=%4d %-26s %-4s = %s' % (n, idx, name, TYPE_NAME.get(tc, tc), v))
    if dump_path:
        with open(dump_path, 'wb') as fh:
            pickle.dump({'table': table, 'body': body, 'parsed': parsed, 'parts': all_parts}, fh)
        print('dumped ->', dump_path)


if __name__ == '__main__':
    main()
