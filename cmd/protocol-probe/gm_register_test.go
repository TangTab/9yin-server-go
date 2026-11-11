package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/role"
)

// TestGMRegisterEndpoint 注册端点全链：建号→改密→空值拒绝，
// 且注册键与登录帧密文哈希同域（auth.AccountKeyFor 直连验证）。
func TestGMRegisterEndpoint(t *testing.T) {
	repository, err := role.OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newGMMux(newGMHub(), repository))
	defer server.Close()

	post := func(body map[string]string) (int, map[string]any) {
		encoded, _ := json.Marshal(body)
		response, err := http.Post(server.URL+"/api/register", "application/json", bytes.NewReader(encoded))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var decoded map[string]any
		_ = json.NewDecoder(response.Body).Decode(&decoded)
		return response.StatusCode, decoded
	}

	status, body := post(map[string]string{"account": "paneluser", "password": "panel-pass"})
	if status != http.StatusOK {
		t.Fatalf("create: status=%d body=%v", status, body)
	}
	if body["account"] != "paneluser" {
		t.Fatalf("create body=%v", body)
	}
	if id, ok := body["account_id"].(float64); !ok || id <= 0 {
		t.Fatalf("create account_id missing: %v", body)
	}

	status, body = post(map[string]string{"account": "paneluser", "password": "changed"})
	if status != http.StatusOK || body["updated"] != true {
		t.Fatalf("update: status=%d body=%v", status, body)
	}

	if status, _ = post(map[string]string{"account": "", "password": "x"}); status != http.StatusBadRequest {
		t.Fatalf("empty account: status=%d", status)
	}
	if status, _ = post(map[string]string{"account": "y", "password": ""}); status != http.StatusBadRequest {
		t.Fatalf("empty password: status=%d", status)
	}

	// 登录侧连通：注册键可被 login 命中且验密生效。
	key, err := auth.AccountKeyFor("paneluser")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FindByKey(context.Background(), role.AccountKey(key)); err != nil {
		t.Fatalf("registered key not found: %v", err)
	}
	verifier := auth.ComputePasswordVerifier("paneluser", "changed")
	if !bytes.Equal(verifier, mustVerifier(t, repository, key)) {
		t.Fatal("stored verifier differs from recomputation")
	}
}

func mustVerifier(t *testing.T, repository *role.JSONRepository, key string) []byte {
	t.Helper()
	account, err := repository.FindByKey(context.Background(), role.AccountKey(key))
	if err != nil {
		t.Fatal(err)
	}
	if len(account.PasswordHash) == 0 {
		t.Fatal("no verifier stored")
	}
	return account.PasswordHash
}
