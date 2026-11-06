package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/local/9yin-go-server/internal/clientdata"
)

var (
	defaultNPCTablePath   = filepath.Join(defaultModernShareRoot, "npc", "npcconfig", "worldnpc", "city_commonnpc.txt")
	defaultNPCCreatorPath = filepath.Join(defaultModernShareRoot, "creator", "npc_creator", "city05_chengdu", "funcnpc.xml")
	// All scene families share the modern npcconfig root. Restricting this to
	// worldnpc silently drops clone, adventure and battle templates even though
	// their creator files and coordinates are present.
	defaultNPCTableDir   = filepath.Join(defaultModernShareRoot, "npc", "npcconfig")
	defaultNPCCreatorDir = filepath.Join(defaultModernShareRoot, "creator", "npc_creator", "city05_chengdu")
)

type npcCatalogStats struct {
	CreatorInstances int
	Resolved         int
	Unresolved       int
	Ambiguous        int
	Samples          []string
}

type npcSceneAuditEntry struct {
	Scene            string   `json:"scene"`
	CreatorInstances int      `json:"creator_instances"`
	Resolved         int      `json:"resolved"`
	Unresolved       int      `json:"unresolved"`
	Ambiguous        int      `json:"ambiguous"`
	Samples          []string `json:"samples,omitempty"`
	Error            string   `json:"error,omitempty"`
}

func loadChengduNPCSpawns(tableDir, creatorDir string, schema clientdata.Schema) ([]npcSpawn, npcCatalogStats, error) {
	tables, err := loadNPCTemplateTables(tableDir)
	if err != nil {
		return nil, npcCatalogStats{}, err
	}
	return loadNPCSceneSpawnsFromTables(tables, creatorDir, schema)
}

// loadNPCTemplateTables loads the shared modern NPC templates once. Scene
// creators only hold placement and per-instance overrides; all scene folders
// resolve against this same table set.
func loadNPCTemplateTables(tableDir string) ([]*clientdata.NPCTemplateTable, error) {
	var tablePaths []string
	err := filepath.WalkDir(tableDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// npcconfig/npc_type.txt is a category dictionary rather than an NPC
		// template. Actual template rows live in the world/clone/battle/life
		// subdirectories and have the required ID column.
		if !entry.IsDir() && filepath.Dir(path) != tableDir && strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			tablePaths = append(tablePaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(tablePaths)
	tables := make([]*clientdata.NPCTemplateTable, 0, len(tablePaths))
	for _, tablePath := range tablePaths {
		file, openErr := os.Open(tablePath)
		if openErr != nil {
			return nil, fmt.Errorf("open template table %s: %w", tablePath, openErr)
		}
		table, loadErr := clientdata.LoadNPCTemplateTable(file, tablePath)
		_ = file.Close()
		if loadErr != nil {
			return nil, loadErr
		}
		tables = append(tables, table)
	}
	return tables, nil
}

// loadNPCSceneSpawnsFromTables reads the scene's declared NPC creator set.
// Each modern scene carries file.ini because its creator subset differs; XML
// beside it such as eventtrigger.xml is not an NPC creator and has no id/x/y/z
// fields, so indiscriminately scanning every XML corrupts the catalog.
func loadNPCSceneSpawnsFromTables(tables []*clientdata.NPCTemplateTable, creatorDir string, schema clientdata.Schema) ([]npcSpawn, npcCatalogStats, error) {
	creatorPaths, err := sceneCreatorPaths(creatorDir)
	if err != nil {
		return nil, npcCatalogStats{}, err
	}
	stats := npcCatalogStats{}
	spawns := make([]npcSpawn, 0, 4096)
	for _, creatorPath := range creatorPaths {
		creatorName := filepath.Base(creatorPath)
		file, openErr := os.Open(creatorPath)
		if openErr != nil {
			return nil, stats, fmt.Errorf("open creator %s: %w", creatorPath, openErr)
		}
		catalog, loadErr := clientdata.LoadNPCCreator(file, creatorPath)
		_ = file.Close()
		if loadErr != nil {
			return nil, stats, loadErr
		}
		stats.CreatorInstances += len(catalog.Instances)
		for _, instance := range catalog.Instances {
			table, ambiguous := selectNPCTemplateTable(tables, creatorName, instance.TemplateID)
			if table == nil {
				stats.Unresolved++
				if ambiguous {
					stats.Ambiguous++
				}
				appendNPCStatSample(&stats, fmt.Sprintf("%s:%s has no unique template", creatorName, instance.TemplateID))
				continue
			}
			resolved, resolveErr := clientdata.ResolveNPC(schema, table, instance, clientdata.ResolveOptions{
				Defaults: clientdata.VisibleNPCSpawnDefaultsCompatV1(),
			})
			if resolveErr != nil {
				stats.Unresolved++
				appendNPCStatSample(&stats, fmt.Sprintf("%s:%s: %v", creatorName, instance.TemplateID, resolveErr))
				continue
			}
			spawn := npcSpawn{resolved: resolved, x: instance.Transform.X, y: instance.Transform.Y, z: instance.Transform.Z}
			if instance.Transform.AY != nil {
				spawn.orient = *instance.Transform.AY
			}
			spawns = append(spawns, spawn)
			stats.Resolved++
		}
	}
	if len(spawns) == 0 {
		return nil, stats, errors.New("no Chengdu NPC creator instance could be resolved")
	}
	return spawns, stats, nil
}

func sceneCreatorPaths(creatorDir string) ([]string, error) {
	manifestPath := filepath.Join(creatorDir, "file.ini")
	manifest, readErr := os.ReadFile(manifestPath)
	if readErr == nil {
		seen := make(map[string]struct{})
		var creatorPaths []string
		for _, line := range strings.Split(string(manifest), "\n") {
			_, rawName, found := strings.Cut(strings.TrimSpace(line), "=")
			if !found || !strings.EqualFold(filepath.Ext(rawName), ".xml") {
				continue
			}
			// EventTrigger.xml is declared by some later school manifests, but
			// its <triggercreator> rows describe scene events and intentionally
			// have no NPC template id. It must never enter the NPC parser.
			if strings.EqualFold(filepath.Base(rawName), "eventtrigger.xml") {
				continue
			}
			path := filepath.Join(creatorDir, rawName)
			key := strings.ToLower(filepath.Clean(path))
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			if _, statErr := os.Stat(path); statErr != nil {
				// Modern resource packages are intentionally partial for some
				// scenes: their file.ini still lists creator variants such as
				// AttackNpc_Creator.xml even when that optional file was omitted.
				// Other declared creators are authoritative and sufficient to
				// enter the scene, so only a real stat failure is fatal.
				if errors.Is(statErr, os.ErrNotExist) {
					continue
				}
				return nil, fmt.Errorf("scene creator manifest %s declares %s: %w", manifestPath, rawName, statErr)
			}
			seen[key] = struct{}{}
			creatorPaths = append(creatorPaths, path)
		}
		if len(creatorPaths) == 0 {
			return nil, fmt.Errorf("scene creator manifest %s declares no XML creator", manifestPath)
		}
		sort.Strings(creatorPaths)
		return creatorPaths, nil
	}
	if !errors.Is(readErr, os.ErrNotExist) {
		return nil, fmt.Errorf("read scene creator manifest %s: %w", manifestPath, readErr)
	}

	// Older or custom directories may not ship file.ini. Keep the fallback
	// intentionally conservative: the known event trigger document is not a
	// spawn source and must never enter the NPC parser.
	var creatorPaths []string
	err := filepath.WalkDir(creatorDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".xml") && !strings.EqualFold(entry.Name(), "eventtrigger.xml") {
			creatorPaths = append(creatorPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("enumerate scene creator %s: %w", creatorDir, err)
	}
	sort.Strings(creatorPaths)
	if len(creatorPaths) == 0 {
		return nil, fmt.Errorf("scene creator %s contains no XML creator", creatorDir)
	}
	return creatorPaths, nil
}

// auditNPCSceneRoot resolves every immediate scene directory independently.
// A broken or obsolete creator file is reported on its own scene instead of
// preventing the remaining modern resources from being inventoried.
func auditNPCSceneRoot(tableDir, creatorRoot string, schema clientdata.Schema) ([]npcSceneAuditEntry, error) {
	tables, err := loadNPCTemplateTables(tableDir)
	if err != nil {
		return nil, err
	}
	directories, err := os.ReadDir(creatorRoot)
	if err != nil {
		return nil, fmt.Errorf("list NPC creator root %s: %w", creatorRoot, err)
	}
	entries := make([]npcSceneAuditEntry, 0, len(directories))
	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}
		name := directory.Name()
		_, stats, loadErr := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, name), schema)
		entry := npcSceneAuditEntry{
			Scene:            name,
			CreatorInstances: stats.CreatorInstances,
			Resolved:         stats.Resolved,
			Unresolved:       stats.Unresolved,
			Ambiguous:        stats.Ambiguous,
			Samples:          stats.Samples,
		}
		if loadErr != nil {
			entry.Error = loadErr.Error()
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Scene < entries[j].Scene })
	return entries, nil
}

func selectNPCTemplateTable(tables []*clientdata.NPCTemplateTable, creatorName, configID string) (*clientdata.NPCTemplateTable, bool) {
	matches := make([]*clientdata.NPCTemplateTable, 0, 2)
	for _, table := range tables {
		if len(table.LookupAll(configID)) == 1 {
			matches = append(matches, table)
		}
	}
	if len(matches) == 0 {
		return nil, false
	}
	preferences := preferredNPCTables(creatorName)
	for _, preferred := range preferences {
		for _, table := range matches {
			if strings.EqualFold(filepath.Base(table.Source), preferred) {
				return table, false
			}
		}
	}
	if len(matches) == 1 {
		return matches[0], false
	}
	return nil, true
}

func preferredNPCTables(creatorName string) []string {
	name := strings.ToLower(creatorName)
	switch {
	case strings.HasPrefix(name, "attacknpc"):
		return []string{"attacknpc.txt", "bossnpc.txt", "monsternpc_commonnpc.txt"}
	case name == "commonnpc.xml" || name == "funcnpc.xml":
		return []string{"city_commonnpc.txt", "scene_commonnpc.txt", "other_commonnpc.txt"}
	default:
		return []string{"other_commonnpc.txt", "scene_commonnpc.txt", "city_commonnpc.txt"}
	}
}

func appendNPCStatSample(stats *npcCatalogStats, sample string) {
	if len(stats.Samples) < 12 {
		stats.Samples = append(stats.Samples, sample)
	}
}

func loadNPCSpawns(tablePath, creatorPath, configID string, schema clientdata.Schema) ([]npcSpawn, error) {
	tableFile, err := os.Open(tablePath)
	if err != nil {
		return nil, fmt.Errorf("open template table %s: %w", tablePath, err)
	}
	defer tableFile.Close()
	table, err := clientdata.LoadNPCTemplateTable(tableFile, tablePath)
	if err != nil {
		return nil, err
	}
	creatorFile, err := os.Open(creatorPath)
	if err != nil {
		return nil, fmt.Errorf("open creator %s: %w", creatorPath, err)
	}
	defer creatorFile.Close()
	catalog, err := clientdata.LoadNPCCreator(creatorFile, creatorPath)
	if err != nil {
		return nil, err
	}
	instances := catalog.InstancesForTemplate(configID)
	if len(instances) == 0 {
		return nil, fmt.Errorf("creator %s contains no instance for %q", creatorPath, configID)
	}
	spawns := make([]npcSpawn, 0, len(instances))
	for _, instance := range instances {
		resolved, resolveErr := clientdata.ResolveNPC(schema, table, instance, clientdata.ResolveOptions{
			Defaults: clientdata.VisibleNPCSpawnDefaultsCompatV1(),
		})
		if resolveErr != nil {
			return nil, resolveErr
		}
		spawn := npcSpawn{resolved: resolved, x: instance.Transform.X, y: instance.Transform.Y, z: instance.Transform.Z}
		if instance.Transform.AY != nil {
			spawn.orient = *instance.Transform.AY
		}
		spawns = append(spawns, spawn)
	}
	return spawns, nil
}

func (n npcSpawn) clone() npcSpawn {
	copyNPC := n
	copyNPC.resolved.Properties = make(map[string]clientdata.PropertyValue, len(n.resolved.Properties))
	for name, value := range n.resolved.Properties {
		copyNPC.resolved.Properties[name] = value
	}
	copyNPC.resolved.Extensions = make(map[string]string, len(n.resolved.Extensions))
	for name, value := range n.resolved.Extensions {
		copyNPC.resolved.Extensions[name] = value
	}
	return copyNPC
}

func (n *npcSpawn) synchronizeTransform() {
	for name, value := range map[string]clientdata.Value{
		"PosiX":  clientdata.Float32Value(n.x),
		"PosiY":  clientdata.Float32Value(n.y),
		"PosiZ":  clientdata.Float32Value(n.z),
		"Orient": clientdata.Float32Value(n.orient),
	} {
		n.resolved.Properties[name] = clientdata.PropertyValue{
			Value:      value,
			Provenance: clientdata.Provenance{Layer: "runtime", Source: "protocol-probe", Field: name},
		}
	}
}

func (n npcSpawn) textProperty(name string) string {
	return n.resolved.Properties[name].Value.Text
}

func (n npcSpawn) int32Property(name string) int32 {
	return n.resolved.Properties[name].Value.I32
}

func appendNPCProperty(msg []byte, property clientdata.IndexedProperty) []byte {
	msg = append(msg, byte(property.Index), byte(property.Index>>8))
	switch property.Value.Type {
	case clientdata.WireByte:
		msg = append(msg, property.Value.U8)
	case clientdata.WireWord:
		msg = append(msg, byte(property.Value.U16), byte(property.Value.U16>>8))
	case clientdata.WireInt32:
		value := uint32(property.Value.I32)
		msg = append(msg, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
	case clientdata.WireFloat32:
		msg = appendFloat32(msg, property.Value.F32)
	case clientdata.WireString:
		msg = appendStringValue(msg, property.Value.Text)
	case clientdata.WireWideString:
		msg = appendWideStringValue(msg, property.Value.Text)
	case clientdata.WireObject:
		msg = binary.LittleEndian.AppendUint64(msg, property.Value.U64)
	default:
		panic(fmt.Sprintf("unsupported NPC wire type %d", property.Value.Type))
	}
	return msg
}

func appendFloat32(msg []byte, value float32) []byte {
	return binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
}

func appendStringValue(msg []byte, value string) []byte {
	msg = binary.LittleEndian.AppendUint32(msg, uint32(len(value)+1))
	msg = append(msg, value...)
	return append(msg, 0)
}

func appendWideStringValue(msg []byte, value string) []byte {
	units := utf16.Encode([]rune(value))
	msg = binary.LittleEndian.AppendUint32(msg, uint32((len(units)+1)*2))
	for _, unit := range units {
		msg = binary.LittleEndian.AppendUint16(msg, unit)
	}
	return binary.LittleEndian.AppendUint16(msg, 0)
}
