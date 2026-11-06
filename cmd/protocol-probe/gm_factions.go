package main

import "fmt"

// gmFactionPreset is deliberately a closed list of IDs that the installed
// modern client recognizes in form_main_player.ini.  GM may change the
// character's School property, but it must never accept arbitrary strings.
type gmFactionPreset struct {
	Label   string
	Faction string
}

var gmFactionPresets = map[string]gmFactionPreset{
	// 原始门派
	"shaolin":     {Label: "少林", Faction: "school_shaolin"},
	"wudang":      {Label: "武当", Faction: "school_wudang"},
	"emei":        {Label: "峨眉", Faction: "school_emei"},
	"gaibang":     {Label: "丐帮", Faction: "school_gaibang"},
	"tangmen":     {Label: "唐门", Faction: "school_tangmen"},
	"junzitang":   {Label: "君子堂", Faction: "school_junzitang"},
	"jinyiwei":    {Label: "锦衣卫", Faction: "school_jinyiwei"},
	"jilegu":      {Label: "极乐谷", Faction: "school_jilegu"},
	"mingjiao":    {Label: "明教", Faction: "school_mingjiao"},
	"tianshan":    {Label: "天山派", Faction: "school_tianshan"},
	"kunlun":      {Label: "昆仑派", Faction: "school_kunlun"},
	"xingtianbao": {Label: "刑天城", Faction: "school_xingtianbao"},
	"wuxu":        {Label: "无虚门", Faction: "school_wuxu"},

	// 江湖势力
	"yihuagong": {Label: "移花宫", Faction: "force_yihua"},
	"taohuadao": {Label: "桃花岛", Faction: "force_taohua"},
	"xujia":     {Label: "徐家庄", Faction: "force_xujia"},
	"wanshou":   {Label: "万兽山庄", Faction: "force_wanshou"},
	"jinzhen":   {Label: "金针沈家", Faction: "force_jinzhen"},
	"wugenmen":  {Label: "无根门", Faction: "force_wugen"},
	"tianlun":   {Label: "天轮寺", Faction: "force_tianlun"},
	"qingyi":    {Label: "青衣阁", Faction: "force_qingyi"},

	// 宗门 / 新门派
	"xuedao":    {Label: "血刀门", Faction: "newschool_xuedao"},
	"huashan":   {Label: "华山派", Faction: "newschool_huashan"},
	"gumu":      {Label: "古墓派", Faction: "newschool_gumu"},
	"damo":      {Label: "达摩派", Faction: "newschool_damo"},
	"shenshui":  {Label: "神水宫", Faction: "newschool_shenshui"},
	"changfeng": {Label: "长风镖局", Faction: "newschool_changfeng"},
	"nianluo":   {Label: "念萝坝", Faction: "newschool_nianluo"},
	"wuxian":    {Label: "无仙教", Faction: "newschool_wuxian"},
	"shenji":    {Label: "神机营", Faction: "newschool_shenji"},
	"xingmiao":  {Label: "星渺阁", Faction: "newschool_xingmiao"},
	"tianya":    {Label: "天涯海阁", Faction: "newschool_tianya"},
	"wanghui":   {Label: "王回洲", Faction: "newschool_wanghui"},
	"shenjihui": {Label: "神机汇", Faction: "newschool_shenjihui"},
}

func gmFactionPresetByKey(key string) (gmFactionPreset, error) {
	preset, found := gmFactionPresets[key]
	if !found {
		return gmFactionPreset{}, fmt.Errorf("unknown GM faction preset %q", key)
	}
	return preset, nil
}
