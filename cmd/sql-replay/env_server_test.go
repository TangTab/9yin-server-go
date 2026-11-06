package main

import "testing"

func TestParseEnvServer(t *testing.T) {
	data := []byte("\xef\xbb\xbf" + // UTF-8 BOM
		":: 本机开发库配置（勿提交）\r\n" +
		"set \"NINEYIN_SERVER_ROOT=F:/Game/Online/9yin/9yin-server-go\"\r\n" +
		"set \"NINEYIN_MYSQL_DSN=nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin\"\r\n" +
		"# hash comment\n" +
		"settings=ignore whole line\n" +      // "settings" 不是 set 语句
		"set NO_EQUAL_SIGN\n" +               // 没有 =，跳过
		"set BARE=value\n" +                  // 无引号也接受
		"\n")
	got := parseEnvServer(data)
	want := map[string]string{
		"NINEYIN_SERVER_ROOT": "F:/Game/Online/9yin/9yin-server-go",
		"NINEYIN_MYSQL_DSN":   "nineyin:nineyin123@tcp(127.0.0.1:3306)/nineyin",
		"BARE":                "value",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries (%v), want %d", len(got), got, len(want))
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
}

func TestParseEnvServerEmpty(t *testing.T) {
	if got := parseEnvServer(nil); len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}
