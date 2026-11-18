// Package fsverify 是官方 FsVerify.dll（BGserverCJ 2014 快照，带 PDB 全符号）
// 的 Go 复刻。原 DLL 是接入服的可选消息校验插件（官方 game_entry.ini 里
// //MessageVerifyDll=fsverify.dll 默认注释禁用），导出
// FxModule_GetType/FxVerify_GetInterface 两个工厂入口，实现 IVerify 虚接口：
//
//	slot2 GetVersion()     恒 300（本构建；2016 线上客户端版本=333 由另版 DLL 供给）
//	slot3 GetEncodeNum()   恒 2048
//	slot4 GetEncodeId(i)   g_encode_order[i]（0..2047 的固定置换表）
//	slot5 GetClientCond(i) g_cond_value[i] 16 字节（SERVER_SET_ENCODE 下发）
//	slot6 GetClientCode(i) g_client_code 变长 blob 的头 16 字节（同上）
//	slot7 EncodeFunc(id, in, out) encode_func_N：8~9 步字节运算程序 → md5 → 16 字节
//	slot8 DecodeAccount(slot, len)  账号槽原地解密（DES 阶梯 + 奇偶掩码）
//	slot9 DecodePassword(slot, len) 密码槽原地解密（同构，独立键系）
//
// 消费方（FsSoloEntry 反汇编实证）：
//
//   - COuterHook::CheckMessageIllegal 按 opcode 分发：op01=CLIENT_RET_ENCODE
//     （≥0x35B，VerifyMessage 比对 16B 令牌）、op02=ClientLogin（版本门
//     vfy->GetVersion() vs 帧 [1:5]，双槽 DecodeAccount/DecodePassword 原地解密，
//     槽长 ≤0x40）、op04=choose_role、op07=select（均内嵌同款令牌）。
//   - COuterHook::SendClientEncode：opcode=2 帧，[0x21..0x2d)=GetClientCond、
//     [0x2d..0x3d)=GetClientCode（各 16B）、[0x3d..)=客户端三字段。
//   - COuterHook::VerifyMessage(vfy, id, token, src, len)：EncodeFunc(id, src)
//     与令牌比对；失败且 PrevEncodeId 有效时以 prev id 再试一次。
//   - vfy==NULL 时 CheckMessageIllegal 恒返回放行——生产配置禁用即全跳过。
//
// 算法与常量三方实证一致：官方 DLL 反汇编（des2key/D2des/两槽键与掩码）、
// B 线 fork 客户端逆向（official-2433-b-line internal/auth/field_cipher.go）、
// 官方 golden 线路帧实测（testdata/captures/login-connect-001.bin 解出
// test001 / drUGtwMnNPbVE9PQ==VjYzaVZab0xjQ0M3bD）。
//
// EncodeFunc 的 2048 个生成函数由 tools/fsverify_gen.py 自动提升（unicorn
// oracle 全量对拍 2048×3 输入零失配）；三表直接从 .data 提取。
package fsverify

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"math/bits"
)

// ModuleVersion 对应 CVerify::GetVersion（mov eax, 0x12c）。
const ModuleVersion = 300

// EncodeNum 对应 CVerify::GetEncodeNum（mov eax, 0x800）。
const EncodeNum = 2048

// ErrEncodeIndex 越界的编码索引（官方方言：EncodeFunc 对 >0x7ff 直接返回）。
var ErrEncodeIndex = errors.New("fsverify: encode index out of range")

// ---------------------------------------------------------------------------
// 双域 DES 阶梯（des2key + D2des 反汇编等价；与 B 线 field_cipher.go 同构）
// ---------------------------------------------------------------------------

// field1（账号槽）/field2（密码槽）16 字节固定域键：
// [0:8]=阶梯 E 级键 kA（des2key 先以 key[0:8] 建立 KnL），[8:16]=D 级键 kB。
var (
	field1Key = [16]byte{0x66, 0xf9, 0x28, 0xe8, 0xa1, 0x00, 0x9b, 0x76, 0x29, 0xa0, 0x1f, 0x8d, 0xef, 0x45, 0x89, 0x19}
	field2Key = [16]byte{0x10, 0x6e, 0xba, 0x55, 0x77, 0x61, 0xc7, 0x7e, 0x1e, 0xea, 0x0d, 0xeb, 0x67, 0x62, 0x93, 0xdc}

	// 块序偶/奇时逐 LE32 异或的 16 字节掩码（DecodeAccount/DecodePassword 立即数）。
	field1Even = [4]uint32{0xf8ebd0a7, 0xc4badc9c, 0xa84fd0a2, 0x74cf24b7}
	field1Odd  = [4]uint32{0x26e24c92, 0x38efda09, 0xc3b2d047, 0x28efd49c}
	field2Even = [4]uint32{0xf24d83e5, 0xab2526d8, 0x32b93688, 0xaffdfe20}
	field2Odd  = [4]uint32{0xe2435ae5, 0x136126b1, 0x52ba6e85, 0x3fa3fb23}
)

type fieldLadder struct {
	keyAEncrypt, keyADecrypt cipher.Block // kA 双向
	keyBEncrypt, keyBDecrypt cipher.Block // kB 双向
	even, odd                [4]uint32
}

func newFieldLadder(key [16]byte, even, odd [4]uint32) fieldLadder {
	a, errA := des.NewCipher(key[0:8])
	b, errB := des.NewCipher(key[8:16])
	if errA != nil || errB != nil {
		// 固定 8 字节键恒有效
		panic("fsverify: invalid field key")
	}
	return fieldLadder{keyAEncrypt: a, keyADecrypt: a, keyBEncrypt: b, keyBDecrypt: b, even: even, odd: odd}
}

// desStep 以 (l, r) 组 8 字节块跑一步标准 DES。
func desStep(c cipher.Block, decrypt bool, l, r uint32) (uint32, uint32) {
	var blk [8]byte
	binary.BigEndian.PutUint32(blk[0:4], l)
	binary.BigEndian.PutUint32(blk[4:8], r)
	if decrypt {
		c.Decrypt(blk[:], blk[:])
	} else {
		c.Encrypt(blk[:], blk[:])
	}
	return binary.BigEndian.Uint32(blk[0:4]), binary.BigEndian.Uint32(blk[4:8])
}

// transform128 加密阶梯（客户端编码方向）：
// E_kA 两半独立 → D_kB 左左/右右交叉 → E_kA 再独立。
func (f fieldLadder) transform128(w0, w1, w2, w3 uint32) [4]uint32 {
	a, b := desStep(f.keyAEncrypt, false, w0, w1)
	c, d := desStep(f.keyAEncrypt, false, w2, w3)
	e, f2 := desStep(f.keyBDecrypt, true, a, c)
	g, h := desStep(f.keyBDecrypt, true, b, d)
	i, j := desStep(f.keyAEncrypt, false, e, g)
	k, l := desStep(f.keyAEncrypt, false, f2, h)
	return [4]uint32{i, j, k, l}
}

// untransform128 解密阶梯（服务端 DecodeAccount/DecodePassword 方向）：
// D_kA → E_kB → D_kA（官方 des2key(key,1)+D2des：KnL=解密 kA、KnR=加密 kB）。
func (f fieldLadder) untransform128(i, j, k, l uint32) [4]uint32 {
	e, g := desStep(f.keyADecrypt, true, i, j)
	f2, h := desStep(f.keyADecrypt, true, k, l)
	a, c := desStep(f.keyBEncrypt, false, e, f2)
	b, d := desStep(f.keyBEncrypt, false, g, h)
	w0, w1 := desStep(f.keyADecrypt, true, a, b)
	w2, w3 := desStep(f.keyADecrypt, true, c, d)
	return [4]uint32{w0, w1, w2, w3}
}

var (
	accountField  = newFieldLadder(field1Key, field1Even, field1Odd)
	passwordField = newFieldLadder(field2Key, field2Even, field2Odd)
)

// decodeField 是 DecodeAccount/DecodePassword 的公共骨架（16 字节整倍逐块）：
// 逆阶梯 → 块序奇偶掩码 XOR。官方 CVerify 方法为原地解密；这里返回新切片。
func decodeField(f fieldLadder, slot []byte) []byte {
	out := make([]byte, len(slot))
	for idx := 0; idx*16 < len(slot); idx++ {
		blk := slot[idx*16 : idx*16+16]
		words := f.untransform128(
			binary.BigEndian.Uint32(blk[0:4]),
			binary.BigEndian.Uint32(blk[4:8]),
			binary.BigEndian.Uint32(blk[8:12]),
			binary.BigEndian.Uint32(blk[12:16]))
		mask := f.even
		if idx%2 == 1 {
			mask = f.odd
		}
		var plain [16]byte
		for w := 0; w < 4; w++ {
			binary.LittleEndian.PutUint32(plain[w*4:w*4+4], bits.ReverseBytes32(words[w])^mask[w])
		}
		copy(out[idx*16:idx*16+16], plain[:])
	}
	return out
}

// encodeField 是 decodeField 的逆（客户端槽编码方向，供测试/种子数据）。
func encodeField(f fieldLadder, padded []byte) []byte {
	out := make([]byte, len(padded))
	for idx := 0; idx*16 < len(padded); idx++ {
		blk := padded[idx*16 : idx*16+16]
		mask := f.even
		if idx%2 == 1 {
			mask = f.odd
		}
		w := [4]uint32{}
		for n := 0; n < 4; n++ {
			w[n] = bits.ReverseBytes32(binary.LittleEndian.Uint32(blk[n*4:n*4+4]) ^ mask[n])
		}
		words := f.transform128(w[0], w[1], w[2], w[3])
		var ct [16]byte
		for n := 0; n < 4; n++ {
			binary.BigEndian.PutUint32(ct[n*4:n*4+4], words[n])
		}
		copy(out[idx*16:idx*16+16], ct[:])
	}
	return out
}

// pad16 零填充到 16 字节整倍（官方账号槽 pad16）。
func pad16(data []byte) []byte {
	if len(data)%16 == 0 {
		return data
	}
	padded := make([]byte, (len(data)+15)&^15)
	copy(padded, data)
	return padded
}

// DecodeAccount 解密 op02 账号槽，返回去零填充的明文。
// 与 CVerify::DecodeAccount(slot, len) 等价（键/掩码/方向同）。
func DecodeAccount(slot []byte) string {
	return trimPad(decodeField(accountField, slot))
}

// DecodePassword 解密 op02 密码槽。解出的是客户端派生形态
// （ComputeLoginPassword 的产物），非原始口令——等值比对即可。
func DecodePassword(slot []byte) string {
	return trimPad(decodeField(passwordField, slot))
}

// EncodeAccountFields 是双槽加密（客户端方向）：确定性，同输入恒同输出。
func EncodeAccountFields(account, derivedPassword string) (accountCiphertext, passwordCiphertext []byte) {
	accountCiphertext = encodeField(accountField, pad16([]byte(account)))
	passwordCiphertext = encodeField(passwordField, pad16([]byte(derivedPassword)))
	return
}

func trimPad(padded []byte) string {
	end := len(padded)
	for end > 0 && padded[end-1] == 0 {
		end--
	}
	return string(padded[:end])
}

// ---------------------------------------------------------------------------
// 编码握手（SERVER_SET_ENCODE / CLIENT_RET_ENCODE）
// ---------------------------------------------------------------------------

var (
	encodeOrder = mustHex(encodeOrderHex)
	clientCond  = mustHex(clientCondHex)
	clientCode  = mustHex(clientCodeHex)
)

// EncodeID 对应 CVerify::GetEncodeId(i)=g_encode_order[i]。
// i≥2048 时官方先触发断言（本实现返回 false）。
func EncodeID(i uint32) (uint32, bool) {
	if i >= EncodeNum {
		return 0, false
	}
	return binary.LittleEndian.Uint32(encodeOrder[i*4 : i*4+4]), true
}

// ClientCond 对应 CVerify::GetClientCond(i)：16 字节条件值。
func ClientCond(i uint32) ([]byte, bool) {
	if i >= EncodeNum {
		return nil, false
	}
	out := make([]byte, 16)
	copy(out, clientCond[i*16:i*16+16])
	return out, true
}

// ClientCode 对应 CVerify::GetClientCode(i)：变长 blob 的头 16 字节
// （SERVER_SET_ENCODE 只拷贝 16 字节，线上仅此片段有语义）。
func ClientCode(i uint32) ([]byte, bool) {
	if i >= EncodeNum {
		return nil, false
	}
	out := make([]byte, 16)
	copy(out, clientCode[i*16:i*16+16])
	return out, true
}

// EncodeFunc 对应 CVerify::EncodeFunc(id, in, out)：
// encode_func_id 的字节程序跑在 in 前 16 字节上，md5(scratch) 写入 out。
func EncodeFunc(id int, in []byte) ([16]byte, error) {
	prog, err := programAt(id)
	if err != nil {
		return [16]byte{}, err
	}
	if len(in) < 16 {
		return [16]byte{}, errors.New("fsverify: encode input shorter than 16 bytes")
	}
	scratch := make([]byte, prog.scratchLen)
	for i, op := range prog.ops {
		v := in[op.src]
		for _, sub := range op.chain {
			switch sub.opcode {
			case opAdd:
				v += sub.imm
			case opSub:
				v -= sub.imm
			case opOr:
				v |= sub.imm
			case opXor:
				v ^= sub.imm
			case opAnd:
				v &= sub.imm
			}
		}
		scratch[i] = v
	}
	return md5.Sum(scratch), nil
}

// VerifyClientToken 对应 COuterHook::VerifyMessage 的比对语义：
// 以 currentID 计算期望令牌；失配且 prevID 可用时以 prevID 再试。
func VerifyClientToken(currentID, prevID uint32, prevValid bool, src, token []byte) bool {
	if expect, err := EncodeFunc(int(currentID), src); err == nil && equal16(expect, token) {
		return true
	}
	if !prevValid || prevID == currentID {
		return false
	}
	expect, err := EncodeFunc(int(prevID), src)
	return err == nil && equal16(expect, token)
}

func equal16(a [16]byte, token []byte) bool {
	if len(token) < 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		if a[i] != token[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// encode_func 程序表（encode_programs.go 的十六进制 blob 解码）
// ---------------------------------------------------------------------------

const (
	opAdd = 1 + iota
	opSub
	opOr
	opXor
	opAnd
)

type subOp struct{ opcode, imm byte }
type encOp struct {
	src   byte
	chain []subOp
}
type encProgram struct {
	scratchLen byte
	ops        []encOp
}

var programsTable [EncodeNum]encProgram

// programAt 取第 id 个程序（init 阶段已全量解码）。
func programAt(id int) (encProgram, error) {
	if id < 0 || id >= EncodeNum {
		return encProgram{}, ErrEncodeIndex
	}
	return programsTable[id], nil
}

func initPrograms() {
	blob := mustHex(encodeProgramsHex)
	pos := 0
	for i := 0; i < EncodeNum; i++ {
		scratchLen := blob[pos]
		pos++
		ops := make([]encOp, scratchLen)
		for k := 0; k < int(scratchLen); k++ {
			src := blob[pos]
			nSub := int(blob[pos+1])
			pos += 2
			chain := make([]subOp, nSub)
			for s := 0; s < nSub; s++ {
				chain[s] = subOp{opcode: blob[pos], imm: blob[pos+1]}
				pos += 2
			}
			ops[k] = encOp{src: src, chain: chain}
		}
		programsTable[i] = encProgram{scratchLen: scratchLen, ops: ops}
	}
	if pos != len(blob) {
		panic("fsverify: encode programs blob length mismatch")
	}
}

func init() {
	initPrograms()
}
