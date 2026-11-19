// Init 装载协议(动态验证定案,docs/official/GPKitSvr全解与Go复刻.md §4):
//
// 官方 slot1(codeDir="DynCodeBin\Server", dataDir="DynCodeBin\Client"):
// FindFirstFileA(codeDir\*.bin) 枚举代码文件;数据文件 =
// codeDir + basename(dataDir) + "\" + basename(代码文件),
// 即 DynCodeBin\Server\Client\X.bin(x64 2013 版实现为 dataDir\X.bin,
// 复刻两种都接受,优先嵌套规则)。
package gpkit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Init 复刻 slot1:装载 codeDir 下的动态码组,返回成功加载条数。
// 返回 -1 对应官方失败路径(目录无效)。
func (s *Svr) Init(codeDir, dataDir string) int {
	s.mu.Lock()
	loader := s.loader
	s.mu.Unlock()
	if loader == nil {
		loader = DefaultPayloadLoader
	}

	st, err := os.Stat(codeDir)
	if err != nil || !st.IsDir() {
		return -1
	}
	bins, err := filepath.Glob(filepath.Join(codeDir, "*.bin"))
	if err != nil {
		return -1
	}

	m := &MainDynCode{}
	dataSub := filepath.Join(codeDir, filepath.Base(strings.TrimRight(dataDir, `\/`)))
	for _, codeFile := range bins {
		code, err := readFileLimited(codeFile, maxBinSize)
		if err != nil {
			continue
		}
		dataPath := filepath.Join(dataSub, filepath.Base(codeFile))
		if _, err := os.Stat(dataPath); err != nil {
			// x64 2013 风格:dataDir\同名.bin
			dataPath = filepath.Join(dataDir, filepath.Base(codeFile))
		}
		// 官方 x86 行为:数据文件 fopen 失败即放弃整个条目。
		data, err := readFileLimited(dataPath, maxBinSize)
		if err != nil {
			continue
		}
		fn1, fn2, err := loader(code)
		if err != nil {
			continue
		}
		if len(m.entries) >= maxEntries {
			break
		}
		name, _ := filepath.Abs(codeFile)
		m.entries = append(m.entries, &CodeEntry{
			Name: name,
			Code: code,
			Data: data,
			Fn1:  fn1,
			Fn2:  fn2,
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// 官方:Clear 旧码组后挂新码组,计数置 1,flag 翻转为 0。
	s.groups = append(s.groups[:0], m)
	s.groupN = 1
	s.cur = m
	s.inited = false
	return len(m.entries)
}

// GroupCount 返回码组数(官方 +0x3F0)。
func (s *Svr) GroupCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.groupN
}

// Count 返回当前码组条目数(官方 CMainDynCode+0x404)。
func (s *Svr) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.active()
	if m == nil {
		return 0
	}
	return m.count()
}

func readFileLimited(path string, limit int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, limit)
	n, err := f.Read(buf)
	if err != nil && n == 0 && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buf[:n], nil
}

// DefaultPayloadLoader 解析本包定义的 Go 载荷签名:
//
//	"GPK1" | u16 genId | u16 checkId   (小端,共 8 字节)
//
// genId/checkId ∈ [0,70) 映射内置 70 对检查(表 A 生成器/表 B 校验器);
// fn1 = 生成器编码,fn2 = 校验器。非法载荷返回错误(官方为执行失败跳过)。
func DefaultPayloadLoader(code []byte) (CheckFunc, CheckFunc, error) {
	if len(code) < 8 || string(code[:4]) != "GPK1" {
		return nil, nil, errors.New("gpkit: bad payload signature")
	}
	gen := uint16(code[4]) | uint16(code[5])<<8
	chk := uint16(code[6]) | uint16(code[7])<<8
	if gen >= 70 || chk >= 70 {
		return nil, nil, fmt.Errorf("gpkit: check id out of range (gen=%d chk=%d)", gen, chk)
	}
	g := builtinCheck(int(gen))
	c := builtinCheck(int(chk))
	return g, c, nil
}
