package npcfunc

import (
	"os"
	"testing"
)

func TestLoadIndexesEnabledFunctionsPerNPC(t *testing.T) {
	path := t.TempDir() + "\\Npc_Func.xml"
	contents := `<Object><Property NpcID="npc_a" FuncID="20" EnableFunc="1"/><Property FuncID="7" NpcID="npc_a"/><Property NpcID="npc_a" FuncID="22" EnableFunc="0"/><Property NpcID="npc_b" FuncID="47"/></Object>`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reg.FuncsForNPC("NPC_A")
	if len(got) != 2 || got[0].FuncID != 7 || got[1].FuncID != 20 {
		t.Fatalf("npc_a functions = %#v", got)
	}
}
