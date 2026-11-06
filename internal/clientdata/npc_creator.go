package clientdata

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type RawValue struct {
	Raw     string
	Present bool
}

type NPCTransform struct {
	X, Y, Z    float32
	AX, AY, AZ *float32
	SX, SY, SZ *float32
}

type NPCCreatorInstance struct {
	Source       string
	Ordinal      int
	PhysicalLine int
	No           string
	TemplateID   string
	Transform    NPCTransform
	Attributes   map[string]RawValue
}

func (i NPCCreatorInstance) InstanceKey() string {
	return fmt.Sprintf("%s#%d", i.Source, i.Ordinal)
}

type NPCCreatorCatalog struct {
	Source    string
	Instances []NPCCreatorInstance
	byID      map[string][]int
}

func (c *NPCCreatorCatalog) InstancesForTemplate(configID string) []NPCCreatorInstance {
	indices := c.byID[configID]
	result := make([]NPCCreatorInstance, 0, len(indices))
	for _, index := range indices {
		result = append(result, c.Instances[index])
	}
	return result
}

func LoadNPCCreator(r io.Reader, source string) (*NPCCreatorCatalog, error) {
	if r == nil {
		return nil, errors.New("clientdata: nil NPC creator reader")
	}
	if source == "" {
		source = "<reader>"
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("clientdata: read NPC creator %s: %w", source, err)
	}
	if !utf8.Valid(data) {
		data, _, err = transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data)
		if err != nil {
			return nil, fmt.Errorf("clientdata: decode GBK NPC creator %s: %w", source, err)
		}
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	catalog := &NPCCreatorCatalog{Source: source, byID: make(map[string][]int)}
	var randomCreator map[string]RawValue
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("clientdata: parse NPC creator %s: %w", source, err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "creator":
				randomCreator = rawAttributeMap(element.Attr)
			case "item":
				line, _ := decoder.InputPos()
				attributes := rawAttributeMap(element.Attr)
				if _, hasX := attributes["x"]; hasX {
					if isEmptyNPCCreatorPlaceholder(attributes) {
						continue
					}
					if err := appendCreatorInstance(catalog, source, line, attributes); err != nil {
						return nil, err
					}
					continue
				}
				if randomCreator == nil {
					ordinal := len(catalog.Instances) + 1
					return nil, creatorFieldError(source, ordinal, line, errors.New("missing x outside random creator"))
				}
				positions, readErr := readRandomItemPositions(decoder)
				if readErr != nil {
					return nil, fmt.Errorf("clientdata: parse random NPC creator %s line %d: %w", source, line, readErr)
				}
				if isEmptyNPCCreatorPlaceholder(attributes) {
					continue
				}
				instances := expandRandomCreatorItem(randomCreator, attributes, positions)
				for _, expanded := range instances {
					if err := appendCreatorInstance(catalog, source, line, expanded); err != nil {
						return nil, err
					}
				}
			}
		case xml.EndElement:
			if element.Name.Local == "creator" {
				randomCreator = nil
			}
		}
	}
	return catalog, nil
}

// isEmptyNPCCreatorPlaceholder 识别的是导出器占位符，而不是
// 数据不完整的 NPC。某些新版 creator 文件会带一个额外的 item，
// 其模板 id 为空且 amount=0，仅用于存放未使用的随机刷新点。
// 它不应导致整个场景目录解析中止。而任何非零/非法 item 上
// 缺失 id 仍然属于数据错误。
func isEmptyNPCCreatorPlaceholder(attributes map[string]RawValue) bool {
	if strings.TrimSpace(attributeText(attributes, "id")) != "" {
		return false
	}
	raw, present := attributes["amount"]
	if !present || !raw.Present {
		return false
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(raw.Raw), 64)
	return err == nil && amount == 0
}

func rawAttributeMap(attributes []xml.Attr) map[string]RawValue {
	result := make(map[string]RawValue, len(attributes))
	for _, attribute := range attributes {
		result[attribute.Name.Local] = RawValue{Raw: attribute.Value, Present: true}
	}
	return result
}

func appendCreatorInstance(catalog *NPCCreatorCatalog, source string, line int, attributes map[string]RawValue) error {
	ordinal := len(catalog.Instances) + 1
	instance := NPCCreatorInstance{
		Source:       source,
		Ordinal:      ordinal,
		PhysicalLine: line,
		No:           attributeText(attributes, "No"),
		TemplateID:   attributeText(attributes, "id"),
		Attributes:   attributes,
	}
	if instance.TemplateID == "" {
		return fmt.Errorf("clientdata: NPC creator %s item %d line %d has no id", source, ordinal, line)
	}
	var err error
	if instance.Transform.X, err = requiredCreatorFloat(attributes, "x"); err != nil {
		return creatorFieldError(source, ordinal, line, err)
	}
	if instance.Transform.Y, err = requiredCreatorFloat(attributes, "y"); err != nil {
		return creatorFieldError(source, ordinal, line, err)
	}
	if instance.Transform.Z, err = requiredCreatorFloat(attributes, "z"); err != nil {
		return creatorFieldError(source, ordinal, line, err)
	}
	optionalAxes := []struct {
		name        string
		destination **float32
	}{
		{"ax", &instance.Transform.AX},
		{"ay", &instance.Transform.AY},
		{"az", &instance.Transform.AZ},
		{"sx", &instance.Transform.SX},
		{"sy", &instance.Transform.SY},
		{"sz", &instance.Transform.SZ},
	}
	for _, axis := range optionalAxes {
		value, present, parseErr := optionalCreatorFloat(attributes, axis.name)
		if parseErr != nil {
			return creatorFieldError(source, ordinal, line, parseErr)
		}
		if present {
			copyValue := value
			*axis.destination = &copyValue
		}
	}
	index := len(catalog.Instances)
	catalog.Instances = append(catalog.Instances, instance)
	catalog.byID[instance.TemplateID] = append(catalog.byID[instance.TemplateID], index)
	return nil
}

func readRandomItemPositions(decoder *xml.Decoder) ([]map[string]RawValue, error) {
	var positions []map[string]RawValue
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			depth++
			if element.Name.Local == "position" {
				positions = append(positions, rawAttributeMap(element.Attr))
			}
		case xml.EndElement:
			depth--
		}
	}
	return positions, nil
}

func expandRandomCreatorItem(creator, item map[string]RawValue, positions []map[string]RawValue) []map[string]RawValue {
	base := mergeRawAttributes(creator, item)
	creatorNo, itemNo := attributeText(creator, "No"), attributeText(item, "No")
	if len(positions) == 0 {
		amount := 1
		if raw := attributeText(item, "amount"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 128 {
				amount = parsed
			}
		}
		positions = make([]map[string]RawValue, amount)
		radius, _, _ := optionalCreatorFloat(creator, "radius")
		centerX, _, _ := optionalCreatorFloat(creator, "x")
		centerZ, _, _ := optionalCreatorFloat(creator, "z")
		for i := 0; i < amount; i++ {
			position := map[string]RawValue{}
			if amount > 1 && radius > 0 {
				angle := 2 * math.Pi * float64(i) / float64(amount)
				position["x"] = RawValue{Raw: strconv.FormatFloat(float64(centerX)+float64(radius)*0.65*math.Cos(angle), 'f', 3, 32), Present: true}
				position["z"] = RawValue{Raw: strconv.FormatFloat(float64(centerZ)+float64(radius)*0.65*math.Sin(angle), 'f', 3, 32), Present: true}
			}
			positions[i] = position
		}
	}
	result := make([]map[string]RawValue, 0, len(positions))
	for i, position := range positions {
		expanded := mergeRawAttributes(base, position)
		expanded["No"] = RawValue{Raw: fmt.Sprintf("%s:%s:%d", creatorNo, itemNo, i+1), Present: true}
		result = append(result, expanded)
	}
	return result
}

func mergeRawAttributes(layers ...map[string]RawValue) map[string]RawValue {
	result := map[string]RawValue{}
	for _, layer := range layers {
		for name, value := range layer {
			result[name] = value
		}
	}
	return result
}

func creatorFieldError(source string, ordinal, line int, err error) error {
	return fmt.Errorf("clientdata: NPC creator %s item %d line %d: %w", source, ordinal, line, err)
}

func attributeText(attributes map[string]RawValue, name string) string {
	return attributes[name].Raw
}

func requiredCreatorFloat(attributes map[string]RawValue, name string) (float32, error) {
	value, present, err := optionalCreatorFloat(attributes, name)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, fmt.Errorf("missing %s", name)
	}
	return value, nil
}

func optionalCreatorFloat(attributes map[string]RawValue, name string) (float32, bool, error) {
	raw, present := attributes[name]
	if !present || !raw.Present {
		return 0, false, nil
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw.Raw), 32)
	if err != nil {
		return 0, true, fmt.Errorf("%s=%q is not float32: %w", name, raw.Raw, err)
	}
	value := float32(parsed)
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 0, true, fmt.Errorf("%s=%q is not finite", name, raw.Raw)
	}
	return value, true, nil
}
