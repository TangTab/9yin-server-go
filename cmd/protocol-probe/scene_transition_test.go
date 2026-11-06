package main

import "testing"

func TestParseScenePosition(t *testing.T) {
	position, err := parseScenePosition("1.25,2.5,-3.75,4")
	if err != nil {
		t.Fatal(err)
	}
	if position.X != 1.25 || position.Y != 2.5 || position.Z != -3.75 || position.Orient != 4 {
		t.Fatalf("position=%+v", position)
	}
}

func TestParseScenePositionRejectsNonFiniteValue(t *testing.T) {
	if _, err := parseScenePosition("NaN,2,3,4"); err == nil {
		t.Fatal("expected NaN to be rejected")
	}
}
