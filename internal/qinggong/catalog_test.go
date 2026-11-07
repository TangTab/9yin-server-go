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

func TestLoadReadsExistBufIDGate(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "qgdefine.ini")
	if err := os.WriteFile(path, []byte("[qinggong_4]\nConsume=14\nType=4\nUseType=0\nExistBufID=buf_DHadd_2rd\n[qinggong_5]\nConsume=15\nType=5\nUseType=0\nExistBufID=buf_DHadd_3rd\n[qinggong_1]\nType=1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	second, ok := catalog.Lookup("qinggong_4")
	if !ok || second.ExistBufID != "buf_DHadd_2rd" {
		t.Fatalf("second=%+v ok=%t", second, ok)
	}
	third, ok := catalog.Lookup("qinggong_5")
	if !ok || third.ExistBufID != "buf_DHadd_3rd" {
		t.Fatalf("third=%+v ok=%t", third, ok)
	}
	// 旧表缺 ExistBufID 键 → 空串 = 无门槛（按现状放行）。
	legacy, ok := catalog.Lookup("qinggong_1")
	if !ok || legacy.ExistBufID != "" {
		t.Fatalf("legacy=%+v ok=%t", legacy, ok)
	}
}
