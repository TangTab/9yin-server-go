package logline

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
)

var lineRe = regexp.MustCompile(`^(?:\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} (\S+) ?(.*))?$`)

// line 一行解析结果：server 空＝空行。
type line struct{ server, msg string }

// withSink 装一个目的地为内存缓冲的转发，换掉 os.Stdout/os.Stderr 与
// 标准库 log；Flush 后再读缓冲（排空完无并发写）。
func withSink(t *testing.T, server string) (outBuf, errBuf *bytes.Buffer) {
	t.Helper()
	var ob, eb bytes.Buffer
	s, err := newSink(server, &ob, &eb)
	if err != nil {
		t.Fatalf("newSink: %v", err)
	}
	oldOut, oldErr, oldFlags := os.Stdout, os.Stderr, log.Flags()
	attach(s)
	go s.out.run()
	go s.err.run()
	t.Cleanup(func() {
		Flush()
		os.Stdout, os.Stderr = oldOut, oldErr
		log.SetFlags(oldFlags)
		log.SetOutput(os.Stderr)
		cur.Store(nil)
	})
	outBuf, errBuf = &ob, &eb
	return outBuf, errBuf
}

func parse(t *testing.T, buf *bytes.Buffer) []line {
	t.Helper()
	data := strings.TrimSuffix(buf.String(), "\n")
	if data == "" {
		return nil
	}
	var out []line
	for _, raw := range strings.Split(data, "\n") {
		if raw == "\r" { // CRLF 空行原样
			out = append(out, line{})
			continue
		}
		m := lineRe.FindStringSubmatch(raw)
		if m == nil {
			t.Fatalf("行不合前缀格式（半行交错/丢前缀？）：%q", raw)
		}
		out = append(out, line{m[1], m[2]})
	}
	return out
}

func wantLines(t *testing.T, buf *bytes.Buffer, want ...line) {
	t.Helper()
	got := parse(t, buf)
	if len(got) != len(want) {
		t.Fatalf("行数：得 %d 要 %d\n%s", len(got), len(want), buf.String())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 行：得 %+v 要 %+v", i+1, got[i], want[i])
		}
	}
}

func TestPrefixEachLine(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Printf("app name %s\n", "unite_11")
	fmt.Println("echo socket 127.0.0.1:15001")
	fmt.Println("中文行：等待发现：store, entry0, public")
	Flush()
	wantLines(t, out,
		line{"world", "app name unite_11"},
		line{"world", "echo socket 127.0.0.1:15001"},
		line{"world", "中文行：等待发现：store, entry0, public"},
	)
}

func TestStderrPrefixed(t *testing.T) {
	_, errb := withSink(t, "entry")
	fmt.Fprintln(os.Stderr, "entry:", errors.New("boom"))
	fmt.Println("stdout 侧")
	Flush()
	wantLines(t, errb, line{"entry", "entry: boom"})
}

// 标准库 log 在包初始化时缓存了 os.Stderr：Install 后也要经前缀，且不
// 保留它自带的日期。
func TestStdLoggerPrefixed(t *testing.T) {
	_, errb := withSink(t, "store")
	log.Printf("dbconn: %s 断连", "nx_base")
	log.Println("dbconn: 重连")
	Flush()
	wantLines(t, errb, line{"store", "dbconn: nx_base 断连"}, line{"store", "dbconn: 重连"})
}

// 启动早期来源未定：先 Install 再 SetServer，之后的行用新来源。
func TestSetServer(t *testing.T) {
	out, _ := withSink(t, "member")
	fmt.Println("early")
	SetServer("member3")
	fmt.Println("late")
	Flush()
	wantLines(t, out, line{"member", "early"}, line{"member3", "late"})
}

func TestBlankLinesStayBlank(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Print("a\n\n\nb\n")
	Flush()
	wantLines(t, out, line{"world", "a"}, line{}, line{}, line{"world", "b"})
}

func TestCRLFPreserved(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Print("line one\r\n\r\n")
	Flush()
	wantLines(t, out, line{"world", "line one\r"}, line{})
}

func TestMultiLineSingleWrite(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Print("x\ny\nz\n")
	Flush()
	wantLines(t, out, line{"world", "x"}, line{"world", "y"}, line{"world", "z"})
}

// 无换行的写只是半行：拼到同一行，不另起前缀；收尾时补发。
func TestPartialLineJoinedThenFlushed(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Print("ab")
	fmt.Print("cd\n")
	fmt.Print("no newline")
	Flush()
	wantLines(t, out, line{"world", "abcd"}, line{"world", "no newline"})
}

func TestTaggedRelabelsLine(t *testing.T) {
	out, _ := withSink(t, "solounite")
	fmt.Println("one")
	Tagged("store", nil)("item %d", 7)
	fmt.Println("two")
	Flush()
	wantLines(t, out,
		line{"solounite", "one"}, line{"store", "item 7"}, line{"solounite", "two"})
}

func TestTaggedMultiLineMessage(t *testing.T) {
	out, _ := withSink(t, "solounite")
	Tagged("entry", nil)("a\nb\n")
	Tagged("entry", nil)("c\n\n")
	Flush()
	wantLines(t, out,
		line{"entry", "a"}, line{"entry", "b"}, line{"entry", "c"}, line{})
}

// 上一行没以换行收尾时来了带标记的写：标记处强制断行，不漏标记字节。
func TestTaggedMidLineForcesBreak(t *testing.T) {
	out, _ := withSink(t, "solounite")
	fmt.Print("abc")
	Tagged("store", nil)("x")
	Flush()
	wantLines(t, out, line{"solounite", "abc"}, line{"store", "x"})
}

// 标记跨读段被截断且再无后续：按正文字节落，不吞行。
func TestTaggedIncompleteMarkerAtEOF(t *testing.T) {
	out, _ := withSink(t, "world")
	fmt.Println("ok")
	os.Stdout.Write([]byte("tail\x00wo"))
	Flush()
	wantLines(t, out, line{"world", "ok"}, line{"world", "tail\x00wo"})
}

func TestTaggedPassesThroughWhenNotInstalled(t *testing.T) {
	cur.Store(nil)
	var got []string
	Tagged("store", func(format string, args ...any) {
		got = append(got, fmt.Sprintf(format, args...))
	})("a %d-%d", 1, 2)
	Tagged("store", nil)("dropped")
	if len(got) != 1 || got[0] != "a 1-2" {
		t.Fatalf("未安装时应原样透传：得 %q", got)
	}
}

// 行首同名来源标记去重；异名/非标记原样保留。
func TestSelfTagDeduped(t *testing.T) {
	out, _ := withSink(t, "monitor")
	fmt.Println("[monitor] echo socket 127.0.0.1:15012")
	fmt.Println("[monitor]")
	fmt.Println("[world] not mine")
	fmt.Println("( SoloStore::DiscoveryWatch)")
	Flush()
	wantLines(t, out,
		line{"monitor", "echo socket 127.0.0.1:15012"},
		line{"monitor", ""},
		line{"monitor", "[world] not mine"},
		line{"monitor", "( SoloStore::DiscoveryWatch)"},
	)

	out2, _ := withSink(t, "world")
	fmt.Println("(world)world state LoadRoles -> LoadScenes")
	Flush()
	wantLines(t, out2, line{"world", "world state LoadRoles -> LoadScenes"})
}

func TestConcurrentWritersNoTornLines(t *testing.T) {
	const workers, per = 16, 200
	out, _ := withSink(t, "solounite")
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < per; i++ {
				if i%5 == 0 {
					Tagged(fmt.Sprintf("member%d", w%2), nil)("t %d %d", w, i)
					continue
				}
				fmt.Printf("w=%d i=%d\n", w, i)
			}
		}(w)
	}
	wg.Wait()
	Flush()

	got := parse(t, out)
	if len(got) != workers*per {
		t.Fatalf("行数：得 %d 要 %d", len(got), workers*per)
	}
	seen := map[string]bool{}
	for _, l := range got {
		var w, i int
		if _, err := fmt.Sscanf(l.msg, "w=%d i=%d", &w, &i); err == nil {
			if l.server != "solounite" {
				t.Errorf("普通行来源：%q", l.server)
			}
		} else if _, err := fmt.Sscanf(l.msg, "t %d %d", &w, &i); err == nil {
			if want := fmt.Sprintf("member%d", w%2); l.server != want {
				t.Errorf("标签行来源：得 %q 要 %q", l.server, want)
			}
		} else {
			t.Errorf("半行交错：%+v", l)
		}
		if seen[l.msg] {
			t.Errorf("重复行：%q", l.msg)
		}
		seen[l.msg] = true
	}
}

// TestHelperProcess 不是测试：被子进程用例当作「装了 logline 的服务」。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("LOGLINE_HELPER") != "1" {
		return
	}
	Install("world")
	defer Flush()
	for i := 0; i < 2000; i++ {
		fmt.Printf("line %d\n", i)
	}
	Tagged("store", nil)("tagged %d", 1)
	fmt.Fprintln(os.Stderr, "fatal thing")
	Exit(3) // 末行刚写完就退——不排空必丢
}

// 真实 Install + Exit：子进程里 2000 行后紧接退出，行数/来源/退出码齐全。
func TestInstallExitSubprocess(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "LOGLINE_HELPER=1")
	raw, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 3 {
		t.Fatalf("退出码：err=%v（要 3）\n%s", err, raw)
	}

	var buf bytes.Buffer
	buf.Write(raw)
	got := parse(t, &buf)
	if len(got) != 2002 {
		t.Fatalf("行数：得 %d 要 2002（丢了 %d 行）", len(got), 2002-len(got))
	}
	for i := 0; i < 2000; i++ {
		if want := (line{"world", fmt.Sprintf("line %d", i)}); got[i] != want {
			t.Fatalf("第 %d 行：得 %+v 要 %+v", i, got[i], want)
		}
	}
	if got[2000] != (line{"store", "tagged 1"}) || got[2001].msg != "fatal thing" {
		t.Fatalf("尾行：%+v %+v", got[2000], got[2001])
	}
}

// 已 Install 的入口不得再直接 os.Exit、且须 defer Flush——绕开即丢末行。
func TestInstalledMainsExitThroughLogline(t *testing.T) {
	mains, err := filepath.Glob(filepath.Join("..", "..", "cmd", "*", "main.go"))
	if err != nil || len(mains) == 0 {
		t.Fatalf("cmd/*/main.go: %v（%d)", err, len(mains))
	}
	n := 0
	for _, f := range mains {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), "logline.Install(") {
			continue
		}
		n++
		if strings.Contains(string(src), "os.Exit(") {
			t.Errorf("%s：装了 logline 仍有 os.Exit（改 logline.Exit）", f)
		}
		if !strings.Contains(string(src), "defer logline.Flush()") {
			t.Errorf("%s：缺 defer logline.Flush()（main 正常返回会丢末行）", f)
		}
	}
	if n == 0 {
		t.Fatal("没有任何入口接入 logline.Install（cmd 路径变了？）")
	}
}
