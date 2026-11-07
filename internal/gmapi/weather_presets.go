package gmapi

import "fmt"

// WeatherPreset 与 FactionPresets 同款约束：刻意闭名单。面板键 =
// 客户端 share/rule/weather/sceneparastore/ 目录名（完整天气渲染包：
// weather.ini/rain.ini/cloud.ini/pp*.ini），共 49 键，与 Java 参考
// GmWeatherPresets 逐键一致。GM 可换天气渲染包，不可注入任意路径。
//
// 协议链（res1/lua64 取证，Java 注释同源）：S2C 0x27 命令名寻址
// on_weather_sky（custom_handler.lua，签名 sub_type,...）；sub_type=4
// = 切换天气效果，客户端 weather_manager:ChangeWeatherEffect(TTID,
// fadeSec, TXID, 4)。命名规约：_001=白天变体、_501=夜晚变体。
type WeatherPreset struct {
	Key   string `json:"Key"`
	Label string `json:"Label"`
}

var WeatherPresets = []WeatherPreset{
	// —— 天气族（官方 tianqi_* 文案口径）——
	{"light_rain_001", "小雨（昼）"},
	{"light_rain_501", "小雨（夜）"},
	{"middling_rain_001", "中雨（昼）"},
	{"middling_rain_501", "中雨（夜）"},
	{"heavy_rain_001", "大雨（昼）"},
	{"heavy_rain_501", "大雨（夜）"},
	{"heavy_ocean_rain", "海上暴雨"},
	{"light_snow_001", "小雪（昼）"},
	{"light_snow_501", "小雪（夜）"},
	{"middling_snow_001", "中雪（昼）"},
	{"middling_snow_501", "中雪（夜）"},
	{"heavy_snow_001", "暴雪（昼）"},
	{"heavy_snow_501", "暴雪（夜）"},
	{"heavy_snow_huodong01", "暴雪·活动"},
	{"light_fog_001", "小雾（昼）"},
	{"light_fog_501", "小雾（夜）"},
	{"middling_fog_001", "中雾（昼）"},
	{"middling_fog_501", "中雾（夜）"},
	{"heavy_fog_001", "大雾（昼）"},
	{"heavy_fog_501", "大雾（夜）"},
	{"light_sandstorm_001", "风沙（昼）"},
	{"light_sandstorm_501", "风沙（夜）"},
	{"middling_sandstorm_001", "沙暴（昼）"},
	{"middling_sandstorm_501", "沙暴（夜）"},
	{"heavy_sandstorm_001", "强沙暴（昼）"},
	{"heavy_sandstorm_501", "强沙暴（夜）"},
	{"lunar_eclipse_001", "月食（天狗吃月）"},
	{"solar_eclipse_001", "日食（天狗食日）"},
	{"moon_night_time4", "月夜"},
	{"heavy_night_time4", "深夜"},
	// —— 场景时段族（各场景基础时段渲染包，测试/演出用）——
	{"born04_time1", "出生场景·时段1"},
	{"born04_time2", "出生场景·时段2"},
	{"born04_time3", "出生场景·时段3"},
	{"born04_time4", "出生场景·时段4"},
	{"born04_time5", "出生场景·时段5"},
	{"born04_time6", "出生场景·时段6"},
	{"clone003_time1", "副本003·时段1"},
	{"clone003_time2", "副本003·时段2"},
	{"clone003_time3", "副本003·时段3"},
	{"clone003_time4", "副本003·时段4"},
	{"groupscene008_time1", "群战008·时段1"},
	{"groupscene008_time2", "群战008·时段2"},
	{"groupscene008_time3", "群战008·时段3"},
	{"groupscene008_time4", "群战008·时段4"},
	{"groupscene023", "群战023·时段"},
	{"groupscene023_02", "群战023·时段二"},
	{"groupscene039", "群战039·时段"},
	{"scene10", "场景10·时段"},
	{"xianglong_show", "降龙演出"},
}

// WeatherPresetByKey 闭名单查键；任意目录串（路径注入）必须被拒绝。
func WeatherPresetByKey(key string) (WeatherPreset, error) {
	for _, preset := range WeatherPresets {
		if preset.Key == key {
			return preset, nil
		}
	}
	return WeatherPreset{}, fmt.Errorf("unknown GM weather preset %q", key)
}

// WeatherCatalogRow 服务面板天气下拉（console-features.js loadWeather
// 读 Key/Label/TXID，字段名大写与 Java gm-web 目录同形）。TXID 特殊特效
// 串恒为空：闭名单只切渲染包，不注入路径。
type WeatherCatalogRow struct {
	Key   string `json:"Key"`
	Label string `json:"Label"`
	TXID  string `json:"TXID"`
}

// WeatherCatalog 返回天气下拉的目录行；内嵌面板与独立面板共用。
func WeatherCatalog() []WeatherCatalogRow {
	rows := make([]WeatherCatalogRow, 0, len(WeatherPresets))
	for _, preset := range WeatherPresets {
		rows = append(rows, WeatherCatalogRow{Key: preset.Key, Label: preset.Label, TXID: ""})
	}
	return rows
}
