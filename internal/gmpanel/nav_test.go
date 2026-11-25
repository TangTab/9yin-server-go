package gmpanel

// nav_test.go 左侧二级菜单与子页清单的一致性：main.js 的 children 与
// panels/test.js 的 SUBS 是两份手写表。此前两次只加了子页没加菜单（邮件、
// 训练木桩）——页面在、路由通，只是入口没有。按序逐项比对，防再漏。
// 不依赖 Node：两份清单都是定形的对象字面量，正则即可取。

import (
	"regexp"
	"strings"
	"testing"
)

var (
	// subsEntry test.js 的 SUBS 条目（行首缩进 + id / title / desc 三连）。
	subsEntry = regexp.MustCompile(`(?m)^\s*\{ id: "([a-z]+)", title: "([^"]*)", desc:`)
	// navChildren main.js 里 children: [ … ] 的整段。
	navChildren = regexp.MustCompile(`children: \[([\s\S]*?)\]`)
	// navEntry children 段内的条目。
	navEntry = regexp.MustCompile(`\{ id: "([a-z]+)", title: "([^"]*)" \}`)
)

func entries(re *regexp.Regexp, text string) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1]+"="+m[2])
	}
	return out
}

func TestTestBenchNavListsEverySubPage(t *testing.T) {
	subsJS, err := assets.ReadFile("web/static/app/panels/test.js")
	if err != nil {
		t.Fatal(err)
	}
	mainJS, err := assets.ReadFile("web/static/app/main.js")
	if err != nil {
		t.Fatal(err)
	}
	subs := entries(subsEntry, string(subsJS))
	block := navChildren.FindStringSubmatch(string(mainJS))
	if block == nil {
		t.Fatal("main.js 里找不到 children: […]（二级菜单清单）")
	}
	nav := entries(navEntry, block[1])
	if len(subs) == 0 {
		t.Fatal("test.js 里取不到 SUBS 条目（清单格式变了？同步本测试的正则）")
	}
	if strings.Join(nav, "|") != strings.Join(subs, "|") {
		t.Errorf("左侧二级菜单与子页清单不一致（同项同序才算齐）:\n menu: %s\n subs: %s", nav, subs)
	}
}
