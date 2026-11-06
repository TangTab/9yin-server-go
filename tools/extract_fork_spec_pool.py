# -*- coding: utf-8 -*-
"""从 fork 二进制 (9yin-game-native-menu.exe) 导出 .data 段静态属性 spec 池。

池位置（fb21001f, 2026-09-14 构建）: file 0xaa3660，2426 项 × 24B。
条目布局: {u64 name_ptr, u64 name_len, u64 wire_type}；
字符串区 VA→file 换算 delta = 0x140001000。
池顺序 == 0x09 线缆表顺序（LastObject=414/SP=467/MaxSP=580 与
官方表逐一命中）；相对官方 2370 项表 +227 新属性 / -62 旧属性，
wire 类型枚举零冲突。

用法:
  python -I extract_fork_spec_pool.py <exe路径> <输出.txt>
"""
import struct
import sys

DELTA = 0x140001000
POOL_FILE_OFFSET = 0xAA3660
TYPE_NAMES = {
    0: "none",
    1: "byte",
    2: "word",
    3: "int32",
    4: "int64",
    5: "float32",
    6: "float64?",
    7: "string",
    8: "widestring",
    9: "object",
}


def load_pool(exe_path):
    data = open(exe_path, "rb").read()
    entries = []
    off = POOL_FILE_OFFSET
    while off + 24 <= len(data):
        ptr, length, wire_type = struct.unpack_from("<QQQ", data, off)
        if not (0x140600000 <= ptr < 0x140700000) or not (1 <= length <= 64):
            break
        raw = data[ptr - DELTA : ptr - DELTA + length]
        name = raw.decode("utf-8")
        if not name.isprintable():
            break
        entries.append((name, wire_type))
        off += 24
    return entries


def main():
    exe_path = sys.argv[1] if len(sys.argv) > 1 else r"F:/Game/Online/9yin/9yin_server1/build/9yin-game-native-menu.exe"
    out_path = sys.argv[2] if len(sys.argv) > 2 else "fork_property_table.txt"
    entries = load_pool(exe_path)
    if len(entries) != 2426:
        print(f"warning: expected 2426 entries, parsed {len(entries)}", file=sys.stderr)
    with open(out_path, "w", encoding="utf-8", newline="\n") as fh:
        fh.write("# fork 9yin-game-native-menu.exe .data 静态属性 spec 池导出\n")
        fh.write(f"# 源: {exe_path} (commit fb21001f, 2026-09-14) file@0xaa3660, {len(entries)} 项\n")
        fh.write("# 顺序 == 0x09 线缆表顺序; type: " + " ".join(f"{k}={v}" for k, v in TYPE_NAMES.items()) + "\n")
        for index, (name, wire_type) in enumerate(entries):
            fh.write(f"{index}\t{name}\t{wire_type}\n")
    print(f"exported {len(entries)} entries -> {out_path}")


if __name__ == "__main__":
    main()
