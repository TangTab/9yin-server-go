// Package world 独立于线上编码拥有服务端的场景实体。
// 协议处理器负责把这些类型化的值转换为客户端档案。
package world

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

type EntityID uint32

type EntityKind uint8

const (
	EntityPlayer EntityKind = iota + 1
	EntityNPC
	EntityObject
)

type Transform struct {
	X, Y, Z, Orient float32
}

type ValueKind uint8

const (
	ValueByte       ValueKind = 1
	ValueWord       ValueKind = 2
	ValueInt32      ValueKind = 3
	ValueFloat32    ValueKind = 5
	ValueString     ValueKind = 7
	ValueWideString ValueKind = 8
)

type Value struct {
	Kind   ValueKind
	Byte   byte
	Word   uint16
	Int32  int32
	Float  float32
	String string
}

func Byte(value byte) Value         { return Value{Kind: ValueByte, Byte: value} }
func Word(value uint16) Value       { return Value{Kind: ValueWord, Word: value} }
func Int32(value int32) Value       { return Value{Kind: ValueInt32, Int32: value} }
func Float32(value float32) Value   { return Value{Kind: ValueFloat32, Float: value} }
func String(value string) Value     { return Value{Kind: ValueString, String: value} }
func WideString(value string) Value { return Value{Kind: ValueWideString, String: value} }
func finite(value float32) bool     { return !float32NaN(value) && !float32Inf(value) }
func float32NaN(value float32) bool { return math.IsNaN(float64(value)) }
func float32Inf(value float32) bool { return math.IsInf(float64(value), 0) }
func (v Value) validate() error {
	switch v.Kind {
	case ValueByte, ValueWord, ValueInt32, ValueString, ValueWideString:
		return nil
	case ValueFloat32:
		if finite(v.Float) {
			return nil
		}
		return errors.New("non-finite float property")
	default:
		return fmt.Errorf("unsupported property kind %d", v.Kind)
	}
}

// PropertyArchive 以稳定的客户端属性名作为键。数字形式的可见
// 索引属于协商出的线上 schema，有意不作为
// world 模型的一部分。
type PropertyArchive map[string]Value

func (archive PropertyArchive) Set(name string, value Value) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("empty property name")
	}
	if err := value.validate(); err != nil {
		return fmt.Errorf("property %s: %w", name, err)
	}
	archive[name] = value
	return nil
}

func (archive PropertyArchive) Clone() PropertyArchive {
	if archive == nil {
		return nil
	}
	copy := make(PropertyArchive, len(archive))
	for name, value := range archive {
		copy[name] = value
	}
	return copy
}

func (archive PropertyArchive) Names() []string {
	names := make([]string, 0, len(archive))
	for name := range archive {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type Entity struct {
	ID         EntityID
	OwnerID    EntityID
	Kind       EntityKind
	Transform  Transform
	Properties PropertyArchive
}

func (entity Entity) Validate() error {
	if entity.ID == 0 {
		return errors.New("entity ID 0 is invalid")
	}
	if entity.Kind < EntityPlayer || entity.Kind > EntityObject {
		return fmt.Errorf("invalid entity kind %d", entity.Kind)
	}
	if !finite(entity.Transform.X) || !finite(entity.Transform.Y) || !finite(entity.Transform.Z) || !finite(entity.Transform.Orient) {
		return errors.New("entity transform contains a non-finite value")
	}
	for name, value := range entity.Properties {
		if strings.TrimSpace(name) == "" {
			return errors.New("entity contains an empty property name")
		}
		if err := value.validate(); err != nil {
			return fmt.Errorf("property %s: %w", name, err)
		}
	}
	return nil
}

func (entity Entity) Clone() Entity {
	entity.Properties = entity.Properties.Clone()
	return entity
}
