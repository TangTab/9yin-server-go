package fscomm

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogPool 复刻官方 LogPool（logpool.cpp）：快速日志文件池。
//
// 两种落盘形态（官方属性直译）：
//   - FileName 固定文件：所有行追加写同一文件
//   - FilePrefix + EveryDayAlone：按日独立文件，命名
//     "%s%04d%02d%02d%02d%02d%02d.log"（前缀+年月日时分秒.log，
//     即当日首写时刻的时间戳），跨日自动切换
//
// 行格式 "[%04d-%02d-%02d %02d:%02d:%02d] %s"（时间戳 + 内容）。
type LogPool struct {
	// FileName 固定文件名（相对 dir 或绝对路径）。
	FileName string
	// FilePrefix 按日文件前缀（与 FileName 二选一，均设以 FileName 优先）。
	FilePrefix string
	// EveryDayAlone 官方同名属性：true=每天独立文件。
	EveryDayAlone bool

	dir string // 落盘目录

	mu    sync.Mutex
	f     *os.File
	fday  string // 当前文件所属日（EveryDayAlone 滚动判据）
	count int64
}

// NewLogPool 创建；dir 为落盘目录（空=当前目录）。
func NewLogPool(dir string) *LogPool { return &LogPool{dir: dir} }

// WriteLine 写一行（自动补 '\n'；加时间戳前缀对齐官方行格式）。
func (p *LogPool) WriteLine(s string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, err := p.file()
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = fmt.Fprintf(f, "[%04d-%02d-%02d %02d:%02d:%02d] %s\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), s)
	if err == nil {
		p.count++
	}
	return err
}

// file 取当前目标文件；EveryDayAlone 且跨日时滚动
// （官方 Start 时按当日时刻建 "%s%04d%02d%02d%02d%02d%02d.log"）。
func (p *LogPool) file() (*os.File, error) {
	now := time.Now()
	if p.FileName != "" {
		if p.f != nil {
			return p.f, nil
		}
		f, err := os.OpenFile(p.resolve(p.FileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		p.f = f
		return f, nil
	}
	day := now.Format("20060102")
	if p.f != nil && p.fday == day {
		return p.f, nil
	}
	if p.f != nil {
		p.f.Close()
		p.f = nil
	}
	name := fmt.Sprintf("%s%04d%02d%02d%02d%02d%02d.log", p.FilePrefix,
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	f, err := os.OpenFile(p.resolve(name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	p.f, p.fday = f, day
	return f, nil
}

func (p *LogPool) resolve(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(p.dir, name)
}

// LineCount 已写行数（官方 Trace 口径的计数面）。
func (p *LogPool) LineCount() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count
}

// Stop 关闭当前文件（官方 LogPool::Stop）。
func (p *LogPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.f == nil {
		return nil
	}
	err := p.f.Close()
	p.f = nil
	return err
}
