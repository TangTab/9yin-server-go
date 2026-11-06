package qinggong

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsOfficialCostFields(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "qgdefine.ini")
	if err := os.WriteFile(path, []byte("[qinggong_1]\nConsume=10\nType=1\nUseType=0\n[qinggong_24]\nConsume=5\nType=24\nUseType=1\nBufferID=buf_qg_drift\n"), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := catalog.Lookup("qinggong_24")
	if !ok || entry.Consume != 5 || entry.Type != 24 || entry.UseType != 1 || entry.BufferID != "buf_qg_drift" {
		t.Fatalf("entry=%+v ok=%t", entry, ok)
	}
}
