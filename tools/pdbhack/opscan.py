# -*- coding: utf-8 -*-
"""批量提取 FsSoloMember Send* 函数的 opcode 写入点与调用结构。
扫描函数体内的 mov byte ptr [dst], imm8 与 SendToClient 调用。"""
import re
import subprocess
import sys

FUNCS = [
    ("CGameScene::SendPropertyTable", 2, 0xFE050),
    ("CGameScene::SendRecordTable", 2, 0xFE1C0),
    ("CGameScene::EntryVisual", 2, 0xF3F10),
    ("CGameScene::LeaveVisual", 2, 0xF4290),
    ("CGameScene::SendLocationInfo", 2, 0xF3860),
    ("CGameScene::SendMovingInfo", 2, 0xF39D0),
    ("CGamePlayer::SendCreateView", 2, 0xEE0C0),
    ("CGamePlayer::SendDeleteView", 2, 0xEE2A0),
    ("CGamePlayer::ViewNotifyAdd", 2, 0xEEE60),
    ("CGamePlayer::ViewNotifyRemove", 2, 0xEF050),
    ("CGamePlayer::ViewNotifyChange", 2, 0xEF100),
    ("CGamePlayer::SendToClient", 2, 0xEC7B0),
    ("CGamePlayer::SendToClient2", 2, 0xEC860),
    ("CGameObject::SendRecAppend", 2, 0xA79E0),
    ("CGameObject::SendRecDelete", 2, 0xA7A20),
    ("CGameObject::SendRecClear", 2, 0xA7A60),
    ("CGameObject::SendRecModify", 2, 0xA7AA0),
    ("CGameObject::SendRecSetRow", 2, 0xA7AE0),
    ("CGameObject::SendVisInfo", 2, 0xA7860),
    ("CGameObject::SendOwnInfo", 2, 0xA78E0),
    ("CGameVisible::SendVisInfo", 2, 0x106370),
    ("CGameVisible::SendOwnInfo", 2, 0x106980),
    ("CGameVisible::SendAllRecord", 2, 0x107150),
    ("CGameVisible::SendLinkInfo", 2, 0x107980),
    ("CGameVisible::SendViewRecord", 2, 0x107AA0),
    ("CGameVisible::SendViewObjRecord", 2, 0x107BC0),
    ("CGameScene::MessageReady", 2, 0xFFE20),
]


def main():
    for name, seg, off in FUNCS:
        out = subprocess.run(
            [sys.executable, "-I", "tools/pdbhack/dis.py", str(seg), hex(off), "0x600"],
            capture_output=True, text=True).stdout
        lines = out.splitlines()
        body = []
        for ln in lines:
            if "int3" in ln:
                if body:
                    break
                continue
            body.append(ln)
        print("=" * 100)
        print("### %s (seg%d off 0x%X)" % (name, seg, off))
        for ln in body:
            # opcode 写入 / 长度常量 / send 调用
            if re.search(r"mov +byte ptr \[.*\], 0x", ln):
                print("  OPC " + ln)
            elif re.search(r"push +0x[0-9a-f]{1,3}$", ln) and not re.search(r"push +0x[0-9a-f]{4,}", ln):
                print("  ARG " + ln)
            elif re.search(r"(call|jmp|je |jne )", ln) and "0x10" in ln:
                pass  # 调用目标略
        for ln in body:
            m = re.match(r"0x([0-9a-f]+)", ln)
            if m and int(m.group(1), 16) > 0x10300000:
                print("  CALL/DATA " + ln)
                break


if __name__ == "__main__":
    main()
