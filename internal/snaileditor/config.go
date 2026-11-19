package snaileditor

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// 模块配置（官方 ini\SnailEditorServerModule.xml）。

// ModuleXML 官方 XML 结构：
//
//	<root>
//	  <config SvnLocalRootPath=".." SvnServerIP=".." SvnServerPort=".."/>
//	  <watch><path ..>?（官方为 watch 元素列表，路径兼容属性/子元素两种形态）</watch>
//	</root>
type ModuleXML struct {
	Config struct {
		SvnLocalRootPath string `xml:"SvnLocalRootPath,attr"`
		SvnServerIP      string `xml:"SvnServerIP,attr"`
		SvnServerPort    int    `xml:"SvnServerPort,attr"`
	} `xml:"config"`
	Watch struct {
		Paths []struct {
			Value string `xml:"value,attr"`
		} `xml:"path"`
	} `xml:"watch"`
}

// LoadModuleConfig 读官方 ini\SnailEditorServerModule.xml。
func LoadModuleConfig(path string) (ModuleXML, error) {
	var cfg ModuleXML
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("load SnailEditorServerModule config file failed: %w", err)
	}
	if err := xml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse SnailEditorServerModule config failed: %w", err)
	}
	return cfg, nil
}

// WatchPaths watch 路径列表（规范化为小写 + 反斜杠）。
func (c ModuleXML) WatchPaths() []string {
	var out []string
	for _, p := range c.Watch.Paths {
		if p.Value != "" {
			out = append(out, foldLower(p.Value))
		}
	}
	return out
}

func foldLower(s string) string {
	s = strings.ReplaceAll(s, "/", `\`)
	return strings.ToLower(s)
}

// DefaultModuleConfigPath 官方相对路径（相对游戏工作目录）。
const DefaultModuleConfigPath = `ini\SnailEditorServerModule.xml`
