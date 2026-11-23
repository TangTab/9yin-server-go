package fsgame

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// resource.go 静态资源面（官方 CXmlFile/PreloadConfig 的 Go 等价物）。
// 官方 Res/ini 的 XML 有两种方言：
//   1) Property 属性表：<Object><Property ID=".." k="v" .../></Object>
//      （door_data.xml / modules.xml / DumpConfig 族）
//   2) schema 表：<object><properties>…</properties><records>…</records></object>
//      （struct/*.xml，见 schema.go）

// readResource 资源读取（root + 相对路径；官方 GetResourcePath + 相对 ini 路径）。
//
// ini/Skill 下的黑金原档已让位给九阴构建产物（tools/skillgen 首跑整体移入
// ini/Skill/heijin_backup/）：规范位不存在时回退备份目录取同名文件。本包是
// FsGameLogic 的复刻模型，技能/buff 资源夹具仍是黑金真档（九阴产物是
// internal/skilldata 的数据源，列结构与黑金不同，不可混读）。
func readResource(root, rel string) ([]byte, error) {
	if root == "" {
		return nil, fmt.Errorf("fsgame: 资源根路径为空")
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		return data, err
	}
	if strings.HasPrefix(strings.ToLower(filepath.ToSlash(rel)), "ini/skill/") {
		if backup, backupErr := os.ReadFile(filepath.Join(root, "ini", "Skill", "heijin_backup", filepath.Base(rel))); backupErr == nil {
			return backup, nil
		}
	}
	return nil, err
}

// ParsePropTable 解析 <Object><Property k="v" .../></Object> 为属性行列表。
// encoding/xml 不能把元素属性直接解进 map，走 Token 流收集 StartElement
// 的全部属性。官方 xml 多为 gb2312 声明（GB18030 超集解码）。
func ParsePropTable(data []byte) ([]map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = gbCharsetReader
	var rows []map[string]string
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fsgame: prop table: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Property" {
			continue
		}
		row := make(map[string]string, len(se.Attr))
		for _, a := range se.Attr {
			row[a.Name.Local] = a.Value
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// gbCharsetReader 官方 gb2312/gbk 声明统一按 GB18030 超集解码
// （与 internal/config objectfile 同一决策）。
func gbCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "gb2312", "gbk", "gb18030":
		return simplifiedchinese.GB18030.NewDecoder().Reader(input), nil
	}
	return nil, fmt.Errorf("fsgame: 不支持的字符集 %q", charset)
}

// ParseDoorDataXML 解析 ini/Rule/door_data.xml（传送门表）。
func ParseDoorDataXML(data []byte) ([]LocationUnit, error) {
	rows, err := ParsePropTable(data)
	if err != nil {
		return nil, err
	}
	out := make([]LocationUnit, 0, len(rows))
	for _, r := range rows {
		u := LocationUnit{
			Name: r["Name"],
		}
		u.ID = atoi32(r["ID"])
		u.Scene = atoi32(r["TargetSceneID"])
		u.X, u.Y, u.Z = atof32(r["TargetX"]), atof32(r["TargetY"]), atof32(r["TargetZ"])
		u.Orient = atof32(r["TargetOrient"])
		u.MoveX, u.MoveY, u.MoveZ = atof32(r["MoveTargetX"]), atof32(r["MoveTargetY"]), atof32(r["MoveTargetZ"])
		if u.MoveX == 0 && u.MoveY == 0 && u.MoveZ == 0 {
			u.MoveX, u.MoveY, u.MoveZ = u.X, u.Y, u.Z
		}
		out = append(out, u)
	}
	return out, nil
}

func atoi32(s string) int32 {
	v, _ := strconv.ParseInt(s, 10, 32)
	return int32(v)
}

func atof32(s string) float32 {
	v, _ := strconv.ParseFloat(s, 32)
	return float32(v)
}
