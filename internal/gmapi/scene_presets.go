package gmapi

import (
	"fmt"

	"github.com/local/9yin-go-server/internal/role"
)

// ScenePreset 是具名的、有资源依据的实机场景目的地。
// 将其与自由填写的 GM 字段分开维护：每个条目都取自
// 已安装的 scenes.ini 和 door_data.ini，
// 这样按钮就不会误发过期的地形别名或来自其他地图的坐标。
type ScenePreset struct {
	Label       string
	Destination string // config,resource,x,y,z,orient
}

// ScenePresets 覆盖八座常用主城/新手出生城镇、最初的
// 八大门派以及八大江湖势力。场景配置名取自
// res/ini/scenes.ini；出生坐标取自各场景已安装 door_data.ini 的
// Target* 值或各势力专属的安全 HomePoint。鸡鸣驿使用现代的
// born01_MoHeZhen 资源。
var ScenePresets = map[string]ScenePreset{
	// 主城 / 常用城镇
	"yanjing":      {Label: "燕京", Destination: `ini\scene\city01_YanJing,city01,837.625,-35.939,-750.860,0.680`},
	"suzhou":       {Label: "苏州", Destination: `ini\scene\city02_SuZhou,city02,438.140,0.800,1188.282,-2.325`},
	"jinling":      {Label: "金陵", Destination: `ini\scene\city03_JinLing,city03,1675.078,10.726,-101.141,0.335`},
	"luoyang":      {Label: "洛阳", Destination: `ini\scene\city04_LuoYang,city04,243.387,43.277,403.821,5.024`},
	"chengdu":      {Label: "成都", Destination: `ini\scene\city05_ChengDu,city05,911.073,31.192,537.350,1.742`},
	"jimingyi":     {Label: "鸡鸣驿", Destination: `ini\scene\born01_MoHeZhen,born01,998.817,8.364,530.187,-1.498`},
	"yanyuzhuang":  {Label: "烟雨庄", Destination: `ini\scene\born03_YanYuZhuang,born03,870.655,58.287,717.367,-1.513`},
	"qiandengzhen": {Label: "千灯镇", Destination: `ini\scene\born04_QianDengZhen,born04,883.000,-29.000,664.000,4.680`},

	// 八大门派
	"jinyiwei":  {Label: "锦衣卫", Destination: `ini\scene\school01_JinYiWei,school01,480.046,49.653,473.986,0.621`},
	"gaibang":   {Label: "丐帮", Destination: `ini\scene\school02_GaiBang,school02,522.804,10.774,694.848,2.056`},
	"junzitang": {Label: "君子堂", Destination: `ini\scene\school03_JunZiTang,school03,1168.737,9.443,515.075,4.740`},
	"jilegu":    {Label: "极乐谷", Destination: `ini\scene\school04_JiLeGu,school04,851.084,14.440,373.077,2.892`},
	"tangmen":   {Label: "唐门", Destination: `ini\scene\school05_TangMen,school05,716.682,6.774,495.249,5.705`},
	"emei":      {Label: "峨眉", Destination: `ini\scene\school06_EMei,school06,687.422,36.738,726.217,5.836`},
	"wudang":    {Label: "武当", Destination: `ini\scene\school07_WuDang,school07,324.751,-0.405,812.974,2.137`},
	"shaolin":   {Label: "少林", Destination: `ini\scene\school08_ShaoLin,school08,517.669,-52.775,670.125,2.537`},

	// 江湖势力
	"yihuagong": {Label: "移花宫", Destination: `ini\scene\school09_YiHuaGong,school09,340.160,47.979,657.284,2.588`},
	// 这些是已安装的地形摆放点，而非外部的 Relive 复活 NPC。
	// 桃花岛预设刻意生成在该岛主 NPC
	// thd_001（黄古萧，见 commonnpc.xml）旁边；苏州传送门目标点在本
	// 客户端版本中是一处水域落点。无根门对应 door_data:[2132]（即
	// 直达“传送去无根门”路线），紧邻常驻 NPC 聚集处。
	"taohuadao": {Label: "桃花岛", Destination: `ini\scene\school10_taohuadao,school10,-603.500,95.779,-259.100,0.255`},
	"wugenmen":  {Label: "无根门", Destination: `ini\scene\school11_wugenmen,school11,34.200,4.600,245.500,1.500`},
	// 其余江湖势力驻地均已在解包出的
	// homepoint.ini 中标注 Safe=1。徐家庄与金针沈家共用成都地形；万兽山庄
	// 共用丐帮地形。天轮寺、青衣阁使用各自已安装的 adv156/adv158
	// 场景配置与 creator 清单。
	"xujia":   {Label: "徐家庄", Destination: `ini\scene\city05_ChengDu,city05,569.750,39.045,272.750,0.000`},
	"jinzhen": {Label: "金针沈家", Destination: `ini\scene\city05_ChengDu,city05,354.750,81.061,800.000,0.000`},
	"wanshou": {Label: "万兽山庄", Destination: `ini\scene\school02_GaiBang,school02,338.321,87.146,1016.193,2.927`},
	"tianlun": {Label: "天轮寺", Destination: `ini\scene\adv156_tianlun,adv156,-45.777,-34.792,-243.900,0.000`},
	"qingyi":  {Label: "青衣阁", Destination: `ini\scene\adv158_qingyi,adv158,1316.718,21.183,1389.666,0.000`},

	// 隐世宗门
	// 长风镖局的默认 HomePoint 实际落在金陵场景；王回洲使用 school28
	// 的独立场景。两处均为 homepoint.ini 标注的 Safe=1 出生点。
	"changfeng": {Label: "长风镖局", Destination: `ini\scene\city03_JinLing,city03,1582.000,9.000,470.000,0.000`},
	"wanghui":   {Label: "王回洲", Destination: `ini\scene\school28_wanghui,school28,119.267,12.734,3.625,0.000`},
	"xuedao":    {Label: "血刀门", Destination: `ini\scene\born02_ERenGU,born02,552.000,32.500,922.000,0.000`},
	"huashan":   {Label: "华山派", Destination: `ini\scene\school17_huashanpai,school17,1563.800,133.500,694.700,0.000`},
	"gumu":      {Label: "古墓派", Destination: `ini\scene\school14_gumupai,school14,-277.595,355.876,-325.481,0.000`},
	"damo":      {Label: "达摩派", Destination: `ini\scene\school18_damo,school18,-373.021,276.227,-573.989,0.000`},
	"shenshui":  {Label: "神水宫", Destination: `ini\scene\school19_shenshuigong,school19,2269.324,162.736,-1379.467,0.000`},
	"nianluo":   {Label: "念萝坝", Destination: `ini\scene\school15_nianluoba,school15,2133.586,108.031,1007.968,0.000`},
	"wuxian":    {Label: "无仙教", Destination: `ini\scene\school12_wuxianjiao,school12,1451.899,49.252,799.405,0.000`},
	"shenji":    {Label: "神机营", Destination: `ini\scene\school21_shenjiying,school21,-45.400,18.600,-46.900,0.000`},
	"xingmiao":  {Label: "星渺阁", Destination: `ini\scene\school23_xingmiaoge,school23,259.598,246.446,70.540,0.000`},
	"tianya":    {Label: "天涯海阁", Destination: `ini\scene\school25_tianya,school25,1745.635,18.729,1066.640,0.000`},
	"shenjihui": {Label: "神机汇", Destination: `ini\scene\school31_shenjihui,school31,-48.765,-153.637,-4770.149,0.000`},
	"wuxu":      {Label: "无虚门", Destination: `ini\scene\school30_wuxu,school30,-3927.438,69.360,-5407.769,0.000`},
}

// ScenePresetDestination 把具名预设解析为已验证的目的地。
func ScenePresetDestination(key string) (role.Location, error) {
	preset, found := ScenePresets[key]
	if !found {
		return role.Location{}, fmt.Errorf("unknown GM scene preset %q", key)
	}
	destination, err := ParseDestination(preset.Destination)
	if err != nil {
		return role.Location{}, fmt.Errorf("invalid GM scene preset %q: %w", key, err)
	}
	return destination, nil
}
