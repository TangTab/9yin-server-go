package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func TestBuildNPCServiceAuditSeparatesConfiguredFromImplemented(t *testing.T) {
	npc := testNPCSpawn()
	npc.resolved.Extensions = map[string]string{
		"template.ShopID":      "Shop_yaopin_00100",
		"template.TransFuncID": "37",
	}
	report := buildNPCServiceAudit(role.Scene{Config: "ini\\scene\\scene_city05", Resource: "city05"}, npcCatalogStats{Resolved: 1}, []npcSpawn{npc}, nil)
	if len(report.NPCs) != 1 || len(report.NPCs[0].Services) != 2 {
		t.Fatalf("audit=%#v", report)
	}
	if !report.NPCs[0].Services[0].Implemented || !report.NPCs[0].Services[0].Ready || report.NPCs[0].Services[1].Implemented || report.NPCs[0].Services[1].Ready {
		t.Fatalf("implemented statuses=%#v", report.NPCs[0].Services)
	}
}
