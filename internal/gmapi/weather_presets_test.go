package gmapi

import "testing"

// TestGMWeatherPresetsClosedList 钉死天气闭名单：49 键、面板目录与
// WeatherPresetByKey 同源、任意目录串（路径注入）必须被拒绝。
func TestGMWeatherPresetsClosedList(t *testing.T) {
	if len(WeatherPresets) != 49 {
		t.Fatalf("weather presets = %d keys, want 49 (Java GmWeatherPresets 同数)", len(WeatherPresets))
	}
	seen := map[string]bool{}
	for _, preset := range WeatherPresets {
		if preset.Key == "" || preset.Label == "" {
			t.Fatalf("incomplete preset %+v", preset)
		}
		if seen[preset.Key] {
			t.Fatalf("duplicate weather key %q", preset.Key)
		}
		seen[preset.Key] = true
		if _, err := WeatherPresetByKey(preset.Key); err != nil {
			t.Fatalf("catalog key %q rejected by lookup: %v", preset.Key, err)
		}
	}
	for _, injected := range []string{"", "..", "../../evil", "heavy_rain_501 ", "HEAVY_RAIN_501"} {
		if _, err := WeatherPresetByKey(injected); err == nil {
			t.Fatalf("injected weather key %q must be rejected", injected)
		}
	}
}
