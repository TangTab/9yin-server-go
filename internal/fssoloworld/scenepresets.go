package fssoloworld

// scenepresets.go GM 场景切换目的地解析：location.ini BornCJ 出生点表
//（rolestate 内嵌镜像，BirthSceneOptions 导出）+ 补充表 + scenedata 场景/
// 驿站表（客户端有地形的全部世界场景）+ 中文别名 + 直填形态。

import (
	"fmt"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/scenedata"
)

// sceneAliases 中文别名 → resource 短名。别名先行于场景表：显式别名可消歧
// （凌霄城 = scene06、星垂城 = scene35——场景表里 scene25 / scene36 各有同名
// 场景）。向阳坡 / 念萝坝的旧写法（襄阳坡 / 捻罗坝）保留兼容。
var sceneAliases = map[string]string{
	"燕京": "city01", "苏州": "city02", "金陵": "city03", "洛阳": "city04", "成都": "city05",
	"鸡鸣驿": "born01", "烟雨庄": "born03", "千灯镇": "born04", "恶人谷": "born02",
	"新手引导": "born_single_01",
	"锦衣卫":  "school01", "丐帮": "school02", "君子堂": "school03", "极乐谷": "school04",
	"唐门": "school05", "峨眉": "school06", "武当": "school07", "少林": "school08",
	"移花宫":   "school09",
	"幽云十六州": "scene01", "蓬莱仙境": "scene02", "泛黄平原": "scene03", "襄阳坡": "scene04", "向阳坡": "scene04",
	"捻罗坝": "scene05", "凌霄城": "scene06", "梅花岭": "scene07", "塞外大漠": "scene08",
	"塞北草原": "scene09", "星垂城": "scene35",
}

// sceneDestination 切换目的地（config 全路径 + resource 短名 + 落点）。
type sceneDestination struct {
	Config   string
	Resource string
	Position [4]float64
}

// extraSceneDestinations BornCJ 与场景表之外的补充目的地（手工锚点，先于
// 两者生效）：
//   - born_single_01：单场景克隆态，且 BornCJ Key 推导按首个 '_' 截断，
//     会被错截成 "born"。落点取引导起点 NPC 旁：引导链（TaskText
//     1339-1346）从沙滩的赵子亟（funnpcborn003031，194.514,5.598,-114.595）
//     开始，偏移 2m 避免模型重叠。
//
// 八派之外的场景（移花宫等）不在此表：落点由 scenedata 场景表按客户端
// 驿站 / 复活点表生成（移花宫 = HomePointschool09A，340.16,47.979,657.284，
// 与 relive_info.ini force_yihua 同值）。
var extraSceneDestinations = map[string]sceneDestination{
	rolestate.GuideScene: {
		Config:   rolestate.GuideScenePath,
		Resource: rolestate.GuideScene,
		Position: rolestate.GuideSceneLanding,
	},
}

// resolveSceneDestination key 形态（按序）："config|resource" 直填（须显式
// 坐标，落点 [4]float64 零值由调用方覆盖）/ 中文别名 → resource 短名 /
// 补充表 / BornCJ 出生点（born03、city01…，落点 = 出生点）/ scenedata 场景
// 表（resource 短名或唯一中文名——BornCJ 未登记的场景：八派之外的全部
// 门派与江湖场景；落点 = 默认驿站，无驿站场景 = 摆放档功能 NPC 旁）。
// 未解析 false。
func resolveSceneDestination(key string) (sceneDestination, bool) {
	if key == "" {
		return sceneDestination{}, false
	}
	if i := strings.IndexByte(key, '|'); i > 0 && i < len(key)-1 {
		return sceneDestination{Config: key[:i], Resource: key[i+1:]}, true
	}
	resourceKey := key
	if alias, ok := sceneAliases[key]; ok {
		resourceKey = alias
	}
	if extra, ok := extraSceneDestinations[resourceKey]; ok {
		return extra, true
	}
	if opt, ok := rolestate.FindSceneOption(resourceKey); ok {
		return sceneDestination{Config: opt.Config, Resource: opt.Key, Position: opt.Position}, true
	}
	if sc, ok := scenedata.Find(resourceKey); ok {
		return sceneOf(sc), true
	}
	if sc, ok := scenedata.FindByName(key); ok {
		return sceneOf(sc), true
	}
	return sceneDestination{}, false
}

// sceneOf scenedata 场景 → 切换目的地（落点 = 默认驿站）。
func sceneOf(sc scenedata.Scene) sceneDestination {
	return sceneDestination{
		Config:   sc.Config,
		Resource: sc.Resource,
		Position: [4]float64{sc.Landing.X, sc.Landing.Y, sc.Landing.Z, sc.Landing.Orient},
	}
}

// scenePresetKeys 可用 key 清单（echo `gm scene` 裸调用打印）：BornCJ 键域 +
// 补充键一行；场景表逐场景一行「类别 中文名(resource) → 默认落点」（BornCJ
// 登记的场景落点 = 出生点，不标）；末行别名提示。
func scenePresetKeys() string {
	opts := rolestate.BirthSceneOptions()
	keys := make([]string, 0, len(opts)+len(extraSceneDestinations))
	born := make(map[string]bool, len(opts))
	for _, opt := range opts {
		keys = append(keys, opt.Key)
		born[opt.Key] = true
	}
	for key := range extraSceneDestinations {
		keys = append(keys, key)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "keys: %s\n", strings.Join(keys, " "))
	for _, sc := range scenedata.Scenes() {
		name := sc.Name
		if name == "" {
			name = sc.Resource
		}
		fmt.Fprintf(&b, "%s %s(%s)", scenedata.CategoryName(sc.Category), name, sc.Resource)
		if !born[sc.Resource] {
			landing := sc.Landing.Title
			if landing == "" {
				landing = sc.Landing.Name
			}
			fmt.Fprintf(&b, " → %s", landing)
		}
		b.WriteString("\n")
	}
	b.WriteString("aliases: 上述场景的唯一中文名或 resource 短名；重名（凌霄城/念萝坝/星垂城）用短名；或 config|resource 直填")
	return b.String()
}
