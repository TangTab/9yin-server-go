package npcfeed

// source_parity_test.go 真档对拍：数据根同时带客户端 share 树与 XML 构建
// 产物时，两条来源读同一份数据必须给出逐行相同的结果——这是「直读替换
// XML 链」的验收口径，也是键名 / 取值口径不漂移的回归锚（以后谁改了
// 直读的列映射，这里先红）。真档不随 git（testdata 层文件）时跳过。

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

// bothSources 报告数据根是否同时带某一域的 share 源与 XML 产物。
func bothSources(t *testing.T, root, sharePath, xmlPath string) bool {
	t.Helper()
	if root == "" {
		t.Skip("数据根缺席（真档不随仓）")
	}
	shareOK, xmlOK := false, false
	if _, err := res.Stat(filepath.Join(root, filepath.FromSlash(sharePath))); err == nil {
		shareOK = true
	}
	if _, err := res.Stat(filepath.Join(root, filepath.FromSlash(xmlPath))); err == nil {
		xmlOK = true
	}
	if !shareOK || !xmlOK {
		t.Skipf("对拍需要两种来源同在：share %s=%v xml %s=%v", sharePath, shareOK, xmlPath, xmlOK)
	}
	return true
}

func subDirs(root, rel string) []string {
	entries, err := res.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, strings.ToLower(e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// TestPlacementSourceParity 摆放域：全部场景的摆放属性行（含随机组展开行）
// 与过滤后的 Placement 列表，直读与 XML 产物逐行相同。
func TestPlacementSourceParity(t *testing.T) {
	root := ResolveRoot()
	bothSources(t, root, shareCreatorDir, "ini/Scene")

	shareScenes, xmlScenes := subDirs(root, shareCreatorDir), subDirs(root, "ini/Scene")
	if len(shareScenes) == 0 {
		t.Skip("share 摆放树无场景目录")
	}
	if !reflect.DeepEqual(shareScenes, xmlScenes) {
		t.Fatalf("两棵摆放树的场景目录集不同：share %d 个 / xml %d 个\n只在 share：%v\n只在 xml：%v",
			len(shareScenes), len(xmlScenes), notIn(xmlScenes, shareScenes), notIn(shareScenes, xmlScenes))
	}
	entries, err := res.ReadDir(filepath.Join(root, filepath.FromSlash(shareCreatorDir)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	totalRows, totalKept := 0, 0
	for _, dir := range names {
		xmlRows, err := placementRows(root, dir, sharedata.SourceXML)
		if err != nil {
			t.Fatalf("%s xml: %v", dir, err)
		}
		shareRows, err := placementRows(root, dir, sharedata.SourceShare)
		if err != nil {
			t.Fatalf("%s share: %v", dir, err)
		}
		if !reflect.DeepEqual(xmlRows, shareRows) {
			t.Fatalf("%s 摆放行不同：xml %d 行 / share %d 行\n首个不同：%s",
				dir, len(xmlRows), len(shareRows), firstRowDiff(xmlRows, shareRows))
		}
		xmlKept, err := loadPlacements(root, dir, sharedata.SourceXML)
		if err != nil {
			t.Fatal(err)
		}
		shareKept, err := loadPlacements(root, dir, sharedata.SourceShare)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(xmlKept, shareKept) {
			t.Fatalf("%s 过滤后摆放不同：xml %d / share %d", dir, len(xmlKept), len(shareKept))
		}
		totalRows += len(shareRows)
		totalKept += len(shareKept)
	}
	t.Logf("摆放对拍一致：场景 %d / 属性行 %d / 白名单有坐标 %d", len(names), totalRows, totalKept)
}

func notIn(list, other []string) []string {
	var out []string
	for _, s := range list {
		found := false
		for _, o := range other {
			if s == o {
				found = true
				break
			}
		}
		if !found && len(out) < 12 {
			out = append(out, s)
		}
	}
	return out
}

// firstRowDiff 定位第一处不同（行号 + 差异的键）。
func firstRowDiff(a, b []map[string]string) string {
	for i := 0; i < len(a) && i < len(b); i++ {
		if reflect.DeepEqual(a[i], b[i]) {
			continue
		}
		keys := map[string]bool{}
		for k := range a[i] {
			keys[k] = true
		}
		for k := range b[i] {
			keys[k] = true
		}
		var sorted []string
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			if a[i][k] != b[i][k] {
				return fmt.Sprintf("第 %d 行 %s：xml=%q share=%q（id xml=%q share=%q）",
					i, k, a[i][k], b[i][k], a[i]["id"], b[i]["id"])
			}
		}
		return fmt.Sprintf("第 %d 行（键集不同：xml %d 键 / share %d 键）", i, len(a[i]), len(b[i]))
	}
	if len(a) != len(b) {
		return fmt.Sprintf("行数不同：第 %d 行起只有一侧有", min(len(a), len(b)))
	}
	return "无差异"
}

// TestTemplateSourceParity 模板 + 战斗域：全部模板行（含全部属性）与战斗行，
// 直读 share 与 XML 产物逐行相同。
func TestTemplateSourceParity(t *testing.T) {
	root := ResolveRoot()
	bothSources(t, root, shareNPCConfigDir, "ini/Npc/NpcConfig")
	bothSources(t, root, shareNpcTemplateFile, "ini/Npc/NpcTemplate.xml")

	xmlTemplates, xmlCombat, err := loadStoreSource(root, sharedata.SourceXML, sharedata.SourceXML)
	if err != nil {
		t.Fatal("xml:", err)
	}
	shareTemplates, shareCombat, err := loadStoreSource(root, sharedata.SourceShare, sharedata.SourceShare)
	if err != nil {
		t.Fatal("share:", err)
	}
	if !reflect.DeepEqual(xmlCombat, shareCombat) {
		t.Fatalf("战斗行不同：xml %d 行 / share %d 行\n%s", len(xmlCombat), len(shareCombat), firstRowDiffRows(xmlCombat, shareCombat))
	}
	drift, staleBorn := templateDrift(xmlTemplates, shareTemplates)
	if len(drift) > 0 {
		t.Fatalf("模板行不同：xml %d 行 / share %d 行\n%s", len(xmlTemplates), len(shareTemplates),
			templateDiff(xmlTemplates, shareTemplates, drift))
	}
	t.Logf("模板对拍一致：模板 %d / 战斗 %d（其中 %d 行直读多出 BornAction——层内 XML 落后于构建器的已知差异）",
		len(shareTemplates), len(shareCombat), staleBorn)
}

// templateDrift 逐行比较，返回真正漂移的 ID 集与「已知差异」行数。唯一
// 放行的形态：直读行比产物行多一个 BornAction、其余全同——层内 XML 可以
// 落后于工作区的构建器映射（出生姿态那次加的 BornAction 直列绑定，当时
// 有意没重建数据），而直读用的是当前映射（npcColumnBindings）。产物重建
// 后这一形态自然消失，放行条件也就不再命中。
func templateDrift(xml, share map[string]TemplateRow) (drift map[string]bool, staleBorn int) {
	drift = map[string]bool{}
	for id, s := range share {
		x, ok := xml[id]
		if !ok {
			drift[id] = true
			continue
		}
		if reflect.DeepEqual(x, s) {
			continue
		}
		if _, has := s.Attrs["BornAction"]; has && x.Attrs["BornAction"] == "" && s.Script == x.Script {
			trimmed := make(map[string]string, len(s.Attrs)-1)
			for k, v := range s.Attrs {
				if k != "BornAction" {
					trimmed[k] = v
				}
			}
			if reflect.DeepEqual(trimmed, x.Attrs) {
				staleBorn++
				continue
			}
		}
		drift[id] = true
	}
	for id := range xml {
		if _, ok := share[id]; !ok {
			drift[id] = true
		}
	}
	return drift, staleBorn
}

// templateDiff 模板表差异摘要（只列 only 里的 ID）：只在一侧的 ID，以及
// 属性不同的 ID 按「差异键形态」（只在 xml / 只在 share / 值不同）归类。
func templateDiff(xml, share map[string]TemplateRow, only map[string]bool) string {
	onlyXML, onlyShare := []string{}, []string{}
	shapes := map[string][]string{}
	for id := range only {
		x, inXML := xml[id]
		s, inShare := share[id]
		switch {
		case !inXML:
			onlyShare = append(onlyShare, id)
		case !inShare:
			onlyXML = append(onlyXML, id)
		default:
			shape := rowShapeDiff(x.Attrs, s.Attrs)
			if x.Script != s.Script {
				shape += " script=" + x.Script + "/" + s.Script
			}
			shapes[shape] = append(shapes[shape], id)
		}
	}
	sort.Strings(onlyXML)
	sort.Strings(onlyShare)
	var sb strings.Builder
	fmt.Fprintf(&sb, "只在 xml %d 个 %v\n只在 share %d 个 %v\n属性不同 %d 个：\n",
		len(onlyXML), clip(onlyXML), len(onlyShare), clip(onlyShare), len(only)-len(onlyXML)-len(onlyShare))
	shapesSorted := make([]string, 0, len(shapes))
	for shape := range shapes {
		shapesSorted = append(shapesSorted, shape)
	}
	sort.Strings(shapesSorted)
	for _, shape := range shapesSorted {
		ids := shapes[shape]
		sort.Strings(ids)
		fmt.Fprintf(&sb, "  [%d] %s：%v\n", len(ids), shape, clip(ids))
	}
	return sb.String()
}

// rowShapeDiff 两行属性的差异形态（"" = 相同）：只在 a / 只在 b / 值不同的键。
func rowShapeDiff(a, b map[string]string) string {
	var onlyA, onlyB, value []string
	for k, va := range a {
		vb, ok := b[k]
		switch {
		case !ok:
			onlyA = append(onlyA, k+"="+va)
		case va != vb:
			value = append(value, k)
		}
	}
	for k, vb := range b {
		if _, ok := a[k]; !ok {
			onlyB = append(onlyB, k+"="+vb)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	sort.Strings(value)
	var parts []string
	if len(onlyA) > 0 {
		parts = append(parts, "只在 xml:"+strings.Join(onlyA, ","))
	}
	if len(onlyB) > 0 {
		parts = append(parts, "只在 share:"+strings.Join(onlyB, ","))
	}
	if len(value) > 0 {
		parts = append(parts, "值不同:"+strings.Join(value, ","))
	}
	return strings.Join(parts, " ")
}

// firstRowDiffRows CombatRow 表的第一处不同。
func firstRowDiffRows(xml, share map[string]CombatRow) string {
	for id, x := range xml {
		if s, ok := share[id]; !ok || !reflect.DeepEqual(x, s) {
			return fmt.Sprintf("ID %s：xml=%v share=%v", id, x.Attrs, s.Attrs)
		}
	}
	for id, s := range share {
		if _, ok := xml[id]; !ok {
			return fmt.Sprintf("ID %s 只在 share：%v", id, s.Attrs)
		}
	}
	return "无差异"
}

func clip(list []string) []string {
	if len(list) > 12 {
		return append(list[:12], "…")
	}
	return list
}

// TestNPCConfigTableOrderMatchesXMLTree 表序（决定同 ID 多表时哪一行胜出）
// 与 XML 产物树的遍历序逐位一致——登记串大小写、未登记表的镜像路径、
// 目录并档这些细节都由这条兜住。
func TestNPCConfigTableOrderMatchesXMLTree(t *testing.T) {
	root := ResolveRoot()
	bothSources(t, root, sharedata.NPCConfigDir, "ini/Npc/NpcConfig")

	tables, err := sharedata.NPCConfigTables(root)
	if err != nil {
		t.Fatal(err)
	}
	xmlDir := filepath.Join(root, "ini", "Npc", "NpcConfig")
	var xmlOrder []string
	_ = res.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(d.Name()), ".xml") {
			xmlOrder = append(xmlOrder, "NpcConfig/"+filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(path), filepath.ToSlash(xmlDir)+"/")))
		}
		return nil
	})
	if len(tables) != len(xmlOrder) {
		t.Fatalf("表数不同：直读 %d / xml %d", len(tables), len(xmlOrder))
	}
	for i := range tables {
		if !strings.EqualFold(tables[i].ServerRel, xmlOrder[i]) {
			t.Fatalf("第 %d 位表不同：直读 %s（%s） vs xml %s", i+1, tables[i].ServerRel, tables[i].Rel, xmlOrder[i])
		}
	}
	t.Logf("表序一致：%d 张", len(tables))
}

// TestCatalogSourceParity 端到端：全部场景的联查目录（摆放 → 模板 → 战斗，
// 即物化帧的输入）在两种来源下逐个 NPC 相同。允许的唯一差异是直读行多出
// 的 BornAction（见 templateDrift）——落到 NPC 上就是静息姿态。
func TestCatalogSourceParity(t *testing.T) {
	root := ResolveRoot()
	bothSources(t, root, shareCreatorDir, "ini/Scene")
	bothSources(t, root, shareNPCConfigDir, "ini/Npc/NpcConfig")
	bothSources(t, root, shareNpcTemplateFile, "ini/Npc/NpcTemplate.xml")

	scenes, err := SceneDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NINEYIN_DATA_SOURCE", "xml")
	ResetStore()
	t.Cleanup(ResetStore) // 还原 env 后清掉 share 态的进程级缓存
	xmlCatalog := catalogSnapshot(t, root, scenes)

	t.Setenv("NINEYIN_DATA_SOURCE", "share")
	ResetStore()
	shareCatalog := catalogSnapshot(t, root, scenes)

	total, keyDiff, rawDiff, poseDiff, other := 0, 0, 0, 0, 0
	var samples []string
	for _, scene := range scenes {
		a, b := xmlCatalog[scene], shareCatalog[scene]
		if len(a) != len(b) {
			t.Fatalf("%s 物化数不同：xml %d / share %d", scene, len(a), len(b))
		}
		for i := range a {
			total++
			x, s := &a[i], &b[i]
			if reflect.DeepEqual(x, s) {
				continue
			}
			// 已知差异：直读的模板行多一个 BornAction（templateDrift）。它按
			// 摆放 > 模板 BornAction > 模板 BornActionRec 参与出生状态，所以
			// 这条 NPC 的 BornAction 可以跟着变；其余字段必须全同。
			if _, known := knownBornActionDiff(x.Template, s.Template); known {
				tplX, tplS := x.Template, s.Template
				bornX, bornS := x.BornAction, s.BornAction
				x.Template, s.Template = nil, nil
				x.BornAction, s.BornAction = "", ""
				same := reflect.DeepEqual(x, s)
				x.Template, s.Template, x.BornAction, s.BornAction = tplX, tplS, bornX, bornS
				if same {
					keyDiff++
					if bornX != bornS {
						rawDiff++
					}
					if x.RestAction() != s.RestAction() {
						poseDiff++
						if len(samples) < 8 {
							samples = append(samples, fmt.Sprintf("%s/%s:%s 静息姿态 %q→%q", scene, s.File, s.ConfigID, x.RestAction(), s.RestAction()))
						}
					}
					continue
				}
			}
			other++
			if len(samples) < 8 {
				samples = append(samples, fmt.Sprintf("%s/%s:%s 其它差异：模板{%s} 物化帧输入差异{%s}",
					scene, s.File, s.ConfigID, rowShapeDiff(x.Template, s.Template), spawnShapeDiff(x, s)))
			}
		}
	}
	if other > 0 {
		t.Fatalf("%d 个 NPC 在两种来源下不同（且不是 BornAction 已知差异）：%v", other, samples)
	}
	t.Logf("目录对拍一致：场景 %d / NPC %d；其中 %d 个的模板行多出 BornAction（出生状态原始值不同 %d 个，静息姿态不同 %d 个）：%v",
		len(scenes), total, keyDiff, rawDiff, poseDiff, samples)
}

// knownBornActionDiff 两条模板行是否只差「直读行多一个 BornAction」；是则
// 返回该值。
func knownBornActionDiff(xml, share map[string]string) (string, bool) {
	born, has := share["BornAction"]
	if !has || xml["BornAction"] != "" {
		return "", false
	}
	trimmed := make(map[string]string, len(share)-1)
	for k, v := range share {
		if k != "BornAction" {
			trimmed[k] = v
		}
	}
	return born, reflect.DeepEqual(trimmed, xml)
}

// spawnShapeDiff 两条 Spawn 的差异字段（坐标 / 类名 / 战斗行）。
func spawnShapeDiff(x, s *Spawn) string {
	var parts []string
	if x.ConfigID != s.ConfigID {
		parts = append(parts, "ConfigID "+x.ConfigID+"/"+s.ConfigID)
	}
	if x.File != s.File {
		parts = append(parts, "File "+x.File+"/"+s.File)
	}
	if x.Script != s.Script {
		parts = append(parts, "Script "+x.Script+"/"+s.Script)
	}
	if x.PathID != s.PathID {
		parts = append(parts, "PathID "+x.PathID+"/"+s.PathID)
	}
	if x.X != s.X || x.Y != s.Y || x.Z != s.Z || x.Orient != s.Orient {
		parts = append(parts, fmt.Sprintf("坐标 %v/%v", [4]float32{x.X, x.Y, x.Z, x.Orient}, [4]float32{s.X, s.Y, s.Z, s.Orient}))
	}
	if !reflect.DeepEqual(x.Combat, s.Combat) {
		parts = append(parts, "战斗行")
	}
	if len(parts) == 0 {
		return "无"
	}
	return strings.Join(parts, "；")
}

// catalogSnapshot 全场景联查目录快照（固定段 ident；不绑巡逻——巡逻与
// 数据来源无关）。
func catalogSnapshot(t *testing.T, root string, scenes []string) map[string][]Spawn {
	t.Helper()
	out := make(map[string][]Spawn, len(scenes))
	for _, scene := range scenes {
		list, _, err := BuildCatalogDir(root, scene)
		if err != nil {
			t.Fatalf("%s: %v", scene, err)
		}
		out[scene] = list
	}
	return out
}

// TestSceneDirsFollowSource 场景目录枚举跟随来源：显式 xml 列 ini/Scene，
// 显式 share 列 share/creator/npc_creator；目录名集合相同。
func TestSceneDirsFollowSource(t *testing.T) {
	root := ResolveRoot()
	bothSources(t, root, shareCreatorDir, "ini/Scene")

	t.Setenv("NINEYIN_DATA_SOURCE", "xml")
	xmlDirs, err := SceneDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if sceneRootDir(root) != filepath.Join(root, "ini", "Scene") {
		t.Fatalf("xml 模式场景根 %s", sceneRootDir(root))
	}
	t.Setenv("NINEYIN_DATA_SOURCE", "share")
	shareDirs, err := SceneDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if sceneRootDir(root) != filepath.Join(root, filepath.FromSlash(shareCreatorDir)) {
		t.Fatalf("share 模式场景根 %s", sceneRootDir(root))
	}
	if len(xmlDirs) == 0 || !reflect.DeepEqual(lowerAll(xmlDirs), lowerAll(shareDirs)) {
		t.Fatalf("两种来源的场景目录集不同：xml %d / share %d", len(xmlDirs), len(shareDirs))
	}
	// 场景键 → 目录名的两条查询在两种来源下同解。
	for _, scene := range []string{"born03", "city03", "school07", "born01"} {
		t.Setenv("NINEYIN_DATA_SOURCE", "xml")
		xmlDir := SceneDirOf(root, scene)
		t.Setenv("NINEYIN_DATA_SOURCE", "share")
		if shareDir := SceneDirOf(root, scene); shareDir == "" || shareDir != xmlDir {
			t.Fatalf("场景 %s 解析不同：xml=%q share=%q", scene, xmlDir, shareDir)
		}
		if CanonicalSceneDir(root, xmlDir) != xmlDir {
			t.Fatalf("share 模式下权威目录解析失败：%s", xmlDir)
		}
	}
}

func lowerAll(list []string) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = strings.ToLower(s)
	}
	sort.Strings(out)
	return out
}
