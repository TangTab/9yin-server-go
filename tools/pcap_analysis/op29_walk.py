# -*- coding: utf-8 -*-
"""0x29 走查器 v2：[00][obj][own][rec u16][flag2 u16][rows u16][行×rows]
行=记录表 cols 列型（01=u8 02=u16 03=u32 04=f32 05=f64 07=串[len u32][GBK+NUL] 08=?）
位图模式（Task_Completed flag2 含 0x80）：512B 位图页。全量验证 235 t=0 帧。"""
import sys, os, struct, pickle
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')

RT = r'E:\Workspace\project\game\9yin-go-server1\internal\clientdata\official_record_table.bin'
d = open(RT, 'rb').read()
cnt = struct.unpack_from('<H', d, 1)[0]
off = 3
REC = {}
for i in range(cnt):
    z = d.find(b'\x00', off)
    ncols = struct.unpack_from('<H', d, z + 1)[0]
    REC[i] = (d[off:z].decode('gbk', 'replace'), d[z + 3:z + 3 + ncols])
    off = z + 3 + ncols

WIDTH = {1: 1, 2: 2, 3: 4, 4: 4, 5: 8, 6: 4, 8: 4, 9: 4}

def read_col(p, pos, t):
    if t in (7, 8):
        n = struct.unpack_from('<I', p, pos)[0]
        if n > 4096 or pos + 4 + n > len(p):
            raise ValueError('bad strlen %d @%#x' % (n, pos))
        s = p[pos + 4:pos + 4 + n]
        return s.rstrip(b'\x00').decode('gbk', 'replace'), pos + 4 + n
    n = WIDTH.get(t)
    if n is None:
        raise ValueError('coltype %#x' % t)
    if pos + n > len(p):
        raise ValueError('col past end')
    if t == 4:
        return round(struct.unpack_from('<f', p, pos)[0], 4), pos + n
    if t == 5:
        return round(struct.unpack_from('<d', p, pos)[0], 4), pos + n
    if t == 6:
        return p[pos:pos + n].hex(), pos + n
    return int.from_bytes(p[pos:pos + n], 'little'), pos + n

def walk(p):
    r = dict(flags=p[0])
    r['obj'], r['own'] = struct.unpack_from('<II', p, 1)
    r['rec'] = struct.unpack_from('<H', p, 9)[0]
    r['flag2'] = struct.unpack_from('<H', p, 11)[0]
    r['rows'] = struct.unpack_from('<H', p, 13)[0]
    r['name'] = REC.get(r['rec'], ('?', b''))[0]
    cols = REC.get(r['rec'], ('', b''))[1]
    pos = 15
    if r['rows'] & 0x8000:
        r['bitmap'] = len(p) - pos
        return r, None
    rows = []
    for i in range(r['rows']):
        row = []
        for t in cols:
            v, pos = read_col(p, pos, t)
            row.append(v)
        rows.append(row)
    r['leftover'] = len(p) - pos
    if r['leftover'] == 512 or r['leftover'] == 511:
        r['bitmap_appendix'] = True
        r['leftover'] = 0
    r['consumed'] = pos
    return r, rows

if __name__ == '__main__':
    OUT = os.path.dirname(os.path.abspath(__file__))
    tot = exact = bitmap = bad = 0
    badlist = []
    for tag in ['c', 'k', 'l', '登陆']:
        pkl = os.path.join(OUT, 'op29_ctx_%s.pkl' % tag)
        if not os.path.exists(pkl):  # 中间产物由 op29_export.py 生成于 tmp/pcap_analysis
            pkl = os.path.abspath(os.path.join(OUT, '..', '..', 'tmp', 'pcap_analysis', 'op29_ctx_%s.pkl' % tag))
        with open(pkl, 'rb') as f:
            ok, _ = pickle.load(f)
        for rr in ok:
            p = rr['plain']
            if p[0] != 0:
                continue
            tot += 1
            try:
                meta, rows = walk(p)
            except Exception as e:
                bad += 1
                badlist.append((tag, rr['outlen'], str(e)))
                continue
            if rows is None:
                bitmap += 1
            elif meta['leftover'] == 0:
                exact += 1
            else:
                bad += 1
                badlist.append((tag, rr['outlen'], 'leftover=%d rows=%d' % (meta['leftover'], meta['rows'])))
    print('t=0 总=%d 精确=%d 位图=%d 失败=%d' % (tot, exact, bitmap, bad))
    from collections import Counter
    c = Counter(b[1] for b in badlist)
    print('失败 outlen 分布:', dict(c.most_common(10)))
    for b in badlist[:10]:
        print('  ', b)
