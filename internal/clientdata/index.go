// Package clientdata indexes files extracted from modern client packages.
//
// It intentionally models FxPackage's virtual-name rules instead of applying
// filepath.Clean: ASCII letters are case-insensitive, but slash and backslash
// remain different bytes. Entries from earlier mounts shadow equal names from
// later mounts.
package clientdata

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrNotFound = errors.New("clientdata: virtual file not found")

// Mount describes one extracted package tree. Directory must be the directory
// corresponding to the package's virtual root (for the usual extractor output,
// this is <name>.package.files\res). VirtualPrefix is prepended without changing
// its separator style. Mount order is package load order and therefore shadow
// priority.
type Mount struct {
	Name          string
	Directory     string
	VirtualPrefix string
}

type Config struct {
	Mounts          []Mount
	FailOnRootError bool
}

// ExtractedPackageMount adapts the directory emitted by the current unpacker.
// The unpacker stores package payload below a physical "res" directory while
// "res" itself is not part of the package's virtual file name.
func ExtractedPackageMount(name, extractionDirectory string) Mount {
	return Mount{Name: name, Directory: filepath.Join(extractionDirectory, "res")}
}

type DiagnosticKind string

const (
	DiagnosticRoot       DiagnosticKind = "root"
	DiagnosticWalk       DiagnosticKind = "walk"
	DiagnosticUnreadable DiagnosticKind = "unreadable"
	DiagnosticShadowed   DiagnosticKind = "shadowed"
)

// Diagnostic is a non-fatal indexing observation unless FailOnRootError makes
// root and walk failures fatal. Error text is retained rather than an error
// value so an Index stays immutable and easy to serialize in tooling.
type Diagnostic struct {
	Kind         DiagnosticKind
	Mount        string
	VirtualPath  string
	PhysicalPath string
	Error        string
}

type Source struct {
	Mount        string
	MountIndex   int
	PhysicalPath string
	RelativePath string
	Size         int64
	Readable     bool
	ReadError    string
}

// Entry contains every source of one engine-equivalent virtual name. Sources
// are in mount priority order; Sources[0] is the effective source.
type Entry struct {
	VirtualPath string
	Sources     []Source
}

func (e Entry) Winner() Source { return e.Sources[0] }
func (e Entry) Shadowed() bool { return len(e.Sources) > 1 }

type LookupResult struct {
	Entry               *Entry
	CaseFolded          bool
	SeparatorCandidates []string
}

// Found reports a strict FxPackage-style match. SeparatorCandidates are only
// diagnostics and never make Found true.
func (r LookupResult) Found() bool { return r.Entry != nil }

type Index struct {
	entries      map[string]*Entry
	ordered      []*Entry
	bySeparators map[string][]*Entry
	diagnostics  []Diagnostic
}

// Build recursively scans configured mounts without modifying any client file.
// Missing roots are recorded as diagnostics by default so a partial extraction
// can still be inspected. Set FailOnRootError for startup validation.
func Build(config Config) (*Index, error) {
	return build(config, probeReadable)
}

type readabilityProbe func(string) error

func build(config Config, probe readabilityProbe) (*Index, error) {
	if len(config.Mounts) == 0 {
		return nil, errors.New("clientdata: no mounts configured")
	}
	index := &Index{
		entries:      make(map[string]*Entry),
		bySeparators: make(map[string][]*Entry),
	}
	seenMounts := make(map[string]struct{}, len(config.Mounts))
	for mountIndex, mount := range config.Mounts {
		mount.Name = strings.TrimSpace(mount.Name)
		if mount.Name == "" {
			return nil, fmt.Errorf("clientdata: mount %d has no name", mountIndex)
		}
		if _, exists := seenMounts[mount.Name]; exists {
			return nil, fmt.Errorf("clientdata: duplicate mount name %q", mount.Name)
		}
		seenMounts[mount.Name] = struct{}{}
		if mount.Directory == "" {
			return nil, fmt.Errorf("clientdata: mount %q has no directory", mount.Name)
		}
		prefix, err := validatePrefix(mount.VirtualPrefix)
		if err != nil {
			return nil, fmt.Errorf("clientdata: mount %q: %w", mount.Name, err)
		}
		root, err := filepath.Abs(mount.Directory)
		if err != nil {
			return nil, fmt.Errorf("clientdata: mount %q: %w", mount.Name, err)
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			if err == nil {
				err = errors.New("not a directory")
			}
			diagnostic := Diagnostic{Kind: DiagnosticRoot, Mount: mount.Name, PhysicalPath: root, Error: err.Error()}
			index.diagnostics = append(index.diagnostics, diagnostic)
			if config.FailOnRootError {
				return nil, fmt.Errorf("clientdata: mount %q root %q: %w", mount.Name, root, err)
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticWalk, Mount: mount.Name, PhysicalPath: path, Error: walkErr.Error()})
				if config.FailOnRootError {
					return walkErr
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticWalk, Mount: mount.Name, PhysicalPath: path, Error: infoErr.Error()})
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			// Extracted package names are recorded with Windows separators. Do
			// not normalize lookup input later: slash inequality is intentional.
			relative = strings.ReplaceAll(relative, "/", `\`)
			virtualPath := relative
			if prefix != "" {
				virtualPath = prefix + `\` + relative
			}
			source := Source{
				Mount:        mount.Name,
				MountIndex:   mountIndex,
				PhysicalPath: path,
				RelativePath: relative,
				Size:         info.Size(),
				Readable:     true,
			}
			if readErr := probe(path); readErr != nil {
				source.Readable = false
				source.ReadError = readErr.Error()
				index.diagnostics = append(index.diagnostics, Diagnostic{Kind: DiagnosticUnreadable, Mount: mount.Name, VirtualPath: virtualPath, PhysicalPath: path, Error: readErr.Error()})
			}
			key := foldASCII(virtualPath)
			if existing := index.entries[key]; existing != nil {
				existing.Sources = append(existing.Sources, source)
				index.diagnostics = append(index.diagnostics, Diagnostic{
					Kind: DiagnosticShadowed, Mount: mount.Name, VirtualPath: virtualPath, PhysicalPath: path,
					Error: fmt.Sprintf("shadowed by mount %q at %s", existing.Sources[0].Mount, existing.Sources[0].PhysicalPath),
				})
				return nil
			}
			entryValue := &Entry{VirtualPath: virtualPath, Sources: []Source{source}}
			index.entries[key] = entryValue
			index.ordered = append(index.ordered, entryValue)
			index.bySeparators[separatorDiagnosticKey(virtualPath)] = append(index.bySeparators[separatorDiagnosticKey(virtualPath)], entryValue)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("clientdata: scan mount %q: %w", mount.Name, err)
		}
	}
	sort.Slice(index.ordered, func(i, j int) bool {
		return foldASCII(index.ordered[i].VirtualPath) < foldASCII(index.ordered[j].VirtualPath)
	})
	return index, nil
}

func validatePrefix(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, `/`) || strings.HasPrefix(value, `\`) || strings.HasSuffix(value, `/`) || strings.HasSuffix(value, `\`) {
		return "", errors.New("virtual prefix must not start or end with a separator")
	}
	if strings.Contains(value, `/`) {
		return "", errors.New("virtual prefix must use backslashes")
	}
	return value, nil
}

func probeReadable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var one [1]byte
	_, readErr := file.Read(one[:])
	if errors.Is(readErr, io.EOF) {
		readErr = nil // An empty regular file is readable.
	}
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	return closeErr
}

func (index *Index) Lookup(virtualPath string) LookupResult {
	if index == nil || virtualPath == "" {
		return LookupResult{}
	}
	if entry := index.entries[foldASCII(virtualPath)]; entry != nil {
		return LookupResult{Entry: cloneEntry(entry), CaseFolded: virtualPath != entry.VirtualPath}
	}
	candidates := index.bySeparators[separatorDiagnosticKey(virtualPath)]
	result := LookupResult{SeparatorCandidates: make([]string, 0, len(candidates))}
	for _, candidate := range candidates {
		result.SeparatorCandidates = append(result.SeparatorCandidates, candidate.VirtualPath)
	}
	sort.Strings(result.SeparatorCandidates)
	return result
}

func (index *Index) Entry(virtualPath string) (Entry, bool) {
	result := index.Lookup(virtualPath)
	if result.Entry == nil {
		return Entry{}, false
	}
	return *result.Entry, true
}

func (index *Index) Entries() []Entry {
	if index == nil {
		return nil
	}
	entries := make([]Entry, len(index.ordered))
	for i, entry := range index.ordered {
		entries[i] = *cloneEntry(entry)
	}
	return entries
}

func (index *Index) Diagnostics() []Diagnostic {
	if index == nil {
		return nil
	}
	return append([]Diagnostic(nil), index.diagnostics...)
}

func (index *Index) ReadFile(virtualPath string) ([]byte, error) {
	entry, ok := index.Entry(virtualPath)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, virtualPath)
	}
	source := entry.Winner()
	if !source.Readable {
		return nil, fmt.Errorf("clientdata: %s from %s is unreadable: %s", virtualPath, source.Mount, source.ReadError)
	}
	data, err := os.ReadFile(source.PhysicalPath)
	if err != nil {
		return nil, fmt.Errorf("clientdata: read %s from %s: %w", virtualPath, source.Mount, err)
	}
	return data, nil
}

func cloneEntry(entry *Entry) *Entry {
	copyValue := *entry
	copyValue.Sources = append([]Source(nil), entry.Sources...)
	return &copyValue
}

func foldASCII(value string) string {
	buffer := []byte(value)
	for i, b := range buffer {
		if b >= 'A' && b <= 'Z' {
			buffer[i] = b + ('a' - 'A')
		}
	}
	return string(buffer)
}

func separatorDiagnosticKey(value string) string {
	buffer := []byte(foldASCII(value))
	for i, b := range buffer {
		if b == '/' {
			buffer[i] = '\\'
		}
	}
	return string(buffer)
}
