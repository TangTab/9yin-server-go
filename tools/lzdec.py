# fxnet2.dll LZ decoder simulator (RVA 0x1000-0x1260, per disassembly 2026-11-11)
# bit stream after 3/9-byte header:
#   flags u32 refilled inline when ==1 sentinel (LSB-first consumption)
#   token bit 0 = literal run: unary count peek via table t[n], n=r11&0xf (n even),
#                t=[4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0]; copy t[n] bytes; consume t[n] bits
#   token bit 1 = match: peek u32; idx=(v>>4)&0xfff; nib=v&0xf
#                nib!=0 -> len=nib+2, consume 2 bytes; nib==0 -> len=byte[+2], consume 3
#   pool: 4096 slots, global across calls; registration idx=((u32@pos>>12)^u32@pos)&0xfff
#   window lags: register positions up to write-3 after each token
# tail: when write >= outlen-0xb, byte-by-byte copy until write==outlen-1 (rsi)
import struct

T = [4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0]

class World:
    """virtual heap + global pool across calls"""
    def __init__(self):
        self.heap = bytearray()      # all decompressed outputs concatenated
        self.pool = {}               # idx -> virtual position
    def alloc(self, outlen):
        base = len(self.heap)
        self.heap.extend(b'\x00' * max(outlen, 0))
        return base

class DecodeError(Exception):
    pass

def decompress(blob, outlen, world, trace=None):
    """blob: bytes starting at mode byte. returns bytes(outlen). raises on pool miss."""
    mode = blob[0]
    hdr = 9 if (mode & 2) else 3
    src = hdr                      # cursor into blob
    base = world.alloc(outlen)     # virtual dst base
    out = bytearray()
    heap = world.heap
    flags = 1                      # sentinel -> refill
    pool = world.pool
    window = -1                    # r8 relative to base (base-1 == -1)
    tail_at = outlen - 0xb         # rbp threshold (write >= -> tail mode)
    steps = 0
    while True:
        steps += 1
        if steps > 100000:
            raise DecodeError('step limit')
        write = len(out)
        # tail mode
        if write >= tail_at:
            # 0x1160: refill handling then byte copy till write==outlen-1
            while write < outlen:
                if flags == 1:
                    if src + 4 > len(blob):
                        raise DecodeError('flags refill past end at tail')
                    flags = struct.unpack_from('<I', blob, src)[0]
                    src += 4
                b = blob[src] if src < len(blob) else 0
                src += 1
                out.append(b)
                write += 1
                flags >>= 1
            # final window registration up to outlen-3
            reg_to = outlen - 3
            while window < reg_to - 1:
                pos = window + 1
                if pos + 4 <= outlen:
                    v = struct.unpack_from('<I', out, pos)[0]
                    idx = ((v >> 12) ^ v) & 0xfff
                    pool[idx] = base + pos
                window = pos
            if len(out) != outlen:
                raise DecodeError('tail overrun')
            return bytes(out)
        # main loop head 0x1060
        if flags == 1:
            if src + 4 > len(blob):
                raise DecodeError('flags refill past end')
            flags = struct.unpack_from('<I', blob, src)[0]
            src += 4
        if flags & 1:
            # match 0x107a
            if src + 4 > len(blob):
                # allow u16 peek at end
                if src + 2 > len(blob):
                    raise DecodeError('match peek past end')
                v = struct.unpack_from('<H', blob, src)[0]
            else:
                v = struct.unpack_from('<I', blob, src)[0]
            idx = (v >> 4) & 0xfff
            nib = v & 0xf
            flags >>= 1
            if nib:
                ln = nib + 2
                src += 2
            else:
                if src + 3 > len(blob):
                    raise DecodeError('extlen past end')
                ln = blob[src + 2]
                src += 3
            if idx not in pool:
                if trace is not None:
                    trace.append(('POOLMISS', idx, ln, write, src))
                raise DecodeError(f'pool miss idx={idx:#x} len={ln} at out[{write:#x}], bloboff={src:#x}')
            srcpos = pool[idx]
            chunk = bytes(heap[srcpos:srcpos+ln])
            if len(chunk) < ln:
                raise DecodeError('pool src past heap')
            out.extend(chunk)
            if trace is not None:
                trace.append(('M', idx, ln, chunk[:12]))
        else:
            # literal 0x1101
            n = flags & 0xf
            size = T[n]
            if size == 0:
                raise DecodeError(f'odd literal nibble {n:#x} at out[{write:#x}] bloboff={src:#x}')
            if src + size > len(blob):
                raise DecodeError('literal past end')
            lit = blob[src:src+size]
            out.extend(lit)
            src += size
            flags >>= size
            if trace is not None:
                trace.append(('L', size, lit))
        # window registration up to write-3 (both paths)
        newwrite = len(out)
        reg_to = newwrite - 3
        while window < reg_to:
            pos = window + 1
            if pos + 4 <= newwrite:
                v = struct.unpack_from('<I', out, pos)[0]
                idx = ((v >> 12) ^ v) & 0xfff
                pool[idx] = base + pos
            window = pos

def try_blob(blob, outlen):
    w = World()
    try:
        return decompress(blob, outlen, w), w
    except DecodeError as e:
        return None, e

if __name__ == '__main__':
    # smoke: all-literal tiny case
    # mode=0x01 (compressed, hdrlen=1... wait hdrlen from mode&2), hdr=3
    # handcraft: mode=01, 2 bytes pad, then flags=0b10 (bit1=0 literal n=2 ->1B), data 'A', then...
    b = bytes([0x01, 0, 0]) + struct.pack('<I', 0b10) + b'A' * 1
    out, err = try_blob(b, 1)
    print('smoke1:', out, err)
