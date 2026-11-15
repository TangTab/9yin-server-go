package main

import "testing"

func TestResolveListenAddr(t *testing.T) {
	cases := []struct {
		spec     string
		defPort  string
		expected string
	}{
		// 原样合法形态。
		{"127.0.0.1:19061", "19061", "127.0.0.1:19061"},
		{"192.168.1.5:9000", "19061", "192.168.1.5:9000"},
		{"[::1]:19061", "19061", "[::1]:19061"},
		{"[fe80::1]:9000", "19061", "[fe80::1]:9000"},
		{":19061", "19061", ":19061"},
		// 无端口：补默认端口。
		{"localhost", "19062", "localhost:19062"},
		{"0.0.0.0", "19061", "0.0.0.0:19061"},
		// 裸 IPv6：自动补括号和默认端口。
		{"::1", "19061", "[::1]:19061"},
		{"fe80::1", "19061", "[fe80::1]:19061"},
		// 通配词：绑全部接口（IPv4+IPv6 双栈）。
		{"*", "19061", ":19061"},
		{"ANY", "19061", ":19061"},
		{"all:19061", "19061", ":19061"},
		{"::", "19061", ":19061"},
		{"[::]:19061", "19061", ":19061"},
		// 空串 = 关闭监听。
		{"", "19061", ""},
		{"  ", "19061", ""},
	}
	for _, c := range cases {
		if got := resolveListenAddr(c.spec, c.defPort); got != c.expected {
			t.Errorf("resolveListenAddr(%q, %q) = %q, want %q", c.spec, c.defPort, got, c.expected)
		}
	}
}
