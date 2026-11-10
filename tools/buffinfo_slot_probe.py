# -*- coding: utf-8 -*-
"""k.pcapng BufferInfo 槽帧(官服表 631..654) 5/6 段旗标语义考证。

提取：
  A. 全部 0x10 帧中 idx∈[631,654] 的字符串槽值（CSV buffId,obj,owner,level,flagA,flagB）
  B. 全部 m169 (0x1E 首参=169) buff 帧：addRemove/buffId/施/承/v7/v8
  C. 按 (buffId,obj) 对齐槽写入与挂/摘时序，输出旗标值域分布
"""
import struct
import sys
from collections import Counter, defaultdict

sys.path.insert(0, r'G:\pojie\ai\9yin-server-go\tools\pcap_analysis')
from pcaplib import build_streams, stream_frames

PCAP = sys.argv[1] if len(sys.argv) > 1 else r'F:\Game\Online\9yin\9yin-server-go\artifacts\k.pcapng'
TABLE = r'G:\pojie\ai\9yin-server-go\internal\clientdata\official_property_table_2433.txt'

# 官服表 index -> (name, wire_type)
types = {}
with open(TABLE, 'rb') as f:
    for line in f:
        parts = line.decode('latin-1').rstrip('\r\n').split('\t')
        if len(parts) >= 3 and parts[1].isdigit():
            types[int(parts[1])] = (parts[1 + 0] if False else parts[0], int(parts[2]))
# 上面列序: index name type —— 重读一遍保证正确
types = {}
with open(TABLE, 'rb') as f:
    for line in f:
        parts = line.decode('latin-1').rstrip('\r\n').split('\t')
        if len(parts) >= 3 and parts[0].isdigit():
            types[int(parts[0])] = (parts[1], int(parts[2]))

BUFF_LO, BUFF_HI = 631, 654


def parse_props(buf, off, count):
    """0x10 属性块解包：idx u16 + 按线型取值。返回 [(idx,name,value,...)]"""
    out = []
    for _ in range(count):
        if off + 2 > len(buf):
            break
        idx = struct.unpack_from('<H', buf, off)[0]
        off += 2
        name, wt = types.get(idx, ('idx%d' % idx, -1))
        if wt == 0:
            val = buf[off] if off < len(buf) else None; off += 1; raw = ('u8', val)
        elif wt == 1:
            val = struct.unpack_from('<B', buf, off)[0] if off < len(buf) else None; off += 1; raw = ('u8', val)
        elif wt == 2:
            val = struct.unpack_from('<H', buf, off)[0] if off + 2 <= len(buf) else None; off += 2; raw = ('u16', val)
        elif wt == 3:
            val = struct.unpack_from('<i', buf, off)[0] if off + 4 <= len(buf) else None; off += 4; raw = ('i32', val)
        elif wt == 4:
            val = struct.unpack_from('<q', buf, off)[0] if off + 8 <= len(buf) else None; off += 8; raw = ('i64', val)
        elif wt == 5:
            val = struct.unpack_from('<f', buf, off)[0] if off + 4 <= len(buf) else None; off += 4; raw = ('f32', val)
        elif wt == 6:
            (n,) = struct.unpack_from('<I', buf, off) if off + 4 <= len(buf) else (None,); off += 4 if n is not None else 0
            val = buf[off:off + n].rstrip(b'\x00') if n is not None else None
            off += n if n is not None else 0
            raw = ('str', val)
        elif wt == 7:
            (n,) = struct.unpack_from('<I', buf, off) if off + 4 <= len(buf) else (None,); off += 4 if n is not None else 0
            val = buf[off:off + n].rstrip(b'\x00') if n is not None else None
            off += n if n is not None else 0
            raw = ('str', val)
        elif wt == 8:
            val = struct.unpack_from('<H', buf, off)[0] if off + 2 <= len(buf) else None; off += 2; raw = ('wstr', val)
        elif wt == 9:
            a, b = struct.unpack_from('<II', buf, off) if off + 8 <= len(buf) else ((None, None))
            off += 8; raw = ('obj', (a, b))
        else:
            raw = ('?', None)
        out.append((idx, name, raw))
    return out, off


def parse_custom(frame):
    """0x1E/0x27: [op][n u16][tag u8][val]... -> [values]"""
    if frame[0] not in (0x1E, 0x27) or len(frame) < 3:
        return None
    (n,) = struct.unpack_from('<H', frame, 1)
    off = 3
    vals = []
    for _ in range(n):
        if off >= len(frame):
            break
        tag = frame[off]; off += 1
        if tag == 2:
            (v,) = struct.unpack_from('<I', frame, off); vals.append(('i', v)); off += 4
        elif tag == 3:
            (v,) = struct.unpack_from('<q', frame, off); vals.append(('i64', v)); off += 8
        elif tag == 4:
            (v,) = struct.unpack_from('<f', frame, off); vals.append(('f32', v)); off += 4
        elif tag == 5:
            (v,) = struct.unpack_from('<d', frame, off); vals.append(('f64', v)); off += 8
        elif tag == 6:
            (ln,) = struct.unpack_from('<I', frame, off); off += 4
            vals.append(('s', frame[off:off + ln].rstrip(b'\x00'))); off += ln
        elif tag == 7:
            (ln,) = struct.unpack_from('<I', frame, off); off += 4
            vals.append(('w', frame[off:off + ln - 2])); off += ln
        elif tag == 8:
            a, b = struct.unpack_from('<II', frame, off); vals.append(('o', (a, b))); off += 8
        else:
            vals.append(('?tag%d' % tag, None)); break
    return vals


def main():
    streams = build_streams(PCAP)
    print('== streams ==')
    game = []
    for st in streams:
        s2c_n = len(st.s2c)
        if s2c_n == 0:
            continue
        frames = stream_frames(st.s2c)
        ops = Counter(f[1][0] for f in frames if f[1])
        has10 = ops.get(0x10, 0)
        print('client=%s server=%s s2c_pkts=%d frames=%d 0x10=%d' % (st.client, st.server, s2c_n, len(frames), has10))
        if has10:
            game.append((st, frames))

    for st, frames in game:
        print('\n===== game stream %s -> %s, %d frames =====' % (st.client, st.server, len(frames)))
        t0 = frames[0][0] if frames else 0
        buff_frames = []   # (ts, fno, obj, owner, isView, [(slot,name,raw)])
        m169_frames = []   # (ts, fno, vals)
        for fno, (ts, frame) in enumerate(frames):
            if not frame:
                continue
            op = frame[0]
            if op == 0x10 and len(frame) >= 12:
                isView = frame[1]
                obj, owner = struct.unpack_from('<II', frame, 2)
                (cnt,) = struct.unpack_from('<H', frame, 10)
                props, _ = parse_props(frame, 12, cnt)
                hits = [(idx, nm, raw) for idx, nm, raw in props if BUFF_LO <= idx <= BUFF_HI]
                if hits:
                    buff_frames.append((ts, fno, obj, owner, isView, hits))
            elif op == 0x0D and len(frame) >= 0x19:
                # 0x0D 档案：头 25B(obj/owner/xyz/orient) 后无 count 的连续属性块
                obj, owner = struct.unpack_from('<II', frame, 1)
                off = 0x19
                hits = []
                while off + 2 <= len(frame):
                    idx = struct.unpack_from('<H', frame, off)[0]
                    name, wt = types.get(idx, ('idx%d' % idx, -1))
                    parsed, noff = parse_props(frame, off, 1)
                    if not parsed or noff <= off:
                        break
                    pidx, pnm, praw = parsed[0]
                    if BUFF_LO <= pidx <= BUFF_HI:
                        hits.append((pidx, pnm, praw))
                    off = noff
                if hits:
                    buff_frames.append((ts, fno, obj, owner, '0D', hits))
            elif op in (0x1E, 0x27):
                vals = parse_custom(frame)
                if vals and vals[0][0] == 'i' and vals[0][1] == 169:
                    m169_frames.append((ts, fno, vals))

        print('-- BufferInfo 槽帧 %d 个, m169 %d 个 --' % (len(buff_frames), len(m169_frames)))

        print('\n-- BufferInfo 槽帧全量 --')
        flag_pairs = Counter()
        per_buff_flags = defaultdict(Counter)
        for ts, fno, obj, owner, isView, hits in buff_frames:
            t = (ts - t0) / 1e6
            for idx, nm, raw in hits:
                wt, val = raw
                if isinstance(val, bytes):
                    try:
                        s = val.decode('gbk')
                    except Exception:
                        s = repr(val)
                else:
                    s = repr(val)
                fields = s.split(',') if ',' in s else [s]
                fa = fields[4].strip() if len(fields) >= 6 else '?'
                fb = fields[5].strip() if len(fields) >= 6 else '?'
                if len(fields) >= 6:
                    flag_pairs[(fa, fb)] += 1
                    per_buff_flags[fields[0].strip()][(fa, fb, fields[3].strip())] += 1
                print('#%d t=%.3f view=%d obj=%08X owner=%08X idx=%d(%s) len=%d CSV=[%s]' % (
                    fno, t, isView, obj, owner, idx, nm, len(fields), s))

        print('\n-- (flagA,flagB) 值域分布 --')
        for k, v in flag_pairs.most_common():
            print('  %s : %d' % (k, v))
        print('\n-- 按 buffId 分组的 (flagA,flagB,level) --')
        for bid in sorted(per_buff_flags):
            print('  %s : %s' % (bid, dict(per_buff_flags[bid])))

        print('\n-- m169 全量 --')
        for ts, fno, vals in m169_frames:
            t = (ts - t0) / 1e6
            desc = []
            for tag, v in vals[1:]:
                if tag == 's' and isinstance(v, bytes):
                    try:
                        v = v.decode('gbk')
                    except Exception:
                        v = repr(v)
                if tag == 'w' and isinstance(v, bytes):
                    try:
                        v = v.decode('utf-16-le')
                    except Exception:
                        v = repr(v)
                desc.append('%s=%s' % (tag, v))
            print('#%d t=%.3f %s' % (fno, t, ' | '.join(desc)))


if __name__ == '__main__':
    main()
