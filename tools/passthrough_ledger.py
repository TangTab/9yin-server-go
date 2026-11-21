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
# 值: (状态, 说明)。状态 ∈ {已接线, 生成器就绪, 未移植, 已原生, 保留透传, 待查明, 暂缓}
#   已接线   = entergen 编排内实际下发（注明函数/位置）
#   生成器就绪 = entergen 有编码器但未编排进序列
#   未移植   = b-line 有生成器，尚未搬到 entergen
#   已原生   = entergen 之外已原生实现（如 msg.go 心跳）
OP_DISPOSITION = {
    0x0B: ('已接线', 'entergen.EntrySceneFrame → PreReadyFrames 首帧（idx178/1468/1447/1448，seq 每次进场自增）'),
    0x0D: ('未移植', '玩家/NPC 建对象大档（b-line sendPlayerSpawn/addObject）——当前由 0x28 化身 + 0x10 物理流承担；NPC 物化期再移植'),
    0x10: ('已接线', 'entergen.PhysicsLoginStreamFrames(×36) + basePropertyFrames + combatAttributeFrames（PreReady/Ready 内）'),
    0x11: ('已接线', 'entergen.RecordAddCellsFrame——挂载计划行推交织（mount.go rowGenerators）+ QingGongRec 逐行'),
    0x12: ('未移植', 'serverRecordDelRow（b-line shortcut_records.go:57）——顺序敏感表整表重写配套'),
    0x13: ('未移植', 'serverRecordGridCell 逐格写（b-line official_frames.go:133）——CD/fight_relation 方言'),
    0x14: ('已接线', 'MountPlan×48（mountplan.go，golden 提取）+ MountSequenceFrames（mount.go）——sub=2 全局 4 + 玩家域 44，行推交织'),
    0x15: ('已接线', 'entergen.CreateViewFrame——空容器族 26 + View40/43/46 建视图（ReadyFrames）'),
    0x16: ('未移植', 'serverDeleteView——删视图（登出/关窗期）'),
    0x17: ('生成器就绪', 'entergen.ViewPropertyFrame 已有编码器未编排（golden ×21；视图属性帧，CJ 引擎 GetViewPropertyMsg 同源）'),
    0x18: ('已接线', 'entergen.ViewAddFrame——View40(27 行)/43(6 行)/46(52 行) 行集（views.go）'),
    0x1F: ('已接线', 'entergen.LocationFrame——ReadyFrames 尾帧（25B，建号出生点）'),
    0x1E: '按消息号细分（MSG_DISPOSITION）',
    0x21: ('未移植', '对象同步帧（b-line server_moving 移动同步族）——AOI/移动期通道'),
    0x27: ('未接线', '0x47 字面量包装已由 0x29 批量通道复用（WrapAs47Literal）；独立 0x27 分节 sync 未编排'),
    0x28: ('已接线', 'entergen.AvatarFrame（0x46 明文，41 基础+27×17 技能行+CurNeiGong；literal47 备用）'),
    0x29: ('已接线', 'entergen.recordBatch0x29（Task_Accepted/CollectSkillRec，0x47 全字面量零池依赖）——真机接受度待验，透传回退'),
    0x2D: ('未移植', '压缩值消息族（b-line switch_sync_official.go 多属性批帧）'),
    0x32: ('保留透传', '会话票据 1026B×2（inner_push_golden，结构已逆向 docs §十二——会话身份类，按需镜像）'),
    0x38: ('已原生', '心跳回显 30B（fssoloworld/msg.go:110，外服形态尾巴）'),
    0x06: ('待查明', 'ready 包首帧 op06 len=1——疑似 entry 内网帧误入 golden 包，核实后剔除'),
    0xCC: ('待查明', 'ready 包次帧 opCC len=28——非正常 S2C opcode 域，同上'),
    0x31: ('待查明', 'ready 包尾帧 op31 len=11——内网寻址族 opcode，需核实是否应下发客户端'),
}

# 0x1E custom 消息号处置（状态 = entergen/fssoloworld 当前真实接线）
MSG_DISPOSITION = {
    52: ('已接线', '服务器时间 [1,unix]——entergen.ReadyInfoFrames（logininfo.go）'),
    169: ('已接线', 'buff 登记 [buff名, obj]——entergen.LoginBuffFrames（组标志+装备内功）'),
    294: ('生成器就绪', 'entergen.TimeSyncReplyFrame 已有——m924→m294 应答接线未做（fssoloworld msg.go 未挂 m924）'),
    366: ('已接线', '客户端 IP 回显——entergen.LoginInfoFrames'),
    416: ('已接线', '九阴志域声明——entergen.JyzDomainFrame(6)'),
    607: ('已接线', '账号+角色名声明——entergen.ReadyInfoFrames'),
    623: ('已接线', '[0,0,3] 服务器信息——entergen.LoginInfoFrames'),
    655: ('已接线', '服务器标识四元组——entergen.LoginInfoFrames（Session 注入）'),
    586: ('已接线', '状态推送簇首 [5]——entergen.ReadyInfoFrames'),
    229: ('已接线', 'NPC 物化帧 c130（16 槽方言定版；npcdata 摆放+模板联查→entergen.NpcListFrames，ReadyFrames 段首；见 外服NPCm229物化方言定版.md）'),
    386: ('未移植', 'UI 开关推送 [slot,id]（b-line funcopen_cluster）——ready 段 ×15'),
    681: ('未移植', '挑战令牌推送（b-line tiaozhan_probe 方言）——enter 段 ×6'),
    101: ('未移植', '面板推送（b-line gm_stats 域）'),
    136: ('未移植', '状态帧（135/136 成对，b-line switch_sync）'),
    137: ('未移植', '状态帧'),
    228: ('未移植', '数值推送'),
    312: ('未移植', '任务标记问答'),
    390: ('未移植', '数值推送'),
    395: ('暂缓', '未定位数值推送——透传保留至语义定案'),
    414: ('未移植', 'gamestep sync'),
    419: ('未移植', '未定位推送'),
    430: ('未移植', '数值推送'),
    496: ('未移植', '数值推送'),
    505: ('未移植', '数值推送'),
    596: ('未移植', '数值推送'),
    616: ('未移植', '数值推送'),
    647: ('未移植', '数值推送'),
    705: ('未移植', '个性配置串——b-line 刻意停发（回推会把客户端控制模式打回传统 3D），移植时保持停发'),
    98: ('未移植', '数值推送 [274]（b-line replay P1 同位帧）'),
    64: ('未移植', '数值推送'),
    1: ('未移植', '系统开关播报'),
    2: ('未移植', '数值推送'),
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
        # m229 族按子命令细分：c130 大族已接线；144/145/154 变体语义未定案不发。
        if mid == 229 and not desc.startswith('m229 [130]'):
            return ('未移植', 'm229 变体轻帧（[144]/[145]/[154]——语义未定案，本期不发；见 外服NPCm229物化方言定版.md §二）')
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
    lines.append('\n## 接线状态总览\n')
    lines.append('| 状态 | 帧数 | 含义 |')
    lines.append('|---|---|---|')
    status_meaning = {
        '已接线': 'entergen 编排内实际下发（native 模式生效）',
        '生成器就绪': '编码器已有，未编排进序列',
        '未移植': 'b-line 有生成器，尚未搬运',
        '未接线': '通道被等价路径复用，独立编排未做',
        '已原生': 'entergen 之外已原生实现',
        '保留透传': '会话身份类，按需镜像',
        '待查明': '语义未定，不得盲目下发',
        '暂缓': '语义未定位，透传保留',
    }
    bydisp = {}
    for (op, head, disp), n in sorted(total.items()):
        bydisp.setdefault(disp, 0)
        bydisp[disp] += n
    for disp, n in sorted(bydisp.items(), key=lambda kv: -kv[1]):
        lines.append('| %s | %d | %s |' % (disp, n, status_meaning.get(disp, '')))
    wired = bydisp.get('已接线', 0) + bydisp.get('已原生', 0)
    lines.append('')
    lines.append('native 覆盖率：**%d/%d 帧（%.0f%%）**；其余帧走 replay 模式或后续接入。'
                 % (wired, sum(bydisp.values()), 100.0 * wired / max(1, sum(bydisp.values()))))

    lines.append('\n## 生成器接线索引（entergen）\n')
    lines.append('| 帧族 | 函数 | 文件 |')
    lines.append('|---|---|---|')
    for op, func, file in (
        ('0x0B 进场', 'EntrySceneFrame', 'frames.go'),
        ('0x10 属性/物理流', 'ObjectPropertyFrame / PhysicsLoginStreamFrames', 'frames.go / physics.go'),
        ('0x11 记录行', 'RecordAddCellsFrame', 'frames.go'),
        ('0x14 挂载', 'MountSequenceFrames + MountPlan×48 + rowGenerators', 'mount.go / mountplan.go'),
        ('0x15/0x17/0x18 视图族', 'CreateViewFrame / ViewPropertyFrame / ViewAddFrame', 'frames.go'),
        ('0x18 视图行集', 'SkillViewFrames / NeiGongViewFrames / QingGongViewFrames / EmptyViewFrames', 'views.go'),
        ('0x1F 位置', 'LocationFrame', 'frames.go'),
        ('0x28 化身', 'AvatarFrame（0x46 明文，0x47 备用）', 'avatar.go'),
        ('0x29 批量', 'recordBatch0x29 + WrapAs47Literal', 'sequence.go / lz47.go'),
        ('0x1E m366/m623/m655', 'LoginInfoFrames', 'logininfo.go'),
        ('0x1E m607/m52/m586', 'ReadyInfoFrames', 'logininfo.go'),
        ('0x1E m169', 'LoginBuffFrames', 'logininfo.go'),
        ('0x1E m416', 'JyzDomainFrame', 'logininfo.go'),
        ('0x1E m294（未接线）', 'TimeSyncReplyFrame', 'logininfo.go'),
        ('编排入口', 'PreReadyFrames / ReadyFrames', 'sequence.go'),
    ):
        lines.append('| %s | `%s` | %s |' % (op, func, file))

    lines.append('\n## 后续接入清单（按优先级）\n')
    lines.append('1. m294 对钟应答接线（fssoloworld msg.go 挂 m924 → TimeSyncReplyFrame）')
    lines.append('2. 0x17 视图属性帧编排（golden ×21）')
    lines.append('3. m229 场景 NPC 列表（需场景 NPC 目录数据源，ready 段 ×63）')
    lines.append('4. m386 UI 开关推送（funcopen_cluster 移植）')
    lines.append('5. 41 条 mount-only 记录的行生成器接入（mount.go rowGenerators 注册）')
    lines.append('6. 待查明三帧（op06/0xCC/0x31）语义核实——疑似内网帧误入，确认后从 golden 包剔除')
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines) + '\n')
    print('written', OUT, len(lines), 'lines')


if __name__ == '__main__':
    main()
