package gmapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// 经脉两动作（set_zhenqi_cap/set_jingmai_tier）的契约层域校验——
// 逐条对齐 Java GmRemote.validateCommand（enabled 0/1 + 阈值 1..10⁷；
// jingMaiId 非空 + 档位 1..216）。

func validateBody(t *testing.T, raw string) (Command, error) {
	t.Helper()
	var body CommandBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return Validate(body)
}

func TestValidateZhenQiCap(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_zhenqi_cap","enabled":1,"value":3000}`)
	if err != nil {
		t.Fatalf("valid cap rejected: %v", err)
	}
	if command.Enabled != 1 || command.Value != 3000 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_zhenqi_cap","enabled":2,"value":3000}`,
		`{"action":"set_zhenqi_cap","enabled":-1,"value":3000}`,
		`{"action":"set_zhenqi_cap","enabled":0,"value":0}`,
		`{"action":"set_zhenqi_cap","enabled":0,"value":10000001}`,
	} {
		if _, err := validateBody(t, raw); err == nil {
			t.Fatalf("invalid cap accepted: %s", raw)
		}
	}
}

func TestValidateLootBoxTTL(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_lootbox_ttl","value":300}`)
	if err != nil {
		t.Fatalf("valid ttl rejected: %v", err)
	}
	if command.Value != 300 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_lootbox_ttl","value":-1}`,
		`{"action":"set_lootbox_ttl","value":86401}`,
		`{"action":"set_lootbox_ttl","value":1.5}`,
	} {
		_, err := validateBody(t, raw)
		if err == nil {
			t.Fatalf("invalid ttl accepted: %s", raw)
		}
		if !strings.Contains(err.Error(), "lootbox ttl") {
			t.Fatalf("error %q should name the offending field", err)
		}
	}
}

func TestValidateJingMaiTier(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":36}`)
	if err != nil {
		t.Fatalf("valid tier rejected: %v", err)
	}
	if command.JingMaiID != "jm_dantian" || command.JingMaiTier != 36 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_jingmai_tier","jingMaiId":"","jingMaiTier":36}`,
		`{"action":"set_jingmai_tier","jingMaiTier":36}`,
		`{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":0}`,
		`{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":217}`,
	} {
		_, err := validateBody(t, raw)
		if err == nil {
			t.Fatalf("invalid tier accepted: %s", raw)
		}
		if !strings.Contains(err.Error(), "jingMai") && !strings.Contains(err.Error(), "jingmai") {
			t.Fatalf("error %q should name the offending field", err)
		}
	}
}
