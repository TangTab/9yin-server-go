package luacodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var defaultKey = []byte("snailgame")

// Transform converts between a standard Lua 5.1 chunk and the FxCore game
// representation. The operation is symmetric: FxCore XORs every LoadBlock
// independently, restarting the key for each block, while leaving the header
// untouched.
func Transform(src []byte) ([]byte, error) {
	return TransformWithKey(src, defaultKey)
}

// TransformWithKey converts a chunk using the runtime key embedded in the
// matching fxgame.exe build.
func TransformWithKey(src, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("empty key")
	}
	if len(src) < 12 || !bytes.Equal(src[:4], []byte{0x1b, 'L', 'u', 'a'}) || src[4] != 0x51 {
		return nil, fmt.Errorf("not a Lua 5.1 binary chunk")
	}
	if src[6] != 1 || src[7] != 4 || src[8] != 8 || src[9] != 4 || src[10] != 8 {
		return nil, fmt.Errorf("unsupported chunk layout: endian=%d int=%d size_t=%d instruction=%d number=%d", src[6], src[7], src[8], src[9], src[10])
	}
	// The first post-header field is a size_t source-string length. Use it to
	// distinguish standard input from game input so parsing uses plain values.
	rawSize := binary.LittleEndian.Uint64(src[12:20])
	x := append([]byte(nil), src[12:20]...)
	for i := range x {
		x[i] ^= key[i%len(key)]
	}
	xorSize := binary.LittleEndian.Uint64(x)
	remaining := uint64(len(src) - 20)
	inputEncoded := rawSize > remaining+1 && xorSize <= remaining+1
	if rawSize > remaining+1 && xorSize > remaining+1 {
		return nil, fmt.Errorf("cannot determine chunk encoding (source sizes %d / %d)", rawSize, xorSize)
	}
	c := &codec{src: src, out: append([]byte(nil), src[:12]...), off: 12, inputEncoded: inputEncoded, key: key}
	if err := c.function(); err != nil {
		return nil, err
	}
	if c.off != len(src) {
		return nil, fmt.Errorf("trailing data at offset %d (%d bytes)", c.off, len(src)-c.off)
	}
	return c.out, nil
}

type codec struct {
	src          []byte
	out          []byte
	off          int
	inputEncoded bool
	key          []byte
}

// block toggles one FxCore LoadBlock and returns its decoded bytes. The input
// may be either standard or game form; callers determine direction by whether
// parsed values are sensible.
func (c *codec) block(n int) ([]byte, error) {
	if n < 0 || c.off+n > len(c.src) {
		return nil, io.ErrUnexpectedEOF
	}
	raw := c.src[c.off : c.off+n]
	decoded := make([]byte, n)
	toggled := make([]byte, n)
	for i, b := range raw {
		toggled[i] = b ^ c.key[i%len(c.key)]
		if c.inputEncoded {
			decoded[i] = toggled[i]
		} else {
			decoded[i] = b
		}
	}
	c.out = append(c.out, toggled...)
	c.off += n
	return decoded, nil
}

func (c *codec) u8() (byte, error) {
	b, err := c.block(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (c *codec) u32() (uint32, error) {
	b, err := c.block(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (c *codec) u64() (uint64, error) {
	b, err := c.block(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (c *codec) count(itemSize uint64) (uint32, error) {
	n, err := c.u32()
	if err != nil {
		return 0, err
	}
	if uint64(n)*itemSize > uint64(len(c.src)-c.off) {
		return 0, fmt.Errorf("invalid count %d at offset %d", n, c.off-4)
	}
	return n, nil
}

func (c *codec) str() error {
	n, err := c.u64()
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	if n > uint64(len(c.src)-c.off) {
		return fmt.Errorf("invalid string size %d at offset %d", n, c.off-8)
	}
	_, err = c.block(int(n))
	return err
}

func (c *codec) function() error {
	if err := c.str(); err != nil {
		return err
	}
	if _, err := c.u32(); err != nil {
		return err
	} // line defined
	if _, err := c.u32(); err != nil {
		return err
	} // last line defined
	for i := 0; i < 4; i++ { // nups, params, vararg, max stack
		if _, err := c.u8(); err != nil {
			return err
		}
	}

	n, err := c.count(4)
	if err != nil {
		return err
	}
	if _, err = c.block(int(n) * 4); err != nil {
		return err
	}

	n, err = c.count(1)
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		t, e := c.u8()
		if e != nil {
			return e
		}
		switch t {
		case 0:
		case 1:
			if _, e = c.u8(); e != nil {
				return e
			}
		case 3:
			if _, e = c.block(8); e != nil {
				return e
			}
		case 4:
			if e = c.str(); e != nil {
				return e
			}
		default:
			return fmt.Errorf("invalid constant type %d at offset %d", t, c.off-1)
		}
	}

	n, err = c.count(1)
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		if err = c.function(); err != nil {
			return fmt.Errorf("prototype %d: %w", i, err)
		}
	}

	n, err = c.count(4)
	if err != nil {
		return err
	}
	if _, err = c.block(int(n) * 4); err != nil {
		return err
	}

	n, err = c.count(1)
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		if err = c.str(); err != nil {
			return err
		}
		if _, err = c.u32(); err != nil {
			return err
		}
		if _, err = c.u32(); err != nil {
			return err
		}
	}

	n, err = c.count(1)
	if err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		if err = c.str(); err != nil {
			return err
		}
	}
	return nil
}
