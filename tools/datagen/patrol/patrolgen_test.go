package patrol

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// 真实档形态的源：GBK 字节 + CRLF 行尾；含一个坏行。
var sourceActive = []byte("; \xc4\xe3\xba\xc3\r\n[AI_a]\r\n" +
	"r=,10,1,20,0,0,stand,nil,walk,nil,1,\r\n" +
	"r=,30,1,40,0,0,stand,nil,walk,nil,1,\r\n" +
	"r=,bad,1,20,0,0,stand,nil,walk,nil,1,\r\n")

var sourceOffline = []byte("[AI_off]\r\nr=,1,1,2,0,0,stand,nil,walk,nil,1,\r\nr=,3,1,4,0,0,stand,nil,walk,nil,1,\r\n")

// 造一份客户端 mappath 目录：路线档 + offline 档 + 空档 + 编辑器文档。
func writeSource(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	for name, data := range map[string][]byte{
		"city01_npc.ini":     sourceActive,
		"city01_offline.ini": sourceOffline,
		"adv009_npc.ini":     {},
		"default.txt":        []byte("[NPC]\r\nOrient=-10.0\r\n"),
	} {
		if err := os.WriteFile(filepath.Join(src, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return src
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readReport(t *testing.T, path string) *Report {
	t.Helper()
	var report Report
	if err := json.Unmarshal(readFile(t, path), &report); err != nil {
		t.Fatal(err)
	}
	return &report
}

// .ini 全量原样搬运（含 offline 与空档），非 .ini 跳过；字节与源相同、
// 不改行尾；报告统计出自服务端解析器；产物可被服务端装载器读到。
func TestRunMirrorsIniVerbatim(t *testing.T) {
	src := writeSource(t)
	out := filepath.Join(t.TempDir(), "mappath")
	reportPath := filepath.Join(t.TempDir(), "report", "patrolgen-report.json")

	if err := Run(Options{In: src, Out: out, ReportPath: reportPath, Verify: true}); err != nil {
		t.Fatal(err)
	}
	names, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range names {
		got = append(got, e.Name())
	}
	if want := []string{"adv009_npc.ini", "city01_npc.ini", "city01_offline.ini"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("输出文件 %v，应为 %v（default.txt 不搬）", got, want)
	}
	for name, source := range map[string][]byte{
		"city01_npc.ini":     sourceActive,
		"city01_offline.ini": sourceOffline,
		"adv009_npc.ini":     {},
	} {
		if data := readFile(t, filepath.Join(out, name)); !bytes.Equal(data, source) {
			t.Fatalf("%s 与源不一致: %q", name, data)
		}
	}

	report := readReport(t, reportPath)
	if report.Files != 3 || report.Changed != 3 {
		t.Fatalf("files=%d changed=%d，应为 3/3", report.Files, report.Changed)
	}
	wantActive := fileStats{Files: 2, Sections: 1, Lines: 3, Rejected: 1, Routes: 1, Waypoints: 2}
	if report.Active != wantActive {
		t.Fatalf("active=%+v，应为 %+v（offline 档不计入装载面）", report.Active, wantActive)
	}
	if want := []string{"default.txt"}; !reflect.DeepEqual(report.Skipped, want) {
		t.Fatalf("skipped=%v，应为 %v", report.Skipped, want)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("坏行应告警 1 条，得 %v", report.Warnings)
	}

	// 服务端装载器读产物：offline 档被排除。
	npcfeed.ResetPatrolCache()
	routes := npcfeed.LoadPatrolRoutes(out, "city01_yanjing")
	route := routes["AI_a"]
	if route == nil || len(route.Waypoints) != 2 || route.Waypoints[1].X != 30 {
		t.Fatalf("装载器未读到产物路线: %+v", route)
	}
	if _, exists := routes["AI_off"]; exists {
		t.Fatal("offline 路线不应被装载")
	}
}

// 重建幂等：产物已在且与源相同 → 不算变更；产物被改成别的行尾/内容 →
// 按源字节写回（不跟随现有行尾）。
func TestRunRebuildIsIdempotent(t *testing.T) {
	src := writeSource(t)
	out := filepath.Join(t.TempDir(), "mappath")
	reportPath := filepath.Join(t.TempDir(), "patrolgen-report.json")
	opt := Options{In: src, Out: out, ReportPath: reportPath, Verify: true}

	if err := Run(opt); err != nil {
		t.Fatal(err)
	}
	if err := Run(opt); err != nil {
		t.Fatal(err)
	}
	if report := readReport(t, reportPath); report.Changed != 0 {
		t.Fatalf("二次重建 changed=%d，应为 0", report.Changed)
	}

	target := filepath.Join(out, "city01_npc.ini")
	if err := os.WriteFile(target, bytes.ReplaceAll(sourceActive, []byte("\r\n"), []byte("\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(opt); err != nil {
		t.Fatal(err)
	}
	if report := readReport(t, reportPath); report.Changed != 1 {
		t.Fatalf("被改动的产物应计 1 档变更，得 %d", report.Changed)
	}
	if data := readFile(t, target); !bytes.Equal(data, sourceActive) {
		t.Fatalf("应按源字节写回（CRLF），得 %q", data)
	}
}

// 输入目录缺席：报错并带上路径（-client 指向）。
func TestRunMissingInput(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	err := Run(Options{In: missing, Out: filepath.Join(t.TempDir(), "mappath"),
		ReportPath: filepath.Join(t.TempDir(), "r.json"), Verify: true})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte(missing)) {
		t.Fatalf("应报输入目录不可用并带路径，得 %v", err)
	}
}
