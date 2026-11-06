package luacodec

import (
	"bytes"
	"os"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	chunk, err := os.ReadFile(`../../artifacts/local/define.standard.luac`)
	if err != nil {
		t.Skip(err)
	}
	key, err := os.ReadFile(`../../artifacts/local/fxgame-lua.key`)
	if err != nil {
		t.Skip(err)
	}
	game, err := TransformWithKey(chunk, key)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := TransformWithKey(game, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, chunk) {
		t.Fatal("round trip changed the Lua chunk")
	}
}
