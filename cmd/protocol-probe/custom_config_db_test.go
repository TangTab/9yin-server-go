package main

import (
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

const customConfigTestRoleID = role.RoleID(77)

func newCustomConfigMockDB(t *testing.T) (sqlmock.Sqlmock, customConfigStores) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return mock, newCustomConfigStores(db)
}

func newSQLMockBytesRow(column string, value string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{column}).AddRow([]byte(value))
}

func newSQLMockResult(lastID, affected int64) driver.Result {
	return sqlmock.NewResult(lastID, affected)
}

func newMockCustomizingDB(t *testing.T) (*customizingDBStore, sqlmock.Sqlmock) {
	t.Helper()
	mock, stores := newCustomConfigMockDB(t)
	return stores.customizing, mock
}

// role_keybinds UPSERT 的两个已踩实的坑（Java RoleInventoryStore）：
// scheme 0 的目标列即 INSERT 列单里的 bind_value（不得出现两次）；
// 其它列首插必须补 bind_value=”（NOT NULL 无默认值）。
func TestKeybindDBSaveSchemeJavaParitySQL(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectExec(`INSERT INTO role_keybinds \(role_id, bind_value\) VALUES \(\?, \?\)`+
		` ON DUPLICATE KEY UPDATE bind_value = VALUES\(bind_value\)`).
		WithArgs(customConfigTestRoleID, "1,87").
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.keybind.SaveScheme(customConfigTestRoleID, 0, "1,87"); err != nil {
		t.Fatal(err)
	}

	mock, stores = newCustomConfigMockDB(t)
	mock.ExpectExec(`INSERT INTO role_keybinds \(role_id, bind_value, shortcut_key1\) VALUES \(\?, '', \?\)`+
		` ON DUPLICATE KEY UPDATE shortcut_key1 = VALUES\(shortcut_key1\)`).
		WithArgs(customConfigTestRoleID, "2,83").
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.keybind.SaveScheme(customConfigTestRoleID, 1, "2,83"); err != nil {
		t.Fatal(err)
	}

	mock, stores = newCustomConfigMockDB(t)
	mock.ExpectExec(`INSERT INTO role_keybinds \(role_id, bind_value, shortcut_key2\) VALUES \(\?, '', \?\)`+
		` ON DUPLICATE KEY UPDATE shortcut_key2 = VALUES\(shortcut_key2\)`).
		WithArgs(customConfigTestRoleID, "3,65").
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.keybind.SaveScheme(customConfigTestRoleID, 2, "3,65"); err != nil {
		t.Fatal(err)
	}

	if err := stores.keybind.SaveScheme(customConfigTestRoleID, 9, "x"); err == nil {
		t.Fatal("unknown scheme must be rejected")
	}
}

func TestKeybindDBSaveOperateModeBootstrapsBindValue(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectExec(`INSERT INTO role_keybinds \(role_id, bind_value, operate_mode\) VALUES \(\?, '', \?\)`+
		` ON DUPLICATE KEY UPDATE operate_mode = VALUES\(operate_mode\)`).
		WithArgs(customConfigTestRoleID, int32(1)).
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.keybind.SaveOperateMode(customConfigTestRoleID, 1); err != nil {
		t.Fatal(err)
	}
}

// operate_mode NULL = 从未设置（不下发，客户端保持本地默认 2.5D）；
// neigong_bind NULL = 从未绑定（同零帧，客户端 get_svr_ng_bind 回落全零）。
func TestKeybindDBLoadDistinguishesNullOperateMode(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectQuery(`SELECT bind_value, shortcut_key1, shortcut_key2, operate_mode, neigong_bind FROM role_keybinds WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"bind_value", "shortcut_key1", "shortcut_key2", "operate_mode", "neigong_bind"}).
			AddRow("1,87", "2,83", "3,65", nil, nil))
	loaded, exists := stores.keybind.Load(customConfigTestRoleID)
	if !exists || loaded.bind != "1,87" || loaded.scheme1 != "2,83" || loaded.scheme2 != "3,65" {
		t.Fatalf("loaded=%+v exists=%t", loaded, exists)
	}
	if loaded.operateModeSet {
		t.Fatal("NULL operate_mode must read as unset")
	}
	if loaded.neigongBind != "" {
		t.Fatalf("NULL neigong_bind must read as empty, got %q", loaded.neigongBind)
	}

	mock, stores = newCustomConfigMockDB(t)
	mock.ExpectQuery(`SELECT bind_value, shortcut_key1, shortcut_key2, operate_mode, neigong_bind FROM role_keybinds WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"bind_value", "shortcut_key1", "shortcut_key2", "operate_mode", "neigong_bind"}).
			AddRow("1,87", "", "", int32(0), "0;1;0;"))
	loaded, exists = stores.keybind.Load(customConfigTestRoleID)
	if !exists || !loaded.operateModeSet || loaded.operateMode != 0 {
		t.Fatalf("loaded=%+v exists=%t", loaded, exists)
	}
	if loaded.neigongBind != "0;1;0;" {
		t.Fatalf("neigong_bind=%q", loaded.neigongBind)
	}

	mock, stores = newCustomConfigMockDB(t)
	mock.ExpectQuery(`SELECT bind_value, shortcut_key1, shortcut_key2, operate_mode, neigong_bind FROM role_keybinds WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"bind_value", "shortcut_key1", "shortcut_key2", "operate_mode", "neigong_bind"}))
	if _, exists := stores.keybind.Load(customConfigTestRoleID); exists {
		t.Fatal("missing row must read as no archive")
	}
}

// SaveNeiGongBind 首插补 bind_value 空串（role_keybinds NOT NULL 同坑）。
func TestKeybindDBSaveNeiGongBindBootstrapsBindValue(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectExec(`INSERT INTO role_keybinds \(role_id, bind_value, neigong_bind\) VALUES \(\?, '', \?\)`+
		` ON DUPLICATE KEY UPDATE neigong_bind = VALUES\(neigong_bind\)`).
		WithArgs(customConfigTestRoleID, "0;1;2;").
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.keybind.SaveNeiGongBind(customConfigTestRoleID, "0;1;2;"); err != nil {
		t.Fatal(err)
	}
}

func TestShortcutDBListOrdersByIndex(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectQuery(`SELECT shortcut_index, kind, id FROM role_shortcuts WHERE role_id = \? ORDER BY shortcut_index`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"shortcut_index", "kind", "id"}).
			AddRow(int64(9), "skill", "zs_default_01").
			AddRow(int64(10), "skill", "CS_jh_xfz01"))
	loaded, exists := stores.shortcuts.List(customConfigTestRoleID)
	if !exists || len(loaded) != 2 {
		t.Fatalf("loaded=%+v exists=%t", loaded, exists)
	}
	if loaded[0].index != 9 || loaded[0].kind != "skill" || loaded[0].id != "zs_default_01" {
		t.Fatalf("row0=%+v", loaded[0])
	}
	if loaded[1].index != 10 || loaded[1].id != "CS_jh_xfz01" {
		t.Fatalf("row1=%+v", loaded[1])
	}
}

// Upsert：已有 shortcut_index 原位 UPDATE（不再 INSERT）；无行才追加
// seq=IFNULL(MAX(seq),0)+1。
func TestShortcutDBUpsertUpdateThenAppend(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectExec(`UPDATE role_shortcuts SET kind = \?, id = \? WHERE role_id = \? AND shortcut_index = \?`).
		WithArgs("skill", "CS_jh_xfz02", customConfigTestRoleID, int32(10)).
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.shortcuts.Upsert(customConfigTestRoleID, playerShortcut{index: 10, kind: "skill", id: "CS_jh_xfz02"}); err != nil {
		t.Fatal(err)
	}

	mock, stores = newCustomConfigMockDB(t)
	mock.ExpectExec(`UPDATE role_shortcuts SET kind = \?, id = \? WHERE role_id = \? AND shortcut_index = \?`).
		WithArgs("skill", "CS_wd_tjq01", customConfigTestRoleID, int32(20)).
		WillReturnResult(newSQLMockResult(0, 0))
	mock.ExpectExec(`INSERT INTO role_shortcuts \(role_id, seq, shortcut_index, kind, id\)`+
		` SELECT \?, IFNULL\(MAX\(seq\), 0\) \+ 1, \?, \?, \? FROM role_shortcuts WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID, int32(20), "skill", "CS_wd_tjq01", customConfigTestRoleID).
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.shortcuts.Upsert(customConfigTestRoleID, playerShortcut{index: 20, kind: "skill", id: "CS_wd_tjq01"}); err != nil {
		t.Fatal(err)
	}
}

func TestShortcutDBDeleteIndexSQL(t *testing.T) {
	mock, stores := newCustomConfigMockDB(t)
	mock.ExpectExec(`DELETE FROM role_shortcuts WHERE role_id = \? AND shortcut_index = \?`).
		WithArgs(customConfigTestRoleID, int32(9)).
		WillReturnResult(newSQLMockResult(0, 1))
	if err := stores.shortcuts.DeleteIndex(customConfigTestRoleID, 9); err != nil {
		t.Fatal(err)
	}
}
