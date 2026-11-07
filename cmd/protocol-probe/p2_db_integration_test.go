package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// P2 域（msg70/71 货币、msg295 称号、msg150/161 经脉）对实库的往返
// 校验。NINEYIN_TEST_MYSQL_DSN 未设置或表未建（migrations 未重放）时
// 跳过；写库测试借用已有 roles 行并在结束恢复原状，不落测试残留。

func p2TestDB(t *testing.T) (*sql.DB, role.RoleID) {
	t.Helper()
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
	t.Cleanup(func() { repository.Close() })
	db := repository.DB()
	var missing []string
	for _, table := range []string{"role_currency", "role_titles", "role_meridians"} {
		var count int
		if err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			table).Scan(&count); err != nil || count == 0 {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		t.Skipf("tables not migrated yet: %v (replay migrations/ to enable)", missing)
	}
	var roleID role.RoleID
	if err := db.QueryRowContext(ctx, "SELECT role_id FROM roles ORDER BY role_id LIMIT 1").Scan(&roleID); err != nil {
		t.Skipf("no roles rows to borrow: %v", err)
	}
	return db, roleID
}

// p2OptionalTableTest 邮件/坐骑表（role_letters/role_mounts）由 Java 端
// 运行时建表（Go 不做 DDL），可能缺表——缺表跳过而非整体失败。
func p2OptionalTableTest(t *testing.T, tables ...string) *sql.DB {
	t.Helper()
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
	t.Cleanup(func() { repository.Close() })
	db := repository.DB()
	for _, table := range tables {
		var count int
		if err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			table).Scan(&count); err != nil || count == 0 {
			t.Skipf("table %s not present (Java runtime creates it)", table)
		}
	}
	return db
}

// restoreTableSnapshot 借行写库测试的兜底：结束后恢复原快照/删测试行。
func restoreTableSnapshot(t *testing.T, db *sql.DB, table string, roleID role.RoleID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var original sql.NullString
	_ = db.QueryRowContext(ctx, "SELECT snapshot FROM "+table+" WHERE role_id = ?", roleID).Scan(&original)
	t.Cleanup(func() {
		if original.Valid {
			_, _ = db.ExecContext(ctx, "UPDATE "+table+" SET snapshot = ? WHERE role_id = ?", original.String, roleID)
		} else {
			_, _ = db.ExecContext(ctx, "DELETE FROM "+table+" WHERE role_id = ?", roleID)
		}
	})
}

func TestLetterDBLoadParsesSnapshot(t *testing.T) {
	db := p2OptionalTableTest(t, "role_letters")
	var roleID role.RoleID
	if err := db.QueryRow("SELECT role_id FROM roles ORDER BY role_id LIMIT 1").Scan(&roleID); err != nil {
		t.Skipf("no roles rows to borrow: %v", err)
	}
	restoreTableSnapshot(t, db, "role_letters", roleID)
	raw := `{"letters": [{"serial": "t-1", "type": 101, "sender": "", "sender_uid": "gm",` +
		` "title": "集成测试", "content": "", "gold": "0", "silver": "0", "appendix_xml": "",` +
		` "date_unix_seconds": 1793892717, "read_flag": true}]}`
	if _, err := db.Exec("INSERT INTO role_letters (role_id, snapshot, version, updated_at) VALUES (?, ?, 1, UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot)", roleID, raw); err != nil {
		t.Fatalf("seed: %v", err)
	}
	letters := newLetterDBStore(db).Load(roleID)
	if len(letters) != 1 || letters[0].Serial != "t-1" || letters[0].Type != 101 || !letters[0].ReadFlag {
		t.Fatalf("letters=%+v", letters)
	}
}

func TestMountDBLoadParsesSnapshot(t *testing.T) {
	db := p2OptionalTableTest(t, "role_mounts")
	var roleID role.RoleID
	if err := db.QueryRow("SELECT role_id FROM roles ORDER BY role_id LIMIT 1").Scan(&roleID); err != nil {
		t.Skipf("no roles rows to borrow: %v", err)
	}
	restoreTableSnapshot(t, db, "role_mounts", roleID)
	raw := `[{"ConfigID": "MOUNT_HAWK_003", "Selected": true, "StaticID": 41523, "UsedDate": 0, "ValidTime": 30}]`
	if _, err := db.Exec("INSERT INTO role_mounts (role_id, snapshot, updated_at) VALUES (?, ?, UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot)", roleID, raw); err != nil {
		t.Fatalf("seed: %v", err)
	}
	mounts := newMountDBStore(db).Load(roleID)
	if len(mounts) != 1 || mounts[0].PackID != 41523 || !mounts[0].Selected || mounts[0].ValidTime != 30 {
		t.Fatalf("mounts=%+v", mounts)
	}
	if got := mountCurrentPack(mounts); got != 41523 {
		t.Fatalf("currentPack=%d, want 41523", got)
	}
}

func TestCurrencyDBSaveLoadRoundtrip(t *testing.T) {
	db, roleID := p2TestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := newCurrencyDBStore(db)

	var original sql.NullString
	_ = db.QueryRowContext(ctx, "SELECT snapshot FROM role_currency WHERE role_id = ?", roleID).Scan(&original)
	t.Cleanup(func() {
		if original.Valid {
			_, _ = db.ExecContext(ctx, "UPDATE role_currency SET snapshot = ? WHERE role_id = ?", original.String, roleID)
		} else {
			_, _ = db.ExecContext(ctx, "DELETE FROM role_currency WHERE role_id = ?", roleID)
		}
	})

	want := currencySnapshot{Silver: 4321}
	if err := store.Save(roleID, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok := store.Load(roleID)
	if !ok || got != want {
		t.Fatalf("load=%+v ok=%t, want %+v", got, ok, want)
	}
}

func TestTitleDBWearAndCancelRoundtrip(t *testing.T) {
	db, roleID := p2TestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := newTitleDBStore(db)

	type titleBackup struct {
		titleID, titleType, isWorn int32
	}
	var backups []titleBackup
	rows, err := db.QueryContext(ctx, "SELECT title_id, title_type, is_worn FROM role_titles WHERE role_id = ?", roleID)
	if err == nil {
		for rows.Next() {
			var backup titleBackup
			if rows.Scan(&backup.titleID, &backup.titleType, &backup.isWorn) == nil {
				backups = append(backups, backup)
			}
		}
		rows.Close()
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM role_titles WHERE role_id = ?", roleID)
		for _, backup := range backups {
			_, _ = db.ExecContext(ctx,
				"INSERT INTO role_titles (role_id, title_id, title_type, is_worn, obtained_unix) VALUES (?, ?, ?, ?, 0)",
				roleID, backup.titleID, backup.titleType, backup.isWorn)
		}
	})

	if err := store.wear(roleID, 30021); err != nil {
		t.Fatalf("wear: %v", err)
	}
	titles := store.Load(roleID)
	var worn int32 = -1
	for _, row := range titles {
		if row.titleID == 30021 && row.worn {
			worn = row.titleID
		}
	}
	if worn != 30021 {
		t.Fatalf("worn=%d after wear, titles=%+v", worn, titles)
	}
	if err := store.wear(roleID, 0); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	for _, row := range store.Load(roleID) {
		if row.worn {
			t.Fatalf("title %d still worn after cancel", row.titleID)
		}
	}
}

func TestJingMaiDBSaveLoadRoundtrip(t *testing.T) {
	db, roleID := p2TestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := newJingMaiDBStore(db)

	var original sql.NullString
	_ = db.QueryRowContext(ctx, "SELECT snapshot FROM role_meridians WHERE role_id = ?", roleID).Scan(&original)
	t.Cleanup(func() {
		if original.Valid {
			_, _ = db.ExecContext(ctx, "UPDATE role_meridians SET snapshot = ? WHERE role_id = ?", original.String, roleID)
		} else {
			_, _ = db.ExecContext(ctx, "DELETE FROM role_meridians WHERE role_id = ?", roleID)
		}
	})

	want := jingMaiSnapshot{
		Cur: "jm_dantian", Last: "jm_renmai",
		Learned:   []string{"jm_dantian", "jm_renmai"},
		Activated: []string{"jm_dantian"},
		Progress:  map[string]jingMaiProgressRow{"jm_dantian": {Level: 2, Fill: 111, Total: 1611}},
		ZhenQi:    jingMaiZhenQiPool{Day: "2026-11-07", Used: 111},
	}
	if err := store.Save(roleID, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok := store.Load(roleID)
	if !ok || got.Cur != want.Cur || len(got.Activated) != 1 || got.Progress["jm_dantian"] != (jingMaiProgressRow{Level: 2, Fill: 111, Total: 1611}) {
		t.Fatalf("load=%+v ok=%t", got, ok)
	}
	if len(got.Learned) != 2 || got.Learned[0] != "jm_dantian" || got.Learned[1] != "jm_renmai" {
		t.Fatalf("learned roundtrip=%v", got.Learned)
	}
}
