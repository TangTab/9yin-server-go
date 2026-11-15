# -*- coding: utf-8 -*-
"""两阶段回放打包器（B 线 2026-11-14 第十轮）。

已定案时序规律（c/登陆/登陆2/初始账号 四份外服抓包交叉验证）：
  0x0B+化身0x28+首爆发 ──(客户端读条3~10s)──> C2S 单字节 0x09
    ──(65ms)──> 第二爆发(op15/0x18行/0x2A/m386…) ──(1~3s)──> AOI 物化簇(0x28+0x0D)

V2.1 全序列单点灌入失败根因=相位错：0x0B 触发新场景读条，其后所有帧
落进读条窗口被丢。两阶段：
  Phase A（clientActivity 时发）：0x0B 起 → 首爆发尾（t<=SPLIT_A 秒）
  Phase B（客户端下个 0x09 时发）：第二爆发起（t>=SPLIT_B 秒）→ 抓包尾

Usage:
  python -I build_replay_phases.py            # 对拍 v2_1_clean 并产两阶段
  python -I build_replay_phases.py --check    # 仅对拍
"""
import struct
import sys

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
import pcaplib  # noqa: E402

CAP = r'F:\Game\Online\9yin\9yin-server-go\artifacts\c.pcapng'
OUT_DIR = r'G:\pojie\ai\9yin-server-go\tmp\op28_analysis'
V21 = OUT_DIR + r'\replay_v2_1_clean.bin'
PHASE_A = OUT_DIR + r'\replay_p1_entry_burst.bin'
PHASE_B = OUT_DIR + r'\replay_p2_post_ready.bin'
SPLIT_A = 0.30   # 首爆发窗（官方 0.128..0.240 后静默到 3.09）
SPLIT_B = 4.60   # 第二爆发 4.735；4.6 起无帧，取安全下沿


def game_s2c(path):
    st = max(pcaplib.build_streams(path), key=lambda s: len(pcaplib.stream_frames(s.s2c)))
    return [(t, f) for t, f in pcaplib.stream_frames(st.s2c) if f]


def v21_frames(tfs):
    """复刻 V2.1 口径：从首个 0x0B 起、跳过 0x09 表、含其余全部 op。"""
    t0 = tfs[0][0]
    start = next(i for i, (_, f) in enumerate(tfs) if f[0] == 0x0B)
    return [(t - t0, f) for t, f in tfs[start:] if f[0] != 0x09]


def pack(frames):
    out = bytearray()
    for _, f in frames:
        out += struct.pack('<I', len(f)) + f
    return bytes(out)


def main():
    check_only = '--check' in sys.argv
    tfs = game_s2c(CAP)
    sel = v21_frames(tfs)
    rebuilt = pack(sel)
    old = open(V21, 'rb').read()
    print('rebuilt=%d frames=%d  v2_1_clean=%d bytes  match=%s'
          % (len(rebuilt), len(sel), len(old), rebuilt == old))
    if len(rebuilt) != len(old):
        # 找首个差异
        for i in range(min(len(rebuilt), len(old))):
            if rebuilt[i] != old[i]:
                print('first diff @%d: rebuilt %s vs old %s' % (i, rebuilt[i:i+8].hex(), old[i:i+8].hex()))
                break
    if check_only:
        return
    a = [(dt, f) for dt, f in sel if dt <= SPLIT_A]
    b = [(dt, f) for dt, f in sel if dt >= SPLIT_B]
    mid = [(dt, f) for dt, f in sel if SPLIT_A < dt < SPLIT_B]
    open(PHASE_A, 'wb').write(pack(a))
    open(PHASE_B, 'wb').write(pack(b))
    from collections import Counter
    print('PhaseA: %d frames %d bytes ops=%s  dt<= %.2fs' % (len(a), len(pack(a)), Counter('%02X' % f[0] for _, f in a).most_common(8), SPLIT_A))
    print('middle dropped: %d frames ops=%s  dts=%s' % (len(mid), Counter('%02X' % f[0] for _, f in mid).most_common(8), ['%.2f' % dt for dt, _ in mid][:10]))
    print('PhaseB: %d frames %d bytes ops=%s  dt>=%.2fs  last dt=%.2fs' % (len(b), len(pack(b)), Counter('%02X' % f[0] for _, f in b).most_common(8), SPLIT_B, b[-1][0]))
    # PhaseB 内物化簇统计
    ents = [(dt, f) for dt, f in b if f[0] in (0x28, 0x0D)]
    print('PhaseB entity frames (0x28/0x0D)=%d  first=%.2fs' % (len(ents), ents[0][0] if ents else -1))


if __name__ == '__main__':
    main()
