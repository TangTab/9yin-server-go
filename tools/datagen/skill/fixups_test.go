package skill

// fixups_test.go 审定修正（tableSpec.Fixups）：源表脏数据在构建期按审定值
// 改写并入报告，使产物里的手修定版可由重建逐字节复现。

import (
	"fmt"
	"strings"
	"testing"
)

func TestConvertTableAppliesAuditedFixups(t *testing.T) {
	raw := []byte(strings.Join([]string{
		"[-1]", "Formula,Src1", "[0]", "40,4", "",
		"[1116]", "Formula=((P1+((P2+P3+P4+P5)*0.065))", "Src1=5", "",
		"[9]", "Formula=(P1+P2)", "Src1=5", "",
		"[77]", "Formula=1", "",
	}, "\r\n"))
	const fixed = "(P1+((P2+P3+P4+P5)*0.065))"
	spec := tableSpec{Source: "formula.ini", Output: "Skill_Formula.xml", Headers: []string{"-1", "0"},
		Fixups: map[string]valueFix{
			"1116": {Key: "Formula", Value: fixed, Reason: "括号不配平"},
			"9":    {Key: "Formula", Value: "(P1+P2)", Reason: "源已自愈"},
			"404":  {Key: "Formula", Value: "x", Reason: "节缺席"},
		}}
	build := convertTable(parseINI(raw, "formula.ini"), spec, &vocabulary{})

	if build.Rows != 3 || len(build.Files) != 1 {
		t.Fatalf("行 %d 文件 %d, want 3/1", build.Rows, len(build.Files))
	}
	text := string(build.Files[0].Data)
	if !strings.Contains(text, `<Property ID="1116" Formula="`+fixed+`" Src1="5"/>`) || strings.Contains(text, "((P1") {
		t.Errorf("1116 应输出审定值:\n%s", text)
	}
	if !strings.Contains(text, `<Property ID="77" Formula="1"/>`) {
		t.Errorf("无关行应原样:\n%s", text)
	}
	// 报告只记命中的节（源值留档）；节缺席的修正项不记也不报错。
	want := []fixReport{
		{ID: "1116", Key: "Formula", From: "((P1+((P2+P3+P4+P5)*0.065))", To: fixed, Reason: "括号不配平"},
		{ID: "9", Key: "Formula", From: "(P1+P2)", To: "(P1+P2)", Reason: "源已自愈"},
	}
	if fmt.Sprint(build.Fixed) != fmt.Sprint(want) {
		t.Errorf("审定修正报告 = %+v, want %+v", build.Fixed, want)
	}
}
