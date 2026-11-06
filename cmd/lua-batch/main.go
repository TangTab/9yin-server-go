package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/local/9yin-go-server/internal/luacodec"
)

type result struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Error string `json:"error,omitempty"`
}

type manifest struct {
	GeneratedAt string   `json:"generated_at"`
	InputRoot   string   `json:"input_root"`
	OutputRoot  string   `json:"output_root"`
	Total       int      `json:"total"`
	Succeeded   int64    `json:"succeeded"`
	Failed      int64    `json:"failed"`
	Files       []result `json:"files"`
}

func main() {
	inRoot := flag.String("in-root", "", "root containing extracted encrypted .lua chunks")
	outRoot := flag.String("out-root", "", "root for readable decompiled Lua")
	fxgame := flag.String("fxgame", "", "matching fxgame.exe")
	unluacCP := flag.String("unluac-cp", "", "compiled unluac classpath; omit to only decode bytecode")
	workers := flag.Int("workers", max(2, runtime.NumCPU()/2), "parallel workers")
	flag.Parse()
	if *inRoot == "" || *outRoot == "" || *fxgame == "" {
		flag.Usage()
		os.Exit(2)
	}
	key, err := extractKey(*fxgame)
	if err != nil {
		fatal(err)
	}
	var files []string
	err = filepath.WalkDir(*inRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".lua") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fatal(err)
	}
	sort.Strings(files)
	if err := os.MkdirAll(*outRoot, 0755); err != nil {
		fatal(err)
	}
	jobs := make(chan string)
	results := make(chan result, len(files))
	var succeeded, failed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				r := process(path, *inRoot, *outRoot, *unluacCP, key)
				if r.Error == "" {
					succeeded.Add(1)
				} else {
					failed.Add(1)
				}
				results <- r
			}
		}()
	}
	go func() {
		for _, path := range files {
			jobs <- path
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	all := make([]result, 0, len(files))
	for r := range results {
		all = append(all, r)
		if (succeeded.Load()+failed.Load())%100 == 0 {
			fmt.Printf("processed %d/%d (ok=%d failed=%d)\n", succeeded.Load()+failed.Load(), len(files), succeeded.Load(), failed.Load())
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	m := manifest{time.Now().Format(time.RFC3339), *inRoot, *outRoot, len(files), succeeded.Load(), failed.Load(), all}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(*outRoot, "manifest.json"), b, 0644); err != nil {
		fatal(err)
	}
	fmt.Printf("complete: total=%d ok=%d failed=%d output=%s\n", len(files), succeeded.Load(), failed.Load(), *outRoot)
}

func process(path, inRoot, outRoot, unluacCP string, key []byte) result {
	rel, _ := filepath.Rel(inRoot, path)
	info, _ := os.Stat(path)
	r := result{Path: filepath.ToSlash(rel)}
	out := filepath.Join(outRoot, rel)
	if info, err := os.Stat(out); err == nil && info.Size() > 0 {
		r.Size = info.Size()
		return r
	}
	if info != nil {
		r.Size = info.Size()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	b, err = luacodec.TransformWithKey(b, key)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		r.Error = err.Error()
		return r
	}
	if unluacCP == "" {
		if err := os.WriteFile(out+"c", b, 0644); err != nil {
			r.Error = err.Error()
		}
		return r
	}
	tmp, err := os.CreateTemp("", "9yin-*.luac")
	if err != nil {
		r.Error = err.Error()
		return r
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		r.Error = err.Error()
		return r
	}
	tmp.Close()
	cmd := exec.Command("java", "-cp", unluacCP, "unluac.Main", tmpName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	decoded, err := cmd.Output()
	if err != nil {
		r.Error = strings.TrimSpace(fmt.Sprintf("%v: %s", err, stderr.String()))
		_ = os.WriteFile(out+".luac", b, 0644)
		return r
	}
	if err := os.WriteFile(out, decoded, 0644); err != nil {
		r.Error = err.Error()
	}
	return r
}

func extractKey(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	start := bytes.Index(b, []byte("abcd464f"))
	if start < 0 {
		return nil, fmt.Errorf("Lua key signature not found in %s", path)
	}
	end := bytes.IndexByte(b[start:], 0)
	if end < 0 {
		return nil, fmt.Errorf("unterminated Lua key")
	}
	return append([]byte(nil), b[start:start+end]...), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "lua-batch:", err)
	os.Exit(1)
}
