package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 镜像 Java SceneNamesTest：表驱动显示名映射 + 档位/剧本键映射 +
// 0x04 帧 f23/f24/f25 端到端断言。夹具自建 GBK scenes.ini 与 UTF-8
// stringname.idres，不依赖仓库 share 资源是否存在。
const sceneNamesFixtureINI = "// 场景配置文件（GBK 注释验证解码）\n" +
	"[1]\nMember=1\nScript=scene\nConfig=ini\\scene\\born04_QianDengZhen\n" +
	"[5]\nMember=1\nScript=scene\nConfig=ini\\scene\\city05_ChengDu\n" +
	"[17]\nMember=1\nScript=scene\nConfig=ini\\scene\\school09_YiHuaGong\n"

const sceneNamesFixtureIDRes = "city05=成都\n" +
	"ini\\scene\\school09_yihuagong=移花宫\n" +
	"ini\\scene\\school09_yihuagong=重复键应取首值\n"

func injectSceneNameTables(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	encodedINI, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(sceneNamesFixtureINI))
	if err != nil {
		t.Fatalf("encode GBK fixture: %v", err)
	}
	oldINI, oldIDRes := modernScenesRulePath, modernStringNamePath
	modernScenesRulePath = filepath.Join(dir, "scenes.ini")
	modernStringNamePath = filepath.Join(dir, "stringname.idres")
	if err := os.WriteFile(modernScenesRulePath, encodedINI, 0o644); err != nil {
		t.Fatalf("write scenes.ini fixture: %v", err)
	}
	if err := os.WriteFile(modernStringNamePath, []byte(sceneNamesFixtureIDRes), 0o644); err != nil {
		t.Fatalf("write stringname.idres fixture: %v", err)
	}
	resetSceneNameTablesForTest()
	t.Cleanup(func() {
		modernScenesRulePath, modernStringNamePath = oldINI, oldIDRes
		resetSceneNameTablesForTest()
	})
}

func resetSceneNameTablesForTest() {
	sceneNameTablesOnce = sync.Once{}
	sceneNameTablesCache = nil
}

func TestSceneDisplayNameResolvesFromTables(t *testing.T) {
	injectSceneNameTables(t)
	if got := sceneDisplayName(`ini\scene\city05_ChengDu`, "city05"); got != "成都" {
		t.Fatalf("config+resource scene name=%q, want 成都", got)
	}
	if got := sceneDisplayName(`ini\scene\school09_YiHuaGong`, "school09"); got != "移花宫" {
		t.Fatalf("school09 scene name=%q, want 移花宫", got)
	}
	// 仅 resource（config 缺失）也能命中。
	if got := sceneDisplayName("", "city05"); got != "成都" {
		t.Fatalf("resource-only scene name=%q, want 成都", got)
	}
	// stringname 未覆盖的场景回退 scenes.ini 场景号键。
	if got := sceneDisplayName(`ini\scene\born04_QianDengZhen`, "born04"); got != "scene_name_001" {
		t.Fatalf("id fallback=%q, want scene_name_001", got)
	}
}

func TestUnknownSceneFallsBackToKey(t *testing.T) {
	injectSceneNameTables(t)
	if got := sceneDisplayName("", ""); got != "scene_name_017" {
		t.Fatalf("unknown scene=%q, want scene_name_017", got)
	}
	if got := sceneDisplayName("ini\\scene\\missing_scene", "missing"); got != "scene_name_017" {
		t.Fatalf("missing scene=%q, want scene_name_017", got)
	}
}

func TestBookIntroKeys(t *testing.T) {
	cases := map[string]string{
		"book1": "introduction_qdz01_title",
		"book5": "introduction_jm01_title",
		"book8": "introduction_wmp01_title",
		"bookX": "bookX",
	}
	for book, want := range cases {
		if got := bookIntroKey(book); got != want {
			t.Fatalf("bookIntroKey(%q)=%q, want %q", book, got, want)
		}
	}
}

func TestRoleListTitleTiers(t *testing.T) {
	cases := map[int]string{
		1:   "title001",
		5:   "title001",
		6:   "title002",
		120: "title024",
		130: "title025",
		150: "title026",
		151: "title027",
	}
	for level, want := range cases {
		if got := roleListTitle(level); got != want {
			t.Fatalf("roleListTitle(%d)=%q, want %q", level, got, want)
		}
	}
}

// roleListFrameStrings 解码 0x04 帧里的两条 type-7 UCS-2 串（名字、rolePara）。
func roleListFrameStrings(t *testing.T, msg []byte) (string, string) {
	t.Helper()
	offset := 0x29
	var texts []string
	for offset+5 <= len(msg) {
		if msg[offset] != 7 {
			offset++
			continue
		}
		size := int(binary.LittleEndian.Uint32(msg[offset+1:]))
		if size < 2 || size%2 != 0 || offset+5+size > len(msg) {
			offset++
			continue
		}
		units := make([]uint16, 0, size/2)
		for i := 0; i+1 < size; i += 2 {
			unit := binary.LittleEndian.Uint16(msg[offset+5+i:])
			if unit == 0 {
				break
			}
			units = append(units, unit)
		}
		texts = append(texts, string(utf16.Decode(units)))
		offset += 5 + size
	}
	if len(texts) < 2 {
		t.Fatalf("role frame carries %d type-7 strings, want 2", len(texts))
	}
	return texts[0], texts[1]
}

func TestOneRoleLoginSuccessSelectionFields(t *testing.T) {
	injectSceneNameTables(t)
	msg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil,
		`ini\scene\school09_YiHuaGong`, "school09")
	name, rolePara := roleListFrameStrings(t, msg)
	if name != "本地角色" {
		t.Fatalf("role name=%q", name)
	}
	fields := strings.Split(rolePara, ",")
	if len(fields) < 36 {
		t.Fatalf("rolePara fields=%d, want >=36", len(fields))
	}
	if got := fields[23]; got != "title027" {
		t.Fatalf("f23 实力称号=%q, want title027（一期种子 151=返璞归真）", got)
	}
	if got := fields[24]; got != "移花宫" {
		t.Fatalf("f24 所在场景=%q, want 移花宫", got)
	}
	if got := fields[25]; got != "introduction_qdz01_title" {
		t.Fatalf("f25 剧本标题=%q, want introduction_qdz01_title", got)
	}
	// f32 仍是 book 原值（客户端用作选人背景图 id）。
	if got := fields[32]; got != "book1" {
		t.Fatalf("f32 bookid=%q, want book1", got)
	}
}

// 表缺失（开发仓库无 share 资源）时整帧仍可构造，f23/f24 回退原键。
func TestOneRoleLoginSuccessWithoutTables(t *testing.T) {
	injectSceneNameTables(t)
	os.Remove(modernScenesRulePath)
	os.Remove(modernStringNamePath)
	resetSceneNameTablesForTest()
	msg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil,
		`ini\scene\school09_YiHuaGong`, "school09")
	_, rolePara := roleListFrameStrings(t, msg)
	fields := strings.Split(rolePara, ",")
	if got := fields[24]; got != "scene_name_017" {
		t.Fatalf("f24 fallback=%q, want scene_name_017", got)
	}
	if !bytes.Contains(msg, []byte{0}) {
		t.Fatal("frame unexpectedly empty of NUL units")
	}
}
