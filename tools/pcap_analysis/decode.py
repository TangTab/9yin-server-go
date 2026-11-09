# -*- coding: utf-8 -*-
"""帧级解码：8 个 pcapng → 事件列表 JSON（供时间线构建）。

每帧产出: {t, dir, op, desc, sig} — sig 用于后续连续合并。
"""
import json
import os
import struct
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import build_streams, stream_frames, parse_tvarlist

DIR = r'E:\Workspace\project\game\adata\official'
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'events')

C2S_MSG = {
    1: '附近聊天', 9: '登录握手', 15: '移动上报', 30: '移动物品/穿脱', 31: '使用物品',
    34: '丢弃物品', 36: '整理背包', 54: '格挡状态', 70: '商店购买', 71: '商店出售',
    107: '外观自定义保存', 150: '经脉修炼', 161: '经脉激活', 180: '风物志卡牌',
    195: '键位方案上报', 200: '快捷键设置', 201: '快捷键删除', 211: '使用技能(出招)',
    212: '冲锋/续招落地', 215: '内功进度', 216: '使用轻功', 217: '激活轻功',
    218: '激活招架', 220: '冲锋接近上报', 240: '打坐', 243: '骑乘开关',
    244: '骑乘技能', 249: '御风蓄力结束', 266: '新骑乘子令', 292: '切换预警',
    295: '设置称号', 298: '刷新NPC任务特效', 431: '选择目标', 770: '九阴志步骤查询',
    825: '系统开关查询', 861: '武器淬火', 864: '生活职业', 914: '招式结束异步位置',
    915: '命中确认上报', 924: 'MSG_DELAY活动心跳', 933: '重铸preview', 934: '重铸lock',
    935: '重铸unlock', 938: '重铸confirm', 961: '骑乘技能(961)', 972: '新手选择',
    973: '新手指南-加入门派', 997: '门派操作', 1006: '活动内功',
}

S2C_MSG = {
    1: '系统提示', 2: '?', 4: '对话标记?', 5: 'NPC菜单', 6: 'NPC对话文本', 7: 'NPC对话对象',
    8: '对话结束', 9: '登录握手回', 15: '移动回', 18: '战斗伤害帧', 19: '锁定伤害段',
    20: '范围施放标记', 24: '通用事件', 25: '技能暂存(skill_pause)', 26: '技能释放',
    27: '?', 52: '服务器时间', 64: '?', 98: '数值推送', 100: '?',
    169: 'BufferInfo8(buff)', 189: '对象通知', 199: '?', 205: '伤害目标列表',
    229: '小数值推送', 250: '?', 252: '?', 256: '?', 259: '?',
    284: '空值标记帧', 287: '面板数据', 294: 'MSG_DELAY回显[client ms, obj, 服务器unix秒, uptime]',
    298: 'NPC任务特效回', 337: '施法回显', 382: '?', 386: 'UI开关推送',
    431: '选目标回', 446: '区域boss进度', 447: '?', 566: '令牌应答串',
    586: '状态推送(簇首)', 603: '?', 607: '?', 640: '?', 645: '状态推送', 646: '状态推送',
    648: '?', 672: '?', 681: '挑战令牌推送', 683: '?', 705: '?', 707: '受理族变体',
    726: '?', 735: '?', 740: '?', 750: '状态推送', 751: '?', 756: '?', 758: '玩家名片',
    759: '?', 772: '?', 915: '?',
}

C2S_OP = {
    0x02: '登录', 0x04: 'op04', 0x07: '对象请求', 0x09: 'op09', 0x18: 'ping',
    0x0A: 'custom信封', 0x21: 'op21', 0x31: 'op31',
}

S2C_OP = {
    0x09: '属性表0x09', 0x10: '属性更新0x10', 0x11: '记录加行0x11', 0x12: '记录删行0x12',
    0x13: '记录写格0x13', 0x15: '建视图0x15', 0x16: '删视图0x16', 0x18: '视图加行0x18',
    0x1E: 'custom0x1E', 0x1F: '位置0x1F', 0x21: '对象同步0x21', 0x27: 'custom0x27',
    0x2D: '多属性0x2D', 0x38: 'ping回0x38',
}

# 官方 0x0A 记录表（internal/clientdata/official_record_table.bin，447 schema）
def load_record_names():
    import struct as _s
    names = {}
    try:
        d = open(r'E:\Workspace\project\game\9yin-go-server1\internal\clientdata\official_record_table.bin', 'rb').read()
        cnt = _s.unpack_from('<H', d, 1)[0]
        off = 3
        for i in range(cnt):
            z = d.find(b'\x00', off)
            names[i] = d[off:z].decode('gbk', 'replace')
            ncols = _s.unpack_from('<H', d, z + 1)[0]
            off = z + 3 + ncols
    except OSError:
        pass
    return names

REC_NAMES = load_record_names()

# 视图编号已知语义（docs 散点考证）
VIEW_NAMES = {1: '(主属性)', 2: '(背包)', 3: '(任务?)', 40: '(武学技能)', 41: '(装备?)',
              43: '(内功)', 44: '(?)', 45: '(经脉激活)', 46: '(轻功)', 47: '(?)', 48: '(?)',
              109: '(CD)', 110: '(?)',
              121: '(背包-装备类)', 123: '(背包-类3)', 125: '(背包-类4)',
              165: '(?)', 170: '(?)', 174: '(?)', 175: '(?)', 176: '(?)', 177: '(?)',
              178: '(?)', 179: '(?)', 180: '(?)', 181: '(?)'}


def obj_str(v):
    return 'obj(%#x,%#x)' % (v & 0xFFFFFFFF, v >> 32)


def val_str(v):
    if isinstance(v, int) and v > 0xFFFFFFFF:
        return obj_str(v)
    if isinstance(v, str):
        return '"%s"' % v[:40]
    return str(v)


def fmt_vals(vals, limit=8):
    return '[' + ', '.join(val_str(v) for v in vals[:limit]) + (']' if len(vals) <= limit else ', …]')


def fmt_pos(v):
    if isinstance(v, float):
        return '%.1f' % v
    return str(v)


def describe_custom(direction, msg_id, vals):
    """按 msgId 生成摘要与合并签名。"""
    name = (C2S_MSG if direction == 'C2S' else S2C_MSG).get(msg_id, '?')
    # 帧形 [msgId, obj, ...] 的消息族：vals[1]=对象，vals[2]=子码
    OBJ_FAMILY = (18, 19, 20, 24, 25, 26, 27, 337, 205, 284, 707, 298, 915, 220)
    if msg_id in OBJ_FAMILY and len(vals) > 1 and isinstance(vals[1], int) and vals[1] > 0xFFFFFFFF:
        obj = vals[1]
        c1 = vals[2] if len(vals) > 2 and isinstance(vals[2], int) else None
    else:
        obj = None
        c1 = vals[1] if len(vals) > 1 and isinstance(vals[1], int) else None
    d = 'm%d %s' % (msg_id, name)
    sig = 'm%d' % msg_id
    if msg_id == 15:  # 移动上报: [15, 意图id, state, x,y,z, ...]
        if len(vals) > 2:
            state = vals[2] if isinstance(vals[2], int) else None
            mid = vals[1] if isinstance(vals[1], int) else 0
            hi = (mid >> 16) & 0xFFFF
            kind = '自动' if hi == 0xA000 else ('手动' if hi in (0x3000, 0x0000) else '?%04x' % hi)
            d += ' %s state=%s pos=(%s)' % (kind, state, ','.join(fmt_pos(v) for v in vals[3:6] if isinstance(v, (int, float))))
            sig += ':state=%s' % state
        return d, sig
    if msg_id == 211:  # 出招: [211, 技能名, x,y,z, ...]
        skill = vals[1] if len(vals) > 1 else '?'
        n = len(vals)
        d += ' %s %d值 %s' % (skill, n, fmt_vals(vals[2:6], 4))
        sig += ':%s:%d值' % (skill, n)
        return d, sig
    if msg_id == 212:
        d += ' ' + fmt_vals(vals[1:5], 4)
        return d, sig
    if msg_id == 24 and direction == 'S2C':  # [24, obj, c1, ...]
        sub = {10: '怒气段', 13: '命中?', 16: 'HP治疗飘字', 17: '伤害飘字', 18: 'MP回蓝飘字',
               29: '击退', 40: '动作开始', 41: '动作结束', 61: '冲锋动画'}.get(c1, 'c1=%s' % c1)
        d += '[%s] %s %s' % (hex(obj) if obj else '?', sub, fmt_vals(vals[3:9], 6))
        sig += ':c1=%s' % c1
        return d, sig
    if msg_id == 54:  # 格挡按下/抬起（两个魔数按会话而异，a 实证 2679078047=按下 2455038072=抬起）
        v = vals[1] if len(vals) > 1 else None
        d += '(值%s)' % v
        sig += ':v%s' % v
        return d, sig
    if msg_id == 169 and direction == 'S2C':  # [169, c1, buff名, obj, obj, 宽串名×2]
        c1v = vals[1] if len(vals) > 1 and isinstance(vals[1], int) else None
        act = {0: '添加', 1: '移除', 2: '刷新?'}.get(c1v, 'c1=%s' % c1v)
        d = 'm169 buff%s %s' % (act, fmt_vals(vals[2:4], 2))
        sig += ':c1=%s' % c1v
        return d, sig
    if msg_id == 924:
        d += ' ' + fmt_vals(vals[1:4], 3)
        return d, sig
    # 默认：展示前几个值，签名含 c1
    if obj:
        d += '[%s] %s' % (hex(obj), fmt_vals(vals[2:8], 6))
    else:
        d += ' ' + fmt_vals(vals[1:7], 6)
    if c1 is not None and msg_id in (1, 18, 19, 20, 25, 26, 337, 169, 205, 707, 284, 298, 915, 220, 27):
        sig += ':c1=%s' % c1
    return d, sig


def read_prop(fr, off, t):
    try:
        if t == 1:
            return fr[off], off + 1
        if t == 2:
            return struct.unpack_from('<H', fr, off)[0], off + 2
        if t == 3:
            return struct.unpack_from('<i', fr, off)[0], off + 4
        if t == 5:
            return round(struct.unpack_from('<f', fr, off)[0], 2), off + 4
        if t == 7:
            n = struct.unpack_from('<I', fr, off)[0]
            off += 4
            return fr[off:off + n].rstrip(b'\x00').decode('gbk', 'replace'), off + n
        if t == 4:
            return struct.unpack_from('<q', fr, off)[0], off + 8
    except (struct.error, IndexError):
        return '<溢出>', off
    return None, off


class Table09:
    """0x09 属性表：idx → (name, wire_type)。每个会话独立。

    无 0x09 帧的中途开抓会话，回退到 testdata/official_property_table_09_7074.txt
    （自 c.pcapng 抽取的官服现行表，文档实证 7071/7074 双线一致）。
    """

    FALLBACK = r'E:\Workspace\project\game\9yin-go-server1\testdata\official_property_table_09_7074.txt'

    def __init__(self):
        self.names = {}  # idx -> name
        self.types = {}  # idx -> type
        try:
            with open(self.FALLBACK, 'rb') as f:
                for line in f:
                    parts = line.decode('utf-8', 'replace').rstrip('\r\n').split('\t')
                    if len(parts) >= 3 and parts[0].strip().isdigit():
                        idx = int(parts[0])
                        self.names[idx] = parts[1]
                        self.types[idx] = int(parts[2])
        except OSError:
            pass

    def feed(self, fr):
        try:
            cnt = struct.unpack_from('<H', fr, 1)[0]
        except struct.error:
            return
        off = 3
        for i in range(cnt):
            z = fr.find(b'\x00', off)
            if z < 0:
                return
            try:
                name = fr[off:z].decode('gbk', 'replace')
                t = fr[z + 1]
            except IndexError:
                return
            self.names[i] = name
            self.types[i] = t
            off = z + 2


def describe_s2c(fr, tbl):
    op = fr[0]
    n = len(fr)
    base = S2C_OP.get(op, 'op%02X' % op)
    if op == 0x09:
        return base, 'op09'
    if op == 0x10:
        if n < 12:
            return base + '(短)', 'op10'
        is_view = fr[1]
        obj, owner, cnt = struct.unpack_from('<IIH', fr, 2)
        off = 12
        props = []
        for _ in range(cnt):
            if off + 2 > n:
                break
            idx = struct.unpack_from('<H', fr, off)[0]
            off += 2
            t = tbl.types.get(idx, 3)
            v, off = read_prop(fr, off, t)
            nm = tbl.names.get(idx, '?%d' % idx)
            props.append('%s=%s' % (nm, v))
        sig = 'op10:%s:%d:%s' % ('view' if is_view else 'obj', obj if is_view else owner,
                                 ','.join(p.split('=')[0] for p in props[:6]))
        return '%s[%s#%s] %s' % (base, 'V' if is_view else 'O', hex(obj if is_view else owner), '; '.join(props[:6])), sig
    if op == 0x2D:
        if n < 11 or fr[1] != 0x01 or fr[2] != 0x00:
            sig = 'op2D?:%d' % fr[1]
            return '%s(非标子码%02X) len=%d %s' % (base, fr[1], n, fr[:16].hex()), sig
        obj, owner, cnt = struct.unpack_from('<IIH', fr, 5)
        off = 11
        props = []
        ok = off + 6 * cnt <= n
        if ok:
            for _ in range(cnt):
                idx, v = struct.unpack_from('<Hi', fr, off)
                off += 6
                if not (0 <= idx < len(tbl.names)):
                    ok = False
                    break
                props.append('%s=%d' % (tbl.names.get(idx, '?%d' % idx), v))
        if not ok:
            sig = 'op2D?:%s' % hex(obj)
            return '%s[%s](布局变体) len=%d %s' % (base, hex(obj), n, fr[:16].hex()), sig
        sig = 'op2D:%s:%s' % (hex(obj), ','.join(p.split('=')[0] for p in props[:6]))
        return '%s[%s] %s' % (base, hex(obj), '; '.join(props[:6])), sig
    if op == 0x11:
        if n < 16:
            return base + '(短)', 'op11'
        obj, owner, rec, nrows = struct.unpack_from('<IIHH', fr, 2)
        rn = REC_NAMES.get(rec, '?')
        sig = 'op11:%d:%d' % (rec, nrows)
        return '%s[%s#%d %s/%s] +%d行' % (base, rn, rec, hex(obj), hex(owner), nrows), sig
    if op == 0x12:
        if n < 14:
            return base + '(短)', 'op12'
        obj, owner, rec = struct.unpack_from('<IIH', fr, 2)
        row = struct.unpack_from('<H', fr, 10)[0] if n >= 12 else -1
        rn = REC_NAMES.get(rec, '?')
        sig = 'op12:%d:%s' % (rec, row)
        return '%s[%s#%d 行%s %s]' % (base, rn, rec, row, hex(obj)), sig
    if op == 0x13:
        if n < 17:
            return base + '(短)', 'op13'
        obj, owner, rec, nrows, row = struct.unpack_from('<IIHHH', fr, 2)
        col = fr[12]
        raw = fr[13:17]
        v = struct.unpack('<i', raw)[0] if len(raw) == 4 else None
        rn = REC_NAMES.get(rec, '?')
        sig = 'op13:%d:%d:%d' % (rec, row, col)
        return '%s[%s#%d 行%d 列%d =%s %s]' % (base, rn, rec, row, col, v, hex(obj)), sig
    if op == 0x15:
        if n < 7:
            return base + '(短)', 'op15'
        view, cap = struct.unpack_from('<HI', fr, 1)
        sig = 'op15:%d' % view
        return '%s view=%d%s cap=%d' % (base, view, VIEW_NAMES.get(view, ''), cap), sig
    if op == 0x16:
        if n < 3:
            return base + '(短)', 'op16'
        view = struct.unpack_from('<H', fr, 1)[0]
        sig = 'op16:%d' % view
        return '%s view=%d%s' % (base, view, VIEW_NAMES.get(view, '')), sig
    if op == 0x18:
        if n < 7:
            return base + '(短)', 'op18'
        view, obj_idx = struct.unpack_from('<HH', fr, 1)
        cnt = struct.unpack_from('<H', fr, 5)[0]
        sig = 'op18:%d' % view
        return '%s view=%d%s obj#%d +%d属性' % (base, view, VIEW_NAMES.get(view, ''), obj_idx, cnt), sig
    if op == 0x1F:
        if n >= 26:
            obj = struct.unpack_from('<I', fr, 2)[0]
            owner = struct.unpack_from('<I', fr, 6)[0]
            x, y, z, o = struct.unpack_from('<ffff', fr, 10)
            sig = 'op1F:%s' % hex(obj)
            return '%s[%s/%s] (%.2f,%.2f,%.2f) o=%.2f' % (base, hex(obj), hex(owner), x, y, z, o), sig
        return base + '(短len=%d)' % n, 'op1F'
    if op in (0x1E, 0x27):
        vals = parse_tvarlist(fr[1:])
        if vals and isinstance(vals[0], int):
            d, s = describe_custom('S2C', vals[0], vals)
            return 'S2C ' + d, 'S2C:' + s
        return base + '(无法解TVar)', 'S2C:op%02X?' % op
    if op == 0x21:  # [21, 01, 00, obj, owner, ...] 周期对象同步（含动作码）
        if n >= 13:
            obj, owner = struct.unpack_from('<II', fr, 3)
            b11 = fr[11]
            sig = 'op21:%s:%d' % (hex(obj), b11)
            return '%s[%s] 码%d len=%d' % (base, hex(obj), b11, n), sig
        return base + '(短)', 'op21'
    if op in (0x28, 0x29):  # 批量记录/行推送（子码 0x47）
        sub = fr[1] if n > 1 else 0
        sig = 'op%02X:%d' % (op, sub)
        return '%s(子码%02X) len=%d' % (base, sub, n), sig
    if op == 0x31:  # [31, sub u16, obj, owner]... 对象对通知
        if n >= 11:
            sub = struct.unpack_from('<H', fr, 1)[0]
            obj, owner = struct.unpack_from('<II', fr, 3)
            sig = 'op31:%d' % sub
            extra = ''
            if n >= 19:
                obj2, owner2 = struct.unpack_from('<II', fr, 11)
                extra = ' →%s' % hex(obj2)
            return '%s(sub%d) %s%s' % (base, sub, hex(obj), extra), sig
        return base + '(短)', 'op31'
    if op == 0x14:
        sig = 'op14'
        return '%s len=%d %s' % (base, n, fr[:12].hex()), sig
    sig = 'op%02X' % op
    return '%s len=%d %s' % (base, n, fr[:20].hex()), sig


def describe_c2s(fr):
    op = fr[0]
    n = len(fr)
    base = C2S_OP.get(op, 'op%02X' % op)
    if op == 0x0A and n >= 21:
        vals = parse_tvarlist(fr[21:])
        if vals and isinstance(vals[0], int):
            seq = struct.unpack_from('<I', fr, 17)[0]
            d, s = describe_custom('C2S', vals[0], vals)
            return d, 'C2S:' + s
        return base + '(无法解TVar len=%d)' % n, 'C2S:op0A?'
    if op == 0x07:
        return '%s len=%d' % (base, n), 'C2S:op07'
    if op == 0x18:
        return base, 'C2S:op18'
    sig = 'C2S:op%02X' % op
    return '%s len=%d %s' % (base, n, fr[:24].hex()), sig


def process(path, tag):
    streams = build_streams(path)
    events = []
    for st in streams:
        srv_port = st.key[3]
        if srv_port in (6001, 7070, 8030):  # 大厅/选服
            hb_c = len(st.c2s)
            hb_s = len(st.s2c)
            if srv_port == 6001:
                c2s_bytes = b''.join(p for _, p in st.c2s)
                if c2s_bytes == b'\x04\x00\x00\x00\x10\xee\x11\x11' * hb_c:
                    events.append({'t': st.c2s[0][0], 'dir': 'C2S', 'op': 'hb',
                                   'desc': '6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×%d' % hb_c,
                                   'sig': 'lobby-hb', 'end': st.c2s[-1][0]})
                    continue
        tbl = Table09()
        c2s_ev = [(ts, describe_c2s(fr), fr) for ts, fr in stream_frames(st.c2s)]
        # 0x09 表帧先喂（无论顺序，表在会话头），再用完整表重解属性帧
        for ts, fr in stream_frames(st.s2c):
            if fr and fr[0] == 0x09:
                tbl.feed(fr)
        s2c_ev = [(ts, describe_s2c(fr, tbl), fr) for ts, fr in stream_frames(st.s2c)]
        t0 = min(c2s_ev[0][0] if c2s_ev else 1e18, s2c_ev[0][0] if s2c_ev else 1e18)
        for ts, (d, s), fr in c2s_ev:
            events.append({'t': ts - t0, 'dir': 'C2S', 'desc': d, 'sig': s, 'len': len(fr)})
        for ts, (d, s), fr in s2c_ev:
            events.append({'t': ts - t0, 'dir': 'S2C', 'desc': d, 'sig': s, 'len': len(fr)})
    events.sort(key=lambda e: e['t'])
    os.makedirs(OUT, exist_ok=True)
    out_path = os.path.join(OUT, tag + '.json')
    with open(out_path, 'w', encoding='utf-8') as f:
        json.dump(events, f, ensure_ascii=False)
    return events


if __name__ == '__main__':
    for name in sorted(os.listdir(DIR)):
        if not name.endswith('.pcapng'):
            continue
        tag = name.split('.')[0]
        ev = process(os.path.join(DIR, name), tag)
        from collections import Counter
        c = Counter((e['dir'], e['sig'].split(':')[0] if ':' in e['sig'] else e['sig']) for e in ev)
        print('%s: %d events, top=%s' % (tag, len(ev), c.most_common(8)))
