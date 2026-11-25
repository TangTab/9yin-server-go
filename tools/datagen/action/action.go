// Package action 把客户端招式表编译为服务端判定节拍产物。
// tools/datagen 的 action 构建器：输入输出路径由统一入口解析后经 Options
// 传入，本包不做路径探测。
//
// 招式表（客户端资源根 ini/action/，zhaoshi_list.ini [include] 列出各档）是
// 动作分段与触发帧的设计态数据：
//
//	[CS_jh_cqgf01]
//	1=tzcq_10;0;45;-1;-1;1                       段号=动作名;起始帧;帧长;入混合;出混合;标志
//	trigger_custom_1=1;trigger_hitbreak;13;...   段号;触发类型;段内帧;参数
//	trigger_custom_2=1;trigger_hitback;18;0.3,hurt_f_1_s,,0.2
//	trigger_custom_6=1;trigger_action_speed;32; ,2.5,CS_fight_0h_jb_16
//
// 官服的命中判定帧 = hitbreak ∪ hitback（同刻去重），30fps，段内帧时刻按
// action_speed 变速折算（docs/combat/寒心恨雪与太极剑舞引导机制.md；女号
// 官服抓包 207.7~228s 逐拍对拍：cqgf01 判定帧 13/18/27 → 433/600/900ms，
// cqgf04 帧 28 起 ×0.5、32 起 ×2.5、46 复位、判定帧 38 → 1280ms，实测 1277）。
//
// 产物 ini/Action/zhaoshi_hits.xml（一行一招式）：
//
//	<Property ID="zs_normal_00_01" Actions="ptgj_0h_01;ptgj_0h_05"
//	          Frames="15;29" Hits="467;1000" Stagger="0;1" Length="1467"/>
//
// Stagger 与 Hits 逐拍对位：1 = 该判定帧的受击反应由服务端下发（硬直触发器
// 不带受击者动画，客户端无从自播——女号官服抓包 cqgf04/06、任务技能
// musejuben_2/4/5 命中时都有受击帧）；0 = 触发器自带受击者动画（hitbreak 的
// 动画名 / hitback 的动画位 / hitdown / hitout / hitclasp），客户端按招式表自播，
// 官服不发任何受击帧（cqgf01 三拍动画 hurt_f_2_s / hurt_f_1_s / hurt_f_1_l，
// 22 次命中零受击帧；飞镖 hw_normal_fb 同）。
//
// 服务端只读产物（装载器 internal/actiondata）；招式表本体不进数据根。
package action

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/internal/actiondata"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/tools/datagen/genutil"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Options 一次招式节拍构建的输入输出。
type Options struct {
	Client     string // 客户端资源解压根（含 ini/action）
	Out        string // 输出根：产物落 <out>/ini/Action
	ReportPath string // 审计报告 JSON
	Verify     bool   // 生成后用服务端装载器回读校验
}

// frameMS 招式表帧长（30fps）。
const frameMS = 1000.0 / 30

// 判定触发类型（命中结算帧）。hitdown/hitout/hitclasp 是受控判定，同样是
// 一次命中（控制规格书 §三帧族）。
var hitTriggerKinds = map[string]bool{
	"trigger_hitbreak": true, "trigger_hitback": true,
	"trigger_hitdown": true, "trigger_hitout": true, "trigger_hitclasp": true,
}

type speedChange struct {
	frame int
	speed float64
}

// hitMark 一个判定帧：animated = 触发器自带受击者动画（客户端自播受击反应）。
type hitMark struct {
	frame    int
	animated bool
}

type segment struct {
	action string
	frames int
	speeds []speedChange
	hits   []hitMark
}

// entry 一招：各段动作与触发帧。
type entry struct {
	id     string
	segs   []segment
	source string
}

// trigger 一条触发行。isSpeed 区分变速触发与判定触发——变速复位
// （speed=1）也是变速，不得当作判定帧。animated 见 hitMark。
type trigger struct {
	seg      int
	kind     string
	frame    int
	speed    float64
	isSpeed  bool
	animated bool
}

// hitAnimated 判定触发是否自带受击者动画：hitbreak 的参数就是动画名
// （`trigger_hitbreak;13;hurt_f_2_s,`），hitback 是「距离,动画,…」
// （`trigger_hitback;18;0.3,hurt_f_1_s,,`）；hitdown / hitout / hitclasp 是
// 受控判定，恒带。参数位为空（`;N;,` / `;N; ,`）即不带。
func hitAnimated(kind string, fields []string) bool {
	switch kind {
	case "trigger_hitbreak":
		return triggerParam(fields, 3, 0) != ""
	case "trigger_hitback":
		return triggerParam(fields, 3, 1) != ""
	}
	return true
}

// triggerParam 触发行第 i 个分号字段（越界取空）的第 j 个逗号子项，去空白与
// 行尾逗号。
func triggerParam(fields []string, i, j int) string {
	if i >= len(fields) {
		return ""
	}
	parts := strings.Split(fields[i], ",")
	if j >= len(parts) {
		return ""
	}
	return strings.TrimSpace(strings.Trim(parts[j], ", \t"))
}

type report struct {
	Input               string     `json:"input"`
	Out                 string     `json:"out"`
	Files               int        `json:"files"`
	Sections            int        `json:"sections"`
	SectionsWithHit     int        `json:"sections_with_hits"`
	HitTriggers         int        `json:"hit_triggers"`
	SectionsWithStagger int        `json:"sections_with_stagger_hits"`
	StaggerBeats        int        `json:"stagger_beats"`
	Warnings            []string   `json:"warnings,omitempty"`
	Verify              *verifyRpt `json:"verify,omitempty"`
}

type verifyRpt struct {
	Rows     int      `json:"rows"`
	Failures []string `json:"failures,omitempty"`
}

// Run 编译招式节拍：落盘 + 报告 + 控制台摘要。
func Run(opt Options) error {
	dir := filepath.Join(opt.Client, "ini", "action")
	if info, err := res.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Errorf("招式表目录不可用 %s（-client 指定客户端资源解压根）", dir)
	}
	files, warnings := includeFiles(dir)
	if len(files) == 0 {
		return fmt.Errorf("招式表清单缺席或为空 %s（zhaoshi_list.ini）", filepath.Join(dir, "zhaoshi_list.ini"))
	}
	entries, hitTriggers := parse(files)
	outDir := filepath.Join(opt.Out, "ini", "Action")
	out := filepath.Join(outDir, actiondata.FileName)
	if err := genutil.WriteFile(out, render(entries)); err != nil {
		return err
	}
	withHits, withStagger, staggerBeats := 0, 0, 0
	for _, e := range entries {
		_, stagger := e.hitMarks()
		if len(stagger) > 0 {
			withHits++
		}
		n := 0
		for _, on := range stagger {
			if on {
				n++
			}
		}
		staggerBeats += n
		if n > 0 {
			withStagger++
		}
	}
	rep := &report{Input: dir, Out: outDir, Files: len(files), Sections: len(entries),
		SectionsWithHit: withHits, HitTriggers: hitTriggers, Warnings: warnings,
		SectionsWithStagger: withStagger, StaggerBeats: staggerBeats}
	if opt.Verify {
		v := &verifyRpt{}
		if catalog, err := actiondata.LoadDir(outDir); err != nil {
			v.Failures = append(v.Failures, err.Error())
		} else {
			v.Rows = catalog.Count()
			if catalog.Count() != len(entries) {
				v.Failures = append(v.Failures, fmt.Sprintf("回读行数 %d != 期望 %d", catalog.Count(), len(entries)))
			}
			for _, e := range entries {
				got, ok := catalog.Lookup(e.id)
				hits, stagger := e.hitMarks()
				if !ok {
					v.Failures = append(v.Failures, fmt.Sprintf("招式 %s 回读缺席", e.id))
				} else if fmt.Sprint(got.HitsMS) != fmt.Sprint(hits) || fmt.Sprint(got.Frames) != fmt.Sprint(e.frameList()) ||
					fmt.Sprint(got.Stagger) != fmt.Sprint(stagger) {
					v.Failures = append(v.Failures, fmt.Sprintf("招式 %s 回读 %v/%v/%v != 期望 %v/%v/%v",
						e.id, got.HitsMS, got.Frames, got.Stagger, hits, e.frameList(), stagger))
				}
				if len(v.Failures) > 8 {
					break
				}
			}
		}
		rep.Verify = v
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		return err
	}
	blob, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(opt.ReportPath, append(blob, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("action: %d 档 %d 招式（带判定 %d，判定触发 %d；服务端发受击帧的 %d 招 / %d 拍）→ %s\n",
		len(files), len(entries), withHits, hitTriggers, withStagger, staggerBeats, out)
	for _, w := range warnings {
		fmt.Println("  警告:", w)
	}
	fmt.Printf("action: 报告 %s\n", opt.ReportPath)
	if rep.Verify != nil {
		for _, failure := range rep.Verify.Failures {
			fmt.Fprintln(os.Stderr, "action: verify 失败:", failure)
		}
		if len(rep.Verify.Failures) > 0 {
			return fmt.Errorf("action: verify 失败 %d 项", len(rep.Verify.Failures))
		}
		fmt.Printf("action: verify 通过（%d 行经 actiondata 回读一致）\n", rep.Verify.Rows)
	}
	return nil
}

// includeFiles zhaoshi_list.ini [include] 的 fileN 清单（值相对客户端资源
// 根，如 ini\action\zhaoshi_player.ini）；缺清单时退回目录下全部 zhaoshi_*.ini。
func includeFiles(dir string) (files []string, warnings []string) {
	client := filepath.Dir(filepath.Dir(dir))
	if raw, err := res.ReadFile(filepath.Join(dir, "zhaoshi_list.ini")); err == nil {
		for _, line := range strings.Split(string(toUTF8(raw)), "\n") {
			line = strings.TrimSpace(line)
			if len(line) < 5 || strings.ToLower(line[:4]) != "file" {
				continue
			}
			_, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			path := filepath.Join(client, filepath.FromSlash(strings.ReplaceAll(value, "\\", "/")))
			if _, err := os.Stat(path); err == nil {
				files = append(files, path)
			} else {
				warnings = append(warnings, fmt.Sprintf("清单项缺席 %s", value))
			}
		}
	}
	if len(files) == 0 {
		files, _ = filepath.Glob(filepath.Join(dir, "zhaoshi_*.ini"))
	}
	return files, warnings
}

// parse 各档招式表 → 招式清单序（同名招式首见胜出）。
func parse(files []string) (entries []*entry, hitTriggers int) {
	index := map[string]*entry{}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cur *entry
		var triggers []trigger
		flush := func() {
			if cur != nil {
				applyTriggers(cur, triggers)
			}
			triggers = nil
		}
		for _, line := range strings.Split(string(toUTF8(raw)), "\n") {
			line = strings.TrimSpace(line)
			switch {
			case line == "":
			case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
				flush()
				name := strings.TrimSpace(line[1 : len(line)-1])
				if _, dup := index[name]; dup {
					cur = nil
					continue
				}
				cur = &entry{id: name, source: filepath.Base(path)}
				index[name] = cur
				entries = append(entries, cur)
			case cur == nil:
			case strings.HasPrefix(line, "trigger_custom_"):
				_, value, _ := strings.Cut(line, "=")
				fields := strings.Split(value, ";")
				if len(fields) < 3 {
					continue
				}
				t := trigger{seg: atoi(fields[0]), kind: strings.ToLower(strings.TrimSpace(fields[1])), frame: atoi(fields[2]), speed: 1}
				if hitTriggerKinds[t.kind] {
					t.animated = hitAnimated(t.kind, fields)
					hitTriggers++
					triggers = append(triggers, t)
				} else if t.kind == "trigger_action_speed" {
					if len(fields) > 3 {
						for _, part := range strings.Split(fields[3], ",") {
							if v, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil && v > 0 {
								t.speed = v
								break
							}
						}
					}
					t.isSpeed = true
					triggers = append(triggers, t)
				}
			case line[0] >= '0' && line[0] <= '9':
				key, value, ok := strings.Cut(line, "=")
				if _, err := strconv.Atoi(key); err != nil || !ok {
					continue
				}
				fields := strings.Split(value, ";")
				seg := segment{action: strings.TrimSpace(fields[0]), frames: -1}
				if len(fields) > 2 {
					seg.frames = atoi(fields[2])
				}
				cur.segs = append(cur.segs, seg)
			}
		}
		flush()
	}
	return entries, hitTriggers
}

// applyTriggers 把触发帧落到各段（段号越界或帧越界的忽略）。
func applyTriggers(e *entry, triggers []trigger) {
	for _, t := range triggers {
		if t.seg < 1 || t.seg > len(e.segs) {
			continue
		}
		seg := &e.segs[t.seg-1]
		if t.isSpeed {
			seg.speeds = append(seg.speeds, speedChange{t.frame, t.speed})
			continue
		}
		if seg.frames >= 0 && t.frame > seg.frames {
			continue
		}
		seg.hits = append(seg.hits, hitMark{frame: t.frame, animated: t.animated})
	}
	for i := range e.segs {
		s := &e.segs[i]
		sort.Slice(s.speeds, func(a, b int) bool { return s.speeds[a].frame < s.speeds[b].frame })
	}
}

// elapsed 段内第 frame 帧起点的时刻（毫秒；按 action_speed 变速折算）。
func (s *segment) elapsed(frame int) int {
	if frame < 0 {
		frame = 0
	}
	speed, at, ms := 1.0, 0, 0.0
	for _, change := range s.speeds {
		if change.frame > frame {
			break
		}
		ms += float64(change.frame-at) * frameMS / speed
		at, speed = change.frame, change.speed
	}
	ms += float64(frame-at) * frameMS / speed
	return int(ms + 0.5)
}

// length 段时长（毫秒；帧长缺省按 0）。
func (s *segment) length() int {
	if s.frames <= 0 {
		return 0
	}
	return s.elapsed(s.frames)
}

// frameList 各段帧长。
func (e *entry) frameList() []int {
	out := make([]int, len(e.segs))
	for i, seg := range e.segs {
		out[i] = seg.frames
	}
	return out
}

// hitMarks 判定时刻（毫秒，升序去重）与逐拍 Stagger：该刻的全部判定触发都
// 不带受击者动画 = 受击反应由服务端下发。
func (e *entry) hitMarks() (hits []int, stagger []bool) {
	start := 0
	animated := map[int]bool{}
	for _, seg := range e.segs {
		for _, mark := range seg.hits {
			ms := start + seg.elapsed(mark.frame)
			if _, seen := animated[ms]; !seen {
				hits = append(hits, ms)
			}
			animated[ms] = animated[ms] || mark.animated
		}
		start += seg.length()
	}
	sort.Ints(hits)
	for _, ms := range hits {
		stagger = append(stagger, !animated[ms])
	}
	return hits, stagger
}

// hitTimes 判定时刻（毫秒，升序去重）。
func (e *entry) hitTimes() []int {
	hits, _ := e.hitMarks()
	return hits
}

func (e *entry) length() int {
	total := 0
	for _, seg := range e.segs {
		total += seg.length()
	}
	return total
}

func render(entries []*entry) []byte {
	var sb strings.Builder
	sb.WriteString(xmlHeader())
	sb.WriteString("<Object>\n")
	for _, e := range entries {
		actions := make([]string, len(e.segs))
		for i, seg := range e.segs {
			actions[i] = seg.action
		}
		hits, stagger := e.hitMarks()
		fmt.Fprintf(&sb, "\t<Property ID=%q Actions=%q Frames=%q Hits=%q Stagger=%q Length=\"%d\"/>\n",
			e.id, strings.Join(actions, ";"), joinInts(e.frameList()), joinInts(hits), joinFlags(stagger), e.length())
	}
	sb.WriteString("</Object>\n")
	return []byte(sb.String())
}

func xmlHeader() string { return "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" }

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ";")
}

func joinFlags(values []bool) string {
	parts := make([]string, len(values))
	for i, v := range values {
		if v {
			parts[i] = "1"
		} else {
			parts[i] = "0"
		}
	}
	return strings.Join(parts, ";")
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码。
func toUTF8(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	if utf8.Valid(raw) {
		return raw
	}
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		return raw
	}
	return decoded
}
