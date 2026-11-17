"""XVAM (.xmod) 网格格式解析器 — 九阴真经角色模型.

逆向定版 (2026-11-17, 多文件解析精确到 EOF 验证; 同日扩展身体/时装大库):

  文件头:
    char[4] "XVAM"
    u16 version            0x1002=正式导出 (全解); 0x1000=test 变体 (顶点布局未全解)
    u16 = 0
    u32 = 2                恒定
    u32 nObj               顶层对象数 (观测恒 1)
    u32len+bytes           根文件名 (可含/不含结尾 NUL)
  全局段:
    u32 = 0
    u32 P                  未解 (3/20/55/41/100…, 疑导出参数)
    f32 30.0               恒定
    f32×6                  文件 AABB min.xyz max.xyz (Y 朝上, 米)
    u32 nObj
  对象块 × nObj:
    u32len+name
    u32 classId            2=网格对象; 3=ragdoll 物理骨架 (无网格, 标不支持)
    u16 tag                0x1005 / 0x1505 (身体/时装)
    u32 = 1
    22×0x00
    f32×4 (1,1,1,1)
    u32 -1
    f32×6 对象 AABB
    u32 nSubmesh
    子网格 × nSubmesh
    对象尾: 绑定表三形态 (见 _parse_object_tail)

  子网格:
    u32len+name            材质槽名 (Standard_N / shader 串 / 与对象同名)
    u32 smIdx              0 起序号
    u16 vflags             顶点布局位域: 0x2000 位 = 每顶点 56B (32B 基础 + 24B 附加通道),
                           其余位 (0x10/0x400/0x4000/0x8000) 不改变步长
    u16 sflags             0x0 / 0x20 / 0x1020 等 (语义未解, 不影响布局)
    u32 vertCount
    vertCount × stride     stride=32: pos f32×3 @0 | normal f32×3 @12 | uv f32×2 @24
                           stride=56 (vflags&0x2000): 前 32B 同上 + 24B 附加通道
                           (附加通道 f32×8 观测均在 [-1,1], 疑 uv2/切线类, 渲染不用)
    u32 idxCount
    idxCount × u16         三角形列表 (0 基, 指向本子网格)
    vertCount × 32B        权重: 4 × (u32 骨名blob偏移, f32 weight), 空槽 (0xFFFFFFFF, 0)
    u32 blobLen            NUL 分隔骨骼名 blob
    f32×6 diffuse+ambient (恒 0.588028)
    [附加浮点 0~5 组]      specular(1,1,1)/gloss/其它 (扫描到贴图串起点为止)
    0~4 × (u32len+str)     贴图串 (以 .dds 结尾的串连续读取)

  步长选择以校验驱动: 优先按 vflags 推断, 解析失败 (结构断裂/EOF 不精确/权重偏移越界)
  则尝试候选步长重解; 全部失败 → 不支持 (ragdoll/physx 熟饪网格等).
"""
from __future__ import annotations

import struct
from dataclasses import dataclass, field


class XModError(Exception):
    pass


class UnsupportedVersion(XModError):
    pass


# 顶点步长候选: vflags bit0x2000 → 56B; 其余 32B; 校验失败时互相兜底
STRIDE_BASE = 32
STRIDE_WIDE = 56
VFLAG_WIDE = 0x2000


@dataclass
class Submesh:
    name: str
    index: int
    flags: int
    positions: list = field(default_factory=list)   # flat f32 ×3
    normals: list = field(default_factory=list)
    uvs: list = field(default_factory=list)         # flat f32 ×2
    indices: list = field(default_factory=list)     # u16 三角列表
    weights: list = field(default_factory=list)     # 每顶点 [(boneIdx, w) ×4]
    bone_names: list = field(default_factory=list)  # blob 内骨骼名 (按偏移序)
    diffuse: tuple = (0.588, 0.588, 0.588)
    ambient: tuple = (0.588, 0.588, 0.588)
    extra_material: bytes = b""                      # 6×0.588 后的附加浮点组 (语义未全解)
    textures: list = field(default_factory=list)    # 材质/贴图串 (原始)
    stride: int = STRIDE_BASE                       # 本子网格实际顶点步长 (诊断用)

    @property
    def vertex_count(self) -> int:
        return len(self.positions) // 3

    @property
    def triangle_count(self) -> int:
        return len(self.indices) // 3


@dataclass
class XObject:
    name: str
    submeshes: list = field(default_factory=list)
    binds: list = field(default_factory=list)  # (骨名, 目标名)


@dataclass
class XMod:
    version: int
    name: str
    aabb: tuple
    objects: list = field(default_factory=list)
    warnings: list = field(default_factory=list)

    @property
    def submeshes(self):
        for obj in self.objects:
            yield from obj.submeshes


class _Reader:
    __slots__ = ("buf", "off", "end", "warnings")

    def __init__(self, buf, warnings):
        self.buf = buf
        self.off = 0
        self.end = len(buf)
        self.warnings = warnings

    def _need(self, n):
        if self.off + n > self.end:
            raise XModError(f"offset 0x{self.off:x}: 需 {n} 字节但仅剩 {self.end - self.off}")

    def u8(self):
        self._need(1)
        v = self.buf[self.off]
        self.off += 1
        return v

    def u16(self):
        self._need(2)
        v = struct.unpack_from("<H", self.buf, self.off)[0]
        self.off += 2
        return v

    def i16(self):
        self._need(2)
        v = struct.unpack_from("<h", self.buf, self.off)[0]
        self.off += 2
        return v

    def u32(self):
        self._need(4)
        v = struct.unpack_from("<I", self.buf, self.off)[0]
        self.off += 4
        return v

    def f32(self):
        self._need(4)
        v = struct.unpack_from("<f", self.buf, self.off)[0]
        self.off += 4
        return v

    def peek_u32(self):
        return struct.unpack_from("<I", self.buf, self.off)[0]

    def peek_f32(self, ahead=0):
        return struct.unpack_from("<f", self.buf, self.off + ahead * 4)[0]

    def raw(self, n):
        v = self.buf[self.off:self.off + n]
        self.off += n
        return v

    def string(self) -> str:
        n = self.u32()
        if n > 4096:
            raise XModError(f"offset 0x{self.off - 4:x}: 疑似非法串长 {n}")
        b = self.raw(n)
        return b.rstrip(b"\x00").decode("gbk", errors="replace")

    def check(self, cond, msg):
        if not cond:
            self.warnings.append(f"offset 0x{self.off:x}: {msg}")


def parse(buf: bytes) -> XMod:
    """解析 .xmod 文件全部内容; 结构不符抛 XModError."""
    warnings: list = []
    r = _Reader(buf, warnings)
    if buf[:4] != b"XVAM":
        raise XModError("非 XVAM 文件")
    r.off = 4
    version = r.u16()
    if version != 0x1002:
        raise UnsupportedVersion(
            f"XVAM 版本 0x{version:04x} 暂不支持 (仅 0x1002 全解; 0x1000 为 test 变体)")
    r.check(r.u16() == 0, "头 u16@6 非 0")
    r.check(r.u32() == 2, "头 u32@8 非 2")
    n_obj = r.u32()
    r.string()                       # 根文件名
    r.check(r.u32() == 0, "全局段首 u32 非 0")
    _p = r.u32()                     # 未解字段 P
    r.check(abs(r.f32() - 30.0) < 1e-4, "全局段 f32 非 30.0")
    aabb = tuple(r.f32() for _ in range(6))
    n_obj2 = r.u32()
    r.check(n_obj2 == n_obj, f"两处 nObj 不一致 {n_obj} vs {n_obj2}")

    model = XMod(version=version, name="", aabb=aabb, warnings=warnings)
    try:
        model.objects.append(_parse_object(r, version))
    except UnsupportedVersion as e:
        if "classId=3" not in str(e):
            raise
        # class-3 开头 = 武器的 dummy/粒子点节点链 (w0_start/pose_*/L_Weapon*…),
        # 节点内部布局不规则 —— 直接扫描后面的网格对象头 (cls∈{1,2}+tag+名字段回溯校验),
        # 从候选处整文件解析验证, 通过即采用
        start = None
        for cand in _mesh_object_candidates(r.buf, r.off):
            probe = _Reader(r.buf, [])
            probe.off = cand
            try:
                obj = _parse_object(probe, version)
            except XModError:
                continue
            if probe.off == probe.end:
                start = cand
                model.objects = [obj]
                model.warnings.append(
                    f"0x{cand:x} 前有 class-3 节点链 {cand - r.off}B 已跳过")
                break
        if start is None:
            raise UnsupportedVersion(
                "class-3 节点链后未找到可解析网格对象 (纯物理/粒子辅助文件)")
        r.off = r.end          # 候选对象已整文件验证, 视为唯一对象
    else:
        for _ in range(n_obj - 1):
            model.objects.append(_parse_object(r, version))
    model.name = model.objects[0].name if model.objects else ""
    return model


def _mesh_object_candidates(buf: bytes, from_off: int):
    """生成可能的网格对象头偏移: 扫描 [u32 cls∈{1,2}][u16 tag∈{0x1005,0x1505}],
    回溯 [可打印名][u32 len==名长] 校验."""
    out = []
    for cls in (1, 2):
        for tag in (0x1005, 0x1505):
            pat = struct.pack("<IH", cls, tag)
            pos = from_off - 1
            while True:
                pos = buf.find(pat, pos + 1)
                if pos < 0 or pos > len(buf) - 4:
                    break
                # 回溯名字 (cls 前是名字, 名字前是 u32 长度)
                e = pos - 1
                s = e
                while s > from_off and 32 <= buf[s - 1] < 127 and e - s + 1 < 256:
                    s -= 1
                if e - s + 1 < 1:
                    continue
                if s < 4:
                    continue
                ln = struct.unpack_from("<I", buf, s - 4)[0]
                if ln == e - s + 1:
                    out.append(s - 4)
    return sorted(set(out))


def _parse_object(r: _Reader, version: int) -> XObject:
    start = r.off
    name = r.string()
    cls = r.u32()
    tag = r.u16()
    if cls not in (1, 2):
        # 3 = ragdoll 物理骨架 (physx 熟饪网格同此); 布局完全不同
        raise UnsupportedVersion(
            f"对象 {name}: classId={cls} 非网格对象 (3=ragdoll/physx), 不支持预览")
    r.check(tag in (0x1005, 0x1505), f"对象 tag 0x{tag:04x} 未知")
    _a = r.u32()                     # 对象 ID (1/2/大整数, 语义弱)
    r.raw(22)                        # 零区 (pivot 等)
    r.raw(16)                        # (1,1,1,1)
    r.check(r.u32() == 0xFFFFFFFF, "父索引非 -1")
    r.raw(24)                        # 对象 AABB
    n_sub = r.u32()

    # 步长策略重试: ① 每子网格本地候选搜索 (vflags 推断优先, 支持同文件混排),
    # ②~⑤ 强制全局步长兜底 (防局部校验偶然通过但整体错位). 全部失败抛最后错误.
    obj = XObject(name=name)
    last_err = None
    for force in (None, STRIDE_BASE, STRIDE_WIDE, 36, 24):
        r.off = start
        obj2 = XObject(name=name)
        try:
            _parse_object_body(r, obj2, n_sub, force)
            if r.off != r.end:
                raise XModError(f"对象 {name}: 解析后剩余 {r.end - r.off} 字节未消费 (0x{r.off:x})")
            return obj2
        except XModError as e:
            last_err = e
    raise last_err


def _parse_object_body(r: _Reader, obj: XObject, n_sub: int, force_stride):
    """按步长策略解对象体; 结构不符抛 XModError (供上层换策略重试)."""
    name = obj.name
    r.string()
    r.u32()                           # classId (已在外层校验)
    r.u16()                           # tag
    r.u32()                           # 对象 ID
    r.raw(22)
    r.raw(16)
    r.u32()                           # 父索引
    r.raw(24)                         # 对象 AABB
    n2 = r.u32()
    if n2 != n_sub:
        raise XModError(f"对象 {name}: nSub 重读不一致 {n_sub} vs {n2}")
    for i in range(n_sub):
        obj.submeshes.append(_parse_submesh(r, i, n_sub, force_stride))
    _parse_object_tail(r, name, obj)


def _parse_object_tail(r: _Reader, name: str, obj: XObject):
    """对象尾: 绑定表 (三种已知形态) + 可选节点块.

    1) 纯字符串对: nBind × [骨名][目标名]  (b_face/tab)
    2) 条目式: nBind × [骨名][目标名][u32 3][u16 0x1005][u16 flags][40/44B 变换]
       — tpose 的穴位挂点表 (百会/涌泉等), flags&0x1000 → 40B
    3) 无法识别 → 整段按不透明尾部跳过 (仅告警, 不影响网格)
    """
    start = r.off
    # 形态 1: 纯对
    try:
        n_bind = r.u32()
        if n_bind > 100000:
            raise XModError(f"nBind={n_bind} 异常")
        binds = [(r.string(), r.string()) for _ in range(n_bind)]
        _tail_nodes(r, name)
        if r.off != r.end:
            raise XModError("纯对式尾部未到 EOF")
        obj.binds.extend(binds)
        return
    except XModError:
        r.off = start
    # 形态 2: 条目式 (tpose 穴位挂点)
    try:
        n_bind = r.u32()
        if n_bind > 100000:
            raise XModError(f"nBind={n_bind} 异常")
        binds = []
        for _ in range(n_bind):
            bone = r.string()
            target = r.string()
            _three = r.u32()
            tag = r.u16()
            flags = r.u16()
            if _three != 3 or tag != 0x1005:
                raise XModError("条目头不符")
            r.raw(40 if (flags & 0x1000) else 44)
            binds.append((bone, target))
        obj.binds.extend(binds)
        if r.off != r.end:
            raise XModError("条目式尾部未到 EOF")
        r.warnings.append(f"对象 {name}: {n_bind} 条挂点表(穴位/绑定点)已跳过")
        return
    except XModError:
        r.off = start
    # 形态 3: 不透明
    tail = r.end - r.off
    r.warnings.append(f"对象 {name}: 尾部 {tail} 字节未识别, 已跳过")
    r.off = r.end


def _tail_nodes(r: _Reader, name: str):
    """绑定表后的节点块 (仅 helper 出现): nX!=0 时剩余按不透明跳过."""
    n_x = r.u32()
    if n_x != 0:
        r.warnings.append(f"对象 {name}: 尾部 nX={n_x} 不透明段 {r.end - r.off}B")
        r.off = r.end


def _parse_material(r: _Reader, sm: Submesh, expect_idx: int, total_sub: int):
    """材质浮点区 + 贴图串. ≥6×0.588 (diffuse+ambient), 其后 0~5 组附加浮点
    (specular(1,1,1)/gloss/其它), 扫描至贴图串起点; 串以 .dds 结尾连续读取."""
    floats = [r.f32() for _ in range(6)]
    sm.diffuse = tuple(floats[0:3])
    sm.ambient = tuple(floats[3:6])
    # diffuse 恒 ~0.588; ambient 可为 (1,1,1) 或 0 (man bones 等辅助件).
    # 只挡明显乱解 (NaN/巨数), 真校验靠贴图串扫描 + 子网格边界检查.
    if not all(-16.0 <= f <= 16.0 for f in floats) or abs(floats[0] - 0.588) > 0.2:
        raise XModError(f"子网格 {sm.name}: 材质区首 6 浮点异常 {[round(f, 3) for f in floats]}")
    is_last = (expect_idx == total_sub - 1)
    extra = bytearray()
    while r.off + 4 <= r.end and not _at_string_start(r):
        # 零贴图串子网格 (face_59 的 Material #488): 直接是下一子网格头或对象尾
        if not is_last and _at_next_submesh(r, expect_idx + 1):
            if extra:
                sm.extra_material = bytes(extra)
            return
        if is_last and r.peek_u32() == 0 and (r.off + 8 > r.end
                                              or struct.unpack_from("<I", r.buf, r.off + 4)[0] <= 4096):
            if extra:
                sm.extra_material = bytes(extra)
            return
        extra += r.raw(4)
        if len(extra) > 64:
            raise XModError(f"子网格 {sm.name}: 材质区后 {len(extra)}B 内找不到贴图串起点")
    if extra:
        sm.extra_material = bytes(extra)

    # 贴图串: 首串必读, 后续串以 .dds 结尾才继续 (子网格名/结构字段都不以 .dds 结尾)
    sm.textures.append(r.string())
    while r.off + 4 <= r.end:
        n = r.peek_u32()
        if 1 <= n <= 512 and r.off + 4 + n <= r.end:
            cand = r.buf[r.off + 4:r.off + 4 + n].rstrip(b"\x00")
            if cand.lower().endswith(b".dds") and all(32 <= c < 127 for c in cand):
                sm.textures.append(r.string())
                continue
        break


def _at_string_start(r: _Reader) -> bool:
    """当前位置是否为材质区串序列的合法起点:
    首串以 .dds 结尾, 或 (材质名 + 紧随的 .dds 串)."""
    n = r.peek_u32()
    if not (1 <= n <= 256) or r.off + 4 + n > r.end:
        return False
    b = r.buf[r.off + 4:r.off + 4 + n].rstrip(b"\x00")
    if not b or not all(32 <= c < 127 for c in b):
        return False
    if b.lower().endswith(b".dds"):
        return True
    o2 = r.off + 4 + n
    if o2 + 4 > r.end:
        return False
    n2 = struct.unpack_from("<I", r.buf, o2)[0]
    if not (1 <= n2 <= 256) or o2 + 4 + n2 > r.end:
        return False
    b2 = r.buf[o2 + 4:o2 + 4 + n2].rstrip(b"\x00")
    return bool(b2) and b2.lower().endswith(b".dds") and all(32 <= c < 127 for c in b2)


def _at_next_submesh(r: _Reader, next_idx: int) -> bool:
    """当前位置是否为下一子网格头: [名长][可打印名][u32 smIdx == next_idx]."""
    n = r.peek_u32()
    if not (1 <= n <= 256) or r.off + 8 + n > r.end:
        return False
    b = r.buf[r.off + 4:r.off + 4 + n].rstrip(b"\x00")
    if not b or not all(32 <= c < 127 for c in b):
        return False
    sm_idx = struct.unpack_from("<I", r.buf, r.off + 4 + n)[0]
    return sm_idx == next_idx


# 步长搜索候选 (经验序: vflags 推断优先 → 常见 → 罕见)
_STRIDE_CANDIDATES = (STRIDE_BASE, STRIDE_WIDE, 36, 24, 40, 44, 48, 52, 60, 64, 68, 72)


def _parse_submesh(r: _Reader, expect_idx: int, total_sub: int, force_stride=None) -> Submesh:
    start = r.off
    name = r.string()
    sm_idx = r.u32()
    vflags = r.u16()
    sflags = r.u16()
    if sm_idx != expect_idx:
        raise XModError(f"子网格序号 {sm_idx} != 期望 {expect_idx}")
    n_vert = r.u32()

    if force_stride is not None:
        strides = [force_stride]
    else:
        hint = STRIDE_WIDE if vflags & VFLAG_WIDE else STRIDE_BASE
        strides = [hint] + [s for s in _STRIDE_CANDIDATES if s != hint]

    last_err = None
    for stride in strides:
        # 权重块候选顺序: sflags==0x20 系 (facecloth 等) 无权重块, 先试无
        has_w_order = (False, True) if (sflags & 0xFF) == 0x20 else (True, False)
        for has_w in has_w_order:
            r.off = start
            sm = Submesh(name=name, index=sm_idx, flags=vflags)
            try:
                return _parse_submesh_body(r, sm, n_vert, stride, has_w, expect_idx, total_sub)
            except XModError as e:
                last_err = e
    raise last_err


def _parse_submesh_body(r: _Reader, sm: Submesh, n_vert: int, stride: int,
                        has_weights: bool, expect_idx: int, total_sub: int) -> Submesh:
    r.string(); r.u32(); r.u16(); r.u16()      # 头 (已由调用方读)
    n2 = r.u32()
    if n2 != n_vert:
        raise XModError(f"子网格 {sm.name}: 顶点数重读不一致")
    if n_vert > 2_000_000:
        raise XModError(f"子网格 {sm.name}: 顶点数 {n_vert} 异常")
    sm.stride = stride
    uv_off = stride - 8            # UV 恒在记录末 8B (32B: @24; 56B: @48 — 三角形 UV
                                    # 连续性实测定位, f6..f10 为切线/次级通道)
    rec = r.buf
    for vi in range(n_vert):
        p0 = r.off + vi * stride
        sm.positions.extend(struct.unpack_from("<3f", rec, p0))
        sm.normals.extend(struct.unpack_from("<3f", rec, p0 + 12))
        sm.uvs.extend(struct.unpack_from("<2f", rec, p0 + uv_off))
    r.off += n_vert * stride

    n_idx = r.u32()
    if n_idx > 4_000_000 or n_idx % 3:
        raise XModError(f"子网格 {sm.name}: 索引数 {n_idx} 异常 (布局误判)")
    r._need(n_idx * 2)
    sm.indices = list(struct.unpack_from(f"<{n_idx}H", r.buf, r.off))
    r.off += n_idx * 2
    if sm.indices and max(sm.indices) >= n_vert:
        raise XModError(f"子网格 {sm.name}: 索引越界 (max {max(sm.indices)} >= {n_vert})")

    if has_weights:
        # 权重: 每顶点 4 × (u32 blob偏移, f32 权重); 骨名 blob 在权重之后, 先记后回填
        raw_w = []
        for _ in range(n_vert):
            slots = []
            for _s in range(4):
                ref = r.u32()
                w = r.f32()
                slots.append((ref, w))
            raw_w.append(slots)

        blob_len = r.u32()
        if blob_len > r.end - r.off:
            raise XModError(f"子网格 {sm.name}: blobLen {blob_len} 越界")
        blob = r.raw(blob_len)
        names, offsets = [], {}
        cur = 0
        while cur < blob_len:
            end = blob.find(b"\x00", cur)
            if end < 0:
                end = blob_len
            nm = blob[cur:end].decode("gbk", errors="replace")
            offsets[cur] = len(names)
            names.append(nm)
            cur = end + 1
        sm.bone_names = names
        for slots in raw_w:
            out = []
            for ref, w in slots:
                if ref == 0xFFFFFFFF or abs(w) < 1e-6:
                    continue
                if ref not in offsets:
                    raise XModError(
                        f"子网格 {sm.name}: 权重骨偏移 {ref} 不在 blob ({len(offsets)} 名)")
                out.append((offsets[ref], w))
            if not out:
                out = [(0, 1.0)]  # 无权重顶点挂第 0 骨
            sm.weights.append(out)
    else:
        # 刚体网格 (无权重块): 验证当前位置即材质区 (首浮点应近 0.588 或直接贴图串)
        if not _at_string_start(r):
            v = r.peek_f32()
            if not (0.4 < v < 0.8):
                raise XModError(f"子网格 {sm.name}: 无权重假设不成立 "
                                f"(材质区首浮点 {v:.3f})")
        sm.weights = [[(0, 1.0)]] * n_vert
        sm.bone_names = []

    _parse_material(r, sm, expect_idx, total_sub)
    # 边界校验: 非末尾子网格解析完必须紧跟下一子网格头, 否则步长/权重假设有误
    if expect_idx < total_sub - 1 and not _at_next_submesh(r, expect_idx + 1):
        raise XModError(f"子网格 {sm.name}: 后续非下一子网格头 (布局误判)")
    return sm


def parse_file(path) -> XMod:
    with open(path, "rb") as f:
        return parse(f.read())
