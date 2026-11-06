package clientdata

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, root, name, contents string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLookupUsesASCIIInsensitiveCaseButStrictSeparators(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "ini/npc/WorldNpc288.ini", "model")
	index, err := Build(Config{Mounts: []Mount{{Name: "ini", Directory: root}}})
	if err != nil {
		t.Fatal(err)
	}

	match := index.Lookup(`INI\NPC\worldnpc288.INI`)
	if !match.Found() || !match.CaseFolded || match.Entry.Winner().Mount != "ini" {
		t.Fatalf("case-folded lookup = %#v", match)
	}

	wrongSlash := index.Lookup(`ini/npc/worldnpc288.ini`)
	if wrongSlash.Found() {
		t.Fatal("slash variant unexpectedly resolved")
	}
	if !reflect.DeepEqual(wrongSlash.SeparatorCandidates, []string{`ini\npc\WorldNpc288.ini`}) {
		t.Fatalf("separator candidates = %#v", wrongSlash.SeparatorCandidates)
	}

	// FxPackage only folds ASCII. Unicode case mappings must not leak in.
	writeFixture(t, root, "配置/数据.ini", "data")
	index, err = Build(Config{Mounts: []Mount{{Name: "ini", Directory: root}}})
	if err != nil {
		t.Fatal(err)
	}
	if index.Lookup(`配置\数据.ini`).Entry == nil {
		t.Fatal("exact Unicode lookup failed")
	}
}

func TestEarlierMountShadowsLaterSourceAndReadFileUsesWinner(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	firstPath := writeFixture(t, first, "obj/NewNpc/model.xmod", "first")
	writeFixture(t, second, "OBJ/newnpc/MODEL.XMOD", "second")
	index, err := Build(Config{Mounts: []Mount{
		{Name: "obj_old", Directory: first},
		{Name: "obj_new", Directory: second},
	}})
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := index.Entry(`obj\newnpc\model.xmod`)
	if !ok || !entry.Shadowed() || len(entry.Sources) != 2 {
		t.Fatalf("entry = %#v, found=%v", entry, ok)
	}
	if entry.Winner().Mount != "obj_old" || entry.Winner().PhysicalPath != firstPath {
		t.Fatalf("winner = %#v", entry.Winner())
	}
	data, err := index.ReadFile(`OBJ\NEWNPC\MODEL.XMOD`)
	if err != nil || string(data) != "first" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	diagnostics := index.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind != DiagnosticShadowed || diagnostics[0].Mount != "obj_new" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestVirtualPrefixAndReturnedValuesAreImmutable(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "npc/a.ini", "a")
	index, err := Build(Config{Mounts: []Mount{{Name: "ini", Directory: root, VirtualPrefix: "ini"}}})
	if err != nil {
		t.Fatal(err)
	}
	entries := index.Entries()
	if len(entries) != 1 || entries[0].VirtualPath != `ini\npc\a.ini` {
		t.Fatalf("entries = %#v", entries)
	}
	entries[0].Sources[0].Mount = "mutated"
	entry, _ := index.Entry(`ini\npc\a.ini`)
	if entry.Winner().Mount != "ini" {
		t.Fatal("caller mutated index source")
	}
}

func TestExtractedPackageMountStripsPhysicalResDirectory(t *testing.T) {
	extraction := t.TempDir()
	writeFixture(t, filepath.Join(extraction, "res"), "ini/npc/a.ini", "a")
	index, err := Build(Config{Mounts: []Mount{ExtractedPackageMount("ini", extraction)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := index.Entry(`ini\npc\a.ini`); !ok {
		t.Fatal("payload below res was not mounted at virtual root")
	}
	if _, ok := index.Entry(`res\ini\npc\a.ini`); ok {
		t.Fatal("physical extraction directory leaked into virtual path")
	}
}

func TestMissingAndUnreadableRootsProduceDiagnostics(t *testing.T) {
	root := t.TempDir()
	file := writeFixture(t, root, "broken.bin", "x")
	missing := filepath.Join(root, "missing")
	probeError := errors.New("synthetic open failure")
	index, err := build(Config{Mounts: []Mount{
		{Name: "data", Directory: root},
		{Name: "missing", Directory: missing},
	}}, func(path string) error {
		if path == file {
			return probeError
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := index.Entry(`broken.bin`)
	if !ok || entry.Winner().Readable || !strings.Contains(entry.Winner().ReadError, probeError.Error()) {
		t.Fatalf("unreadable source = %#v", entry)
	}
	kinds := map[DiagnosticKind]bool{}
	for _, diagnostic := range index.Diagnostics() {
		kinds[diagnostic.Kind] = true
	}
	if !kinds[DiagnosticUnreadable] || !kinds[DiagnosticRoot] {
		t.Fatalf("diagnostic kinds = %#v", kinds)
	}

	_, err = Build(Config{FailOnRootError: true, Mounts: []Mount{{Name: "missing", Directory: missing}}})
	if err == nil {
		t.Fatal("strict missing root accepted")
	}
}

func TestInputValidationAndNotFound(t *testing.T) {
	root := t.TempDir()
	if _, err := Build(Config{}); err == nil {
		t.Fatal("empty config accepted")
	}
	if _, err := Build(Config{Mounts: []Mount{{Name: "bad", Directory: root, VirtualPrefix: `/ini`}}}); err == nil {
		t.Fatal("invalid prefix accepted")
	}
	index, err := Build(Config{Mounts: []Mount{{Name: "empty", Directory: root}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.ReadFile(`missing.ini`); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not-found error = %v", err)
	}
}
