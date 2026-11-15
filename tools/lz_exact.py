# -*- coding: utf-8 -*-
"""fxnet2 LZ 编解码器——按 fxnet2_rebuilt.dll（fxgame.DMP 运行时抠取，2026-11-15）
逐指令反汇编定案的精确实现。旧 tools/lzdec.py 按磁盘版+近似语义写，本文件为准。

== 反汇编权威（RVA 均相对 0x10600000，ImageBase=运行时基址）==
解码核心  0x1420  (rcx=frame含mode, rdx=dst, r8=outlen, r9=pool ctx)
帧解包    0x1730  mode&2→头3/9字节; mode&1→压缩否则memcpy; 每帧后 ctx+0x9000清零
压缩包装  0x1620  outlen<0xd8→3字节头; 池 pos 字段每次调用全清零 ⇒ 官方帧自包含
压缩核心  0x10e0  单探针+key低24位比较; 距离≤2仅允许RLE(≥6重复); len<0x12→2字节token
全局池    .data+0x32F0 (0x106552F0) 大小0x9008: [0,0x8000)=4096 qword 指针槽
                         [0x8000,0x9000)=valid位图(注册置1,match取指针从不查位图)
帧头:  3字节=[mode][帧总长u8][outlen u8]   9字节=[mode][帧总长u32][outlen u32]
mode:  |0x44  bit1=9字节头  bit0=LZ压缩   (0x44=raw3, 0x45=LZ3, 0x46=raw9, 0x47=LZ9)

== 解码语义（与 lzdec.py 的差异标注 [D*]）==
- flags u32, ==1 即重填(LSB先); 首词进主循环前必重填
- bit0=1 match: peek u32; idx=(v>>4)&0xfff; nib=v&0xf
      nib!=0→len=nib+2,src+=2; nib==0→len=byte[src+2],src+=3
      源=pool[idx]指针; 前向按4字节块复制, 块数=ceil(len/3), 可 overrun ≤3 字节
      [D1] lzdec 用预写切片 ⇒ 距离<len 的 RLE 复制全错, 且无 overrun 字节污染
- bit0=0 literal: n=flags&0xf; k=T[n](T=[4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0])
      **先写 peek 的 u32 共4字节到 dst[write] (peek 溢出字节留在缓冲区!)**
      再 write+=k, src+=k, flags>>=k
      [D2] 溢出字节参与后续边界注册的 hash ⇒ 官方流可引用这些槽
- 注册(literal后): pos=window+1..write_new-3 **含端点**, hash=u32@pos(活缓冲区,含溢出)
      [D3] lzdec 只注册 pos+4<=write 的 ⇒ 漏 write-3 边界槽 ⇒ 官方流 POOLMISS
- 注册(match后): pos=window+1..write_old **含端点**; 然后 window=write_new-1
      (match 内部位置永不注册——编码器镜像同规则)
- 尾段: write>=outlen-0xb 起, 每字节1bit(bit值无关); 重填=**跳过u32不读**, flags=0x80000000
      (官方 filler 恒 0x80000000, 故 lzdec 读入等价, 字节面对齐一致)
- 尾后终注册: pos=window+1..outlen-4 含端点
- 池生命周期: 客户端进程级单例不清零, 但官方编码器每帧清池 ⇒ 帧自包含, 每帧新池即精确
"""
import struct

T = [4, 0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0]


class DecodeError(Exception):
    pass


class DstBuffer:
    """dst 缓冲区含 slack，保留 peek 溢出与 match overrun 字节。"""

    def __init__(self, outlen, slack=64):
        self.outlen = outlen
        self.b = bytearray(outlen + slack)
        self.write = 0

    def u32(self, pos):
        if pos + 4 > len(self.b):
            raise DecodeError(f'u32 read past buffer at {pos:#x}')
        return struct.unpack_from('<I', self.b, pos)[0]


def decode(frame: bytes, outlen: int, trace=None):
    """frame: 含 mode 字节的完整帧。返回 bytes(outlen)。每帧独立池（官方方言）。"""
    mode = frame[0]
    src = 9 if (mode & 2) else 3
    dst = DstBuffer(outlen)
    b = dst.b
    pool = {}          # idx -> dst 内相对 pos（帧内自包含）
    window = -1        # 最近已注册 pos
    flags = 1
    tail_at = outlen - 0xB
    steps = 0
    while True:
        steps += 1
        if steps > 2_000_000:
            raise DecodeError('step limit')
        if flags == 1:
            if src + 4 > len(frame):
                raise DecodeError('flags refill past end')
            flags = struct.unpack_from('<I', frame, src)[0]
            src += 4
        if flags & 1:
            # match（尾段区无闸——真码只在字面量分支查尾段）
            write = dst.write
            if src + 4 > len(frame):
                if src + 2 > len(frame):
                    raise DecodeError('match peek past end')
                v = struct.unpack_from('<H', frame, src)[0]
            else:
                v = struct.unpack_from('<I', frame, src)[0]
            idx = (v >> 4) & 0xFFF
            nib = v & 0xF
            flags >>= 1
            if nib:
                ln = nib + 2
                src += 2
            else:
                if src + 3 > len(frame):
                    raise DecodeError('extlen past end')
                ln = frame[src + 2]
                src += 3
            if idx not in pool:
                raise DecodeError(
                    f'pool miss idx={idx:#x} len={ln} at out[{write:#x}] bloboff={src:#x}')
            srcpos = pool[idx]
            # 前向块复制（0x14c8-0x14e2）：步长 3、宽 4 的重叠写，块数 ceil(ln/3)
            # 读活缓冲区 srcpos+3k ⇒ 距离<len 时自引用=RLE 复制；末块溢出 ≤3 字节
            nchunk = (ln + 2) // 3
            for c in range(nchunk):
                v = dst.u32(srcpos + 3 * c)
                struct.pack_into('<I', b, write + 3 * c, v)
            write += ln
            dst.write = write
            if trace is not None:
                trace.append(('M', idx, ln, srcpos))
            # 注册到 write_old（=本 token 前的 write），再跳过 match 内部
            reg_to = write - ln
            while window < reg_to:
                pos = window + 1
                v = dst.u32(pos)
                pool[((v >> 12) ^ v) & 0xFFF] = pos
                window = pos
            window = write - 1
        else:
            # 尾段闸门只在字面量分支（0x152f）——尾段区 match 照常解码
            if dst.write >= tail_at:
                write = dst.write
                while write < outlen:
                    if flags == 1:
                        src += 4          # 重填=跳过 filler u32，不读值
                        flags = 0x80000000
                    if src >= len(frame):
                        raise DecodeError('tail past end')
                    b[write] = frame[src]
                    src += 1
                    write += 1
                    flags >>= 1
                dst.write = write
                reg_to = outlen - 4
                while window < reg_to:
                    pos = window + 1
                    v = dst.u32(pos)
                    pool[((v >> 12) ^ v) & 0xFFF] = pos
                    window = pos
                return bytes(b[:outlen])
            # literal：先写 peek u32（4字节，含溢出），再前进 k
            write = dst.write
            n = flags & 0xF
            k = T[n]
            if k == 0:
                raise DecodeError(f'odd literal nibble {n:#x} at out[{write:#x}] bloboff={src:#x}')
            if src + 4 > len(frame):
                raise DecodeError('literal peek past end')
            struct.pack_into('<I', b, write, struct.unpack_from('<I', frame, src)[0])
            dst.write = write + k
            src += k
            flags >>= k
            if trace is not None:
                trace.append(('L', k, bytes(frame[src - k:src])))
            write = dst.write
            reg_to = write - 3
            while window < reg_to:
                pos = window + 1
                v = dst.u32(pos)
                pool[((v >> 12) ^ v) & 0xFFF] = pos
                window = pos


class Session:
    """S2C 会话态：持久池 + 持久会话缓冲（客户端 dst=[rbx+0x88]+1 每帧复用，
    前缀被逐帧覆写、溢出字节残留）——op 0x2D 官方帧首 token 即 match 的铁证模型。"""

    def __init__(self):
        self.buf = bytearray()   # 会话缓冲虚拟镜像
        self.pool = {}           # idx -> 缓冲偏移（跨帧持久，槽可指向旧帧残留区）

    def decode(self, frame: bytes, outlen: int, trace=None) -> bytes:
        if len(self.buf) < outlen + 64:
            self.buf.extend(b'\x00' * (outlen + 64 - len(self.buf)))
        mode = frame[0]
        src = 9 if (mode & 2) else 3
        b = self.buf
        pool = self.pool
        write = 0
        window = -1
        flags = 1
        tail_at = outlen - 0xB
        steps = 0

        def reg(upto):
            nonlocal window
            while window < upto:
                pos = window + 1
                if pos + 4 <= len(b):
                    v = struct.unpack_from('<I', b, pos)[0]
                    pool[((v >> 12) ^ v) & 0xFFF] = pos
                window = pos

        while True:
            steps += 1
            if steps > 2_000_000:
                raise DecodeError('step limit')
            if flags == 1:
                if src + 4 > len(frame):
                    raise DecodeError('flags refill past end')
                flags = struct.unpack_from('<I', frame, src)[0]
                src += 4
            if flags & 1:
                if src + 4 > len(frame):
                    if src + 2 > len(frame):
                        raise DecodeError('match peek past end')
                    v = struct.unpack_from('<H', frame, src)[0]
                else:
                    v = struct.unpack_from('<I', frame, src)[0]
                idx = (v >> 4) & 0xFFF
                nib = v & 0xF
                flags >>= 1
                if nib:
                    ln = nib + 2
                    src += 2
                else:
                    if src + 3 > len(frame):
                        raise DecodeError('extlen past end')
                    ln = frame[src + 2]
                    src += 3
                if idx not in pool:
                    raise DecodeError(
                        f'pool miss idx={idx:#x} len={ln} at out[{write:#x}] bloboff={src:#x}')
                srcpos = pool[idx]
                nchunk = (ln + 2) // 3
                for c in range(nchunk):
                    v = struct.unpack_from('<I', b, srcpos + 3 * c)[0]
                    struct.pack_into('<I', b, write + 3 * c, v)
                if trace is not None:
                    trace.append(('M', idx, ln, srcpos, srcpos >= write))
                reg(write)
                write += ln
                window = write - 1
            else:
                if write >= tail_at:
                    while write < outlen:
                        if flags == 1:
                            src += 4
                            flags = 0x80000000
                        if src >= len(frame):
                            raise DecodeError('tail past end')
                        b[write] = frame[src]
                        src += 1
                        write += 1
                        flags >>= 1
                    reg(outlen - 4)
                    return bytes(b[:outlen])
                n = flags & 0xF
                k = T[n]
                if k == 0:
                    raise DecodeError(f'odd literal nibble {n:#x} at out[{write:#x}] bloboff={src:#x}')
                if src + 4 > len(frame):
                    raise DecodeError('literal peek past end')
                struct.pack_into('<I', b, write, struct.unpack_from('<I', frame, src)[0])
                src += k
                flags >>= k
                if trace is not None:
                    trace.append(('L', k, write))
                write += k
                reg(write - 3)


def unframe(p: bytes):
    """p: 含 op 头的线帧（p[0]=op tag, p[1]=mode）。返回 (payload, outlen)。"""
    mode = p[1]
    r9 = 4 if (mode & 2) else 1
    outlen = struct.unpack_from('<I', p, 2 + r9)[0] & (0xFFFFFFFF >> ((4 - r9) * 8))
    frame = bytes(p[1:])
    if mode & 1:
        return decode(frame, outlen), outlen
    off = 1 + 2 * r9
    return bytes(p[off:off + outlen]), outlen


# ---------------------------------------------------------------- 压缩端 ----

def wrap_compress(body: bytes) -> bytes:
    """完整帧：[mode][帧总长][outlen]+体。头长按 outlen<0xd8 选择（0x1620）。"""
    core = compress_core(body)
    hdr9 = len(body) >= 0xD8
    total = (9 if hdr9 else 3) + len(core)
    mode = 0x44 | 1 | (2 if hdr9 else 0)
    if hdr9:
        head = struct.pack('<BII', mode, total, len(body))
    else:
        head = bytes([mode, total & 0xFF, len(body) & 0xFF])
    return head + core


class _Enc:
    """编码器状态机（供 compress_core 驱动）。"""

    def __init__(self):
        self.reg = 0x80000000
        self.slots = []       # 输出事件流：('w', u32) / ('d', bytes)
        self.cur_data = bytearray()

    def bit(self, val):
        if self.reg & 1:
            # 词满（哨兵落底）——先归一化存词再开新词（0x1180）
            w = (self.reg >> 1) | 0x80000000
            self.slots.append(('w', struct.pack('<I', w)))
            self.slots.append(('d', bytes(self.cur_data)))
            self.cur_data = bytearray()
            self.reg = 0x80000000
        self.reg >>= 1
        if val:
            self.reg |= 0x80000000

    def data(self, chunk: bytes):
        self.cur_data.extend(chunk)

    def finish(self):
        while not (self.reg & 1):
            self.reg >>= 1
        w = (self.reg >> 1) | 0x80000000
        self.slots.append(('w', struct.pack('<I', w)))
        if self.cur_data:
            self.slots.append(('d', bytes(self.cur_data)))
        buf = bytearray()
        for _, chunk in self.slots:
            buf.extend(chunk)
        return bytes(buf)


def compress_core(body: bytes) -> bytes:
    n = len(body)
    if n == 0:
        return b''
    end = n - 1
    tail_at = end - 0xA
    enc = _Enc()
    key = [0] * 0x1000
    pos = [0] * 0x1000
    pad = body + b'\0' * 8

    def peek(i):
        return struct.unpack_from('<I', pad, i)[0]

    cur = 0
    litrun = 0   # edx：连续字面字节数（match 后清零，0x125c/0x12f1）
    while cur <= tail_at:
        v = peek(cur)
        idx = ((v >> 12) ^ v) & 0xFFF
        old_key, old_pos = key[idx], pos[idx]
        key[idx], pos[idx] = v, cur + 1
        cand = old_pos - 1
        mlen = 0
        dist = 0
        if (old_key ^ v) & 0xFFFFFF == 0 and cand >= 0:
            dist = cur - cand
            if dist > 2:
                mlen = 3
            elif dist == 1 and litrun >= 3 and cur > 3:
                # 0x1204-0x1230 RLE：body[cur-3..cur+3] 共 7 字节全等
                base = body[cur - 3]
                if all(body[cur - 3 + k] == base for k in range(7)):
                    mlen = 3
        if mlen == 3 and (old_key ^ v) == 0:
            # 全 32 位匹配：扩展（0x1263+）；cap = end-(cur+4)+2（0x1292 在 r10=cur+4 处取）
            cap = min(end - cur - 2, 0xFF)
            mlen = 4
            while mlen < cap and cur + mlen < n and body[cur + mlen] == body[cur + mlen - dist]:
                mlen += 1
        if mlen >= 3:
            enc.bit(1)
            if mlen < 0x12:
                enc.data(struct.pack('<H', (idx << 4 | (mlen - 2)) & 0xFFFF))
            else:
                # 3 字节 token（第 4 字节是会被告后续写入覆写的溢出，不可入流）
                enc.data(struct.pack('<I', ((mlen & 0xFF) << 16) | (idx << 4))[:3])
            cur += mlen
            litrun = 0
        else:
            enc.bit(0)
            enc.data(body[cur:cur + 1])
            cur += 1
            litrun += 1
    while cur <= end:
        if cur <= end - 3:
            v = peek(cur)
            idx = ((v >> 12) ^ v) & 0xFFF
            key[idx], pos[idx] = v, cur + 1
        enc.bit(0)
        enc.data(body[cur:cur + 1])
        cur += 1
    return enc.finish()


def compress(body: bytes) -> bytes:   # noqa: F811 —— 保留命名入口
    return compress_core(body)


if __name__ == '__main__':
    # 自检：官方明文 round-trip
    import os
    rng = sorted(__import__('random').Random(1).randrange(256) for _ in range(0))
    samples = {
        'ascii': (b'hello world hello world hello world!! ' * 30),
        'rle': b'A' * 300 + b'BCDEF' * 40 + b'Z' * 50,
        'mixed': os.urandom(64) + b'PATTERN' * 20 + os.urandom(32) + b'PATTERN' * 5,
    }
    for name, body in samples.items():
        core = compress_core(body)
        frame = wrap_compress(body)
        got = decode(frame, len(body))
        print(f'{name}: body={len(body)} core={len(core)} frame={len(frame)} '
              f'roundtrip={"OK" if got == body else "FAIL"}')
        if got != body:
            for i, (a, b2) in enumerate(zip(body, got)):
                if a != b2:
                    print(f'  first diff @{i:#x}: {a:#x} vs {b2:#x}')
                    break
