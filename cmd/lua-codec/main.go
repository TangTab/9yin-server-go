package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/local/9yin-go-server/internal/luacodec"
)

func main() {
	in := flag.String("in", "", "input Lua 5.1 binary chunk")
	source := flag.String("source", "", "plain Lua source to compile and encode")
	luac := flag.String("luac", "", "official Lua 5.1 luac executable (required with -source)")
	out := flag.String("out", "", "output path")
	keyFile := flag.String("key", "", "raw key file (optional when -fxgame is used)")
	fxgame := flag.String("fxgame", "", "matching fxgame.exe; extracts its embedded Lua key")
	flag.Parse()
	if (*in == "") == (*source == "") || *out == "" || (*source != "" && *luac == "") {
		flag.Usage()
		os.Exit(2)
	}
	input := *in
	cleanup := func() {}
	if *source != "" {
		var err error
		input, cleanup, err = compile(*luac, *source)
		if err != nil {
			fatal(err)
		}
		defer cleanup()
	}
	b, err := os.ReadFile(input)
	if err != nil {
		fatal(err)
	}
	key, err := loadKey(*keyFile, *fxgame)
	if err != nil {
		fatal(err)
	}
	b, err = luacodec.TransformWithKey(b, key)
	if err != nil {
		fatal(err)
	}
	if err = os.WriteFile(*out, b, 0644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d-byte game chunk: %s -> %s\n", len(b), map[bool]string{true: *source, false: *in}[*source != ""], *out)
}

func compile(luac, source string) (string, func(), error) {
	luac, err := filepath.Abs(luac)
	if err != nil {
		return "", func() {}, err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return "", func() {}, err
	}
	f, err := os.CreateTemp("", "9yin-*.luac")
	if err != nil {
		return "", func() {}, err
	}
	name := f.Name()
	f.Close()
	cleanup := func() { _ = os.Remove(name) }
	cmd := exec.Command(luac, "-o", name, source)
	cmd.Dir = filepath.Dir(source)
	if output, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("luac failed: %w: %s", err, output)
	}
	return name, cleanup, nil
}

func loadKey(keyFile, fxgame string) ([]byte, error) {
	if keyFile != "" {
		return os.ReadFile(keyFile)
	}
	if fxgame == "" {
		return []byte("snailgame"), nil
	}
	b, err := os.ReadFile(fxgame)
	if err != nil {
		return nil, err
	}
	start := bytes.Index(b, []byte("abcd464f"))
	if start < 0 {
		return nil, fmt.Errorf("Lua key signature not found in %s", fxgame)
	}
	end := bytes.IndexByte(b[start:], 0)
	if end < 0 {
		return nil, fmt.Errorf("unterminated Lua key in %s", fxgame)
	}
	key := append([]byte(nil), b[start:start+end]...)
	if len(key) > 1024 {
		return nil, fmt.Errorf("implausible Lua key length %d", len(key))
	}
	fmt.Printf("extracted %d-byte Lua key from %s\n", len(key), fxgame)
	return key, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "lua-codec:", err)
	os.Exit(1)
}
