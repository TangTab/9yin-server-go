# Pure-python minidump triage: exception record + module map + naive stack scan
# Usage: python -I parse_dumps.py <dump.dmp> [...]
import struct, sys, os

def u32(b, o): return struct.unpack_from('<I', b, o)[0]
def u64(b, o): return struct.unpack_from('<Q', b, o)[0]

def parse(path):
    d = open(path, 'rb').read()
    if d[:4] != b'MDMP':
        print(f'{path}: not a minidump'); return
    n_streams = u32(d, 8)
    dir_rva = u32(d, 12)
    streams = {}
    for i in range(n_streams):
        o = dir_rva + 12*i
        st, ds, rva = u32(d, o), u32(d, o+4), u32(d, o+8)
        streams.setdefault(st, []).append((ds, rva))

    # ---- module list (4) ----
    mods = []
    if 4 in streams:
        ds, rva = streams[4][0]
        n = u32(d, rva)
        for i in range(n):
            o = rva + 4 + 108*i
            base = u64(d, o)
            size = u32(d, o+8)
            name_rva = u32(d, o+20)
            slen = u32(d, name_rva)
            name = d[name_rva+4:name_rva+4+slen].decode('utf-16le', 'replace')
            mods.append((base, size, name))
    def locate(addr):
        for base, size, name in mods:
            if base <= addr < base+size:
                return f'{name}+{addr-base:#x}'
        return None

    print(f'===== {os.path.basename(path)} =====')
    print(f'modules: {len(mods)}')
    fx = [m for m in mods if 'fx' in m[2].lower() or 'snail' in m[2].lower() or 'beast' in m[2].lower()]
    for base, size, name in sorted(fx):
        print(f'  {base:#x} {size:#8x} {os.path.basename(name)}')

    # ---- exception stream (6) ----
    if 6 not in streams:
        print('no exception stream'); return
    ds, rva = streams[6][0]
    tid = u32(d, rva)
    exc_code = u32(d, rva+8)
    exc_addr = u64(d, rva+24)
    nparams = u32(d, rva+32)
    params = [u64(d, rva+40+8*i) for i in range(min(nparams, 4))]
    ctx_size = u32(d, rva+160)
    ctx_rva = u32(d, rva+164)
    rip = u64(d, ctx_rva+0xF8)
    rsp = u64(d, ctx_rva+0x98)
    rbp = u64(d, ctx_rva+0xA0)
    rax = u64(d, ctx_rva+0x78)
    rcx = u64(d, ctx_rva+0x80)
    rdx = u64(d, ctx_rva+0x88)
    rbx = u64(d, ctx_rva+0x90)
    rsi = u64(d, ctx_rva+0xA8)
    rdi = u64(d, ctx_rva+0xB0)

    codes = {0xC0000005: 'ACCESS_VIOLATION', 0xC0000409: 'STACK_BUFFER_OVERRUN/FASTFAIL',
             0x80000003: 'BREAKPOINT', 0xC00000FD: 'STACK_OVERFLOW', 0xC000001D: 'ILLEGAL_INSTRUCTION'}
    print(f'thread {tid} exception {exc_code:#x} ({codes.get(exc_code, "?")}) at {exc_addr:#x}')
    print(f'  -> faulting: {locate(exc_addr)}')
    if exc_code == 0xC0000005 and nparams >= 2:
        kind = {0: 'READ', 1: 'WRITE', 8: 'DEP-EXEC'}.get(params[0], params[0])
        print(f'  -> AV {kind} at {params[1]:#x}  (map: {locate(params[1])})')
    print(f'  RIP={rip:#x} ({locate(rip)})')
    print(f'  RAX={rax:#x} RCX={rcx:#x} RDX={rdx:#x} RBX={rbx:#x}')
    print(f'  RSP={rsp:#x} RBP={rbp:#x} RSI={rsi:#x} RDI={rdi:#x}')

    # ---- stack memory: gather from Memory64 (9) / Memory (5) ----
    ranges = []
    if 9 in streams:
        ds2, rva2 = streams[9][0]
        n = u64(d, rva2)
        base_rva = u64(d, rva2+8)
        off = base_rva
        for i in range(n):
            o = rva2 + 16 + 16*i
            start = u64(d, o)
            sz = u64(d, o+8)
            ranges.append((start, off, sz))
            off += sz
    if 5 in streams:
        ds2, rva2 = streams[5][0]
        n = u32(d, rva2)
        for i in range(n):
            o = rva2 + 4 + 16*i
            start = u64(d, o)
            sz = u32(d, o+8)
            mrva = u32(d, o+12)
            ranges.append((start, mrva, sz))

    # find stack range containing rsp
    stack = None
    for start, off, sz in ranges:
        if start <= rsp < start+sz:
            stack = (start, off, sz)
            break
    if not stack:
        print('  (no stack memory in dump)')
        return
    start, off, sz = stack
    lo = rsp - start
    blob = d[off+lo: off+sz]
    print(f'  stack scan {sz:#x} bytes from RSP, return-address candidates:')
    hits = 0
    for i in range(0, len(blob)-8, 8):
        v = u64(blob, i)
        loc = locate(v)
        if loc and hits < 40:
            # naive: inside fx modules or any module, list it
            print(f'    [rsp+{i:#05x}] {v:#x} {loc}')
            hits += 1
    print()

for p in sys.argv[1:]:
    try:
        parse(p)
    except Exception as ex:
        print(f'{p}: ERROR {ex}')
