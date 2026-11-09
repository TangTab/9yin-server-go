# -*- coding: utf-8 -*-
"""九阴官服 pcapng 解析库：pcapng → TCP 流(带时间戳) → XOR+EE 帧解码。

帧格式（capture_deep.py 同源）：
  线上字节：明文帧逐字节 XOR(KEY=CC 3B 6D FA, i%4) 后 0xEE 转义（EE→EE 00，帧尾 EE EE）。
  解码：按 EE EE 切帧 → EE 00 还原 EE → 帧内偏移 i%4 异或 KEY。
"""
import struct
import sys

KEY = bytes([0xCC, 0x3B, 0x6D, 0xFA])


# ---------- pcapng 原生读取（避免依赖 scapy 的性能与内存开销） ----------

def read_pcapng(path):
    """极简 pcapng 读取器。yield (ts_us, linktype, packet_bytes)。仅支持 LE。"""
    with open(path, 'rb') as f:
        data = f.read()
    off = 0
    n = len(data)
    linktype = None
    ts_div = 1
    if data[:4] == b'\x0a\x0d\x0d\x0a':
        bom = struct.unpack_from('<I', data, 8)[0]
        if bom != 0x1A2B3C4D:
            raise ValueError('big-endian pcapng not supported')
        ts_div = 1000  # default if_tsresol=us
    while off + 12 <= n:
        btype, blen = struct.unpack_from('<II', data, off)
        if blen < 12 or off + blen > n:
            break
        body = data[off:off + blen]
        if btype == 0x0A0D0D0A:  # SHB
            pass
        elif btype == 0x00000001:  # IDB
            lt = body[8:12]
            linktype = lt[0] | (lt[1] << 8)
            # option 扫描 if_tsresol
            i = 16
            while i + 4 <= len(body):
                opt_type, opt_len = struct.unpack_from('<HH', body, i)
                if opt_type == 0:
                    break
                val = body[i + 4:i + 4 + opt_len]
                if opt_type == 9 and val:  # if_tsresol
                    r = val[0]
                    if r < 64:
                        ts_div = 10 ** r
                    else:
                        ts_div = 2 ** (r - 64)
                i += 4 + ((opt_len + 3) & ~3)
        elif btype & 0x80000000 == 0 and btype in (2, 6, 0x40000000):  # EPB/SPB/older
            if btype == 6:  # EPB: btype blen iid ts_hi ts_lo caplen pktlen data
                ts_hi, ts_lo, caplen, pktlen = struct.unpack_from('<IIII', body, 12)
                pktdat = body[28:28 + caplen]
            elif btype == 2:  # SPB: btype blen origlen data（无时间戳）
                orig_len = struct.unpack_from('<I', body, 4)[0]
                pktdat = body[8:8 + min(orig_len, blen - 16)]
                ts_hi = ts_lo = 0
            else:
                pktdat = None
            if pktdat is not None:
                ts_us = ((ts_hi << 32) | ts_lo) * 1e6 / ts_div
                yield ts_us, linktype, pktdat
        off += blen


def parse_eth_ip_tcp(pkt, linktype):
    """返回 (src, sport, dst, dport, seq, payload) 或 None。仅 IPv4。"""
    if linktype == 1:  # Ethernet
        if len(pkt) < 14:
            return None
        if pkt[12] != 0x08 or pkt[13] != 0x00:
            return None
        ip = pkt[14:]
    elif linktype == 101:  # Raw IP
        ip = pkt
    elif linktype == 0:  # Null/BSD loopback
        if len(pkt) < 4:
            return None
        ip = pkt[4:]
    else:
        return None
    if len(ip) < 20:
        return None
    if (ip[0] >> 4) != 4:
        return None
    ihl = (ip[0] & 0xF) * 4
    total = struct.unpack_from('>H', ip, 2)[0]
    if ip[9] != 6:  # TCP only
        return None
    src = '.'.join(str(b) for b in ip[12:16])
    dst = '.'.join(str(b) for b in ip[16:20])
    tcp = ip[ihl:max(ihl + 20, min(len(ip), total))]
    if len(tcp) < 20:
        return None
    sport, dport, seq = struct.unpack_from('>HHI', tcp, 0)
    doff = (tcp[12] >> 4) * 4
    payload = tcp[doff:]
    if ip[6] & 0x3F:  # frag
        return None
    return src, sport, dst, dport, seq, payload


class Stream:
    __slots__ = ('key', 'sport', 'dport', 'server_ip', 'server_port', 'c2s', 's2c')

    def __init__(self, key):
        self.key = key  # (client_ip, cport, server_ip, sport)
        self.c2s = []  # (ts_us, bytes) client→server
        self.s2c = []  # (ts_us, bytes)

    @property
    def client(self):
        return '%s:%d' % (self.key[0], self.key[1])

    @property
    def server(self):
        return '%s:%d' % (self.key[2], self.key[3])


def build_streams(path):
    """pcapng → 按 TCP 会话重组的裸字节流（不处理重传/乱序，抓包通常干净）。"""
    streams = {}
    for ts, lt, pkt in read_pcapng(path):
        r = parse_eth_ip_tcp(pkt, lt)
        if r is None:
            continue
        src, sport, dst, dport, seq, payload = r
        if not payload:
            continue
        # 客户端=小端口一侧启发（>1024 视为客户端口）；若两侧都大，按先发 SYN 者为客户端
        if sport > 1024 and (dport <= 1024 or sport > dport):
            ckey = (src, sport, dst, dport)
        elif dport > 1024 and (sport <= 1024 or dport > sport):
            ckey = (dst, dport, src, sport)
            payload_dir = 's2c'
        else:
            ckey = (src, sport, dst, dport)
        st = streams.setdefault(ckey, Stream(ckey))
        if (src, sport) == (ckey[0], ckey[1]):
            st.c2s.append((ts, payload))
        else:
            st.s2c.append((ts, payload))
    return list(streams.values())


# ---------- 帧解码 ----------

def decode_frames(buf):
    """EE EE 切帧 → 去转义 → 帧内偏移对齐 KEY 异或。返回明文帧列表。"""
    frames, body = [], bytearray()
    i = 0
    ln = len(buf)
    while i < ln:
        b = buf[i]
        if b == 0xEE and i + 1 < ln:
            nxt = buf[i + 1]
            if nxt == 0x00:
                body.append(0xEE)
                i += 2
                continue
            elif nxt == 0xEE:
                frames.append(bytes(x ^ KEY[j % 4] for j, x in enumerate(body)))
                body = bytearray()
                i += 2
                continue
            else:
                # 非法转义：保守当作帧尾切掉（记录为 None 占位）
                frames.append(bytes(x ^ KEY[j % 4] for j, x in enumerate(body)))
                body = bytearray()
                i += 2
                continue
        else:
            body.append(b)
            i += 1
    if body:
        frames.append(bytes(x ^ KEY[j % 4] for j, x in enumerate(body)))
    return frames


def stream_frames(chunks):
    """带时间戳版：把 (ts, bytes) 流切帧。返回 [(ts_us, frame_bytes)]。

    时间戳取「覆盖该帧尾 EE EE 的那个 TCP 包」的时间。
    """
    out = []
    body = bytearray()
    start_ts = chunks[0][0] if chunks else 0
    for ts, payload in chunks:
        i = 0
        ln = len(payload)
        while i < ln:
            b = payload[i]
            if b == 0xEE and i + 1 < ln:
                nxt = payload[i + 1]
                if nxt == 0x00:
                    body.append(0xEE)
                    i += 2
                    continue
                else:  # EE EE 或非法 → 帧尾
                    out.append((ts, bytes(x ^ KEY[j % 4] for j, x in enumerate(body))))
                    body = bytearray()
                    i += 2
                    continue
            elif b == 0xEE and i + 1 == ln:
                body.append(b)  # 留给下一包判断
                i += 1
                continue
            else:
                body.append(b)
                i += 1
                continue
    if body:
        out.append((chunks[-1][0], bytes(x ^ KEY[j % 4] for j, x in enumerate(body))))
    return out


# ---------- TVarList ----------

def parse_tvarlist(payload):
    if len(payload) < 2:
        return None
    count = struct.unpack_from('<H', payload)[0]
    if count == 0 or count > 1024:
        return None
    rest = payload[2:]
    vals = []
    for _ in range(count):
        if not rest:
            return None
        tag = rest[0]
        rest = rest[1:]
        try:
            if tag == 2:
                vals.append(struct.unpack('<i', rest[:4])[0]); rest = rest[4:]
            elif tag == 3:
                vals.append(struct.unpack('<q', rest[:8])[0]); rest = rest[8:]
            elif tag == 4:
                vals.append(round(struct.unpack('<f', rest[:4])[0], 3)); rest = rest[4:]
            elif tag == 5:
                vals.append(round(struct.unpack('<d', rest[:8])[0], 3)); rest = rest[8:]
            elif tag in (6, 7):
                n = struct.unpack_from('<I', rest)[0]
                rest = rest[4:]
                raw, rest = rest[:n], rest[n:]
                try:
                    s = raw.decode('utf-16-le') if tag == 7 else raw.rstrip(b'\x00').decode('gbk', 'replace')
                except UnicodeDecodeError:
                    s = raw.hex()
                vals.append(s)
            elif tag == 8:
                if len(rest) < 8:
                    return None
                oid, owner = struct.unpack('<II', rest[:8])
                vals.append(oid | (owner << 32)); rest = rest[8:]
            else:
                return None
        except struct.error:
            return None
    return vals if not rest else None
