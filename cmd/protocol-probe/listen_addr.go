package main

import (
	"net"
	"strings"
)

// resolveListenAddr 把 -listen/-gm-listen/-gm-internal-listen 的取值规范化成
// 可直接交给 net.Listen / http.Server 的地址，让绑定 IP 可以用任意习惯写法
// 表达（IPv4 字面量 / IPv6 字面量 / 通配）：
//
//	127.0.0.1:19061、192.168.1.5:19061   IPv4 字面量（原样）
//	[::1]:19061、[fe80::1]:19061         带括号 IPv6（原样）
//	::1、fe80::1                          裸 IPv6，自动补括号和默认端口
//	localhost、192.168.1.5               无端口，补默认端口
//	*、any、all（可带 :port）             任意 IP，绑全部接口（IPv4+IPv6 双栈）
//	:19061、[::]:19061                    显式通配（语义不变，统一成 :port）
//
// 0.0.0.0 保持原义（仅 IPv4 全接口）；裸 IPv6 不要自带端口（"::1:19061" 无法
// 消歧，应写 "[::1]:19061"）。spec 为空表示关闭该监听，原样返回空串。
func resolveListenAddr(spec, defaultPort string) string {
	s := strings.TrimSpace(spec)
	if s == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		// 无端口（或裸 IPv6）：整体视为 host，补默认端口。
		host, port = strings.Trim(s, "[]"), defaultPort
	} else {
		host = strings.Trim(host, "[]")
	}
	switch strings.ToLower(host) {
	case "*", "any", "all", "::":
		// 通配：空 host 即全部接口。Go 的 ":port" 在双栈系统上是
		// IPv6 socket + v4-mapped，一份监听同时收 IPv4 与 IPv6 连接。
		host = ""
	}
	return net.JoinHostPort(host, port)
}
