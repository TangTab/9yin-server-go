// Package neigong 装载内功特效模块脚本（scripts/module/neigong/buf_ng_*.lua，
// 自参考服拷贝的 155 本），并向未来的战斗分发层提供钩子调用桥。
//
// 宿主机制（沙盒护栏、装载软语义、运行期坏脚本只禁用该本）由
// internal/modulevm 提供并以薄封装复用；本包负责内功域：M.meta 目录
// （Books/Book/ByBook）、钩子别名链、ctx 表构建与效果指令白名单转换。
// 错误策略与调用方契约见 modulevm 包注释：无书/无钩子/坏脚本 → 零效果，
// 出招链永不被脚本层阻断。
package neigong

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/iceisfun/golua/v2/vm"
	"github.com/local/9yin-go-server/internal/modulevm"
)

// Logger 与标准库 log.Printf 兼容；宿主注入，包内不直接打日志。
type Logger = modulevm.Logger

// hookNames 是内功域探测的钩子函数全集（M.<name>）。on_hit/on_be_hit/on_tick
// 是生成桩对 on_attack/on_victim/on_periodic 的别名，经 hookChain 兜底；
// on_dodge（成功闪避事件）为本仓对参考服钩子集的扩展。
var hookNames = []string{
	"on_attack", "on_victim", "on_parry", "on_periodic",
	"on_tick", "on_hit", "on_be_hit", "on_dodge", "passive",
}

// hookChain 把战斗事件名解析为按序探测的钩子名：主钩子缺失时尝试别名
// （对齐参考服 NeiGongEffects 的分发语义）。不在映射内的名字按原样直连，
// 因此调用方也可以直接传裸钩子名（on_hit 等）。
func hookChain(event string) []string {
	switch event {
	case "on_attack":
		return []string{"on_attack", "on_hit"}
	case "on_victim":
		return []string{"on_victim", "on_be_hit"}
	case "on_periodic":
		return []string{"on_periodic", "on_tick"}
	default:
		return []string{event}
	}
}

// Book 是一本内功特效脚本的目录项（装载时读自 M.meta）。
type Book struct {
	ID       string // buff id，即模块键（buf_ng_bg_001）
	Name     string // 内功名（大巧若拙经）
	Book     string // 书 id（ng_bg_001）
	Category string // 分类（hit/passive/…）
	MaxTier  int    // 层数
}

// Effects 是内功特效模块宿主的域包装。零值不可用，经 New 创建。
type Effects struct {
	host *modulevm.Host

	// mu 只保护域目录（books/byBook/order）；锁纪律：持有 host.mu 的
	// EachModule 回调内可再取 mu（host→domain 单向），反向绝不发生。
	mu     sync.Mutex
	books  map[string]Book
	byBook map[string]string // 书 id → buff id
	order  []string          // 排序后的 buff id（Books 的稳定序）
}

// New 创建内功特效宿主。dir 为空表示禁用：所有调用返回零效果，Load 为空
// 操作。logger 为 nil 时告警静默丢弃（测试可用）。
func New(dir string, logger Logger) *Effects {
	return &Effects{host: modulevm.New(dir, logger)}
}

// SetTimeout 调整单次调用预算（测试用）。
func (e *Effects) SetTimeout(d time.Duration) { e.host.SetTimeout(d) }

// Enabled 报告宿主是否配置了脚本目录（-neigong-scripts 非空）。
func (e *Effects) Enabled() bool { return e.host.Enabled() }

// Loaded 返回最近一次装载成功的模块数。
func (e *Effects) Loaded() int { return e.host.Loaded() }

// Generation 返回装载代数（热重载递增）；被动属性等静态快照据此失效。
func (e *Effects) Generation() int64 { return e.host.Generation() }

// Count 是 Loaded 的别名，与其余目录型注册表（npcfunc 等）的命名对齐。
func (e *Effects) Count() int { return e.Loaded() }

// Broken 返回运行期出错或域校验失败被禁用的 buff id（排序后）。
func (e *Effects) Broken() []string { return e.host.Broken() }

// Books 返回全部已装载内功的目录项（按 buff id 排序）。
func (e *Effects) Books() []Book {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Book, 0, len(e.order))
	for _, id := range e.order {
		out = append(out, e.books[id])
	}
	return out
}

// Book 按 buff id 查目录项。
func (e *Effects) Book(id string) (Book, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	book, ok := e.books[id]
	return book, ok
}

// ByBook 按书 id（ng_jh_001）反查 buff id（buf_ng_jh_001）。
func (e *Effects) ByBook(book string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	id, ok := e.byBook[book]
	return id, ok
}

// Load （重新）装载脚本目录：modulevm 装载全部返回表的模块，本包随后读
// M.meta 建域目录。meta 缺失/不完整的模块被禁用（Broken）并计入返回错误；
// 其余装载失败沿用 modulevm 软语义（跳过、汇总、不中断）。
func (e *Effects) Load() (int, error) {
	loaded, loadErr := e.host.Load()
	e.mu.Lock()
	e.books = make(map[string]Book)
	e.byBook = make(map[string]string)
	e.order = nil
	e.mu.Unlock()

	var invalid []string
	failures := e.host.EachModule(func(name string, table vm.LuaTable) error {
		meta, metaErr := readMeta(table, name)
		if metaErr != nil {
			invalid = append(invalid, name)
			return fmt.Errorf("neigong: %s: %w", name, metaErr)
		}
		if meta.ID != name {
			e.host.Warnf("neigong: %s: meta.id %q 与文件名不符，以文件名为键 %q", name, meta.ID, name)
			meta.ID = name
		}
		e.mu.Lock()
		e.books[name] = meta
		if meta.Book != "" {
			if prev, dup := e.byBook[meta.Book]; !dup || name < prev {
				e.byBook[meta.Book] = name
			}
		}
		e.mu.Unlock()
		return nil
	})
	for _, name := range invalid {
		e.host.MarkBroken(name, "meta 域校验失败")
	}
	e.mu.Lock()
	e.order = make([]string, 0, len(e.books))
	for id := range e.books {
		e.order = append(e.order, id)
	}
	sort.Strings(e.order)
	e.mu.Unlock()
	return loaded, errors.Join(append([]error{loadErr}, failures...)...)
}

// Reload 重新装载脚本目录（GM 热重载）。
func (e *Effects) Reload() (int, error) { return e.Load() }

// Close 释放 VM。之后的 Hook/Passive 返回零效果。
func (e *Effects) Close() error {
	e.mu.Lock()
	e.books = nil
	e.byBook = nil
	e.order = nil
	e.mu.Unlock()
	return e.host.Close()
}

// readMeta 从模块表读 M.meta{id,name,book,category,max_tier}。meta 表或
// id/name 缺失视为坏脚本（155 本仓内脚本均有完整 meta）。
func readMeta(table vm.LuaTable, stem string) (Book, error) {
	metaValue := modulevm.Field(table, "meta")
	if !metaValue.IsTable() {
		return Book{}, fmt.Errorf("missing M.meta table")
	}
	meta := metaValue.AsTable()
	book := Book{
		ID:       modulevm.Field(meta, "id").AsString(),
		Name:     modulevm.Field(meta, "name").AsString(),
		Book:     modulevm.Field(meta, "book").AsString(),
		Category: modulevm.Field(meta, "category").AsString(),
		MaxTier:  1,
	}
	if book.ID == "" || book.Name == "" {
		return Book{}, fmt.Errorf("M.meta needs id and name")
	}
	if tier := modulevm.Field(meta, "max_tier"); tier.IsNumber() {
		if t, ok := modulevm.LuaInt(tier); ok && t > 0 {
			book.MaxTier = int(t)
		}
	}
	return book, nil
}

// Hook 调用一本内功的事件钩子。event 为战斗事件名（on_attack/on_victim/
// on_parry/on_periodic）或裸钩子名，别名链见 hookChain。返回效果指令；
// nil 表示零效果（无模块/无钩子/已禁用/脚本返回 nil/返回值不可用）。
// 脚本运行期出错只禁用该本，不上抛错误——战斗链永不被脚本层阻断。
func (e *Effects) Hook(buffID, event string, ctx *Ctx) *Effect {
	if ctx == nil {
		ctx = &Ctx{}
	}
	result, ok := e.host.Invoke(buffID, hookChain(event), func(v *vm.VM) vm.Value {
		return vm.NewTable(buildCtxTable(v, ctx))
	})
	if !ok || result.IsNil() || (result.IsBool() && !result.AsBool()) {
		return nil
	}
	if !result.IsTable() {
		e.host.WarnOnce(buffID+":return", "neigong: %s: %s 返回非表值 %s，按零效果处理", buffID, event, modulevm.ValueString(result))
		return nil
	}
	return e.convertEffect(buffID, event, result.AsTable())
}

// Passive 调用 M.passive，返回扁平数值属性表（max_mp_add/attack_add 等，
// 键开放、值为数值）。无钩子/坏脚本返回 nil。
func (e *Effects) Passive(buffID string, ctx *Ctx) map[string]float64 {
	if ctx == nil {
		ctx = &Ctx{}
	}
	result, ok := e.host.Invoke(buffID, []string{"passive"}, func(v *vm.VM) vm.Value {
		return vm.NewTable(buildCtxTable(v, ctx))
	})
	if !ok || result.IsNil() || !result.IsTable() {
		return nil
	}
	table := result.AsTable()
	out := make(map[string]float64)
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() || !value.IsNumber() {
			e.host.WarnOnce(buffID+":passive", "neigong: passive 非数值条目 %s=%s 已丢弃", modulevm.ValueString(nextKey), modulevm.ValueString(value))
			continue
		}
		out[nextKey.AsString()] = value.AsFloat()
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
