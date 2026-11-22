package fssoloworld

// scenepresets.go GM 场景切换目的地解析：location.ini BornCJ 出生点表
//（rolestate 内嵌镜像，BirthSceneOptions 导出）+ 中文别名 + 直填形态。

import (
	"fmt"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// sceneAliases 中文别名 → resource 短名（BornCJ 键域）。主城/出生镇/八大门派
// 及主要江湖势力（坐标取各自场景 BornCJ 出生点）。
var sceneAliases = map[string]string{
	"燕京": "city01", "苏州": "city02", "金陵": "city03", "洛阳": "city04", "成都": "city05",
	"鸡鸣驿": "born01", "烟雨庄": "born03", "千灯镇": "born04", "恶人谷": "born02",
	"锦衣卫": "school01", "丐帮": "school02", "君子堂": "school03", "极乐谷": "school04",
	"唐门": "school05", "峨眉": "school06", "武当": "school07", "少林": "school08",
	"移花宫":   "school09",
	"幽云十六州": "scene01", "蓬莱仙境": "scene02", "泛黄平原": "scene03", "襄阳坡": "scene04",
	"捻罗坝": "scene05", "凌霄城": "scene06", "梅花岭": "scene07", "塞外大漠": "scene08",
	"塞北草原": "scene09",
}

// sceneDestination 切换目的地（config 全路径 + resource 短名 + 落点）。
type sceneDestination struct {
	Config   string
	Resource string
	Position [4]float64
}

// resolveSceneDestination key 三形态：中文别名 / resource 短名（born03、
// city01…，落点=BornCJ 出生点）/ "config|resource" 直填（须显式坐标，
// 落点 [4]float64 零值由调用方覆盖）。未解析 false。
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
	if opt, ok := rolestate.FindSceneOption(resourceKey); ok {
		return sceneDestination{Config: opt.Config, Resource: opt.Key, Position: opt.Position}, true
	}
	return sceneDestination{}, false
}

// scenePresetKeys 可用 key 清单（echo `gm scene` 裸调用打印； BornCJ 键域
// 按序 + 别名提示）。
func scenePresetKeys() string {
	opts := rolestate.BirthSceneOptions()
	keys := make([]string, 0, len(opts))
	for _, opt := range opts {
		keys = append(keys, opt.Key)
	}
	return fmt.Sprintf("keys: %s\naliases: 燕京 苏州 金陵 洛阳 成都 鸡鸣驿 烟雨庄 千灯镇 恶人谷 锦衣卫 丐帮 君子堂 极乐谷 唐门 峨眉 武当 少林 移花宫 …（或 config|resource 直填）",
		strings.Join(keys, " "))
}
