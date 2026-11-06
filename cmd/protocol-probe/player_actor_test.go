package main

import "testing"

func TestQingGongMotionPropertiesUseModernRushRangeConfig(t *testing.T) {
	want := map[uint16]float32{
		137: 5.0,
		138: 1.5,
		139: 8.0,
		140: 6.5,
		141: 8.0,
		142: 6.5,
	}
	got := make(map[uint16]float32)
	for _, property := range qinggongMotionProperties() {
		if _, needed := want[property.Index]; needed {
			got[property.Index] = property.Value.F32
		}
	}
	for index, expected := range want {
		if actual, ok := got[index]; !ok || actual != expected {
			t.Fatalf("property %d = %v, want %v", index, actual, expected)
		}
	}
}
