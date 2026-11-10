# 解析官方 mode=0x46 明文 0x28 帧 → 精确字段图（供 Go v3 生成器布局定版）
import pickle, struct, sys

WIRE_TYPES = {1:('u8',1), 2:('u16',2), 3:('u32',4), 4:('u64',8), 5:('f32',4), 7:('str',None), 8:('wstr',None), 9:('obj',4)}

def load_table():
    t = {}
    for ln in open(r'G:/pojie/ai/9yin-server-go/internal/clientdata/official_property_table_2433.txt', encoding='utf-8'):
        parts = ln.rstrip('\n').split('\t')
        if len(parts) >= 3:
            try:
                t[int(parts[0])] = (parts[1], int(parts[2]))
            except ValueError:
                pass
    return t

def parse_frame(p, table):
    assert p[0] == 0x28 and p[1] == 0x46, 'not 0x46 frame'
    ln1 = struct.unpack_from('<I', p, 2)[0]
    outlen = struct.unpack_from('<I', p, 6)[0]
    print(f'wire len={len(p)} len-1={ln1} ({ln1==len(p)-1}) outlen={outlen} ({outlen==len(p)-10})')
    body = p[10:]
    obj, own = struct.unpack_from('<II', body, 0)
    print(f'obj={obj:#x} own={own:#x}')
    # transform block: floats until we can locate the field area.
    # scan for [u16 fieldcount][first idx] pattern: first idx should be 179 (CollideRadius) per all samples
    # find offset where struct body[o:o+2] = plausible count and body[o+2:o+4]=0x00b3
    for o in range(8, 0x60, 2):
        cnt = struct.unpack_from('<H', body, o)[0]
        first = struct.unpack_from('<H', body, o+2)[0]
        if 5 <= cnt <= 60 and first == 179:
            print(f'transform block: body[8:{o}] = {o-8}B ({(o-8)//4} f32 slots)')
            fl = struct.unpack_from(f'<{(o-8)//4}f', body, 8)
            print('  floats:', ['%.4g' % x for x in fl])
            print(f'fieldcount@body[{o:#x}] = {cnt}')
            fo = o + 2
            n = 0
            while fo < len(body) and n < cnt:
                idx = struct.unpack_from('<H', body, fo)[0]
                if idx not in table:
                    print(f'  field#{n} @{fo:#x}: idx {idx:#x} NOT IN TABLE — stop')
                    break
                name, tc = table[idx]
                wt = WIRE_TYPES.get(tc)
                if wt is None:
                    print(f'  field#{n} @{fo:#x}: idx={idx} {name} typecode={tc} unmapped')
                    break
                kind, w = wt
                if kind in ('str', 'wstr'):
                    slen = struct.unpack_from('<I', body, fo+2)[0]
                    raw = body[fo+6:fo+6+slen]
                    try:
                        val = raw.decode('gbk', 'replace') if kind == 'str' else raw.decode('utf-16le', 'replace')
                    except Exception:
                        val = repr(raw)
                    print(f'  field#{n:2d} @{fo:#04x} idx={idx:4d} {name:14s} {kind} len={slen}: {val!r}')
                    fo += 6 + slen
                else:
                    fmt = {'u8':'<B','u16':'<H','u32':'<I','u64':'<Q','f32':'<f','obj':'<I'}[kind]
                    val = struct.unpack_from(fmt, body, fo+2)[0]
                    extra = f' ({val:g})' if kind == 'f32' else ''
                    print(f'  field#{n:2d} @{fo:#04x} idx={idx:4d} {name:14s} {kind} = {val}{extra}')
                    fo += 2 + w
                n += 1
            print(f'parsed {n}/{cnt} fields, end at body[{fo:#x}], body len={len(body)}')
            return
    print('field area anchor not found')

table = load_table()
with open(r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis/l_frames.pkl', 'rb') as f:
    lf = pickle.load(f)
p46 = [bytes(p) for p in lf if isinstance(p, (bytes, bytearray)) and len(p) > 10 and p[0] == 0x28 and p[1] == 0x46]
for which in (int(sys.argv[1]) if len(sys.argv) > 1 else 0,):
    print(f'\n===== 0x46 frame #{which} =====')
    parse_frame(p46[which], table)
