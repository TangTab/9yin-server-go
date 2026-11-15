# -*- coding: utf-8 -*-
"""c.pcapng AOI 物化簇逐字节解码（第十四轮）：外服创建怪的帧到底长什么样。

线序喂 World2 池（非 0x27-0x30 族帧明文按 4B 滑窗注册），解出簇内
0x28 明文 + 0x0D 原文，与我方 NINEYIN_NPC_ADD_AS_0X28 通道生成的帧
（captures/local 最新会话）对比。

Usage: python -I c_aoi_cluster_decode.py
"""
import struct
import sys
import os
import importlib.util

sys.stdout.reconfigure(encoding='utf-8')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\pcap_analysis')
sys.path.insert(0, r'G:\pojie\ai\9yin-server-go-2433\tools\op28')
import pcaplib  # noqa: E402
import lzdec  # noqa: E402

STITCH = r'G:\pojie\ai\9yin-server-go\tools\op46_profile_stitch.py'
spec = importlib.util.spec_from_file_location('stitch', STITCH)
stitch = importlib.util.module_from_spec(spec)
spec.loader.exec_module(stitch)
World2 = stitch.World2

FAMILY = set(range(0x27, 0x31))
CAP = r'F:\Game\Online\9yin\9yin-server-go\artifacts\c.pcapng'


def frame_decode(p, world):
    mode = p[1]
    hdrlen = 4 if (mode & 2) else 1
    edx = struct.unpack_from('<I', p, 2 + hdrlen)[0]
    outlen = edx & (0xFFFFFFFF >> ((4 - hdrlen) * 8))
    return lzdec.decompress(p[1:], outlen, world), outlen


def main():
    st = max(pcaplib.build_streams(CAP), key=lambda s: len(pcaplib.stream_frames(s.s2c)))
    frames = [f for _, f in pcaplib.stream_frames(st.s2c) if f]
    world = World2()
    ok = fail = 0
    ents = []  # (idx, op, wirelen, outlen, plaintext/None)
    for idx, fr in enumerate(frames):
        if fr[0] in FAMILY:
            if fr[0] == 0x28:
                try:
                    out, outlen = frame_decode(bytes(fr), world)
                    ents.append((idx, 0x28, len(fr), outlen, out))
                    ok += 1
                except lzdec.DecodeError:
                    ents.append((idx, 0x28, len(fr), None, None))
                    fail += 1
                    world.alloc(0)  # 保持池状态推进尽力
            else:
                try:
                    frame_decode(bytes(fr), world)
                except lzdec.DecodeError:
                    pass
        elif fr[0] == 0x0D:
            ents.append((idx, 0x0D, len(fr), None, bytes(fr)))
            world.register_raw(bytes(fr))
        else:
            world.register_raw(bytes(fr))
    print(f'0x28 decode ok={ok} fail={fail}')
    # 只看 AOI 簇（>=首个 0x0D 出现的簇；c 里 0x0D×5 全在簇内）
    for idx, op, wl, outlen, pl in ents:
        if op == 0x0D:
            print(f'#{idx} 0x0D wire={wl} :: {pl[:48].hex()}')
    for idx, op, wl, outlen, pl in ents:
        if op == 0x28 and outlen and outlen < 400:  # 簇内 NPC detail 帧（非 5KB 玩家档案）
            if idx < 200:  # 首爆发的 21 帧里也有很多；全打
                pass
            obj, own = struct.unpack_from('<II', pl, 0)
            f = struct.unpack_from('<13f', pl, 8)
            cnt = struct.unpack_from('<H', pl, 60)[0]
            print(f'#{idx} 0x28 wire={wl} out={outlen} obj={obj:08x} own={own:08x} cnt={cnt} '
                  f'pos=({f[0]:.1f},{f[1]:.1f},{f[2]:.1f}) yaw={f[3]:.1f} scale={f[6]:.2f}')
    # 保存簇区明文供走查
    out_dir = r'G:\pojie\ai\9yin-server-go\tmp\op28_analysis'
    import pickle
    pickle.dump(ents, open(os.path.join(out_dir, 'c_entities_decoded.pkl'), 'wb'))
    print('saved c_entities_decoded.pkl')


if __name__ == '__main__':
    main()
