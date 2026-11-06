package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 对实库 role_faculty 的只读往返校验（不写库）：每行快照经
// facultyProgressSnapshot 解码再编码后，顶层与 books[] 书内的
// JSON 键集合必须与源完全一致——passthrough 保证 Go 写回不抹掉
// Java/新版服导入的超集字段。NINEYIN_TEST_MYSQL_DSN 未设置时跳过。
func TestFacultyDBRoundTripsLiveRows(t *testing.T) {
	dsn := os.Getenv("NINEYIN_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NINEYIN_TEST_MYSQL_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repository, err := role.OpenMySQLRepository(ctx, dsn, role.MySQLConfig{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer repository.Close()
	store := newFacultyDBStore(repository.DB())
	rows, err := store.db.QueryContext(ctx, `SELECT role_id, snapshot FROM role_faculty`)
	if err != nil {
		t.Fatalf("select role_faculty: %v", err)
	}
	defer rows.Close()
	checked := 0
	for rows.Next() {
		var roleID int64
		var raw []byte
		if err := rows.Scan(&roleID, &raw); err != nil {
			t.Fatalf("scan role=%d: %v", roleID, err)
		}
		var snapshot facultyProgressSnapshot
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			t.Errorf("role=%d decode: %v", roleID, err)
			continue
		}
		rendered, err := json.Marshal(snapshot)
		if err != nil {
			t.Errorf("role=%d encode: %v", roleID, err)
			continue
		}
		assertSameJSONKeys(t, roleID, raw, rendered)
		checked++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Skip("role_faculty has no rows to verify")
	}
	t.Logf("round-tripped %d live faculty rows with full key preservation", checked)
}

// assertSameJSONKeys 比较两段 JSON 的键集合（顶层与 books 数组逐元素）。
func assertSameJSONKeys(t *testing.T, roleID int64, source, rendered []byte) {
	t.Helper()
	var sourceTop, renderedTop map[string]json.RawMessage
	if err := json.Unmarshal(source, &sourceTop); err != nil {
		t.Errorf("role=%d source top-level decode: %v", roleID, err)
		return
	}
	if err := json.Unmarshal(rendered, &renderedTop); err != nil {
		t.Errorf("role=%d rendered top-level decode: %v", roleID, err)
		return
	}
	for key := range sourceTop {
		if _, exists := renderedTop[key]; !exists {
			t.Errorf("role=%d top-level key %q lost", roleID, key)
		}
	}
	sourceBooks, sourceHasBooks := sourceTop["books"]
	renderedBooks, renderedHasBooks := renderedTop["books"]
	if !sourceHasBooks || !renderedHasBooks {
		return
	}
	var sourceList, renderedList []map[string]json.RawMessage
	if string(sourceBooks) == "null" || string(renderedBooks) == "null" {
		return
	}
	if err := json.Unmarshal(sourceBooks, &sourceList); err != nil {
		t.Errorf("role=%d source books decode: %v", roleID, err)
		return
	}
	if err := json.Unmarshal(renderedBooks, &renderedList); err != nil {
		t.Errorf("role=%d rendered books decode: %v", roleID, err)
		return
	}
	if len(sourceList) != len(renderedList) {
		t.Errorf("role=%d books len=%d want %d", roleID, len(renderedList), len(sourceList))
		return
	}
	for index, book := range sourceList {
		for key := range book {
			if _, exists := renderedList[index][key]; !exists {
				t.Errorf("role=%d books[%d] key %q lost", roleID, index, key)
			}
		}
	}
}
