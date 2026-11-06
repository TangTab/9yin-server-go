package clientdata

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	ErrNPCTemplateNotFound  = errors.New("clientdata: NPC template not found")
	ErrNPCTemplateAmbiguous = errors.New("clientdata: NPC template ID is ambiguous")
)

type RawCell struct {
	ColumnIndex int
	ColumnName  string
	Raw         string
	Set         bool
}

type NPCTemplateRow struct {
	Source        string
	LogicalRecord int
	PhysicalLine  int
	ConfigID      string
	ScriptClass   string
	Cells         []RawCell
	columnsByName map[string][]int
}

func (r NPCTemplateRow) CellsNamed(name string) []RawCell {
	indices := r.columnsByName[name]
	result := make([]RawCell, 0, len(indices))
	for _, index := range indices {
		result = append(result, r.Cells[index])
	}
	return result
}

func (r NPCTemplateRow) CellAt(index int) (RawCell, bool) {
	if index < 0 || index >= len(r.Cells) {
		return RawCell{}, false
	}
	return r.Cells[index], true
}

type NPCTemplateTable struct {
	Source        string
	Comments      []string
	DeclaredTypes []string
	Columns       []string
	Rows          []NPCTemplateRow
	byID          map[string][]int
	columnsByName map[string][]int
}

func (t *NPCTemplateTable) ColumnIndices(name string) []int {
	return append([]int(nil), t.columnsByName[name]...)
}

func (t *NPCTemplateTable) LookupAll(configID string) []NPCTemplateRow {
	indices := t.byID[configID]
	result := make([]NPCTemplateRow, 0, len(indices))
	for _, index := range indices {
		result = append(result, t.Rows[index])
	}
	return result
}

func (t *NPCTemplateTable) Lookup(configID string) (NPCTemplateRow, error) {
	indices := t.byID[configID]
	switch len(indices) {
	case 0:
		return NPCTemplateRow{}, fmt.Errorf("%w: %q in %s", ErrNPCTemplateNotFound, configID, t.Source)
	case 1:
		return t.Rows[indices[0]], nil
	default:
		records := make([]string, 0, len(indices))
		for _, index := range indices {
			records = append(records, fmt.Sprintf("%d", t.Rows[index].LogicalRecord))
		}
		return NPCTemplateRow{}, fmt.Errorf("%w: %q in %s at logical records %s", ErrNPCTemplateAmbiguous, configID, t.Source, strings.Join(records, ","))
	}
}

// LoadNPCTemplateTable reads the client's GBK-encoded, quoted TSV format.
// encoding/csv is deliberate: some fields contain quoted embedded newlines.
func LoadNPCTemplateTable(r io.Reader, source string) (*NPCTemplateTable, error) {
	if r == nil {
		return nil, errors.New("clientdata: nil NPC template reader")
	}
	if source == "" {
		source = "<reader>"
	}
	reader := csv.NewReader(transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false

	readHeader := func(recordNumber int) ([]string, int, error) {
		record, err := reader.Read()
		if err != nil {
			return nil, 0, fmt.Errorf("clientdata: read NPC template %s logical record %d: %w", source, recordNumber, err)
		}
		line, _ := reader.FieldPos(0)
		return record, line, nil
	}
	comments, _, err := readHeader(1)
	if err != nil {
		return nil, err
	}
	declared, _, err := readHeader(2)
	if err != nil {
		return nil, err
	}
	columns, _, err := readHeader(3)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("clientdata: NPC template %s has no columns", source)
	}
	if len(comments) != len(columns) || len(declared) != len(columns) {
		return nil, fmt.Errorf("clientdata: NPC template %s header widths comments=%d types=%d columns=%d", source, len(comments), len(declared), len(columns))
	}
	reader.FieldsPerRecord = len(columns)

	table := &NPCTemplateTable{
		Source:        source,
		Comments:      append([]string(nil), comments...),
		DeclaredTypes: append([]string(nil), declared...),
		Columns:       append([]string(nil), columns...),
		byID:          make(map[string][]int),
		columnsByName: make(map[string][]int),
	}
	for index, name := range columns {
		table.columnsByName[name] = append(table.columnsByName[name], index)
	}
	idColumns := table.columnsByName["ID"]
	if len(idColumns) != 1 {
		return nil, fmt.Errorf("clientdata: NPC template %s has %d ID columns, want 1", source, len(idColumns))
	}
	scriptColumns := table.columnsByName["script"]

	logicalRecord := 3
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("clientdata: read NPC template %s after logical record %d: %w", source, logicalRecord, readErr)
		}
		allEmpty := true
		for _, raw := range record {
			if raw != "" {
				allEmpty = false
				break
			}
		}
		// The shipped table contains separator rows made entirely of empty TSV
		// cells. They are layout, not logical data records.
		if allEmpty {
			continue
		}
		logicalRecord++
		line, _ := reader.FieldPos(0)
		row := NPCTemplateRow{
			Source:        source,
			LogicalRecord: logicalRecord,
			PhysicalLine:  line,
			Cells:         make([]RawCell, len(record)),
			columnsByName: table.columnsByName,
		}
		for index, raw := range record {
			row.Cells[index] = RawCell{ColumnIndex: index, ColumnName: columns[index], Raw: raw, Set: raw != ""}
		}
		row.ConfigID = record[idColumns[0]]
		if len(scriptColumns) == 1 {
			row.ScriptClass = record[scriptColumns[0]]
		}
		rowIndex := len(table.Rows)
		table.Rows = append(table.Rows, row)
		if row.ConfigID != "" {
			table.byID[row.ConfigID] = append(table.byID[row.ConfigID], rowIndex)
		}
	}
	return table, nil
}
