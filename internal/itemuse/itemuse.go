// Package itemuse 装载使用物效果模块脚本（scripts/module/item/*.lua，自参考
// 服拷贝）：M.on_use(ctx) 返回使用效果指令（msg31 使用链的脚本层）。
//
// 宿主机制由 internal/modulevm 提供（坏脚本只禁用该模块、VM 不重建）。
// Use 返回 nil（无脚本/坏脚本/返回 nil）时调用方走 INI 数据回退（ItemUseCatalog
// 等价物，使用链落地时接线）——使用链永不被脚本层拒绝。
//
// 契约要点（参考服 skill-script-framework.md item 节）：随机数由调用方预掷
// （rand{r_drop,r_proc}，脚本不掷骰）；consume 缺省 true。
package itemuse

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

// maxGrantItems 是 grant_items 指令的条目上限（契约写死 16）。
const maxGrantItems = 16

// Item 是一个使用物脚本的目录项（装载时读自 M.meta）。
type Item struct {
	ID   string // 物品 config id，即模块键
	Name string // 物品名
}

// Catalog 是使用物效果模块宿主的域包装。零值不可用，经 New 创建。
type Catalog struct {
	host *modulevm.Host

	// mu 只保护域目录；锁纪律同 neigong（host→domain 单向）。
	mu    sync.Mutex
	items map[string]Item
	order []string
}

// New 创建使用物效果宿主。dir 为空表示禁用：Use 恒返回 nil。
func New(dir string, logger Logger) *Catalog {
	return &Catalog{host: modulevm.New(dir, logger)}
}

// SetTimeout 调整单次调用预算（测试用）。
func (c *Catalog) SetTimeout(d time.Duration) { c.host.SetTimeout(d) }

// Enabled 报告宿主是否配置了脚本目录（-item-scripts 非空）。
func (c *Catalog) Enabled() bool { return c.host.Enabled() }

// Loaded 返回最近一次装载成功的模块数。
func (c *Catalog) Loaded() int { return c.host.Loaded() }

// Count 是 Loaded 的别名，与目录型注册表的命名对齐。
func (c *Catalog) Count() int { return c.Loaded() }

// Broken 返回被禁用的物品 id（运行期出错或 meta 域校验失败）。
func (c *Catalog) Broken() []string { return c.host.Broken() }

// Reload 重新装载脚本目录（GM 热重载）。
func (c *Catalog) Reload() (int, error) { return c.Load() }

// Close 释放 VM。之后的 Use 返回 nil。
func (c *Catalog) Close() error {
	c.mu.Lock()
	c.items = nil
	c.order = nil
	c.mu.Unlock()
	return c.host.Close()
}

// Load 装载脚本目录并读 M.meta{id,name} 建域目录。meta 缺失的模块禁用并
// 计入返回错误（仓内脚本均带 meta）；id 与文件名不符以文件名为键并告警。
func (c *Catalog) Load() (int, error) {
	loaded, loadErr := c.host.Load()
	c.mu.Lock()
	c.items = make(map[string]Item)
	c.order = nil
	c.mu.Unlock()

	var invalid []string
	failures := c.host.EachModule(func(name string, table vm.LuaTable) error {
		item, metaErr := readMeta(table, name)
		if metaErr != nil {
			invalid = append(invalid, name)
			return fmt.Errorf("itemuse: %s: %w", name, metaErr)
		}
		if item.ID != name {
			c.host.Warnf("itemuse: %s: meta.id %q 与文件名不符，以文件名为键 %q", name, item.ID, name)
			item.ID = name
		}
		c.mu.Lock()
		c.items[name] = item
		c.mu.Unlock()
		return nil
	})
	for _, name := range invalid {
		c.host.MarkBroken(name, "meta 域校验失败")
	}
	c.mu.Lock()
	c.order = make([]string, 0, len(c.items))
	for id := range c.items {
		c.order = append(c.order, id)
	}
	sort.Strings(c.order)
	c.mu.Unlock()
	return loaded, errors.Join(append([]error{loadErr}, failures...)...)
}

// Items 返回全部目录项（按 id 排序）。
func (c *Catalog) Items() []Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Item, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.items[id])
	}
	return out
}

// Item 按 id 查目录项。
func (c *Catalog) Item(id string) (Item, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[id]
	return item, ok
}

// readMeta 读 M.meta{id,name}；meta 表或 id 缺失视为坏脚本。
func readMeta(table vm.LuaTable, stem string) (Item, error) {
	metaValue := modulevm.Field(table, "meta")
	if !metaValue.IsTable() {
		return Item{}, fmt.Errorf("missing M.meta table")
	}
	meta := metaValue.AsTable()
	item := Item{
		ID:   modulevm.Field(meta, "id").AsString(),
		Name: modulevm.Field(meta, "name").AsString(),
	}
	if item.ID == "" {
		return Item{}, fmt.Errorf("M.meta needs id")
	}
	return item, nil
}

// SelfStats 是脚本可见的使用者快照（ctx.self）。
type SelfStats struct {
	HP, MaxHP int64
	MP, MaxMP int64
	Silver    int64
}

// Ctx 是一次 on_use 调用的环境快照（只读，调用方预填；随机数预掷）。
// 零值合法。
type Ctx struct {
	ItemID string // 物品 config id（透传给脚本）
	Amount int64  // 使用数量（现链路恒 1）
	NowMS  int64
	Self   SelfStats
	RDrop  float64 // rand.r_drop：开包掉落掷 [0,1)
	RProc  float64 // rand.r_proc：触发掷 [0,1)
}

// OpenBox 是开包指令：按掉落包权重抽 rolls 次。
type OpenBox struct {
	DropID string
	Rolls  int64
}

// GrantItem 是确定性直接给物指令。
type GrantItem struct {
	ID     string
	Amount int64
}

// TempStat 是限时属性指令（同源替换，到期自清）。
type TempStat struct {
	AttackAdd  int64
	DefenseAdd int64
	MaxHPAdd   int64
	LifeMS     int64
}

// Effect 是 on_use 返回指令白名单的 Go 形态；nil 字段/零值表示未下发。
// 白名单外的键与畸形值 WARN 丢弃，不阻断使用链。
type Effect struct {
	HealHP     int64       // heal_hp：回复气血
	HealMP     int64       // heal_mp：回复内力
	OpenBox    *OpenBox    // open_box
	GrantItems []GrantItem // grant_items（≤16）
	TempStat   *TempStat   // temp_stat
	Consume    bool        // consume：是否消耗 1 个（缺省 true）
}

// Use 调用 M.on_use。nil 表示无效果（无脚本/坏脚本/返回 nil），调用方走
// INI 回退。
func (c *Catalog) Use(itemID string, ctx *Ctx) *Effect {
	if ctx == nil {
		ctx = &Ctx{}
	}
	result, ok := c.host.Invoke(itemID, []string{"on_use"}, func(v *vm.VM) vm.Value {
		return vm.NewTable(buildCtxTable(v, itemID, ctx))
	})
	if !ok || result.IsNil() || !result.IsTable() {
		return nil
	}
	return c.convertEffect(itemID, result.AsTable())
}

// buildCtxTable 构建只读 ctx 表（字段名按参考服契约）。
func buildCtxTable(v *vm.VM, itemID string, ctx *Ctx) *vm.Table {
	t := vm.NewEmptyTable()
	t.SetString("item_id", vm.NewString(itemID))
	t.SetString("amount", vm.NewInt(ctx.Amount))
	t.SetString("now", vm.NewInt(ctx.NowMS))
	self := vm.NewEmptyTable()
	self.SetString("hp", vm.NewInt(ctx.Self.HP))
	self.SetString("max_hp", vm.NewInt(ctx.Self.MaxHP))
	self.SetString("mp", vm.NewInt(ctx.Self.MP))
	self.SetString("max_mp", vm.NewInt(ctx.Self.MaxMP))
	self.SetString("silver", vm.NewInt(ctx.Self.Silver))
	t.SetString("self", vm.NewTable(self))
	rand := vm.NewEmptyTable()
	rand.SetString("r_drop", vm.NewFloat(ctx.RDrop))
	rand.SetString("r_proc", vm.NewFloat(ctx.RProc))
	t.SetString("rand", vm.NewTable(rand))
	return t
}

// convertEffect 按白名单转换返回表；全无效时返回 nil。
func (c *Catalog) convertEffect(itemID string, table vm.LuaTable) *Effect {
	effect := &Effect{Consume: true}
	set := false
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() {
			c.host.WarnOnce(itemID+":key", "itemuse: %s 返回非字符串键 %s，已丢弃", itemID, modulevm.ValueString(nextKey))
			continue
		}
		name := nextKey.AsString()
		switch name {
		case "heal_hp", "heal_mp":
			amount, ok := modulevm.LuaInt(value)
			if !ok || amount <= 0 {
				c.host.WarnOnce(itemID+":"+name, "itemuse: %s %s %s 非正/畸形，已丢弃", itemID, name, modulevm.ValueString(value))
				continue
			}
			if name == "heal_hp" {
				effect.HealHP = amount
			} else {
				effect.HealMP = amount
			}
			set = true
		case "open_box":
			if box, ok := c.convertOpenBox(itemID, value); ok {
				effect.OpenBox, set = box, true
			}
		case "grant_items":
			if grants, ok := c.convertGrantItems(itemID, value); ok {
				effect.GrantItems, set = grants, true
			}
		case "temp_stat":
			if stat, ok := c.convertTempStat(itemID, value); ok {
				effect.TempStat, set = stat, true
			}
		case "consume":
			if value.IsBool() {
				effect.Consume = value.AsBool()
				set = true
			} else {
				c.host.WarnOnce(itemID+":consume", "itemuse: %s consume %s 非布尔，已丢弃", itemID, modulevm.ValueString(value))
			}
		default:
			c.host.WarnOnce(itemID+":"+name, "itemuse: %s 返回白名单外指令 %s=%s，已丢弃", itemID, name, modulevm.ValueString(value))
		}
	}
	if !set {
		return nil
	}
	return effect
}

func (c *Catalog) convertOpenBox(itemID string, value vm.Value) (*OpenBox, bool) {
	if !value.IsTable() {
		c.host.WarnOnce(itemID+":open_box", "itemuse: %s open_box %s 非表，整条丢弃", itemID, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	dropID := modulevm.Field(table, "drop_id").AsString()
	rolls, rollsOK := modulevm.LuaInt(modulevm.Field(table, "rolls"))
	if dropID == "" || !rollsOK || rolls < 1 || rolls > 10 {
		c.host.WarnOnce(itemID+":open_box", "itemuse: %s open_box {drop_id=%q, rolls=%s} 畸形（rolls 1..10），整条丢弃",
			itemID, dropID, modulevm.ValueString(modulevm.Field(table, "rolls")))
		return nil, false
	}
	return &OpenBox{DropID: dropID, Rolls: rolls}, true
}

func (c *Catalog) convertGrantItems(itemID string, value vm.Value) ([]GrantItem, bool) {
	if !value.IsTable() {
		c.host.WarnOnce(itemID+":grant_items", "itemuse: %s grant_items %s 非表，整条丢弃", itemID, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	n := table.Len()
	if n <= 0 {
		c.host.WarnOnce(itemID+":grant_items", "itemuse: %s grant_items 为空，整条丢弃", itemID)
		return nil, false
	}
	grants := make([]GrantItem, 0, n)
	for i := 1; i <= n; i++ {
		entry := modulevm.Index(table, i)
		if !entry.IsTable() {
			c.host.WarnOnce(itemID+":grant_items", "itemuse: %s grant_items[%d] 非表，已丢弃", itemID, i)
			continue
		}
		item := entry.AsTable()
		id := modulevm.Field(item, "id").AsString()
		amount, amountOK := modulevm.LuaInt(modulevm.Field(item, "amount"))
		if id == "" || !amountOK || amount <= 0 {
			c.host.WarnOnce(itemID+":grant_items:"+fmt.Sprint(i), "itemuse: %s grant_items[%d] {id=%q, amount=%s} 畸形，已丢弃",
				itemID, i, id, modulevm.ValueString(modulevm.Field(item, "amount")))
			continue
		}
		if len(grants) >= maxGrantItems {
			c.host.WarnOnce(itemID+":grant_items:cap", "itemuse: %s grant_items 超过 %d 条上限，多余丢弃", itemID, maxGrantItems)
			break
		}
		grants = append(grants, GrantItem{ID: id, Amount: amount})
	}
	if len(grants) == 0 {
		return nil, false
	}
	return grants, true
}

func (c *Catalog) convertTempStat(itemID string, value vm.Value) (*TempStat, bool) {
	if !value.IsTable() {
		c.host.WarnOnce(itemID+":temp_stat", "itemuse: %s temp_stat %s 非表，整条丢弃", itemID, modulevm.ValueString(value))
		return nil, false
	}
	table := value.AsTable()
	attack, attackOK := modulevm.LuaInt(modulevm.Field(table, "attack_add"))
	defense, defenseOK := modulevm.LuaInt(modulevm.Field(table, "defense_add"))
	maxHP, maxHPOK := modulevm.LuaInt(modulevm.Field(table, "max_hp_add"))
	life, lifeOK := modulevm.LuaInt(modulevm.Field(table, "life_ms"))
	if !attackOK || !defenseOK || !maxHPOK || !lifeOK || life <= 0 {
		c.host.WarnOnce(itemID+":temp_stat", "itemuse: %s temp_stat 畸形（life_ms 须正，加成字段须数值），整条丢弃", itemID)
		return nil, false
	}
	return &TempStat{AttackAdd: attack, DefenseAdd: defense, MaxHPAdd: maxHP, LifeMS: life}, true
}
