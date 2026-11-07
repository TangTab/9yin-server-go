package main

import (
	"bytes"
	"encoding/json"
	"sort"
	"time"
)

// facultyProgressSnapshot 是内功/进阶进度的持久化快照（现落
// role_faculty.snapshot）。字段集为 Java 参考服 PlayerProgress.Snapshot
// 的子集；Java/新版服写入的超集字段（vitals、生活槽位、基础五维等）
// 经 extras 原样 passthrough，Go 写回时不丢失。
// facultyBookSnapshot 的已知键；书内其余键（如新版服的 max_level）
// 同样经 extras passthrough。
type facultyBookSnapshot struct {
	ConfigID string `json:"config_id"`
	Level    int32  `json:"level"`
	Fill     int32  `json:"fill"`
	Total    int32  `json:"total"`
	Power    int32  `json:"power"`
	MaxPower int32  `json:"max_power"`

	extras map[string]json.RawMessage
}

var facultyBookJSONKeys = map[string]bool{
	"config_id": true,
	"level":     true,
	"fill":      true,
	"total":     true,
	"power":     true,
	"max_power": true,
}

func (book *facultyBookSnapshot) UnmarshalJSON(data []byte) error {
	type bookFields facultyBookSnapshot
	var fields bookFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*book = facultyBookSnapshot(fields)
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for key, value := range all {
		if facultyBookJSONKeys[key] {
			continue
		}
		if book.extras == nil {
			book.extras = make(map[string]json.RawMessage)
		}
		book.extras[key] = value
	}
	return nil
}

func (book facultyBookSnapshot) MarshalJSON() ([]byte, error) {
	type bookFields facultyBookSnapshot
	base, err := json.Marshal(bookFields(book))
	if err != nil {
		return nil, err
	}
	if len(book.extras) == 0 {
		return base, nil
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}
	for key, value := range book.extras {
		merged[key] = value
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	out.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			out.WriteByte(',')
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		out.Write(keyJSON)
		out.WriteByte(':')
		out.Write(merged[key])
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// facultyVitalsSnapshot 是 role_faculty 快照的 vitals 节点（GM
// set_vitals 生存属性层）。键名与 Java FacultyStore.renderSnapshot 逐字
// 一致（共享 MySQL，三服互读）；nil = 旧档未设置，不落节点。
type facultyVitalsSnapshot struct {
	Ene           int32 `json:"ene"`
	MaxEne        int32 `json:"max_ene"`
	ResumeEne     int32 `json:"resume_ene"`
	Sat           int32 `json:"sat"`
	MaxSat        int32 `json:"max_sat"`
	PKValue       int32 `json:"pk_value"`
	DrunkLevel    int32 `json:"drunk_level"`
	MaxHPOverride int32 `json:"max_hp_override"`
	MaxMPOverride int32 `json:"max_mp_override"`
}

type facultyProgressSnapshot struct {
	CurNeiGong     string                `json:"cur_neigong"`
	Faculty        int32                 `json:"faculty"`
	FacultyState   int32                 `json:"faculty_state"`
	FacultyStyle   int32                 `json:"faculty_style"`
	FacultyName    string                `json:"faculty_name"`
	FillSpeed      int32                 `json:"fill_speed"`
	LastAdvanceUTC time.Time             `json:"last_advance_utc"`
	Books          []facultyBookSnapshot `json:"books"`

	// —— 以下为 GM 面板域（货币/属性/状态批次）显式晋升的 Java 键。
	// 此前它们经 extras 原样透传；显式建模后 Go 侧可读写，而旧档
	// （Go/Java 互写）缺省零值还原语义见 playerProgress.restore。 ——

	// 修速拆分层（GM set_fill_speed）：旧档缺省 200×1。
	FillSpeedBase       int32   `json:"fill_speed_base"`
	FillSpeedMultiplier float32 `json:"fill_speed_multiplier"`
	// 演武每日碎银额度层（GM set_act_day_limit）：单位文。
	ActDayValue     int32 `json:"act_day_value"`
	ActCostSuiYin   int32 `json:"act_cost_suiyin"`
	ActDayValueMax  int32 `json:"act_day_value_max"`
	ActDayValueUsed int32 `json:"act_day_value_used"`
	ActDayValueLast int32 `json:"act_day_value_last"`
	// 基础五维（GM set_five_attributes）：旧档缺省 0 → 还原不覆盖
	// 运行时值（Java FacultyStore 注释同口径）。
	BaseStr int32 `json:"base_str"`
	BaseSta int32 `json:"base_sta"`
	BaseDex int32 `json:"base_dex"`
	BaseIng int32 `json:"base_ing"`
	BaseSpi int32 `json:"base_spi"`
	// 生存属性层（GM set_vitals）：nil = 旧档未设置。
	Vitals *facultyVitalsSnapshot `json:"vitals,omitempty"`

	// extras 保存解码时遇到的本结构未知字段（非导出，encoding/json
	// 不直接编解码），编码时按 key 排序原样回写。
	extras map[string]json.RawMessage
}

// facultySnapshotJSONKeys 是本结构已显式建模的 JSON 键；其余键进 extras。
var facultySnapshotJSONKeys = map[string]bool{
	"cur_neigong":           true,
	"faculty":               true,
	"faculty_state":         true,
	"faculty_style":         true,
	"faculty_name":          true,
	"fill_speed":            true,
	"last_advance_utc":      true,
	"books":                 true,
	"fill_speed_base":       true,
	"fill_speed_multiplier": true,
	"act_day_value":         true,
	"act_cost_suiyin":       true,
	"act_day_value_max":     true,
	"act_day_value_used":    true,
	"act_day_value_last":    true,
	"base_str":              true,
	"base_sta":              true,
	"base_dex":              true,
	"base_ing":              true,
	"base_spi":              true,
	"vitals":                true,
}

func (snapshot *facultyProgressSnapshot) UnmarshalJSON(data []byte) error {
	type snapshotFields facultyProgressSnapshot
	var fields snapshotFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*snapshot = facultyProgressSnapshot(fields)
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for key, value := range all {
		if facultySnapshotJSONKeys[key] {
			continue
		}
		if snapshot.extras == nil {
			snapshot.extras = make(map[string]json.RawMessage)
		}
		snapshot.extras[key] = value
	}
	return nil
}

func (snapshot facultyProgressSnapshot) MarshalJSON() ([]byte, error) {
	type snapshotFields facultyProgressSnapshot
	base, err := json.Marshal(snapshotFields(snapshot))
	if err != nil {
		return nil, err
	}
	if len(snapshot.extras) == 0 {
		return base, nil
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}
	for key, value := range snapshot.extras {
		merged[key] = value
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out bytes.Buffer
	out.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			out.WriteByte(',')
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		out.Write(keyJSON)
		out.WriteByte(':')
		out.Write(merged[key])
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

func cloneFacultyProgress(value facultyProgressSnapshot) facultyProgressSnapshot {
	value.Books = append([]facultyBookSnapshot(nil), value.Books...)
	for index, book := range value.Books {
		if source := book.extras; source != nil {
			book.extras = make(map[string]json.RawMessage, len(source))
			for key, raw := range source {
				book.extras[key] = raw
			}
		}
		value.Books[index] = book
	}
	if source := value.extras; source != nil {
		value.extras = make(map[string]json.RawMessage, len(source))
		for key, raw := range source {
			value.extras[key] = raw
		}
	}
	return value
}
