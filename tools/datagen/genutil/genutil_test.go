package genutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestToCRLF(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"纯 LF", "<a/>\n<b/>\n", "<a/>\r\n<b/>\r\n"},
		{"已是 CRLF", "<a/>\r\n<b/>\r\n", "<a/>\r\n<b/>\r\n"},
		{"混合", "<a/>\r\n<b/>\n", "<a/>\r\n<b/>\r\n"},
		{"无换行", "<a/>", "<a/>"},
		{"空", "", ""},
	}
	for _, c := range cases {
		if got := string(toCRLF([]byte(c.in))); got != c.want {
			t.Errorf("%s: toCRLF(%q) = %q，应为 %q", c.name, c.in, got, c.want)
		}
	}
}

// 没有同名现有文件：原样写出（LF 不动）；有文件且首换行是 CRLF：跟随。
func TestWriteFileFollowsExistingLineEndings(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a.xml")

	fresh := filepath.Join(root, "sub", "fresh.xml")
	if err := WriteFile(fresh, []byte("<a/>\n<b/>\n")); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(fresh); err != nil || string(data) != "<a/>\n<b/>\n" {
		t.Fatalf("无现有文件时应原样写出: %q（err=%v）", data, err)
	}

	if err := os.WriteFile(target, []byte("<old/>\r\n<old/>\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(target, []byte("<new/>\n<new/>\n")); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "<new/>\r\n<new/>\r\n" {
		t.Fatalf("现有文件是 CRLF 时应跟随: %q（err=%v）", data, err)
	}

	if err := os.WriteFile(target, []byte("<old/>\n<old/>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(target, []byte("<new/>\n<new/>\n")); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "<new/>\n<new/>\n" {
		t.Fatalf("现有文件是 LF 时应保持 LF: %q（err=%v）", data, err)
	}
}

// useFakeWrite 把实际写盘换成 fake，并把重试间隔缩到最短；测试结束恢复。
func useFakeWrite(t *testing.T, fake func(string, []byte, os.FileMode) error) {
	t.Helper()
	oldWrite, oldDelay := writeFile, writeRetryDelay
	writeFile, writeRetryDelay = fake, time.Millisecond
	t.Cleanup(func() { writeFile, writeRetryDelay = oldWrite, oldDelay })
}

// 占用类错误重试后成功：前两次短暂占用，第三次写成。
func TestWriteFileRetriesTransientLock(t *testing.T) {
	calls := 0
	useFakeWrite(t, func(path string, data []byte, perm os.FileMode) error {
		calls++
		if calls <= 2 {
			return &fs.PathError{Op: "open", Path: path, Err: transientLockError()}
		}
		return os.WriteFile(path, data, perm)
	})

	target := filepath.Join(t.TempDir(), "a.xml")
	if err := WriteFile(target, []byte("<a/>\n")); err != nil {
		t.Fatalf("短暂占用应重试到成功: %v", err)
	}
	if calls != 3 {
		t.Fatalf("应恰好写 3 次，实得 %d", calls)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "<a/>\n" {
		t.Fatalf("内容不符: %q（err=%v）", data, err)
	}
}

// 占用一直不释放：重试次数用尽后把错误原样返回，不无限等。
func TestWriteFileGivesUpAfterRetries(t *testing.T) {
	calls := 0
	useFakeWrite(t, func(string, []byte, os.FileMode) error {
		calls++
		return &fs.PathError{Op: "open", Err: transientLockError()}
	})

	err := WriteFile(filepath.Join(t.TempDir(), "a.xml"), []byte("<a/>\n"))
	if err == nil {
		t.Fatal("持续占用应最终报错")
	}
	if calls != writeRetries+1 {
		t.Fatalf("应写 %d 次后放弃，实得 %d", writeRetries+1, calls)
	}
	if !isTransientWriteError(err) {
		t.Fatalf("返回的应是原始的占用类错误: %v", err)
	}
}

// 非占用类错误不重试，立即返回。
func TestWriteFileDoesNotRetryOtherErrors(t *testing.T) {
	calls := 0
	useFakeWrite(t, func(string, []byte, os.FileMode) error {
		calls++
		// Windows 上是 access denied；类 Unix 上是 EIO。两者都不是占用类错误。
		return &fs.PathError{Op: "open", Err: syscall.Errno(5)}
	})

	if err := WriteFile(filepath.Join(t.TempDir(), "a.xml"), []byte("<a/>\n")); err == nil {
		t.Fatal("应报错")
	}
	if calls != 1 {
		t.Fatalf("非占用类错误不应重试，实得 %d 次", calls)
	}
}
