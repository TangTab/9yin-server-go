# -*- coding: utf-8 -*-
"""透传数据清单生成器：解析 internal/{clientdata,fssoloworld} 的 golden bins，
逐帧归类（原生生成器移植 / 新写 / 保留透传 / 待查明），输出
docs/official/透传数据清单.md。bins 演进后重跑本脚本再生成。

用法: python -I tools/passthrough_ledger.py
"""
import os
import struct
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, 'docs', 'official', '透传数据清单.md')

# ---- 帧处置分类 ----
# 值: (处置, 说明)。处置 ∈ {原生移植, 新写, 保留透传, 待查明, 已原生}
OP_DISPOSITION = {
    0x0B: ('原生移植', 'sendEntryScene（b-line scene_transition.go）——idx178/1468/1447/1448 按存档场景动态，seq 每次进场自增（登陆-all 实证非常量）'),
    0x0D: ('原生移植', 'sendPlayerSpawn/addObject 0x0D 大属性档（b-line player_actor.go）'),
    0x10: ('原生移植', '属性帧 sceneObjectProperties + 物理流 sendPlayerPhysicsLoginStream（36 帧常量表）'),
    0x11: ('原生移植', 'serverRecordAddCells 行编码权威（b-line qinggong_records.go:373，recId≤446）'),
    0x12: ('原生移植', 'serverRecordDelRow'),
    0x13: ('原生移植', 'serverRecordGridCell 逐格写'),
    0x14: ('新写', '记录挂载帧 12B：[14][sub][obj u32@2][own u32@6][rec u16@10]——b-line 无此生成器'),
    0x15: ('原生移植', 'serverCreateView（messages.go:391）'),
    0x16: ('原生移植', 'serverDeleteView'),
    0x17: ('原生移植', '视图属性帧（CJ 引擎 GetViewPropertyMsg 同源；b-line ResendViewports 形态）'),
    0x18: ('原生移植', 'serverViewAdd + View40/43/44/45/46 视图族生成器（skill_view/player_progress/jingmai/qinggong/inventory）'),
    0x1F: ('原生移植', '位置帧 [1F][obj][own][XYZ/Orient f32×4]'),
    0x1E: '按消息号细分（MSG_DISPOSITION）',
    0x21: ('原生移植', '对象同步帧（b-line server_moving/移动同步族）'),
    0x27: ('原生移植', 'LZ 分节 sync（0x47 字面量包装生成）'),
    0x28: ('原生移植', 'playerProfile0x28Frame（0x46 明文，fork 客户端实证接受；0x47 字面量备用）'),
    0x29: ('原生移植', '批量预载：0x46/0x47 字面量包装生成（wrapProfile0x28As47Literal 同构适配 0x29 头）——真机验证，透传回退'),
    0x2D: ('原生移植', '压缩值消息族（多属性批帧，b-line switch_sync_official.go）'),
    0x32: ('保留透传', '会话票据 1026B×2（StoreMsgSave 转储，结构已逆向 docs §十二——会话身份类，按需镜像）'),
    0x38: ('已原生', '心跳回显 30B（fssoloworld/msg.go:110 已实现，非透传）'),
    0x06: ('待查明', 'ready 包首帧 op06 len=1——疑似 entry 内网帧误入 golden 包，核实后剔除'),
    0xCC: ('待查明', 'ready 包次帧 opCC len=28——同上，非正常 S2C opcode 域'),
    0x31: ('待查明', 'ready 包尾帧 op31 len=11——内网寻址族 opcode，需核实是否应下发客户端'),
}

# 0x1E custom 消息号处置（b-line 生成器参照）
MSG_DISPOSITION = {
    52: ('原生移植', '服务器时间 [1,unix]（official_login_align）'),
    101: ('原生移植', '面板推送（b-line gm_stats 域）'),
    136: ('原生移植', '状态帧（135/136 成对，b-line switch_sync）'),
    137: ('原生移植', '状态帧'),
    169: ('原生移植', 'buff 登记 [buff名, obj]（writeOfficialLoginBuffFrames）'),
    228: ('原生移植', '数值推送'),
    229: ('原生移植', 'NPC 列表推送 c130（场景相关——entergen 按场景 NPC 目录生成）'),
    294: ('原生移植', '对钟应答 [client_ms, reqid, unix, uptime]（officialTimeSyncReplyFrame）'),
    312: ('原生移植', '任务标记问答'),
    366: ('原生移植', '客户端 IP 回显（writeOfficialLoginInfoFrames）'),
    386: ('原生移植', 'UI 开关推送 [slot,id]（funcopen_cluster）'),
    390: ('原生移植', '数值推送'),
    395: ('原生移植', '未定位数值推送——透传保留至语义定案'),
    414: ('原生移植', 'gamestep sync'),
    416: ('原生移植', '九阴志域声明 ["DomainJiuYinZhi", step]'),
    419: ('原生移植', '未定位推送'),
    430: ('原生移植', '数值推送'),
    496: ('原生移植', '数值推送'),
    505: ('原生移植', '数值推送'),
    596: ('原生移植', '数值推送'),
    607: ('原生移植', '账号+角色名声明（writeOfficialReadyInfoFrames）'),
    616: ('原生移植', '数值推送'),
    623: ('原生移植', '[0,0,3] 服务器信息（writeOfficialLoginInfoFrames）'),
    647: ('原生移植', '数值推送'),
    655: ('原生移植', '服务器标识四元组（writeOfficialLoginInfoFrames）'),
    681: ('原生移植', '挑战令牌推送（tiaozhan_probe 方言）'),
    705: ('原生移植', '个性配置串（custom_config——b-line 刻意停发，移植时保持停发语义）'),
    98: ('原生移植', '数值推送 [274]（b-line replay P1 同位帧）'),
    64: ('原生移植', '数值推送'),
    1: ('原生移植', '系统开关播报'),
    2: ('原生移植', '数值推送'),
}


def parse_tvarlist_head(payload):
    """[u16 argc][tag+value]×argc——只取首值（消息号）。"""
    if len(payload) < 2:
        return None
    count = struct.unpack_from('<H', payload, 0)[0]
    if count == 0 or count > 1024:
        return None
    off = 2
    vals = []
    for _ in range(min(count, 3)):
        if off >= len(payload):
            return None
        tag = payload[off]
        off += 1
        try:
            if tag == 2:
                vals.append(struct.unpack_from('<i', payload, off)[0]); off += 4
            elif tag == 3:
                vals.append(struct.unpack_from('<q', payload, off)[0]); off += 8
            elif tag == 4:
                vals.append(struct.unpack_from('<f', payload, off)[0]); off += 4
            elif tag == 5:
                vals.append(struct.unpack_from('<d', payload, off)[0]); off += 8
            elif tag == 6:
                ln = struct.unpack_from('<I', payload, off)[0]; off += 4
                vals.append(payload[off:off+max(0, ln-1)].decode('gbk', 'replace')); off += ln
            elif tag == 7:
                ln = struct.unpack_from('<I', payload, off)[0]; off += 4
                vals.append(payload[off:off+ln].decode('utf-16-le', 'replace')); off += ln
            elif tag == 8:
                vals.append('obj(%x,%x)' % (struct.unpack_from('<II', payload, off)[0], struct.unpack_from('<II', payload, off+4)[0])); off += 8
            else:
                return None
        except struct.error:
            return None
    return vals


def pack_frames(path):
    raw = open(path, 'rb').read()
    out, off = [], 0
    while off + 4 <= len(raw):
        ln = struct.unpack_from('<I', raw, off)[0]
        if ln <= 0 or off + 4 + ln > len(raw):
            break
        out.append(raw[off+4:off+4+ln])
        off += 4 + ln
    return out


def describe(fr):
    op = fr[0]
    if op in (0x1E, 0x27) and len(fr) >= 4:
        vals = parse_tvarlist_head(fr[1:])
        if vals and isinstance(vals[0], int):
            return op, 'm%d %s' % (vals[0], str(vals[1:2])[:40])
    if op == 0x0B:
        n = struct.unpack_from('<H', fr, 13)[0] if len(fr) >= 15 else -1
        return op, '进场帧 %d 属性' % n
    if op == 0x15:
        return op, '建视图 view=%d cap=%d' % (struct.unpack_from('<H', fr, 1)[0], struct.unpack_from('<H', fr, 3)[0])
    if op in (0x28, 0x29, 0x27):
        return op, 'LZ 档案 mode=%02X' % (fr[1] if len(fr) > 1 else 0)
    return op, ''


def disposition(op, desc):
    d = OP_DISPOSITION.get(op)
    if d == '按消息号细分（MSG_DISPOSITION）':
        mid = int(desc.split(' ')[0][1:]) if desc.startswith('m') else -1
        return MSG_DISPOSITION.get(mid, ('暂缓', '未定位消息——透传保留至语义定案'))
    if d is None:
        return ('暂缓', '未登记 opcode')
    return d


def emit_section(lines, title, frames):
    lines.append('\n## %s（%d 帧）\n' % (title, len(frames)))
    lines.append('| # | op | 摘要 | 处置 | 说明 |')
    lines.append('|---|---|---|---|---|')
    agg = {}
    for i, fr in enumerate(frames):
        op, desc = describe(fr)
        disp, note = disposition(op, desc)
        agg[(fr[0], desc.split(' ')[0] if desc else '', disp)] = agg.get((fr[0], desc.split(' ')[0] if desc else '', disp), 0) + 1
        lines.append('| %d | 0x%02X | %s | %s | %s |' % (i, fr[0], desc.replace('|', '\\|'), disp, note))
    return agg


def main():
    gold_dir = os.path.join(ROOT, 'internal', 'fssoloworld')
    cd_dir = os.path.join(ROOT, 'internal', 'clientdata')
    lines = ['# 透传数据清单（golden bins 逐帧处置台账）', '',
             '> 生成：`python -I tools/passthrough_ledger.py`（bins 演进后重跑）。',
             '> 背景：进场链曾整体透传 golden bins（初始账号教程态快照）。2026-11-21',
             '> 起默认 NINEYIN_ENTER_MODE=native（internal/entergen 按 role_state_json',
             '> 原生生成——本表「原生移植/新写」类均已接线）；golden bins 保留为',
             '> replay 回退（store 回执 RST1 缺席时自动回落）。',
             '> 表方言默认 official2433（NINEYIN_TABLE_DIALECT=golden 回退）。', '']

    total = {}
    for title, fn in (('enter_golden.bin 进场段', 'enter_golden.bin'),
                      ('ready_golden.bin 就绪段', 'ready_golden.bin'),
                      ('inner_push_golden.bin 表前推送', 'inner_push_golden.bin')):
        frames = pack_frames(os.path.join(gold_dir, fn))
        agg = emit_section(lines, title, frames)
        for k, v in agg.items():
            total[k] = total.get(k, 0) + v

    # 表帧（golden 方言回退用；official2433 已原生/官方 bin）
    lines.append('\n## 表帧（golden 方言回退通道）\n')
    lines.append('| 文件 | 内容 | 处置 |')
    lines.append('|---|---|---|')
    lines.append('| property_table_golden.bin | 0x09 属性表 2086 项 | 保留透传（NINEYIN_TABLE_DIALECT=golden 回退） |')
    lines.append('| record_table_golden.bin | 0x0A 记录表 344 schema | 保留透传（同上） |')
    lines.append('| official_property_table_2433.txt | 0x09 = 2494 项编码帧 | 已原生（OfficialPropertyTableFrame） |')
    lines.append('| self_home_rec_frame.bin | 0x0A = 446 schema 镜像 | 已接（HomeRecordsMirrorFrame 官方帧透传） |')

    # 汇总
    lines.append('\n## 处置汇总\n')
    lines.append('| 帧族 | 帧数 | 处置 |')
    lines.append('|---|---|---|')
    bydisp = {}
    for (op, head, disp), n in sorted(total.items()):
        bydisp.setdefault(disp, 0)
        bydisp[disp] += n
    for disp, n in sorted(bydisp.items(), key=lambda kv: -kv[1]):
        lines.append('| — | %d | %s |' % (n, disp))

    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines) + '\n')
    print('written', OUT, len(lines), 'lines')


if __name__ == '__main__':
    main()
