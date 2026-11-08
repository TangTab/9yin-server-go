package combatcalc

import "testing"

// golden 用例取自 formula.ini 原文（客户端 share/skill/formula.ini）：
// [1] 标准武器公式、[2] 段数分摊、[44] 暗器、[494] 太极拳系威力系数、
// [25] 双重分摊、[63] 全角括号、[10] 纯数字、[16] 拼招、[5] 距离引用。
func TestTranspileFormulaGolden(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"formula1", "P1+P2+((P4+R[P5,P6])/P3)", "PV[1]+PV[2]+((PV[4]+R(PV[5],PV[6]))/PV[3])"},
		{"formula2", "(P1+P2+P3+R[P4,P5])/P6", "(PV[1]+PV[2]+PV[3]+R(PV[4],PV[5]))/PV[6]"},
		{"formula44", "(P1*P4+P2+(R[P5,P6]*4/P3))/P7", "(PV[1]*PV[4]+PV[2]+(R(PV[5],PV[6])*4/PV[3]))/PV[7]"},
		{"formula494", "(P1+((P4+P5+P6+P7)*P2*0.195))/P3", "(PV[1]+((PV[4]+PV[5]+PV[6]+PV[7])*PV[2]*0.195))/PV[3]"},
		{"formula25", "P1/(P2*P3)", "PV[1]/(PV[2]*PV[3])"},
		{"formula63_fullwidth", "（P1*4+R[1,200]）*P2", "(PV[1]*4+R(1,200))*PV[2]"},
		{"formula10_plain", "9999999", "9999999"},
		{"formula16", "(P1+P2*R[P3,P4])/P5", "(PV[1]+PV[2]*R(PV[3],PV[4]))/PV[5]"},
		{"formula14", "R[P1,P2]*P3", "R(PV[1],PV[2])*PV[3]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transpileFormula(tc.in)
			if err != nil {
				t.Fatalf("transpile(%q) error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("transpile(%q)\n got  %s\n want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestTranspileFormulaRejects(t *testing.T) {
	for _, in := range []string{"sin(P1)", "P", "R[P1", "#", "", "  ", "P1+foo"} {
		if _, err := transpileFormula(in); err == nil {
			t.Fatalf("transpile(%q) 应报错却成功", in)
		}
	}
}

func TestClosureSource(t *testing.T) {
	got := closureSource("PV[1]")
	want := "return function(PV, R, Q) return PV[1] end"
	if got != want {
		t.Fatalf("closureSource = %q, want %q", got, want)
	}
}
