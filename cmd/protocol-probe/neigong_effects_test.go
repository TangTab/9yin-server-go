package main

import (
	"testing"
	"time"
)

// This is deliberately resource-backed: it protects the real modern chain
// StaticData -> exact current-level varprop -> PropModify packs, rather than
// preserving the old level-30 starter-book constants.
func TestModernNeiGongEffectUsesCurrentLevelVarProp(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Fatalf("load modern inner-power catalog: %v", err)
	}
	effect, err := catalog.effect(114, 32) // ng_yh_001 / 明玉神功, role 2's persisted level.
	if err != nil {
		t.Fatalf("evaluate ng_yh_001 level 32: %v", err)
	}
	if effect.buffID != "buf_ng_yh_001" || effect.staticData != 13073 || effect.buffLevel != 3 {
		t.Fatalf("unexpected buff route: id=%q static=%d level=%d", effect.buffID, effect.staticData, effect.buffLevel)
	}
	want := map[string]int32{
		"StrAdd": 57, "StaAdd": 59, "DexAdd": 72, "IngAdd": 101, "SpiAdd": 69,
		"MaxHPAdd": 4230, "MaxMPAdd": 1180, "MaxParryAdd": 5160, "MinMagicDefAdd": 105,
	}
	for name, value := range want {
		if got := effect.stats[name]; got != value {
			t.Errorf("%s = %d, want %d", name, got, value)
		}
	}
}

func TestPlayerProgressRestoresExactEquippedEffect(t *testing.T) {
	progress := newPlayerProgress()
	saved := progress.snapshot()
	saved.CurNeiGong = "ng_yh_001"
	for index := range saved.Books {
		if saved.Books[index].ConfigID == "ng_yh_001" {
			saved.Books[index].Level = 32
		}
	}
	progress.restore(saved, time.Now().UTC())
	book := progress.book(progress.curNeiGong)
	if book == nil {
		t.Fatal("missing restored equipped book")
	}
	if book.level != 32 || book.maxHPAdd != 4230 || book.maxMPAdd != 1180 || book.ingAdd != 101 {
		t.Fatalf("restored effect = level %d hp %d mp %d ing %d, want 32/4230/1180/101", book.level, book.maxHPAdd, book.maxMPAdd, book.ingAdd)
	}
}

func TestEveryStarterBookResolvesItsExactModernLevel(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, book := range newPlayerProgress().books {
		_, err := catalog.effect(book.staticData, book.level)
		if err != nil {
			t.Errorf("%s StaticData=%d level=%d: %v", book.configID, book.staticData, book.level, err)
		}
	}
}
