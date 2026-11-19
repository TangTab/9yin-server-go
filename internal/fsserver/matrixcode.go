package fsserver

import "fmt"

// MatrixCode 复刻 CMatrixCode（matrixcode.cpp，§4.3）：343,000 项数字
// 一次性密码矩阵 + 位置轮转 + 三坐标挑战串。
//
// 官方 ctor：数组顺序填 0..9（i%10）→ srand → 343,000 次 SwapValue 洗牌。
// GetMaxtrixCode(pos)=matrix[(curPos+pos)%343000]；IncMatrixPos 每秒 tick 前移。
type MatrixCode struct {
	matrix [matrixSize]byte
	pos    int
}

const (
	matrixSize = 343000 // 0x53BD8（ctor 循环上限断言）
	seedInit   = 342999 // 0x53BD7（ctor 立即数）
)

// msvcRand 复刻 MSVCCRT rand()：LCG，((seed>>16)&0x7FFF)。
type msvcRand struct{ state uint32 }

func (r *msvcRand) next() int {
	r.state = r.state*214013 + 2531011
	return int((r.state >> 16) & 0x7FFF)
}

// NewMatrixCode 按官方算法确定性构造：顺序填 i%10 → srand(0x53BD7) →
// 343,000 次 (rand()×rand()) 换位（ctor 0x1009CF80 循环 + SwapValue）。
// 同 seed 同矩阵；挑战串/取码/位置轮转与官方逐 API 对齐。
func NewMatrixCode() *MatrixCode {
	m := &MatrixCode{}
	for i := 0; i < matrixSize; i++ {
		m.matrix[i] = byte(i % 10)
	}
	rnd := &msvcRand{state: seedInit}
	for i := 0; i < matrixSize; i++ {
		a := (rnd.next() * rnd.next()) % matrixSize
		m.matrix[i], m.matrix[a] = m.matrix[a], m.matrix[i]
	}
	return m
}

// GetMaxtrixCode 位置取码（官方拼写保留）。
func (m *MatrixCode) GetMaxtrixCode(pos int) int {
	return int(m.matrix[((m.pos+pos)%matrixSize+matrixSize)%matrixSize])
}

// SetMatrixPos / IncMatrixPos 位置驱动。
func (m *MatrixCode) SetMatrixPos(pos int) { m.pos = ((pos % matrixSize) + matrixSize) % matrixSize }

// IncMatrixPos …
func (m *MatrixCode) IncMatrixPos() { m.pos = (m.pos + 1) % matrixSize }

// Pos …
func (m *MatrixCode) Pos() int { return m.pos }

// 矩阵坐标词表（GetMatrixString 0x1009D1A0 .rdata 串实证）。
var (
	matrixCols = [7]string{"A", "B", "C", "D", "E", "F", "G"}
	matrixRows = [10]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
)

// GetMatrixString 挑战串 "%s%s,%s%s,%s%s"：pos 数字序列逐位拆十进制，
// 位对 (列字母, 行号) 三组（官方解码序：pos%10 → pos/=10 循环）。
func (m *MatrixCode) GetMatrixString(pos int) string {
	digits := make([]int, 0, 6)
	p := pos
	if p == 0 {
		digits = append(digits, 0)
	}
	for p > 0 && len(digits) < 6 {
		digits = append(digits, p%10)
		p /= 10
	}
	for len(digits) < 6 {
		digits = append(digits, 0)
	}
	// 每组两位：(列 0..6, 行 0..9)
	pick := func(a, b int) string {
		return matrixCols[a%len(matrixCols)] + matrixRows[b%len(matrixRows)]
	}
	return fmt.Sprintf("%s,%s,%s",
		pick(digits[0], digits[1]), pick(digits[2], digits[3]), pick(digits[4], digits[5]))
}
