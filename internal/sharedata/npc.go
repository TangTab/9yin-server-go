package sharedata

// npc.go NPC 模板表清单（share/npc/npcconfig）与装载序。
//
// 表序决定「同 ID 多表」时哪一行胜出：76 张表里有 44 个 ID 出现在两张
// 表里，其中 7 个是两个完全不同的 NPC 共用一个 ID（如 ngsj_npc_xhq_016
// 既是 newworldnpc.txt 里的「静心清泉」物件，又是别的表里的战斗怪）。
// XML 产物链按 ini/Npc/NpcConfig 的目录遍历序首档胜出，而那棵树的目录 /
// 文件名大小写出自 npc_type.ini 的登记（WorldNpc/city_CommonNpc.xml），
// 与客户端表的全小写名（worldnpc/city_commonnpc.txt）排序不同——直接按
// 客户端树遍历会换掉这 7 个 ID 的胜出行。这里按「构建器会把它放到哪」
// 排序，使直读与 XML 产物链的胜出者一致（npcfeed.TestTemplateSourceParity）。
//
// npc_type.ini（[file] 段，键名官方拼作 flie<N>）形如：
//
//	flie33=ini\Npc\NpcConfig\WorldNpc\city_CommonNpc.txt
//
// 即每张客户端表在服务端的原路径——客户端表就是服务端表的镜像。未登记
// 的表构建器按镜像路径输出（NpcConfig/<客户端相对路径>.xml，保留磁盘
// 大小写）。清单缺席或解析不出登记项时退回客户端树的遍历序。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

const (
	// NPCConfigDir NPC 模板表树（客户端 share 下，/ 分隔）。
	NPCConfigDir = "share/npc/npcconfig"
	// CreatorDir 场景摆放 creator XML 树。
	CreatorDir = "share/creator/npc_creator"
	// NpcTemplateFile 战斗模板主档（同目录 npctemplate*.tab 为补充档）。
	NpcTemplateFile = "share/task/npctemplate.tab"
	// QuestDomainDir 任务域表树（share/task/task/<域>/*.txt）。
	QuestDomainDir = "share/task/task"
	// TaskTextDir 任务文案（不在 share 下）。
	TaskTextDir = "text/chineses"
)

// NPCConfigTable 一张 NPC 模板表。
type NPCConfigTable struct {
	Path      string // 数据根下的路径（读取用）
	Rel       string // npcconfig 下的相对路径（/ 分隔，磁盘大小写）
	ServerRel string // 构建器产物路径（NpcConfig/…xml；仅定序与溯源）
}

// NPCConfigTables NPC 模板表清单，按构建器产物树的遍历序（口径见文件头）。
// npc_type.txt 是清单的文档镜像，不算表。
func NPCConfigTables(root string) ([]NPCConfigTable, error) {
	dir := filepath.Join(root, filepath.FromSlash(NPCConfigDir))
	var tables []NPCConfigTable
	err := res.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".txt") || strings.EqualFold(d.Name(), "npc_type.txt") {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		tables = append(tables, NPCConfigTable{Path: path, Rel: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// res.WalkDir 已是客户端树的遍历序（每层按名字排序）。
	declared, _ := NPCTypeManifest(root)
	if len(declared) == 0 {
		for i := range tables {
			tables[i].ServerRel = serverPathOf(tables[i].Rel, declared)
		}
		return tables, nil
	}
	// 构建器按客户端路径序逐表落盘；登记串里的目录大小写并不统一
	// （ini\Npc\NpcConfig\… 与 ini\Npc\npcConfig\… 并存），在不分大小写的
	// 文件系统上都并进首次建出的那个目录——产物树的目录名取首次出现的
	// 大小写，文件名保留登记串（未登记表保留磁盘名）的大小写。
	dirCase := map[string]string{} // 小写目录路径 → 首次出现的大小写
	for i := range tables {
		parts := strings.Split(serverPathOf(tables[i].Rel, declared), "/")
		prefix, canonical := "", ""
		for d := 0; d+1 < len(parts); d++ { // 各级目录（末段是文件名）
			if d > 0 {
				prefix += "/"
			}
			prefix += parts[d]
			if _, seen := dirCase[strings.ToLower(prefix)]; !seen {
				dirCase[strings.ToLower(prefix)] = prefix
			}
			canonical = dirCase[strings.ToLower(prefix)]
		}
		tables[i].ServerRel = canonical + "/" + parts[len(parts)-1]
	}
	SortWalkOrder(tables, func(i int) string { return tables[i].ServerRel })
	return tables, nil
}

// serverPathOf 客户端相对路径 → 构建器产物路径：npc_type.ini 登记的为准，
// 未登记的按镜像路径（保留磁盘大小写）。
func serverPathOf(rel string, declared map[string]string) string {
	if server, ok := declared[strings.ToLower(rel)]; ok {
		return server
	}
	return "NpcConfig/" + strings.TrimSuffix(rel, filepath.Ext(rel)) + ".xml"
}

// NPCTypeManifest 解析 npc_type.ini：小写客户端相对路径（worldnpc/
// city_commonnpc.txt）→ 产物路径（NpcConfig/WorldNpc/city_CommonNpc.xml）。
// 文件缺席、无登记项或解析失败返回空表（调用方退回客户端树遍历序）。
func NPCTypeManifest(root string) (map[string]string, error) {
	raw, err := res.ReadFile(filepath.Join(root, filepath.FromSlash(NPCConfigDir), "npc_type.ini"))
	if err != nil {
		return nil, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, fmt.Errorf("sharedata: 解码 npc_type.ini: %w", err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "[") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		value := strings.TrimSpace(line[eq+1:])
		if value == "" {
			continue
		}
		server, client, ok := splitServerNPCPath(value)
		if !ok {
			continue
		}
		out[strings.ToLower(client)] = server
	}
	return out, nil
}

// splitServerNPCPath 把 ini\Npc\NpcConfig\WorldNpc\city_CommonNpc.txt 拆成
// （产物路径 NpcConfig/WorldNpc/city_CommonNpc.xml，客户端相对路径
// WorldNpc/city_CommonNpc.txt——即 npcconfig 下的磁盘路径）。路径里没有
// ini/npc/ 或不是 .txt 的值不是表登记。
func splitServerNPCPath(value string) (serverRel, clientRel string, ok bool) {
	normalized := filepath.ToSlash(value)
	const marker = "ini/npc/" // 在 ToLower 后的串里搜，必须全小写
	idx := strings.Index(strings.ToLower(normalized), marker)
	if idx < 0 {
		return "", "", false
	}
	after := normalized[idx+len(marker):]
	if !strings.HasSuffix(strings.ToLower(after), ".txt") {
		return "", "", false
	}
	serverRel = strings.TrimSuffix(after, ".txt") + ".xml"
	clientRel = strings.TrimSuffix(after, ".txt") + ".txt"
	// clientRel 去掉 NpcConfig/ 层（npcconfig 磁盘树从组目录开始）。
	const groupLayer = "NpcConfig/"
	if len(clientRel) > len(groupLayer) && strings.EqualFold(clientRel[:len(groupLayer)], groupLayer) {
		clientRel = clientRel[len(groupLayer):]
	}
	return serverRel, clientRel, true
}

// SortWalkOrder 按目录树遍历序就地重排 items（等价 filepath.WalkDir 的访问
// 序：每个目录的直接子项——文件与子目录混在一起——按名字字节序，深度
// 优先）。pathOf 给出每项的 / 分隔路径。不能直接按整串排序：目录名与
// 文件名比较时不带分隔符（"Fun" < "Fun.xml"，而 "Fun/x" > "Fun.xml"）。
func SortWalkOrder[T any](items []T, pathOf func(int) string) {
	type node struct {
		name string
		kids map[string]*node
		leaf []int // 落在这个节点上的项（同路径重复项保持原序）
	}
	root := &node{kids: map[string]*node{}}
	for i := range items {
		cur := root
		for _, part := range strings.Split(pathOf(i), "/") {
			if part == "" {
				continue
			}
			kid, ok := cur.kids[part]
			if !ok {
				kid = &node{name: part, kids: map[string]*node{}}
				cur.kids[part] = kid
			}
			cur = kid
		}
		cur.leaf = append(cur.leaf, i)
	}
	order := make([]int, 0, len(items))
	var visit func(*node)
	visit = func(n *node) {
		order = append(order, n.leaf...)
		names := make([]string, 0, len(n.kids))
		for name := range n.kids {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			visit(n.kids[name])
		}
	}
	visit(root)
	if len(order) != len(items) {
		return // 防御：路径拆分异常时保持原序
	}
	sorted := make([]T, len(items))
	for pos, i := range order {
		sorted[pos] = items[i]
	}
	copy(items, sorted)
}
