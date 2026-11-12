package gmapi

import "testing"

func TestETagMatches(t *testing.T) {
	etag := ETagValue("abc")
	cases := []struct {
		name   string
		header string
		want   bool
	}{
		{"空头不命中", "", false},
		{"精确命中", etag, true},
		{"弱前缀命中", `W/"abc"`, true},
		{"列表中命中", `W/"x", ` + etag, true},
		{"星号全命中", "*", true},
		{"他值不命中", `W/"zzz"`, false},
		{"子串不命中", `W/"ab"`, false},
	}
	for _, testCase := range cases {
		if got := ETagMatches(testCase.header, etag); got != testCase.want {
			t.Errorf("%s: ETagMatches(%q)=%v want %v", testCase.name, testCase.header, got, testCase.want)
		}
	}
}

func TestHashTagDeterministicAndDistinct(t *testing.T) {
	first := HashTag("items", "久仙刀", "1")
	if first != HashTag("items", "久仙刀", "1") {
		t.Fatal("same parts must hash equal")
	}
	if first == HashTag("items", "久仙刀2", "1") {
		t.Fatal("different parts must hash distinct")
	}
	// 拼接歧义防护：("ab","c") ≠ ("a","bc")
	if HashTag("ab", "c") == HashTag("a", "bc") {
		t.Fatal("separator must prevent concatenation ambiguity")
	}
	if len(first) != 16 {
		t.Fatalf("hash length=%d want 16", len(first))
	}
}

func TestWeatherTagStable(t *testing.T) {
	if WeatherTag() != WeatherTag() {
		t.Fatal("weather tag must be stable within process")
	}
	if WeatherTag() == "" {
		t.Fatal("weather tag must not be empty")
	}
}
