package clientdata

import (
	"os"
	"path/filepath"
	"testing"
)

// 本可选测试基于本地解包出的客户端数据验证索引：
//
//	NINEYIN_EXTRACTED_ROOT=E:\jiuyin go test -run TestLocalExtractedPackages ./internal/clientdata
func TestLocalExtractedPackages(t *testing.T) {
	root := os.Getenv("NINEYIN_EXTRACTED_ROOT")
	if root == "" {
		t.Skip("NINEYIN_EXTRACTED_ROOT is not configured")
	}
	index, err := Build(Config{Mounts: []Mount{
		ExtractedPackageMount("share", filepath.Join(root, "share.package.files")),
		ExtractedPackageMount("ini", filepath.Join(root, "ini.package.files")),
		ExtractedPackageMount("obj_newnpc", filepath.Join(root, "obj_newnpc.package.files")),
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, virtualPath := range []string{
		`share\npc\npcconfig\worldnpc\school_commonnpc.txt`,
		`ini\npc\worldnpc288.ini`,
	} {
		entry, ok := index.Entry(virtualPath)
		if !ok {
			t.Errorf("missing known client resource %s", virtualPath)
			continue
		}
		if !entry.Winner().Readable {
			t.Errorf("known client resource unreadable: %#v", entry.Winner())
		}
	}
}
