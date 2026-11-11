# -*- coding: utf-8 -*-
"""官服 NPC 出招距离探针：msg337(施法者坐标)×msg20(NPC 出招)×受害者位置 → 出招时水平距离分布。

受害者坐标双源：
  1) S2C 0x21 ServerAllDest 逐帧回显（每实体 44B = [obj u32][owner u32][9×f32]，
     首三 f32 = x/y/z）——只覆盖移动中实体，站桩挨打者不在其中；
  2) 本地玩家专属：C2S 211/220 上报坐标 + 玩家自己的 337 出招同步。
坑：pcap 时间戳是纳秒量纲（~1.79e15），秒窗判断必须乘 1e9；
    337 也为玩家自己出招而发——先锚定本地玩家 obj 再排除，否则
    「怪逼近玩家」其实是玩家自己在走。

用法: python -I npc_attack_distance_probe.py <pcapng> [pcapng ...]
产出: 每近战怪 n/min/p25/p50/max + 全体分布（官服野怪 ≈ min0.8/p50 2.3/p75 6.5m，
即停靠 ~3m、命中有效 ~6m 的双分位模型）。
"""
import math
import struct
import sys
from collections import Counter

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools')
from pcaplib import build_streams, stream_frames  # noqa: E402
from multihit_timing_probe import c2s_custom, s2c_custom  # noqa: E402

NS = 1e9
MELEE = ('atkhit', 'jn_mon', 'b_atk', 'mon')  # 近战族技能名；wuji_*/CS_* 是玩家/远程


def entries_0x21(frame):
    """0x21 | u16 count | count×44B；返回 [(obj, x, z)]。"""
    if not frame or frame[0] != 0x21 or len(frame) < 3:
        return []
    (n,) = struct.unpack_from('<H', frame, 1)
    out, off = [], 3
    for _ in range(n):
        if off + 44 > len(frame):
            break
        obj, _owner = struct.unpack_from('<II', frame, off)
        x, y, z = struct.unpack_from('<3f', frame, off + 8)
        out.append((obj, x, z))
        off += 44
    return out


def analyze(path):
    picked = None
    for st in build_streams(path):
        if not st.s2c:
            continue
        sf = stream_frames(st.s2c)
        if Counter(f[1][0] for f in sf if f[1]).get(0x10, 0) > 50:
            picked = (st, sf, stream_frames(st.c2s) if st.c2s else [])
            break
    if not picked:
        print('%s: no game stream' % path)
        return
    st, s2c, c2s = picked
    print('\n===== %s' % path)

    creports = []  # 本地玩家上报坐标（211/220 首三 f32）
    for ts, frame in c2s:
        vals = c2s_custom(frame)
        if not vals or vals[0][0] != 'i':
            continue
        fs = [v for t, v in vals if t == 'f32']
        if vals[0][1] in (211, 220) and len(fs) >= 3:
            creports.append((ts, fs[0], fs[2]))

    events = []
    track21 = {}  # obj -> [(ts, x, z)]（全体实体逐帧）
    for ts, frame in s2c:
        if frame and frame[0] == 0x21:
            for obj, x, z in entries_0x21(frame):
                track21.setdefault(obj, []).append((ts, x, z))
            continue
        vals = s2c_custom(frame)
        if not vals or vals[0][0] != 'i':
            continue
        op = vals[0][1]
        if op == 337 and len(vals) >= 3 and vals[1][0] == 'o':
            fs = [v for t, v in vals if t == 'f32']
            if len(fs) >= 3:
                events.append((ts, '337', vals[1][1], fs[0], fs[2], None, None))
        elif op == 20 and len(vals) >= 4 and vals[1][0] == 'o':
            skill = None
            for v in vals[2:]:
                if v[0] == 's':
                    skill = v[1].decode('gbk', 'replace')
                    break
            tgt = vals[3][1] if vals[3][0] == 'o' else None
            events.append((ts, '20', vals[1][1], None, None, skill, tgt))
    events.sort()

    # 锚定本地玩家：211 上报后 ≤200ms/±3m 内出现的 337 caster（服务端回显玩家出招）。
    votes = Counter()
    for ts, kind, obj, x, z, _sk, _tg in events:
        if kind != '337':
            continue
        for ts2, x2, z2 in creports:
            if 0 <= ts - ts2 <= 0.2 * NS and math.hypot(x - x2, z - z2) < 3:
                votes[obj] += 1
                break
    player = votes.most_common(1)[0][0] if votes else None
    print(' 本地玩家 obj=%s（337 回显锚定）' % (player,))

    ptrack = list(creports) + [(ts, x, z) for ts, kind, obj, x, z, _sk, _tg in events
                               if kind == '337' and obj == player]
    ptrack.sort()

    def fresh(track, ts, window=0.5):
        for t2, x2, z2 in reversed(track):
            if 0 <= ts - t2 <= window * NS:
                return x2, z2
            if ts - t2 > 30 * NS:
                break
        return None

    last337 = {}
    rows = []
    for ts, kind, obj, x, z, skill, tgt in events:
        if kind == '337':
            last337[obj] = (ts, x, z)
            continue
        if not skill or not any(m in skill for m in MELEE) or not tgt:
            continue
        hit = last337.get(obj)
        if not hit or ts - hit[0] > 2.0 * NS:
            continue
        p = fresh(ptrack, ts) if tgt == player else fresh(track21.get(tgt, []), ts)
        if not p:
            continue
        _, nx, nz = hit
        rows.append((obj, math.hypot(nx - p[0], nz - p[1])))
    per = {}
    for obj, d in rows:
        per.setdefault(obj, []).append(d)
    for obj, ds in sorted(per.items(), key=lambda kv: -len(kv[1])):
        ds.sort()
        print(' mob %s n=%d min=%.2f p25=%.2f p50=%.2f max=%.2f' % (
            obj, len(ds), ds[0], ds[len(ds) // 4], ds[len(ds) // 2], ds[-1]))
    allds = sorted(d for ds in per.values() for d in ds)
    if allds:
        print(' 总分布 n=%d: min=%.2f p10=%.2f p25=%.2f p50=%.2f p75=%.2f p90=%.2f max=%.2f' % (
            len(allds), allds[0], allds[len(allds) // 10], allds[len(allds) // 4],
            allds[len(allds) // 2], allds[len(allds) * 3 // 4], allds[len(allds) * 9 // 10], allds[-1]))


for _path in sys.argv[1:]:
    analyze(_path)
