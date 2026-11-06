package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
)

func writeCLIFile(t *testing.T, name, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverPackagesKeepsLoadOrderAndMissingMounts(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "packages.ini")
	writeCLIFile(t, config, "[ini]\nFile=res\\ini.package\n[obj_newnpc]\nFile=res\\obj_newnpc.package\n")
	writeCLIFile(t, filepath.Join(root, "data", "ini.package.files", "res", "ini", "root.ini"), "[COMPOSITE]\n")
	packages, mounts, err := discoverPackages(config, filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 2 || !packages[0].Extracted || packages[1].Extracted || len(mounts) != 1 {
		t.Fatalf("packages=%#v mounts=%#v", packages, mounts)
	}
	if packages[1].LoadOrder != 1 || !reflect.DeepEqual(packages[1].Prefixes, []string{`obj\newnpc`}) {
		t.Fatalf("obj package = %#v", packages[1])
	}
}

func TestRunWritesMachineReadableClosure(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "packages.ini")
	dataRoot := filepath.Join(root, "data")
	writeCLIFile(t, config, "[ini]\nFile=res\\ini.package\n[obj_newnpc]\nFile=res\\obj_newnpc.package\n")
	writeCLIFile(t, filepath.Join(dataRoot, "ini.package.files", "res", "ini", "npc", "sample.ini"), "[COMPOSITE]\nAction=obj\\NewNpc\\action.ini\n")
	var output bytes.Buffer
	err := run(cliOptions{dataRoot: dataRoot, packagesINI: config, entry: `ini\npc\sample.ini`, output: "-", pretty: false, maxResources: 100}, &output)
	if err != nil {
		t.Fatal(err)
	}
	var report clientdata.ClosureReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output.String())
	}
	if report.SchemaVersion != clientdata.ClosureSchemaVersion || report.Summary.Resources != 2 || report.Summary.PackagesNotExtracted != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunStrictReturnsErrorAfterWritingReport(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "packages.ini")
	dataRoot := filepath.Join(root, "data")
	writeCLIFile(t, config, "[ini]\nFile=res\\ini.package\n")
	writeCLIFile(t, filepath.Join(dataRoot, "ini.package.files", "res", "ini", "root.ini"), "[COMPOSITE]\nModel=missing\\model.xmod\n")
	var output bytes.Buffer
	err := run(cliOptions{dataRoot: dataRoot, packagesINI: config, entry: `ini\root.ini`, output: "-", pretty: false, failIncomplete: true}, &output)
	if err == nil || output.Len() == 0 {
		t.Fatalf("strict err=%v output=%q", err, output.String())
	}
}
