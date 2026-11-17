// dummy-migrate —— 人群假人数据脚本：审计 + 生成权威存档迁移。
//
// 用法：
//
//	# 数据审计（不入库、不写文件）：总量/场景分布/特性计数/卡牌覆盖率/重名
//	go run ./cmd/dummy-migrate -stats
//
//	# 生成迁移存档（幂等可重放；升级序号与日期按实际编排）
//	go run ./cmd/dummy-migrate -out migrations/0021_dummy_players_20261117.sql.gz
//
// 数据流（单向，server1 运行时不读任何 JSON）：
//
//	server2 data/dummy_players.json + card_models.json
//	  → 本工具 → migrations/NNNN_*.sql.gz（仓库权威存档）
//	  → go run ./cmd/sql-replay（或 upgrade-db.bat）重放进 nineyin 库
//	  → cmd/protocol-probe 运行时只读 MySQL（server_dummy_players/game_cards）
//
// 产物内容（幂等，可重放）：
//  1. CREATE TABLE IF NOT EXISTS server_dummy_players —— Java 参考服
//     0002_data_tables.sql 的 11 列 + 7 个扩展列（title_id/nei_gong/card/
//     patrol/patrol_random/move_mode/yufeng；Java 导入器按列名 INSERT 不受
//     影响）；
//  2. 对已存在的旧表逐列 information_schema 条件 ALTER（MySQL 8 无
//     ADD COLUMN IF NOT EXISTS，migrations/README 惯例）；
//  3. REPLACE INTO 全量行（dummy_id 显式取 JSON 序号 1..N，重放确定性
//     覆盖；position/appearance/patrol 三 JSON 列按源字段原样序列化）；
//  4. game_cards 建表 + REPLACE INTO（防本地库未跑过 Java 导入）。
//
// 确定性契约：同一输入文件生成的 SQL 字节完全一致（迁移 runner 的校验和
// = 内部 SQL 的 SHA-256，重新生成不得漂移——改输出格式即改校验和，已应用
// 的迁移会拒绝重放）。结构重构时用重新生成的包内 SQL 与存档逐字节比对
// 验证。
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---- 源数据结构（server2 JSON 形状；与运行时 dummyPlayer 各自独立，字段
// 演进互不牵连）----

type dummyPosition struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Orient float32 `json:"orient"`
}

type dummyAppearanceCfg struct {
	Photo        string `json:"photo"`
	Face         string `json:"face"`
	Hair         string `json:"hair"`
	Hat          string `json:"hat"`
	Cloth        string `json:"cloth"`
	Pants        string `json:"pants"`
	Shoes        string `json:"shoes"`
	Weapon       string `json:"weapon"`
	WeaponName   string `json:"weaponName"`
	WeaponMode   string `json:"weaponMode"`
	ActionSet    string `json:"actionSet"`
	WeaponConfig string `json:"weaponConfig"`
}

type dummyPlayer struct {
	Name         string             `json:"name"`
	Sex          byte               `json:"sex"`
	School       string             `json:"school"`
	Level        int                `json:"level"`
	TitleID      int                `json:"titleId"`
	PowerLevel   int                `json:"powerLevel"`
	NeiGong      string             `json:"neiGong"`
	Character    int                `json:"character"`
	Card         int32              `json:"card"`
	Patrol       []dummyPosition    `json:"patrol"`
	PatrolRandom bool               `json:"patrolRandom"`
	MoveMode     string             `json:"moveMode"`
	Yufeng       bool               `json:"yufeng"`
	NeiGongBuff  string             `json:"neiGongBuff"`
	Scene        string             `json:"scene"`
	Action       string             `json:"action"`
	Pos          dummyPosition      `json:"position"`
	Look         dummyAppearanceCfg `json:"appearance"`
}

type dummyPlayerConfig struct {
	Dummies []dummyPlayer `json:"dummies"`
}

type cardCatalogEntry struct {
	Name            string `json:"name"`
	ConfigID        int32  `json:"config_id"`
	ItemType        int32  `json:"item_type"`
	Level           int32  `json:"level"`
	MainType        int32  `json:"main_type"`
	SubType         int32  `json:"sub_type"`
	Flag            int32  `json:"flag"`
	FemaleModelData string `json:"female_model_data"`
	MaleModelData   string `json:"male_model_data"`
}

type cardCatalogFile struct {
	Cards map[string]cardCatalogEntry `json:"cards"`
}

func main() {
	source := flag.String("source", `E:\Workspace\project\game\9yin-go-server2\data\dummy_players.json`, "server2 的 dummy_players.json 源路径")
	cards := flag.String("cards", `E:\Workspace\project\game\9yin-go-server2\data\card_models.json`, "server2 的 card_models.json 源路径")
	out := flag.String("out", "", "输出 .sql.gz 路径（如 migrations/0021_dummy_players_20261117.sql.gz）")
	stats := flag.Bool("stats", false, "只审计源数据（总量/场景分布/特性计数/卡牌覆盖/重名），不生成文件")
	flag.Parse()

	dummies := loadDummies(*source)
	cardIDs, cardRows := loadCards(*cards)
	if *stats {
		printStats(dummies, cardIDs)
		return
	}
	if *out == "" {
		log.Fatal("必须指定 -out（migrations/NNNN_dummy_players_YYYYMMDD.sql.gz），或用 -stats 只审计数据")
	}

	sql := buildSQL(dummies, cardIDs, cardRows)
	if err := writeSQLArchive(*out, sql); err != nil {
		log.Fatalf("写 %s: %v", *out, err)
	}
	log.Printf("已生成 %s：dummy=%d 条 卡牌=%d 张 SQL=%.1f MB",
		*out, len(dummies), len(cardRows), float64(len(sql))/1024/1024)
}

// ---- 源装载 ----

func loadDummies(path string) []dummyPlayer {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读 %s: %v", path, err)
	}
	var config dummyPlayerConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		log.Fatalf("解析 %s: %v", path, err)
	}
	if len(config.Dummies) == 0 {
		log.Fatalf("%s 无 dummies 条目", path)
	}
	return config.Dummies
}

func loadCards(path string) ([]int32, map[int32]cardCatalogEntry) {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读 %s: %v", path, err)
	}
	var catalog cardCatalogFile
	if err := json.Unmarshal(raw, &catalog); err != nil {
		log.Fatalf("解析 %s: %v", path, err)
	}
	byID := make(map[int32]cardCatalogEntry, len(catalog.Cards))
	ids := make([]int32, 0, len(catalog.Cards))
	for key, entry := range catalog.Cards {
		var id int32
		if _, err := fmt.Sscanf(key, "%d", &id); err != nil || id <= 0 {
			continue
		}
		byID[id] = entry
		ids = append(ids, id)
	}
	// card_id 升序保证产物确定性。
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, byID
}

// ---- 数据审计（-stats）----

func printStats(dummies []dummyPlayer, cardIDs []int32) {
	scenes := map[string]int{}
	characters := map[int]int{}
	feature := map[string]int{}
	names := map[string]int{}
	cardUse := map[int32]int{}
	for _, d := range dummies {
		scenes[d.Scene]++
		characters[d.Character]++
		names[d.Name]++
		if d.Card > 0 {
			feature["card"]++
			cardUse[d.Card]++
		}
		if len(d.Patrol) >= 2 {
			feature["patrol(固定路点)"]++
		}
		if d.PatrolRandom {
			feature["patrolRandom(随机人流)"]++
		}
		if d.MoveMode == "walk" {
			feature["walk(步行)"]++
		}
		if d.Yufeng {
			feature["yufeng(御风)"]++
		}
		if d.NeiGongBuff != "" {
			feature["neiGongBuff(内功buff)"]++
		}
		if d.Look.WeaponConfig != "" || d.Look.Weapon != "" || d.Look.WeaponName != "" {
			feature["weapon(带武器)"]++
		}
	}
	log.Printf("总量：%d 条；卡牌目录：%d 张", len(dummies), len(cardIDs))

	keys := make([]string, 0, len(scenes))
	for key := range scenes {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if scenes[keys[i]] != scenes[keys[j]] {
			return scenes[keys[i]] > scenes[keys[j]]
		}
		return keys[i] < keys[j]
	})
	var scenesOut strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&scenesOut, "\n    %-16s %5d", sceneLabel(key), scenes[key])
	}
	log.Printf("场景分布（%d 个）：%s", len(scenes), scenesOut.String())

	var featureOut strings.Builder
	featureKeys := make([]string, 0, len(feature))
	for key := range feature {
		featureKeys = append(featureKeys, key)
	}
	sort.Strings(featureKeys)
	for _, key := range featureKeys {
		fmt.Fprintf(&featureOut, "\n    %-24s %5d", key, feature[key])
	}
	log.Printf("特性计数：%s", featureOut.String())

	labels := map[int]string{0: "无", 1: "侠", 2: "恶", 3: "狂", 4: "邪"}
	var characterOut strings.Builder
	for _, key := range []int{0, 1, 2, 3, 4} {
		fmt.Fprintf(&characterOut, "\n    %-4s %5d", labels[key], characters[key])
	}
	log.Printf("性格标签：%s", characterOut.String())

	// 卡牌覆盖率：假人引用的 card_id 必须能在 game_cards 目录解析，缺失
	// 的运行时会回落基础外观（applyCardOverrides 打日志）。
	known := make(map[int32]struct{}, len(cardIDs))
	for _, id := range cardIDs {
		known[id] = struct{}{}
	}
	missingCards := map[int32]int{}
	for id, use := range cardUse {
		if _, ok := known[id]; !ok {
			missingCards[id] = use
		}
	}
	if len(missingCards) == 0 {
		log.Printf("卡牌覆盖：引用 %d 种 card_id 全部命中目录", len(cardUse))
	} else {
		var missingOut strings.Builder
		for id, use := range missingCards {
			fmt.Fprintf(&missingOut, "\n    card_id=%d 被引用 %d 次（目录缺失）", id, use)
		}
		log.Printf("卡牌覆盖：缺失 %d 种 card_id：%s", len(missingCards), missingOut.String())
	}

	// 重名检查：同场景重名不致冲突（dummy_id 唯一），仅提示观感。
	duplicates := 0
	for _, count := range names {
		if count > 1 {
			duplicates += count - 1
		}
	}
	log.Printf("重名：%d 条多余同名（不同场景可接受；同场景重名仅观感问题）", duplicates)
}

func sceneLabel(scene string) string {
	if scene == "" {
		return "(全场景)"
	}
	return scene
}

// ---- SQL 生成（注意：任何输出格式的改动都会改变迁移校验和）----

// batchRows 单条多行 VALUES 的行数。
const batchRows = 250

func buildSQL(dummies []dummyPlayer, cardIDs []int32, cards map[int32]cardCatalogEntry) []byte {
	var b strings.Builder
	writeArchiveHeader(&b)
	writeDummyTableSchema(&b)
	writeDummyColumnAlters(&b)
	writeDummyRows(&b, dummies)
	writeCardsTable(&b)
	writeCardRows(&b, cardIDs, cards)
	return []byte(b.String())
}

func writeArchiveHeader(b *strings.Builder) {
	b.WriteString("-- 假人系统权威存档：server_dummy_players（扩展 18 列）+ game_cards。\n")
	b.WriteString("-- 源数据：9yin-go-server2 data/dummy_players.json + card_models.json。\n")
	b.WriteString("-- 幂等：建表 IF NOT EXISTS；旧表逐列 information_schema 条件 ALTER；\n")
	b.WriteString("-- 数据 REPLACE INTO（dummy_id 显式 = JSON 序号，重放确定性覆盖）。\n\n")
}

func writeDummyTableSchema(b *strings.Builder) {
	b.WriteString(`CREATE TABLE IF NOT EXISTS server_dummy_players (
  dummy_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL DEFAULT '',
  sex TINYINT UNSIGNED NOT NULL DEFAULT 0,
  school VARCHAR(64) NOT NULL DEFAULT '',
  level INT NOT NULL DEFAULT 0,
  power_level INT NOT NULL DEFAULT 0,
  character_type INT NOT NULL DEFAULT 0,             -- 源字段 character（保留字）
  scene VARCHAR(64) NOT NULL DEFAULT '',
  action VARCHAR(32) NOT NULL DEFAULT '',
  neigong_buff VARCHAR(64) NOT NULL DEFAULT '',
  position JSON NULL,
  appearance JSON NULL,
  title_id INT NOT NULL DEFAULT 0,
  nei_gong VARCHAR(64) NULL DEFAULT NULL,
  card INT NOT NULL DEFAULT 0,
  patrol JSON NULL,
  patrol_random TINYINT(1) NOT NULL DEFAULT 0,
  move_mode VARCHAR(16) NULL DEFAULT NULL,
  yufeng TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (dummy_id),
  KEY ix_server_dummy_players_scene (scene),
  KEY ix_server_dummy_players_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

`)
}

// writeDummyColumnAlters 对 Java 0002 导入版旧表（11 列）逐列条件补扩展列。
func writeDummyColumnAlters(b *strings.Builder) {
	alters := []struct {
		name string
		ddl  string
	}{
		{"title_id", "ALTER TABLE server_dummy_players ADD COLUMN title_id INT NOT NULL DEFAULT 0 AFTER appearance"},
		{"nei_gong", "ALTER TABLE server_dummy_players ADD COLUMN nei_gong VARCHAR(64) NULL DEFAULT NULL AFTER title_id"},
		{"card", "ALTER TABLE server_dummy_players ADD COLUMN card INT NOT NULL DEFAULT 0 AFTER nei_gong"},
		{"patrol", "ALTER TABLE server_dummy_players ADD COLUMN patrol JSON NULL AFTER card"},
		{"patrol_random", "ALTER TABLE server_dummy_players ADD COLUMN patrol_random TINYINT(1) NOT NULL DEFAULT 0 AFTER patrol"},
		{"move_mode", "ALTER TABLE server_dummy_players ADD COLUMN move_mode VARCHAR(16) NULL DEFAULT NULL AFTER patrol_random"},
		{"yufeng", "ALTER TABLE server_dummy_players ADD COLUMN yufeng TINYINT(1) NOT NULL DEFAULT 0 AFTER move_mode"},
	}
	for _, alter := range alters {
		stmtVar := "has_" + alter.name
		b.WriteString(fmt.Sprintf("SET @%s = (\n  SELECT COUNT(*) FROM information_schema.COLUMNS\n  WHERE TABLE_SCHEMA = DATABASE()\n    AND TABLE_NAME = 'server_dummy_players'\n    AND COLUMN_NAME = '%s'\n);\n", stmtVar, alter.name))
		b.WriteString(fmt.Sprintf("SET @ddl_%s = IF(@%s = 0,\n  '%s',\n  'SELECT 1');\n", alter.name, stmtVar, alter.ddl))
		b.WriteString(fmt.Sprintf("PREPARE stmt_%s FROM @ddl_%s;\n", alter.name, alter.name))
		b.WriteString(fmt.Sprintf("EXECUTE stmt_%s;\n", alter.name))
		b.WriteString(fmt.Sprintf("DEALLOCATE PREPARE stmt_%s;\n\n", alter.name))
	}
}

func writeDummyRows(b *strings.Builder, dummies []dummyPlayer) {
	const columns = "dummy_id, name, sex, school, level, power_level, character_type, scene, action, neigong_buff, position, appearance, title_id, nei_gong, card, patrol, patrol_random, move_mode, yufeng"
	for start := 0; start < len(dummies); start += batchRows {
		end := start + batchRows
		if end > len(dummies) {
			end = len(dummies)
		}
		b.WriteString(fmt.Sprintf("REPLACE INTO server_dummy_players (%s) VALUES\n", columns))
		for i := start; i < end; i++ {
			d := dummies[i]
			patrol := "NULL"
			if len(d.Patrol) > 0 {
				patrol = sqlString(jsonText(d.Patrol))
			}
			b.WriteString(fmt.Sprintf("(%d, %s, %d, %s, %d, %d, %d, %s, %s, %s, %s, %s, %d, %s, %d, %s, %d, %s, %d)",
				i+1,
				sqlString(d.Name), d.Sex, sqlString(d.School), d.Level, d.PowerLevel, d.Character,
				sqlString(d.Scene), sqlString(d.Action), sqlString(d.NeiGongBuff),
				sqlString(jsonText(d.Pos)), sqlString(jsonText(d.Look)),
				d.TitleID, sqlString(d.NeiGong), d.Card,
				patrol, boolInt(d.PatrolRandom), sqlString(d.MoveMode), boolInt(d.Yufeng),
			))
			if i+1 < end {
				b.WriteString(",\n")
			} else {
				b.WriteString(";\n")
			}
		}
		if start+batchRows < len(dummies) {
			b.WriteString("\n")
		}
	}
}

func writeCardsTable(b *strings.Builder) {
	b.WriteString(`
CREATE TABLE IF NOT EXISTS game_cards (
  card_id INT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL DEFAULT '',
  config_id INT UNSIGNED NOT NULL DEFAULT 0,
  item_type INT NOT NULL DEFAULT 0,
  level INT NOT NULL DEFAULT 0,
  main_type INT NOT NULL DEFAULT 0,
  sub_type INT NOT NULL DEFAULT 0,
  flag INT NOT NULL DEFAULT 0,
  female_model_data VARCHAR(255) NOT NULL DEFAULT '',
  male_model_data VARCHAR(255) NOT NULL DEFAULT '',
  PRIMARY KEY (card_id),
  KEY ix_game_cards_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

`)
}

func writeCardRows(b *strings.Builder, cardIDs []int32, cards map[int32]cardCatalogEntry) {
	for start := 0; start < len(cardIDs); start += batchRows {
		end := start + batchRows
		if end > len(cardIDs) {
			end = len(cardIDs)
		}
		b.WriteString("REPLACE INTO game_cards (card_id, name, config_id, item_type, level, main_type, sub_type, flag, female_model_data, male_model_data) VALUES\n")
		for i := start; i < end; i++ {
			id := cardIDs[i]
			c := cards[id]
			b.WriteString(fmt.Sprintf("(%d, %s, %d, %d, %d, %d, %d, %d, %s, %s)",
				id, sqlString(c.Name), c.ConfigID, c.ItemType, c.Level, c.MainType, c.SubType, c.Flag,
				sqlString(c.FemaleModelData), sqlString(c.MaleModelData)))
			if i+1 < end {
				b.WriteString(",\n")
			} else {
				b.WriteString(";\n")
			}
		}
		if start+batchRows < len(cardIDs) {
			b.WriteString("\n")
		}
	}
}

// ---- 编码助手 ----

// jsonText 序列化 JSON 列（禁用 HTML 转义保证字节稳定）。
func jsonText(value any) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(buffer.String(), "\n")
}

// sqlString 单引号串字面量：反斜杠与单引号转义（MySQL 默认反斜杠转义
// 方言；migrations runner 的语句切分器只认反斜杠转义，双单引号写法会被
// 误判为串结尾切断语句，勿改回）。
func sqlString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
	return "'" + replacer.Replace(value) + "'"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// ---- 打包 ----

// writeSQLArchive 按 migrations 仓库约定打包：tar+gzip 单成员，包内文件名
// 与 .sql.gz 同名（去 .gz 后缀）。
func writeSQLArchive(path string, sql []byte) error {
	inner := strings.TrimSuffix(filepath.Base(path), ".gz")
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: inner, Mode: 0o644, Size: int64(len(sql))}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write(sql); err != nil {
		return err
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, archive.Bytes(), 0o644)
}
