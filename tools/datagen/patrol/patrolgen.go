// Package patrol 是 tools/datagen 的 patrol 构建器：NPC 巡逻路线（mappath）
// 入数据根。路径由统一入口解析后经 Options 传入，本包不做路径探测。
//
// 输入（Options.In = 客户端资源根下 share/creator/mappath）：
//   - <场景短名>_<类别>.ini  官方巡逻路线表（GBK；节名=摆放 PathId，每行
//     r= 一个路点）。文件名含 offline 的是离线态路线，服务端装载器按
//     文件名排除（实测无摆放引用）。
//   - default.txt            路点列说明（编辑器文档，服务端不读）
//
// 输出：<Out>/<同名>.ini —— .ini 全量原样字节透传（不改编码、不改行尾），
// 与源目录互为镜像。<Out> 是数据根下的 share/creator/mappath（与客户端
// 资源树同路径——服务端装载器 npcfeed.LoadPatrolRoutes 读的就是这里；客户端
// share 整树并入数据根时路线随之到位，本构建器只剩审计与最小投放的用途）。
// 不做格式转换：服务端装载器
// （internal/npcfeed.ParseRouteTable）读的就是这份 r= 行方言，黑金服务端
// 自带的 ini/Scene/creator/MapPath/*.ini 同形；逐字节相同才能在并入资源层
// 时与既有层去重。
//
// 报告（<ReportPath>）：逐档节数 / 行数 / 坏行 / 不成路线节数 / 成路线数
// 与路点数，均出自服务端解析器——构建期审计的就是运行时读到的。
package patrol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/genutil"
)

// Options 一次巡逻路线构建的输入输出。
type Options struct {
	In         string // 客户端资源根下 share/creator/mappath
	Out        string // 输出目录（数据根下的 share/creator/mappath，与客户端资源树同路径）
	ReportPath string // 审计报告 JSON
	Verify     bool   // 写出后逐档回读，与源字节比对
}

// fileReport 单档的解析统计。
type fileReport struct {
	File      string `json:"file"`
	Bytes     int    `json:"bytes"`
	Offline   bool   `json:"offline,omitempty"`
	Sections  int    `json:"sections"`
	Lines     int    `json:"lines"`
	Rejected  int    `json:"rejected"`
	TooShort  int    `json:"too_short"`
	Routes    int    `json:"routes"`
	Waypoints int    `json:"waypoints"`
}

// Report 审计报告。
type Report struct {
	Source   string       `json:"source"`
	Out      string       `json:"out"`
	Files    int          `json:"files"`   // 搬运的 .ini 数（含 offline）
	Changed  int          `json:"changed"` // 写前与既有产物内容不同的档数
	Bytes    int          `json:"bytes"`
	Active   fileStats    `json:"active"`  // 不含 offline 的合计（服务端实际装载面）
	Tables   []fileReport `json:"tables"`  // 字典序
	Skipped  []string     `json:"skipped"` // 非 .ini（编辑器文档等）
	Warnings []string     `json:"warnings,omitempty"`
}

// fileStats 多档合计。
type fileStats struct {
	Files     int `json:"files"`
	Sections  int `json:"sections"`
	Lines     int `json:"lines"`
	Rejected  int `json:"rejected"`
	TooShort  int `json:"too_short"`
	Routes    int `json:"routes"`
	Waypoints int `json:"waypoints"`
}

// Run 把客户端 mappath 路线表原样搬进数据根：落盘 + 回读校验 + 报告 +
// 控制台摘要。
func Run(opt Options) error {
	entries, err := res.ReadDir(opt.In)
	if err != nil {
		return fmt.Errorf("patrol: 输入目录不可用 %s: %w", opt.In, err)
	}
	report := &Report{Source: opt.In, Out: opt.Out}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".ini") {
			report.Skipped = append(report.Skipped, name)
			continue
		}
		data, err := res.ReadFile(filepath.Join(opt.In, name))
		if err != nil {
			return fmt.Errorf("patrol: 读 %s: %w", name, err)
		}
		_, stats := npcfeed.ParseRouteTable(data)
		offline := strings.Contains(strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name))), "offline")
		row := fileReport{
			File: name, Bytes: len(data), Offline: offline,
			Sections: stats.Sections, Lines: stats.Lines, Rejected: stats.Rejected,
			TooShort: stats.TooShort, Routes: stats.Routes, Waypoints: stats.Waypoints,
		}
		report.Tables = append(report.Tables, row)
		report.Files++
		report.Bytes += len(data)
		if !offline {
			report.Active.Files++
			report.Active.Sections += stats.Sections
			report.Active.Lines += stats.Lines
			report.Active.Rejected += stats.Rejected
			report.Active.TooShort += stats.TooShort
			report.Active.Routes += stats.Routes
			report.Active.Waypoints += stats.Waypoints
			// 告警只看装载面：offline 档服务端不读，其坏行只进报告。
			if stats.Rejected > 0 {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %d 行 r= 解析失败（服务端丢弃该路点）", name, stats.Rejected))
			}
		}

		outPath := filepath.Join(opt.Out, name)
		if existing, err := res.ReadFileShared(outPath); err != nil || !bytes.Equal(existing, data) {
			report.Changed++
		}
		if err := genutil.WriteFileVerbatim(outPath, data); err != nil {
			return fmt.Errorf("patrol: 写 %s: %w", outPath, err)
		}
		if opt.Verify {
			written, err := res.ReadFile(outPath)
			if err != nil {
				return fmt.Errorf("patrol: verify 回读 %s: %w", outPath, err)
			}
			if !bytes.Equal(written, data) {
				return fmt.Errorf("patrol: verify %s：回读与源不一致", outPath)
			}
		}
	}
	sort.Slice(report.Tables, func(i, j int) bool { return report.Tables[i].File < report.Tables[j].File })
	sort.Strings(report.Skipped)

	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	blob, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(opt.ReportPath, append(blob, '\n'), 0o644); err != nil {
		return err
	}

	fmt.Printf("patrol: %d 档 %s → %s（其中 offline %d 档；%d 档与既有产物不同）\n",
		report.Files, sizeLabel(report.Bytes), opt.Out, report.Files-report.Active.Files, report.Changed)
	fmt.Printf("patrol: 服务端装载面 %d 档：路线 %d 条 / 路点 %d（坏行 %d，点数不足弃 %d 节）\n",
		report.Active.Files, report.Active.Routes, report.Active.Waypoints, report.Active.Rejected, report.Active.TooShort)
	fmt.Printf("patrol: 报告 %s\n", opt.ReportPath)
	for _, w := range report.Warnings {
		fmt.Println("  警告:", w)
	}
	if opt.Verify {
		fmt.Printf("patrol: verify 通过（%d 档逐字节回读一致）\n", report.Files)
	}
	return nil
}

func sizeLabel(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
