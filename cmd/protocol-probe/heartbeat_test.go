package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// 0x18 ping → 0x38 纯回显（fork 49840 抓包 326 对逐字节钉死）。
func TestHeartbeatReply(t *testing.T) {
	cases := []struct{ ping, want string }{
		{"183154470000140000000000", "383154470000140000000000"},
		{"18f15b470000150000000000", "38f15b470000150000000000"},
	}
	for _, c := range cases {
		ping, err := hex.DecodeString(c.ping)
		if err != nil {
			t.Fatal(err)
		}
		want, err := hex.DecodeString(c.want)
		if err != nil {
			t.Fatal(err)
		}
		if got := heartbeatReply(ping); !bytes.Equal(got, want) {
			t.Fatalf("heartbeatReply(%x)=%x, want %x", ping, got, want)
		}
	}
	// 回显帧是拷贝，不修改入参。
	ping, _ := hex.DecodeString("183154470000140000000000")
	before := append([]byte(nil), ping...)
	_ = heartbeatReply(ping)
	if !bytes.Equal(ping, before) {
		t.Fatal("heartbeatReply mutated the incoming ping frame")
	}
}
