package clientdata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func gbkReader(t *testing.T, utf8Text string) *bytes.Reader {
	t.Helper()
	encoded, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), utf8Text)
	if err != nil {
		t.Fatalf("encode GBK fixture: %v", err)
	}
	return bytes.NewReader([]byte(encoded))
}

func TestLoadNPCTemplateTableQuotedMultilineAndDuplicateColumns(t *testing.T) {
	text := strings.Join([]string{
		"编号\t脚本\t说明\t停留一\t停留二",
		"STRING\tSTRING\tSTRING\tSTRING\tSTRING",
		"ID\tscript\tName\tStopTime\tStopTime",
		"first\tCommonNpc\t\"第一行,坐标\n第二行\"\t3\t10",
		"duplicate\tCommonNpc\t甲\t\t11",
		"duplicate\tCommonNpc\t乙\t\t12",
	}, "\n")
	table, err := LoadNPCTemplateTable(gbkReader(t, text), "synthetic.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(table.Rows), 3; got != want {
		t.Fatalf("rows=%d, want %d", got, want)
	}
	row, err := table.Lookup("first")
	if err != nil {
		t.Fatal(err)
	}
	if got := row.CellsNamed("Name")[0].Raw; got != "第一行,坐标\n第二行" {
		t.Fatalf("quoted multiline Name=%q", got)
	}
	if got := table.ColumnIndices("StopTime"); fmt.Sprint(got) != "[3 4]" {
		t.Fatalf("StopTime indices=%v", got)
	}
	if row.PhysicalLine != 4 || table.Rows[1].PhysicalLine != 6 {
		t.Fatalf("physical lines first=%d second=%d", row.PhysicalLine, table.Rows[1].PhysicalLine)
	}
	_, err = table.Lookup("duplicate")
	if !errors.Is(err, ErrNPCTemplateAmbiguous) {
		t.Fatalf("duplicate lookup error=%v, want ErrNPCTemplateAmbiguous", err)
	}
	if got := len(table.LookupAll("duplicate")); got != 2 {
		t.Fatalf("LookupAll duplicate=%d, want 2", got)
	}
}

func TestLoadNPCCreatorPreservesOrdinalAndRepeatedIdentity(t *testing.T) {
	xmlText := `<?xml version="1.0" encoding="utf-8"?>
<object><staticcreator>
<item No="same" x="1" y="2" z="3" ay="0.5" id="npc-a" NotPositive="1"/>
<item No="same" x="4" y="5" z="6" ay="1.5" id="npc-a" NotPositive="0"/>
</staticcreator></object>`
	catalog, err := LoadNPCCreator(strings.NewReader(xmlText), "creator.xml")
	if err != nil {
		t.Fatal(err)
	}
	instances := catalog.InstancesForTemplate("npc-a")
	if got := len(instances); got != 2 {
		t.Fatalf("instances=%d, want 2", got)
	}
	if instances[0].Ordinal != 1 || instances[1].Ordinal != 2 || instances[0].InstanceKey() == instances[1].InstanceKey() {
		t.Fatalf("ordinals/keys not stable: %#v %#v", instances[0], instances[1])
	}
	if instances[0].No != instances[1].No {
		t.Fatalf("test requires repeated No")
	}
	if got := instances[1].Attributes["NotPositive"].Raw; got != "0" {
		t.Fatalf("explicit zero lost: %q", got)
	}
}

func TestLoadNPCRandomCreatorExpandsPositionsAndAmount(t *testing.T) {
	xmlText := `<object><randomcreator>
<creator No="group-a" x="100" y="20" z="200" radius="10">
  <item No="1" id="npc-a" ay="1" amount="2"><position x="101" z="201"/><position x="102" z="202" ay="2"/></item>
  <item No="2" id="npc-b" amount="3"/>
</creator></randomcreator></object>`
	catalog, err := LoadNPCCreator(strings.NewReader(xmlText), "random.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(catalog.Instances); got != 5 {
		t.Fatalf("instances=%d, want 5", got)
	}
	first := catalog.Instances[0]
	if first.TemplateID != "npc-a" || first.Transform.X != 101 || first.Transform.Y != 20 || first.Transform.Z != 201 || first.Transform.AY == nil || *first.Transform.AY != 1 {
		t.Fatalf("first expanded instance=%#v", first)
	}
	second := catalog.Instances[1]
	if second.Transform.AY == nil || *second.Transform.AY != 2 {
		t.Fatalf("position attribute did not override item: %#v", second)
	}
	for _, instance := range catalog.Instances[2:] {
		if instance.TemplateID != "npc-b" || instance.Transform.Y != 20 {
			t.Fatalf("amount fallback instance=%#v", instance)
		}
	}
}

func TestLoadNPCCreatorSkipsEmptyZeroAmountPlaceholder(t *testing.T) {
	xmlText := `<object><randomcreator>
<creator No="group-a" x="100" y="20" z="200" radius="10">
  <item No="1" id="npc-a" amount="1"><position x="101" z="201"/></item>
  <item No="2" id="" amount="0"><position x="0" z="0"/></item>
</creator></randomcreator><staticcreator>
  <item No="placeholder" id="" amount="0" x="0" y="0" z="0"/>
  <item No="live" id="npc-b" x="4" y="5" z="6"/>
</staticcreator></object>`
	catalog, err := LoadNPCCreator(strings.NewReader(xmlText), "placeholder.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(catalog.Instances), 2; got != want {
		t.Fatalf("instances=%d, want %d", got, want)
	}
	if catalog.Instances[0].TemplateID != "npc-a" || catalog.Instances[1].TemplateID != "npc-b" {
		t.Fatalf("unexpected templates: %#v", catalog.Instances)
	}
}

func TestLoadNPCCreatorStillRejectsMissingIDWithoutZeroAmount(t *testing.T) {
	_, err := LoadNPCCreator(strings.NewReader(`<object><staticcreator><item x="1" y="2" z="3" id="" amount="1"/></staticcreator></object>`), "invalid.xml")
	if err == nil || !strings.Contains(err.Error(), "has no id") {
		t.Fatalf("err=%v, want missing id", err)
	}
}

func TestLoadNPCCreatorAcceptsWhitespacePaddedNumericAttributes(t *testing.T) {
	catalog, err := LoadNPCCreator(strings.NewReader(`<object><staticcreator><item x=" 1.25 " y="2" z="3.5 " id="npc-a"/></staticcreator></object>`), "padded.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Instances[0].Transform; got.X != 1.25 || got.Y != 2 || got.Z != 3.5 {
		t.Fatalf("transform=%+v", got)
	}
}

func TestResolveNPCLayeringAndRepeatedStopTimeBinding(t *testing.T) {
	text := strings.Join([]string{
		"编号\t脚本\t名称\t类型\t模式\t消极\t停止1\t停止2",
		"STRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING\tSTRING",
		"ID\tscript\tName\tNpcType\tWeaponMode\tNotPositive\tStopTime\tStopTime",
		"npc-a\tCommonNpc\t测试人\t229\thand1_1\t1\t3\t10",
	}, "\n")
	table, err := LoadNPCTemplateTable(gbkReader(t, text), "table.tsv")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := LoadNPCCreator(strings.NewReader(`<object><staticcreator><item No="n" x="1" y="2" z="3" ay="0.5" id="npc-a" WeaponMode="hand10_1" NotPositive="0"/></staticcreator></object>`), "creator.xml")
	if err != nil {
		t.Fatal(err)
	}
	schema := VisibleNPCModernV1()
	resolved, err := ResolveNPC(schema, table, catalog.Instances[0], ResolveOptions{
		Defaults:  map[string]Value{"NotPositive": Int32Value(9), "HP": Int32Value(100)},
		Runtime:   map[string]Value{"HP": Int32Value(80)},
		Persisted: map[string]Value{"HP": Int32Value(75)},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertValue := func(name string, want any, layer string) {
		t.Helper()
		got := resolved.Properties[name]
		if fmt.Sprint(got.Value.Native()) != fmt.Sprint(want) || got.Provenance.Layer != layer {
			t.Fatalf("%s=%v layer=%s, want %v layer=%s", name, got.Value.Native(), got.Provenance.Layer, want, layer)
		}
	}
	assertValue("WeaponMode", "hand10_1", "creator")
	assertValue("NotPositive", int32(0), "creator")
	assertValue("StopTime", int32(10), "template")
	assertValue("HP", int32(75), "persisted")
	assertValue("PosiX", float32(1), "runtime")
	if resolved.ScriptClass != "CommonNpc" {
		t.Fatalf("ScriptClass=%q", resolved.ScriptClass)
	}
	if _, wronglyMapped := resolved.Properties["LuaScript"]; wronglyMapped {
		t.Fatal("script column was incorrectly aliased to LuaScript")
	}
}

func TestResolveNPCSingleStopTimeBinding(t *testing.T) {
	text := strings.Join([]string{
		"ID\tscript\tName\tNpcType\tStopTime",
		"STRING\tSTRING\tSTRING\tSTRING\tSTRING",
		"ID\tscript\tName\tNpcType\tStopTime",
		"city_npc\tCommonNpc\tCity NPC\t8\t25",
	}, "\n")
	table, err := LoadNPCTemplateTable(gbkReader(t, text), "city.tsv")
	if err != nil {
		t.Fatal(err)
	}
	instance := NPCCreatorInstance{TemplateID: "city_npc", Source: "city.xml", Ordinal: 1}
	resolved, err := ResolveNPC(VisibleNPCModernV1(), table, instance, ResolveOptions{Defaults: VisibleNPCSpawnDefaultsCompatV1()})
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.Properties["StopTime"].Value.I32; got != 25 {
		t.Fatalf("StopTime=%d, want 25", got)
	}
}

func TestWireValueValidation(t *testing.T) {
	for _, test := range []struct {
		wire WireType
		raw  string
	}{
		{WireByte, "256"},
		{WireWord, "65536"},
		{WireInt32, "30000.5"},
		{WireFloat32, "NaN"},
		{WireFloat32, "+Inf"},
	} {
		if _, err := parseWireValue(test.wire, test.raw); err == nil {
			t.Errorf("parseWireValue(%s,%q) unexpectedly succeeded", test.wire, test.raw)
		}
	}
	value, err := parseWireValue(WireInt32, "30000.0")
	if err != nil || value.I32 != 30000 {
		t.Fatalf("integral decimal=%v err=%v", value.Native(), err)
	}
}

func TestModernNPCSchemaRetainsNpcTypesAboveByteRange(t *testing.T) {
	compat, _ := VisibleNPCCompatV1().Field("NpcType")
	modern, _ := VisibleNPCModernV1().Field("NpcType")
	if compat.Type != WireByte || modern.Type != WireInt32 {
		t.Fatalf("NpcType schema compat=%s modern=%s", compat.Type, modern.Type)
	}
	value, err := parseWireValue(modern.Type, "1228")
	if err != nil || value.I32 != 1228 {
		t.Fatalf("modern NpcType 1228=%v err=%v", value.Native(), err)
	}
}

func TestVisibleNPCSchemaGoldens(t *testing.T) {
	for _, test := range []struct {
		schema Schema
		want   string
	}{
		{VisibleNPCCompatV1(), "aacd3667bfe351ce472c5b3ca12547cd6ab46aee512db44999b8e9442a6eb71d"},
		{VisibleNPCModernV1(), "9d4abf699557c65e71a7384f93a4faa9cd178b4ea067390382d78b97c1ef3ad3"},
	} {
		var canonical strings.Builder
		for _, field := range test.schema.Fields {
			fmt.Fprintf(&canonical, "%d:%s:%d;", field.Index, field.Name, field.Type)
		}
		got := fmt.Sprintf("%x", sha256.Sum256([]byte(canonical.String())))
		if got != test.want {
			t.Fatalf("schema %s golden=%s, want %s", test.schema.ID, got, test.want)
		}
	}
}

type npcLoaderGolden struct {
	Sources struct {
		Template struct {
			SHA256  string `json:"sha256"`
			Columns int    `json:"columns"`
			Records int    `json:"logical_records_including_headers"`
		} `json:"template_table"`
		Creator struct {
			SHA256 string `json:"sha256"`
			Items  int    `json:"items"`
		} `json:"creator"`
	} `json:"sources"`
	Template struct {
		ConfigID      string            `json:"config_id"`
		LogicalRecord int               `json:"logical_record"`
		PhysicalLine  int               `json:"physical_start_line"`
		RawNonEmpty   map[string]string `json:"raw_non_empty"`
	} `json:"template"`
	CreatorInstance struct {
		Ordinal    int               `json:"ordinal_one_based"`
		Line       int               `json:"physical_line"`
		No         string            `json:"no"`
		TemplateID string            `json:"template_id"`
		Attributes map[string]string `json:"attributes"`
	} `json:"creator_instance"`
	Expected struct {
		Resolved map[string]struct {
			WireType string          `json:"wire_type"`
			Value    json.RawMessage `json:"value"`
			Source   string          `json:"source"`
		} `json:"resolved_properties"`
	} `json:"expected"`
}

func TestFuncNpc01608GoldenAgainstModernClientData(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	fixtureBytes, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "clientdata", "npc-template-loader-fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture npcLoaderGolden
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatalf("decode golden fixture: %v", err)
	}
	shareRoot := os.Getenv("NINEYIN_SHARE_ROOT")
	if shareRoot == "" {
		shareRoot = filepath.Join(repoRoot, "resources", "modern", "share")
	}
	tablePath := filepath.Join(shareRoot, "npc", "npcconfig", "worldnpc", "school_commonnpc.txt")
	creatorPath := filepath.Join(shareRoot, "creator", "npc_creator", "school07_wudang", "commonnpc.xml")
	if _, err := os.Stat(tablePath); err != nil {
		t.Skipf("modern client data unavailable: %v", err)
	}
	if got := fileSHA256(t, tablePath); !strings.EqualFold(got, fixture.Sources.Template.SHA256) {
		t.Fatalf("template SHA-256=%s, want %s", got, fixture.Sources.Template.SHA256)
	}
	if got := fileSHA256(t, creatorPath); !strings.EqualFold(got, fixture.Sources.Creator.SHA256) {
		t.Fatalf("creator SHA-256=%s, want %s", got, fixture.Sources.Creator.SHA256)
	}
	tableFile, err := os.Open(tablePath)
	if err != nil {
		t.Fatal(err)
	}
	defer tableFile.Close()
	table, err := LoadNPCTemplateTable(tableFile, tablePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != fixture.Sources.Template.Columns || len(table.Rows)+3 != fixture.Sources.Template.Records {
		t.Fatalf("table dimensions columns=%d records=%d", len(table.Columns), len(table.Rows)+3)
	}
	row, err := table.Lookup(fixture.Template.ConfigID)
	if err != nil {
		t.Fatal(err)
	}
	if row.LogicalRecord != fixture.Template.LogicalRecord || row.PhysicalLine != fixture.Template.PhysicalLine {
		t.Fatalf("template location logical=%d physical=%d", row.LogicalRecord, row.PhysicalLine)
	}
	for name, want := range fixture.Template.RawNonEmpty {
		cells := row.CellsNamed(name)
		var got string
		for _, cell := range cells {
			if cell.Set {
				got = cell.Raw
			}
		}
		if got != want {
			t.Fatalf("template %s=%q, want %q", name, got, want)
		}
	}
	creatorFile, err := os.Open(creatorPath)
	if err != nil {
		t.Fatal(err)
	}
	defer creatorFile.Close()
	catalog, err := LoadNPCCreator(creatorFile, creatorPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Instances) != fixture.Sources.Creator.Items {
		t.Fatalf("creator items=%d, want %d", len(catalog.Instances), fixture.Sources.Creator.Items)
	}
	instance := catalog.Instances[fixture.CreatorInstance.Ordinal-1]
	if instance.TemplateID != fixture.CreatorInstance.TemplateID || instance.No != fixture.CreatorInstance.No || instance.PhysicalLine != fixture.CreatorInstance.Line {
		t.Fatalf("creator target=%#v", instance)
	}
	for name, want := range fixture.CreatorInstance.Attributes {
		got, present := instance.Attributes[name]
		if !present || !got.Present || got.Raw != want {
			t.Fatalf("creator attribute %s=%#v, want %q", name, got, want)
		}
	}
	modernSchema := VisibleNPCModernV1()
	resolved, err := ResolveNPC(modernSchema, table, instance, ResolveOptions{Defaults: VisibleNPCSpawnDefaultsCompatV1()})
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range fixture.Expected.Resolved {
		actual, exists := resolved.Properties[name]
		if !exists {
			t.Fatalf("resolved property %s missing", name)
		}
		if actual.Value.Type.String() != expected.WireType {
			t.Fatalf("resolved %s type=%s, want %s", name, actual.Value.Type, expected.WireType)
		}
		assertGoldenNativeValue(t, name, actual.Value, expected.Value)
		wantSource := strings.SplitN(expected.Source, ".", 2)
		if actual.Provenance.Layer != wantSource[0] || actual.Provenance.Field != wantSource[1] {
			t.Fatalf("resolved %s provenance=%#v, want %s", name, actual.Provenance, expected.Source)
		}
	}
	ordered, err := resolved.OrderedProperties(modernSchema)
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 83 {
		t.Fatalf("materialized compatibility properties=%d, want 83", len(ordered))
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func assertGoldenNativeValue(t *testing.T, name string, actual Value, raw json.RawMessage) {
	t.Helper()
	switch actual.Type {
	case WireString, WireWideString:
		var want string
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		if actual.Text != want {
			t.Fatalf("resolved %s=%q, want %q", name, actual.Text, want)
		}
	default:
		var want float64
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		var got float64
		switch actual.Type {
		case WireByte:
			got = float64(actual.U8)
		case WireWord:
			got = float64(actual.U16)
		case WireInt32:
			got = float64(actual.I32)
		case WireFloat32:
			got = float64(actual.F32)
		}
		if math.Abs(got-want) > 0.0001 {
			t.Fatalf("resolved %s=%v, want %v", name, got, want)
		}
	}
}
