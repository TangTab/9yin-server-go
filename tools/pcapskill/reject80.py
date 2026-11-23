# reject80: 超距拒绝帧 m24[目标, 80, x, y, z, f] 的末位浮点取证。只读分析。
#
# 对每条拒绝帧，取其之前该目标对象最近一条服务端位置记录（0x1F ServerLocation
# 或 0x21 ServerAllDest 条目——二者都带朝向），逐位比对末位浮点与该记录的朝向。
# 结论（女号抓包 7 条拒绝帧全部成立）：末位浮点 = 目标当前朝向（服务端口径，
# 弧度，atan2(dx,dz) 约定），前三个浮点 = 目标当前服务端位置。
#
# 用法: go run ./tools/pcapskill <file.pcapng> <port> | python -I tools/pcapskill/reject80.py
import math
import struct
import sys

sys.stdout.reconfigure(encoding="utf-8", errors="replace")

u16 = lambda b, o: struct.unpack_from("<H", b, o)[0]
u32 = lambda b, o: struct.unpack_from("<I", b, o)[0]
i32 = lambda b, o: struct.unpack_from("<i", b, o)[0]
f32 = lambda b, o: struct.unpack_from("<f", b, o)[0]

frames = []
for line in sys.stdin:
    # pcapskill 行形：`t=  195.822 S2C #2716 op=0x1E len=42   1e 07 00 …`
    tokens = line.split()
    if len(tokens) < 8 or tokens[2] != "S2C" or not tokens[1][0].isdigit():
        continue
    try:
        raw = bytes(int(x, 16) for x in tokens[6:])
    except ValueError:
        continue
    frames.append((float(tokens[1]), raw))


def location_records(raw):
    """帧内的服务端位置记录：[(obj, x, y, z, orient)]。"""
    op, out = raw[0], []
    if op == 0x1F and len(raw) == 25:
        out.append((u32(raw, 1), f32(raw, 9), f32(raw, 13), f32(raw, 17), f32(raw, 21)))
    elif op == 0x21 and len(raw) >= 3:
        for entry in range(u16(raw, 1)):
            base = 3 + entry * 44
            if base + 44 > len(raw):
                break
            out.append((u32(raw, base), f32(raw, base + 8), f32(raw, base + 12),
                        f32(raw, base + 16), f32(raw, base + 20)))
    return out


def reject_frame(raw):
    """m24[目标, 80, x, y, z, f] → (目标 obj, x, y, z, 末位浮点)；非此形态返回 None。"""
    if raw[0] != 0x1E or len(raw) != 42 or raw[3] != 2 or i32(raw, 4) != 24 or raw[8] != 8 or i32(raw, 18) != 80:
        return None
    if raw[22] != 4 or raw[27] != 4 or raw[32] != 4 or raw[37] != 4:
        return None
    return u32(raw, 9), f32(raw, 23), f32(raw, 28), f32(raw, 33), f32(raw, 38)


def body(raw):
    """0x1E m337 [obj, x, y, z, 朝向, 0]：取 obj 与朝向（也是一条位置记录），用于佐证。"""
    if raw[0] == 0x1E and len(raw) == 42 and i32(raw, 4) == 337 and raw[8] == 8:
        return u32(raw, 9), f32(raw, 33)
    return None


checked = matched = 0
for index, (ts, raw) in enumerate(frames):
    reject = reject_frame(raw)
    if reject is None:
        continue
    obj, x, y, z, tail = reject
    # 该目标最近一条服务端位置记录（0x1F / 0x21 / m337）。
    latest = None
    for ts2, raw2 in frames[:index]:
        records = location_records(raw2)
        m337 = body(raw2)
        if m337 is not None:
            records.append((m337[0], None, None, None, m337[1]))
        for rec in records:
            if rec[0] == obj:
                latest = (ts2, raw2[0], rec)
    checked += 1
    same = latest is not None and struct.pack("<f", latest[2][4]) == struct.pack("<f", tail)
    matched += same
    note = ""
    if latest is not None and latest[2][1] is not None:
        dx, dy, dz = latest[2][1] - x, latest[2][2] - y, latest[2][3] - z
        if math.hypot(dx, dz) > 0.01:
            note = "（目标在移动：拒绝帧位置比最近记录前进了 %.2fm）" % math.hypot(dx, dz)
    print("t=%8.3f 拒绝 目标=%08X/%08X 位置=(%.3f, %.3f, %.3f) 末位浮点=%r" % (ts, obj, u32(raw, 13), x, y, z, tail))
    if latest is None:
        print("           此前无该目标的服务端位置记录")
    else:
        print("           最近记录 t=%.3f op=0x%02X 朝向=%r → %s%s" % (
            latest[0], latest[1], latest[2][4], "逐位相同" if same else "不同", note))

print("\n拒绝帧 %d 条，末位浮点与目标最近服务端位置记录的朝向逐位相同 %d 条" % (checked, matched))
