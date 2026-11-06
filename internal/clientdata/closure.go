package clientdata

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ClosureSchemaVersion = "nineyin.clientdata.resource-closure.v1"

type ResourceKind string

const (
	ResourceComposite ResourceKind = "composite"
	ResourceAction    ResourceKind = "action"
	ResourceModel     ResourceKind = "model"
	ResourceSkeleton  ResourceKind = "skeleton"
	ResourceUnknown   ResourceKind = "unknown"
)

type ResourceStatus string

const (
	ResourceHit                 ResourceStatus = "hit"
	ResourceUnreadable          ResourceStatus = "unreadable"
	ResourceSeparatorMismatch   ResourceStatus = "separator_mismatch"
	ResourcePackageNotExtracted ResourceStatus = "package_not_extracted"
	ResourceMissing             ResourceStatus = "missing"
)

// KnownPackage records packages.ini knowledge even when its package has not
// been extracted. Prefixes are audit hints, not lookup aliases: the strict
// FxPackage separator behavior still comes exclusively from Index.Lookup.
type KnownPackage struct {
	Name        string   `json:"name"`
	File        string   `json:"file"`
	Prefixes    []string `json:"prefixes,omitempty"`
	LoadOrder   int      `json:"load_order"`
	Extracted   bool     `json:"extracted"`
	ExtractRoot string   `json:"extract_root,omitempty"`
}

type ClosureOptions struct {
	Entry        string
	Packages     []KnownPackage
	VirtualRoots []string
	GeneratedAt  time.Time
	MaxResources int
}

type ClosureSource struct {
	Package      string `json:"package"`
	PhysicalPath string `json:"physical_path"`
	RelativePath string `json:"relative_path"`
	Size         int64  `json:"size"`
	Readable     bool   `json:"readable"`
	ReadError    string `json:"read_error,omitempty"`
	Winner       bool   `json:"winner"`
	Shadowed     bool   `json:"shadowed"`
}

type ResourceNode struct {
	ID                  int             `json:"id"`
	Path                string          `json:"path"`
	Kind                ResourceKind    `json:"kind"`
	Status              ResourceStatus  `json:"status"`
	CaseFolded          bool            `json:"case_folded,omitempty"`
	SeparatorCandidates []string        `json:"separator_candidates,omitempty"`
	ExpectedPackages    []string        `json:"expected_packages,omitempty"`
	Sources             []ClosureSource `json:"sources,omitempty"`
	ParseError          string          `json:"parse_error,omitempty"`
}

type ResourceEdge struct {
	From     int    `json:"from"`
	To       int    `json:"to"`
	Key      string `json:"key"`
	Relation string `json:"relation"`
	Raw      string `json:"raw"`
	Resolved string `json:"resolved"`
}

type ClosureSummary struct {
	Resources            int `json:"resources"`
	Edges                int `json:"edges"`
	Hits                 int `json:"hits"`
	Unreadable           int `json:"unreadable"`
	SeparatorMismatches  int `json:"separator_mismatches"`
	PackagesNotExtracted int `json:"packages_not_extracted"`
	Missing              int `json:"missing"`
	Shadowed             int `json:"shadowed"`
}

type ClosureReport struct {
	SchemaVersion string         `json:"schema_version"`
	GeneratedAt   string         `json:"generated_at"`
	Entry         string         `json:"entry"`
	Packages      []KnownPackage `json:"packages"`
	Resources     []ResourceNode `json:"resources"`
	Edges         []ResourceEdge `json:"edges"`
	Summary       ClosureSummary `json:"summary"`
}

// AuditResourceClosure follows the dependency subset observed in the modern
// model stack: COMPOSITE Action/Model/main_model and action file
// ACTION_BASE_FILE/ACTION_CHILD_FILE/Skeleton. Model and skeleton binaries are
// intentionally reported as leaves; parsing proprietary xmod geometry chunks
// belongs to a separate binary-format pass.
func AuditResourceClosure(index *Index, options ClosureOptions) (ClosureReport, error) {
	if index == nil {
		return ClosureReport{}, fmt.Errorf("clientdata: nil index")
	}
	entry := strings.TrimSpace(options.Entry)
	if entry == "" {
		return ClosureReport{}, fmt.Errorf("clientdata: resource closure has no entry")
	}
	if options.MaxResources <= 0 {
		options.MaxResources = 10000
	}
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	report := ClosureReport{
		SchemaVersion: ClosureSchemaVersion,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		Entry:         entry,
		Packages:      append([]KnownPackage(nil), options.Packages...),
	}
	roots := make(map[string]bool)
	for _, root := range options.VirtualRoots {
		roots[strings.ToLower(strings.Trim(root, `/\`))] = true
	}
	for _, pkg := range options.Packages {
		for _, prefix := range pkg.Prefixes {
			first := strings.SplitN(strings.ReplaceAll(prefix, `\`, "/"), "/", 2)[0]
			roots[strings.ToLower(first)] = true
		}
	}

	type pending struct {
		path string
		kind ResourceKind
	}
	queue := []pending{{path: entry, kind: ResourceComposite}}
	ids := make(map[string]int)
	for len(queue) != 0 {
		item := queue[0]
		queue = queue[1:]
		key := foldASCII(item.path)
		if _, exists := ids[key]; exists {
			continue
		}
		if len(report.Resources) >= options.MaxResources {
			return report, fmt.Errorf("clientdata: resource closure exceeds limit %d", options.MaxResources)
		}
		node := inspectResource(index, item.path, item.kind, options.Packages)
		node.ID = len(report.Resources)
		ids[key] = node.ID
		report.Resources = append(report.Resources, node)
		if node.Status != ResourceHit || (item.kind != ResourceComposite && item.kind != ResourceAction) {
			continue
		}
		data, err := index.ReadFile(item.path)
		if err != nil {
			report.Resources[node.ID].Status = ResourceUnreadable
			report.Resources[node.ID].ParseError = err.Error()
			continue
		}
		keys := actionDependencyKeys
		if item.kind == ResourceComposite {
			keys = compositeDependencyKeys
		}
		for _, ref := range parseINIReferences(data, keys) {
			resolved := resolveReference(item.path, ref.Raw, roots)
			if resolved == "" {
				continue
			}
			childKey := foldASCII(resolved)
			childID, exists := ids[childKey]
			if !exists {
				childID = len(report.Resources) + len(queue)
				// Multiple parents may enqueue the same child. The final ID is fixed
				// after traversal, so edges are repaired below by resolved path.
				queue = append(queue, pending{path: resolved, kind: ref.Kind})
			}
			report.Edges = append(report.Edges, ResourceEdge{
				From: node.ID, To: childID, Key: ref.Key, Relation: ref.Relation,
				Raw: ref.Raw, Resolved: resolved,
			})
		}
	}
	// Repair speculative child IDs after breadth-first traversal and stable-sort
	// edges so JSON diffs remain deterministic.
	for i := range report.Edges {
		report.Edges[i].To = ids[foldASCII(report.Edges[i].Resolved)]
	}
	sort.SliceStable(report.Edges, func(i, j int) bool {
		if report.Edges[i].From != report.Edges[j].From {
			return report.Edges[i].From < report.Edges[j].From
		}
		if report.Edges[i].Relation != report.Edges[j].Relation {
			return report.Edges[i].Relation < report.Edges[j].Relation
		}
		return foldASCII(report.Edges[i].Resolved) < foldASCII(report.Edges[j].Resolved)
	})
	computeClosureSummary(&report)
	return report, nil
}

func inspectResource(index *Index, virtualPath string, kind ResourceKind, packages []KnownPackage) ResourceNode {
	result := index.Lookup(virtualPath)
	node := ResourceNode{Path: virtualPath, Kind: kind, CaseFolded: result.CaseFolded}
	node.ExpectedPackages = expectedPackageNames(virtualPath, packages)
	if result.Entry == nil {
		node.SeparatorCandidates = append([]string(nil), result.SeparatorCandidates...)
		if len(result.SeparatorCandidates) != 0 {
			node.Status = ResourceSeparatorMismatch
			return node
		}
		for _, name := range node.ExpectedPackages {
			for _, pkg := range packages {
				if pkg.Name == name && !pkg.Extracted {
					node.Status = ResourcePackageNotExtracted
					return node
				}
			}
		}
		node.Status = ResourceMissing
		return node
	}
	node.Status = ResourceHit
	for i, source := range result.Entry.Sources {
		node.Sources = append(node.Sources, ClosureSource{
			Package: source.Mount, PhysicalPath: filepath.Clean(source.PhysicalPath),
			RelativePath: source.RelativePath, Size: source.Size, Readable: source.Readable,
			ReadError: source.ReadError, Winner: i == 0, Shadowed: i > 0,
		})
		if i == 0 && !source.Readable {
			node.Status = ResourceUnreadable
		}
	}
	return node
}

func expectedPackageNames(virtualPath string, packages []KnownPackage) []string {
	pathValue := strings.ToLower(strings.ReplaceAll(virtualPath, "/", `\`))
	best := -1
	var names []string
	for _, pkg := range packages {
		for _, prefix := range pkg.Prefixes {
			prefix = strings.ToLower(strings.Trim(prefix, `/\`))
			if prefix == "" || !(pathValue == prefix || strings.HasPrefix(pathValue, prefix+`\`) || strings.HasPrefix(pathValue, prefix+"_")) {
				continue
			}
			if len(prefix) > best {
				best, names = len(prefix), []string{pkg.Name}
			} else if len(prefix) == best {
				names = append(names, pkg.Name)
			}
		}
	}
	return names
}

func computeClosureSummary(report *ClosureReport) {
	report.Summary.Resources = len(report.Resources)
	report.Summary.Edges = len(report.Edges)
	for _, node := range report.Resources {
		switch node.Status {
		case ResourceHit:
			report.Summary.Hits++
		case ResourceUnreadable:
			report.Summary.Unreadable++
		case ResourceSeparatorMismatch:
			report.Summary.SeparatorMismatches++
		case ResourcePackageNotExtracted:
			report.Summary.PackagesNotExtracted++
		case ResourceMissing:
			report.Summary.Missing++
		}
		for _, source := range node.Sources {
			if source.Shadowed {
				report.Summary.Shadowed++
			}
		}
	}
}
