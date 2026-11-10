# 全量会话 LZ 解码：按线序处理 l_frames.pkl 所有 0x27-0x30 族帧，共享池建全会话字典状态，
# 统计通过率并导出 0x28 帧解压明文（f6 类=outlen 0x122）→ 供 v3 生成器语法提取。
# 用法: python -I lz_session_decode.py [out_dir]
import pickle, struct, sys, os, json
import importlib.util
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lzdec import World, decompress, DecodeError

FAMILY = set(range(0x27, 0x31))

def frame_decode(p, world):
    """op-level compressed value: tag@[0], mode@[1]"""
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    if 2 + hdrlen + 4 > len(p):
        raise DecodeError('frame too short')
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    if outlen > 0xFFFF:
        raise DecodeError(f'outlen {outlen:#x} > 0xffff')
    return decompress(p[1:], outlen, world), outlen

def main():
    src = r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis/l_frames.pkl'
    out_dir = sys.argv[1] if len(sys.argv) > 1 else r'G:/pojie/ai/9yin-server-go/tmp/op28_analysis'
    with open(src, 'rb') as f:
        frames = pickle.load(f)
    world = World()
    stats = {'ok': 0, 'fail': 0, 'skipped': 0}
    fails = []
    decoded_28 = []   # (idx, outlen, plaintext)
    for idx, p in enumerate(frames):
        if not isinstance(p, (bytes, bytearray)) or len(p) < 2:
            stats['skipped'] += 1
            continue
        op = p[0]
        if op not in FAMILY:
            continue
        try:
            out, outlen = frame_decode(bytes(p), world)
            stats['ok'] += 1
            if op == 0x28:
                decoded_28.append((idx, outlen, bytes(out)))
        except DecodeError as e:
            stats['fail'] += 1
            fails.append((idx, op, str(e)))
    print(f"family frames: ok={stats['ok']} fail={stats['fail']}")
    for idx, op, msg in fails[:12]:
        print(f'  fail #{idx} op={op:#x}: {msg}')
    # f6 class: outlen == 0x122
    f6 = [(i, o) for i, ol, o in decoded_28 if ol == 0x122]
    print(f'0x28 decoded ok: {len(decoded_28)}, f6-class(outlen 0x122): {len(f6)}')
    # group all 0x28 by outlen
    from collections import Counter
    c = Counter(ol for _, ol, _ in decoded_28)
    print('0x28 outlen histogram:', dict(sorted(c.items())))
    # dump f6 plaintexts + one sample of top classes
    with open(os.path.join(out_dir, 'f6_plaintexts.pkl'), 'wb') as f:
        pickle.dump(f6, f)
    # also dump every decoded 0x28 for grammar work
    with open(os.path.join(out_dir, 'decoded_28.pkl'), 'wb') as f:
        pickle.dump(decoded_28, f)
    print('saved f6_plaintexts.pkl / decoded_28.pkl')
    # print first f6 plaintext annotated
    if f6:
        i, out = f6[0]
        print(f'\n=== f6 frame#{i} plaintext ({len(out)}B) ===')
        for off in range(0, min(len(out), 0x140), 16):
            chunk = out[off:off+16]
            hexs = ' '.join(f'{b:02x}' for b in chunk)
            asc = ''.join(chr(b) if 32 <= b < 127 else '.' for b in chunk)
            print(f'{off:04x}: {hexs:<48} {asc}')

if __name__ == '__main__':
    main()
