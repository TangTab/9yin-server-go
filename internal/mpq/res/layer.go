package res

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// 层文件命名：<前缀><序号>.rep，序号即层号，越大优先级越高。
//
//	b001.rep b002.rep …      仓内测试数据根（testdata）
//	data001.rep data002.rep … 正式服务端数据根（Res）
//	h001.rep h002.rep …      参考资源根（testdata/references）：黑金等原档，只读
//
// 读入时不限序号位数（b1.rep 与 b001.rep 同为第 1 层），写出固定三位。
const (
	// PrefixTest 是仓内测试数据根的层文件名前缀。
	PrefixTest = "b"
	// PrefixData 是正式服务端数据根的层文件名前缀。
	PrefixData = "data"
	// PrefixReference 是参考资源根的层文件名前缀：黑金等原档资源包
	// （testdata/references/h001.rep），只读，供离线构建取样本与测试夹具。
	PrefixReference = "h"

	// deletedFile 是层内登记删除的内部文件：每行一个名字。某层登记的
	// 名字对其以下各层视为不存在；更高层重新收录或根下出现同名散文件
	// 即恢复。MPQ 覆盖链只能"顶掉"不能"删掉"，删除靠它表达。
	deletedFile = "(deleted)"

	layerDigits   = 3
	maxLayerCount = 999999
)

var layerNameRE = regexp.MustCompile(`(?i)^(data|b|h)([0-9]+)\.rep$`)

// familyOrder 是自动探测与报错时的族序。
var familyOrder = []string{PrefixData, PrefixTest, PrefixReference}

// IsPrefix 报告 name 是否为层文件族前缀（PrefixData / PrefixTest /
// PrefixReference 之一，不分大小写）。
func IsPrefix(name string) bool {
	name = canonicalPrefix(name)
	for _, prefix := range familyOrder {
		if prefix == name {
			return true
		}
	}
	return false
}

// Families 返回全部层文件族前缀（自动探测时的序）。
func Families() []string { return append([]string(nil), familyOrder...) }

// layerFile 是磁盘上的一个层文件。
type layerFile struct {
	number int
	name   string // 磁盘文件名（原样大小写）
}

// label 返回层名（层文件名去扩展名的小写形式，如 b002）。
func (f layerFile) label() string {
	return strings.TrimSuffix(strings.ToLower(f.name), ".rep")
}

// canonicalPrefix 规范化前缀为小写；不认识的前缀原样返回（由调用方报错）。
func canonicalPrefix(prefix string) string {
	return strings.ToLower(strings.TrimSpace(prefix))
}

// parseLayerName 解析层文件名；不是层文件时 ok=false。
func parseLayerName(file string) (prefix string, number int, ok bool) {
	m := layerNameRE.FindStringSubmatch(file)
	if m == nil {
		return "", 0, false
	}
	number, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, false
	}
	return strings.ToLower(m[1]), number, true
}

// isLayerFile 报告文件名是否为层容器（任何前缀、任何序号）。
func isLayerFile(file string) bool {
	_, _, ok := parseLayerName(file)
	return ok
}

// layerFileName 给出层号对应的规范文件名。
func layerFileName(prefix string, number int) string {
	return fmt.Sprintf("%s%0*d.rep", prefix, layerDigits, number)
}

// scanLayers 列出根下各族的层文件，按层号升序。
func scanLayers(root string) (map[string][]layerFile, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("res: 资源根不可读：%s（%w）", root, err)
	}
	families := map[string][]layerFile{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		prefix, number, ok := parseLayerName(e.Name())
		if !ok {
			continue
		}
		families[prefix] = append(families[prefix], layerFile{number: number, name: e.Name()})
	}
	for prefix := range families {
		files := families[prefix]
		sort.Slice(files, func(i, j int) bool { return files[i].number < files[j].number })
	}
	return families, nil
}
