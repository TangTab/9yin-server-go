package clientdata

import (
	"bufio"
	"bytes"
	"path"
	"strings"
)

// dependencyKey describes the resource references understood by the modern
// composite/action loader. Keeping this parser separate from the filesystem
// index makes it possible to test the expansion rules without package files.
type dependencyKey struct {
	Key      string
	Relation string
	Kind     ResourceKind
}

var compositeDependencyKeys = map[string]dependencyKey{
	"action":     {Key: "Action", Relation: "action", Kind: ResourceAction},
	"model":      {Key: "Model", Relation: "model", Kind: ResourceModel},
	"main_model": {Key: "main_model", Relation: "main_model", Kind: ResourceModel},
}

var actionDependencyKeys = map[string]dependencyKey{
	"action_base_file":  {Key: "ACTION_BASE_FILE", Relation: "action_base", Kind: ResourceAction},
	"action_child_file": {Key: "ACTION_CHILD_FILE", Relation: "action_child", Kind: ResourceAction},
	"skeleton":          {Key: "Skeleton", Relation: "skeleton", Kind: ResourceSkeleton},
}

type parsedReference struct {
	Key      string
	Relation string
	Raw      string
	Kind     ResourceKind
}

// parseINIReferences deliberately preserves duplicate keys. Some action files
// have multiple ACTION_CHILD_FILE entries and a map-based INI reader silently
// loses all but one of them.
func parseINIReferences(data []byte, keys map[string]dependencyKey) []parsedReference {
	var refs []parsedReference
	s := bufio.NewScanner(bytes.NewReader(stripUTF8BOM(data)))
	// Asset INI lines can contain long action lists. Do not inherit Scanner's
	// small default token limit.
	s.Buffer(make([]byte, 4096), 4<<20)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 1 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		spec, ok := keys[strings.ToLower(key)]
		if !ok {
			continue
		}
		value := trimINIValue(line[i+1:])
		if value == "" {
			continue
		}
		// Child files are found both as repeated keys and as pipe/comma/semicolon
		// lists in client data. Splitting is restricted to CHILD_FILE so ordinary
		// Windows paths are never changed.
		values := []string{value}
		if strings.EqualFold(spec.Key, "ACTION_CHILD_FILE") {
			values = splitChildFiles(value)
		}
		for _, value := range values {
			if value != "" {
				refs = append(refs, parsedReference{Key: spec.Key, Relation: spec.Relation, Raw: value, Kind: spec.Kind})
			}
		}
	}
	return refs
}

func stripUTF8BOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
}

func trimINIValue(value string) string {
	value = strings.TrimSpace(value)
	// The game data uses semicolon comments after whitespace. A semicolon in a
	// CHILD_FILE list is handled before this only when it has no leading space.
	if i := strings.Index(value, " ;"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return strings.Trim(strings.TrimSpace(value), "\"'")
}

func splitChildFiles(value string) []string {
	f := func(r rune) bool { return r == '|' || r == ',' || r == ';' }
	parts := strings.FieldsFunc(value, f)
	for i := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(parts[i]), "\"'")
	}
	return parts
}

// resolveReference follows FxModelAdv's observed action-file behavior: a path
// with an explicit virtual-root prefix is kept absolute; all other paths are
// expanded relative to the referring file. It intentionally retains the input
// separator style so the report can expose '/' versus '\\' lookup failures.
func resolveReference(from, raw string, virtualRoots map[string]bool) string {
	raw = strings.TrimSpace(strings.Trim(raw, "\"'"))
	if raw == "" {
		return ""
	}
	sep := `\`
	if strings.Contains(raw, "/") && !strings.Contains(raw, `\`) {
		sep = "/"
	}
	normalized := strings.ReplaceAll(raw, `\`, "/")
	first := strings.ToLower(strings.SplitN(strings.TrimLeft(normalized, "/"), "/", 2)[0])
	if strings.HasPrefix(raw, `/`) || strings.HasPrefix(raw, `\`) || virtualRoots[first] {
		normalized = strings.TrimLeft(normalized, "/")
	} else {
		base := path.Dir(strings.ReplaceAll(from, `\`, "/"))
		normalized = path.Join(base, normalized)
	}
	normalized = path.Clean(normalized)
	if normalized == "." {
		return ""
	}
	if sep == `\` {
		return strings.ReplaceAll(normalized, "/", `\`)
	}
	return normalized
}
