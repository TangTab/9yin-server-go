package ecs

// EntityID 是全部跨系统传递的对象句柄：低 32 位 index、高 32 位 generation。
// generation 防悬挂：实体销毁后 index 复用，旧句柄因 gen 不匹配而失效。
// NullEntity == 0（index 0 / gen 0）永不签发。
type EntityID uint64

const NullEntity EntityID = 0

const (
	entityIndexBits = 32
	entityIndexMask = 1<<entityIndexBits - 1 // 无类型常量，可直接与 EntityID 运算
	entityNullGen   = 0                      // 槽位未使用过的 gen 值；首次签发从 1 起
)

// Index 返回实体槽位下标。
func (e EntityID) Index() uint32 { return uint32(e & entityIndexMask) }

// Gen 返回实体代数。
func (e EntityID) Gen() uint32 { return uint32(e >> entityIndexBits) }

// IsNull 报告是否为空句柄。
func (e EntityID) IsNull() bool { return e == NullEntity }

func makeEntity(index, gen uint32) EntityID {
	return EntityID(uint64(gen)<<entityIndexBits | uint64(index)&entityIndexMask)
}
