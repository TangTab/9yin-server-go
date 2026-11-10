# decode crash-night s2c capture: EE-unescape -> XOR InitialKey -> frames
import struct, sys

KEY = bytes([0xCC, 0x3B, 0x6D, 0xFA])  # 0xFA6D3BCC LE

def unescape(d):
    frames = []
    body = bytearray()
    i = 0
    n = len(d)
    while i < n:
        b = d[i]
        if b != 0xEE:
            body.append(b)
            i += 1
        else:
            if i+1 >= n:
                break
            nxt = d[i+1]
            if nxt == 0x00:
                body.append(0xEE)
                i += 2
            elif nxt == 0xEE:
                frames.append(bytes(body))
                body = bytearray()
                i += 2
            else:
                # invalid escape -> resync: treat as literal
                body.append(b)
                i += 1
    return frames

def xor_key(body):
    return bytes(b ^ KEY[i & 3] for i, b in enumerate(body))

path = sys.argv[1] if len(sys.argv) > 1 else r'F:/Game/Online/9yin/9yin-server-go/captures/local/127-0-0-1-60809_s2c.bin'
d = open(path, 'rb').read()
raw = unescape(d)
print(f'frames: {len(raw)}')
ops = {}
dec = []
for body in raw:
    p = xor_key(body)
    dec.append(p)
    op = p[0] if p else -1
    ops[op] = ops.get(op, 0) + 1
print('op histogram:', {hex(k): v for k, v in sorted(ops.items(), key=lambda kv: -kv[1])[:16]})
x28 = [(i, p) for i, p in enumerate(dec) if p and p[0] == 0x28]
print(f'0x28 frames: {len(x28)} at idx {[i for i,_ in x28]}')
for i, p in x28[-1:]:
    print(f'--- last 0x28 frame#{i} len={len(p)} ---')
    for off in range(0, min(len(p), 0x200), 16):
        chunk = p[off:off+16]
        hexs = ' '.join(f'{b:02x}' for b in chunk)
        asc = ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)
        print(f'{off:04x}: {hexs:<48} {asc}')
    if len(p) > 0x200:
        print(f'... ({len(p)-0x200} more bytes)')
