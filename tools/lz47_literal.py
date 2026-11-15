# -*- coding: utf-8 -*-
"""fxnet2 mode-0x47 全字面量 LZ 编码器（2026-11-15「重登回走」终局实验）。

语义=纯字面量（零 match、零池依赖）：解码端走 0x47 LZ 解压路径但输出
逐字节等于输入。与 mode-0x46 明文等义，用于判别「跑/走」触发器是否在
解压路径本身。

解码器语义（tools/lzdec.py，逐位对齐）：
- 字面量 token：flags bit0=0；n=flags&0xf（含 bit0）；字节数 k=T[n]∈{1,2,3,4}
  （n=2→1B/耗1bit，n=4→2B/耗2bit，n=8→3B/耗3bit，n=0→4B/耗4bit）；
  字节取自压缩流游标。
- 尾段：write≥outlen-0xb 后逐字节拷贝，每字节耗 1 bit。
- flags 哨兵：寄存器==1 时重填 u32——编码端每词在「已用位数的最高位」
  置 1（哨兵位），保证消费完 token 位后寄存器恰=1 触发重填。

Usage: python -I tools/lz47_literal.py <in.bin> <out.bin>
"""
import struct
import sys

T = [4, 0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0]  # n -> 字节数
BITS = {1: 0b0010, 2: 0b0100, 3: 0b1000, 4: 0b0000}   # k -> n 低 4 位
CONSUME = {1: 1, 2: 2, 3: 3, 4: 4}                    # k -> 消耗 bit 数


class BitWord:
    """一个 32 位 flags 词：低位装 token 位，k 位哨兵=1。"""

    def __init__(self):
        self.bits = []  # 已装 token 位（0/1）

    def room(self):
        return 31 - len(self.bits)  # 最高位留哨兵

    def push(self, token_bits):
        self.bits.extend(token_bits)

    def word(self):
        w = 0
        for i, b in enumerate(self.bits):
            w |= b << i
        w |= 1 << len(self.bits)  # 哨兵位
        return w


def encode(body: bytes) -> bytes:
    out = bytearray()
    words = []  # flags 词与「词后数据游标」成对记录
    stream = bytearray()  # 字面量数据流（与词交错：词 u32 + 其 token 字节）
    cur_word = BitWord()
    cur_data = bytearray()

    def flush_word():
        nonlocal cur_word, cur_data
        words.append(struct.pack('<I', cur_word.word()))
        stream.extend(cur_data)
        cur_word = BitWord()
        cur_data = bytearray()

    def emit_literal(chunk: bytes):
        nonlocal cur_word, cur_data
        k = len(chunk)
        need = CONSUME[k]
        if cur_word.room() < need:
            flush_word()
        cur_word.push([(BITS[k] >> i) & 1 for i in range(4)])
        cur_data.extend(chunk)

    def emit_tail_byte():
        nonlocal cur_word, cur_data
        if cur_word.room() < 1:
            flush_word()
        cur_word.push([0])  # 尾字节 1 bit（值无关）
        cur_data.append(0)  # 占位：真正字节直接追加（见下）

    main = body[:len(body) - 0xb] if len(body) > 0xb else b''
    tail = body[len(body) - 0xb:] if len(body) > 0xb else b''
    i = 0
    while i < len(main):
        k = min(4, len(main) - i)
        # 组 token 不越界
        emit_literal(main[i:i + k])
        i += k
    # 尾段 11 字节逐字节（bit 流 + 原字节顺序追加）
    # 注意解码端尾字节取自游标顺序：每 bit 对应一字节，字节按序排列
    for b in tail:
        if cur_word.room() < 1:
            flush_word()
        cur_word.push([0])
        cur_data.append(b)
    flush_word()

    # 交错输出：词 u32 + 该词覆盖的数据字节 —— 但解码端词重填发生在
    # 「上一词 token 位耗尽」时，数据流是连续的；正确布局=全部词与数据
    # 交错按消费顺序：词1, 数据1, 词2, 数据2, ...
    # emit_* 已把数据按词分组，重新交错：
    out = bytearray()
    wi = 0
    di = 0
    # 重建：words[i] 对应 data 块 i —— flush 时已配对，直接交替
    # （flush_word 顺序 append，逐对输出）
    return out


def encode2(body: bytes) -> bytes:
    """正确实现：词与数据按消费顺序成对交错。"""
    events = []  # ('w', word) / ('d', bytes)
    cur_bits = []
    cur_data = bytearray()

    def flush():
        nonlocal cur_bits, cur_data
        w = 0
        for i, b in enumerate(cur_bits):
            w |= b << i
        w |= 1 << len(cur_bits)
        events.append(('w', struct.pack('<I', w)))
        if cur_data:
            events.append(('d', bytes(cur_data)))
        cur_bits = []
        cur_data = bytearray()

    main_len = max(len(body) - 0xb, 0)
    i = 0
    while i < main_len:
        k = min(4, main_len - i)
        need = CONSUME[k]
        if len(cur_bits) + need > 31:
            flush()
        n = BITS[k]
        for j in range(4):
            cur_bits.append((n >> j) & 1)
        cur_data.extend(body[i:i + k])
        i += k
    for b in body[main_len:]:
        if len(cur_bits) + 1 > 31:
            flush()
        cur_bits.append(0)
        cur_data.append(b)
    flush()
    out = bytearray()
    for _, chunk in events:
        out.extend(chunk)
    return bytes(out)


def wrap_frame(body: bytes) -> bytes:
    """[0x28][0x47][len-1 u32][bodylen u32][压缩体]。"""
    comp = encode2(body)
    frame = bytearray()
    frame.append(0x28)
    frame.append(0x47)
    frame += struct.pack('<I', 9 + len(body) + 10)  # len-1（帧总长-1），保守=体长+头
    frame += struct.pack('<I', len(body))
    frame += comp
    return bytes(frame)


if __name__ == '__main__':
    src, dst = sys.argv[1], sys.argv[2]
    body = open(src, 'rb').read()
    open(dst, 'wb').write(wrap_frame(body))
    print('body', len(body), '-> frame written', dst)
