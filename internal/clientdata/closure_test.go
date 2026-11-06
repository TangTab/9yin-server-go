package clientdata

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAuditResourceClosureExpandsCompositeAndRecursiveActions(t *testing.T) {
	iniRoot, objRoot := t.TempDir(), t.TempDir()
	writeFixture(t, iniRoot, "ini/npc/worldnpc288.ini", `[COMPOSITE]
Action=obj\NewNpc\man\worldnpc288\action.ini
Model=obj\NewNpc\man\worldnpc288\worldnpc288.xmod
main_model=obj\npc_tpose\tpose_male\Tpose.xmod
`)
	writeFixture(t, objRoot, "obj/NewNpc/man/worldnpc288/action.ini", `[ACTION]
ACTION_BASE_FILE=..\common\base.ini
ACTION_CHILD_FILE=child_a.ini|child_b.ini
Skeleton=skeleton\worldnpc288.xskt
`)
	writeFixture(t, objRoot, "obj/NewNpc/man/common/base.ini", "[ACTION]\nSkeleton=base.xskt\n")
	writeFixture(t, objRoot, "obj/NewNpc/man/worldnpc288/child_a.ini", "[ACTION]\n")
	writeFixture(t, objRoot, "obj/NewNpc/man/worldnpc288/child_b.ini", "[ACTION]\n")
	writeFixture(t, objRoot, "obj/NewNpc/man/worldnpc288/skeleton/worldnpc288.xskt", "skeleton")
	writeFixture(t, objRoot, "obj/NewNpc/man/common/base.xskt", "base skeleton")
	writeFixture(t, objRoot, "obj/NewNpc/man/worldnpc288/worldnpc288.xmod", "model")
	writeFixture(t, objRoot, "obj/npc_tpose/tpose_male/Tpose.xmod", "main")
	index, err := Build(Config{Mounts: []Mount{
		{Name: "ini", Directory: iniRoot},
		{Name: "obj_newnpc", Directory: objRoot},
	}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := AuditResourceClosure(index, ClosureOptions{
		Entry:        `ini\npc\worldnpc288.ini`,
		VirtualRoots: []string{"ini", "obj"},
		GeneratedAt:  time.Unix(0, 0),
		Packages: []KnownPackage{
			{Name: "ini", Prefixes: []string{"ini"}, Extracted: true},
			{Name: "obj_newnpc", Prefixes: []string{`obj\newnpc`}, Extracted: true},
			{Name: "obj_npc", Prefixes: []string{`obj\npc`}, Extracted: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != ClosureSchemaVersion || report.GeneratedAt != "1970-01-01T00:00:00Z" {
		t.Fatalf("report header = %#v", report)
	}
	if report.Summary.Resources != 9 || report.Summary.Edges != 8 || report.Summary.Hits != 9 {
		t.Fatalf("summary = %#v", report.Summary)
	}
	wantPaths := map[string]bool{
		`obj\NewNpc\man\common\base.ini`:                       true,
		`obj\NewNpc\man\worldnpc288\child_a.ini`:               true,
		`obj\NewNpc\man\worldnpc288\skeleton\worldnpc288.xskt`: true,
		`obj\NewNpc\man\common\base.xskt`:                      true,
	}
	for _, node := range report.Resources {
		delete(wantPaths, node.Path)
	}
	if len(wantPaths) != 0 {
		t.Fatalf("unvisited recursive paths: %#v", wantPaths)
	}
	for _, edge := range report.Edges {
		if edge.To < 0 || edge.To >= len(report.Resources) || report.Resources[edge.To].Path != edge.Resolved {
			t.Fatalf("broken edge %#v", edge)
		}
	}
}

func TestAuditDistinguishesSeparatorMismatchUnextractedAndMissing(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "ini/root.ini", `[COMPOSITE]
Action=obj/NewNpc/action.ini
Model=obj\NewNpc\missing.xmod
main_model=unknown\missing.xmod
`)
	writeFixture(t, root, "obj/NewNpc/action.ini", "[ACTION]\n")
	index, err := Build(Config{Mounts: []Mount{{Name: "extracted", Directory: root}}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := AuditResourceClosure(index, ClosureOptions{
		Entry: `ini\root.ini`, VirtualRoots: []string{"ini", "obj", "unknown"},
		Packages: []KnownPackage{{Name: "obj_newnpc", Prefixes: []string{`obj\newnpc`}, Extracted: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]ResourceStatus{}
	for _, node := range report.Resources {
		statuses[node.Path] = node.Status
	}
	if statuses[`obj/NewNpc/action.ini`] != ResourceSeparatorMismatch {
		t.Fatalf("slash status = %q", statuses[`obj/NewNpc/action.ini`])
	}
	if statuses[`obj\NewNpc\missing.xmod`] != ResourcePackageNotExtracted {
		t.Fatalf("unextracted status = %q", statuses[`obj\NewNpc\missing.xmod`])
	}
	if statuses[`unknown\missing.xmod`] != ResourceMissing {
		t.Fatalf("missing status = %q", statuses[`unknown\missing.xmod`])
	}
	if report.Summary.SeparatorMismatches != 1 || report.Summary.PackagesNotExtracted != 1 || report.Summary.Missing != 1 {
		t.Fatalf("summary = %#v", report.Summary)
	}
}

func TestParserPreservesDuplicateChildFilesAndSeparatorStyle(t *testing.T) {
	refs := parseINIReferences([]byte("\xef\xbb\xbfACTION_CHILD_FILE=a.ini\nACTION_CHILD_FILE=b.ini,c.ini\nSkeleton='bones\\npc.xskt'\n"), actionDependencyKeys)
	var raw []string
	for _, ref := range refs {
		raw = append(raw, ref.Raw)
	}
	if !reflect.DeepEqual(raw, []string{"a.ini", "b.ini", "c.ini", `bones\npc.xskt`}) {
		t.Fatalf("refs = %#v", refs)
	}
	roots := map[string]bool{"obj": true}
	if got := resolveReference(`obj\npc\action.ini`, `..\base\base.ini`, roots); got != `obj\base\base.ini` {
		t.Fatalf("backslash relative = %q", got)
	}
	if got := resolveReference(`obj\npc\action.ini`, `child/action.ini`, roots); got != `obj/npc/child/action.ini` {
		t.Fatalf("slash relative = %q", got)
	}
}

func TestClosureReportsShadowedSources(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writeFixture(t, first, "ini/root.ini", "[COMPOSITE]\n")
	secondFile := writeFixture(t, second, "ini/root.ini", "[COMPOSITE]\n")
	index, err := Build(Config{Mounts: []Mount{{Name: "first", Directory: first}, {Name: "second", Directory: second}}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := AuditResourceClosure(index, ClosureOptions{Entry: `ini\root.ini`})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Shadowed != 1 || len(report.Resources[0].Sources) != 2 || !report.Resources[0].Sources[1].Shadowed {
		t.Fatalf("shadow report = %#v", report)
	}
	if report.Resources[0].Sources[1].PhysicalPath != filepath.Clean(secondFile) {
		t.Fatalf("second source = %#v", report.Resources[0].Sources[1])
	}
}
