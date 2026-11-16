"""XSAM (.xskt) 骨骼+动画格式解析器 — 九阴真经.

逆向定版 (2026-11-17, 多文件解析精确到 EOF 验证):

  文件头:
    char[4] "XSAM"
    u16 version        0x1005 主流 / 0x1003 (diedown 系 8 文件, 节点少 C 字段)
    u16 = 0
    u32 = 6            恒定
    u32len+name        内部名 (GBK, 可与文件名不同)
    u32 = 0
    u32 N              总帧数 (密集采样, 无轨道头)
    f32 fps            恒 30.0
    f32×6              bbox
    u32 nRoot          恒 1

  骨骼树 (递归先序, 无字节对齐):
    u32len+name        GBK 骨名
    u32 A              4=骨骼(带轨道) / 3=挂点(无轨道)
    u32 B              A==4: 2=18B/帧(旋转+位置, 观测仅根骨) / 0=6B/帧(仅旋转); A==3: 0
    u32 C=1            仅 v0x1005 且 A==4
    f32×10             bind 局部变换: pos.xyz + quat.xyzw + scale.xyz
    if A==4: 轨道 N 帧
      帧: i16×3 欧拉角(pitch,yaw,roll)×(2π/65535), D3DX YawPitchRoll 成 quat (见 _decode_quat)
          B==2 追加 f32×3 位置 (米, 绝对局部)
    u32 childCount
    子节点 × childCount (递归)

  每个动画文件自包含完整骨架; tpose.xskt 即 N=3 的平凡动画.

  已知坑:
    v0x1003 根骨轨道为未破译的变长打包 (~50B/3帧, 混排 i16 旋转 + f32 位置),
    本解析器用「子树结构回溯验证」暴力定位轨道长度后跳过, 根骨按静态 bind 处理.
"""
from __future__ import annotations

import math
import struct
from dataclasses import dataclass, field


class XSktError(Exception):
    pass


@dataclass
class Bone:
    name: str
    kind: int                     # A: 4=骨骼 / 3=挂点
    has_pos: bool                 # B==2: 帧内含位置
    has_scale: bool = False       # B==1: 帧内含缩放 (特效骨: HAIR/Gua/xz 等)
    bind_pos: tuple = (0.0, 0.0, 0.0)
    bind_quat: tuple = (0.0, 0.0, 0.0, 1.0)   # xyzw
    bind_scale: tuple = (1.0, 1.0, 1.0)
    quats: list = field(default_factory=list)  # N × (x,y,z) 已解码 f32
    poss: list = field(default_factory=list)   # N × (x,y,z) 或空
    scales: list = field(default_factory=list)  # N × (x,y,z) 或空 (B==1)
    parent: int = -1
    children: list = field(default_factory=list)


@dataclass
class XSkt:
    version: int
    name: str
    frame_count: int
    fps: float
    bbox: tuple
    bones: list = field(default_factory=list)  # 先序平铺
    warnings: list = field(default_factory=list)

    @property
    def duration(self) -> float:
        return self.frame_count / self.fps if self.fps else 0.0

    def bone_index(self, name: str) -> int:
        for i, b in enumerate(self.bones):
            if b.name == name:
                return i
        return -1


def _decode_quat(s0: int, s1: int, s2: int) -> tuple:
    """i16×3 欧拉角 → 单位四元数 xyzw. 终版 (112/112 骨 0.0° 拟合).

    引擎铁证 (fxmodeladv_rebuilt.dll @0x52a2/0x5372/0x541f):
      angle = i16 × π × 2 ÷ 65535 (常量池 π/2/65535, ±32767 → ±π)
      call d3dx9!D3DXQuaternionRotationYawPitchRoll(out, Yaw=s1, Pitch=s0, Roll=s2)
    等价 quat = qy(yaw) · qx(pitch) · qz(roll) (tpose.xskt 全 112 骨 f0 vs bind 0.0° 验证):
      x = cy·sx·cz + sy·cx·sz
      y = sy·cx·cz − cy·sx·sz
      z = cy·cx·sz − sy·sx·cz
      w = cy·cx·cz + sy·sx·sz
    """
    pitch = s0 * (math.pi / 32767.5)
    yaw = s1 * (math.pi / 32767.5)
    roll = s2 * (math.pi / 32767.5)
    cy, sy = math.cos(yaw * 0.5), math.sin(yaw * 0.5)
    cx, sx = math.cos(pitch * 0.5), math.sin(pitch * 0.5)
    cz, sz = math.cos(roll * 0.5), math.sin(roll * 0.5)
    return (
        cy * sx * cz + sy * cx * sz,
        sy * cx * cz - cy * sx * sz,
        cy * cx * sz - sy * sx * cz,
        cy * cx * cz + sy * sx * sz,
    )


class _Walker:
    """结构遍历器: 解码数据到 Bone 树 (decode=True) 或仅算结构 (decode=False)."""

    MAX_NAME = 4096
    MAX_CHILDREN = 255

    def __init__(self, buf, decode, root_skip=None):
        self.buf = buf
        self.off = 0
        self.decode = decode
        self.root_skip = root_skip   # v0x1003 根骨轨道跳过字节数 (None=正常解)
        self.warnings = []

    def _need(self, n):
        if self.off + n > len(self.buf):
            raise XSktError(f"offset 0x{self.off:x}: 需 {n} 字节但仅剩 {len(self.buf) - self.off}")

    def u32(self):
        self._need(4)
        v = struct.unpack_from("<I", self.buf, self.off)[0]
        self.off += 4
        return v

    def f32x(self, n):
        self._need(4 * n)
        v = struct.unpack_from(f"<{n}f", self.buf, self.off)
        self.off += 4 * n
        return v

    def string(self):
        n = self.u32()
        if n > self.MAX_NAME:
            raise XSktError(f"offset 0x{self.off - 4:x}: 疑似非法串长 {n}")
        self._need(n)
        s = self.buf[self.off:self.off + n]
        self.off += n
        return s.decode("gbk", errors="replace")

    def header(self):
        buf = self.buf
        if buf[:4] != b"XSAM":
            raise XSktError("非 XSAM 文件")
        self.off = 4
        version = struct.unpack_from("<H", buf, self.off)[0]
        if version not in (0x1005, 0x1003):
            raise XSktError(f"XSAM 版本 0x{version:04x} 不支持")
        if version == 0x1003:
            # diedown 系 8 文件: 根骨 B=1 亦为 18B/帧, 但 HeadNub→HAIR 段仍有
            # 未破译变体, 走到第 8 骨必失步 — 标记不支持, 不做暴力试探
            raise XSktError("v0x1003 (diedown 系列) 布局变体未全解, 暂不支持预览")
        if struct.unpack_from("<H", buf, 6)[0] != 0:
            self.warnings.append("头 u16@6 非 0")
        self.off = 8
        if self.u32() != 6:
            self.warnings.append("头 u32@8 非 6")
        name = self.string()
        if self.u32() != 0:
            self.warnings.append("名字后 u32 非 0")
        n_frames = self.u32()
        fps = self.f32x(1)[0]
        bbox = self.f32x(6)
        n_root = self.u32()
        if not (1 <= n_root <= 64):
            raise XSktError(f"nRoot={n_root} 异常")
        return dict(version=version, name=name, frame_count=n_frames,
                    fps=fps, bbox=tuple(bbox), n_root=n_root)

    def node(self, skt, n_frames, parent, is_root):
        b = Bone(name=self.string(), kind=self.u32(), has_pos=False, parent=parent)
        b_flag = self.u32()
        if b.kind not in (3, 4):
            raise XSktError(f"骨 {b.name}: A={b.kind} 未知节点类型")
        if b.kind == 4:
            has_track = True
            if skt.version == 0x1005:
                # C = 轨道开关: 1=有轨道, 0=纯骨架参考文件 (half 系列) 无轨道
                has_track = (self.u32() != 0)
            if b_flag not in (0, 1, 2):
                raise XSktError(f"骨 {b.name}: B={b_flag} 未知轨道类型")
            b.has_pos = has_track and (b_flag == 2)   # 18B: [欧拉 i16×3][f32×3 pos]
            b.has_scale = has_track and (b_flag == 1)  # 18B: [欧拉 i16×3][f32×3 scale]
            b.bind_pos = self.f32x(3)
            b.bind_quat = self.f32x(4)
            b.bind_scale = self.f32x(3)
            if has_track:
                # 引擎铁证 (fxmodeladv @0x5372/0x541f): 轨道元素 = [欧拉 i16×3][f32×3 pos/scale]
                # 欧拉在前, f32 尾巴在后 (B==1 是缩放, 顺序与早先猜测相反)
                key = 6 if b_flag == 0 else 18
                self._need(n_frames * key)
                for _ in range(n_frames):
                    if self.decode:
                        qx, qy, qz = struct.unpack_from("<hhh", self.buf, self.off)
                        b.quats.append(_decode_quat(qx, qy, qz))
                    self.off += 6
                    if b_flag != 0:
                        if self.decode:
                            tail = struct.unpack_from("<fff", self.buf, self.off)
                            (b.poss if b_flag == 2 else b.scales).append(tail)
                        self.off += 12
        else:  # kind==3 挂点
            # B==6 (武器/粒子挂点): bind 44B (尾多一个 ~1.0 浮点) + N×18B 轨道
            # [i16×3 quat][f32×3 pos]; B==0: bind 40B 无轨道
            if b_flag == 0:
                b.bind_pos = self.f32x(3)
                b.bind_quat = self.f32x(4)
                b.bind_scale = self.f32x(3)
            else:
                b.bind_pos = self.f32x(3)
                b.bind_quat = self.f32x(4)
                b.bind_scale = self.f32x(3)
                self.f32x(1)  # 尾浮点
                self._need(n_frames * 18)
                for _ in range(n_frames):
                    if self.decode:
                        qx, qy, qz = struct.unpack_from("<hhh", self.buf, self.off)
                        b.quats.append(_decode_quat(qx, qy, qz))
                        b.poss.append(struct.unpack_from("<fff", self.buf, self.off + 6))
                    self.off += 18
                b.has_pos = True
        idx = len(skt.bones)
        skt.bones.append(b)
        if parent >= 0:
            skt.bones[parent].children.append(idx)
        n_child = self.u32()
        if n_child > self.MAX_CHILDREN:
            raise XSktError(f"骨 {b.name}: childCount={n_child} 异常")
        for _ in range(n_child):
            self.node(skt, n_frames, idx, False)
        return b

    def body(self, hdr):
        skt = XSkt(version=hdr["version"], name=hdr["name"],
                   frame_count=hdr["frame_count"], fps=hdr["fps"],
                   bbox=hdr["bbox"], warnings=self.warnings)
        for i in range(hdr["n_root"]):
            self.node(skt, hdr["frame_count"], -1, is_root=(i == 0))
        if self.off != len(self.buf):
            raise XSktError(
                f"解析后剩余 {len(self.buf) - self.off} 字节未消费 (0x{self.off:x}, 共 0x{len(self.buf):x})")
        return skt


def parse(buf: bytes, decode_tracks: bool = True) -> XSkt:
    w = _Walker(buf, decode_tracks, root_skip=None)
    return w.body(w.header())


def parse_file(path, decode_tracks: bool = True) -> XSkt:
    with open(path, "rb") as f:
        return parse(f.read(), decode_tracks)
