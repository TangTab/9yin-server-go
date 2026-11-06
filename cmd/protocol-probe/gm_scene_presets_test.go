package main

import "testing"

func TestGMScenePresetsHaveValidDestinations(t *testing.T) {
	if got, want := len(gmScenePresets), 38; got != want {
		t.Fatalf("preset count=%d want=%d", got, want)
	}
	for key, preset := range gmScenePresets {
		destination, err := gmScenePresetDestination(key)
		if err != nil {
			t.Fatalf("preset %q (%s): %v", key, preset.Label, err)
		}
		if destination.location.Scene.Config == "" || destination.location.Scene.Resource == "" {
			t.Fatalf("preset %q (%s) has empty scene: %+v", key, preset.Label, destination)
		}
	}
}

func TestGMScenePresetRejectsUnknownKey(t *testing.T) {
	if _, err := gmScenePresetDestination("not-a-map"); err == nil {
		t.Fatal("unknown preset was accepted")
	}
}

func TestForcePresetsUseOfficialDestinations(t *testing.T) {
	cases := map[string]struct {
		x, y, z, orient float32
	}{
		"taohuadao": {-603.500, 95.779, -259.100, 0.255},
		"wugenmen":  {34.200, 4.600, 245.500, 1.500},
		"xujia":     {569.750, 39.045, 272.750, 0.000},
		"jinzhen":   {354.750, 81.061, 800.000, 0.000},
		"wanshou":   {338.321, 87.146, 1016.193, 2.927},
		"tianlun":   {-45.777, -34.792, -243.900, 0.000},
		"qingyi":    {1316.718, 21.183, 1389.666, 0.000},
		"changfeng": {1582.000, 9.000, 470.000, 0.000},
		"wanghui":   {119.267, 12.734, 3.625, 0.000},
		"xuedao":    {552.000, 32.500, 922.000, 0.000},
		"huashan":   {1563.800, 133.500, 694.700, 0.000},
		"gumu":      {-277.595, 355.876, -325.481, 0.000},
		"damo":      {-373.021, 276.227, -573.989, 0.000},
		"shenshui":  {2269.324, 162.736, -1379.467, 0.000},
		"nianluo":   {2133.586, 108.031, 1007.968, 0.000},
		"wuxian":    {1451.899, 49.252, 799.405, 0.000},
		"shenji":    {-45.400, 18.600, -46.900, 0.000},
		"xingmiao":  {259.598, 246.446, 70.540, 0.000},
		"tianya":    {1745.635, 18.729, 1066.640, 0.000},
		"shenjihui": {-48.765, -153.637, -4770.149, 0.000},
		"wuxu":      {-3927.438, 69.360, -5407.769, 0.000},
	}
	for key, want := range cases {
		destination, err := gmScenePresetDestination(key)
		if err != nil {
			t.Fatalf("preset %q: %v", key, err)
		}
		got := destination.location.Position
		if got.X != want.x || got.Y != want.y || got.Z != want.z || got.Orient != want.orient {
			t.Fatalf("preset %q position=%+v want=%+v", key, got, want)
		}
	}
}
