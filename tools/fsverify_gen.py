# -*- coding: utf-8 -*-
"""FsVerify.dll → Go 复刻生成器。

1. unicorn oracle（fsverify_oracle）执行官方 DLL 全部 encode_func_N；
2. capstone 提升每个函数的字节运算程序（K 步：in[off] 经 {+,-,|,^,&} 链 → scratch）；
3. 程序 + 标准 MD5 在 python 复算，与 oracle 逐字节对拍（全 2048 × 多输入）；
4. 提取 g_encode_order / cond16 / code16 三表；
5. 产出 internal/fsverify/*.go 与 golden 向量文件。

用法：
  python fsverify_gen.py            # 全流程
  python fsverify_gen.py --check    # 只做对拍校验
"""
import argparse
import hashlib
import random
import re
import struct
import sys
import bisect

sys.path.insert(0, __file__.rsplit("\\", 1)[0])
from fsverify_oracle import Oracle, IMAGE_BASE, DLL_PATH  # noqa: E402

from capstone import Cs, CS_ARCH_X86, CS_MODE_32  # noqa: E402
from pdb_syms import Pdb, iter_syms, S_PUB32, demangle  # noqa: E402

GO_DIR = r"G:\pojie\ai\9yin-server-go\internal\fsverify"
GOLDEN = r"G:\pojie\ai\9yin-server-go\testdata\fsverify\encode-vectors.txt"

# 已知 thiscall 助手（thunk 一跳后的真身），lifter 靠它识别 md5 骨架
HELPER_MD5_CTOR = 0x100D7980
HELPER_MD5_UPDATE = 0x100D7CB0
HELPER_MD5_FINALIZE = 0x100D7E20
HELPER_MD5_DIGEST = 0x100D79E0
HELPER_MEMCPY = 0x10213C80


def load_syms():
    pdb = Pdb(DLL_PATH.replace(".dll", ".pdb"))
    dll = open(DLL_PATH, "rb").read()
    pe = struct.unpack_from("<I", dll, 0x3C)[0]
    nsec = struct.unpack_from("<H", dll, pe + 6)[0]
    optsz = struct.unpack_from("<H", dll, pe + 20)[0]
    imagebase = struct.unpack_from("<I", dll, pe + 24 + 28)[0]
    secs = []
    so = pe + 24 + optsz
    for i in range(nsec):
        o = so + i * 40
        name = dll[o : o + 8].rstrip(b"\0").decode()
        vsz, va, rsz, raw = struct.unpack_from("<IIII", dll, o + 8)
        secs.append((name, imagebase + va, vsz, raw, rsz))
    def seg2rva(seg, off):
        if 1 <= seg <= len(secs):
            return secs[seg - 1][1] - imagebase + off
        return None
    syms = {}
    for rt, off, seg, ln, name in iter_syms(pdb.streams[pdb.dbi_header()["symrec"]]):
        if rt == S_PUB32 and name:
            rva = seg2rva(seg, off)
            if rva is not None and rva not in syms:
                syms[rva] = demangle(name)
    return dll, secs, syms


class Lifter:
    def __init__(self, dll, secs, syms, encode_addrs):
        self.dll = dll
        self.va2off = lambda va: self._va2off(secs, va)
        self.syms = syms
        self.addrs = sorted(syms)
        self.encode_addrs = encode_addrs
        self.md = Cs(CS_ARCH_X86, CS_MODE_32)
        self.thunk_cache = {}

    def _va2off(self, secs, va):
        for n, sva, vsz, raw, rsz in secs:
            if raw and sva <= va < sva + max(vsz, rsz):
                return raw + (va - sva)
        return None

    def hop(self, va):
        if va in self.thunk_cache:
            return self.thunk_cache[va]
        off = self.va2off(va)
        r = va
        if off and self.dll[off] == 0xE9:
            r = va + 5 + struct.unpack_from("<i", self.dll, off + 1)[0]
        self.thunk_cache[va] = r
        return r

    def func_end(self, va):
        i = bisect.bisect_right(self.addrs, va)
        if i < len(self.addrs):
            return IMAGE_BASE + self.addrs[i]
        return va + 0x400

    def lift(self, idx):
        """返回 (ops, scratch_len)；ops = [(src_off, [(op,imm)...]), ...]

        轻量数据流：追踪"当前持有输入指针"的寄存器（mov reg,[ebp+8]），
        顺序识别「字节装载（movzx r32 / mov r8l）+ 算术链 + 存储到游标」
        三段式操作；md5 Update 的长度参数即 scratch 长度。
        """
        va = self.encode_addrs[idx]
        end = self.func_end(va)
        off = self.va2off(va)
        insns = [i for i in self.md.disasm(self.dll[off : off + (end - va)], va) if i.mnemonic != "int3"]
        # 符号间隙会让窗口溢出到下一个函数——在首个 ret 处截断
        for k, ins in enumerate(insns):
            if ins.mnemonic == "ret":
                insns = insns[: k + 1]
                break
        n = len(insns)
        ops = []
        scratch_len = None
        inregs = set()  # 持有输入指针的寄存器
        RE_BYTE_LD = r"(\w+), byte ptr \[(\w+)(?: \+ (0x[0-9a-f]+|[0-9]+))?\]"
        i = 0
        while i < n:
            ins = insns[i]
            op_str = ins.op_str
            # mov reg, dword ptr [ebp + 8] → 输入指针装载
            if ins.mnemonic == "mov" and op_str in ("eax, dword ptr [ebp + 8]",) :
                inregs.add("eax")
                i += 1
                continue
            if ins.mnemonic == "mov" and re.fullmatch(r"(\w+), dword ptr \[ebp \+ 8\]", op_str):
                inregs.add(re.fullmatch(r"(\w+), dword ptr \[ebp \+ 8\]", op_str).group(1))
                i += 1
                continue
            acc = None
            m = re.fullmatch(RE_BYTE_LD, op_str)
            if m and m.group(2) in inregs:
                if ins.mnemonic == "movzx" and len(m.group(1)) in (3, 4):
                    acc = m.group(1)
                    src_off = int(m.group(3), 0) if m.group(3) else 0
                elif ins.mnemonic == "mov" and len(m.group(1)) == 2 and m.group(1)[1] in "lb":
                    acc = m.group(1)
                    src_off = int(m.group(3), 0) if m.group(3) else 0
            if acc:
                # 算术链（同寄存器，字面立即数；8 位名或 32 位名）
                chain = []
                j = i + 1
                lo = {"eax": "al", "ecx": "cl", "edx": "dl", "ebx": "bl", "esi": "sil", "edi": "dil"}.get(acc, acc)
                while j < n:
                    op = insns[j]
                    mm = re.fullmatch(r"(%s), (0x[0-9a-f]+|\d+)" % re.escape(acc), op.op_str)
                    if op.mnemonic in ("add", "sub", "or", "xor", "and") and mm:
                        chain.append((op.mnemonic, int(mm.group(2), 0) & 0xFF))
                        j += 1
                        continue
                    break
                # 存储窗口：找 mov byte ptr [X], lo（≤4 条内）
                k = j
                stored = False
                while k < min(j + 4, n):
                    st = insns[k]
                    if st.mnemonic == "mov" and re.fullmatch(
                        r"byte ptr \[\w+(?: \+ [^\]]+)?\], %s" % re.escape(lo), st.op_str
                    ):
                        stored = True
                        break
                    k += 1
                if stored:
                    ops.append((src_off, chain))
                    i = k + 1
                    continue
                i = j
                continue
            # push LEN … call md5_update → scratch 长度
            if ins.mnemonic == "push":
                try:
                    v = int(op_str, 0)
                except ValueError:
                    v = None
                if v is not None and 1 <= v <= 64:
                    for k2 in range(i + 1, min(i + 5, n)):
                        if insns[k2].mnemonic == "call":
                            t = self.hop(int(insns[k2].op_str, 16))
                            if t == HELPER_MD5_UPDATE:
                                scratch_len = v
                            break
            # 任何写入都可能改变寄存器含义：粗略清理 inregs
            if ins.mnemonic in ("mov", "lea") and "," in op_str:
                dst = op_str.split(",")[0]
                inregs.discard(dst)
            i += 1
        if scratch_len is None:
            return None, None
        if len(ops) != scratch_len:
            return None, ("op-count %d != scratch %d" % (len(ops), scratch_len))
        return ops, scratch_len


def py_encode(ops, scratch_len, in16):
    scratch = bytearray(scratch_len)
    for i, (src_off, chain) in enumerate(ops):
        v = in16[src_off]
        for op, imm in chain:
            if op == "add":
                v = (v + imm) & 0xFF
            elif op == "sub":
                v = (v - imm) & 0xFF
            elif op == "or":
                v |= imm
            elif op == "xor":
                v ^= imm
            elif op == "and":
                v &= imm
        scratch[i] = v
    return hashlib.md5(bytes(scratch)).digest()


def extract_tables(o, dll, secs):
    def rva_off(rva):
        for n, sva, vsz, raw, rsz in secs:
            if raw and sva <= IMAGE_BASE + rva < sva + max(vsz, rsz):
                return raw + (rva - (sva - IMAGE_BASE))
        return None

    # g_encode_order: u32[2048] @ 0x285b28
    encode_order = struct.unpack_from("<2048I", dll, rva_off(0x285B28))
    # cond: 2048×16B @ 0x287b28
    cond = dll[rva_off(0x287B28) : rva_off(0x287B28) + 2048 * 16]
    # code: get_client_code 跳转表 @ rva 0xe7058 → case 体 mov eax, imm32 → blob 头 16B
    jt = rva_off(0xE7058)
    entries = struct.unpack_from("<2048I", dll, jt)
    code_blob = bytearray()
    for case_va in entries:
        # case: mov eax, imm32 (B8 xx xx xx xx) ; jmp common
        coff = rva_off(case_va - IMAGE_BASE)
        assert dll[coff] == 0xB8, hex(case_va)
        blob_va = struct.unpack_from("<I", dll, coff + 1)[0]
        boff = rva_off(blob_va - IMAGE_BASE)
        code_blob += dll[boff : boff + 16]
    return encode_order, cond, bytes(code_blob)


def wrap_hex(data, per_line=32):
    h = data.hex()
    return "\n".join(h[i : i + per_line * 2] for i in range(0, len(h), per_line * 2))


def go_header(comment):
    return (
        "// Code generated by tools/fsverify_gen.py from FsVerify.dll "
        "(BGserverCJ 官方快照). DO NOT EDIT.\n"
        "// %s\npackage fsverify\n\n" % comment
    )


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()

    print("[*] 初始化 oracle …")
    o = Oracle()
    dll, secs, syms = load_syms()
    lifter = Lifter(dll, secs, syms, o.encode_addrs)

    print("[*] 提升 2048 个 encode_func …")
    programs = {}
    bad = []
    for idx in range(2048):
        ops, info = lifter.lift(idx)
        if ops is None:
            bad.append((idx, info))
        else:
            programs[idx] = (ops, info)
    print("    lifted=%d bad=%d" % (len(programs), len(bad)))
    if bad:
        for idx, why in bad[:10]:
            print("    bad#%d: %s" % (idx, why))

    print("[*] oracle 对拍（每函数 3 输入）…")
    rng = random.Random(20261119)
    vectors = {}
    mism = []
    for idx in range(2048):
        if idx not in programs:
            continue
        ops, ln = programs[idx]
        for k in range(3):
            inp = bytes(rng.randrange(256) for _ in range(16))
            exp = o.encode(idx, inp)
            got = py_encode(ops, ln, inp)
            if exp != got:
                mism.append((idx, k))
                break
        vectors[idx] = [(inp, exp)]
    print("    mismatch=%d / %d" % (len(mism), len(programs)))
    if mism:
        print("    sample:", mism[:5])
        sys.exit(1)

    print("[*] 提取三表 …")
    encode_order, cond, code = extract_tables(o, dll, secs)
    print("    encode_order[:4]=", [hex(x) for x in encode_order[:4]])
    print("    cond[0]=", cond[:16].hex())
    print("    code[0]=", code[:16].hex())

    if args.check:
        print("[check] 全部通过")
        return

    import os

    os.makedirs(GO_DIR, exist_ok=True)
    os.makedirs(GOLDEN.replace("\\encode-vectors.txt", ""), exist_ok=True)

    print("[*] 生成 Go 表 …")
    # programs → 紧凑二进制：每函数 [u8 len][每op: u8 src_off, u8 nsub, subs…]
    blob = bytearray()
    for idx in range(2048):
        ops, ln = programs[idx]
        blob.append(ln)
        for src_off, chain in ops:
            blob.append(src_off)
            blob.append(len(chain))
            for op, imm in chain:
                blob.append({"add": 1, "sub": 2, "or": 3, "xor": 4, "and": 5}[op])
                blob.append(imm)
    with open(GO_DIR + r"\encode_programs.go", "w", encoding="utf-8", newline="\n") as f:
        f.write(go_header("2048 个 encode_func_N 的字节运算程序（提升自反汇编，oracle 全量对拍）。"))
        f.write("// 编码：每函数 = u8 scratch长度 + N×(u8 src偏移, u8 子操作数, 子操作 (u8 opcode, u8 imm))；\n")
        f.write("// opcode: 1=add 2=sub 3=or 4=xor 5=and。输出 = md5(scratch)。\n\n")
        f.write("const encodeProgramsHex = \"\" +\n")
        h = blob.hex()
        for i in range(0, len(h), 72):
            f.write('\t"%s" +\n' % h[i : i + 72])
        f.write('\t""\n')

    with open(GO_DIR + r"\tables.go", "w", encoding="utf-8", newline="\n") as f:
        f.write(go_header("g_encode_order（u32[2048] 置换）/ g_cond_value（2048×16B）/ g_client_code 头 16B 三表。"))
        f.write("import \"encoding/hex\"\n\n")
        f.write("func mustHex(s string) []byte {\n\tb, err := hex.DecodeString(s)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\treturn b\n}\n\n")
        f.write("const encodeOrderHex = \"\" +\n")
        h = struct.pack("<2048I", *encode_order).hex()
        for i in range(0, len(h), 72):
            f.write('\t"%s" +\n' % h[i : i + 72])
        f.write('\t""\n\n')
        f.write("const clientCondHex = \"\" +\n")
        h = cond.hex()
        for i in range(0, len(h), 72):
            f.write('\t"%s" +\n' % h[i : i + 72])
        f.write('\t""\n\n')
        f.write("const clientCodeHex = \"\" +\n")
        h = code.hex()
        for i in range(0, len(h), 72):
            f.write('\t"%s" +\n' % h[i : i + 72])
        f.write('\t""\n')

    print("[*] 生成 golden 向量 …")
    with open(GOLDEN, "w", encoding="utf-8", newline="\n") as f:
        for idx in range(2048):
            inp, exp = vectors[idx][0]
            f.write("%d %s %s\n" % (idx, inp.hex(), exp.hex()))

    print("[+] 完成：internal/fsverify/ 生成 %d 程序 + 三表 + %d 向量" % (len(programs), len(vectors)))


if __name__ == "__main__":
    main()
