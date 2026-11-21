"""DDS 头探测 — 只读像素格式/尺寸信息 (数据由前端 DDSLoader 解码).

九阴角色贴图全部为传统 DDS (无 DX10 扩展头), FourCC ∈ {DXT1, DXT3, DXT5}.
"""
from __future__ import annotations

import struct

FOURCC_MAP = {b"DXT1": "DXT1", b"DXT2": "DXT2", b"DXT3": "DXT3",
              b"DXT4": "DXT4", b"DXT5": "DXT5"}


def probe(buf: bytes) -> dict:
    """返回 {fourcc, width, height, mips} 或抛 ValueError."""
    if buf[:4] != b"DDS ":
        raise ValueError("非 DDS 文件")
    size = struct.unpack_from("<I", buf, 4)[0]
    if size != 124:
        raise ValueError(f"DDS 头大小 {size} 非 124 (疑 DX10)")
    flags, height, width = struct.unpack_from("<III", buf, 8)
    mips = struct.unpack_from("<I", buf, 28)[0] or 1  # dwMipMapCount@28
    pf_flags = struct.unpack_from("<I", buf, 80)[0]
    fourcc = buf[84:88]
    if pf_flags & 0x4 and fourcc in FOURCC_MAP:
        fmt = FOURCC_MAP[fourcc]
    elif pf_flags & 0x40:  # DDSPF_RGB
        fmt = "RGB"
    else:
        fmt = f"0x{fourcc:08x}"
    return {"fourcc": fmt, "width": width, "height": height, "mips": mips}


def probe_file(path) -> dict:
    with open(path, "rb") as f:
        return probe(f.read(128))
