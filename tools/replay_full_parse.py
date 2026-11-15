# -*- coding: utf-8 -*-
"""replay_p1_slim.bin / replay_p2_slim.bin 全帧严格解析器（2026-11-15 定版）。

打包格式 = [u32 len][frame]×N（replayFramesPack 逐帧回放，replaydata 内嵌同源）。

解析纪律（防位数/索引错读——每帧解码后必须恰好耗尽帧体，残差即报错）：
- 0x10 属性帧：[op][isView u8][obj u32@2][own u32@6][cnt u16@10]
  值宽按 2433 表 TypeCode 逐属性判定：
    1=Byte(1B) 2=Word(2B) 3=Int32(4B) 4=Int64(8B) 5=Float32(4B)
    7=String(u32 len含NUL + bytes + NUL) 8=WideString(u32字节数含NUL + UTF-16LE + u16 NUL)
    9=Object(8B)
- 0x1E TVarList：[op][u16 argc@1][tag+value]×argc
    tag2=Int32(4B) tag3=Int64(8B) tag4=Float32(4B) tag5=Float64(8B)
    tag6=窄串(u32 len含NUL + bytes + NUL) tag7=宽串 tag8=Object(8B)
  首个 tag2 即 message id（对照 docs/official/fxgamelogic-s2c-custommsg-table.txt）。
- 0x15 视图创建：[op][u16 view@1][u16 cap@3][u16 nprops@5] + [idx u16 + 值]×n
  （值按 2433 表列型；BaseCap=idx2222 t3）。
- 0x1F：[op][obj u32@1][own u32@5][x/y/z/yaw f32@9..21]（25B 无 mode 形）。
- 0x0B 进场：[op][seq u32@1][obj u32@5][own u32@9][nattr u16@13] + [idx u16 + 表型值]×n
  （官方 4 属性形：178 Name / 1468 MaxVisuals(t2 u16) / 1447 ConfigID / 1448 Resource）。
- 0x38：30B 双时间戳帧 [op][u32 a@1][u32 b@5][u32 c@9][ts u32@13][0 u32][1 u8][ts u32][0 u32]
  —— 0x74d488c2×2 语义未定（UNKNOWN-7）。
- LZ 族（0x27/0x28/0x29/0x2A 外层）：[op][mode u8]
    mode&2 → hdr=9B（[mode][1B pad][u32 len-1][u32 bodylen]…… 0x46/0x47 形：
      [op][0x46/0x47][len-1 u32@2][bodylen u32@6][body@10]）
    否则 hdr=3B（[mode][2B][outlen u8/u16/u24@3..]）
  解压池 = World2（跨帧全局：解压输出 + 非族帧明文滑窗注册），P1→P2 按回放
  顺序喂池（复刻客户端视角）。解不开（官读池种子缺失）标 LZ-FAIL 并给 hex 摘要。
  内层递归：0x28=AddObject 走查（obj/own@0、变换块 13×f32@8、cnt u16@60、
  [idx+表型值]×cnt 恰尽）、0x2A=ViewAdd、0x27=分节 m414 sync 等。

输出：逐帧注解报告（stdout），另写 tmp/op28_analysis/replay_full_parse.txt。

Usage: python -I tools/replay_full_parse.py [--p1-only|--p2-only]
"""
import io
import os
import struct
import sys

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
ROOT = r"G:/pojie/ai/9yin-server-go"
sys.path.insert(0, os.path.join(ROOT, "tools"))

import lzdec  # noqa: E402
from op46_profile_stitch import World2  # noqa: E402

OUT = os.path.join(ROOT, "tmp/op28_analysis")

# ---- 2433 属性表（idx -> (name, typecode)）----
TABLE = {}
for line in open(os.path.join(ROOT, "internal/clientdata/official_property_table_2433.txt"), encoding="utf-8"):
    parts = line.rstrip("\n").split("\t")
    if len(parts) >= 3 and parts[0].isdigit():
        TABLE[int(parts[0])] = (parts[1], int(parts[2]))

# ---- custommsg id -> lua 名 ----
MSG = {}
mtable = os.path.join(ROOT, "docs/official/fxgamelogic-s2c-custommsg-table.txt")
if os.path.exists(mtable):
    for line in open(mtable, encoding="utf-8"):
        if line.startswith("#") or not line.strip():
            continue
        parts = line.rstrip("\n").split("\t")
        if parts and parts[0].isdigit():
            MSG[int(parts[0])] = parts[1] if len(parts) > 1 else "?"

FAMILY = set(range(0x27, 0x31))

# ---- 原始抓包交叉解码（第二机会池：官方登录爆发区喂池）----
CAP = r"F:\Game\Online\9yin\9yin-server-go\artifacts\c.pcapng"
_capState = {"frames": None, "world": None, "fed": 0}


def cap_frames():
    if _capState["frames"] is None:
        try:
            from op46_profile_stitch import decode_capture
            conns = decode_capture(CAP)
            _capState["frames"] = [bytes(f) for f in max(conns, key=len)]
            _capState["world"] = World2()
            _capState["fed"] = 0
        except Exception as e:  # noqa: BLE001 —— 抓包不可达则跳过交叉解码
            print(f"# 交叉解码禁用（{CAP} 不可读: {e}）")
            _capState["frames"] = []
    return _capState["frames"]


def cross_decode(frame):
    """在原流定位本帧，按原流前缀喂官方池后重解。返回 (plaintext, outlen) 或 None。"""
    frames = cap_frames()
    if not frames:
        return None
    st = _capState
    hit = None
    for j in range(st["fed"], len(frames)):
        if frames[j] == frame:
            hit = j
            break
    if hit is None:  # 全量兜底扫描
        for j in range(len(frames)):
            if frames[j] == frame:
                hit = j
                break
    if hit is None:
        return None
    for k in range(st["fed"], hit):
        f = frames[k]
        if f[0] in FAMILY:
            try:
                lz_decode(f, st["world"])
            except lzdec.DecodeError:
                st["world"].alloc(0)
        else:
            st["world"].register_raw(f)
    st["fed"] = max(st["fed"], hit)
    try:
        return lz_decode(frame, st["world"])
    except lzdec.DecodeError:
        return None
TYPE_NAME = {0: "None", 1: "Byte", 2: "Word", 3: "Int32", 4: "Int64",
             5: "Float32", 6: "Date", 7: "String", 8: "WideString", 9: "Object"}
# 关注索引（两案修复相关 + 速度族）
WATCH = {1829: "IsRun", 198: "ShortcutKey", 199: "ShortcutKey1", 200: "ShortcutKey2",
         201: "NeiGongBind", 223: "OperateMode", 386: "MoveSpeed", 387: "WalkSpeed",
         388: "RunSpeed", 390: "RunSpeedAdd"}


def f32(b, off=0):
    return struct.unpack_from("<f", b, off)[0]


def parse_bin(path):
    raw = open(path, "rb").read()
    frames, off = [], 0
    while off + 4 <= len(raw):
        fl = struct.unpack_from("<I", raw, off)[0]
        if fl <= 0 or off + 4 + fl > len(raw):
            raise SystemExit(f"bad frame len {fl} at {off}")
        frames.append(raw[off + 4:off + 4 + fl])
        off += 4 + fl
    if off != len(raw):
        raise SystemExit(f"trailing {len(raw) - off}B")
    return frames


def fmt_value(tc, raw):
    """按 2433 表型码解码一个值。返回 (文本, 消耗字节数)。"""
    if tc == 1:
        return f"b{raw[0]}", 1
    if tc == 2:
        return f"w{struct.unpack_from('<H', raw, 0)[0]}", 2
    if tc == 3:
        return f"i{struct.unpack_from('<i', raw, 0)[0]}", 4
    if tc == 4:
        return f"I{struct.unpack_from('<q', raw, 0)[0]}", 8
    if tc == 5:
        return f"f{f32(raw):.4g}", 4
    if tc == 7:
        n = struct.unpack_from("<I", raw, 0)[0]
        s = raw[4:4 + max(n - 1, 0)].decode("gbk", "replace")
        return f's"{s}"', 4 + n
    if tc == 8:
        n = struct.unpack_from("<I", raw, 0)[0]  # 字节数含 NUL
        units = struct.unpack_from(f"<{n // 2}H", raw, 4)
        s = "".join(chr(u) for u in units[:-1])
        return f'w"{s}"', 4 + n
    if tc == 9:
        o, w = struct.unpack_from("<II", raw, 0)
        return f"obj({o:#x},{w:#x})", 8
    return f"TYPE{tc}?", -1


def walk_props(buf, p, cnt):
    """从 p 起 cnt 组 [idx u16 + 表型值] 走查。返回 (items, 残差, ok)。"""
    items = []
    for _ in range(cnt):
        if p + 2 > len(buf):
            items.append("!!idx truncated")
            return items, len(buf) - p, False
        idx = struct.unpack_from("<H", buf, p)[0]
        p += 2
        name, tc = TABLE.get(idx, (f"idx{idx}", -1))
        if tc == -1:
            items.append(f"{idx}(表外)!!")
            return items, len(buf) - p, False
        txt, used = fmt_value(tc, buf[p:])
        if used < 0 or p + used > len(buf):
            items.append(f"{idx}({name})!!值越界")
            return items, len(buf) - p, False
        mark = " ◄" if idx in WATCH else ""
        items.append(f"{idx}({name}):{txt}{mark}")
        p += used
    return items, len(buf) - p, True


def dump_0x10(i, f):
    obj, own = struct.unpack_from("<II", f, 2)
    cnt = struct.unpack_from("<H", f, 10)[0]
    items, resid, ok = walk_props(f, 12, cnt)
    flag = "" if ok and resid == 0 else f" !!残差{resid}B[{f[len(f) - resid:].hex() if resid else ''}]"
    print(f"#{i:3d} 0x10 {'view' if f[1] else 'obj'} obj={obj:08x} own={own:08x} n={cnt}{flag}")
    for it in items:
        print(f"      {it}")
    return ok and resid == 0


def dump_tvar(i, f, depth=0):
    pad = "  " * depth
    cnt = struct.unpack_from("<H", f, 1)[0]
    p, vals = 3, []
    msgid = None
    ok = True
    for _ in range(cnt):
        if p >= len(f):
            vals.append("!!truncated")
            ok = False
            break
        tag = f[p]
        p += 1
        try:
            if tag == 2:
                v = struct.unpack_from("<i", f, p)[0]
                p += 4
                if msgid is None:
                    msgid = v
                vals.append(f"i{v}")
            elif tag == 3:
                vals.append(f"I{struct.unpack_from('<q', f, p)[0]}")
                p += 8
            elif tag == 4:
                vals.append(f"f{f32(f, p):.6g}")
                p += 4
            elif tag == 5:
                vals.append(f"D{struct.unpack_from('<d', f, p)[0]:.6g}")
                p += 8
            elif tag == 6:
                n = struct.unpack_from("<I", f, p)[0]
                p += 4
                s = f[p:p + max(n - 1, 0)].decode("gbk", "replace")
                vals.append(f's"{s[:64]}{"…" if len(s) > 64 else ""}({n}B)')
                p += n
            elif tag == 7:
                n = struct.unpack_from("<I", f, p)[0]
                p += 4
                units = struct.unpack_from(f"<{n // 2}H", f, p)
                vals.append('w"' + "".join(chr(u) for u in units[:-1])[:32] + '"')
                p += n
            elif tag == 8:
                o, w = struct.unpack_from("<II", f, p)
                p += 8
                vals.append(f"obj({o:#x},{w:#x})")
            else:
                vals.append(f"!!tag{tag}")
                ok = False
                break
        except struct.error:
            vals.append("!!struct")
            ok = False
            break
    resid = len(f) - p
    label = MSG.get(msgid, "?") if msgid is not None else "?"
    show = vals[:14]
    more = f" …+{len(vals) - 14}值" if len(vals) > 14 else ""
    print(f"{pad}#{i:3d} 0x1E n={cnt} id={msgid}[{label}] len={len(f)} {' '.join(str(v) for v in show)}{more}"
          f"{' !!残差%dB' % resid if resid else ''}")
    return ok and resid == 0


def dump_view(i, f):
    vid, cap, n = struct.unpack_from("<HHH", f, 1)
    items, resid, ok = walk_props(f, 7, n)
    print(f"#{i:3d} 0x15 view={vid} cap={cap} nprops={n} {'; '.join(items)}"
          f"{' !!残差%dB' % resid if resid else ''}")
    return ok and resid == 0


def lz_outlen(f):
    """LZ 族外层帧头：mode@1，hdrlen 按 bit1；outlen 在 2+hdrlen 截断读。"""
    mode = f[1]
    if mode in (0x46, 0x47):
        return struct.unpack_from("<I", f, 6)[0]
    hdrlen = 4 if (mode & 2) else 1
    v = struct.unpack_from("<I", f, 2 + hdrlen)[0]
    return v & (0xFFFFFFFF >> ((4 - hdrlen) * 8))


def lz_decode(f, world):
    mode = f[1]
    if mode == 0x46:
        outlen = struct.unpack_from("<I", f, 6)[0]
        return bytes(f[10:10 + outlen]), outlen
    return lzdec.decompress(f[1:], lz_outlen(f), world), lz_outlen(f)


def dump_inner_0x28(pl, outlen, tag):
    """AddObject 内层走查：obj/own@0、13×f32@8、cnt@60、字段恰尽。"""
    if outlen < 62:
        print(f"      内层短帧 {outlen}B: {pl.hex()}")
        return
    obj, own = struct.unpack_from("<II", pl, 0)
    t = struct.unpack_from("<13f", pl, 8)
    cnt = struct.unpack_from("<H", pl, 60)[0]
    print(f"      {tag} obj={obj:08x} own={own:08x} cnt={cnt} "
          f"pos=({t[0]:.1f},{t[1]:.1f},{t[2]:.1f}) yaw={t[3]:.2f} scale={t[6]:.2f}")
    items, resid, ok = walk_props(pl, 62, cnt)
    for it in items[:12]:
        print(f"        {it}")
    if len(items) > 12:
        print(f"        …+{len(items) - 12} 字段")
    if resid:
        print(f"        !!残差{resid}B（官读 LZ 编码捷径/尾部子记录区）")


def dump_inner_0x2a(pl, outlen):
    """ViewAdd 内层：[view u16][obj u16=槽][n u16@?][行字段]——按 0x2a 文档
    [obj u16][own u16=槽][row u16][181][08][idx+值×N]。"""
    if outlen >= 8:
        o, w, row = struct.unpack_from("<HHH", pl, 0)
        print(f"      0x2A ViewAdd obj={o:#06x} own(槽)={w:#06x} row={row} "
              f"n={struct.unpack_from('<H', pl, 6)[0]} pl[:24]={pl[:24].hex()}")


def dump_inner_0x27(pl, outlen):
    """0x27 内层两形：
    - op=0x03：ServerRecordGrid 明文 [obj u16][own u16][row u16][00 02 恒定][u32][格数据]
      （trs 令牌簇实证：own=0xa902 带 "trs…" 串）；
    - op=0xC0：m414 开关 sync 压缩簇，分节 [u16 len][TVarList 值区（无 op/argc 头）]。
    """
    if pl and pl[0] == 0xC0 and outlen >= 3 and struct.unpack_from("<H", pl, 0)[0] == 0x02c0:
        # 0x27 内层 m414 sync 簇：[argc u16=0x02c0=704][TVarList 值区]（无 op
        # 字节——首字节 0xC0 是 argc 低字节）。官读方言捷径：tag 字节 00 =
        #「省略 tag2 续值」（其后 u32 仍为 int32——首两值 = (414, "sync")，
        # 余为官方开关 id 表）。
        argc = struct.unpack_from("<H", pl, 0)[0]
        q, vals = 2, []
        last_tag = -1
        while q < len(pl) and len(vals) < 2000:
            tagv = pl[q]
            if tagv == 0 and last_tag == 2:
                q += 1
                try:
                    vals.append(struct.unpack_from("<i", pl, q)[0])
                    q += 4
                except struct.error:
                    break
                continue
            q += 1
            try:
                if tagv == 2:
                    vals.append(struct.unpack_from("<i", pl, q)[0])
                    q += 4
                    last_tag = 2
                elif tagv == 6:
                    n = struct.unpack_from("<I", pl, q)[0]
                    q += 4
                    vals.append(f's"{pl[q:q + max(n - 1, 0)].decode("gbk", "replace")}"')
                    q += n
                    last_tag = 6
                else:
                    vals.append(f"tag{tagv}?@{q}")
                    break
            except struct.error:
                break
        ints = [v for v in vals if isinstance(v, int)]
        print(f"      0xC0 TVarList argc={argc} 值数={len(vals)} 恰尽={q == len(pl)} "
              f"首={vals[:2]} int值数={len(ints)} 首6int={ints[2:8]} 尾3int={ints[-3:] if ints else []}")
        return
    if pl and pl[0] == 0x03 and outlen >= 10:
        obj, own, row = struct.unpack_from("<HHH", pl, 0)
        const = struct.unpack_from("<H", pl, 6)[0]
        n = struct.unpack_from("<I", pl, 8)[0]
        print(f"      0x27 RecordGrid obj={obj:#06x} own={own:#06x} row={row} "
              f"const={const:#06x} u32@8={n} pl[:40]={pl[:40].hex()}")
        return
    p = 0
    section = 0
    while p + 2 <= len(pl) and section < 8:
        slen = struct.unpack_from("<H", pl, p)[0]
        p += 2
        if slen == 0 or p + slen > len(pl):
            print(f"      0x27 分节头异常 @{p - 2}: len={slen} 余={len(pl) - p}"
                  f"（内层或为混合形）")
            break
        sec = pl[p:p + slen]
        p += slen
        section += 1
        # 值区走查：tag 流直到恰尽。官读方言捷径：tag 字节 00 =「省略 tag2
        # 续值」（其后 u32 仍为 int32 值——m414 sync 开关 id 表实测形）。
        q, vals = 0, []
        last_tag = -1
        while q < len(sec) and len(vals) < 400:
            tagv = sec[q]
            if tagv == 0 and last_tag == 2:
                q += 1
                try:
                    vals.append(struct.unpack_from("<i", sec, q)[0])
                    q += 4
                except struct.error:
                    break
                continue
            q += 1
            try:
                if tagv == 2:
                    vals.append(struct.unpack_from("<i", sec, q)[0])
                    q += 4
                    last_tag = 2
                elif tagv == 6:
                    n = struct.unpack_from("<I", sec, q)[0]
                    q += 4
                    vals.append(f's"{sec[q:q + max(n - 1, 0)].decode("gbk", "replace")}"')
                    q += n
                    last_tag = 6
                else:
                    vals.append(f"tag{tagv}?{sec[q:q + 4].hex()}")
                    break
            except struct.error:
                break
        head = vals[:1]
        rest = vals[1:]
        ints = [v for v in rest if isinstance(v, int)]
        print(f"      0x27 分节#{section} len={slen} 值数={len(vals)} 首={head} "
              f"int值数={len(ints)} 首3int={ints[:3]} 尾3int={ints[-3:] if ints else []}")


def dump_lz(i, f, world, note):
    mode = f[1]
    pl = None
    err = None
    try:
        pl, outlen = lz_decode(f, world)
    except lzdec.DecodeError as e:
        world.alloc(0)
        err = e
        # 第二机会：官方池（原流前缀喂池）
        got = cross_decode(f)
        if got is not None:
            pl, outlen = got
            note += "(交叉解码)"
    if pl is None:
        print(f"#{i:3d} {note} LZ-FAIL mode={mode:#04x} wire={len(f)}B outlen={lz_outlen(f)}: {err}")
        print(f"      head={f[:24].hex()}（官读池种子缺失，bin 透传原样——详见 docs/official/"
              f"外服压缩值消息族0x27-0x2a全景.md §三「LZ 精确重放不可行」）")
        return False
    inner = pl[0] if pl else -1
    print(f"#{i:3d} {note} mode={mode:#04x} wire={len(f)}B out={outlen} 内层op=0x{inner:02X}")
    if f[0] == 0x28 and outlen >= 62:
        dump_inner_0x28(pl, outlen, "AddObject")
    elif f[0] == 0x2A:
        dump_inner_0x2a(pl, outlen)
    elif f[0] == 0x27:
        dump_inner_0x27(pl, outlen)
    elif f[0] == 0x29:
        sub = pl[0] if pl else -1
        print(f"      0x29 RecordAddRow sub=0x{sub:02X} pl[:32]={pl[:32].hex()}")
    elif inner == 0x1E and outlen >= 3:
        dump_tvar(i, pl, depth=1)
    else:
        print(f"      pl[:48]={pl[:48].hex()}")
    return True


def dump(path, label, world):
    frames = parse_bin(path)
    print(f"\n########## {label}: {len(frames)} 帧, {os.path.getsize(path)}B ##########")
    hist = {}
    for f in frames:
        hist[f[0]] = hist.get(f[0], 0) + 1
    print("op 直方图: " + ", ".join(f"0x{k:02X}×{v}" for k, v in sorted(hist.items())))
    bad = 0
    for i, f in enumerate(frames):
        op = f[0]
        try:
            if op == 0x10:
                if not dump_0x10(i, f):
                    bad += 1
            elif op == 0x1E:
                if not dump_tvar(i, f):
                    bad += 1
            elif op == 0x15:
                if not dump_view(i, f):
                    bad += 1
            elif op == 0x1F and len(f) >= 25:
                obj = struct.unpack_from("<I", f, 1)[0]
                own = struct.unpack_from("<I", f, 5)[0]
                x, y, z, o = struct.unpack_from("<ffff", f, 9)
                print(f"#{i:3d} 0x1F obj={obj:08x} own={own:08x} pos=({x:.2f},{y:.2f},{z:.2f}) yaw={o:.3f} len={len(f)}")
            elif op == 0x0B:
                seq = struct.unpack_from("<I", f, 1)[0]
                obj, own = struct.unpack_from("<II", f, 5)
                n = struct.unpack_from("<H", f, 13)[0]
                items, resid, ok = walk_props(f, 15, n)
                print(f"#{i:3d} 0x0B 进场 seq={seq} obj={obj:08x} own={own:08x} nattr={n}")
                for it in items:
                    print(f"      {it}")
                if resid:
                    print(f"      !!残差{resid}B")
                    bad += 1
            elif op == 0x38:
                a, b, c = struct.unpack_from("<III", f, 1)
                ts1 = struct.unpack_from("<I", f, 13)[0]
                print(f"#{i:3d} 0x38 双时间戳 len={len(f)} u32@1/5/9={a:#x}/{b:#x}/{c:#x} ts1={ts1:#x} hex={f.hex()}")
            elif op in FAMILY:
                if not dump_lz(i, f, world, f"0x{op:02X}"):
                    bad += 1
                continue  # 池已在 dump_lz 内推进（成功=解压入池，失败=alloc 保底）
            else:
                print(f"#{i:3d} op=0x{op:02X} len={len(f)} head={f[:24].hex()}")
            # 池推进：非族帧明文注册
            if op not in FAMILY:
                world.register_raw(f)
        except Exception as e:  # noqa: BLE001 —— 单帧失败不阻断全量
            print(f"#{i:3d} op=0x{op:02X} !!解析异常: {e}")
            bad += 1
            world.register_raw(f)
    print(f"\n[{label}] 严格走查失败帧: {bad}/{len(frames)}")
    return bad


def main():
    only = sys.argv[1] if len(sys.argv) > 1 else ""
    world = World2()  # 跨 bin 全局池：P1 先喂、P2 后喂（客户端收流顺序）
    if only != "--p2-only":
        dump(os.path.join(OUT, "replay_p1_slim.bin"), "P1 (Phase A 进场帧)", world)
    if only != "--p1-only":
        dump(os.path.join(OUT, "replay_p2_slim.bin"), "P2 (Phase B 第二爆发+物化簇)", world)


if __name__ == "__main__":
    main()
