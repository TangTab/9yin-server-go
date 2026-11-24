package sharedata

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gbk(t *testing.T, s string) string {
	t.Helper()
	out, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// 编码自适应：UTF-8 直通、BOM 剥离、GBK 解码。
func TestDecodeText(t *testing.T) {
	utf8Text := "名称\t值\n"
	if got, err := DecodeText([]byte(utf8Text)); err != nil || string(got) != utf8Text {
		t.Fatalf("UTF-8 直通失败: %q %v", got, err)
	}
	if got, _ := DecodeText(append([]byte("\xEF\xBB\xBF"), utf8Text...)); string(got) != utf8Text {
		t.Fatalf("BOM 未剥离: %q", got)
	}
	if got, _ := DecodeText([]byte(gbk(t, "马六\n"))); string(got) != "马六\n" {
		t.Fatalf("GBK 解码失败: %q", got)
	}
}

// 3 行头严格表：GBK、排版空行跳过、带引号内嵌换行的字段不按行硬切、
// 同名列取首见；头宽不一致与数据行过窄报错。
func TestReadTemplateTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "npc.txt")
	write(t, path, gbk(t, "编号\t名字\t脚本\t名字\n"+
		"STRING\tSTRING\tSTRING\tSTRING\n"+
		"ID\tName\tscript\tName\n"+
		"\"//v01,排版行\"\t\t\t\n"+
		"\n"+
		"N1\t马六\tCommonNpc\t马六2\n"+
		"N2\t\"带\n换行\"\tFuncNpc\t\n"))
	table, err := ReadTemplateTable(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"ID", "Name", "script", "Name"}; !reflect.DeepEqual(table.Columns, want) {
		t.Fatalf("列键 %v，应为 %v", table.Columns, want)
	}
	if table.Comments[0] != "编号" || table.Types[0] != "STRING" {
		t.Fatalf("注释/类型行解码错: %v %v", table.Comments, table.Types)
	}
	// 排版行（首格是 "//v01,…" 说明、其余格空）ID 格非空，照常成行——
	// XML 产物里也有这种行（没有摆放引用它），直读保持一致。
	if len(table.Rows) != 3 {
		t.Fatalf("数据行 %d，应为 3（排版说明行 1 + 空行跳过 + 数据 2）", len(table.Rows))
	}
	// csv 把引号字段里的 \r\n / \n 规范成 \n。
	if got := table.Rows[2][1]; got != "带\n换行" {
		t.Fatalf("内嵌换行字段 %q", got)
	}
	if table.Column("Name") != 1 || table.Column("Nope") != -1 {
		t.Fatalf("同名列应取首见、无此列 -1：%d %d", table.Column("Name"), table.Column("Nope"))
	}

	bad := filepath.Join(t.TempDir(), "bad.txt")
	write(t, bad, gbk(t, "注释A\t注释B\n类型A\t类型B\nID\tName\tx\n"))
	if _, err := ReadTemplateTable(bad); err == nil {
		t.Fatal("头宽不一致应报错")
	}
	narrow := filepath.Join(t.TempDir(), "narrow.txt")
	write(t, narrow, gbk(t, "a\tb\nc\td\nID\tName\nN1\n"))
	if _, err := ReadTemplateTable(narrow); err == nil {
		t.Fatal("数据行窄于列宽应报错")
	}
	if _, err := ReadTemplateTable(filepath.Join(t.TempDir(), "absent.txt")); !os.IsNotExist(err) {
		t.Fatalf("缺席档应报 not-exist，得 %v", err)
	}
}

// 2 行头宽松表：列键裁空白并裁尾部空列键、短行补齐、超宽截断（多余字段
// 带值才计）；1 行头表无类型行。
func TestReadDomainTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.txt")
	write(t, path, "int\tint\t\tstr\t\nID\tCount\t\t Name\t\n1\t\t\t甲\n2\t5\t\t乙\t\t多余\n3\t7\t\t丙\t\t\t\n")
	table, err := ReadDomainTable(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"ID", "Count", "", "Name"}; !reflect.DeepEqual(table.Columns, want) {
		t.Fatalf("列键 %v，应为 %v", table.Columns, want)
	}
	if len(table.Types) != 4 {
		t.Fatalf("类型行应裁到列宽：%v", table.Types)
	}
	if table.Truncated != 1 {
		t.Fatalf("超宽截断行 %d，应为 1（只有带值的多余字段才算）", table.Truncated)
	}
	if got := table.Rows[0]; !reflect.DeepEqual(got, []string{"1", "", "", "甲"}) {
		t.Fatalf("短行应补齐：%q", got)
	}
	if got := table.Rows[1]; !reflect.DeepEqual(got, []string{"2", "5", "", "乙"}) {
		t.Fatalf("超宽行应截到列宽：%q", got)
	}

	one := filepath.Join(t.TempDir(), "npctemplate.tab")
	write(t, one, "ID\tMaxLife\n1\t100\n")
	tab, err := ReadDomainTable(one, 1)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Types != nil || tab.Column("MaxLife") != 1 || tab.Rows[0][1] != "100" {
		t.Fatalf("1 行头表解析错: %+v", tab)
	}
	empty := filepath.Join(t.TempDir(), "empty.txt")
	write(t, empty, "int\n \t \n")
	if _, err := ReadDomainTable(empty, 2); err == nil {
		t.Fatal("无列键应报错")
	}
}

// creator 摆放展开：静态项 + 随机组逐点 / 无点保留、eventtrigger 与非 xml
// 排除、各档按文件名排序、空值属性不收、File 溯源、坏档跳过不废场景、
// GBK 档自适应。
func TestReadCreatorDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "born03_yanyuzhuang")
	write(t, filepath.Join(dir, "commonnpc.xml"), `<?xml version="1.0" encoding="utf-8" ?>
<object>
    <staticcreator>
        <item No="A" x="1.5" y="2" z="3.25" ay="4.6" id="Npc1" BornAction="standfree01" desc="" />
        <item No="B" x="10" y="2" z="30" id="Npc2" />
    </staticcreator>
</object>
`)
	write(t, filepath.Join(dir, "gathernpc_creator.xml"), `<?xml version="1.0" encoding="utf-8" ?>
<object>
    <randomcreator>
        <creator No="C1" x="249.988" y="2.677" z="1027.186" radius="6.0" WakeupRange="31.0">
            <item No="1" id="Gather6" ay="-1" amount="6" desc="三点金草">
                <position x="253.294" z="1027.405" ay="-1" />
                <position x="253.753" z="1024.350" ay="9" />
            </item>
            <item No="2" id="Gather7" amount="3" />
        </creator>
        <creator No="C2" x="1" y="2" z="3" radius="6.0">
            <item No="1" id="Gather8"><position x="5" z="6" /></item>
        </creator>
    </randomcreator>
</object>
`)
	write(t, filepath.Join(dir, "eventtrigger.xml"), `<object><staticcreator><item x="9" id="Trigger"/></staticcreator></object>`)
	write(t, filepath.Join(dir, "file.ini"), "[file]\n0=CommonNpc.xml\n")
	write(t, filepath.Join(dir, "broken.xml"), `<object><staticcreator><item x="1" id="K" <bad/></staticcreator></object>`)

	rows, files, skipped, err := ReadCreatorDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 || len(skipped) != 1 || !strings.HasPrefix(skipped[0], "broken.xml") {
		t.Fatalf("files=%d skipped=%v（坏档应跳过 1，eventtrigger / ini 不计）", files, skipped)
	}
	if len(rows) != 6 {
		t.Fatalf("行数 %d，应为 6（静态 2 + 逐点 2 + 无点 1 + 第二组逐点 1）", len(rows))
	}
	first := rows[0]
	if first["File"] != "commonnpc.xml" || first["id"] != "Npc1" || first["x"] != "1.5" || first["z"] != "3.25" {
		t.Fatalf("静态项解析错: %v", first)
	}
	if _, has := first["desc"]; has {
		t.Fatal("空值属性不应入行（与 XML 产物一致）")
	}
	// 随机项逐点：creator 与 item 属性合并、position 坐标并入、No=组:项:序。
	p1 := rows[2]
	want := map[string]string{
		"File": "gathernpc_creator.xml", "CreatorNo": "C1", "No": "C1:1:1", "Random": "1",
		"x": "253.294", "y": "2.677", "z": "1027.405", "ay": "-1", "radius": "6.0", "WakeupRange": "31.0",
		"id": "Gather6", "amount": "6", "desc": "三点金草",
	}
	if !reflect.DeepEqual(p1, want) {
		t.Fatalf("随机项第 1 点:\n 得 %v\n 应 %v", p1, want)
	}
	if rows[3]["No"] != "C1:1:2" || rows[3]["ay"] != "9" || rows[3]["z"] != "1024.350" {
		t.Fatalf("随机项第 2 点（position 坐标与 ay 覆盖 item 的 ay）: %v", rows[3])
	}
	// 无 position 的随机项：单行保留，No=组:项，creator 的 y/z 随行。
	p3 := rows[4]
	if p3["No"] != "C1:2" || p3["id"] != "Gather7" || p3["amount"] != "3" || p3["y"] != "2.677" {
		t.Fatalf("无点随机项: %v", p3)
	}
	// 第二组只有一个点：随机段随 </creator> 结束，下一组重新开始。
	if p4 := rows[5]; p4["No"] != "C2:1:1" || p4["x"] != "5" || p4["z"] != "6" || p4["y"] != "2" {
		t.Fatalf("第二组逐点项: %v", p4)
	}

	// GBK 档（声明 utf-8 但字节是 GBK）：自适应解码。
	gbkDir := filepath.Join(t.TempDir(), "scene")
	write(t, filepath.Join(gbkDir, "funcnpc.xml"), gbk(t, `<?xml version="1.0" encoding="utf-8" ?><object><staticcreator><item x="1" id="F1" desc="车夫"/></staticcreator></object>`))
	gr, _, skipped, err := ReadCreatorDir(gbkDir)
	if err != nil || len(skipped) != 0 || len(gr) != 1 || gr[0]["desc"] != "车夫" {
		t.Fatalf("GBK 摆放档: rows=%v skipped=%v err=%v", gr, skipped, err)
	}
	if _, _, _, err := ReadCreatorDir(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("目录缺席应报错")
	}
}

// 键值行档：注释、首个 '=' 为界、重复键首见胜出、目录级跨档首见胜出。
func TestReadIDres(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "stringtask_normal")
	write(t, filepath.Join(dir, "a.idres"), "title_1=初入江湖\n; 注释\n// 注释2\ntitle_2=a=b\n\n bad\n title_1 = 重复\n")
	write(t, filepath.Join(dir, "b.idres"), "\xEF\xBB\xBFtitle_1=后来者\ntitle_3=第三\n")
	entries, files, conflicts, err := ReadIDresDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"title_1": "初入江湖", "title_2": "a=b", "title_3": "第三"}
	if files != 2 || conflicts != 2 || !reflect.DeepEqual(entries, want) {
		t.Fatalf("files=%d conflicts=%d entries=%v", files, conflicts, entries)
	}
}

// SortWalkOrder 等价目录树遍历序：每个目录的直接子项（文件与子目录混排）
// 按名字字节序、深度优先——大小写参与排序，且目录名比较时不带分隔符
// （"Fun.xml" 排在子目录 "FunNpc" 之前）。
func TestSortWalkOrder(t *testing.T) {
	paths := []string{
		"NpcConfig/WorldNpc/city.xml", "NpcConfig/worldnpc/x.xml", "NpcConfig/WorldNpc/FunNpc/Box.xml",
		"NpcConfig/CloneNpc/a.xml", "NpcConfig/WorldNpc/NewWorld.xml", "NpcConfig/WorldNpc/Fun.xml",
	}
	SortWalkOrder(paths, func(i int) string { return paths[i] })
	want := []string{
		"NpcConfig/CloneNpc/a.xml", "NpcConfig/WorldNpc/Fun.xml", "NpcConfig/WorldNpc/FunNpc/Box.xml",
		"NpcConfig/WorldNpc/NewWorld.xml", "NpcConfig/WorldNpc/city.xml", "NpcConfig/worldnpc/x.xml",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("遍历序 %v\n应为 %v", paths, want)
	}
}

// splitServerNPCPath 只认 ini\Npc\…\X.txt 形态的登记值。
func TestSplitServerNPCPath(t *testing.T) {
	server, client, ok := splitServerNPCPath(`ini\Npc\NpcConfig\WorldNpc\city_CommonNpc.txt`)
	if !ok || server != "NpcConfig/WorldNpc/city_CommonNpc.xml" || client != "WorldNpc/city_CommonNpc.txt" {
		t.Fatalf("got %q %q %v", server, client, ok)
	}
	for _, bad := range []string{`ini\Scene\x.txt`, `ini\Npc\NpcConfig\A.xlsx`, `share\npc\x.txt`} {
		if _, _, ok := splitServerNPCPath(bad); ok {
			t.Fatalf("%q 不应识别为表登记", bad)
		}
	}
}

// NPCConfigTables：npc_type.txt 不算表；产物路径以登记为准、未登记按镜像；
// 登记串里目录大小写不统一时取首次出现的大小写（构建器在不分大小写的
// 文件系统上落盘的形态）；清单缺席退回客户端树遍历序。
func TestNPCConfigTables(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "share", "npc", "npcconfig")
	body := "注释A\t注释B\n类型A\t类型B\nID\tName\nN1\t甲\n"
	for _, name := range []string{"city_commonnpc.txt", "newschoolnpc.txt", "other_commonnpc.txt", "unregistered.txt", "npc_type.txt"} {
		write(t, filepath.Join(dir, "worldnpc", name), body)
	}
	write(t, filepath.Join(dir, "npc_type.ini"), gbk(t, "//文件数据来源\n[file]\nflie1=ini\\Npc\\NpcConfig\\WorldNpc\\city_CommonNpc.txt\r\n"+
		"flie2=ini\\Npc\\npcConfig\\WorldNpc\\NewSchoolNpc.txt\r\n"+
		"flie3=ini\\Npc\\NpcConfig\\WorldNpc\\other_CommonNpc.txt\r\n"+
		"flie4=不是表登记\r\n"))

	tables, err := NPCConfigTables(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tb := range tables {
		got = append(got, tb.Rel+" → "+tb.ServerRel)
	}
	want := []string{
		"worldnpc/newschoolnpc.txt → NpcConfig/WorldNpc/NewSchoolNpc.xml",
		"worldnpc/city_commonnpc.txt → NpcConfig/WorldNpc/city_CommonNpc.xml",
		"worldnpc/other_commonnpc.txt → NpcConfig/WorldNpc/other_CommonNpc.xml",
		"worldnpc/unregistered.txt → NpcConfig/WorldNpc/unregistered.xml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("表清单 / 产物路径 / 装载序：\n 得 %v\n 应 %v", got, want)
	}

	// 清单缺席：镜像路径（保留磁盘大小写）+ 客户端树遍历序。
	if err := os.Remove(filepath.Join(dir, "npc_type.ini")); err != nil {
		t.Fatal(err)
	}
	tables, err = NPCConfigTables(root)
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, tb := range tables {
		got = append(got, tb.Rel)
	}
	if want := []string{"worldnpc/city_commonnpc.txt", "worldnpc/newschoolnpc.txt", "worldnpc/other_commonnpc.txt", "worldnpc/unregistered.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("无清单时的遍历序 %v，应为 %v", got, want)
	}
	if _, err := NPCConfigTables(filepath.Join(root, "absent")); err == nil {
		t.Fatal("目录缺席应报错")
	}
}

// 来源开关：显式 xml 照办；显式 share / auto 时目录在即直读、不在回落 XML。
func TestEffectiveSource(t *testing.T) {
	root := t.TempDir()
	shareDir := filepath.Join(root, "share", "creator", "npc_creator")
	t.Setenv(sourceEnv, "")
	if got := EffectiveSource(root, "share/creator/npc_creator"); got != SourceXML {
		t.Fatalf("auto + 目录缺席应回落 XML，得 %v", got)
	}
	if err := os.MkdirAll(shareDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := EffectiveSource(root, "share/creator/npc_creator"); got != SourceShare {
		t.Fatalf("auto + 目录在应直读，得 %v", got)
	}
	t.Setenv(sourceEnv, "share")
	if got := EffectiveSource(root, "share/npc/npcconfig"); got != SourceXML {
		t.Fatalf("显式 share + 源缺席应回落 XML，得 %v", got)
	}
	// 源可以是单个文件（战斗模板 share/task/npctemplate.tab）。
	write(t, filepath.Join(root, "share", "task", "npctemplate.tab"), "ID\n")
	if got := EffectiveSource(root, "share/task/npctemplate.tab"); got != SourceShare {
		t.Fatalf("显式 share + 源文件在应直读，得 %v", got)
	}
	t.Setenv(sourceEnv, "XML")
	if got := EffectiveSource(root, "share/creator/npc_creator"); got != SourceXML {
		t.Fatalf("显式 xml 应照办，得 %v", got)
	}
	t.Setenv(sourceEnv, "whatever")
	if got := SourceFromEnv(); got != SourceAuto {
		t.Fatalf("未知取值应为 auto，得 %v", got)
	}
	if SourceShare.String() != "share" || SourceXML.String() != "xml" || SourceAuto.String() != "auto" {
		t.Fatal("Source.String() 方言")
	}
}

// 旧 share 保险：share 摆放树与 XML 产物并存时，share 缺的场景比多的多
// （产物按新树建、share 是旧快照）→ auto 回落 XML 并给出说明；显式 share
// 照办；share 不比产物旧（相等、或 share 更多）照常直读；只有一种来源
// 不判；数据根缺席一律 XML。
func TestStaleShareGuard(t *testing.T) {
	ResetSourceCache()
	t.Cleanup(ResetSourceCache)
	root := t.TempDir()
	mkdir := func(names ...string) {
		t.Helper()
		for _, name := range names {
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(name)), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv(sourceEnv, "")

	// 只有 share（没有产物）：不判，直读。
	mkdir("share/creator/npc_creator/a", "share/npc/npcconfig")
	if got := EffectiveSource(root, NPCConfigDir); got != SourceShare {
		t.Fatalf("只有 share 应直读，得 %v", got)
	}
	if ShareOlderThanXML(root) || StaleShareNote(root) != "" {
		t.Fatal("没有 XML 产物时不应判旧")
	}

	// 两边相等：直读。
	mkdir("ini/Scene/a")
	if got := EffectiveSource(root, NPCConfigDir); got != SourceShare {
		t.Fatalf("两边场景数相等应直读，得 %v", got)
	}

	// share 比产物多（产物过期）：不拦，直读。
	ResetSourceCache()
	mkdir("share/creator/npc_creator/b")
	if got := EffectiveSource(root, NPCConfigDir); got != SourceShare || ShareOlderThanXML(root) {
		t.Fatalf("share 更全时应直读，得 %v（stale=%v）", got, ShareOlderThanXML(root))
	}

	// share 是旧快照：产物多出两个场景、share 只多一个 → auto 回落 XML。
	ResetSourceCache()
	mkdir("ini/Scene/c", "ini/Scene/d", "ini/Scene/e")
	t.Setenv(sourceEnv, "")
	if !ShareOlderThanXML(root) {
		t.Fatal("share 缺 3 个、只多 1 个，应判为旧快照")
	}
	if got := EffectiveSource(root, NPCConfigDir); got != SourceXML {
		t.Fatalf("旧快照下 auto 应回落 XML，得 %v", got)
	}
	note := StaleShareNote(root)
	if !strings.Contains(note, "少 3 个场景") || !strings.Contains(note, "另有 1 个") || !strings.Contains(note, "已回落 xml") {
		t.Fatalf("说明应讲清缺口与当前去向：%q", note)
	}
	// 判定按数据根缓存：改了摆放树要重新判定。
	mkdir("share/creator/npc_creator/c", "share/creator/npc_creator/d", "share/creator/npc_creator/e")
	if !ShareOlderThanXML(root) {
		t.Fatal("补齐场景后未重新判定前仍应是缓存里的旧结论")
	}
	ResetSourceCache()
	if ShareOlderThanXML(root) {
		t.Fatal("补齐场景后不应再判旧")
	}

	// 显式 share：绕过判定（照直读，说明里点明读的是旧树）。
	mkdir("ini/Scene/f", "ini/Scene/g")
	ResetSourceCache()
	t.Setenv(sourceEnv, "share")
	if got := EffectiveSource(root, NPCConfigDir); got != SourceShare {
		t.Fatalf("显式 share 应直读，得 %v", got)
	}
	if note := StaleShareNote(root); !strings.Contains(note, "强制直读") {
		t.Fatalf("显式 share 下说明应点明取自旧树：%q", note)
	}
	t.Setenv(sourceEnv, "")
	if got := EffectiveSource("", NPCConfigDir); got != SourceXML {
		t.Fatalf("数据根缺席应一律 XML，得 %v", got)
	}
}
