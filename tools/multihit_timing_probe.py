# -*- coding: utf-8 -*-
"""多段技能伤害触发时序探针：官服 pcapng → C2S 211 出招 ↔ S2C msg24 c1=17 伤害跳时间轴。

对每笔出招，取其后同技能的伤害飘字，输出：
  cast_ts - msg20/action 帧延迟、每跳相对出招的毫秒延迟、跳间隔。
用法:
  python -I multihit_timing_probe.py <a.pcapng> [b.pcapng ...]
"""
import struct
import sys
from collections import Counter, defaultdict

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames  # noqa: E402


def parse_tvarlist(buf, off):
    """count u16 + [tag val]*；返回 (values, new_off) 或 (None, off)。"""
    if off + 2 > len(buf):
        return None, off
    (n,) = struct.unpack_from('<H', buf, off)
    off += 2
    if n == 0 or n > 1024:
        return None, off
    vals = []
    for _ in range(n):
        if off >= len(buf):
            return None, off
        tag = buf[off]
        off += 1
        if tag == 2:
            vals.append(('i', struct.unpack_from('<i', buf, off)[0])); off += 4
        elif tag == 3:
            vals.append(('i64', struct.unpack_from('<q', buf, off)[0])); off += 8
        elif tag == 4:
            vals.append(('f32', struct.unpack_from('<f', buf, off)[0])); off += 4
        elif tag == 5:
            vals.append(('f64', struct.unpack_from('<d', buf, off)[0])); off += 8
        elif tag == 6:
            (ln,) = struct.unpack_from('<I', buf, off); off += 4
            vals.append(('s', buf[off:off + ln].rstrip(b'\x00'))); off += ln
        elif tag == 7:
            (ln,) = struct.unpack_from('<I', buf, off); off += 4
            vals.append(('w', buf[off:off + ln - 2])); off += ln
        elif tag == 8:
            a, b = struct.unpack_from('<II', buf, off)
            vals.append(('o', (a, b))); off += 8
        else:
            return None, off
    return vals, off


def c2s_custom(frame):
    """0x0A | 16B session | u32 seq | TVarList → values 或 None。"""
    if not frame or frame[0] != 0x0A or len(frame) < 23:
        return None
    return parse_tvarlist(frame, 21)[0]


def s2c_custom(frame):
    if not frame or frame[0] not in (0x1E, 0x27) or len(frame) < 3:
        return None
    (n,) = struct.unpack_from('<H', frame, 1)
    if n == 0 or n > 1024:
        return None
    vals = []
    off = 3
    for _ in range(n):
        if off >= len(frame):
            return None
        tag = frame[off]
        off += 1
        if tag == 2:
            vals.append(('i', struct.unpack_from('<i', frame, off)[0])); off += 4
        elif tag == 3:
            vals.append(('i64', struct.unpack_from('<q', frame, off)[0])); off += 8
        elif tag == 4:
            vals.append(('f32', struct.unpack_from('<f', frame, off)[0])); off += 4
        elif tag == 5:
            vals.append(('f64', struct.unpack_from('<d', frame, off)[0])); off += 8
        elif tag == 6:
            (ln,) = struct.unpack_from('<I', frame, off); off += 4
            vals.append(('s', frame[off:off + ln].rstrip(b'\x00'))); off += ln
        elif tag == 7:
            (ln,) = struct.unpack_from('<I', frame, off); off += 4
            vals.append(('w', frame[off:off + ln - 2])); off += ln
        elif tag == 8:
            a, b = struct.unpack_from('<II', frame, off)
            vals.append(('o', (a, b))); off += 8
        else:
            return None
    return vals


def txt(v):
    return v[1].decode('gbk', 'replace') if v[0] in ('s', 'w') else None


def analyze(path):
    streams = build_streams(path)
    game = None
    for st in streams:
        if not st.s2c:
            continue
        sf = stream_frames(st.s2c)
        ops = Counter(f[1][0] for f in sf if f[1])
        if ops.get(0x10, 0) > 50:
            game = (st, sf, stream_frames(st.c2s) if st.c2s else [])
            break
    if not game:
        print('%s: no game stream' % path)
        return
    st, s2c, c2s = game
    print('\n===== %s  client=%s server=%s  s2c=%d c2s=%d =====' % (
        path, st.client, st.server, len(s2c), len(c2s)))

    casts = []        # (ts, skill)
    for ts, frame in c2s:
        vals = c2s_custom(frame)
        if not vals or len(vals) < 2:
            continue
        if vals[0] == ('i', 211) and vals[1][0] == 's':
            casts.append((ts, vals[1][1].decode('gbk', 'replace')))

    # S2C msg20 技能动作帧（native skill action: op20 + obj + str 技能id...）
    msg20 = []        # (ts, skill)
    damages = []      # (ts, skill, dmg, victim)
    for ts, frame in s2c:
        if not frame:
            continue
        vals = s2c_custom(frame)
        if not vals:
            continue
        op = vals[0]
        if op == ('i', 20) and len(vals) >= 2 and vals[1][0] == 'o':
            # [20, obj(施法者), 窄串技能ID/动作名, ...]（NPC 动作名无 CS_ 前缀）
            for v in vals[2:]:
                if v[0] == 's':
                    msg20.append((ts, v[1].decode('gbk', 'replace')))
                    break
        elif op == ('i', 24) and len(vals) >= 7 and vals[1][0] == 'o':
            # 官服形态：[24, obj(受害者), i 17, i 伤害, i 0, i 2, s 技能ID, ...]
            if vals[2] == ('i', 17):
                skill = txt(vals[6])
                dmg = vals[3][1]
                victim = vals[1][1][0]
                damages.append((ts, skill, dmg, victim))

    print('casts=%d msg20=%d damages(c1=17)=%d' % (len(casts), len(msg20), len(damages)))
    print('cast skills: %s' % dict(Counter(s for _, s in casts)))

    # 每笔出招 → 匹配其后 12s 内同技能的伤害跳（victim 不限：多段可换目标）
    per_skill_ticks = defaultdict(list)   # skill -> [ [delta_ms,...] per cast ]
    for i, (cts, skill) in enumerate(casts):
        next_cast_ts = casts[i + 1][0] if i + 1 < len(casts) else cts + 30e6
        ticks = [d for d in damages if d[0] >= cts and d[0] < min(cts + 12e6, next_cast_ts) and d[1] == skill]
        # 找该窗口内最近的 msg20（官方回显延迟）
        m20 = [m for m in msg20 if m[0] >= cts and m[0] < cts + 2e6 and m[1] == skill]
        m20d = '%.0f' % ((m20[0][0] - cts) / 1e3) if m20 else '-'
        deltas = []
        prev = None
        for d in ticks:
            ms = (d[0] - cts) / 1e3
            if prev is not None and abs(ms - prev) < 30:
                deltas[-1] = deltas[-1] + '*'  # 同毫秒=同跳多目标
                continue
            deltas.append('%.0f' % ms)
            prev = ms
        print('cast #%d %-20s t=+%.3fs msg20@+%sms dmg_ticks=[%s]' % (
            i, skill, (cts - casts[0][0]) / 1e6, m20d, ', '.join(deltas) if deltas else 'NONE'))
        if ticks:
            dedup = []
            prev = None
            for d in ticks:
                ms = (d[0] - cts) / 1e3
                if prev is not None and abs(ms - prev) < 30:
                    continue
                dedup.append(ms)
                prev = ms
            per_skill_ticks[skill].append(dedup)

    print('\n-- per-skill tick offset summary (ms after cast) --')
    for skill, casts_ticks in sorted(per_skill_ticks.items()):
        n = max(len(t) for t in casts_ticks)
        for k in range(n):
            vals = [t[k] for t in casts_ticks if len(t) > k]
            if vals:
                print('  %-14s tick#%d  n=%d  min=%.0f max=%.0f avg=%.0f' % (
                    skill, k + 1, len(vals), min(vals), max(vals), sum(vals) / len(vals)))


if __name__ == '__main__':
    for p in sys.argv[1:]:
        analyze(p)
