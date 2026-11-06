// Command clientdata-audit builds a machine-readable recursive resource
// closure from extracted Nine Yin package trees. It never opens or modifies the
// live .package files.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/local/9yin-go-server/internal/clientdata"
)

type cliOptions struct {
	dataRoot       string
	packagesINI    string
	entry          string
	output         string
	pretty         bool
	failIncomplete bool
	maxResources   int
}

func main() {
	options := cliOptions{}
	flag.StringVar(&options.dataRoot, "data-root", `E:\jiuyin`, "directory containing *.package.files extraction trees")
	flag.StringVar(&options.packagesINI, "packages", `D:\Program Files (x86)\游戏蜗牛\9yinjh\bin64\packages.ini`, "modern client packages.ini")
	flag.StringVar(&options.entry, "entry", `ini\npc\worldnpc288.ini`, "virtual composite INI path")
	flag.StringVar(&options.output, "out", "-", "JSON output file, or - for stdout")
	flag.BoolVar(&options.pretty, "pretty", true, "indent JSON output")
	flag.BoolVar(&options.failIncomplete, "fail-on-incomplete", false, "exit non-zero on missing/unreadable/separator mismatches")
	flag.IntVar(&options.maxResources, "max-resources", 10000, "cycle/runaway protection limit")
	flag.Parse()
	if err := run(options, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "clientdata-audit:", err)
		os.Exit(1)
	}
}

func run(options cliOptions, stdout io.Writer) error {
	packages, mounts, err := discoverPackages(options.packagesINI, options.dataRoot)
	if err != nil {
		return err
	}
	if len(mounts) == 0 {
		return fmt.Errorf("no extracted package roots from %s were found under %s", options.packagesINI, options.dataRoot)
	}
	index, err := clientdata.Build(clientdata.Config{Mounts: mounts})
	if err != nil {
		return err
	}
	report, err := clientdata.AuditResourceClosure(index, clientdata.ClosureOptions{
		Entry: options.entry, Packages: packages, MaxResources: options.maxResources,
		VirtualRoots: []string{"ini", "obj", "map", "share", "skin", "eff", "icon", "snd"},
	})
	if err != nil {
		return err
	}
	var output io.Writer = stdout
	var file *os.File
	if options.output != "" && options.output != "-" {
		if err := os.MkdirAll(filepath.Dir(options.output), 0o755); err != nil && filepath.Dir(options.output) != "." {
			return err
		}
		file, err = os.Create(options.output)
		if err != nil {
			return err
		}
		defer file.Close()
		output = file
	}
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if options.pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(report); err != nil {
		return err
	}
	if options.failIncomplete && (report.Summary.Unreadable != 0 || report.Summary.SeparatorMismatches != 0 || report.Summary.PackagesNotExtracted != 0 || report.Summary.Missing != 0) {
		return fmt.Errorf("resource closure is incomplete: unreadable=%d separator_mismatch=%d package_not_extracted=%d missing=%d",
			report.Summary.Unreadable, report.Summary.SeparatorMismatches, report.Summary.PackagesNotExtracted, report.Summary.Missing)
	}
	return nil
}

func discoverPackages(configPath, dataRoot string) ([]clientdata.KnownPackage, []clientdata.Mount, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open packages.ini: %w", err)
	}
	defer file.Close()
	type section struct{ name, file string }
	var sections []section
	current := -1
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(line[1 : len(line)-1])
			sections = append(sections, section{name: name})
			current = len(sections) - 1
			continue
		}
		if current < 0 {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 0 || !strings.EqualFold(strings.TrimSpace(line[:i]), "File") {
			continue
		}
		sections[current].file = strings.Trim(strings.TrimSpace(line[i+1:]), "\"'")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	if len(sections) == 0 {
		return nil, nil, errors.New("packages.ini has no sections")
	}
	packages := make([]clientdata.KnownPackage, 0, len(sections))
	var mounts []clientdata.Mount
	for order, section := range sections {
		if section.file == "" {
			continue
		}
		base := filepath.Base(strings.ReplaceAll(section.file, `\`, string(filepath.Separator)))
		extraction := filepath.Join(dataRoot, base+".files")
		root := filepath.Join(extraction, "res")
		if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
			root = extraction
		}
		extracted := false
		if info, statErr := os.Stat(root); statErr == nil && info.IsDir() {
			extracted = true
		}
		known := clientdata.KnownPackage{
			Name: section.name, File: section.file, Prefixes: inferPrefixes(section.name),
			LoadOrder: order, Extracted: extracted,
		}
		if extracted {
			known.ExtractRoot = root
			mounts = append(mounts, clientdata.Mount{Name: section.name, Directory: root})
		}
		packages = append(packages, known)
	}
	return packages, mounts, nil
}

func inferPrefixes(section string) []string {
	lower := strings.ToLower(strings.TrimSpace(section))
	for _, group := range []string{"obj", "map"} {
		if strings.HasPrefix(lower, group+"_") {
			return []string{group + `\` + strings.TrimPrefix(lower, group+"_")}
		}
	}
	return []string{lower}
}
