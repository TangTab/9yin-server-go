package fssoloauction

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"unicode/utf16"

	"github.com/lizhanyin/9yin-server-go/internal/auction"
	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// ---- PUB4 归档（AuctionDataArchive.obj）----
//
// FsSoloAuction 本身不落盘、无 SQL（导入表/GBK 串扫描定案）；存取全部
// 经 inner 发给 store（comm 4..7），块格式自有二进制 "PUB4" 归档：
//
//	attr 块（SaveAttr @0x1008D8D0；0x41 发送 / 0x29 回放）：
//	  +0x00 u32 magic 0x50554234 "PUB4"   +0x04 u32 保留
//	  +0x08 char uid[0x24]  +0x2C char name[0x24]（GBK）
//	  +0x50 u16 0          +0x52 attr 流：{ str(name) | u8 type | value }…
//	record 块（SaveRecordHead @0x1008E470 + SaveRecordBody @0x1008E950；
//	0x42 发送 / 0x2A 回放）：
//	  固定头 0xAC（+0x2C ANSI 记录名[0x24]、+0x98 rowMax、+0x9C rows、
//	  +0xA0 cols、+0xA4 索引数、+0xA8 0）→ 流：
//	  每列 u8 col_type；每索引 {u32, u8}；WriteInt(索引数)；
//	  每索引 { str(name), 参数… }；随后体：WriteInt(start) WriteInt(end)
//	  + 每行每列 { u8 脏标志, value(按列类型，编码同 vary 值) }。
//	行分片：SaveRecordBody 每片 ≤ 0x100 行（InnerSaveAuctionData）。
//
// 回放消息体（0x29/0x2A）：{ u32 result } + 块（result==0 →
// "load pub data failed"）；0x2B LoadRecordOk：{ u32 result }（非 0 →
// Opened + OnAuctionDataLoaded）。
const (
	pub4Magic      = 0x50554234
	attrUIDOff     = 0x08
	attrNameOff    = 0x2C
	attrStreamOff  = 0x52
	attrBlockSize  = 0x52 // 定长前缀（流起点）
	recHeadSize    = 0xAC
	recNameOff     = 0x2C
	recRowMaxOff   = 0x98
	recRowsOff     = 0x9C
	recColsOff     = 0xA0
	recIndexCntOff = 0xA4
	recShardRows   = 0x100
	storeMsgLoad   = 0x40 // 拍卖→store：加载请求 {district,server}
	storeMsgAttr   = 0x41 // 拍卖→store：存属性块
	storeMsgRecord = 0x42 // 拍卖→store：存记录分片
	storeBackAttr  = 0x29 // store→拍卖：回放属性块
	storeBackRec   = 0x2A // store→拍卖：回放记录块
	storeBackOK    = 0x2B // store→拍卖：加载完成
)

// mainCols 是 MainRecord 18 列线型（官方建表列类型序列，fsauctionlogic
// 文档 §四 RecConfig 实证），与 auction.Row 字段一一对应。
var mainCols = []byte{
	proto.VarString, proto.VarString, proto.VarInt32, proto.VarString,
	proto.VarString, proto.VarInt64, proto.VarInt64, proto.VarFloat,
	proto.VarInt64, proto.VarInt64, proto.VarString, proto.VarInt64,
	proto.VarInt32, proto.VarInt32, proto.VarWide, proto.VarInt32,
	proto.VarInt32, proto.VarString,
}

// rowValues 把一行 18 列打包为 proto.Value（列序=mainCols）。
func rowValues(r auction.Row) []proto.Value {
	return []proto.Value{
		{Type: proto.VarString, Text: r.UniqueID},
		{Type: proto.VarString, Text: r.ConfigID},
		{Type: proto.VarInt32, I64: int64(r.SortType)},
		{Type: proto.VarString, Text: r.ItemString},
		{Type: proto.VarString, Text: r.SellerUid},
		{Type: proto.VarInt64, I64: r.StartPrice},
		{Type: proto.VarInt64, I64: r.FixedPrice},
		{Type: proto.VarFloat, F64: float64(r.TransFeeCoe)},
		{Type: proto.VarInt64, I64: r.KeepingFee},
		{Type: proto.VarInt64, I64: r.OverTime},
		{Type: proto.VarString, Text: r.CurBiderUid},
		{Type: proto.VarInt64, I64: r.CurPrice},
		{Type: proto.VarInt32, I64: int64(r.State)},
		{Type: proto.VarInt32, I64: int64(r.ItemType)},
		{Type: proto.VarWide, Text: r.SellerName},
		{Type: proto.VarInt32, I64: int64(r.Level)},
		{Type: proto.VarInt32, I64: int64(r.Quality)},
		{Type: proto.VarString, Text: r.ExtraStr},
	}
}

// valuesRow 是 rowValues 的逆。
func valuesRow(vs []proto.Value) (auction.Row, error) {
	if len(vs) != len(mainCols) {
		return auction.Row{}, fmt.Errorf("列数 %d ≠ %d", len(vs), len(mainCols))
	}
	for i, t := range mainCols {
		if vs[i].Type != t {
			return auction.Row{}, fmt.Errorf("第 %d 列类型 %d ≠ %d", i, vs[i].Type, t)
		}
	}
	return auction.Row{
		UniqueID:    vs[0].Text,
		ConfigID:    vs[1].Text,
		SortType:    int32(vs[2].I64),
		ItemString:  vs[3].Text,
		SellerUid:   vs[4].Text,
		StartPrice:  vs[5].I64,
		FixedPrice:  vs[6].I64,
		TransFeeCoe: float32(vs[7].F64),
		KeepingFee:  vs[8].I64,
		OverTime:    vs[9].I64,
		CurBiderUid: vs[10].Text,
		CurPrice:    vs[11].I64,
		State:       int32(vs[12].I64),
		ItemType:    int32(vs[13].I64),
		SellerName:  vs[14].Text,
		Level:       int32(vs[15].I64),
		Quality:     int32(vs[16].I64),
		ExtraStr:    vs[17].Text,
	}, nil
}

// ---- TStoreArchive/CLoadArchive 原语的 Go 等价 ----

type storeWriter struct{ b []byte }

func (w *storeWriter) byte(v byte)   { w.b = append(w.b, v) }
func (w *storeWriter) u32(v uint32)  { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *storeWriter) i32(v int32)   { w.u32(uint32(v)) }
func (w *storeWriter) i64(v int64)   { w.b = binary.LittleEndian.AppendUint64(w.b, uint64(v)) }
func (w *storeWriter) f32(v float32) { w.u32(math.Float32bits(v)) }
func (w *storeWriter) f64v(v float64) {
	w.b = binary.LittleEndian.AppendUint64(w.b, math.Float64bits(v))
}

// str 官方 WriteString：u32 长度（含 NUL）+ 字节 + NUL。
func (w *storeWriter) str(s string) {
	w.u32(uint32(len(s) + 1))
	w.b = append(w.b, s...)
	w.b = append(w.b, 0)
}

// wide 官方 WriteWideStr：u32 字节长（含 L'\0'）+ UTF-16LE + 00 00。
func (w *storeWriter) wide(s string) {
	units := utf16.Encode([]rune(s))
	w.u32(uint32((len(units) + 1) * 2))
	var b [2]byte
	for _, u := range units {
		binary.LittleEndian.PutUint16(b[:], u)
		w.b = append(w.b, b[:]...)
	}
	w.b = append(w.b, 0, 0)
}

// value 按类型写值（SaveRecordBody 跳表 [2..7] 的七型）。
func (w *storeWriter) value(v proto.Value) error {
	switch v.Type {
	case proto.VarInt32:
		w.i32(int32(v.I64))
	case proto.VarInt64:
		w.i64(v.I64)
	case proto.VarFloat:
		w.f32(float32(v.F64))
	case proto.VarDouble:
		w.f64v(v.F64)
	case proto.VarString:
		w.str(v.Text)
	case proto.VarWide:
		w.wide(v.Text)
	default:
		return fmt.Errorf("不支持的列类型 %d", v.Type)
	}
	return nil
}

// storeReader 官方 CLoadArchive（越界报错等价 "Out range"）。
type storeReader struct {
	b   []byte
	pos int
}

func (r *storeReader) remain() int { return len(r.b) - r.pos }
func (r *storeReader) take(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.b) {
		return nil, fmt.Errorf("Out range：取 %d 余 %d", n, r.remain())
	}
	p := r.b[r.pos : r.pos+n]
	r.pos += n
	return p, nil
}
func (r *storeReader) readByte() (byte, error) {
	p, err := r.take(1)
	if err != nil {
		return 0, err
	}
	return p[0], nil
}
func (r *storeReader) readU32() (uint32, error) {
	p, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(p), nil
}
func (r *storeReader) readStr() (string, error) {
	n, err := r.readU32()
	if err != nil {
		return "", err
	}
	p, err := r.take(int(n))
	if err != nil {
		return "", err
	}
	return cstr(p), nil
}
func (r *storeReader) readWide() (string, error) {
	n, err := r.readU32()
	if err != nil {
		return "", err
	}
	p, err := r.take(int(n))
	if err != nil {
		return "", err
	}
	units := make([]uint16, 0, n/2)
	for i := 0; i+1 < len(p); i += 2 {
		u := binary.LittleEndian.Uint16(p[i:])
		if u == 0 {
			break
		}
		units = append(units, u)
	}
	return string(utf16.Decode(units)), nil
}
func (r *storeReader) readValue(t byte) (proto.Value, error) {
	v := proto.Value{Type: t}
	switch t {
	case proto.VarInt32:
		n, err := r.readU32()
		v.I64 = int64(int32(n))
		return v, err
	case proto.VarInt64:
		p, err := r.take(8)
		if err != nil {
			return v, err
		}
		v.I64 = int64(binary.LittleEndian.Uint64(p))
		return v, nil
	case proto.VarFloat:
		n, err := r.readU32()
		v.F64 = float64(math.Float32frombits(n))
		return v, err
	case proto.VarDouble:
		p, err := r.take(8)
		if err != nil {
			return v, err
		}
		v.F64 = math.Float64frombits(binary.LittleEndian.Uint64(p))
		return v, nil
	case proto.VarString:
		s, err := r.readStr()
		v.Text = s
		return v, err
	case proto.VarWide:
		s, err := r.readWide()
		v.Text = s
		return v, err
	}
	return v, fmt.Errorf("不支持的列类型 %d", t)
}

// ---- attr 块 ----

// storeAttr 是属性流条目 {名, 类型化值}。
type storeAttr struct {
	Name  string
	Value proto.Value
}

// encodeAttrBlock 构造 0x41 属性块（uid/name 0x24 定长域；attrs 为
// 呈现性键值——官方数据对象属性由逻辑 DLL 定义，本实现带分区/服号）。
func encodeAttrBlock(uid, name string, attrs []storeAttr) []byte {
	buf := make([]byte, attrBlockSize)
	binary.LittleEndian.PutUint32(buf[0:], pub4Magic)
	copy(buf[attrUIDOff:], uid)
	copy(buf[attrNameOff:], name)
	w := &storeWriter{b: buf}
	for _, a := range attrs {
		w.str(a.Name)
		w.byte(a.Value.Type)
		if err := w.value(a.Value); err != nil {
			panic(err) // 类型已由构造侧保证
		}
	}
	return w.b
}

// decodeAttrBlock 回放属性块（0x29 body 去掉 4 字节 result 头后）。
func decodeAttrBlock(b []byte) (uid, name string, attrs []storeAttr, err error) {
	if len(b) < attrBlockSize {
		return "", "", nil, fmt.Errorf("attr 块太短 %d", len(b))
	}
	if binary.LittleEndian.Uint32(b[0:]) != pub4Magic {
		return "", "", nil, fmt.Errorf("attr 块 magic 不符")
	}
	uid, name = cstr(b[attrUIDOff:attrNameOff]), cstr(b[attrNameOff:attrStreamOff])
	r := &storeReader{b: b, pos: attrStreamOff}
	for r.remain() > 0 {
		key, err := r.readStr()
		if err != nil {
			return "", "", nil, err
		}
		t, err := r.readByte()
		if err != nil {
			return "", "", nil, err
		}
		v, err := r.readValue(t)
		if err != nil {
			return "", "", nil, err
		}
		attrs = append(attrs, storeAttr{Name: key, Value: v})
	}
	return uid, name, attrs, nil
}

// ---- record 块 ----

// encodeRecordBlocks 构造 0x42 记录块序列（头块 + 每 0x100 行一片）。
// 索引数 0（排序视图不持久化，加载后 Rebuild 重建）。
func encodeRecordBlocks(name string, rowMax int, colTypes []byte, rows [][]proto.Value) [][]byte {
	var blocks [][]byte
	for start := 0; start < len(rows) || start == 0; start += recShardRows {
		end := start + recShardRows
		if end > len(rows) {
			end = len(rows)
		}
		shard := rows[start:end]
		head := make([]byte, recHeadSize)
		copy(head[recNameOff:], name)
		binary.LittleEndian.PutUint32(head[recRowMaxOff:], uint32(rowMax))
		binary.LittleEndian.PutUint32(head[recRowsOff:], uint32(len(shard)))
		binary.LittleEndian.PutUint32(head[recColsOff:], uint32(len(colTypes)))
		binary.LittleEndian.PutUint32(head[recIndexCntOff:], 0)
		w := &storeWriter{b: head}
		for _, t := range colTypes {
			w.byte(t)
		}
		// 索引数 0：{u32,u8} 循环与定义循环皆空，WriteInt(0) 收尾
		w.i32(0)
		// 体：start/end + 每行每列 {脏标志 1, value}
		w.i32(int32(start))
		w.i32(int32(end))
		for _, row := range shard {
			for _, v := range row {
				w.byte(1)
				_ = w.value(v)
			}
		}
		blocks = append(blocks, w.b)
		if end >= len(rows) {
			break
		}
	}
	return blocks
}

// decodeRecordBlock 回放一片记录块（0x2A body 去 4 字节 result 头后），
// 返回 (记录名, 起始行, 行值序列)。
func decodeRecordBlock(b []byte) (name string, start int, rows [][]proto.Value, err error) {
	if len(b) < recHeadSize {
		return "", 0, nil, fmt.Errorf("record 块太短 %d", len(b))
	}
	name = cstr(b[recNameOff : recNameOff+0x24])
	rows2 := int(binary.LittleEndian.Uint32(b[recRowsOff:]))
	cols := int(binary.LittleEndian.Uint32(b[recColsOff:]))
	indexCnt := int(binary.LittleEndian.Uint32(b[recIndexCntOff:]))
	r := &storeReader{b: b, pos: recHeadSize}
	colTypes := make([]byte, cols)
	for i := range colTypes {
		if colTypes[i], err = r.readByte(); err != nil {
			return "", 0, nil, err
		}
	}
	for i := 0; i < indexCnt; i++ { // 每索引 {u32, u8}
		if _, err = r.readU32(); err != nil {
			return "", 0, nil, err
		}
		if _, err = r.readByte(); err != nil {
			return "", 0, nil, err
		}
	}
	if _, err = r.readU32(); err != nil { // WriteInt(索引数) 再现
		return "", 0, nil, err
	}
	if start, err = readInt(r); err != nil {
		return "", 0, nil, err
	}
	end, err := readInt(r)
	if err != nil {
		return "", 0, nil, err
	}
	if end-start != rows2 {
		return "", 0, nil, fmt.Errorf("record %s 行数不符：头 %d 片 [%d,%d)", name, rows2, start, end)
	}
	for i := 0; i < rows2; i++ {
		row := make([]proto.Value, cols)
		for c := 0; c < cols; c++ {
			dirty, err := r.readByte()
			if err != nil {
				return "", 0, nil, err
			}
			if dirty == 0 {
				continue // 干脏标志 0：不跟值
			}
			if row[c], err = r.readValue(colTypes[c]); err != nil {
				return "", 0, nil, err
			}
		}
		rows = append(rows, row)
	}
	return name, start, rows, nil
}

func readInt(r *storeReader) (int, error) {
	n, err := r.readU32()
	return int(int32(n)), err
}

// ---- LocalStore（Go 扩展：本地文件 store 垫片）----
//
// 官方持久化完全依赖外部 store 进程（广播发现 + 0x40/0x41/0x42/0x29/
// 0x2A/0x2B 协议）。本垫片以同一套消息语义在进程内完成加载序并把
// PUB4 块原样落盘（dir/main.attr / dir/main.record），保证单进程部署
// 数据不丢；接真 store 时网络路径优先，本垫片仅作双写镜像。

// LocalStore 是本地文件 store。
type LocalStore struct {
	Dir  string
	Logf func(format string, args ...any)
}

// Load 读本地块并按官方加载序回调（0x29 → 0x2A* → 0x2B 的块体等价；
// result 头由消息面承担，此处直接给块）。文件缺失视为空库。
func (ls *LocalStore) Load(onAttr func(block []byte), onRecord func(block []byte), onOK func()) {
	attrPath := filepath.Join(ls.Dir, "main.attr")
	recPath := filepath.Join(ls.Dir, "main.record")
	if data, err := os.ReadFile(attrPath); err == nil {
		onAttr(data)
	}
	if data, err := os.ReadFile(recPath); err == nil {
		onRecord(data)
	}
	onOK()
	ls.Logf("local store：已回放 %s / %s", attrPath, recPath)
}

// SaveAttr 落属性块。
func (ls *LocalStore) SaveAttr(block []byte) {
	if err := os.MkdirAll(ls.Dir, 0o755); err != nil {
		ls.Logf("local store：建目录失败: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(ls.Dir, "main.attr"), block, 0o644); err != nil {
		ls.Logf("local store：写 attr 失败: %v", err)
	}
}

// SaveRecord 落记录块（整文件覆写——单记录场景等价官方分片追加）。
func (ls *LocalStore) SaveRecord(block []byte) {
	if err := os.MkdirAll(ls.Dir, 0o755); err != nil {
		ls.Logf("local store：建目录失败: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(ls.Dir, "main.record"), block, 0o644); err != nil {
		ls.Logf("local store：写 record 失败: %v", err)
	}
}
