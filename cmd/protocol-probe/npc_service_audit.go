package main

import (
	"sort"
	"strings"

	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
)

// npcServiceAuditReport is a data-only inventory. It deliberately separates
// configuration discovered from an NPC from functionality implemented by this
// server, so a menu cannot silently turn an unimplemented service into a fake
// shop or transport action.
type npcServiceAuditReport struct {
	Scene role.Scene           `json:"scene"`
	Stats npcCatalogStats      `json:"catalog_stats"`
	NPCs  []npcServiceAuditNPC `json:"npcs"`
}

type npcServiceAuditNPC struct {
	ConfigID    string                   `json:"config_id"`
	InstanceKey string                   `json:"instance_key"`
	ScriptClass string                   `json:"script_class,omitempty"`
	Position    npcServiceAuditPosition  `json:"position"`
	Services    []npcServiceAuditService `json:"services,omitempty"`
	NpcFuncIDs  []int                    `json:"npc_func_ids,omitempty"`
}

type npcServiceAuditPosition struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type npcServiceAuditService struct {
	Mark        uint16 `json:"mark"`
	Label       string `json:"label"`
	Value       string `json:"value,omitempty"`
	Source      string `json:"source"`
	Implemented bool   `json:"implemented"`
	Ready       bool   `json:"ready"`
	Issue       string `json:"issue,omitempty"`
}

func buildNPCServiceAudit(scene role.Scene, stats npcCatalogStats, catalog []npcSpawn, funcs *npcfunc.Registry) npcServiceAuditReport {
	report := npcServiceAuditReport{
		Scene: scene,
		Stats: stats,
		NPCs:  make([]npcServiceAuditNPC, 0, len(catalog)),
	}
	for _, npc := range catalog {
		entry := npcServiceAuditNPC{
			ConfigID:    npc.resolved.ConfigID,
			InstanceKey: npc.resolved.InstanceKey,
			ScriptClass: npc.resolved.ScriptClass,
			Position:    npcServiceAuditPosition{X: npc.x, Y: npc.y, Z: npc.z},
		}
		for _, service := range modernNPCServices(npc) {
			implemented, ready, issue := auditNPCServiceReadiness(service)
			entry.Services = append(entry.Services, npcServiceAuditService{
				Mark:        service.mark,
				Label:       service.label,
				Value:       service.value,
				Source:      service.source,
				Implemented: implemented,
				Ready:       ready,
				Issue:       issue,
			})
		}
		for _, binding := range funcs.FuncsForNPC(npc.resolved.ConfigID) {
			entry.NpcFuncIDs = append(entry.NpcFuncIDs, binding.FuncID)
		}
		sort.Ints(entry.NpcFuncIDs)
		report.NPCs = append(report.NPCs, entry)
	}
	sort.SliceStable(report.NPCs, func(i, j int) bool {
		if report.NPCs[i].ConfigID == report.NPCs[j].ConfigID {
			return report.NPCs[i].InstanceKey < report.NPCs[j].InstanceKey
		}
		return report.NPCs[i].ConfigID < report.NPCs[j].ConfigID
	})
	return report
}

func auditNPCServiceReadiness(service npcService) (implemented, ready bool, issue string) {
	switch service.mark {
	case markDepot:
		return true, true, ""
	case markShop:
		if _, _, _, err := loadShopCatalogSection(defaultShopINIPath, service.value); err != nil {
			return true, false, err.Error()
		}
		return true, true, ""
	default:
		return false, false, "server handler is not implemented"
	}
}

func auditFileComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
