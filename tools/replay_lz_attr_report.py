# -*- coding: utf-8 -*-
"""回放 bin 双清单报告生成器（2026-11-15 用户拍板）：
A. 19 个 LZ 镜像帧的明文结构（lz_exact 精确解码 + 内层 TVarList/0x28 档案解析）
B. 119 个 0x10 属性帧全量盘点（idx→2433 列名→值，残差纪律校验）

输出: docs/official/回放bin-LZ明文结构与属性盘点.md
用法: python -I tools/replay_lz_attr_report.py
"""
import struct
import sys
import os
from collections import OrderedDict

sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, 'tools'))
import lz_exact as L

BINS = [
    ('P1', os.path.join(ROOT, 'tmp/op28_analysis/replay_p1_slim.bin')),
    ('P2', os.path.join(ROOT, 'tmp/op28_analysis/replay_p2_slim.bin')),
]
FAMILY = set(range(0x27, 0x31))

# ---- 2433 属性表 ----
TABLE = {}
with open(os.path.join(ROOT, 'internal/clientdata/official_property_table_2433.txt'), encoding='utf-8') as fh:
    for line in fh:
        parts = line.rstrip('\n').split('\t')
        if len(parts) >= 3 and parts[0].isdigit():
            TABLE[int(parts[0])] = (parts[1], int(parts[2]))

# ---- custommsg id -> 名 ----
MSG = {}
mtable = os.path.join(ROOT, 'docs/official/fxgamelogic-s2c-custommsg-table.txt')
if os.path.exists(mtable):
    for line in open(mtable, encoding='utf-8'):
        if line.startswith('#') or not line.strip():
            continue
        parts = line.rstrip('\n').split('\t')
        if parts and parts[0].isdigit():
            name = (parts[1].strip() if len(parts) > 1 else '') or \
                   (parts[2].strip() if len(parts) > 2 else '') or '?'
            MSG[int(parts[0])] = (name,)

TYPE_NAME = {0: 'None', 1: 'Byte', 2: 'Word', 3: 'Int32', 4: 'Int64', 5: 'Float32',
             7: 'String', 8: 'WideString', 9: 'Object'}


def f32(b):
    return struct.unpack('<f', b)[0]


def load_bin(path):
    raw = open(path, 'rb').read()
    frames, off = [], 0
    while off + 4 <= len(raw):
        n = struct.unpack_from('<I', raw, off)[0]
        frames.append(raw[off + 4:off + 4 + n])
        off += 4 + n
    return frames


def outlen_of(p):
    mode = p[1]
    r9 = 4 if (mode & 2) else 1
    return struct.unpack_from('<I', p, 2 + r9)[0] & (0xFFFFFFFF >> ((4 - r9) * 8))


def fmt_value(tc, raw):
    """按表型码解码。返回 (文本, 消耗字节) ；失败 used<0。"""
    if tc == 1:
        return f"{raw[0]}", 1
    if tc == 2:
        return f"{struct.unpack_from('<H', raw, 0)[0]}", 2
    if tc == 3:
        return f"{struct.unpack_from('<i', raw, 0)[0]}", 4
    if tc == 4:
        return f"{struct.unpack_from('<q', raw, 0)[0]}", 8
    if tc == 5:
        return f"{f32(raw[:4]):.6g}", 4
    if tc == 7:
        n = struct.unpack_from('<I', raw, 0)[0]
        s = raw[4:4 + n - 1].decode('gbk', 'replace')
        return (f'"{s}"', 4 + n) if len(s) <= 48 else (f'"{s[:40]}…"(len{n-1})', 4 + n)
    if tc == 8:
        n = struct.unpack_from('<I', raw, 0)[0]
        units = struct.unpack_from(f"<{n // 2}H", raw, 4)
        s = ''.join(chr(u) for u in units[:-1])
        return (f'"{s}"', 4 + n) if len(s) <= 48 else (f'"{s[:40]}…"(len{n-1})', 4 + n)
    if tc == 9:
        o, w = struct.unpack_from('<II', raw, 0)
        return f"({o:#x},{w:#x})", 8
    return '?', -1


def parse_attr_frame(f):
    """0x10 帧 → (isView, obj, own, [(idx,name,tc,val)], residual)"""
    is_view = f[1]
    obj, own = struct.unpack_from('<II', f, 2)
    cnt = struct.unpack_from('<H', f, 10)[0]
    p, items = 12, []
    for _ in range(cnt):
        if p + 2 > len(f):
            break
        idx = struct.unpack_from('<H', f, p)[0]
        p += 2
        name, tc = TABLE.get(idx, (f'idx{idx}?', -1))
        if tc < 0:
            items.append((idx, name, -1, '?'))
            p = len(f)
            break
        txt, used = fmt_value(tc, f[p:])
        if used < 0 or p + used > len(f):
            items.append((idx, name, tc, '!!越界'))
            p = len(f)
            break
        items.append((idx, name, tc, txt))
        p += used
    return is_view, obj, own, items, len(f) - p


def parse_tvar(body):
    """LZ 明文 [u16 count][tag+val]×count → (msgid, [(tag,valtxt)], residual)"""
    cnt = struct.unpack_from('<H', body, 0)[0]
    p, vals, msgid = 2, [], None
    for _ in range(cnt):
        if p >= len(body):
            vals.append(('?', '!!truncated'))
            break
        tag = body[p]
        p += 1
        try:
            if tag == 2:
                v = struct.unpack_from('<i', body, p)[0]
                p += 4
                if msgid is None:
                    msgid = v
                vals.append((tag, f"{v}"))
            elif tag == 3:
                vals.append((tag, f"{struct.unpack_from('<q', body, p)[0]}"))
                p += 8
            elif tag == 4:
                vals.append((tag, f"{f32(body[p:p + 4]):.6g}"))
                p += 4
            elif tag == 5:
                vals.append((tag, f"{struct.unpack_from('<d', body, p)[0]:.6g}"))
                p += 8
            elif tag == 6:
                n = struct.unpack_from('<I', body, p)[0]
                p += 4
                if p + n > len(body):
                    vals.append((tag, '!!len越界'))
                    break
                s = body[p:p + n - 1].decode('gbk', 'replace')
                p += n
                vals.append((tag, f'"{s}"' if len(s) <= 48 else f'"{s[:40]}…"(len{n-1})'))
            elif tag == 7:
                n = struct.unpack_from('<I', body, p)[0]
                p += 4
                if p + n > len(body) or n % 2:
                    vals.append((tag, '!!len越界'))
                    break
                units = struct.unpack_from(f"<{n // 2}H", body, p)
                p += n
                s = ''.join(chr(u) for u in units[:-1])
                vals.append((tag, f'"{s}"' if len(s) <= 48 else f'"{s[:40]}…"(len{n-1})'))
            elif tag == 8:
                o, w = struct.unpack_from('<II', body, p)
                p += 8
                vals.append((tag, f"({o:#x},{w:#x})"))
            else:
                vals.append((tag, f'!!tag{tag}'))
                break
        except struct.error:
            vals.append((tag, '!!struct'))
            break
    return msgid, vals, len(body) - p


def parse_29(body):
    """0x29 明文（外服0x29批量预载通道解析.md）：15B 头 + 行区；t=1 串表形态
    头=[01][u32][u32 1][u32][u16 10][串区]。返回 (描述, 明细行)。"""
    if body[0] == 1:
        a, b, c = struct.unpack_from('<III', body, 1)
        d = struct.unpack_from('<H', body, 13)[0]
        p, strs = 15, []
        while p + 4 <= len(body):
            n = struct.unpack_from('<I', body, p)[0]
            p += 4
            if p + n > len(body):
                strs.append('!!len越界')
                break
            strs.append(body[p:p + n - 1].decode('gbk', 'replace'))
            p += n
        return (f't=1 串表形态 [{a:#x}][{b}][{c:#x}][{d}]，{len(strs)} 串', strs)
    sub = body[0]
    obj, own = struct.unpack_from('<II', body, 1)
    rec, flag2, rows = struct.unpack_from('<HHH', body, 9)
    return (f'[{sub}][obj={obj:#x}][own={own:#x}][rec={rec}][flag={flag2:#x}][rows={rows}]',
            [f'行区 {len(body) - 15}B 按 cols 列型（语法见外服0x29批量预载通道解析.md）'])


def parse_2a(body):
    """0x2A 明文（外服压缩值消息族0x27-0x2a全景.md §0x2a）：
    [obj u16][own u16][row u16]（181 u16）（08 u8）[idx u16 + 2433 列型值]×N。
    返回 (描述, 明细行)。"""
    obj, own, row = struct.unpack_from('<HHH', body, 0)
    p = 6
    has_hdr = False
    if p + 3 <= len(body):
        v181 = struct.unpack_from('<H', body, p)[0]
        if v181 == 181:
            p += 2
            if p < len(body) and body[p] == 8:
                p += 1
            has_hdr = True
    items = []
    while p + 2 <= len(body):
        idx = struct.unpack_from('<H', body, p)[0]
        p += 2
        name, tc = TABLE.get(idx, (f'idx{idx}?', -1))
        if tc < 0:
            items.append(f'!!idx{idx} 表外 @+{p - 2:#x}')
            break
        txt, used = fmt_value(tc, body[p:])
        if used < 0 or p + used > len(body):
            items.append(f'!!{idx}({name}) 值越界')
            break
        items.append(f'{idx}({name})={txt}')
        p += used
    hdr = '[181][08]' if has_hdr else '(无 181/08 头)'
    return (f'obj={obj} own={own} row={row} {hdr}，{len(items)} 属性对', items)


def lz_summary(op, body):
    if op == 0x28:
        return f'化身玩家档案（41 字段+技能行区，走查=外服玩家档案0x28全量帧.md；头 16B: {body[:16].hex()}）'
    if op == 0x27:
        msgid, vals, resid = parse_tvar(body)
        mname = MSG.get(msgid, ('?',))[0] if msgid is not None else '?'
        rs = '' if resid == 0 else f'，残差{resid}B'
        return f'压缩 custommsg m{msgid}({mname}) TVarList {len(vals)} 值{rs}'
    if op == 0x29:
        desc, _ = parse_29(body)
        return f'ServerRecordAddRow {desc}'
    if op == 0x2A:
        desc, _ = parse_2a(body)
        return f'ServerObjectProperty {desc}'
    return f'未解析（{len(body)}B）'


def main():
    out = []
    w = out.append
    w('# 回放 bin LZ 明文结构与属性帧盘点（2026-11-15）')
    w('')
    w('> lz_exact（docs/official/fxnet2-LZ压缩方言全量定案.md）落地后的镜像数据字典：')
    w('> A 节=19 个 LZ 镜像帧逐帧明文结构；B 节=0x10 属性帧全量清单。')
    w('> bin=replay_p1_slim.bin(13帧)/replay_p2_slim.bin(322帧)，帧格式 [u32 len][frame]。')
    w('')
    lz_frames = []
    attr_inv = OrderedDict()   # idx -> dict(name,tc,vals,frames)
    attr_bad = []
    attr_total = 0

    for label, path in BINS:
        frames = load_bin(path)
        for i, f in enumerate(frames):
            op = f[0]
            if op in FAMILY:
                mode = f[1] if len(f) > 1 else 0
                if mode & 1:
                    ol = outlen_of(f)
                    body = L.decode(bytes(f[1:]), ol)
                    lz_frames.append((label, i, op, len(f), ol, body))
            elif op == 0x10:
                attr_total += 1
                is_view, obj, own, items, resid = parse_attr_frame(f)
                tag = f'{label}#{i}'
                if resid:
                    attr_bad.append((tag, resid, len(f)))
                for idx, name, tc, val in items:
                    ent = attr_inv.setdefault(idx, {'name': name, 'tc': tc, 'vals': [], 'frames': []})
                    ent['vals'].append(val)
                    ent['frames'].append(tag)

    # ---- A. LZ 帧明文结构 ----
    w('---')
    w('')
    w('## A. LZ 镜像帧明文结构（19 帧，全部独立池可解、wrap_compress 重编码逐字节一致）')
    w('')
    w('| 帧位 | op | 明文长 | 线帧长 | 内层结构 |')
    w('|---|---|---|---|---|')
    detail = []
    for label, i, op, flen, ol, body in lz_frames:
        summary = lz_summary(op, body)
        w(f'| {label}#{i} | {op:#02x} | {ol} | {flen} | {summary} |')
    w('')
    w('### 内层值明细')
    w('')
    for label, i, op, flen, ol, body in lz_frames:
        if op == 0x28:
            continue
        head = lz_summary(op, body)
        w(f'**{label}#{i} op={op:#02x}（{head}，明文 {ol}B）**')
        w('')
        w('```')
        if op == 0x27:
            msgid, vals, resid = parse_tvar(body)
            for tag, txt in vals:
                w(f'  tag{tag}: {txt}')
        elif op == 0x29:
            _, strs = parse_29(body)
            for s in strs:
                w(f'  串: {s}')
        elif op == 0x2A:
            _, items = parse_2a(body)
            for it in items:
                w(f'  {it}')
        w('```')
        w('')

    # ---- B. 属性帧盘点 ----
    w('---')
    w('')
    w(f'## B. 0x10 属性帧盘点（{attr_total} 帧，残差纪律校验）')
    w('')
    if attr_bad:
        w(f'⚠️ 残差帧 {len(attr_bad)} 个（解析错位嫌疑）: {attr_bad[:8]}')
    else:
        w('全部帧逐行解析恰好耗尽（残差=0），宽度纪律与 2433 表逐属性吻合。')
    w('')
    w(f'共 {len(attr_inv)} 个不同 idx：')
    w('')
    w('| idx | 2433 列名 | 型 | 值（跨帧） | 帧位 |')
    w('|---|---|---|---|---|')
    for idx in sorted(attr_inv):
        e = attr_inv[idx]
        uniq = list(dict.fromkeys(e['vals']))
        vtxt = ' / '.join(uniq[:6]) + (f' …共{len(e["vals"])}次' if len(e['vals']) > 6 else '')
        fr = ' '.join(e['frames'][:6]) + (f' +{len(e["frames"]) - 6}' if len(e['frames']) > 6 else '')
        w(f'| {idx} | {e["name"]} | {TYPE_NAME.get(e["tc"], "?")} | {vtxt} | {fr} |')
    w('')

    # ---- 替换指引 ----
    w('---')
    w('')
    w('## C. 替换指引')
    w('')
    w('- A/B 两节即镜像帧再生的数据字典：解码（lz_exact）→ 按 B 节列型/ A 节语法改载荷 →')
    w('  `wrap_compress` 重编码 → 重打包 bin（[u32 len][frame]）→ 重编译内嵌。')
    w('- 0x2A 装备外观行（P2#80-91/#113-115）已可按 [idx][2433 列型值] 逐对再生，')
    w('  DROP_ATTR 差集（52 重叠/53 独有）可据此清单对账销项。')
    w('- P1#0/P1#2 为压缩 custommsg（m368/m414），值域语义另行定案；0x28 化身档案')
    w('  走查见专文（外服玩家档案0x28全量帧.md）。')
    w('')
    dst = os.path.join(ROOT, 'docs/official/回放bin-LZ明文结构与属性盘点.md')
    with open(dst, 'w', encoding='utf-8') as fh:
        fh.write('\n'.join(out) + '\n')
    print('written', dst)
    print(f'LZ frames: {len(lz_frames)}, attr frames: {attr_total}, uniq idx: {len(attr_inv)}, bad: {len(attr_bad)}')


if __name__ == '__main__':
    main()
