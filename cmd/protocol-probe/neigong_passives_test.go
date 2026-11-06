package main

import (
	"strings"
	"testing"
	"time"
)

func TestYihuaPassiveProfileUsesInstalledLevelThreeDescription(t *testing.T) {
	catalog, err := loadInnerPowerPassiveCatalog()
	if err != nil {
		t.Fatalf("load inner-power descriptions: %v", err)
	}
	if len(catalog.descriptions) < 100 {
		t.Fatalf("compiled only %d inner-power descriptions", len(catalog.descriptions))
	}
	t.Logf("inner-power source families=%d executable passive families=%d", len(catalog.descriptions), len(catalog.profiles))
	profile, ok := catalog.profiles["buf_ng_yh_001"][3]
	if !ok {
		t.Fatal("missing 明玉神功 BuffLevel 3 profile")
	}
	if profile.procChance != 50 || profile.dodgeAdd != 20 || profile.procLifetime != 7*time.Second || profile.procCooldown != 32*time.Second {
		t.Fatalf("unexpected 料敌先机 profile: %+v", profile)
	}
	if profile.healPercent != 3 || profile.healCooldown != 4*time.Second {
		t.Fatalf("unexpected 移花接玉 profile: %+v", profile)
	}
	if profile.procBuffID != "buf_ng_yh_001_5" || profile.procStatic != 13107 {
		t.Fatalf("unexpected native proc Buff: %+v", profile)
	}
}

func TestYihuaPassiveTriggersNativeBuffAndHealsOnDodge(t *testing.T) {
	start := time.Unix(1_700_000_000, 0).UTC()
	for second := int64(0); second < 1_000; second++ {
		player := newPlayerActor("明玉测试", 0)
		if err := player.equipNeiGong("ng_yh_001"); err != nil {
			t.Fatalf("equip 明玉神功: %v", err)
		}
		player.actor.ApplyDamage(1_000)
		before := player.actor.Snapshot().HP
		now := start.Add(time.Duration(second) * time.Second)
		outcome := player.resolveNPCAttack(77, 10, now)
		if !outcome.innerPowerTriggered || !outcome.dodged {
			continue
		}
		if outcome.innerPowerHeal <= 0 || player.actor.Snapshot().HP <= before {
			t.Fatalf("移花接玉 did not heal: outcome=%+v before=%d after=%d", outcome, before, player.actor.Snapshot().HP)
		}
		if got := player.bufferInfo(yihuaPassiveBuffSlot, now); !strings.HasPrefix(got, "3333,") {
			t.Fatalf("料敌先机 BufferInfo=%q, want StaticData 0x3333", got)
		}
		if got := player.bufferListString(now); !strings.Contains(got, "buf_ng_yh_001_5,local-10,3,1-1,") {
			t.Fatalf("料敌先机 BufferListStr=%q", got)
		}
		return
	}
	t.Fatal("no deterministic test tick produced 明玉神功 proc+dodge")
}
