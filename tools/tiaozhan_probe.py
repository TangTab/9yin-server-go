# -*- coding: utf-8 -*-
"""挑战.pcapng 专项探针：寒心恨雪(CS_jh_hyjf08,怒气定跳数) 与 太极剑舞
(CS_wd_tjj07,施法中可移动) 两机制取证。

输出：
  1) 0x09 表定位 SP/LogicState 等属性索引（本包官服表为准）
  2) 每次 hyjf08 施放：前 3s 怒气轨迹 → 出招 → 怒气扣除帧时序 → 伤害跳数/间隔
  3) 每次 tjj07 施放：8s 窗口全事件流（msg20/伤害/移动 0x21/C2S15/轻功/LogicState）
用法:
  python -I tiaozhan_probe.py <挑战.pcapng>
"""
import struct
import sys
from collections import Counter

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames, parse_tvarlist  # noqa: E402

HYJF08 = 'CS_jh_hyjf08'
TJJ07 = 'CS_wd_tjj07'


def load_capture(path):
    streams = build_streams(path)
    for st in streams:
        if not st.s2c:
            continue
        sf = stream_frames(st.s2c)
        ops = Counter(f[1][0] for f in sf if f[1])
        if ops.get(0x10, 0) > 50:
            cf = stream_frames(st.c2s) if st.c2s else []
            return [(ts, fr) for ts, fr in sf if fr], [(ts, fr) for ts, fr in cf if fr]
    return None, None


def parse_09(frame):
    """0x09 | u16 count | (name\\0 u8 type)* → [(name, ptype)] 按位次。"""
    if not frame or frame[0] != 0x09 or len(frame) < 3:
        return None
    (n,) = struct.unpack_from('<H', frame, 1)
    off = 3
    out = []
    for _ in range(n):
        try:
            end = frame.index(b'\x00', off)
        except ValueError:
            return None
        out.append((frame[off:end].decode('latin-1'), frame[end + 1]))
        off = end + 2
    return out


def attr_size(ptype):
    return {1: 1, 2: 2, 3: 4, 5: 4, 6: 8, 8: 8}.get(ptype)


def parse_10(frame, entries):
    """0x10 属性帧 → {name: value}（int/float 按线型）。"""
    if not frame or frame[0] != 0x10 or len(frame) < 12:
        return None
    isview = frame[1]
    oid, owner = struct.unpack_from('<II', frame, 2)
    (cnt,) = struct.unpack_from('<H', frame, 10)
    off = 12
    props = {}
    for _ in range(cnt):
        if off + 2 > len(frame):
            break
        idx = struct.unpack_from('<H', frame, off)[0]
        off += 2
        if idx >= len(entries):
            return None
        name, ptype = entries[idx]
        sz = attr_size(ptype)
        if sz is None:
            break
        if off + sz > len(frame):
            break
        if ptype == 1:
            props[name] = frame[off]
        elif ptype == 2:
            props[name] = struct.unpack_from('<H', frame, off)[0]
        elif ptype == 3:
            props[name] = struct.unpack_from('<i', frame, off)[0]
        elif ptype == 5:
            props[name] = round(struct.unpack_from('<f', frame, off)[0], 3)
        off += sz
    return (isview, oid, owner, props)


def txt(v):
    return v[1].decode('gbk', 'replace') if v[0] in ('s', 'w') else None


def s2c_custom(frame):
    """0x1E/0x27 信封 → pcaplib 裸值列表（count@1, values@3）。"""
    if not frame or frame[0] not in (0x1E, 0x27) or len(frame) < 3:
        return None
    return parse_tvarlist(frame[1:])


def c2s_custom(frame):
    if not frame or frame[0] != 0x0A or len(frame) < 23:
        return None
    return parse_tvarlist(frame[21:])


def fmt_v(v):
    return repr(v)


def main(path):
    s2c, c2s = load_capture(path)
    if not s2c:
        print('%s: no game stream' % path)
        return
    t0 = s2c[0][0]

    # 0x09 表 → 索引；本包无 0x09（开局即战斗会话）则回退官服 2433 表
    entries = None
    for _, fr in s2c:
        e = parse_09(fr)
        if e and len(e) > 100:
            entries = e
            break
    if not entries:
        tbl = r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt'
        byidx = {}
        maxidx = 0
        for line in open(tbl, encoding='ascii'):
            parts = line.rstrip('\n').split('\t')
            if len(parts) == 3:
                i = int(parts[0])
                byidx[i] = (parts[1], int(parts[2]))
                maxidx = max(maxidx, i)
        # None 行是占位，必须保留位置（跳过会整体 +1 错位）
        entries = [byidx.get(i, ('?%d' % i, 3)) for i in range(maxidx + 1)]
        print('(no 0x09 in capture; using official 2433 table)')
    idx_of = {n: i for i, (n, _) in enumerate(entries)} if entries else {}
    print('0x09 entries=%d SP@%s LogicState@%s MaxSP@%s' % (
        len(entries), idx_of.get('SP'), idx_of.get('LogicState'), idx_of.get('MaxSP')))

    # ---- 事件抽取 ----
    casts = []        # (ts, skill, vals)
    for ts, fr in c2s:
        vals = c2s_custom(fr)
        if vals and len(vals) >= 2 and vals[0] == 211 and isinstance(vals[1], str):
            casts.append((ts, vals[1], vals))

    # 玩家 object id：SP 单属性帧最多的对象（本地玩家）；退路=msg20 施法者
    sp_owner = {}
    for ts, fr in s2c:
        r = parse_10(fr, entries)
        if r and r[0] == 0 and 'SP' in r[3]:
            sp_owner[r[1]] = sp_owner.get(r[1], 0) + 1
    player_id = max(sp_owner, key=sp_owner.get) if sp_owner else None
    if player_id is None:
        for ts, fr in s2c:
            vals = s2c_custom(fr)
            if vals and vals[0] == 20 and len(vals) >= 2 and isinstance(vals[1], int):
                player_id = vals[1] & 0xFFFFFFFF
                break
    print('player_id=%s casts=%d' % (player_id, len(casts)))

    # 属性帧流（玩家自身 0x10，isview=0）
    attr_events = []   # (ts, props)
    for ts, fr in s2c:
        r = parse_10(fr, entries)
        if r and r[0] == 0 and r[1] == player_id:
            attr_events.append((ts, r[3]))
    print('player 0x10 frames=%d' % len(attr_events))

    def attr_at(ts, name):
        v = None
        for ats, props in attr_events:
            if ats > ts:
                break
            if name in props:
                v = props[name]
        return v

    def attr_changes(win_lo, win_hi, name):
        out = []
        for ats, props in attr_events:
            if win_lo <= ats <= win_hi and name in props:
                out.append(((ats - t0) / 1e6, props[name]))
        return out

    # 伤害/动作/移动事件
    dmg = []       # (ts, skill, value, victim, vals6th)
    m20 = []       # (ts, skill, caster)
    m169 = []      # (ts, vals 简述)
    m24other = Counter()
    for ts, fr in s2c:
        vals = s2c_custom(fr)
        if not vals:
            continue
        op = vals[0]
        if op == 24 and len(vals) >= 7 and isinstance(vals[1], int):
            if vals[2] in (17, 16, 18):
                dmg.append((ts, vals[6], vals[3], vals[1] & 0xFFFFFFFF, vals[2:6], vals[0]))
            else:
                m24other[vals[2]] += 1
        elif op == 20 and len(vals) >= 2 and isinstance(vals[1], int):
            for v in vals[2:]:
                if isinstance(v, str):
                    m20.append((ts, v, vals[1] & 0xFFFFFFFF))
                    break
        elif op == 169:
            m169.append((ts, vals[1:9]))
    print('dmg(16/17/18)=%d m20=%d m169=%d m24other=%s' % (
        len(dmg), len(m20), len(m169), dict(m24other.most_common(8))))

    # C2S 15 移动上报 / 220 / 266 / 431
    c15 = []
    cmisc = []
    for ts, fr in c2s:
        vals = c2s_custom(fr)
        if not vals:
            continue
        mid = vals[0] if vals and isinstance(vals[0], int) else None
        if mid == 15:
            c15.append((ts, vals))
        elif mid in (220, 266, 431, 54):
            cmisc.append((ts, mid, vals))

    # S2C 0x21 玩家回显（47B 记录、尾槽 5.5/9.9 特征）
    p21 = []
    for ts, fr in s2c:
        if not fr or fr[0] != 0x21 or len(fr) < 3:
            continue
        (n,) = struct.unpack_from('<H', fr, 1)
        for i in range(n):
            off = 3 + i * 44
            if off + 44 > len(fr):
                continue
            oid = struct.unpack_from('<I', fr, off)[0]
            if oid != player_id:
                continue
            dx, dy, dz, ang, spd, orient = struct.unpack_from('<6f', fr, off + 8)
            tail = struct.unpack_from('<3f', fr, off + 32)
            p21.append((ts, dx, dy, dz, spd, orient, tail))

    print('c15=%d cmisc=%d player0x21=%d' % (len(c15), len(cmisc), len(p21)))
    print()

    # ---- A. hyjf08：怒气 → 跳数 ----
    print('======== A. 寒心恨雪 %s：怒气 vs 跳数 ========' % HYJF08)
    hy_casts = [(i, c) for i, c in enumerate(casts) if c[1] == HYJF08]
    for i, (cts, skill, cvals) in hy_casts:
        lo, hi = cts - 4e6, cts + 9e6
        sp_seq = attr_changes(lo, cts + 2e6, 'SP')
        ticks = [d for d in dmg if d[0] >= cts and d[0] < hi and d[1] == skill and d[5] == 24]
        # 去同拍多目标
        dedup, prev = [], None
        for d in sorted(ticks, key=lambda x: x[0]):
            ms = (d[0] - cts) / 1e3
            if prev is not None and abs(ms - prev) < 30:
                continue
            dedup.append(d)
            prev = ms
        cval_str = ', '.join(fmt_v(v) for v in cvals[1:])
        print('#cast t=+%.3fs SP_seq=%s ticks=%d' % (
            (cts - t0) / 1e6, ['%.2fs→%d' % (t, v) for t, v in sp_seq], len(dedup)))
        print('   c2s211: [%s]' % cval_str)
        if dedup:
            offs = ['%.0f' % ((d[0] - cts) / 1e3) for d in dedup]
            print('   tick_ms=[%s] victims=%s' % (
                ', '.join(offs), Counter(d[3] for d in dedup).most_common(3)))
            print('   tick_dmg=%s' % [d[2] for d in dedup])

    # ---- B. tjj07：施法中移动 ----
    print()
    print('======== B. 太极剑舞 %s：施法窗口全事件 ========' % TJJ07)
    for i, (cts, skill, cvals) in enumerate(casts):
        if skill != TJJ07:
            continue
        lo, hi = cts - 1e6, cts + 10e6
        print('--- cast #%d t=+%.3fs ---' % (i, (cts - t0) / 1e6))
        print('   c2s211: [%s]' % ', '.join(fmt_v(v) for v in cvals[1:]))
        rows = []
        for ts, sk, caster in m20:
            if lo <= ts <= hi:
                rows.append((ts, 'm20', '%s caster=%d' % (sk, caster)))
        for ts, sk, val, victim, extra, op in dmg:
            if lo <= ts <= hi and ts >= cts:
                rows.append((ts, 'dmg', '%s v=%d victim=%d +%dms' % (sk, val, victim, (ts - cts) / 1e3)))
        for ts, vals in c15:
            if lo <= ts <= hi:
                rows.append((ts, 'c15', ', '.join(fmt_v(v) for v in vals[1:9])))
        for ts, mid, vals in cmisc:
            if lo <= ts <= hi:
                rows.append((ts, 'c%d' % mid, ', '.join(fmt_v(v) for v in vals[1:5])))
        for ts, dx, dy, dz, spd, orient, tail in p21:
            if lo <= ts <= hi:
                rows.append((ts, '0x21', 'dest=(%.1f,%.1f,%.1f) spd=%.1f tail=%s' % (
                    dx, dy, dz, spd, ['%.1f' % t for t in tail])))
        for ts, props in attr_events:
            if lo <= ts <= hi and ('LogicState' in props or 'SP' in props):
                tag = []
                if 'LogicState' in props:
                    tag.append('LogicState=%d' % props['LogicState'])
                if 'SP' in props:
                    tag.append('SP=%d' % props['SP'])
                rows.append((ts, 'attr', ' '.join(tag)))
        for ts, desc in m169:
            if lo <= ts <= hi:
                rows.append((ts, 'm169', str(desc)))
        rows.sort(key=lambda r: r[0])
        for ts, kind, desc in rows:
            print('   +%7.3fs %-5s %s' % ((ts - cts) / 1e6, kind, desc))
        # 窗口内出招序列
        inner = [(j, c) for j, c in enumerate(casts) if cts - 3e5 <= c[0] <= hi]
        print('   window casts: %s' % [(j, c[1], '%+.2fs' % ((c[0] - cts) / 1e6)) for j, c in inner])


if __name__ == '__main__':
    main(sys.argv[1])
