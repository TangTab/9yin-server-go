# -*- coding: utf-8 -*-
"""frames pkl → 0x27-0x2a 压缩值消息族 LZ 会话池解码 + 0x28 字段走查。

会话池模型（World2）：线序喂入全部帧明文（非族帧原始明文按 4B 滑窗注册，
族帧解压输出自然入池）。帧布局与字段方言见 docs/官服玩家档案0x28全量帧.md
与 docs/官服模式NPC渲染收束.md。

用法：
  python -I npc28_decode.py <frames.pkl>                  # 概览：全部 0x28 的 obj/own/位姿/字段名
  python -I npc28_decode.py <frames.pkl> --at 30.456      # 走查 ±50ms 窗口内 0x28 全字段值
  python -I npc28_decode.py <frames.pkl> --obj 0x700014b  # 按对象过滤
"""
import argparse
import os
import pickle
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, '..', 'op28'))
sys.path.insert(0, os.path.join(HERE, '..', 'pcap_analysis'))
import lzdec  # noqa: E402
from lzdec import DecodeError  # noqa: E402
import decode as padecode  # noqa: E402


class World2(lzdec.World):
    """heap 上追加非族帧原始明文，注册 4B 滑窗（收流种子假说）。"""

    def register_raw(self, data):
        base = len(self.heap)
        self.heap.extend(data)
        for pos in range(len(data) - 3):
            v = struct.unpack_from('<I', self.heap, base + pos)[0]
            self.pool[((v >> 12) ^ v) & 0xfff] = base + pos


FAMILY = set(range(0x27, 0x31))


def decode_with(p, world):
    mode = p[1]
    if mode == 0x46:
        outlen = struct.unpack_from('<I', p, 6)[0]
        return bytes(p[10:10 + outlen]), outlen
    hdrlen = 4 if (mode & 2) else 1
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    if outlen > 0x100000:
        raise DecodeError('outlen absurd %d' % outlen)
    return bytes(lzdec.decompress(p[1:], outlen, world)), outlen


def walk_fields(body, tbl):
    """body=[obj][own][52B 变换][cnt u16][字段×N]；返回 (obj, own, tf, fields, used, total)。

    官方编码器捷径（len=0 cstring/[lo FF] flag/[00跑+01 00] 标记）会表现为
    OVERFLOW/None 行——对发射侧无阻塞（明文路径可绕开，见档案文档 §四）。
    """
    obj, own = struct.unpack_from('<II', body, 0)
    tf = struct.unpack_from('<13f', body, 8)
    cnt = struct.unpack_from('<H', body, 60)[0]
    off = 62
    fields = []
    for _ in range(cnt):
        if off + 2 > len(body):
            fields.append((-1, 'OVERFLOW', body[off:].hex()))
            break
        idx = struct.unpack_from('<H', body, off)[0]
        off += 2
        t = tbl.types.get(idx, 3)
        nm = tbl.names.get(idx, '?%d' % idx)
        try:
            if t == 1:
                v = body[off]; off += 1
            elif t == 2:
                v = struct.unpack_from('<H', body, off)[0]; off += 2
            elif t == 3:
                v = struct.unpack_from('<i', body, off)[0]; off += 4
            elif t == 4:
                v = struct.unpack_from('<q', body, off)[0]; off += 8
            elif t == 5:
                v = round(struct.unpack_from('<f', body, off)[0], 3); off += 4
            elif t in (7, 8):
                n = struct.unpack_from('<I', body, off)[0]; off += 4
                v = body[off:off + n].rstrip(b'\x00').decode('gbk', 'replace'); off += n
            else:
                v = '<t%d?>' % t
        except (struct.error, IndexError):
            fields.append((idx, nm, '<OVERFLOW>'))
            break
        fields.append((idx, nm, v))
    return obj, own, tf, fields, off, len(body)


def run(pkl_path, at_times, obj_filter):
    data = pickle.load(open(pkl_path, 'rb'))
    for st in data['streams']:
        if st['dir'] != 's2c':
            continue
        tbl = padecode.Table09()
        for _, fr in st['frames']:
            if fr and fr[0] == 0x09:
                tbl.feed(fr)
        world = World2()
        decoded = []  # (rel_s, op, plaintext, ok)
        fam_stats = {'ok': 0, 'fail': 0}
        for ts, fr in st['frames']:
            if not fr:
                continue
            if fr[0] in FAMILY:
                try:
                    out, _ = decode_with(fr, world)
                    fam_stats['ok'] += 1
                    decoded.append((ts / 1e6, fr[0], out, True))
                except DecodeError:
                    fam_stats['fail'] += 1
            else:
                world.register_raw(fr)
        print('== 流 %s:%d 族帧 ok=%d fail=%d ==' % (
            st['key'][0], st['key'][1], fam_stats['ok'], fam_stats['fail']))
        for rel, op, out, ok in decoded:
            if op != 0x28 or not ok or len(out) < 12:
                continue
            try:
                obj, own, tf, fields, used, total = walk_fields(out, tbl)
            except Exception:  # noqa: BLE001
                continue
            if obj_filter and obj != obj_filter:
                continue
            if not at_times:
                print('  %9.3f obj=%#010x own=%#010x posA=(%.1f,%.1f,%.1f) B=(%.1f,%.1f) '
                      'scale=%.2f 字段%d/%d: %s' % (
                          rel, obj, own, tf[0], tf[1], tf[2], tf[4], tf[6], tf[8],
                          len(fields), total, ','.join(n for _, n, _ in fields)))
                continue
            for want in at_times:
                if abs(rel - want) < 0.05:
                    print('=== %9.3f op28 obj=%#x own=%#x fields=%d used=%d/%d ===' % (
                        rel, obj, own, len(fields), used, total))
                    print('  posA=(%.2f,%.2f,%.2f) yaw=%.2f  posB=(%.2f,%.2f,%.2f) scale=%.2f' % (
                        tf[0], tf[1], tf[2], tf[3], tf[4], tf[5], tf[6], tf[8]))
                    for idx, nm, v in fields:
                        print('  %-4d %-20s %s' % (idx, nm, v))
                    break


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument('in_pkl')
    ap.add_argument('--at', default='', help='走查这些时刻(秒,逗号分隔)±50ms 的 0x28 全字段')
    ap.add_argument('--obj', default=0, type=lambda x: int(x, 0), help='只看该对象(0x 前缀)')
    args = ap.parse_args()
    at = [float(x) for x in args.at.split(',') if x.strip()] if args.at else []
    run(args.in_pkl, at, args.obj)


if __name__ == '__main__':
    main()
